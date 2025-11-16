package structs

type Chirp struct {
	Body string `json:"body"`
}


type ChirpError struct {
	Error string `json:"error"`
}


type ValidChirp struct {
	Valid bool `json:"valid"`
}