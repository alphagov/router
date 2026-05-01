# Router

GOV.UK Router is an HTTP reverse proxy built on top of [`triemux`][tm].

## How router loads routes

Router loads its routing table from [Content Store's](https://github.com/alphagov/content-store/) PostgreSQL database (or optionally from a flat file) into a [trie data structure](https://en.wikipedia.org/wiki/Trie) for fast path lookups.

Router can reload routes without restarting:
1. Automatically via PostgreSQL's `LISTEN/NOTIFY` mechanism
2. Periodic schedule
3. Manually via the API server

Internally these use a Go channel to send reload requests that causes Router to reload from `content-store's` PostgreSQL database.

## Routes

Routes can be one of two types:
- **exact**: The path must match exactly (e.g. the exact route `/government` only matches a request for `/government`)
- **prefix**: The path prefix must match (e.g. the prefix route `/government` matches requests for `/government`, `/government/policies`, etc.)

The route type and URL path determine which route gets matched to a particular request.

Router maintains two separate tries:
1. Exact path matches
2. Prefix matches

Once a request comes in, Router uses the URL path to first check for an exact match, then falls back to the longest prefix match.

Suppose we have the following routes:
1. Prefix route on `/foo`
2. Exact route on `/foo/bar`
3. Exact route on `/bar`

Then Router will:
1. Returns `404` if a request is made for the children of an exact route (e.g. `/bar/foo/`).
2. Match on the prefix route if the request is made for `/foo`
3. Match on the exact route if the request is made for `/foo/bar`
4. Match on the prefix route if the request is made for `/foo/bar/baz` as there is no matching exact route

See [route_selection_test.go](https://github.com/alphagov/router/blob/2c46c40d43ff4feefeb112cd6aa1e44f0da4b417/integration_tests/route_selection_test.go) for more cases.


### Handling

Routes have a `schemaName` property:
1. Backend
2. Redirect
3. Gone

Once a request is matched to a route, Router uses the `schemaName` property to determine how the request should be handled.

There are 3 handler types to handle a request:
1. **backend**: Reverse proxies the request to a backend application server
2. **redirect**: Returns an HTTP `301` redirect to a new location
3. **gone**: Returns an HTTP `410` Gone response for deleted content

Note: some `Gone` routes are also handled by the `backend` handler.

Router otherwise:
- serves `503` if no routes are loaded
- serves `404` if the route can't be found

### Redirect routes

Redirect routes have a flag that is used to determine whether the URL path in the request should be preserved.

If the source path is `/source` and the redirect target is `/target` then the target URL will preserve the path as follows:

```
https://source.example.com/target/path/subpath?q1=a&q2=b
```

Otherwise the URL will be:

```
https://source.example.com/target/
```

Redirect routes will only redirect to a lowercase route if the URL path is in all caps (e.g. `/GOVERNMENT/GUIDANCE` will redirect to `/government/guidance`).

For details on the route data structure and handler configuration, see [docs/data-structure.md](docs/data-structure.md).

## Request flow

```mermaid
graph LR;
    A[Fastly]-->B[Router Load Balancer];
    B[Router Load Balancer]-->C[Router nginx];
    C[Router nginx]-->D[Router];
    D[Router]-->E[Backend];
```

Router's [load balancer](https://docs.aws.amazon.com/elasticloadbalancing/latest/application/x-forwarded-headers.html) adds the following headers:
1. `X-Forwarded-For`
2. `X-Forwarded-Proto`
3. `X-Forwarded-Port`

Router doesn't proxy redirect and gone routes to a backend but simply returns the response to the client.

### Draft stack

The [draft stack](https://docs.publishing.service.gov.uk/manual/content-preview.html) consists of 'draft' deployments of Router, 
content store and backends.

Here the request passes through an [authenticating proxy](https://github.com/alphagov/authenticating-proxy/) before it hits draft router:

```mermaid
graph LR;
    A[Authenticating Proxy Load Balancer]-->B[Authenticating Proxy nginx];
    B[Authenticating Proxy nginx]-->C[Authenticating Proxy];
    C[Authenticating Proxy]-->D[Draft Router nginx];
    D[Draft Router nginx]-->E[Draft Router];
    E[Draft Router]-->F[Draft backend];
```

In addition to the headers added by the load balancer authenticating proxy adds the following headers:
1. `X_GOVUK_AUTHENTICATED_USER_ORGANISATION`
2. `X_GOVUK_AUTHENTICATED_USER`
3. `X-Forwarded-Host` replaces `Host`

As before draft router doesn't proxy redirect and gone routes to a backend.

### Nginx

Router runs an nginx instance that proxies traffic to Router. The configuration for both live and draft stack live in [govuk-helm-charts](https://github.com/alphagov/govuk-helm-charts/blob/7a6e0b1e8964e2c25bf1539f048f9795ffb8629a/charts/app-config/templates/router-nginx-config.tpl)

The nginx instance also provides:
1. Healthcheck endpoints
2. Static error pages
3. `robots.txt` and `humans.txt`
4. Google Search Console verification files
5. Licensify endpoint

It also sets and hides some HTTP headers.

## API server

Router runs two HTTP servers:
1. Public server (default `:8080`) for handling requests
2. API server (default `:8081`) for admin operations

The API server exposes the following routes inside the cluster:
1. `/reload`
2. `/healthcheck`
3. `/memory-stats`
4. `/metrics`

## Configuration

Router is configured via environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `ROUTER_PUBADDR` | `:8080` | Public request server address |
| `ROUTER_APIADDR` | `:8081` | API/admin server address |
| `ROUTER_BACKEND_CONNECT_TIMEOUT` | `1s` | Backend connection timeout |
| `ROUTER_BACKEND_HEADER_TIMEOUT` | `20s` | Backend response header timeout |
| `ROUTER_FRONTEND_READ_TIMEOUT` | `60s` | Client request read timeout |
| `ROUTER_FRONTEND_WRITE_TIMEOUT` | `60s` | Client response write timeout |
| `ROUTER_ROUTE_RELOAD_INTERVAL` | `1m` | Periodic route reload interval |
| `ROUTER_TLS_SKIP_VERIFY` | unset | Skip TLS verification |
| `ROUTER_DEBUG` | unset | Enable debug logging |
| `ROUTER_ERROR_LOG` | `STDERR` | Error log file path |
| `ROUTER_ROUTES_FILE` | unset | Load routes from JSONL file instead of PostgreSQL |
| `CONTENT_STORE_DATABASE_URL` | unset | PostgreSQL connection string |
| `SENTRY_DSN` | unset | Sentry error tracking DSN |
| `SENTRY_ENVIRONMENT` | unset | Sentry environment tag |

Backend applications are configured with `BACKEND_URL_<backend_id>` environment variables:

```bash
export BACKEND_URL_frontend=http://localhost:3000
export BACKEND_URL_publisher=http://localhost:3001
```

Routes reference these backends by their ID (e.g., "frontend", "publisher").

### Serving routes from a flat file

When `ROUTER_ROUTES_FILE` is set, Router will load routes from the specified [JSONL file](https://jsonlines.org/) (one JSON object per line).
Router will also no longer load routes from PostgreSQL, and periodic route updates are disabled.

Example file:

```jsonl
{"BackendID":"frontend","IncomingPath":"/government","RouteType":"prefix","RedirectTo":null,"SegmentsMode":null,"SchemaName":null,"Details":null}
{"BackendID":null,"IncomingPath":"/old-page","RouteType":"exact","RedirectTo":"/new-page","SegmentsMode":"ignore","SchemaName":"redirect","Details":null}
{"BackendID":null,"IncomingPath":"/deleted","RouteType":"exact","RedirectTo":null,"SegmentsMode":null,"SchemaName":"gone","Details":null}
```

You can export routes from PostgreSQL to a JSONL file using:

```bash
ROUTER_ROUTES_FILE=/path/to/routes.jsonl ./router -export-routes
```

This can be used to continue serving routes when Content Store's database is down for maintenance.

For details on how to configure Router to load from a file see [docs/how-to-serve-routes-from-flat-file.md](docs/how-to-serve-routes-from-flat-file.md).

## Probe endpoints

Several probe endpoints are provided which allow you to test the functionality of router externally.

If no other routes are loaded, then none of the probe endpoint routes will be loaded and every request will return an HTTP 503 Service Unavailable..

When run with the [router-probe-backend](https://github.com/alphagov/govuk-helm-charts/tree/main/charts/router-probe-backend) backend service the available probes are:

Route                              | Provided by          | HTTP Response Expected
-----------------------------------|----------------------|------------
`/__probe__/gone`                  | router               | Returns 410 Gone status from internally within router
`/__probe__/router-redirect`       | router               | Returns 301 Moved Permenantly to `/__probe__/redirected`
`/__probe__/ok`                    | router-probe-backend | Returns 200 OK
`/__probe__/redirect`              | router-probe-backend | Returns 301 Moved Permenantly to `/__probe__/redirected`
`/__probe__/redirected`            | router-probe-backend | Returns 200 Ok
`/__probe__/not-found`             | router-probe-backend | Returns 404 Not Found
`/__probe__/internal-server-error` | router-probe-backend | Returns 500 Internal Server Error
`/__probe__/get`                   | router-probe-backend | Returns 200 Ok if the HTTP method is GET, otherwise returns 403 Forbidden
`/__probe__/post`                  | router-probe-backend | Returns 200 Ok if the HTTP method is POST, otherwise returns 403 Forbidden
`/__probe__/headers/get`           | router-probe-backend | Returns 200 Ok if the HTTP method is GET, with a JSON body which includes the key `requestHeaders` which is an Object of headers on the HTTP Request (with some values redacted)
`/__probe__/headers/post`          | router-probe-backend | Returns 200 Ok if the HTTP method is POST, with a JSON body which includes the key `requestHeaders` which is an Object of headers on the HTTP Request (with some values redacted)
`/__probe__/__canary__`            | router-probe-backend | Returns 200 Ok with a JSON body of `{"message": "Tweet tweet"}\n'}`

## Technical documentation

Recommended reading: [How to Write Go Code](https://golang.org/doc/code.html)

### Run the test suite

Checks run automatically on GitHub on PR and push. For faster feedback, you can
run the tests locally.

The lint check uses [golangci-lint](https://golangci-lint.run/), which you can
install via Homebrew or your favourite package manager:

```sh
brew install golangci-lint
```

You can run all tests (some of which need Docker installed) by running:

```
make test
```

You can also run just the unit tests or just the integration tests, using the
`unit_tests` and `integration_tests` targets. The unit tests don't need Docker.

The `trie` and `triemux` packages have unit tests. To run these on their own:

```
go test -bench=. ./trie ./triemux
```

The integration tests need Docker in order to run PostgreSQL. They are intended
to cover Router's overall request handling, error reporting and performance.

You can use `--ginkgo.focus <partial regex>` to run a subset of the integration
tests, for example:

```
go test ./integration_tests -v --ginkgo.focus 'redirect should preserve the query string'
```

See [Site tests](site_tests/README.md) on how to run the site tests.

### Debug output

To see debug messages when running tests, set both the `ROUTER_DEBUG` and
`ROUTER_DEBUG_TESTS` environment variables:

```sh
ROUTER_DEBUG=1 ROUTER_DEBUG_TESTS=1 make test
```

### Update the dependencies

This project uses [Go Modules](https://github.com/golang/go/wiki/Modules) to vendor its dependencies. To update the dependencies:

1. Update all the dependencies, including test dependencies, in your working copy:

   ```sh
   make update_deps
   ```

1. Check for any errors and commit.

   ```sh
   git commit -- go.{mod,sum} vendor
   ```

1. [Run the Router test suite](#run-the-test-suite). If you need to fix a
   failing test, keep your changes in separate commits to the `go get` /
   `go mod` commit.

1. Run the tests for all dependencies:

   ```sh
   go test all
   ```

   - If there are failures, look into each one and determine whether it needs
     fixing.
   - If anything under `vendor/` needs changing then either raise a PR with
     the upstream project or revert to a set of versions that work together.
     Only `go get` and `go mod` should touch files in `vendor/`.

1. Raise a PR.

### View the test coverage report

A coverage report is generated which includes both unit and integration test coverage merged into a single report.

Runing the make target `coverage_report` will generate a coverage report which covers everything if you have already run `unit_tests` and `integration_tests` (this doesn't happen automatically for the sake of GitHub actions CI checks):

```
make unit_tests
make integration_tests
make coverage_report
```

The end of the coverage_report output will be a high level package statement coverage percentage, as well as an overall coverage percentage:

```
$ make coverage_report
go tool covdata merge -i coverage/unit,coverage/integration/,coverage/version -o coverage/merged/
go tool covdata textfmt -i coverage/merged/ -o coverage/report/textfmt.txt
go tool cover -html coverage/report/textfmt.txt -o coverage/report/coverage.html
go tool cover -func=coverage/report/textfmt.txt | tail -n 1 > coverage/report/overall-coverage.txt
./coverage/generate-markdown-summary.sh


Overall Coverage: 84.8%

github.com/alphagov/router           coverage:  82.3%   of  statements
github.com/alphagov/router/handlers  coverage:  95.6%   of  statements
github.com/alphagov/router/lib       coverage:  80.8%   of  statements
github.com/alphagov/router/trie      coverage:  97.7%   of  statements
github.com/alphagov/router/triemux   coverage:  100.0%  of  statements
````

There is also a detailed textfmt report output into `coverage/report/textfmt.txt` which can be used by various other tooling.

Finally there's an HTML version of the report which allows you to see individaul lines coverage broken down by file, this is output `coverage/report/coverage.html`

### Further documentation

- [Data structure](docs/data-structure.md)
- [Original thinking behind the router](https://technology.blog.gov.uk/2013/12/05/building-a-new-router-for-gov-uk/)
- [Example of adding a metric](https://github.com/alphagov/router/commit/b443d3d) using the [Go prometheus client library](https://pkg.go.dev/github.com/prometheus/client_golang/prometheus)
- [Site tests](site_tests/README.md)
- [Triemux](triemux/README.md)
- [Trie](trie/README.md)

## Team

[GOV.UK Platform
Engineering](https://github.com/orgs/alphagov/teams/gov-uk-platform-engineering)
team looks after this repo. If you're inside GDS, you can find us in
[#govuk-platform-engineering] or view our [kanban
board](https://github.com/orgs/alphagov/projects/71).

## Licence

[MIT License](LICENCE)

[#govuk-platform-engineering]: https://gds.slack.com/channels/govuk-platform-engineering
[tm]: https://github.com/alphagov/router/tree/main/triemux
