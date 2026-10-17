package core

import (
	"strings"
	"unicode/utf8"
)

const maxRoomNameLength = 255

type RoomName struct {
	value string
}

func NewRoomName(value string) (RoomName, error) {
	value = strings.TrimSpace(value)
	if value == "" || utf8.RuneCountInString(value) > maxRoomNameLength {
		return RoomName{}, ErrInvalidRoomName
	}
	return RoomName{value: value}, nil
}

func (n RoomName) String() string {
	return n.value
}

func (n RoomName) valid() bool {
	return n.value != "" && utf8.RuneCountInString(n.value) <= maxRoomNameLength
}
