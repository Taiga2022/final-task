package core

type Room struct {
	ID           string
	PasswordHash PasswordHash
	MemberCount  int64
}
