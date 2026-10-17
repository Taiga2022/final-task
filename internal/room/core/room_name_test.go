package core

import (
	"errors"
	"strings"
	"testing"
)

func TestNewRoomNameTrimsWhitespace(t *testing.T) {
	name, err := NewRoomName("  movie night  ")
	if err != nil {
		t.Fatal(err)
	}
	if name.String() != "movie night" {
		t.Fatalf("name = %q", name.String())
	}
}

func TestNewRoomNameRejectsInvalidValues(t *testing.T) {
	for _, value := range []string{" ", strings.Repeat("a", 256)} {
		if _, err := NewRoomName(value); !errors.Is(err, ErrInvalidRoomName) {
			t.Errorf("NewRoomName(%q) error = %v", value, err)
		}
	}
}
