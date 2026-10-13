package core

import "time"

type ViewingHistory struct {
	UserID       string
	EpisodeID    string
	LastViewedAt time.Time
}
