-# router

This is a HTTP reverse proxy router built on top of [`triemux`][tm]. It
loads a routing table into memory from a MongoDB database and acts as a:

- Reverse proxy, forwarding requests to and serving responses from multiple
  backend servers on a single domain.
- Redirector, serving HTTP `301` and `302` redirects to new URLs.
- Gone responder, serving HTTP `410` responses for resources that used to
  but no longer exist.

The sister project [`router-api`][router-api] provides a read/write
interface to the underlying database.

[tm]: https://github.com/alphagov/router/tree/master/triemux
[router-api]: https://github.com/alphagov/router-api

## Technical documentation

Recommended reading: [How to Write Go Code](https://golang.org/doc/code.html)

### Running the test suite

You can run all tests (some of which need Docker installed) by running:

```
make test
```

You can also run just the unit tests or just the integration tests, using the
`unit_tests` and `integration_tests` targets. The unit tests don't need Docker.

The `trie` and `triemux` sub-packages have unit tests and benchmarks written
in Go's own testing framework. To run them individually:

```
go test -bench=. ./trie ./triemux
```

The integration tests require Docker in order to run MongoDB. They are intended
to cover Router's overall request handling, error reporting and performance.

```
go test ./integration_tests
```

### Lint

Checks run automatically on GitHub on PR and push. For faster feedback, you can
install and run the [linter](https://golangci-lint.run/) yourself, or configure
your editor/IDE to do so. For example:

```sh
brew install golangci-lint
```

```sh
make lint
```

### Debug output

To see debug messages when running tests, set both the `DEBUG` and
`DEBUG_ROUTER` environment variables.

```sh
export DEBUG=1 DEBUG_ROUTER=1
```

or equivalently for a single run:

```sh
DEBUG=1 DEBUG_ROUTER=1 make test
```

### Updating dependencies

This project uses [Go Modules](https://github.com/golang/go/wiki/Modules) to vendor its dependencies. To update the dependencies:

    go mod vendor

### Updating the version of Go

Dependabot raises PR's to update the dependencies for Router. This includes raising a PR when a new version of Go is available. However to update the version of Go, it's necessary to do more than just merge this dependabot PR. Here is an [example PR](https://github.com/alphagov/router/pull/345/files) with all the below changes, and here are the steps:

1. Dependabot's PR will modify the Go version in the Dockerfile (and thus what is build in the Kubernetes engine), but you also need to update the version number in the file `.go-version`.
2. You will also have to update the Go version in `go.mod`. This will necessitate having Go installed on your local machine, changing the version number and running in terminal `go mod tidy` and `go mod vendor` in sequence to update correctly. This may have no changes at all, but see [example pr](https://github.com/alphagov/router/pull/307/commits/c0e4d753a48c71e84a3e4734389191e36bae9611) for a larger update. Also see [Upgrading Go Modules](#upgrading-go-modules).
3. Finally you need to update the go version in `ci.yml`.
4. Before you merge this PR, put the branch onto staging and leave it there for a couple of weekdays. Check for anything unexpected in icinga and sentry.
5. If you are confident that the version bump is safe for production, you can merge your PR and deploy it to production. It is best to do this at a quiet time of the day (such as 7am) to minimise any potential disruption.
6. Make sure govuk-docker is updated to match the new version. See [here](https://github.com/alphagov/govuk-docker/pull/643/files).

#### Upgrading Go Modules

Sometimes modules will need to be manually upgraded after the above steps. This will satisfy dependencies that are old and do not use the `go.mod` file management system. Most likely you will see errors that require this when there is a failure to properly vendor `go.mod` due to an unsupported feature call in a dependency.

To do this, you'll require GoLang installed on your machine.

1. First, follow point 3 of the above [guide for upgrating](#updating-the-version-of-go) the version of Go.
2. If you determine through test failures that a module will need to be upgraded, in terminal at the root of `router` type in the following: `go get -u [repo-of-module]` - For example: `go get -u github.com/streadway/quantile`
3. Run `go mod tidy` and `go mod vendor`. Check for any errors and commit.

### Further documentation

- [Data structure](docs/data-structure.md)
- [Original thinking behind the router](https://gdstechnology.blog.gov.uk/2013/12/05/building-a-new-router-for-gov-uk)
- [Example of adding a metric](https://github.com/alphagov/router/commit/b443d3dd9cf776143eed270d01bd98d2233caea6) using the [Go prometheus client library](https://godoc.org/github.com/dnesting/client_golang/prometheus)


## Licence

[MIT License](LICENCE)
