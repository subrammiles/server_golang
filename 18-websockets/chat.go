// Package chat is a realtime chat server with WebSockets (github.com/coder/websocket):
// rooms, presence, broadcast, input limits and heartbeats.
package chat

import (
	"context"
	"embed"
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

//go:embed index.html
var page embed.FS

type Message struct {
	Type   string   `json:"type"` // join, leave, message, error
	Room   string   `json:"room,omitempty"`
	From   string   `json:"from,omitempty"`
	Text   string   `json:"text,omitempty"`
	Online []string `json:"online,omitempty"`
}

type client struct {
	name, room string
	send       chan Message // buffered OUTBOX: a slow client never blocks the others
}

// Hub tracks connected clients. One mutex guards the map; each client has its own writer goroutine.
type Hub struct {
	mu        sync.Mutex
	clients   map[*client]bool
	heartbeat time.Duration
}

func NewHub(heartbeat time.Duration) *Hub {
	return &Hub{clients: map[*client]bool{}, heartbeat: heartbeat}
}

func (h *Hub) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /{$}", http.FileServerFS(page))
	mux.HandleFunc("GET /chat", h.serveWS)
	return mux
}

func (h *Hub) serveWS(w http.ResponseWriter, r *http.Request) {
	// Accept upgrades the HTTP connection to a WebSocket. OriginPatterns restricts which web pages may connect.
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{OriginPatterns: []string{"localhost:*", "127.0.0.1:*"}})
	if err != nil {
		return
	}
	conn.SetReadLimit(4096) // reject huge messages

	c := &client{
		name: limit(r.URL.Query().Get("name"), "anonymous"),
		room: limit(r.URL.Query().Get("room"), "lobby"),
		send: make(chan Message, 32),
	}
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	h.join(c)
	defer h.leave(c)

	go h.writeLoop(ctx, cancel, conn, c)

	for { // READ LOOP: one per connection
		// Read raw frames and decode ourselves: wsjson.Read would CLOSE the connection on invalid JSON,
		// but a typo from a client shouldn't cost them their connection.
		typ, data, err := conn.Read(ctx)
		if err != nil {
			return // closed or broken connection
		}
		var in Message
		if typ != websocket.MessageText || json.Unmarshal(data, &in) != nil {
			c.trySend(Message{Type: "error", Text: `messages must be JSON like {"text":"hi"}`})
			continue
		}
		text := strings.TrimSpace(in.Text)
		if text == "" {
			c.trySend(Message{Type: "error", Text: "text is required"})
			continue
		}
		h.broadcast(c.room, Message{Type: "message", Room: c.room, From: c.name, Text: limit(text, "")})
	}
}

// writeLoop is the ONLY goroutine writing to this connection (concurrent writes are not allowed).
// It also sends PINGs: a client that stops answering (network gone) is disconnected.
func (h *Hub) writeLoop(ctx context.Context, cancel context.CancelFunc, conn *websocket.Conn, c *client) {
	defer cancel()
	ticker := time.NewTicker(h.heartbeat)
	defer ticker.Stop()
	for {
		select {
		case msg := <-c.send:
			wctx, done := context.WithTimeout(ctx, 5*time.Second)
			err := wsjson.Write(wctx, conn, msg)
			done()
			if err != nil {
				return
			}
		case <-ticker.C:
			pctx, done := context.WithTimeout(ctx, h.heartbeat)
			err := conn.Ping(pctx)
			done()
			if err != nil {
				conn.Close(websocket.StatusGoingAway, "heartbeat timeout")
				return
			}
		case <-ctx.Done():
			conn.Close(websocket.StatusNormalClosure, "")
			return
		}
	}
}

func (h *Hub) join(c *client) {
	h.mu.Lock()
	h.clients[c] = true
	h.mu.Unlock()
	h.broadcast(c.room, Message{Type: "join", Room: c.room, From: c.name, Online: h.online(c.room)})
}

func (h *Hub) leave(c *client) {
	h.mu.Lock()
	delete(h.clients, c)
	h.mu.Unlock()
	h.broadcast(c.room, Message{Type: "leave", Room: c.room, From: c.name, Online: h.online(c.room)})
}

func (h *Hub) broadcast(room string, msg Message) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.clients {
		if c.room == room {
			c.trySend(msg)
		}
	}
}

func (h *Hub) online(room string) []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	var names []string
	for c := range h.clients {
		if c.room == room {
			names = append(names, c.name)
		}
	}
	slices.Sort(names)
	return names
}

// trySend never blocks: if a client's outbox is full it is too slow, and the message is dropped for it.
func (c *client) trySend(m Message) {
	select {
	case c.send <- m:
	default:
	}
}

func limit(s, fallback string) string {
	if s == "" {
		return fallback
	}
	if len(s) > 500 {
		return s[:500]
	}
	return s
}
