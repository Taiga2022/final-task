package core

import "context"

type Repository interface {
	Create(ctx context.Context, roomID, userID string, roomName RoomName, passwordHash PasswordHash) error
	FindByName(ctx context.Context, roomName RoomName) (Room, error)
	Join(ctx context.Context, roomID, userID string) (memberCount int64, err error)
	Leave(ctx context.Context, roomID, userID string) error
}
