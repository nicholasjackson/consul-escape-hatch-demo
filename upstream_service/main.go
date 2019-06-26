package main

import (
	"fmt"
	"math/rand"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {

		// randomly fail
		if rand.Intn(2) == 0 {
			fmt.Println("Randomly erroring")
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		// return ok
		fmt.Fprintf(rw, "request ok from upstream")
	})

	http.ListenAndServe(":9001", nil)
}
