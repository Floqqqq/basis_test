package handlers

import (
	"net/http"

	"task-manager/internal/middleware"
	"task-manager/internal/service"
)

type ActivityHandler struct {
	activity *service.ActivityService
}

func NewActivityHandler(activity *service.ActivityService) *ActivityHandler {
	return &ActivityHandler{activity: activity}
}

func (h *ActivityHandler) List(w http.ResponseWriter, r *http.Request) {
	teamID, err := parseIDParam(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid team id")
		return
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

	activity, err := h.activity.List(r.Context(), middleware.GetUserID(r), teamID, limit, offset)
	if err != nil {
		writeServiceError(w, err, "cannot get team activity")
		return
	}
	writeJSON(w, http.StatusOK, activity)
}
