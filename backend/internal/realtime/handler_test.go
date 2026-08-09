package realtime

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"

	"task-manager/internal/events"
	"task-manager/internal/middleware"
	"task-manager/internal/service"
	"task-manager/internal/telemetry"
)

type fakeTeamAccess struct {
	members map[int64]map[int64]bool
}

func (a fakeTeamAccess) IsTeamMember(ctx context.Context, teamID, userID int64) (bool, error) {
	return a.members[teamID][userID], nil
}

func TestHandlerRejectsOutsider(t *testing.T) {
	hub := NewHub()
	t.Cleanup(hub.Close)
	auth := service.NewAuthService("secret")
	server := realtimeServer(t, hub, auth, fakeTeamAccess{members: map[int64]map[int64]bool{}})
	token, err := auth.GenerateToken(42)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	_, response, err := dialTeam(server.URL, 5, token)
	if err == nil {
		t.Fatal("outsider websocket dial error = nil")
	}
	if response == nil || response.StatusCode != 403 {
		t.Fatalf("outsider status = %v, want 403", response)
	}
}

func TestHubBroadcastsOnlyToEventTeam(t *testing.T) {
	hub := NewHub()
	t.Cleanup(hub.Close)
	auth := service.NewAuthService("secret")
	access := fakeTeamAccess{members: map[int64]map[int64]bool{
		1: {42: true},
		2: {42: true},
	}}
	server := realtimeServer(t, hub, auth, access)
	token, err := auth.GenerateToken(42)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	teamOne, _, err := dialTeam(server.URL, 1, token)
	if err != nil {
		t.Fatalf("dial team 1: %v", err)
	}
	defer teamOne.Close()
	teamTwo, _, err := dialTeam(server.URL, 2, token)
	if err != nil {
		t.Fatalf("dial team 2: %v", err)
	}
	defer teamTwo.Close()

	hub.Publish(events.New(events.TaskCreated, 42, events.TaskPayload{TaskID: 10, TeamID: 1, Title: "Backend"}))
	_ = teamOne.SetReadDeadline(time.Now().Add(time.Second))
	_, payload, err := teamOne.ReadMessage()
	if err != nil {
		t.Fatalf("read team 1 event: %v", err)
	}
	var message Message
	if err := json.Unmarshal(payload, &message); err != nil {
		t.Fatalf("decode websocket event: %v", err)
	}
	if message.TeamID != 1 || message.EventType != events.TaskCreated || message.EntityID != 10 {
		t.Fatalf("message = %+v, want task.created for team 1 task 10", message)
	}

	_ = teamTwo.SetReadDeadline(time.Now().Add(150 * time.Millisecond))
	if _, _, err := teamTwo.ReadMessage(); err == nil {
		t.Fatal("team 2 received event for team 1")
	} else if timeout, ok := err.(net.Error); !ok || !timeout.Timeout() {
		t.Fatalf("team 2 read error = %v, want timeout", err)
	}
}

func realtimeServer(t *testing.T, hub *Hub, auth *service.AuthService, access TeamAccess) *httptest.Server {
	t.Helper()
	router := chi.NewRouter()
	router.Get("/api/v1/teams/{id}/ws", NewHandler(hub, auth, access).Connect)
	server := httptest.NewServer(telemetry.HTTPHandler(middleware.Metrics(router), "realtime-test"))
	t.Cleanup(server.Close)
	return server
}

func dialTeam(serverURL string, teamID int64, token string) (*websocket.Conn, *http.Response, error) {
	url := "ws" + strings.TrimPrefix(serverURL, "http") + "/api/v1/teams/" + strconv.FormatInt(teamID, 10) + "/ws"
	dialer := websocket.Dialer{Subprotocols: []string{"access_token", token}}
	return dialer.Dial(url, nil)
}
