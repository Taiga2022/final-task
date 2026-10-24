package core

type Status string

const (
	Waiting    Status = "waiting_for_partner"
	Processing Status = "processing"
	Completed  Status = "completed"
	Failed     Status = "failed"
)

type Member struct {
	UserID string
	Ready  bool
}
type State struct {
	RoomID  string
	Members []Member
	Status  Status
}
type Failure struct {
	Code    string
	Message string
}

// Event is a transport-independent snapshot. Consumers must not mutate it.
type Event struct {
	Type    string
	State   State
	Items   []Series
	Failure Failure
}
