package core

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

const maxPasswordLength = 72

type Password struct {
	value string
}

func NewPassword(value string) (Password, error) {
	if value == "" || len(value) > maxPasswordLength {
		return Password{}, ErrInvalidPassword
	}
	return Password{value: value}, nil
}

func (p Password) Hash() (PasswordHash, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(p.value), bcrypt.DefaultCost)
	if err != nil {
		return PasswordHash{}, err
	}
	return PasswordHash{value: string(hash)}, nil
}

func (p Password) Matches(hash PasswordHash) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash.value), []byte(p.value)) == nil
}

func (p Password) valid() bool {
	return p.value != "" && len(p.value) <= maxPasswordLength
}

type PasswordHash struct {
	value string
}

// NewPasswordHash restores a hash read from persistent storage.
func NewPasswordHash(value string) (PasswordHash, error) {
	if _, err := bcrypt.Cost([]byte(value)); err != nil {
		return PasswordHash{}, fmt.Errorf("%w: %v", ErrInvalidPasswordHash, err)
	}
	return PasswordHash{value: value}, nil
}

func (h PasswordHash) String() string {
	return h.value
}
