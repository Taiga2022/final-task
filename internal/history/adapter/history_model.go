package adapter

import "time"

type HistoryModel struct {
	UserID       string    `gorm:"column:user_id;type:varchar(64);primaryKey"`
	EpisodeID    string    `gorm:"column:episode_id;type:varchar(64);primaryKey"`
	LastViewedAt time.Time `gorm:"column:last_viewed_at;not null;autoCreateTime"`
}

func (HistoryModel) TableName() string {
	return "viewing_histories"
}
