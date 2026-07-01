package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"task-manager/internal/middleware"
	"task-manager/internal/repository"
	"task-manager/internal/service"
)

type TeamHandler struct {
	teams        *repository.TeamRepository
	inviteSender service.InviteSender
}

func NewTeamHandler(teams *repository.TeamRepository, inviteSender service.InviteSender) *TeamHandler {
	return &TeamHandler{
		teams:        teams,
		inviteSender: inviteSender,
	}
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

	teamID, err := h.teams.Create(r.Context(), req.Name, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "cannot create team")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"team_id": teamID,
	})
}

func (h *TeamHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)

	teams, err := h.teams.ListByUser(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "cannot get teams")
		return
	}

	writeJSON(w, http.StatusOK, teams)
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

	role, err := h.teams.GetUserRole(r.Context(), teamID, currentUserID)
	if err != nil || (role != "owner" && role != "admin") {
		writeError(w, http.StatusForbidden, "forbidden")
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

	if err := h.teams.Invite(r.Context(), teamID, req.UserID, req.Role); err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}

		if errors.Is(err, repository.ErrTeamMemberExists) {
			writeError(w, http.StatusConflict, "user is already team member")
			return
		}

		writeError(w, http.StatusInternalServerError, "cannot invite user")
		return
	}

	if err := h.inviteSender.SendInvite(r.Context(), teamID, req.UserID, req.Role); err != nil {
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
