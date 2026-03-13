package user

import (
	"testing"
)

func TestHashPassword(t *testing.T) {
	password := "test_password_123"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	if hash == "" {
		t.Error("HashPassword returned empty string")
	}

	if hash == password {
		t.Error("HashPassword should not return plain password")
	}
}

func TestVerifyPasswordSuccess(t *testing.T) {
	password := "test_password_123"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	if !VerifyPassword(hash, password) {
		t.Error("VerifyPassword should return true for correct password")
	}
}

func TestVerifyPasswordFailure(t *testing.T) {
	password := "test_password_123"
	wrongPassword := "wrong_password"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}

	if VerifyPassword(hash, wrongPassword) {
		t.Error("VerifyPassword should return false for incorrect password")
	}
}
