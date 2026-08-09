package notifications

import "time"

var retrySchedule = []time.Duration{
	5 * time.Second,
	30 * time.Second,
	2 * time.Minute,
	10 * time.Minute,
	30 * time.Minute,
}

func RetryDelay(attempt int) time.Duration {
	if attempt <= 1 {
		return retrySchedule[0]
	}
	index := attempt - 1
	if index >= len(retrySchedule) {
		index = len(retrySchedule) - 1
	}
	return retrySchedule[index]
}
