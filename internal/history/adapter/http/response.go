package http

type SeriesResponse struct {
	SeriesID    string `json:"series_id"`
	SeriesTitle string `json:"series_title"`
}
type HistoryResponse struct {
	Items      []SeriesResponse `json:"items"`
	Pagination Pagination       `json:"pagination"`
}
type Pagination struct {
	NextCursor *string `json:"next_cursor"`
	HasMore    bool    `json:"has_more"`
}
