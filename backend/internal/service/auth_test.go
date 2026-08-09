package service

import "testing"

func TestAuthServiceGenerateAndParseToken(t *testing.T) {
	auth := NewAuthService("test-secret")

	token, err := auth.GenerateToken(42)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	userID, err := auth.ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken() error = %v", err)
	}
	if userID != 42 {
		t.Fatalf("ParseToken() userID = %d, want 42", userID)
	}
}

func TestAuthServiceRejectsTokenSignedWithDifferentSecret(t *testing.T) {
	auth := NewAuthService("test-secret")
	other := NewAuthService("other-secret")

	token, err := other.GenerateToken(42)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	if _, err := auth.ParseToken(token); err == nil {
		t.Fatal("ParseToken() error = nil, want error")
	}
}
