package core

import "context"

type Reader interface {
	Read(ctx context.Context, roomID, actorID, targetID, after string, limit int) ([]Series, error)
}
