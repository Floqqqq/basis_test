package notifications

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel/attribute"

	"task-manager/internal/events"
	"task-manager/internal/repository"
)

type OutboxStore interface {
	FetchPending(ctx context.Context, limit int) ([]repository.OutboxEntry, error)
	MarkProcessed(ctx context.Context, id int64) error
	ScheduleRetry(ctx context.Context, id int64, eventErr error, retryAt time.Time) error
	MarkFailed(ctx context.Context, id int64, eventErr error, failedAt time.Time) error
	PendingCount(ctx context.Context) (int, error)
}

type EventProcessor interface {
	ProcessEvent(ctx context.Context, event events.Event) error
}

type Worker struct {
	outbox       OutboxStore
	processor    EventProcessor
	batchSize    int
	pollInterval time.Duration
	maxAttempts  int
	now          func() time.Time
}

func NewWorker(outbox OutboxStore, processor EventProcessor, batchSize, maxAttempts int, pollInterval time.Duration) *Worker {
	return &Worker{
		outbox:       outbox,
		processor:    processor,
		batchSize:    batchSize,
		pollInterval: pollInterval,
		maxAttempts:  maxAttempts,
		now:          time.Now,
	}
}

func (w *Worker) Run(ctx context.Context) error {
	for {
		processed, err := w.ProcessBatch(ctx)
		if err != nil && ctx.Err() == nil {
			slog.Error("notification worker batch failed", "error", err)
		}
		if ctx.Err() != nil {
			return nil
		}
		if processed == w.batchSize {
			continue
		}
		timer := time.NewTimer(w.pollInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil
		case <-timer.C:
		}
	}
}

func (w *Worker) ProcessBatch(ctx context.Context) (processed int, err error) {
	ctx, span := tracer.Start(ctx, "OutboxWorker.ProcessBatch")
	defer func() { finishSpan(span, err) }()

	entries, err := w.outbox.FetchPending(ctx, w.batchSize)
	if err != nil {
		return 0, err
	}
	span.SetAttributes(attribute.Int("outbox.batch_size", len(entries)))

	var firstErr error
	for _, entry := range entries {
		if processErr := w.processEntry(ctx, entry); processErr != nil && firstErr == nil {
			firstErr = processErr
		}
	}
	if count, countErr := w.outbox.PendingCount(ctx); countErr == nil {
		OutboxPending.Set(float64(count))
	} else if firstErr == nil {
		firstErr = countErr
	}
	return len(entries), firstErr
}

func (w *Worker) processEntry(ctx context.Context, entry repository.OutboxEntry) error {
	event, err := events.Decode(entry.Payload)
	if err == nil {
		err = w.processor.ProcessEvent(ctx, event)
	}
	if err == nil {
		if err := w.outbox.MarkProcessed(ctx, entry.ID); err != nil {
			return err
		}
		NotificationProcessed.WithLabelValues(string(entry.EventType)).Inc()
		return nil
	}

	NotificationFailed.WithLabelValues(string(entry.EventType)).Inc()
	nextAttempt := entry.Attempts + 1
	if nextAttempt >= w.maxAttempts {
		if markErr := w.outbox.MarkFailed(ctx, entry.ID, err, w.now()); markErr != nil {
			return fmt.Errorf("mark notification failed after processing error %v: %w", err, markErr)
		}
		return nil
	}
	retryAt := w.now().Add(RetryDelay(nextAttempt))
	if scheduleErr := w.outbox.ScheduleRetry(ctx, entry.ID, err, retryAt); scheduleErr != nil {
		return fmt.Errorf("schedule notification retry after processing error %v: %w", err, scheduleErr)
	}
	return nil
}
