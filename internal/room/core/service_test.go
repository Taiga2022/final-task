package core

import (
	"context"
	"errors"
	"testing"
)

type repositoryStub struct {
	room            Room
	findErr         error
	joinCount       int64
	joinErr         error
	createErr       error
	createdRoomID   string
	createdHash     string
	createdUserID   string
	createdRoomName string
	leftRoomID      string
	leftUserID      string
}

func (r *repositoryStub) Create(_ context.Context, roomID, userID string, roomName RoomName, passwordHash PasswordHash) error {
	r.createdRoomID, r.createdUserID = roomID, userID
	r.createdRoomName, r.createdHash = roomName.String(), passwordHash.String()
	return r.createErr
}

func (r *repositoryStub) FindByName(context.Context, RoomName) (Room, error) {
	return r.room, r.findErr
}

func (r *repositoryStub) Join(context.Context, string, string) (int64, error) {
	return r.joinCount, r.joinErr
}

func (r *repositoryStub) Leave(_ context.Context, roomID, userID string) error {
	r.leftRoomID, r.leftUserID = roomID, userID
	return nil
}

func TestCreateHashesPasswordAndJoinsOwner(t *testing.T) {
	repository := &repositoryStub{}
	roomName, _ := NewRoomName("movie night")
	password, _ := NewPassword("secret")
	result, err := NewService(repository).Create(context.Background(), "user-1", roomName, password)
	if err != nil {
		t.Fatal(err)
	}
	if result.RoomID == "" || result.RoomID != repository.createdRoomID || result.Status != "waiting" {
		t.Fatalf("result = %#v, stored ID = %q", result, repository.createdRoomID)
	}
	if repository.createdUserID != "user-1" || repository.createdRoomName != "movie night" {
		t.Fatalf("created user/name = %q/%q", repository.createdUserID, repository.createdRoomName)
	}
	storedHash, err := NewPasswordHash(repository.createdHash)
	if err != nil || repository.createdHash == "secret" || !password.Matches(storedHash) {
		t.Fatal("password was not stored as a valid bcrypt hash")
	}
}

func TestJoinRejectsIncorrectPassword(t *testing.T) {
	roomName, _ := NewRoomName("movie night")
	correctPassword, _ := NewPassword("secret")
	hash, err := correctPassword.Hash()
	if err != nil {
		t.Fatal(err)
	}
	repository := &repositoryStub{room: Room{ID: "room-1", PasswordHash: hash}}
	wrongPassword, _ := NewPassword("wrong")

	_, err = NewService(repository).Join(context.Background(), "user-2", roomName, wrongPassword)
	if !errors.Is(err, ErrIncorrectPassword) {
		t.Fatalf("error = %v, want ErrIncorrectPassword", err)
	}
}

func TestJoinReturnsReadyForTwoMembers(t *testing.T) {
	roomName, _ := NewRoomName("movie night")
	password, _ := NewPassword("secret")
	hash, err := password.Hash()
	if err != nil {
		t.Fatal(err)
	}
	repository := &repositoryStub{
		room:      Room{ID: "room-1", PasswordHash: hash},
		joinCount: 2,
	}

	result, err := NewService(repository).Join(context.Background(), "user-2", roomName, password)
	if err != nil {
		t.Fatal(err)
	}
	if result.RoomID != "room-1" || result.Status != "ready" {
		t.Fatalf("result = %#v", result)
	}
}
