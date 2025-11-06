package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	port := 8080
	mux := http.NewServeMux()
	server := &http.Server{
		Handler: mux,
		Addr: fmt.Sprintf(":%d", port),
	}
	mux.Handle("/", http.FileServer(http.Dir(".")))
	log.Println("Listening on port:", port)
	log.Fatal(server.ListenAndServe())
}