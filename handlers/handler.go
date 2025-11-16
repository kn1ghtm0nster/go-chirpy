package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/kn1ghtm0nster/structs"
)

func ReadinessHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func ChirpValidationHandler(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)

	params := structs.Chirp{}
	err := decoder.Decode(&params)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		errorResponse := structs.ChirpError{
			Error: "Something went wrong",
		}
		json.NewEncoder(w).Encode(errorResponse)
		return
	}

	// if the chirp is longer than 140 characters, return an error
	if len(params.Body) > 140 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		errorResponse := structs.ChirpError{
			Error: "Chirp is too long",
		}
		json.NewEncoder(w).Encode(errorResponse)
		return
	}

	responseBody := structs.ValidChirp{
		Valid: true,
	}
	data, err := json.Marshal(responseBody)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		errorResponse := structs.ChirpError{
			Error: "Something went wrong",
		}
		json.NewEncoder(w).Encode(errorResponse)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}