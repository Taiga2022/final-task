package db

import (
	"context"
	"errors"

	"github.com/Taiga2022/final-task/internal/recommendation/core"
	room "github.com/Taiga2022/final-task/internal/room/core"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) Snapshot(ctx context.Context, roomID string) (core.RoomSnapshot, error) {
	var result core.RoomSnapshot
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row roomModel
		if err := tx.Clauses(clause.Locking{Strength: "SHARE"}).Where("id = ?", roomID).Take(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return room.ErrRoomNotFound
			}
			return err
		}
		weight, err := core.NewWeight(row.Weight)
		if err != nil {
			return err
		}
		var members []string
		if err := tx.Table("room_members").Where("room_id = ?", roomID).Order("user_id").Pluck("user_id", &members).Error; err != nil {
			return err
		}
		result = core.RoomSnapshot{ID: row.ID, OwnerID: row.OwnerUserID, Weight: weight, Members: members}
		return nil
	})
	return result, err
}
func (r *Repository) ValidateSelection(ctx context.Context, userID string, ids []string) (bool, error) {
	if len(ids) == 0 {
		return true, nil
	}
	var count int64
	err := r.db.WithContext(ctx).Table("viewing_histories AS h").
		Joins("JOIN episode AS e ON e.id = h.episode_id").
		Joins("JOIN series AS s ON s.id = e.series_id").
		Where("h.user_id = ? AND s.id IN ?", userID, ids).Distinct("s.id").Count(&count).Error
	return count == int64(len(ids)), err
}
func (r *Repository) SetWeight(ctx context.Context, roomID string, weight core.Weight) error {
	return r.db.WithContext(ctx).Table("rooms").Where("id = ?", roomID).Update("weight", weight.Value()).Error
}
func (r *Repository) Series(ctx context.Context) ([]core.Series, error) {
	var rows []seriesModel
	if err := r.db.WithContext(ctx).Table("series").Select("id, title, COALESCE(genre_id, '') AS genre_id").Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]core.Series, 0, len(rows))
	for _, row := range rows {
		result = append(result, core.Series{ID: row.ID, Title: row.Title, Genre: row.GenreID})
	}
	return result, nil
}
