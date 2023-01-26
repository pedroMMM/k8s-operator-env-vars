package main

import (
	"fmt"
	"net/http"
	"os"
)

func main() {
	http.HandleFunc("/", envVars)
	http.ListenAndServe(":8080", nil)
}

func envVars(w http.ResponseWriter, req *http.Request) {
	for _, e := range os.Environ() {
		fmt.Fprintln(w, e)
	}
}
