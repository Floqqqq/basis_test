package handlers

import "testing"

func TestValidEmail(t *testing.T) {
	for _, email := range []string{"user@example.com", "a@b.co"} {
		if !validEmail(email) {
			t.Fatalf("validEmail(%q) = false, want true", email)
		}
	}

	for _, email := range []string{"", "missing-at", "@example.com", "user@", "user@example", "user @example.com"} {
		if validEmail(email) {
			t.Fatalf("validEmail(%q) = true, want false", email)
		}
	}
}
