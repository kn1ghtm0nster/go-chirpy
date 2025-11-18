package utils

import (
	"strings"
)


func CleanProfanity(chirp string) string {
    bannedWords := []string{"kerfuffle", "sharbert", "fornax"}
    chirpWords := strings.Split(chirp, " ")

    for i, word := range chirpWords {
        lowercaseWord := strings.ToLower(word)
        for _, bannedWord := range bannedWords {
            if lowercaseWord == bannedWord {
                chirpWords[i] = "****"
            }
        }
    }

    cleanedChirp := strings.Join(chirpWords, " ")
    return cleanedChirp
}