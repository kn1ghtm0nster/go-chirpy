package main

import (
	"fmt"
	"log"
	"net/http"
	"sync/atomic"

	"github.com/kn1ghtm0nster/handlers"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
	
}

func (cfg *apiConfig) metricsHandler(w http.ResponseWriter, r *http.Request) {
	currCount := cfg.fileserverHits.Load()
	payload := fmt.Sprintf(`
		<html>
			<body>
				<h1>Welcome, Chirpy Admin</h1>
				<p>Chirpy has been visited %d times!</p>
			</body>
		</html>`, currCount)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(payload))
}


func (cfg *apiConfig) resetMetricsHandler(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits.Store(0)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK\n"))
}

func main() {
	port := 8080
	mux := http.NewServeMux()
	server := &http.Server{
		Handler: mux,
		Addr: fmt.Sprintf(":%d", port),
	}
	apiConfig := &apiConfig{
		fileserverHits: atomic.Int32{},
	}

	mux.HandleFunc("GET /api/healthz", handlers.ReadinessHandler)
	mux.HandleFunc("POST /admin/reset", apiConfig.resetMetricsHandler)
	mux.HandleFunc("GET /admin/metrics", apiConfig.metricsHandler)
	mux.Handle("/app/", http.StripPrefix("/app/", apiConfig.middlewareMetricsInc(http.FileServer(http.Dir(".")))))
    mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("./assets"))))
	log.Println("Listening on port:", port)
	log.Fatal(server.ListenAndServe())
}