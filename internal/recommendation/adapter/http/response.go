package http

import (
	"github.com/Taiga2022/final-task/internal/recommendation/core"
	"github.com/Taiga2022/final-task/internal/shared/api"
)

type SubmissionResponse struct {
	Status core.Status `json:"status"`
}
type PreferenceResponse struct {
	Status string  `json:"status"`
	Weight float64 `json:"weight"`
}
type MemberResponse struct {
	UserID string `json:"user_id"`
	Ready  bool   `json:"ready"`
}
type StateResponse struct {
	RoomID  string           `json:"room_id"`
	Members []MemberResponse `json:"members"`
	Status  core.Status      `json:"status"`
}
type SeriesResponse struct {
	ID    string `json:"series_id"`
	Title string `json:"series_title"`
}
type RecommendationResponse struct {
	RoomID string           `json:"room_id"`
	Items  []SeriesResponse `json:"items"`
}
type FailureResponse struct {
	RoomID string          `json:"room_id"`
	Error  api.ErrorDetail `json:"error"`
}
type EventResponse struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

func eventResponse(event core.Event) EventResponse {
	var data any
	switch event.Type {
	case "room.updated":
		members := make([]MemberResponse, 0, len(event.State.Members))
		for _, m := range event.State.Members {
			members = append(members, MemberResponse{m.UserID, m.Ready})
		}
		data = StateResponse{event.State.RoomID, members, event.State.Status}
	case "recommendation.completed":
		items := make([]SeriesResponse, 0, len(event.Items))
		for _, s := range event.Items {
			items = append(items, SeriesResponse{s.ID, s.Title})
		}
		data = RecommendationResponse{event.State.RoomID, items}
	case "recommendation.failed":
		data = FailureResponse{event.State.RoomID, api.ErrorDetail{Code: event.Failure.Code, Message: event.Failure.Message}}
	}
	return EventResponse{event.Type, data}
}
