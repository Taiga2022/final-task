package core

import "errors"

var (
	ErrInvalidRoomName       = errors.New("invalid room name")
	ErrInvalidPassword       = errors.New("invalid password")
	ErrInvalidPasswordHash   = errors.New("invalid password hash")
	ErrRoomNameAlreadyExists = errors.New("room name already exists")
	ErrRoomNotFound          = errors.New("room not found")
	ErrIncorrectPassword     = errors.New("incorrect password")
	ErrRoomFull              = errors.New("room is full")
)
