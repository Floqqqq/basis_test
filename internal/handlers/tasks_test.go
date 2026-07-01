package handlers

import (
	"testing"

	"task-manager/internal/cache"
)

func TestTaskCacheKeyIncludesFiltersAndPagination(t *testing.T) {
	var assigneeID int64 = 7

	first := cache.TaskListKey(1, "todo", &assigneeID, 20, 0)
	second := cache.TaskListKey(1, "done", &assigneeID, 20, 0)
	third := cache.TaskListKey(1, "todo", nil, 20, 0)
	fourth := cache.TaskListKey(1, "todo", &assigneeID, 10, 20)

	keys := map[string]bool{}
	for _, key := range []string{first, second, third, fourth} {
		if keys[key] {
			t.Fatalf("duplicate cache key %q", key)
		}
		keys[key] = true
	}
}

func TestValidTaskStatus(t *testing.T) {
	for _, status := range []string{"todo", "in_progress", "done"} {
		if !validTaskStatus(status) {
			t.Fatalf("validTaskStatus(%q) = false, want true", status)
		}
	}

	if validTaskStatus("archived") {
		t.Fatal("validTaskStatus(\"archived\") = true, want false")
	}
}

func TestParseNonNegativeInt(t *testing.T) {
	value, err := parseNonNegativeInt("", 20)
	if err != nil || value != 20 {
		t.Fatalf("parseNonNegativeInt empty = %d, %v; want 20, nil", value, err)
	}

	value, err = parseNonNegativeInt("15", 20)
	if err != nil || value != 15 {
		t.Fatalf("parseNonNegativeInt 15 = %d, %v; want 15, nil", value, err)
	}

	if _, err := parseNonNegativeInt("-1", 20); err == nil {
		t.Fatal("parseNonNegativeInt -1 error = nil, want error")
	}
}
