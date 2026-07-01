package handlers

import "testing"

func TestValidInviteRole(t *testing.T) {
	for _, role := range []string{"admin", "member"} {
		if !validInviteRole(role) {
			t.Fatalf("validInviteRole(%q) = false, want true", role)
		}
	}

	for _, role := range []string{"owner", "", "viewer"} {
		if validInviteRole(role) {
			t.Fatalf("validInviteRole(%q) = true, want false", role)
		}
	}
}
