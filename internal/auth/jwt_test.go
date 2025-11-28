package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestMakeAndValidateJWT(t *testing.T) {
	userID := uuid.New()
	secret := "supersecret"
	expires := time.Minute * 5

	token, err := MakeJWT(userID, secret, expires)
	if err != nil {
		t.Fatalf("MakeJWT returned error: %v", err)
	}
	if token == "" {
		t.Fatalf("MakeJWT returned empty token")
	}

	// Validate
	parsedID, err := ValidateJWT(token, secret)
	if err != nil {
		t.Fatalf("ValidateJWT returned error: %v", err)
	}

	if parsedID != userID {
		t.Fatalf("expected userID %s, got %s", userID, parsedID)
	}
}

func TestValidateJWT_WrongSecret(t *testing.T) {
	userID := uuid.New()
	secret := "correctsecret"
	badSecret := "wrongsecret"

	token, err := MakeJWT(userID, secret, time.Minute)
	if err != nil {
		t.Fatalf("MakeJWT returned error: %v", err)
	}

	_, err = ValidateJWT(token, badSecret)
	if err == nil {
		t.Fatalf("expected error when using wrong secret, got nil")
	}
}

func TestValidateJWT_ExpiredToken(t *testing.T) {
	userID := uuid.New()
	secret := "mysecret"

	// Create token that expired 1 second ago
	token, err := MakeJWT(userID, secret, -1*time.Second)
	if err != nil {
		t.Fatalf("MakeJWT returned error: %v", err)
	}

	_, err = ValidateJWT(token, secret)
	if err == nil {
		t.Fatalf("expected error for expired token, got nil")
	}
}

func TestValidateJWT_InvalidTokenString(t *testing.T) {
	secret := "secret"

	_, err := ValidateJWT("this-is-not-a-token", secret)
	if err == nil {
		t.Fatalf("expected error for invalid token string")
	}
}