package db

import (
	"context"
	"errors"

	"github.com/Taiga2022/final-task/internal/history/core"
	room "github.com/Taiga2022/final-task/internal/room/core"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type HistoryReader struct{ db *gorm.DB }

func NewHistoryReader(db *gorm.DB) *HistoryReader { return &HistoryReader{db: db} }

func (r *HistoryReader) Read(ctx context.Context, roomID, actorID, targetID, after string, limit int) ([]core.Series, error) {
	var result []core.Series
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var roomRow struct{ ID string }
		if err := tx.Table("rooms").Clauses(clause.Locking{Strength: "SHARE"}).Where("id = ?", roomID).Take(&roomRow).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return room.ErrRoomNotFound
			}
			return err
		}
		var members int64
		if err := tx.Table("room_members").Where("room_id = ? AND user_id IN ?", roomID, []string{actorID, targetID}).Count(&members).Error; err != nil {
			return err
		}
		expected := int64(2)
		if actorID == targetID {
			expected = 1
		}
		if members != expected {
			return room.ErrRoomNotFound
		}
		query := tx.Table("viewing_histories AS h").
			Joins("JOIN episode AS e ON e.id = h.episode_id").
			Joins("JOIN series AS s ON s.id = e.series_id").
			Where("h.user_id = ?", targetID).
			Distinct("s.id", "s.title").Order("s.id").Limit(limit)
		if after != "" {
			query = query.Where("s.id > ?", after)
		}
		return query.Scan(&result).Error
	})
	return result, err
}
