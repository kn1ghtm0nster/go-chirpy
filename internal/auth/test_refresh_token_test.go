package auth

import (
	"testing"
)

func TestMakeRefreshToken(t *testing.T) {
	token, err := MakeRefreshToken()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// ensure the token is 64 characters long (32 bytes in hex)
	if len(token) != 64 {
		t.Fatalf("Expected token length of 64, got %d", len(token))
	}
}