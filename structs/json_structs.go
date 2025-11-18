package structs

type Chirp struct {
	Body string `json:"body"`
}


type ChirpError struct {
	Error string `json:"error"`
}


type ValidChirp struct {
	CleanedBody string `json:"cleaned_body"`
}