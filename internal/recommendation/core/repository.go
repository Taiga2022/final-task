package core

import "context"

type Repository interface {
	Snapshot(ctx context.Context, roomID string) (RoomSnapshot, error)
	ValidateSelection(ctx context.Context, userID string, ids []string) (bool, error)
	SetWeight(ctx context.Context, roomID string, weight Weight) error
}

type Catalog interface {
	Series(ctx context.Context) ([]Series, error)
}
type Recommender interface {
	Recommend(ctx context.Context, input Input) ([]Series, error)
}
