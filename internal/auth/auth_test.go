package auth

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
)

func TestMakeAndValidateJWT(t *testing.T) {
	userID := uuid.New()
	secret := "supersecret"

	token, err := MakeJWT(userID, secret)
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

	token, err := MakeJWT(userID, secret)
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
	token, err := MakeJWT(userID, secret)
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

func TestGetBearerToken(t *testing.T) {
	t.Run("valid bearer token", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("Authorization", "Bearer abc123")

		token, err := GetBearerToken(headers)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if token != "abc123" {
			t.Fatalf("expected abc123, got %s", token)
		}
	})

	t.Run("missing Authorization header", func(t *testing.T) {
		headers := http.Header{}

		_, err := GetBearerToken(headers)
		if err == nil {
			t.Fatalf("expected error, got none")
		}
	})

	t.Run("wrong prefix", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("Authorization", "Token abc123")

		_, err := GetBearerToken(headers)
		if err == nil {
			t.Fatalf("expected error for wrong prefix, got none")
		}
	})

	t.Run("empty bearer after prefix", func(t *testing.T) {
		headers := http.Header{}
		headers.Set("Authorization", "Bearer ")

		_, err := GetBearerToken(headers)
		if err == nil {
			t.Fatalf("expected error for empty token, got none")
		}
	})
}