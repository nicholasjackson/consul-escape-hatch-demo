package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"time"
)

var serviceType = flag.String("type", "downstream", "upstream or downstream service type")
var upstreamURI = flag.String("upstream-uri", "localhost:9000", "URI for upstream service")
var bindAdddress = flag.String("bind-address", ":9090", "Bind address for the service")
var upstreamErrors = flag.Float64("upstream-errors", 0, "Decimal percentage of errors")

func main() {
	flag.Parse()

	handler := downstream
	if *serviceType == "upstream" {
		handler = upstream
	}

	http.HandleFunc("/", handler)

	log.Println("Starting service on", *bindAdddress)
	http.ListenAndServe(*bindAdddress, nil)
}

func downstream(rw http.ResponseWriter, r *http.Request) {
	log.Println("Calling upstream")

	resp, err := http.Get(*upstreamURI)
	if err != nil {
		log.Println("Error calling upstream", err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(rw, "Received error %d from server", resp.StatusCode)
		return
	}

	data, _ := ioutil.ReadAll(resp.Body)
	fmt.Fprintf(rw, "Response %s", string(data))
}

func upstream(rw http.ResponseWriter, r *http.Request) {
	log.Println("Got request")

	errNum := rand.Intn(100)
	if *upstreamErrors*100.0 > float64(errNum) {
		log.Println("throwing error")
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	time.Sleep(100 * time.Millisecond)

	// return ok
	fmt.Fprintf(rw, "request ok from upstream")
}
