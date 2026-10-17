package db

import (
	"fmt"

	historycore "github.com/Taiga2022/final-task/internal/history/core"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type HistoryWriter struct {
	db *gorm.DB
}

func NewHistoryWriter(db *gorm.DB) *HistoryWriter {
	return &HistoryWriter{db: db}
}

func (writer *HistoryWriter) UpsertHistories(histories []historycore.ViewingHistory) error {
	models := make([]historyModel, len(histories))
	for index, history := range histories {
		models[index] = historyModel{
			UserID:       history.UserID,
			EpisodeID:    history.EpisodeID,
			LastViewedAt: history.LastViewedAt,
		}
	}

	result := writer.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "user_id"},
			{Name: "episode_id"},
		},
		DoUpdates: clause.Assignments(map[string]any{
			"last_viewed_at": gorm.Expr("GREATEST(last_viewed_at, VALUES(last_viewed_at))"),
		}),
	}).Create(&models)
	if result.Error != nil {
		return fmt.Errorf("upsert viewing histories: %w", result.Error)
	}

	return nil
}
