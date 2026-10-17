package core

import (
	"errors"
	"strings"
	"testing"
)

func TestNewPasswordRejectsInvalidValues(t *testing.T) {
	for _, value := range []string{"", strings.Repeat("a", 73)} {
		if _, err := NewPassword(value); !errors.Is(err, ErrInvalidPassword) {
			t.Errorf("NewPassword(%q) error = %v", value, err)
		}
	}
}

func TestPasswordHashAndMatches(t *testing.T) {
	password, _ := NewPassword("secret")
	hash, err := password.Hash()
	if err != nil {
		t.Fatal(err)
	}
	if !password.Matches(hash) {
		t.Fatal("password should match its hash")
	}
	other, _ := NewPassword("other")
	if other.Matches(hash) {
		t.Fatal("different password matched hash")
	}
}

func TestNewPasswordHashRejectsInvalidHash(t *testing.T) {
	if _, err := NewPasswordHash("not-a-bcrypt-hash"); !errors.Is(err, ErrInvalidPasswordHash) {
		t.Fatalf("error = %v, want ErrInvalidPasswordHash", err)
	}
}
