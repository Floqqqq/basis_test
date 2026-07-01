package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"task-manager/internal/cache"
	"task-manager/internal/middleware"
	"task-manager/internal/models"
	"task-manager/internal/repository"
	"task-manager/internal/service"
)

type TaskHandler struct {
	tasks     *repository.TaskRepository
	policy    *service.TaskPolicy
	taskCache *cache.TaskCache
}

func NewTaskHandler(tasks *repository.TaskRepository, policy *service.TaskPolicy, taskCache *cache.TaskCache) *TaskHandler {
	return &TaskHandler{
		tasks:     tasks,
		policy:    policy,
		taskCache: taskCache,
	}
}

type createTaskRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
	AssigneeID  *int64 `json:"assignee_id"`
	TeamID      int64  `json:"team_id"`
}

func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)

	var req createTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}

	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}
	if len(req.Title) > 255 {
		writeError(w, http.StatusBadRequest, "title is too long")
		return
	}
	if req.TeamID <= 0 {
		writeError(w, http.StatusBadRequest, "team_id is required")
		return
	}

	isMember, err := h.policy.IsTeamMember(r.Context(), req.TeamID, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "cannot check team membership")
		return
	}
	if !isMember {
		writeError(w, http.StatusForbidden, "user is not team member")
		return
	}

	if req.AssigneeID != nil && *req.AssigneeID <= 0 {
		writeError(w, http.StatusBadRequest, "invalid assignee_id")
		return
	}

	assigneeValid, err := h.policy.IsAssigneeValidForTeam(r.Context(), req.TeamID, req.AssigneeID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "cannot check assignee")
		return
	}
	if !assigneeValid {
		writeError(w, http.StatusBadRequest, "assignee is not team member")
		return
	}

	if req.Status == "" {
		req.Status = "todo"
	}
	if !validTaskStatus(req.Status) {
		writeError(w, http.StatusBadRequest, "invalid status")
		return
	}

	taskID, err := h.tasks.Create(r.Context(), models.Task{
		Title:       req.Title,
		Description: req.Description,
		Status:      req.Status,
		AssigneeID:  req.AssigneeID,
		TeamID:      req.TeamID,
		CreatedBy:   userID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "cannot create task")
		return
	}

	if err := h.taskCache.InvalidateTeam(r.Context(), req.TeamID); err != nil {
		// Cache invalidation is best-effort; the database write is already complete.
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"task_id": taskID,
	})
}

func (h *TaskHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)

	teamID, err := strconv.ParseInt(r.URL.Query().Get("team_id"), 10, 64)
	if err != nil || teamID <= 0 {
		writeError(w, http.StatusBadRequest, "team_id is required")
		return
	}

	status := r.URL.Query().Get("status")
	if status != "" && !validTaskStatus(status) {
		writeError(w, http.StatusBadRequest, "invalid status")
		return
	}

	var assigneeID *int64
	if value := r.URL.Query().Get("assignee_id"); value != "" {
		id, err := strconv.ParseInt(value, 10, 64)
		if err != nil || id <= 0 {
			writeError(w, http.StatusBadRequest, "invalid assignee_id")
			return
		}
		assigneeID = &id
	}

	limit, err := parsePositiveIntQuery(r, "limit", 20, 100)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid limit")
		return
	}

	offset, err := parseNonNegativeInt(r.URL.Query().Get("offset"), 0)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid offset")
		return
	}

	isMember, err := h.policy.IsTeamMember(r.Context(), teamID, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "cannot check team membership")
		return
	}
	if !isMember {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	assigneeValid, err := h.policy.IsAssigneeValidForTeam(r.Context(), teamID, assigneeID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "cannot check assignee")
		return
	}
	if !assigneeValid {
		writeError(w, http.StatusBadRequest, "assignee is not team member")
		return
	}

	cached, ok, err := h.taskCache.GetList(r.Context(), teamID, status, assigneeID, limit, offset)
	if err == nil && ok {
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write(cached); err != nil {
			return
		}
		return
	}

	tasks, err := h.tasks.List(r.Context(), teamID, status, assigneeID, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "cannot get tasks")
		return
	}

	body, err := json.Marshal(tasks)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "cannot encode tasks")
		return
	}

	if err := h.taskCache.SetList(r.Context(), teamID, status, assigneeID, limit, offset, body); err != nil {
	}

	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(body); err != nil {
		return
	}
}

type updateTaskRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	Status      *string `json:"status"`
	AssigneeID  *int64  `json:"assignee_id"`
}

func (h *TaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)

	taskID, err := parseIDParam(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid task id")
		return
	}

	task, err := h.tasks.GetByID(r.Context(), taskID)
	if err != nil {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}

	canUpdate, err := h.policy.CanUpdateTask(r.Context(), task, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "cannot check permissions")
		return
	}
	if !canUpdate {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	var req updateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}

	if req.Title == nil && req.Description == nil && req.Status == nil && req.AssigneeID == nil {
		writeError(w, http.StatusBadRequest, "no fields to update")
		return
	}

	updated := *task

	if req.Title != nil {
		updated.Title = strings.TrimSpace(*req.Title)
		if updated.Title == "" {
			writeError(w, http.StatusBadRequest, "title cannot be empty")
			return
		}
		if len(updated.Title) > 255 {
			writeError(w, http.StatusBadRequest, "title is too long")
			return
		}
	}

	if req.Description != nil {
		updated.Description = *req.Description
	}

	if req.Status != nil {
		if !validTaskStatus(*req.Status) {
			writeError(w, http.StatusBadRequest, "invalid status")
			return
		}
		updated.Status = *req.Status
	}

	if req.AssigneeID != nil {
		if *req.AssigneeID <= 0 {
			writeError(w, http.StatusBadRequest, "invalid assignee_id")
			return
		}

		assigneeValid, err := h.policy.IsAssigneeValidForTeam(r.Context(), task.TeamID, req.AssigneeID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "cannot check assignee")
			return
		}
		if !assigneeValid {
			writeError(w, http.StatusBadRequest, "assignee is not team member")
			return
		}

		updated.AssigneeID = req.AssigneeID
	}

	if err := h.tasks.Update(r.Context(), userID, updated); err != nil {
		writeError(w, http.StatusInternalServerError, "cannot update task")
		return
	}

	if err := h.taskCache.InvalidateTeam(r.Context(), task.TeamID); err != nil {
		// Cache invalidation is best-effort; the database write is already complete.
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status": "updated",
	})
}

func (h *TaskHandler) History(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)

	taskID, err := parseIDParam(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid task id")
		return
	}

	task, err := h.tasks.GetByID(r.Context(), taskID)
	if err != nil {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}

	canView, err := h.policy.CanViewTaskHistory(r.Context(), task, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "cannot check permissions")
		return
	}
	if !canView {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	history, err := h.tasks.History(r.Context(), taskID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "cannot get history")
		return
	}

	writeJSON(w, http.StatusOK, history)
}

func validTaskStatus(status string) bool {
	switch status {
	case "todo", "in_progress", "done":
		return true
	default:
		return false
	}
}

func parseNonNegativeInt(value string, fallback int) (int, error) {
	if value == "" {
		return fallback, nil
	}
	i, err := strconv.Atoi(value)
	if err != nil || i < 0 {
		return 0, fmt.Errorf("invalid non-negative int")
	}
	return i, nil
}
