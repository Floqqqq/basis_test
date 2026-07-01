package redisclient

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

func NewRedis(addr string) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr: addr,
	})

	if err := pingWithRetry(client, 30, time.Second); err != nil {
		if closeErr := client.Close(); closeErr != nil {
			return nil, fmt.Errorf("ping redis: %w; close redis: %v", err, closeErr)
		}
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	return client, nil
}

func pingWithRetry(client *redis.Client, attempts int, delay time.Duration) error {
	var err error
	for i := 0; i < attempts; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), delay)
		err = client.Ping(ctx).Err()
		cancel()
		if err == nil {
			return nil
		}
		time.Sleep(delay)
	}
	return err
}
