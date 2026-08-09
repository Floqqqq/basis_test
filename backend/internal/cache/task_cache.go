package cache

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	taskListKeyPrefix = "team_tasks"
)

type TaskCache struct {
	redis *redis.Client
	ttl   time.Duration
}

func NewTaskCache(redis *redis.Client, ttl time.Duration) *TaskCache {
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	return &TaskCache{redis: redis, ttl: ttl}
}

func (c *TaskCache) GetList(ctx context.Context, teamID int64, status string, assigneeID *int64, limit, offset int) ([]byte, bool, error) {
	value, err := c.redis.Get(ctx, TaskListKey(teamID, status, assigneeID, limit, offset)).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("get task list cache: %w", err)
	}

	return value, true, nil
}

func (c *TaskCache) SetList(ctx context.Context, teamID int64, status string, assigneeID *int64, limit, offset int, body []byte) error {
	if err := c.redis.Set(ctx, TaskListKey(teamID, status, assigneeID, limit, offset), body, c.ttl).Err(); err != nil {
		return fmt.Errorf("set task list cache: %w", err)
	}
	return nil
}

func (c *TaskCache) InvalidateTeam(ctx context.Context, teamID int64) error {
	pattern := taskListKeyPrefix + ":" + strconv.FormatInt(teamID, 10) + ":*"
	var cursor uint64
	for {
		keys, next, err := c.redis.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return fmt.Errorf("scan task list cache keys: %w", err)
		}
		if len(keys) > 0 {
			if err := c.redis.Del(ctx, keys...).Err(); err != nil {
				return fmt.Errorf("delete task list cache keys: %w", err)
			}
		}
		cursor = next
		if cursor == 0 {
			return nil
		}
	}
}

func TaskListKey(teamID int64, status string, assigneeID *int64, limit, offset int) string {
	assignee := "none"
	if assigneeID != nil {
		assignee = strconv.FormatInt(*assigneeID, 10)
	}
	if status == "" {
		status = "all"
	}

	return fmt.Sprintf("%s:%d:status=%s:assignee=%s:limit=%d:offset=%d", taskListKeyPrefix, teamID, status, assignee, limit, offset)
}
