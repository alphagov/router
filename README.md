router
======

This is an experimental HTTP(S) reverse proxy router built on top of
[`triemux`][tm]. It loads a routing table into memory from a MongoDB database,
and acts as a reverse proxy, serving responses from multiple backend servers on
a single domain.

Please note that this project is at a very early stage of development, so
probably shouldn't be used in production environments without extensive testing.

[tm]: https://github.com/alphagov/router/tree/master/triemux

Build
-----

If you have a working [Go][go] development setup, you should be able to run:

    go install github.com/alphagov/router
    $GOPATH/bin/router -h

If you've just checked out this repository and have the `go` tool on your $PATH,
you can just build the router in-place:

    go build

[go]: http://golang.org

Environment assumptions
-----------------------

Our usage of `router` places it behind and in front of Nginx and/or Varnish.

As such, there are some things that we are guarded against:

- Response buffering for slow clients
- Basic request sanitisation

And some features that we have no need to implement:

- SSL
- Health check probes
- Custom header mangling
- Response rewriting
- Authentication

Benchmarks
----------

If you have a local mongo instance, you can load a set of routes which are
useful for running benchmark tools with

    ./tools/benchsetup

You can then run a test backend with

    go run testserver/testserver.go -randomBody

And start the router against the benchmark database with

    go run *.go -mongoDbName=routerbench

And then benchmark against the URLs in `testdata/benchurls`.

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
  "handler"       : ["backend", "redirect"],
}
```

The behaviour is determined by `handler`. See below for extra fields
corresponding to `handler` types.

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
