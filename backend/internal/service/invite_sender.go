package service

import (
	"context"
	"errors"
	"sync"
	"time"
)

var ErrCircuitBreakerOpen = errors.New("circuit breaker is open")

type InviteSender interface {
	SendInvite(ctx context.Context, teamID, userID int64, role string) error
}

type MockInviteSender struct{}

func NewMockInviteSender() *MockInviteSender {
	return &MockInviteSender{}
}

func (s *MockInviteSender) SendInvite(ctx context.Context, teamID, userID int64, role string) error {
	return ctx.Err()
}

type CircuitBreakerInviteSender struct {
	next             InviteSender
	failureThreshold int
	resetTimeout     time.Duration

	mu           sync.Mutex
	failures     int
	openedAt     time.Time
	halfOpenCall bool
}

func NewCircuitBreakerInviteSender(next InviteSender, failureThreshold int, resetTimeout time.Duration) *CircuitBreakerInviteSender {
	if failureThreshold <= 0 {
		failureThreshold = 3
	}
	if resetTimeout <= 0 {
		resetTimeout = 30 * time.Second
	}

	return &CircuitBreakerInviteSender{
		next:             next,
		failureThreshold: failureThreshold,
		resetTimeout:     resetTimeout,
	}
}

func (s *CircuitBreakerInviteSender) SendInvite(ctx context.Context, teamID, userID int64, role string) error {
	if err := s.beforeCall(); err != nil {
		return err
	}

	err := s.next.SendInvite(ctx, teamID, userID, role)
	s.afterCall(err)

	return err
}

func (s *CircuitBreakerInviteSender) beforeCall() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.openedAt.IsZero() {
		return nil
	}

	if time.Since(s.openedAt) < s.resetTimeout {
		return ErrCircuitBreakerOpen
	}

	if s.halfOpenCall {
		return ErrCircuitBreakerOpen
	}

	s.halfOpenCall = true
	return nil
}

func (s *CircuitBreakerInviteSender) afterCall(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.halfOpenCall = false

	if err == nil {
		s.failures = 0
		s.openedAt = time.Time{}
		return
	}

	s.failures++
	if s.failures >= s.failureThreshold {
		s.openedAt = time.Now()
	}
}
