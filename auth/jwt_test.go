package auth

import (
	"testing"

	"github.com/alecthomas/assert/v2"
	"github.com/google/uuid"
)

func TestTokenCreation(t *testing.T) {
	tok := NewT()
	userID := uuid.NewString()
	tokenString, err := tok.Create(userID)
	if err != nil {
		t.Fatalf("create token failed: %v", err)
	}

	assert.NotEqual(t, "", tokenString)
	assert.Equal(t, tok.Jwt, tokenString)
}

func TestTokenVerification(t *testing.T) {
	tok := NewT()
	userID := uuid.NewString()
	tokenString, err := tok.Create(userID)
	if err != nil {
		t.Fatalf("create token failed: %v", err)
	}

	id, err := tok.Verify(tokenString)
	if err != nil {
		t.Fatalf("token validation failed: %v", err)
	}

	assert.NotEqual(t, "", id)
	assert.Equal(t, userID, id)
}

func TestTokenExtraction(t *testing.T) {
	tok := NewT()
	userID := uuid.NewString()
	rawToken, err := tok.Create(userID)
	if err != nil {
		t.Fatalf("create token failed: %v", err)
	}
	fakeHeader := "Bearer " + rawToken
	tokenString, err := tok.Extract(fakeHeader)
	if err != nil {
		t.Fatalf("token extraction failed: %v", err)
	}
	assert.NotContains(t, tokenString, "Bearer")
	assert.NotEqual(t, "", tokenString)
}
