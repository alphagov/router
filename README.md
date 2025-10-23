# Router

GOV.UK Router is an HTTP reverse proxy built on top of [`triemux`][tm]. It
loads a routing table into memory from a PostgreSQL database and:

- forwards requests to backend application servers according to the path in the
  request URL
- serves HTTP `301` and `302` redirects for moved content and short URLs
- serves `410 Gone` responses for resources that no longer exist

## How it works

Router loads its routing table from [Content Store's](https://github.com/alphagov/content-store/) PostgreSQL database (or optionally from a flat file). It uses a [trie data structure](https://en.wikipedia.org/wiki/Trie) for fast path lookups, maintaining two separate tries: one for exact path matches and one for prefix matches. When a request comes in, Router first checks for an exact match, then falls back to the longest prefix match.

Router can reload routes without restarting, either automatically via PostgreSQL's `LISTEN/NOTIFY`, on a periodic schedule, or manually via the API.

Routes can be one of two types:
- **exact**: The path must match exactly (e.g., `/government` matches only `/government`)
- **prefix**: The path prefix must match (e.g., `/government` matches `/government`, `/government/policies`, etc.)

Each matched route is handled by one of three handler types:
- **backend**: Reverse proxies the request to a backend application server
- **redirect**: Returns an HTTP 301 redirect to a new location
- **gone**: Returns an HTTP 410 Gone response for deleted content

Router runs two HTTP servers: a public server (default `:8080`) for handling requests, and an API server (default `:8081`) for admin operations like reloading routes and exposing metrics.

For details on the route data structure and handler configuration, see [docs/data-structure.md](docs/data-structure.md).

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

When `ROUTER_ROUTES_FILE` is set, Router will load routes from the specified JSONL file (one JSON object per line):

```jsonl
{"BackendID":"frontend","IncomingPath":"/government","RouteType":"prefix","RedirectTo":null,"SegmentsMode":null,"SchemaName":null,"Details":null}
{"BackendID":null,"IncomingPath":"/old-page","RouteType":"exact","RedirectTo":"/new-page","SegmentsMode":"ignore","SchemaName":"redirect","Details":null}
{"BackendID":null,"IncomingPath":"/deleted","RouteType":"exact","RedirectTo":null,"SegmentsMode":null,"SchemaName":"gone","Details":null}
```

You can export routes from PostgreSQL to JSONL format using:

```bash
./router -export-routes > routes.jsonl
```

This can be used to continue serving routes when Content Store's database is down for maintenance.

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

### Further documentation

- [Data structure](docs/data-structure.md)
- [Original thinking behind the router](https://technology.blog.gov.uk/2013/12/05/building-a-new-router-for-gov-uk/)
- [Example of adding a metric](https://github.com/alphagov/router/commit/b443d3d) using the [Go prometheus client library](https://pkg.go.dev/github.com/prometheus/client_golang/prometheus)

## Team

[GOV.UK Platform
Engineering](https://github.com/orgs/alphagov/teams/gov-uk-platform-engineering)
team looks after this repo. If you're inside GDS, you can find us in
[#govuk-platform-engineering] or view our [kanban
board](https://github.com/orgs/alphagov/projects/71).

## Licence

[MIT License](LICENCE)

[#govuk-platform-engineering]: https://gds.slack.com/channels/govuk-platform-engineering
[router-api]: https://github.com/alphagov/router-api
[tm]: https://github.com/alphagov/router/tree/main/triemux
