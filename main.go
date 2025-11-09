package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/kn1ghtm0nster/handlers"
)

func main() {
	port := 8080
	mux := http.NewServeMux()
	server := &http.Server{
		Handler: mux,
		Addr: fmt.Sprintf(":%d", port),
	}
	mux.HandleFunc("/healthz", handlers.ReadinessHandler)
	mux.Handle("/app/", http.StripPrefix("/app/", http.FileServer(http.Dir("."))))
    mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("./assets"))))
	log.Println("Listening on port:", port)
	log.Fatal(server.ListenAndServe())
}