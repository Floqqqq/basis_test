package cache

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/redis/go-redis/v9"
)

func TestTaskCacheGetSetTTLAndMiss(t *testing.T) {
	ctx := context.Background()
	client, mock := redismock.NewClientMock()
	defer client.Close()

	cache := NewTaskCache(client, time.Minute)
	assigneeID := int64(7)
	key := TaskListKey(1, "todo", &assigneeID, 20, 0)

	mock.ExpectGet(key).RedisNil()
	if value, ok, err := cache.GetList(ctx, 1, "todo", &assigneeID, 20, 0); err != nil || ok || value != nil {
		t.Fatalf("cache miss = %q, %v, %v; want nil, false, nil", value, ok, err)
	}

	body := []byte(`[{"id":1}]`)
	mock.ExpectSet(key, body, time.Minute).SetVal("OK")
	if err := cache.SetList(ctx, 1, "todo", &assigneeID, 20, 0, body); err != nil {
		t.Fatalf("SetList() error = %v", err)
	}

	mock.ExpectGet(key).SetVal(string(body))
	got, ok, err := cache.GetList(ctx, 1, "todo", &assigneeID, 20, 0)
	if err != nil {
		t.Fatalf("GetList() error = %v", err)
	}
	if !ok || string(got) != string(body) {
		t.Fatalf("GetList() = %q, %v; want %q, true", got, ok, body)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet redis expectations: %v", err)
	}
}

func TestTaskCacheGetWrapsUnexpectedRedisError(t *testing.T) {
	ctx := context.Background()
	client, mock := redismock.NewClientMock()
	defer client.Close()

	cache := NewTaskCache(client, time.Minute)
	upstreamErr := errors.New("redis down")
	mock.ExpectGet(TaskListKey(1, "", nil, 20, 0)).SetErr(upstreamErr)

	_, ok, err := cache.GetList(ctx, 1, "", nil, 20, 0)
	if ok {
		t.Fatal("GetList() ok = true, want false")
	}
	if !errors.Is(err, upstreamErr) {
		t.Fatalf("GetList() error = %v, want wrapped redis error", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet redis expectations: %v", err)
	}
}

func TestTaskCacheInvalidateTeam(t *testing.T) {
	ctx := context.Background()
	client, mock := redismock.NewClientMock()
	defer client.Close()

	cache := NewTaskCache(client, time.Minute)
	key := TaskListKey(1, "todo", nil, 20, 0)

	mock.ExpectScan(0, "team_tasks:1:*", 100).SetVal([]string{key}, 0)
	mock.ExpectDel(key).SetVal(1)

	if err := cache.InvalidateTeam(ctx, 1); err != nil {
		t.Fatalf("InvalidateTeam() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet redis expectations: %v", err)
	}
}

func TestTaskCacheRedisNilIsCacheMiss(t *testing.T) {
	ctx := context.Background()
	client, mock := redismock.NewClientMock()
	defer client.Close()

	cache := NewTaskCache(client, time.Minute)
	mock.ExpectGet(TaskListKey(1, "", nil, 20, 0)).SetErr(redis.Nil)

	if value, ok, err := cache.GetList(ctx, 1, "", nil, 20, 0); err != nil || ok || value != nil {
		t.Fatalf("redis.Nil result = %q, %v, %v; want nil, false, nil", value, ok, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet redis expectations: %v", err)
	}
}

func TestTaskListKeyIncludesFiltersAndPagination(t *testing.T) {
	assigneeID := int64(7)

	keys := map[string]bool{}
	for _, key := range []string{
		TaskListKey(1, "todo", &assigneeID, 20, 0),
		TaskListKey(1, "done", &assigneeID, 20, 0),
		TaskListKey(1, "todo", nil, 20, 0),
		TaskListKey(1, "todo", &assigneeID, 10, 20),
		TaskListKey(2, "todo", &assigneeID, 20, 0),
	} {
		if keys[key] {
			t.Fatalf("duplicate cache key %q", key)
		}
		keys[key] = true
	}
}
