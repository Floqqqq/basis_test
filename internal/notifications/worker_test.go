package notifications

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"task-manager/internal/events"
	"task-manager/internal/repository"
)

type fakeOutboxStore struct {
	entries      []repository.OutboxEntry
	processed    []int64
	retried      []int64
	failed       []int64
	retryAt      time.Time
	pendingCount int
}

func (s *fakeOutboxStore) FetchPending(context.Context, int) ([]repository.OutboxEntry, error) {
	return s.entries, nil
}

func (s *fakeOutboxStore) MarkProcessed(_ context.Context, id int64) error {
	s.processed = append(s.processed, id)
	return nil
}

func (s *fakeOutboxStore) ScheduleRetry(_ context.Context, id int64, _ error, retryAt time.Time) error {
	s.retried = append(s.retried, id)
	s.retryAt = retryAt
	return nil
}

func (s *fakeOutboxStore) MarkFailed(_ context.Context, id int64, _ error, _ time.Time) error {
	s.failed = append(s.failed, id)
	return nil
}

func (s *fakeOutboxStore) PendingCount(context.Context) (int, error) {
	return s.pendingCount, nil
}

type fakeEventProcessor struct{ err error }

func (p fakeEventProcessor) ProcessEvent(context.Context, events.Event) error { return p.err }

func outboxEntry(t *testing.T, attempts int) repository.OutboxEntry {
	t.Helper()
	event := events.New(events.TeamMemberAdded, 1, events.TeamMemberPayload{TeamID: 5, UserID: 2, Role: "member"})
	payload, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	return repository.OutboxEntry{ID: 7, EventID: event.EventID, EventType: event.EventType, Payload: payload, Attempts: attempts}
}

func TestWorkerMarksSuccessfulEventProcessed(t *testing.T) {
	store := &fakeOutboxStore{entries: []repository.OutboxEntry{outboxEntry(t, 0)}}
	worker := NewWorker(store, fakeEventProcessor{}, 10, 5, time.Second)

	processed, err := worker.ProcessBatch(context.Background())
	if err != nil {
		t.Fatalf("ProcessBatch() error = %v", err)
	}
	if processed != 1 || len(store.processed) != 1 || len(store.retried) != 0 || len(store.failed) != 0 {
		t.Fatalf("processed=%d marked=%v retried=%v failed=%v", processed, store.processed, store.retried, store.failed)
	}
}

func TestWorkerSchedulesRetry(t *testing.T) {
	store := &fakeOutboxStore{entries: []repository.OutboxEntry{outboxEntry(t, 0)}}
	worker := NewWorker(store, fakeEventProcessor{err: errors.New("SMTP unavailable")}, 10, 5, time.Second)
	now := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	worker.now = func() time.Time { return now }

	if _, err := worker.ProcessBatch(context.Background()); err != nil {
		t.Fatalf("ProcessBatch() error = %v", err)
	}
	if len(store.retried) != 1 || !store.retryAt.Equal(now.Add(5*time.Second)) || len(store.failed) != 0 {
		t.Fatalf("retried=%v retryAt=%v failed=%v", store.retried, store.retryAt, store.failed)
	}
}

func TestWorkerMarksEventFailedAtMaxAttempts(t *testing.T) {
	store := &fakeOutboxStore{entries: []repository.OutboxEntry{outboxEntry(t, 4)}}
	worker := NewWorker(store, fakeEventProcessor{err: errors.New("SMTP unavailable")}, 10, 5, time.Second)

	if _, err := worker.ProcessBatch(context.Background()); err != nil {
		t.Fatalf("ProcessBatch() error = %v", err)
	}
	if len(store.failed) != 1 || len(store.retried) != 0 {
		t.Fatalf("failed=%v retried=%v", store.failed, store.retried)
	}
}
