package auth

import (
	"testing"
)

func TestArgon2PasswordHashing(t *testing.T) {
	password := "voxion-super-secret-password-123!"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	if len(hash) == 0 {
		t.Fatal("expected non-empty hash string")
	}

	// Verify with correct password
	valid, err := VerifyPassword(password, hash)
	if err != nil {
		t.Fatalf("verification returned error: %v", err)
	}
	if !valid {
		t.Error("expected password to match hash")
	}

	// Verify with wrong password
	invalid, err := VerifyPassword("wrong-password", hash)
	if err != nil {
		t.Fatalf("verification returned error: %v", err)
	}
	if invalid {
		t.Error("expected wrong password to fail verification")
	}
}

func TestVerifyInvalidHash(t *testing.T) {
	_, err := VerifyPassword("password", "invalid-hash-string")
	if err == nil {
		t.Error("expected error for malformed hash format")
	}
}
