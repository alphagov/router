# router

This is a HTTP reverse proxy router built on top of [`triemux`][tm]. It
loads a routing table into memory from a MongoDB database and acts as a:

- Reverse proxy, forwarding requests to and serving responses from multiple
  backend servers on a single domain.
- Redirector, serving HTTP `301` and `302` redirects to new URLs.
- Gone responder, serving HTTP `410` responses for resources that used to
  but no longer exist.

The sister project [`router-api`][router-api] provides a read/write
interface to the underlying database and route reloading.

[tm]: https://github.com/alphagov/router/tree/master/triemux
[router-api]: https://github.com/alphagov/router-api

## Technical documentation

Recommended reading: [How to Write Go Code](https://golang.org/doc/code.html)

You can use the [GOV.UK Docker environment](https://github.com/alphagov/govuk-docker) to run the application and its tests with all the necessary dependencies. Follow [the usage instructions](https://github.com/alphagov/govuk-docker#usage) to get started.

**Use GOV.UK Docker to run any commands that follow.**

### Running the test suite

You can run all tests by running:

```
make test
```

The `trie` and `triemux` sub-packages have unit tests and benchmarks written
in Go's own testing framework. To run them individually:

```
go test -bench=. ./trie ./triemux
```

The `router` itself doesn't really benefit from having unit tests around
individual functions. Instead it has a comprehensive set of integration
tests to exercise it's HTTP handling, error reporting, and performance.

```
go test ./integration_tests
```

### Updating dependencies

This project uses [Go Modules](https://github.com/golang/go/wiki/Modules) to vendor its dependencies. To update the dependencies:

    go mod vendor

### Updating the version of Go

Dependabot raises PR's to update the dependencies for Router. This includes raising a PR when a new version of Go is available. However to update the version of Go, it's necessary to do more than just merge this dependabot PR. Here are the steps:

1. Install the new version of Go on the CI machines. You do this via puppet. See [here](https://github.com/alphagov/govuk-puppet/pull/11457/files) for an example PR. Once this PR has been approved and merged wait for puppet to run, or trigger a puppet run yourself as per [the developer docs](https://docs.publishing.service.gov.uk/manual/deploy-puppet.html#convergence).
2. Dependabot's PR will modify the Go version in the Dockerfile, but you also need to update the version number in the file `.go-version` See [here](https://github.com/alphagov/router/pull/241/files) for an example PR.
3. Before you merge this PR, put the branch onto staging and leave it there for a couple of weekdays. Check for anything unexpected in icinga and sentry.
4. If you are confident that the version bump is safe for production, you can merge your PR and deploy it to production. It is best to do this at a quiet time of the day (such as 7am) to minimise any potential disruption.

### Further documentation

- [Data structure](docs/data-structure.md)
- [Original thinking behind the router](https://gdstechnology.blog.gov.uk/2013/12/05/building-a-new-router-for-gov-uk)
- [Example of adding a metric](https://github.com/alphagov/router/commit/b443d3dd9cf776143eed270d01bd98d2233caea6) using the [Go prometheus client library](https://godoc.org/github.com/dnesting/client_golang/prometheus)

## License

[MIT Licence](LICENSE)
