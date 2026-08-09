package realtime

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"task-manager/internal/events"
)

type Message struct {
	EventType  events.EventType `json:"event_type"`
	TeamID     int64            `json:"team_id"`
	EntityType string           `json:"entity_type"`
	EntityID   int64            `json:"entity_id"`
	OccurredAt time.Time        `json:"occurred_at"`
}

type Hub struct {
	mu     sync.RWMutex
	teams  map[int64]map[int64]map[*client]struct{}
	closed bool
}

type client struct {
	hub       *Hub
	conn      *websocket.Conn
	teamID    int64
	userID    int64
	send      chan []byte
	closeOnce sync.Once
}

func NewHub() *Hub {
	return &Hub{teams: make(map[int64]map[int64]map[*client]struct{})}
}

func (h *Hub) Publish(event events.Event) {
	teamID, entityType, entityID, err := events.Route(event)
	if err != nil {
		return
	}
	payload, err := json.Marshal(Message{
		EventType:  event.EventType,
		TeamID:     teamID,
		EntityType: entityType,
		EntityID:   entityID,
		OccurredAt: event.OccurredAt,
	})
	if err != nil {
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, userClients := range h.teams[teamID] {
		for connection := range userClients {
			select {
			case connection.send <- payload:
			default:
				go connection.close()
			}
		}
	}
}

func (h *Hub) Add(conn *websocket.Conn, teamID, userID int64) bool {
	connection := &client{
		hub:    h,
		conn:   conn,
		teamID: teamID,
		userID: userID,
		send:   make(chan []byte, 32),
	}

	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		_ = conn.Close()
		return false
	}
	if h.teams[teamID] == nil {
		h.teams[teamID] = make(map[int64]map[*client]struct{})
	}
	if h.teams[teamID][userID] == nil {
		h.teams[teamID][userID] = make(map[*client]struct{})
	}
	h.teams[teamID][userID][connection] = struct{}{}
	h.mu.Unlock()

	go connection.writeLoop()
	go connection.readLoop()
	return true
}

func (h *Hub) remove(connection *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	userClients := h.teams[connection.teamID][connection.userID]
	delete(userClients, connection)
	if len(userClients) == 0 {
		delete(h.teams[connection.teamID], connection.userID)
	}
	if len(h.teams[connection.teamID]) == 0 {
		delete(h.teams, connection.teamID)
	}
}

func (h *Hub) Close() {
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		return
	}
	h.closed = true
	connections := make([]*client, 0)
	for _, teamClients := range h.teams {
		for _, userClients := range teamClients {
			for connection := range userClients {
				connections = append(connections, connection)
			}
		}
	}
	h.mu.Unlock()

	for _, connection := range connections {
		connection.close()
	}
}

func (c *client) close() {
	c.closeOnce.Do(func() {
		c.hub.remove(c)
		close(c.send)
		_ = c.conn.Close()
	})
}

func (c *client) readLoop() {
	defer c.close()
	c.conn.SetReadLimit(1024)
	_ = c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	})
	for {
		if _, _, err := c.conn.ReadMessage(); err != nil {
			return
		}
	}
}

func (c *client) writeLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.close()
	}()
	for {
		select {
		case payload, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, nil)
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, payload); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
