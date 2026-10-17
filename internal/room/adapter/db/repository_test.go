package db

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Taiga2022/final-task/internal/room/core"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func testRepository(t *testing.T) (*Repository, sqlmock.Sqlmock) {
	t.Helper()
	conn, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	database, err := gorm.Open(mysql.New(mysql.Config{Conn: conn, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Error(err)
		}
		mock.ExpectClose()
		if err := conn.Close(); err != nil {
			t.Error(err)
		}
	})
	return NewRepository(database), mock
}

func TestLeave(t *testing.T) {
	updateErr := errors.New("update failed")
	for _, tc := range []struct {
		name       string
		user       string
		exists     bool
		member     bool
		remaining  []string
		failUpdate bool
		wantErr    error
	}{
		{name: "owner transfers ownership and resets weight", user: "owner", exists: true, member: true, remaining: []string{"partner"}},
		{name: "partner leaves without changing owner", user: "partner", exists: true, member: true, remaining: []string{"owner"}},
		{name: "last member deletes room", user: "owner", exists: true, member: true},
		{name: "nonmember rejected", user: "outsider", exists: true, wantErr: core.ErrRoomNotFound},
		{name: "missing room", user: "owner", wantErr: core.ErrRoomNotFound},
		{name: "transfer failure rolls back membership deletion", user: "owner", exists: true, member: true, remaining: []string{"partner"}, failUpdate: true, wantErr: updateErr},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repository, mock := testRepository(t)
			mock.ExpectBegin()
			rooms := sqlmock.NewRows([]string{"id", "owner_user_id"})
			if tc.exists {
				rooms.AddRow("room-1", "owner")
			}
			mock.ExpectQuery("SELECT .* FROM .*rooms.*FOR UPDATE").WithArgs("room-1", 1).WillReturnRows(rooms)
			if tc.exists {
				var affected int64
				if tc.member {
					affected = 1
				}
				mock.ExpectExec("DELETE FROM .*room_members.*").WithArgs("room-1", tc.user).WillReturnResult(sqlmock.NewResult(0, affected))
				if tc.member {
					members := sqlmock.NewRows([]string{"room_id", "user_id"})
					for _, user := range tc.remaining {
						members.AddRow("room-1", user)
					}
					mock.ExpectQuery("SELECT .* FROM .*room_members.*ORDER BY user_id").WithArgs("room-1").WillReturnRows(members)
					if len(tc.remaining) == 0 {
						mock.ExpectExec("DELETE FROM .*rooms.*").WithArgs("room-1").WillReturnResult(sqlmock.NewResult(0, 1))
					} else if tc.user == "owner" {
						update := mock.ExpectExec("UPDATE .*rooms.*SET .*owner_user_id.*weight.*").WithArgs(tc.remaining[0], 0.5, "room-1")
						if tc.failUpdate {
							update.WillReturnError(updateErr)
						} else {
							update.WillReturnResult(sqlmock.NewResult(0, 1))
						}
					}
				}
			}
			if tc.wantErr != nil {
				mock.ExpectRollback()
			} else {
				mock.ExpectCommit()
			}
			if err := repository.Leave(t.Context(), "room-1", tc.user); !errors.Is(err, tc.wantErr) {
				t.Fatalf("Leave error = %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func TestJoinCapacityAndRejoin(t *testing.T) {
	for _, tc := range []struct {
		name     string
		existing int64
		count    int64
		wantErr  error
	}{
		{"second member", 0, 1, nil},
		{"full room", 0, 2, core.ErrRoomFull},
		{"existing member in full room", 1, 2, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repository, mock := testRepository(t)
			mock.ExpectBegin()
			mock.ExpectQuery("SELECT .* FROM .*rooms.*FOR UPDATE").WithArgs("room-1", 1).
				WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("room-1"))
			mock.ExpectQuery("SELECT count.*FROM .*room_members.*").WithArgs("room-1", "user-2").
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(tc.existing))
			count := tc.count
			if tc.existing == 0 {
				mock.ExpectQuery("SELECT count.*FROM .*room_members.*").WithArgs("room-1").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(count))
				if tc.wantErr == nil {
					mock.ExpectExec("INSERT INTO .*room_members.*").WithArgs("room-1", "user-2").WillReturnResult(sqlmock.NewResult(0, 1))
					count++
				}
			}
			if tc.wantErr != nil {
				mock.ExpectRollback()
			} else {
				mock.ExpectQuery("SELECT count.*FROM .*room_members.*").WithArgs("room-1").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(count))
				mock.ExpectCommit()
			}
			got, err := repository.Join(t.Context(), "room-1", "user-2")
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("Join error = %v, want %v", err, tc.wantErr)
			}
			if err == nil && got != count {
				t.Fatalf("member count = %d, want %d", got, count)
			}
		})
	}
}
