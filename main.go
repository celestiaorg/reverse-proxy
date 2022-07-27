package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/AdamSLevy/jsonrpc2"
)

const (
	noopURL = "http://ethermint0:8545"
	swapURL = "http://proxy:8080"
)

type baseHandle struct{}

func (h *baseHandle) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("%s %s %s\n", r.RemoteAddr, r.Method, r.URL)

	// Read all bytes of HTTP request body.
	reqBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.Write([]byte("500 ReadAll"))
		return
	}

	// Check for JSON parsing issues.
	if !json.Valid(reqBytes) {
		w.Write([]byte("500 Invalid JSON"))
		return
	}
	rawReqs := make([]json.RawMessage, 1)
	if json.Unmarshal(reqBytes, &rawReqs) != nil {
		rawReqs[0] = json.RawMessage(reqBytes)
	}
	// Catch empty batch requests.
	if len(rawReqs) == 0 {
		w.Write([]byte("500"))
	}
	if len(rawReqs) > 1 {
		// Batch Request
		remoteUrl, err := url.Parse(noopURL)
		if err != nil {
			log.Println("target parse fail:", err)
			return
		}
		proxy := httputil.NewSingleHostReverseProxy(remoteUrl)
		r.Body = ioutil.NopCloser(bytes.NewBuffer(reqBytes))
		log.Printf("proxy to %s\n", remoteUrl)
		proxy.ServeHTTP(w, r)
		return
	}
	var req jsonrpc2.Request
	if err := json.Unmarshal(rawReqs[0], &req); err != nil {
		w.Write([]byte("500 rawReqs[0]"))
	}
	switch meth := req.Method; meth {
	case "eth_getBlockByHash", "eth_getBlockByNumber":
		remoteUrl, err := url.Parse(swapURL)
		if err != nil {
			log.Println("target parse fail:", err)
			return
		}
		proxy := httputil.NewSingleHostReverseProxy(remoteUrl)
		r.Body = ioutil.NopCloser(bytes.NewBuffer(reqBytes))
		log.Printf("proxy to %s\n", remoteUrl)
		proxy.ServeHTTP(w, r)
		return
	default:
		remoteUrl, err := url.Parse(noopURL)
		if err != nil {
			log.Println("target parse fail:", err)
			return
		}
		proxy := httputil.NewSingleHostReverseProxy(remoteUrl)
		r.Body = ioutil.NopCloser(bytes.NewBuffer(reqBytes))
		log.Printf("proxy to %s\n", remoteUrl)
		proxy.ServeHTTP(w, r)
		return
	}
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
