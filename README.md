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

1. Install the new version of Go on the CI machines (See [Why do we install Go on CI machines rather than Cache machines?](#why-do-we-install-go-on-ci-machines-rather-than-cache-machines)). You do this via puppet. See [here](https://github.com/alphagov/govuk-puppet/pull/11457/files) for an example PR. Once this PR has been approved and merged wait for puppet to run, or trigger a puppet run yourself as per [the developer docs](https://docs.publishing.service.gov.uk/manual/deploy-puppet.html#convergence).
2. Dependabot's PR will modify the Go version in the Dockerfile, but you also need to update the version number in the file `.go-version` See [here](https://github.com/alphagov/router/pull/241/files) for an example PR.
3. You will also have to update the Go version in `go.mod`. This will necessitate having Go installed on your local machine, changing the version number and running in terminal `go mod tidy` and `go mod vendor` in sequence to update correctly. See [example pr](https://github.com/alphagov/router/pull/307/commits/c0e4d753a48c71e84a3e4734389191e36bae9611). Also see [Upgrading Go Modules](#upgrading-go-modules).
4. Before you merge this PR, put the branch onto staging and leave it there for a couple of weekdays. Check for anything unexpected in icinga and sentry.
5. If you are confident that the version bump is safe for production, you can merge your PR and deploy it to production. It is best to do this at a quiet time of the day (such as 7am) to minimise any potential disruption.

#### Upgrading Go Modules

Sometimes modules will need to be manually upgraded after the above steps. This will satisfy dependencies that are old and do not use the `go.mod` file management system. Most likely you will see errors that require this when there is a failure to properly vendor `go.mod` due to an unsupported feature call in a dependency.

To do this, you'll require GoLang installed on your machine.

1. First, follow point 3 of the above [guide for upgrating](#updating-the-version-of-go) the version of Go.
2. If you determine through test failures that a module will need to be upgraded, in terminal at the root of `router` type in the following: `go get -u [repo-of-module]` - For example: `go get -u github.com/streadway/quantile`
3. Run `go mod tidy` and `go mod vendor`. Check for any errors and commit.

You may have to push your branch to see if this causes further errors on the Jenkins CI machines.

#### Why do we install Go on CI machines rather than Cache machines?

Router is built on CI machines; the artefact is then [uploaded to S3](https://github.com/alphagov/router/blob/main/Jenkinsfile#L68) for use in deployment. In  deployment, the artefact is [retrieved from S3](https://github.com/alphagov/govuk-app-deployment/blob/master/router%2Fconfig%2Fdeploy.rb#L30) and uploaded to the cache machines prior to a restart.

The updated version of golang is [only uploaded to Jenkins and CI agents](https://github.com/alphagov/govuk-puppet/search?q=golang).

### Further documentation

- [Data structure](docs/data-structure.md)
- [Original thinking behind the router](https://gdstechnology.blog.gov.uk/2013/12/05/building-a-new-router-for-gov-uk)
- [Example of adding a metric](https://github.com/alphagov/router/commit/b443d3dd9cf776143eed270d01bd98d2233caea6) using the [Go prometheus client library](https://godoc.org/github.com/dnesting/client_golang/prometheus)


## Licence

[MIT License](LICENCE)
