package core

type RoomSnapshot struct {
	ID      string
	OwnerID string
	Weight  Weight
	Members []string
}

type Series struct {
	ID    string
	Title string
	Genre string
}

// Input is an immutable snapshot of the two members' selections and weight.
type Input struct {
	OwnerIDs   []string
	PartnerIDs []string
	Weight     Weight
}
