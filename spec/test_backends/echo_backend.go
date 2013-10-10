package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

var port = flag.Int("port", 3160, "The port to listen on")

func echoResponse(w http.ResponseWriter, r *http.Request) {
	// Simulate a Via response header if given this query param
	if via := r.URL.Query().Get("simulate_response_via"); via != "" {
		w.Header().Set("Via", via)
	}

	data := make(map[string]interface{})
	data["Request"] = r

	body, _ := ioutil.ReadAll(r.Body)
	data["Body"] = string(body)

	json_data, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	json_data = append(json_data, 10) // Add final newline
	w.Write(json_data)
}

func main() {
	flag.Parse()

	http.HandleFunc("/", echoResponse)

	addr := fmt.Sprintf(":%d", *port)
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
