router
======

This is an experimental HTTP(S) reverse proxy router built on top of
[`triemux`][tm]. It loads a routing table into memory from a MongoDB database,
and acts as a reverse proxy, serving responses from multiple backend servers on
a single domain.

Please note that this project is at a very early stage of development, so
probably shouldn't be used in production environments without extensive testing.

[tm]: https://github.com/nickstenning/router/tree/master/triemux

Build
-----

If you have a working [Go][go] development setup, you should be able to run:

    go install github.com/nickstenning/router
    $GOPATH/bin/router -h

If you've just checked out this repository and have the `go` tool on your $PATH,
you can just build the router in-place:

    go build

[go]: http://golang.org

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

The Router requires two MongoDB collections: `routes` and `applications`.

### Routes

The `routes` collection uses the following data structure:

```json
{
  "_id"           : ObjectId(),
  "RouteType"     : ["Prefix",nil],
  "IncomingPath"  : "/url-path/here",
  "ApplicationId" : "arbitrary-slug-or-name-from-applications-collection"
}
```

### Applications

The `applications` collection uses the following data structure:

```json
{
  "_id"           : ObjectId(),
  "ApplicationId" : "arbitrary-slug-or-name",
  "BackendURL"    : "https://example.com:port/"
}
```

License
-------

`router` is released under the MIT license, a copy of which can be found in
`LICENSE`.
