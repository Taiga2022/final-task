package core

import (
	"context"
	"errors"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	room "github.com/Taiga2022/final-task/internal/room/core"
)

type repositoryStub struct {
	snapshot  RoomSnapshot
	valid     bool
	weightErr error
	weights   []Weight
}

func (r *repositoryStub) Snapshot(context.Context, string) (RoomSnapshot, error) {
	return r.snapshot, nil
}
func (r *repositoryStub) ValidateSelection(context.Context, string, []string) (bool, error) {
	return r.valid, nil
}
func (r *repositoryStub) SetWeight(_ context.Context, _ string, w Weight) error {
	if r.weightErr != nil {
		return r.weightErr
	}
	r.weights = append(r.weights, w)
	return nil
}

type recommendFunc func(context.Context, Input) ([]Series, error)

func (f recommendFunc) Recommend(ctx context.Context, in Input) ([]Series, error) { return f(ctx, in) }
func fixture(t *testing.T, f recommendFunc) (*Service, *repositoryStub) {
	t.Helper()
	weight, _ := NewWeight(0.5)
	repo := &repositoryStub{snapshot: RoomSnapshot{ID: "r", OwnerID: "a", Weight: weight, Members: []string{"a", "b"}}, valid: true}
	s := NewService(repo, f)
	t.Cleanup(s.Close)
	return s, repo
}
func selection(t *testing.T, ids ...string) Selection {
	t.Helper()
	if ids == nil {
		ids = []string{}
	}
	s, err := NewSelection(ids)
	if err != nil {
		t.Fatal(err)
	}
	return s
}
func subscribe(t *testing.T, s *Service, user string) *Subscription {
	t.Helper()
	sub, err := s.Subscribe(t.Context(), "r", user)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(sub.Close)
	return sub
}
func nextEvent(t *testing.T, sub *Subscription) Event {
	t.Helper()
	select {
	case e, ok := <-sub.Events:
		if !ok {
			t.Fatal("subscription closed")
		}
		return e
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for event")
		return Event{}
	}
}
func awaitStatus(t *testing.T, sub *Subscription, want Status) Event {
	t.Helper()
	for {
		e := nextEvent(t, sub)
		if e.Type == "room.updated" && e.State.Status == want {
			return e
		}
	}
}

func TestSubmissionLifecycleAndReconnect(t *testing.T) {
	var calls atomic.Int32
	s, _ := fixture(t, func(context.Context, Input) ([]Series, error) {
		calls.Add(1)
		return []Series{{ID: "result", Title: "Result"}}, nil
	})
	sub := subscribe(t, s, "a")
	initial := nextEvent(t, sub)
	if initial.State.Status != Waiting || initial.State.Members[0].Ready {
		t.Fatalf("initial: %+v", initial)
	}
	if status, err := s.Submit(t.Context(), "r", "a", selection(t, "s1")); err != nil || status != Waiting {
		t.Fatalf("first submit: %s %v", status, err)
	}
	if status, err := s.Submit(t.Context(), "r", "b", selection(t, "s2")); err != nil || status != Processing {
		t.Fatalf("second submit: %s %v", status, err)
	}
	awaitStatus(t, sub, Completed)
	completed := nextEvent(t, sub)
	if completed.Type != "recommendation.completed" || completed.Items[0].ID != "result" {
		t.Fatalf("event: %+v", completed)
	}
	reconnected := subscribe(t, s, "b")
	if nextEvent(t, reconnected).State.Status != Completed || nextEvent(t, reconnected).Type != "recommendation.completed" {
		t.Fatal("latest result not replayed")
	}
	if status, err := s.Submit(t.Context(), "r", "b", selection(t, "s2")); err != nil || status != Completed {
		t.Fatalf("duplicate: %s %v", status, err)
	}
	if calls.Load() != 1 {
		t.Fatalf("duplicate calculated again")
	}
	if status, err := s.Submit(t.Context(), "r", "a", selection(t)); err != nil || status != Waiting {
		t.Fatalf("clear: %s %v", status, err)
	}
	cleared := awaitStatus(t, sub, Waiting)
	if cleared.State.Members[0].Ready {
		t.Fatal("empty selection must clear readiness")
	}
	fresh := subscribe(t, s, "a")
	if nextEvent(t, fresh).State.Status != Waiting {
		t.Fatal("stale result retained")
	}
}

func TestFailedSubmissionCanRetry(t *testing.T) {
	var calls atomic.Int32
	s, _ := fixture(t, func(context.Context, Input) ([]Series, error) {
		if calls.Add(1) == 1 {
			return nil, errors.New("private DB failure")
		}
		return []Series{}, nil
	})
	sub := subscribe(t, s, "a")
	if _, err := s.Submit(t.Context(), "r", "a", selection(t, "s1")); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Submit(t.Context(), "r", "b", selection(t, "s2")); err != nil {
		t.Fatal(err)
	}
	awaitStatus(t, sub, Failed)
	failure := nextEvent(t, sub)
	if failure.Failure.Code != "INTERNAL_ERROR" || failure.Failure.Message == "private DB failure" {
		t.Fatalf("failure: %+v", failure)
	}
	reconnect := subscribe(t, s, "a")
	if nextEvent(t, reconnect).State.Status != Failed || nextEvent(t, reconnect).Type != "recommendation.failed" {
		t.Fatal("failure not replayed")
	}
	if status, err := s.Submit(t.Context(), "r", "b", selection(t, "s2")); err != nil || status != Processing {
		t.Fatalf("retry: %s %v", status, err)
	}
	awaitStatus(t, sub, Completed)
	if calls.Load() != 2 {
		t.Fatalf("calls = %d", calls.Load())
	}
}

func TestConcurrentSubmissionsStartOneJobAndLeaveInvalidatesIt(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	var calls atomic.Int32
	s, _ := fixture(t, func(context.Context, Input) ([]Series, error) {
		calls.Add(1)
		close(started)
		<-release // Simulates work that doesn't cooperate with cancellation.
		return []Series{{ID: "stale"}}, nil
	})
	var releaseOnce sync.Once
	t.Cleanup(func() { releaseOnce.Do(func() { close(release) }) })
	subA, subB := subscribe(t, s, "a"), subscribe(t, s, "b")
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for _, user := range []string{"a", "b"} {
		input := selection(t, user)
		wg.Go(func() { _, err := s.Submit(t.Context(), "r", user, input); errs <- err })
	}
	wg.Wait()
	for range 2 {
		if err := <-errs; err != nil {
			t.Fatal(err)
		}
	}
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("job not started")
	}
	if _, err := s.Submit(t.Context(), "r", "a", selection(t, "other")); !errors.Is(err, ErrInProgress) {
		t.Fatalf("busy submit = %v", err)
	}
	weight, _ := NewWeight(0.8)
	if err := s.Preference(t.Context(), "r", "a", weight); !errors.Is(err, ErrInProgress) {
		t.Fatalf("busy preference = %v", err)
	}
	awaitStatus(t, subB, Processing)
	if err := s.ChangeMembership(t.Context(), "r", "a", false, func() error { return nil }); err != nil {
		t.Fatal(err)
	}
	select {
	case <-subA.Done:
	default:
		t.Fatal("departed subscriber not closed")
	}
	waiting := awaitStatus(t, subB, Waiting)
	if len(waiting.State.Members) != 1 || waiting.State.Members[0].UserID != "b" {
		t.Fatalf("members: %+v", waiting.State.Members)
	}
	releaseOnce.Do(func() { close(release) })
	s.workers.Wait()
	replay := subscribe(t, s, "b")
	if nextEvent(t, replay).State.Status != Waiting {
		t.Fatal("stale computation published")
	}
	if calls.Load() != 1 {
		t.Fatalf("calls = %d", calls.Load())
	}
	if _, err := s.Subscribe(t.Context(), "r", "a"); !errors.Is(err, room.ErrRoomNotFound) {
		t.Fatalf("departed user: %v", err)
	}
}

func TestAuthorizationValidationAndWeight(t *testing.T) {
	inputs := make(chan Input, 4)
	s, repo := fixture(t, func(_ context.Context, in Input) ([]Series, error) { inputs <- in; return nil, nil })
	if _, err := s.Subscribe(t.Context(), "r", "outsider"); !errors.Is(err, room.ErrRoomNotFound) {
		t.Fatalf("subscribe: %v", err)
	}
	if _, err := s.Submit(t.Context(), "r", "outsider", selection(t, "s")); !errors.Is(err, room.ErrRoomNotFound) {
		t.Fatalf("submit: %v", err)
	}
	repo.valid = false
	if _, err := s.Submit(t.Context(), "r", "a", selection(t, "s")); !errors.Is(err, ErrInvalidSelection) {
		t.Fatalf("foreign series: %v", err)
	}
	repo.valid = true
	weight, _ := NewWeight(0.8)
	if err := s.Preference(t.Context(), "r", "a", weight); err != nil {
		t.Fatal(err)
	}
	if len(repo.weights) != 1 || repo.weights[0].Value() != 0.8 {
		t.Fatal("weight not persisted")
	}
	sub := subscribe(t, s, "a")
	if _, err := s.Submit(t.Context(), "r", "a", selection(t, "s1")); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Submit(t.Context(), "r", "b", selection(t, "s2")); err != nil {
		t.Fatal(err)
	}
	awaitStatus(t, sub, Completed)
	input := <-inputs
	if input.Weight.Value() != 0.8 || !slices.Equal(input.OwnerIDs, []string{"s1"}) {
		t.Fatalf("input: %+v", input)
	}
	updated, _ := NewWeight(0.2)
	if err := s.Preference(t.Context(), "r", "a", updated); err != nil {
		t.Fatal(err)
	}
	awaitStatus(t, sub, Completed)
	if (<-inputs).Weight.Value() != 0.2 {
		t.Fatal("weight change not recalculated")
	}
	if err := s.ChangeMembership(t.Context(), "r", "b", false, func() error { return nil }); err != nil {
		t.Fatal(err)
	}
	if err := s.Preference(t.Context(), "r", "a", weight); !errors.Is(err, ErrRoomNotReady) {
		t.Fatalf("single member: %v", err)
	}
}

func TestMembershipFailureDoesNotChangeLiveState(t *testing.T) {
	s, _ := fixture(t, func(context.Context, Input) ([]Series, error) { return nil, nil })
	sub := subscribe(t, s, "a")
	nextEvent(t, sub)
	failure := errors.New("DB failed")
	if err := s.ChangeMembership(t.Context(), "r", "a", false, func() error { return failure }); !errors.Is(err, failure) {
		t.Fatal(err)
	}
	select {
	case <-sub.Done:
		t.Fatal("connection closed after failed leave")
	default:
	}
	if len(nextEvent(t, subscribe(t, s, "a")).State.Members) != 2 {
		t.Fatal("membership changed on rollback")
	}
}

func TestRestartResetsReadinessAndSlowSubscribersAreClosed(t *testing.T) {
	s, repo := fixture(t, func(context.Context, Input) ([]Series, error) { return nil, nil })
	if _, err := s.Submit(t.Context(), "r", "a", selection(t, "s")); err != nil {
		t.Fatal(err)
	}
	fresh := NewService(repo, recommendFunc(func(context.Context, Input) ([]Series, error) { return nil, nil }))
	defer fresh.Close()
	for _, m := range nextEvent(t, subscribe(t, fresh, "a")).State.Members {
		if m.Ready {
			t.Fatal("readiness survived process reset")
		}
	}
	slow := subscribe(t, s, "a")
	for i := range 40 {
		ids := []string{"s"}
		if i%2 == 0 {
			ids = []string{}
		}
		if _, err := s.Submit(t.Context(), "r", "a", selection(t, ids...)); err != nil {
			t.Fatal(err)
		}
	}
	select {
	case <-slow.Done:
	default:
		t.Fatal("slow client not evicted")
	}
}
