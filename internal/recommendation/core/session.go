package core

import (
	"context"
	"slices"
	"sync"
)

type session struct {
	mu          sync.Mutex
	room        RoomSnapshot
	loaded      bool
	selections  map[string][]string
	status      Status
	generation  uint64
	items       []Series
	failure     Failure
	cancel      context.CancelFunc
	subscribers map[*subscriber]string
}
type subscriber struct {
	events chan Event
	done   chan struct{}
}
type Subscription struct {
	Events <-chan Event
	Done   <-chan struct{}
	cancel func()
}

func (s *Subscription) Close() { s.cancel() }

func newSession() *session {
	return &session{selections: make(map[string][]string), status: Waiting, subscribers: make(map[*subscriber]string)}
}
func (s *session) state() State {
	members := make([]Member, 0, len(s.room.Members))
	for _, user := range s.room.Members {
		members = append(members, Member{user, len(s.selections[user]) > 0})
	}
	return State{RoomID: s.room.ID, Members: members, Status: s.status}
}
func (s *session) events() []Event {
	state := s.state()
	events := []Event{{Type: "room.updated", State: state}}
	if s.status == Completed {
		events = append(events, Event{Type: "recommendation.completed", State: state, Items: slices.Clone(s.items)})
	}
	if s.status == Failed {
		events = append(events, Event{Type: "recommendation.failed", State: state, Failure: s.failure})
	}
	return events
}
func cloneEvent(e Event) Event {
	e.State.Members = slices.Clone(e.State.Members)
	e.Items = slices.Clone(e.Items)
	return e
}
func (s *session) drop(sub *subscriber) {
	if _, ok := s.subscribers[sub]; ok {
		delete(s.subscribers, sub)
		close(sub.done)
		close(sub.events)
	}
}
func (s *session) publish() {
	for _, event := range s.events() {
		for sub := range s.subscribers {
			select {
			case sub.events <- cloneEvent(event):
			default:
				// A slow client must reconnect for a complete snapshot, never block mutations.
				s.drop(sub)
			}
		}
	}
}
func (s *session) invalidate() {
	s.generation++
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
	s.items = nil
	s.failure = Failure{}
	s.status = Waiting
}
