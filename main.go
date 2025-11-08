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
    mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("./assets"))))
	log.Println("Listening on port:", port)
	log.Fatal(server.ListenAndServe())
}