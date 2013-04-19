package main

import (
	"flag"
	"fmt"
	"github.com/nickstenning/govmux/proxymux"
	"net/http"
	"net/url"
	"os"
)

var addr = flag.String("extAddr", ":1250", "Address on which to serve requests")

var quit = make(chan int)

func main() {
	flag.Parse()

	p := proxymux.NewProxyMux()
	b1, _ := url.Parse("http://localhost:6061/proxyprefix")
	i1 := p.AddBackend(b1)
	b2, _ := url.Parse("http://localhost:6062")
	i2 := p.AddBackend(b2)
	p.Register("/foo/a", false, i1)
	p.Register("/foo/b", false, i1)
	p.Register("/bar", true, i2)

	fmt.Fprintln(os.Stderr, "Listening for requests on "+*addr)

	http.ListenAndServe(*addr, p)
}
