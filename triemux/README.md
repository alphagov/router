triemux
=======

`triemux` is an implementation of [Go]'s [`http.Handler`][handler] that
multiplexes a set of other handlers onto a single path hierarchy, using an
efficient prefix trie, or "trie", to map request paths to handlers. It's
designed to be used as a first-line router, allowing different paths on the same
domain to be served by different applications.

API documentation for `triemux` can be found at [godoc.org][docs].

[handler]: https://pkg.go.dev/net/http#Handler
[Go]: https://go.dev/
[docs]: https://pkg.go.dev/github.com/alphagov/router/triemux

Install
-------

    go install github.com/alphagov/router/triemux

Usage
-----

    mux := triemux.NewMux()

    com := httputil.NewSingleHostReverseProxy(url.Parse("https://example.com"))
    org := httputil.NewSingleHostReverseProxy(url.Parse("https://example.org"))

    // Register a prefix route pointing to the "com" backend (all requests to
    // "/com<anything>" will go to this backend).
    mux.Handle("/com", true, com)

    // Register an exact (non-prefix) route pointing to the "org" backend.
    mux.Handle("/org", false, org)

    ...

    srv := &http.Server{
            Addr:         ":8080",
            Handler:      mux,
            ReadTimeout:  60 * time.Second(),
            WriteTimeout: 60 * time.Second(),
    }
    srv.ListenAndServe()

Licence
-------

`triemux` is released under the MIT licence, a copy of which can be found in `LICENCE`.
