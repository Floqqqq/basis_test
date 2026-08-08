package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"task-manager/internal/middleware"
	"task-manager/internal/service"
)

type TeamHandler struct {
	teams *service.TeamService
}

func NewTeamHandler(teams *service.TeamService) *TeamHandler {
	return &TeamHandler{teams: teams}
}

type createTeamRequest struct {
	Name string `json:"name"`
}

func (h *TeamHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)

	var req createTeamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if len(req.Name) > 255 {
		writeError(w, http.StatusBadRequest, "name is too long")
		return
	}

	teamID, err := h.teams.Create(r.Context(), userID, req.Name)
	if err != nil {
		writeServiceError(w, err, "cannot create team")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"team_id": teamID,
	})
}

func (h *TeamHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)

	teams, err := h.teams.List(r.Context(), userID)
	if err != nil {
		writeServiceError(w, err, "cannot get teams")
		return
	}

	writeJSON(w, http.StatusOK, teams)
}

func (h *TeamHandler) Members(w http.ResponseWriter, r *http.Request) {
	teamID, err := parseIDParam(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid team id")
		return
	}

	members, err := h.teams.ListMembers(r.Context(), middleware.GetUserID(r), teamID)
	if err != nil {
		writeServiceError(w, err, "cannot get team members")
		return
	}

	writeJSON(w, http.StatusOK, members)
}

type inviteRequest struct {
	UserID int64  `json:"user_id"`
	Role   string `json:"role"`
}

func (h *TeamHandler) Invite(w http.ResponseWriter, r *http.Request) {
	currentUserID := middleware.GetUserID(r)

	teamID, err := parseIDParam(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid team id")
		return
	}

	var req inviteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}

	if req.UserID <= 0 {
		writeError(w, http.StatusBadRequest, "user_id is required")
		return
	}

	if req.Role == "" {
		req.Role = "member"
	}
	if !validInviteRole(req.Role) {
		writeError(w, http.StatusBadRequest, "invalid role")
		return
	}

	if err := h.teams.Invite(r.Context(), currentUserID, teamID, req.UserID, req.Role); err != nil {
		writeServiceError(w, err, "cannot invite user")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status": "invited",
	})
}

func validInviteRole(role string) bool {
	switch role {
	case "admin", "member":
		return true
	default:
		return false
	}
}
