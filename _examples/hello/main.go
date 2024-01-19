package main

import (
	"fmt"
	"log"
	"net/http"

	"go.destructure.dev/webmux"
)

func main() {
	mux := webmux.NewMux()

	greet := func(w http.ResponseWriter, r *http.Request) error {
		m, _ := webmux.FromContext(r.Context())

		name := m.Param("name")

		_, err := fmt.Fprintf(w, "Hello, %s!", name)

		return err
	}

	mux.HandleFunc(http.MethodGet, "/greet/:name", greet)

	log.Fatal(http.ListenAndServe(":3030", mux))
}
