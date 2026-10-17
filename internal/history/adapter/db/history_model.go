package db

import "time"

type historyModel struct {
	UserID       string    `gorm:"column:user_id;type:varchar(64);primaryKey"`
	EpisodeID    string    `gorm:"column:episode_id;type:varchar(64);primaryKey"`
	LastViewedAt time.Time `gorm:"column:last_viewed_at;not null;autoCreateTime"`
}

func (historyModel) TableName() string {
	return "viewing_histories"
}
