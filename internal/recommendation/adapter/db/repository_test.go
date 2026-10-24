package db

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Taiga2022/final-task/internal/recommendation/core"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestRecommendationRepository(t *testing.T) {
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
	r := NewRepository(database)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT .* FROM .*rooms.*FOR SHARE").WithArgs("r", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "owner_user_id", "weight"}).AddRow("r", "a", 0.7))
	mock.ExpectQuery("SELECT .*user_id.* FROM .*room_members.*ORDER BY user_id").WithArgs("r").
		WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow("a").AddRow("b"))
	mock.ExpectCommit()
	snapshot, err := r.Snapshot(t.Context(), "r")
	if err != nil || snapshot.OwnerID != "a" || snapshot.Weight.Value() != 0.7 || len(snapshot.Members) != 2 {
		t.Fatalf("%+v %v", snapshot, err)
	}
	mock.ExpectQuery("SELECT COUNT.*DISTINCT.*JOIN episode.*JOIN series.*").
		WithArgs("a", "s1", "s2").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	if ok, err := r.ValidateSelection(t.Context(), "a", []string{"s1", "s2"}); err != nil || ok {
		t.Fatalf("selection %v %v", ok, err)
	}
	if ok, err := r.ValidateSelection(t.Context(), "a", nil); err != nil || !ok {
		t.Fatalf("empty selection %v %v", ok, err)
	}
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE .*rooms.*SET .*weight.*").WithArgs(0.2, "r").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	w, _ := core.NewWeight(0.2)
	if err := r.SetWeight(t.Context(), "r", w); err != nil {
		t.Fatal(err)
	}
	mock.ExpectQuery("SELECT id, title, COALESCE.*FROM .*series.*").
		WillReturnRows(sqlmock.NewRows([]string{"id", "title", "genre_id"}).AddRow("s", "Series", "comedy"))
	rows, err := r.Series(t.Context())
	if err != nil || len(rows) != 1 || rows[0].Genre != "comedy" {
		t.Fatalf("%+v %v", rows, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}
