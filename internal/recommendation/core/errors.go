package core

import "errors"

var (
	ErrInvalidWeight         = errors.New("invalid weight")
	ErrInvalidSelectionInput = errors.New("invalid selection input")
	ErrInvalidSelection      = errors.New("selection contains series outside viewing history")
	ErrRoomNotReady          = errors.New("room needs two members")
	ErrInProgress            = errors.New("recommendation is in progress")
	ErrClosed                = errors.New("recommendation service is closed")
)
