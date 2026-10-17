package core

type Room struct {
	ID           string
	PasswordHash PasswordHash
	MemberCount  int64
}

// CanJoin checks capacity for a new member; existing members may rejoin.
func (r Room) CanJoin() error {
	if r.MemberCount >= 2 {
		return ErrRoomFull
	}
	return nil
}
