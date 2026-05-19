package websocket

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	gws "github.com/gorilla/websocket"

	"github.com/Ze-uus/talent-backend/internal/ctxkeys"
	"github.com/Ze-uus/talent-backend/internal/domain"
)

// ─── Message types ────────────────────────────────────────────────────────────

type ws_message_type string

const (
	msg_anomaly    ws_message_type = "anomaly"
	msg_conversion ws_message_type = "conversion"
)

type WSMessage struct {
	Type      ws_message_type `json:"type"`
	Cycle_id  string          `json:"cycle_id"`
	Talent_id string          `json:"talent_id,omitempty"`
	Payload   any             `json:"payload"`
	Timestamp time.Time       `json:"timestamp"`
}

// ─── Hub ─────────────────────────────────────────────────────────────────────

type Hub struct {
	anomaly_ch    <-chan domain.AnomalyEvent
	conversion_ch <-chan domain.ConversionEvent

	// admin_clients: cycle_id → set of clients (see all events for that cycle)
	admin_clients map[string]map[*client]bool
	// talent_clients: talent_id:cycle_id → set of clients (own events only)
	talent_clients map[string]map[*client]bool

	register   chan *client
	unregister chan *client
	mu         sync.Mutex
	log        *slog.Logger
}

func NewHub(anomaly_ch <-chan domain.AnomalyEvent, conversion_ch <-chan domain.ConversionEvent, log *slog.Logger) *Hub {
	return &Hub{
		anomaly_ch:     anomaly_ch,
		conversion_ch:  conversion_ch,
		admin_clients:  make(map[string]map[*client]bool),
		talent_clients: make(map[string]map[*client]bool),
		register:       make(chan *client, 32),
		unregister:     make(chan *client, 32),
		log:            log,
	}
}

func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return

		case c := <-h.register:
			h.mu.Lock()
			if c.is_admin {
				if h.admin_clients[c.cycle_id] == nil {
					h.admin_clients[c.cycle_id] = make(map[*client]bool)
				}
				h.admin_clients[c.cycle_id][c] = true
			} else {
				key := c.talent_id + ":" + c.cycle_id
				if h.talent_clients[key] == nil {
					h.talent_clients[key] = make(map[*client]bool)
				}
				h.talent_clients[key][c] = true
			}
			h.mu.Unlock()

		case c := <-h.unregister:
			h.mu.Lock()
			if c.is_admin {
				if clients, ok := h.admin_clients[c.cycle_id]; ok {
					delete(clients, c)
					if len(clients) == 0 {
						delete(h.admin_clients, c.cycle_id)
					}
				}
			} else {
				key := c.talent_id + ":" + c.cycle_id
				if clients, ok := h.talent_clients[key]; ok {
					delete(clients, c)
					if len(clients) == 0 {
						delete(h.talent_clients, key)
					}
				}
			}
			h.mu.Unlock()

		case ev := <-h.conversion_ch:
			msg := WSMessage{
				Type:      msg_conversion,
				Cycle_id:  ev.Cycle_id,
				Talent_id: ev.Talent_id,
				Payload:   ev,
				Timestamp: time.Now().UTC(),
			}
			h.broadcastToAdmins(ev.Cycle_id, msg)
			h.broadcastToTalent(ev.Talent_id, ev.Cycle_id, msg)

		case ev := <-h.anomaly_ch:
			msg := WSMessage{
				Type:      msg_anomaly,
				Cycle_id:  ev.Cycle_id,
				Talent_id: ev.Talent_id,
				Payload:   ev,
				Timestamp: time.Now().UTC(),
			}
			h.broadcastToAdmins(ev.Cycle_id, msg)
		}
	}
}

func (h *Hub) broadcastToAdmins(cycle_id string, msg WSMessage) {
	b, err := json.Marshal(msg)
	if err != nil {
		return
	}
	h.mu.Lock()
	clients := h.admin_clients[cycle_id]
	h.mu.Unlock()
	for c := range clients {
		select {
		case c.send <- b:
		default:
			h.log.Warn("ws: admin client send buffer full, dropping message", "cycle_id", cycle_id)
		}
	}
}

func (h *Hub) broadcastToTalent(talent_id, cycle_id string, msg WSMessage) {
	b, err := json.Marshal(msg)
	if err != nil {
		return
	}
	key := talent_id + ":" + cycle_id
	h.mu.Lock()
	clients := h.talent_clients[key]
	h.mu.Unlock()
	for c := range clients {
		select {
		case c.send <- b:
		default:
			h.log.Warn("ws: talent client send buffer full, dropping message", "talent_id", talent_id)
		}
	}
}

// ─── WS upgrade ──────────────────────────────────────────────────────────────

var upgrader = gws.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// ServeAdmin upgrades an admin connection for a cycle's real-time event stream.
// Route: GET /ws/admin/{cycle_id}
func (h *Hub) ServeAdmin(w http.ResponseWriter, r *http.Request) {
	_, ok := ctxkeys.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthenticated", http.StatusUnauthorized)
		return
	}
	cycle_id := r.PathValue("cycle_id")
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.log.Error("ws admin upgrade failed", "err", err)
		return
	}
	c := &client{hub: h, conn: conn, send: make(chan []byte, 256), cycle_id: cycle_id, is_admin: true}
	h.register <- c
	go c.writePump()
	go c.readPump()
}

// ServeTalent upgrades a talent connection for their own cycle event stream.
// Route: GET /ws/talent/{cycle_id}
func (h *Hub) ServeTalent(w http.ResponseWriter, r *http.Request) {
	u, ok := ctxkeys.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthenticated", http.StatusUnauthorized)
		return
	}
	cycle_id := r.PathValue("cycle_id")
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.log.Error("ws talent upgrade failed", "err", err)
		return
	}
	c := &client{hub: h, conn: conn, send: make(chan []byte, 256), cycle_id: cycle_id, talent_id: u.ID, is_admin: false}
	h.register <- c
	go c.writePump()
	go c.readPump()
}

// ─── Client ───────────────────────────────────────────────────────────────────

type client struct {
	hub       *Hub
	conn      *gws.Conn
	send      chan []byte
	cycle_id  string
	talent_id string
	is_admin  bool
}

const (
	write_wait      = 10 * time.Second
	pong_wait       = 60 * time.Second
	ping_period     = (pong_wait * 9) / 10
	max_message_size = 512
)

func (c *client) readPump() {
	defer func() {
		c.hub.unregister <- c
		_ = c.conn.Close()
	}()
	c.conn.SetReadLimit(max_message_size)
	_ = c.conn.SetReadDeadline(time.Now().Add(pong_wait))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(pong_wait))
		return nil
	})
	for {
		if _, _, err := c.conn.ReadMessage(); err != nil {
			break
		}
	}
}

func (c *client) writePump() {
	ticker := time.NewTicker(ping_period)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()
	for {
		select {
		case msg, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(write_wait))
			if !ok {
				_ = c.conn.WriteMessage(gws.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(gws.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(write_wait))
			if err := c.conn.WriteMessage(gws.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
