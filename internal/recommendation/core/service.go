package core

import (
	"context"
	"slices"
	"sync"
	"time"

	room "github.com/Taiga2022/final-task/internal/room/core"
)

type Service struct {
	repository  Repository
	recommender Recommender
	mu          sync.Mutex
	sessions    map[string]*session
	ctx         context.Context
	cancel      context.CancelFunc
	workers     sync.WaitGroup
	closed      bool
}

func NewService(repository Repository, recommender Recommender) *Service {
	ctx, cancel := context.WithCancel(context.Background())
	return &Service{repository: repository, recommender: recommender, sessions: make(map[string]*session), ctx: ctx, cancel: cancel}
}

func (s *Service) entry(roomID string) (*session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil, ErrClosed
	}
	if s.sessions[roomID] == nil {
		s.sessions[roomID] = newSession()
	}
	return s.sessions[roomID], nil
}

func (s *Service) load(ctx context.Context, e *session, roomID string) error {
	if e.loaded {
		return nil
	}
	snapshot, err := s.repository.Snapshot(ctx, roomID)
	if err != nil {
		return err
	}
	snapshot.Members = slices.Clone(snapshot.Members)
	slices.Sort(snapshot.Members)
	e.room, e.loaded = snapshot, true
	return nil
}

func (s *Service) member(ctx context.Context, e *session, roomID, userID string) error {
	if err := s.load(ctx, e, roomID); err != nil {
		return err
	}
	if !slices.Contains(e.room.Members, userID) {
		return room.ErrRoomNotFound
	}
	return nil
}

func (s *Service) ChangeMembership(ctx context.Context, roomID, userID string, joining bool, change func() error) error {
	e, err := s.entry(roomID)
	if err != nil {
		return err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := s.load(ctx, e, roomID); err != nil {
		return err
	}
	if err := change(); err != nil {
		return err
	}
	if joining {
		if slices.Contains(e.room.Members, userID) {
			return nil
		}
		e.room.Members = append(e.room.Members, userID)
		slices.Sort(e.room.Members)
	} else {
		e.room.Members = slices.DeleteFunc(e.room.Members, func(id string) bool { return id == userID })
		delete(e.selections, userID)
		if e.room.OwnerID == userID && len(e.room.Members) > 0 {
			e.room.OwnerID = e.room.Members[0]
			e.room.Weight, _ = NewWeight(0.5)
		}
		for sub, user := range e.subscribers {
			if user == userID {
				e.drop(sub)
			}
		}
	}
	e.invalidate()
	e.publish()
	return nil
}

func (s *Service) Submit(ctx context.Context, roomID, userID string, selection Selection) (Status, error) {
	if selection.ids == nil {
		return "", ErrInvalidSelectionInput
	}
	e, err := s.entry(roomID)
	if err != nil {
		return "", err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := s.member(ctx, e, roomID, userID); err != nil {
		return "", err
	}
	if e.status == Processing {
		return "", ErrInProgress
	}
	valid, err := s.repository.ValidateSelection(ctx, userID, selection.ids)
	if err != nil {
		return "", err
	}
	if !valid {
		return "", ErrInvalidSelection
	}
	if slices.Equal(e.selections[userID], selection.ids) && e.status != Failed {
		return e.status, nil
	}
	e.selections[userID] = selection.IDs()
	e.invalidate()
	if err := s.start(e); err != nil {
		return "", err
	}
	e.publish()
	return e.status, nil
}

func (s *Service) Preference(ctx context.Context, roomID, userID string, weight Weight) error {
	e, err := s.entry(roomID)
	if err != nil {
		return err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := s.member(ctx, e, roomID, userID); err != nil {
		return err
	}
	if len(e.room.Members) != 2 {
		return ErrRoomNotReady
	}
	if e.status == Processing {
		return ErrInProgress
	}
	if e.room.Weight == weight {
		return nil
	}
	if err := s.repository.SetWeight(ctx, roomID, weight); err != nil {
		return err
	}
	e.room.Weight = weight
	e.invalidate()
	if err := s.start(e); err != nil {
		return err
	}
	e.publish()
	return nil
}

func (s *Service) Subscribe(ctx context.Context, roomID, userID string) (*Subscription, error) {
	e, err := s.entry(roomID)
	if err != nil {
		return nil, err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := s.member(ctx, e, roomID, userID); err != nil {
		return nil, err
	}
	sub := &subscriber{events: make(chan Event, 32), done: make(chan struct{})}
	e.subscribers[sub] = userID
	for _, event := range e.events() {
		sub.events <- cloneEvent(event)
	}
	return &Subscription{
		Events: sub.events, Done: sub.done,
		cancel: func() { e.mu.Lock(); defer e.mu.Unlock(); e.drop(sub) },
	}, nil
}

func (s *Service) Close() {
	s.mu.Lock()
	s.closed = true
	s.cancel()
	entries := make([]*session, 0, len(s.sessions))
	for _, e := range s.sessions {
		entries = append(entries, e)
	}
	s.mu.Unlock()
	s.workers.Wait()
	for _, e := range entries {
		e.mu.Lock()
		for sub := range e.subscribers {
			e.drop(sub)
		}
		e.mu.Unlock()
	}
}

func (s *Service) start(e *session) error {
	if len(e.room.Members) != 2 {
		return nil
	}
	input := Input{OwnerIDs: slices.Clone(e.selections[e.room.OwnerID]), Weight: e.room.Weight}
	for _, id := range e.room.Members {
		if id != e.room.OwnerID {
			input.PartnerIDs = slices.Clone(e.selections[id])
		}
	}
	if len(input.OwnerIDs) == 0 || len(input.PartnerIDs) == 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ErrClosed
	}
	ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
	e.cancel = cancel
	e.status = Processing
	generation := e.generation
	s.workers.Go(func() {
		defer cancel()
		items, err := s.recommender.Recommend(ctx, input)
		if err == nil {
			err = ctx.Err()
		}
		e.mu.Lock()
		defer e.mu.Unlock()
		if e.generation != generation || s.ctx.Err() != nil {
			return
		}
		e.cancel = nil
		if err != nil {
			e.status = Failed
			e.failure = Failure{Code: "INTERNAL_ERROR", Message: "recommendation calculation failed"}
		} else {
			e.status = Completed
			e.items = slices.Clone(items[:min(10, len(items))])
		}
		e.publish()
	})
	return nil
}
