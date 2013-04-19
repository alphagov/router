package main

import (
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"runtime"
	"sync"
	"time"
)

var randomBody = flag.Bool("randomBody", false, "Whether or not to send large(r) randomly generated response bodies")

var quit = make(chan int)

type randomDataMaker struct {
	sync.Mutex
	src rand.Source
}

// Read will fill the byte slice with random ASCII bytes
func (r *randomDataMaker) Read(p []byte) (n int, err error) {
	for i := range p {
		p[i] = byte(r.src.Int63() & 0x7f) // 0x7f ensures the result is ASCII
	}
	return len(p), nil
}

func makeBenchmarkServer(name string) http.HandlerFunc {
	var randSrc = &randomDataMaker{src: rand.NewSource(time.Now().UTC().UnixNano())}

	return func(w http.ResponseWriter, r *http.Request) {
		randSrc.Lock()
		defer randSrc.Unlock()

		w.Header().Set("Content-Type", "text/plain")

		fmt.Fprintln(w, name, ":", r.URL.Path)
		fmt.Fprintln(w)

		// pick a size between 8 and 72KiB
		sz := 8192 + rand.Intn(1024*64)
		io.CopyN(w, randSrc, int64(sz))
	}
}

func makeDebugServer(name string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")

		fmt.Fprintln(w, name, ":", r.URL.Path)
	}
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	flag.Parse()

	startPort := 6061
	servers := []string{"one", "two", "three", "four", "five", "six"}

	for i, s := range servers {
		mux := http.NewServeMux()
		if *randomBody {
			mux.HandleFunc("/", makeBenchmarkServer(s))
		} else {
			mux.HandleFunc("/", makeDebugServer(s))
		}
		addr := fmt.Sprintf(":%d", startPort+i)
		go http.ListenAndServe(addr, mux)
		fmt.Println("Server", s, "listening on", addr)
	}

	<-quit
}
