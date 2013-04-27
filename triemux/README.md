proxymux
========

`proxymux` is an implementation of [Go][go]'s [`http.Handler`][handler] that
multiplexes a set of backend servers onto a single path hierarchy and acts as a
reverse proxy. It's designed to be used as a first-line router, allowing
different paths on the same domain to be served by different applications.

API documentation for `proxymux` can be found at [godoc.org][docs].

[handler]: http://golang.org/pkg/net/http/#Handler
[go]: http://golang.org
[docs]: http://godoc.org/github.com/nickstenning/proxymux

Install
-------

    go install github.com/nickstenning/proxymux

Usage
-----

    mux := proxymux.NewMux()

    goog := mux.AddBackend(url.Parse("http://google.com"))
    aapl := mux.AddBackend(url.Parse("http://google.com"))

    // register a prefix route pointing to the Google backend (all requests to
    // "/google<anything>" will go to this backend)
    mux.Register("/google", true, goog)

    // register an exact (non-prefix) route pointing to the Apple backend
    mux.Register("/apple", false, aapl)

    ...

    http.ListenAndServe(":8080", mux)

License
-------

`proxymux` is released under the MIT license, a copy of which can be found in
`LICENSE`.
