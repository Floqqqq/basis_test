package realtime

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"task-manager/internal/service"
)

type TeamAccess interface {
	IsTeamMember(ctx context.Context, teamID, userID int64) (bool, error)
}

type Handler struct {
	hub      *Hub
	auth     *service.AuthService
	teams    TeamAccess
	upgrader websocket.Upgrader
}

func NewHandler(hub *Hub, auth *service.AuthService, teams TeamAccess) *Handler {
	handler := &Handler{hub: hub, auth: auth, teams: teams}
	handler.upgrader = websocket.Upgrader{
		Subprotocols: []string{"access_token"},
		CheckOrigin:  sameOrigin,
	}
	return handler
}

func (h *Handler) Connect(w http.ResponseWriter, r *http.Request) {
	teamID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || teamID <= 0 {
		writeError(w, http.StatusBadRequest, "invalid team id")
		return
	}
	protocols := websocket.Subprotocols(r)
	if len(protocols) != 2 || protocols[0] != "access_token" {
		writeError(w, http.StatusUnauthorized, "invalid token")
		return
	}
	userID, err := h.auth.ParseToken(protocols[1])
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid token")
		return
	}

	ctx, span := otel.Tracer("task-manager/realtime").Start(r.Context(), "WebSocket.Connect")
	span.SetAttributes(attribute.Int64("team.id", teamID), attribute.Int64("user.id", userID))
	defer span.End()

	isMember, err := h.teams.IsTeamMember(ctx, teamID, userID)
	if err != nil {
		span.RecordError(err)
		writeError(w, http.StatusInternalServerError, "cannot establish websocket connection")
		return
	}
	if !isMember {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		span.RecordError(err)
		return
	}
	h.hub.Add(conn, teamID, userID)
}

func sameOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	parsed, err := url.Parse(origin)
	return err == nil && parsed.Host == r.Host
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
