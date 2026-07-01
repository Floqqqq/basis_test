package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"task-manager/internal/repository"
	"task-manager/internal/service"
)

type AuthHandler struct {
	users *repository.UserRepository
	auth  *service.AuthService
}

func NewAuthHandler(users *repository.UserRepository, auth *service.AuthService) *AuthHandler {
	return &AuthHandler{
		users: users,
		auth:  auth,
	}
}

type authRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req authRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}

	req.Email = strings.TrimSpace(req.Email)
	if !validEmail(req.Email) {
		writeError(w, http.StatusBadRequest, "invalid email")
		return
	}
	if len(req.Password) < 6 {
		writeError(w, http.StatusBadRequest, "password must be at least 6 characters")
		return
	}

	hash, err := h.auth.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "cannot hash password")
		return
	}

	userID, err := h.users.Create(r.Context(), req.Email, hash)
	if err != nil {
		writeError(w, http.StatusConflict, "user already exists")
		return
	}

	token, err := h.auth.GenerateToken(userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "cannot generate token")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"user_id": userID,
		"token":   token,
	})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req authRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}

	req.Email = strings.TrimSpace(req.Email)
	if !validEmail(req.Email) {
		writeError(w, http.StatusBadRequest, "invalid email")
		return
	}
	if len(req.Password) < 6 {
		writeError(w, http.StatusBadRequest, "password must be at least 6 characters")
		return
	}

	user, err := h.users.GetByEmail(r.Context(), req.Email)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	if !h.auth.CheckPassword(user.PasswordHash, req.Password) {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	token, err := h.auth.GenerateToken(user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "cannot generate token")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"user_id": user.ID,
		"token":   token,
	})
}

func validEmail(email string) bool {
	if len(email) < 3 || len(email) > 255 {
		return false
	}
	if strings.ContainsAny(email, " \t\r\n") {
		return false
	}
	at := strings.Index(email, "@")
	return at > 0 && at < len(email)-1 && strings.Contains(email[at+1:], ".")
}
