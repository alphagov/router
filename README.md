# Router

GOV.UK Router is an HTTP reverse proxy built on top of [`triemux`][tm]. It
loads a routing table into memory from a Postgres database (in practice,
the database created and maintained by [Content Store][content-store]
and:

- forwards requests to backend application servers according to the path in the
  request URL
- serves HTTP `301` and `302` redirects for moved content and short URLs
- serves `410 Gone` responses for resources that no longer exist

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

The integration tests need Docker in order to run Postgres. They are intended
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
[content-store]: https://github.com/alphagov/content-store
[tm]: https://github.com/alphagov/router/tree/main/triemux
