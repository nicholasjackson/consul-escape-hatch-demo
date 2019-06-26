package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		log.Println("Calling upstream")

		resp, err := http.Get("http://localhost:9001")
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
	})

	http.ListenAndServe(":9000", nil)
}
