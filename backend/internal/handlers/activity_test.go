package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"task-manager/internal/middleware"
	"task-manager/internal/models"
	"task-manager/internal/service"
)

type activityReaderStub struct {
	limit  int
	offset int
}

func (r *activityReaderStub) List(ctx context.Context, teamID int64, limit, offset int) ([]models.ActivityEvent, error) {
	r.limit = limit
	r.offset = offset
	return []models.ActivityEvent{}, nil
}

type activityTeamAccessStub struct {
	member bool
}

func (a activityTeamAccessStub) GetUserRole(ctx context.Context, teamID, userID int64) (string, error) {
	return "member", nil
}

func (a activityTeamAccessStub) IsTeamMember(ctx context.Context, teamID, userID int64) (bool, error) {
	return a.member, nil
}

func TestActivityHandlerRejectsOutsider(t *testing.T) {
	handler := NewActivityHandler(service.NewActivityService(&activityReaderStub{}, activityTeamAccessStub{}))
	recorder := httptest.NewRecorder()
	request := activityRequest("/teams/5/activity", "5")

	handler.List(recorder, request)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", recorder.Code)
	}
}

func TestActivityHandlerPassesPagination(t *testing.T) {
	reader := &activityReaderStub{}
	handler := NewActivityHandler(service.NewActivityService(reader, activityTeamAccessStub{member: true}))
	recorder := httptest.NewRecorder()
	request := activityRequest("/teams/5/activity?limit=15&offset=30", "5")

	handler.List(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if reader.limit != 15 || reader.offset != 30 {
		t.Fatalf("pagination = %d/%d, want 15/30", reader.limit, reader.offset)
	}
}

func activityRequest(target, teamID string) *http.Request {
	request := httptest.NewRequest(http.MethodGet, target, nil)
	request = request.WithContext(context.WithValue(request.Context(), middleware.UserIDKey, int64(42)))
	return withURLParam(request, "id", teamID)
}
