package core

import (
	"context"
	"fmt"
)

type Service struct {
	repository Repository
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) Create(ctx context.Context, userID string, roomName RoomName, password Password) (JoinedRoom, error) {
	if !roomName.valid() {
		return JoinedRoom{}, ErrInvalidRoomName
	}
	if !password.valid() {
		return JoinedRoom{}, ErrInvalidPassword
	}

	roomID, err := newRoomID()
	if err != nil {
		return JoinedRoom{}, fmt.Errorf("generate room ID: %w", err)
	}

	passwordHash, err := password.Hash()
	if err != nil {
		return JoinedRoom{}, fmt.Errorf("hash password: %w", err)
	}

	if err := s.repository.Create(ctx, roomID, userID, roomName, passwordHash); err != nil {
		return JoinedRoom{}, err
	}
	return JoinedRoom{RoomID: roomID, Status: "waiting"}, nil
}

func (s *Service) Join(ctx context.Context, userID string, roomName RoomName, password Password) (JoinedRoom, error) {
	if !roomName.valid() {
		return JoinedRoom{}, ErrInvalidRoomName
	}
	if !password.valid() {
		return JoinedRoom{}, ErrInvalidPassword
	}

	room, err := s.repository.FindByName(ctx, roomName)
	if err != nil {
		return JoinedRoom{}, err
	}
	if !password.Matches(room.PasswordHash) {
		return JoinedRoom{}, ErrIncorrectPassword
	}

	memberCount, err := s.repository.Join(ctx, room.ID, userID)
	if err != nil {
		return JoinedRoom{}, err
	}
	status := "waiting"
	if memberCount >= 2 {
		status = "ready"
	}
	return JoinedRoom{RoomID: room.ID, Status: status}, nil
}

func (s *Service) Leave(ctx context.Context, userID, roomID string) error {
	return s.repository.Leave(ctx, roomID, userID)
}
