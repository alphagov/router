router
======

This is a HTTP reverse proxy router built on top of [`triemux`][tm]. It
loads a routing table into memory from a MongoDB database and acts as a:

- Reverse proxy, forwarding requests to and serving responses from multiple
  backend servers on a single domain.
- Redirector, serving HTTP `301` and `302` redirects to new URLs.
- Gone responder, serving HTTP `410` responses for resources that used to
  but no longer exist.

The sister project [`router-api`][router-api] provides a read/write
interface to the underlying database and route reloading.

Some of the thinking behind the router is documented in [this 2013 blog post][post].

[tm]: https://github.com/alphagov/router/tree/master/triemux
[router-api]: https://github.com/alphagov/router-api
[post]: https://gdstechnology.blog.gov.uk/2013/12/05/building-a-new-router-for-gov-uk/

Environment assumptions
-----------------------

Our usage of `router` places it behind and in front of Nginx and/or Varnish.

As such, there are some things that we are guarded against:

- Response buffering for slow clients
- Basic request sanitisation

And some features that we have no need to implement:

- Access logging (but error logging is implemented)
- SSL
- Health check probes
- Custom header mangling
- Response rewriting
- Authentication

Local Setup & Build
-------------------

The router needs to be built from within a Go [workspace][workspace] for it to
correctly identify the vendored dependencies.  Assuming you already have Go and
a `$GOPATH` set up, you can simply run `go get github.com/alphagov/router` to
pull the source into your workspace.

Once you have the code set up correctly, building and running the application
is simple:

    make
    ./router -h

[workspace]: https://golang.org/doc/code.html#Workspaces

Tests
-----

You can run all tests by running:

    make test

The `trie` and `triemux` sub-packages have unit tests and benchmarks written
in Go's own testing framework. To run them individually:

    go test -bench=. ./trie ./triemux

The `router` itself doesn't really benefit from having unit tests around
individual functions. Instead it has a comprehensive set of integration
tests to exercise it's HTTP handling, error reporting, and performance.

These require a local MongoDB instance and can be run with:

    go test ./integration_tests

Some of the integration tests are optional because they have certain
environment requirements that make them unfeasible to run within CI.

Dependencies
------------

This project uses [Go Modules](https://github.com/golang/go/wiki/Modules) to vendor its dependencies.
If you have a working [Go][go] development setup, you should be able to update the dependencies via:

    go mod vendor

Data structure
-----------------

The Router requires two MongoDB collections: `routes` and `backends`.

### Routes

The `routes` collection uses the following data structure:

```json
{
  "_id"           : ObjectId(),
  "route_type"    : ["prefix","exact"],
  "incoming_path" : "/url-path/here",
  "handler"       : ["backend", "redirect", "gone"],
  "disabled"      : false
}
```

Incoming paths with special characters must be in their % encoded form in the
database (eg spaces must be stored as `%20`).

The behaviour of an enabled route is determined by `handler`. See below for
extra fields corresponding to `handler` types.

If a route is disabled, the router will return a 503 for all matching requests.
This is typically used if a service needs to be taken offline for maintenance
etc.

#### `backend` handler

The `backend` handler causes the Router to reverse proxy to a named
`backend`. The following extra fields are supported:

```json
{
  "backend_id" : "backend-id-corresponding-to-backends-collection"
}
```

#### `redirect` handler

The `redirect` handler causes the Router to redirect the given
`incoming_path` to the path stored in `redirect_to`. The following
extra fields are supported:

```json
{
  "redirect_to"   : "/target-of-redirect",
  "redirect_type" : ["permanent", "temporary"]
}
```

#### `gone` handler

The `gone` handler causes the Router to return a 410 response.

### Backends

The `backends` collection uses the following data structure:

```json
{
  "_id"         : ObjectId(),
  "backend_id"  : "arbitrary-slug-or-name",
  "backend_url" : "https://example.com:port/"
}
```

License
-------

`router` is released under the MIT license, a copy of which can be found in
`LICENSE`.
