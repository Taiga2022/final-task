package http

type RoomCredentials struct {
	RoomName string `json:"room_name"`
	Password string `json:"password"`
}

type RoomJoined struct {
	RoomID string `json:"room_id"`
	Status string `json:"status"`
}

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
