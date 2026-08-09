package notifications

import (
	"testing"
	"time"
)

func TestRetryDelay(t *testing.T) {
	tests := []struct {
		attempt int
		want    time.Duration
	}{
		{attempt: 1, want: 5 * time.Second},
		{attempt: 2, want: 30 * time.Second},
		{attempt: 3, want: 2 * time.Minute},
		{attempt: 4, want: 10 * time.Minute},
		{attempt: 10, want: 30 * time.Minute},
	}
	for _, test := range tests {
		if got := RetryDelay(test.attempt); got != test.want {
			t.Fatalf("RetryDelay(%d) = %v, want %v", test.attempt, got, test.want)
		}
	}
}
