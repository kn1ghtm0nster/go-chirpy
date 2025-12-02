package auth

import (
	"testing"
)

func TestHashPassword(t *testing.T) {
	password := "mySecurePassword"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword returned an error: %v", err)
	}

	if hash == "" {
		t.Fatal("HashPassword returned an empty hash")
	}
}

func TestCheckPasswordHash_Correct(t *testing.T) {
	password := "mySecurePassword"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword returned an error: %v", err)
	}

	match, err := CheckPasswordHash(password, hash)
	if err != nil {
		t.Fatalf("CheckPasswordHash returned an error: %v", err)
	}

	if !match {
		t.Fatal("CheckPasswordHash returned false for correct password")
	}
}

func TestCheckPasswordHash_Incorrect(t *testing.T) {
	password := "mySecurePassword"
	wrongPassword := "wrongPassword"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword returned an error: %v", err)
	}

	// verify wrong password
	match, err := CheckPasswordHash(wrongPassword, hash)
	if err != nil {
		t.Fatalf("CheckPasswordHash returned an error: %v", err)
	}

	if match {
		t.Fatal("CheckPasswordHash returned true for incorrect password")
	}
}

func TestCheckPasswordHash_InvalidHash(t *testing.T) {
	password := "mySecurePassword"
	invalidHash := "invalidHashString"
	
	_, err := CheckPasswordHash(password, invalidHash)
	if err == nil {
		t.Fatal("CheckPasswordHash did not return an error for invalid hash")
	}
}