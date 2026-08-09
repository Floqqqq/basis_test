package service

import (
	"context"
	"errors"
	"testing"
	"time"
)

type inviteSenderFunc func(ctx context.Context, teamID, userID int64, role string) error

func (f inviteSenderFunc) SendInvite(ctx context.Context, teamID, userID int64, role string) error {
	return f(ctx, teamID, userID, role)
}

func TestCircuitBreakerInviteSenderOpensAfterFailures(t *testing.T) {
	upstreamErr := errors.New("upstream failed")
	calls := 0
	sender := NewCircuitBreakerInviteSender(inviteSenderFunc(func(ctx context.Context, teamID, userID int64, role string) error {
		calls++
		return upstreamErr
	}), 2, time.Minute)

	if err := sender.SendInvite(context.Background(), 1, 2, "member"); !errors.Is(err, upstreamErr) {
		t.Fatalf("first SendInvite() error = %v, want upstream error", err)
	}
	if err := sender.SendInvite(context.Background(), 1, 2, "member"); !errors.Is(err, upstreamErr) {
		t.Fatalf("second SendInvite() error = %v, want upstream error", err)
	}
	if err := sender.SendInvite(context.Background(), 1, 2, "member"); !errors.Is(err, ErrCircuitBreakerOpen) {
		t.Fatalf("third SendInvite() error = %v, want circuit open", err)
	}
	if calls != 2 {
		t.Fatalf("upstream calls = %d, want 2", calls)
	}
}

func TestCircuitBreakerInviteSenderResetsAfterSuccess(t *testing.T) {
	upstreamErr := errors.New("upstream failed")
	fail := true
	sender := NewCircuitBreakerInviteSender(inviteSenderFunc(func(ctx context.Context, teamID, userID int64, role string) error {
		if fail {
			return upstreamErr
		}
		return nil
	}), 1, time.Nanosecond)

	if err := sender.SendInvite(context.Background(), 1, 2, "member"); !errors.Is(err, upstreamErr) {
		t.Fatalf("first SendInvite() error = %v, want upstream error", err)
	}

	time.Sleep(time.Millisecond)
	fail = false

	if err := sender.SendInvite(context.Background(), 1, 2, "member"); err != nil {
		t.Fatalf("half-open SendInvite() error = %v, want nil", err)
	}
	if err := sender.SendInvite(context.Background(), 1, 2, "member"); err != nil {
		t.Fatalf("closed SendInvite() error = %v, want nil", err)
	}
}
