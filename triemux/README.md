triemux
=======

`triemux` is an implementation of [Go][go]'s [`http.Handler`][handler] that
multiplexes a set of other handlers onto a single path hierarchy, using an
efficient prefix trie, or "trie", to map request paths to handlers. It's
designed to be used as a first-line router, allowing different paths on the same
domain to be served by different applications.

API documentation for `triemux` can be found at [godoc.org][docs].

[handler]: http://golang.org/pkg/net/http/#Handler
[go]: http://golang.org
[docs]: http://godoc.org/github.com/nickstenning/router/triemux

Install
-------

    go install github.com/nickstenning/router/triemux

Usage
-----

    mux := triemux.NewMux()

    goog := httputil.NewSingleHostReverseProxy(url.Parse("http://google.com"))
    aapl := httputil.NewSingleHostReverseProxy(url.Parse("http://apple.com"))

    // register a prefix route pointing to the Google backend (all requests to
    // "/google<anything>" will go to this backend)
    mux.Handle("/google", true, goog)

    // register an exact (non-prefix) route pointing to the Apple backend
    mux.Handle("/apple", false, aapl)

    ...

    http.ListenAndServe(":8080", mux)

License
-------

`triemux` is released under the MIT license, a copy of which can be found in
`LICENSE`.
