package db

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	room "github.com/Taiga2022/final-task/internal/room/core"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestHistoryReaderAuthorizationAndQuery(t *testing.T) {
	for _, tc := range []struct {
		name, actor, target, after string
		members                    int64
		exists                     bool
		wantErr                    error
	}{
		{"partner history", "a", "b", "s1", 2, true, nil},
		{"own history", "a", "a", "", 1, true, nil},
		{"outsider", "outsider", "b", "", 1, true, room.ErrRoomNotFound},
		{"missing target", "a", "outsider", "", 1, true, room.ErrRoomNotFound},
		{"missing room", "a", "b", "", 0, false, room.ErrRoomNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			conn, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				mock.ExpectClose()
				if err := conn.Close(); err != nil {
					t.Error(err)
				}
			}()
			database, err := gorm.Open(mysql.New(mysql.Config{Conn: conn, SkipInitializeWithVersion: true}), &gorm.Config{})
			if err != nil {
				t.Fatal(err)
			}
			mock.ExpectBegin()
			rooms := sqlmock.NewRows([]string{"id"})
			if tc.exists {
				rooms.AddRow("r")
			}
			mock.ExpectQuery("SELECT .* FROM .*rooms.*FOR SHARE").WithArgs("r", 1).WillReturnRows(rooms)
			if tc.exists {
				mock.ExpectQuery("SELECT count.*FROM .*room_members.*").WithArgs("r", tc.actor, tc.target).
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(tc.members))
			}
			if tc.wantErr == nil {
				query := mock.ExpectQuery("SELECT DISTINCT .*s.id.*s.title.*JOIN episode.*JOIN series.*ORDER BY s.id LIMIT")
				if tc.after != "" {
					query.WithArgs(tc.target, tc.after, 21)
				} else {
					query.WithArgs(tc.target, 21)
				}
				query.WillReturnRows(sqlmock.NewRows([]string{"id", "title"}).AddRow("s2", "Series Two"))
				mock.ExpectCommit()
			} else {
				mock.ExpectRollback()
			}
			got, err := NewHistoryReader(database).Read(t.Context(), "r", tc.actor, tc.target, tc.after, 21)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("error %v want %v", err, tc.wantErr)
			}
			if err == nil && (len(got) != 1 || got[0].ID != "s2") {
				t.Fatalf("result %+v", got)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Error(err)
			}
		})
	}
}
