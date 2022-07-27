package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"

	"github.com/AdamSLevy/jsonrpc2"
)

const noop = "noop"
const swap = "swap"

var (
	hostTarget = map[string]string{
		swap: "http://localhost:8080",
		noop: "http://localhost:8545",
	}
	hostProxy map[string]*httputil.ReverseProxy = map[string]*httputil.ReverseProxy{}
)

type baseHandle struct{}

func respond(w http.ResponseWriter, res interface{}) {
	enc := json.NewEncoder(w)
	if err := enc.Encode(res); err != nil {
		// We should never have an error encoding our Response.
		panic(err)
	}
}

func (h *baseHandle) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Read all bytes of HTTP request body.
	reqBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.Write([]byte("500"))
		// respondError(w, InvalidRequest)
		return
	}

	// Check for JSON parsing issues.
	if !json.Valid(reqBytes) {
		// respondError(w, ParseError)
		return
	}
	rawReqs := make([]json.RawMessage, 1)
	if json.Unmarshal(reqBytes, &rawReqs) != nil {
		// Since the JSON is valid, this Unmarshal error indicates that
		// this is just a single request.
		rawReqs[0] = json.RawMessage(reqBytes)
	}
	// Catch empty batch requests.
	if len(rawReqs) == 0 {
		w.Write([]byte("500"))
	}
	if len(rawReqs) > 1 {
		if fn, ok := hostProxy[noop]; ok {
			fn.ServeHTTP(w, r)
			return
		}
	}
	var req jsonrpc2.Request
	if err := json.Unmarshal(rawReqs[0], &req); err != nil {
		w.Write([]byte("500"))
	}
	if req.Method == "eth_getBlockByHash" {
		if fn, ok := hostProxy[swap]; ok {
			fn.ServeHTTP(w, r)
			return
		}
	}
	w.Write([]byte("500"))

}

func main() {

	h := &baseHandle{}
	http.Handle("/", h)

	server := &http.Server{
		Addr:    ":8082",
		Handler: h,
	}
	log.Fatal(server.ListenAndServe())
}
