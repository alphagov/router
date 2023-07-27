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

1. Run `go mod tidy && go mod vendor`.
1. Check for any errors and commit.

Occasionally an old module may need updating explicitly via `go get -u
<repo-of-module>`, for example `go get -u github.com/streadway/quantile`


### Further documentation

- [Data structure](docs/data-structure.md)
- [Original thinking behind the router](https://gdstechnology.blog.gov.uk/2013/12/05/building-a-new-router-for-gov-uk)
- [Example of adding a metric](https://github.com/alphagov/router/commit/b443d3d) using the [Go prometheus client library](https://godoc.org/github.com/dnesting/client_golang/prometheus)

## Licence

[MIT License](LICENCE)
