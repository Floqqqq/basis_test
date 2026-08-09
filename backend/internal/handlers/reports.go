package handlers

import (
	"net/http"

	"task-manager/internal/middleware"
	"task-manager/internal/repository"
)

type ReportHandler struct {
	reports *repository.ReportRepository
}

func NewReportHandler(reports *repository.ReportRepository) *ReportHandler {
	return &ReportHandler{reports: reports}
}

func (h *ReportHandler) TeamStats(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)

	result, err := h.reports.TeamStats(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "cannot get stats")
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *ReportHandler) TopUsers(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)

	result, err := h.reports.TopUsers(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "cannot get top users")
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *ReportHandler) InvalidAssignees(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)

	result, err := h.reports.InvalidAssignees(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "cannot get invalid tasks")
		return
	}

	writeJSON(w, http.StatusOK, result)
}
