package http

type SelectionRequest struct {
	IncludedSeriesIDs []string `json:"included_series_ids"`
}
type PreferenceRequest struct {
	Weight *float64 `json:"weight"`
}
