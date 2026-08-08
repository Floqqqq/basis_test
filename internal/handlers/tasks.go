package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"task-manager/internal/middleware"
	"task-manager/internal/models"
	"task-manager/internal/service"
)

type TaskHandler struct {
	tasks *service.TaskService
}

func NewTaskHandler(tasks *service.TaskService) *TaskHandler {
	return &TaskHandler{tasks: tasks}
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
	if req.AssigneeID != nil && *req.AssigneeID <= 0 {
		writeError(w, http.StatusBadRequest, "invalid assignee_id")
		return
	}

	if req.Status == "" {
		req.Status = "todo"
	}
	if !validTaskStatus(req.Status) {
		writeError(w, http.StatusBadRequest, "invalid status")
		return
	}

	taskID, err := h.tasks.Create(r.Context(), userID, models.Task{
		Title:       req.Title,
		Description: req.Description,
		Status:      req.Status,
		AssigneeID:  req.AssigneeID,
		TeamID:      req.TeamID,
	})
	if err != nil {
		writeServiceError(w, err, "cannot create task")
		return
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

	tasks, err := h.tasks.List(r.Context(), userID, teamID, status, assigneeID, limit, offset)
	if err != nil {
		writeServiceError(w, err, "cannot get tasks")
		return
	}

	writeJSON(w, http.StatusOK, tasks)
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

	var req updateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}

	if req.Title == nil && req.Description == nil && req.Status == nil && req.AssigneeID == nil {
		writeError(w, http.StatusBadRequest, "no fields to update")
		return
	}

	update := service.TaskUpdate{}

	if req.Title != nil {
		title := strings.TrimSpace(*req.Title)
		if title == "" {
			writeError(w, http.StatusBadRequest, "title cannot be empty")
			return
		}
		if len(title) > 255 {
			writeError(w, http.StatusBadRequest, "title is too long")
			return
		}
		update.Title = &title
	}

	if req.Description != nil {
		update.Description = req.Description
	}

	if req.Status != nil {
		if !validTaskStatus(*req.Status) {
			writeError(w, http.StatusBadRequest, "invalid status")
			return
		}
		update.Status = req.Status
	}

	if req.AssigneeID != nil {
		if *req.AssigneeID <= 0 {
			writeError(w, http.StatusBadRequest, "invalid assignee_id")
			return
		}
		update.AssigneeID = req.AssigneeID
	}

	if err := h.tasks.Update(r.Context(), userID, taskID, update); err != nil {
		writeServiceError(w, err, "cannot update task")
		return
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

	history, err := h.tasks.History(r.Context(), userID, taskID)
	if err != nil {
		writeServiceError(w, err, "cannot get history")
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

func writeServiceError(w http.ResponseWriter, err error, fallback string) {
	switch {
	case errors.Is(err, service.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden")
	case errors.Is(err, service.ErrNotFound):
		writeError(w, http.StatusNotFound, "not found")
	case errors.Is(err, service.ErrConflict):
		writeError(w, http.StatusConflict, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, fallback)
	}
}
