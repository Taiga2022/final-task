package core

type Series struct{ ID, Title string }

type Page struct {
	Items   []Series
	HasMore bool
	LastID  string
}
