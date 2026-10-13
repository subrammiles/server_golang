package chat

import (
	"context"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

type testClient struct {
	conn  *websocket.Conn
	mu    sync.Mutex
	inbox []Message
}

func dial(t *testing.T, srv *httptest.Server, name, room string) *testClient {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/chat?name=" + name + "&room=" + room
	conn, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	c := &testClient{conn: conn}
	go func() {
		for {
			var m Message
			if err := wsjson.Read(context.Background(), conn, &m); err != nil {
				return
			}
			c.mu.Lock()
			c.inbox = append(c.inbox, m)
			c.mu.Unlock()
		}
	}()
	t.Cleanup(func() { conn.Close(websocket.StatusNormalClosure, "") })
	return c
}

func (c *testClient) waitFor(t *testing.T, match func(Message) bool) Message {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		c.mu.Lock()
		for _, m := range c.inbox {
			if match(m) {
				c.mu.Unlock()
				return m
			}
		}
		c.mu.Unlock()
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("expected message never arrived")
	return Message{}
}

func (c *testClient) has(match func(Message) bool) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return slices.ContainsFunc(c.inbox, match)
}

func newServer(t *testing.T) *httptest.Server {
	srv := httptest.NewServer(NewHub(time.Second).Handler())
	t.Cleanup(srv.Close)
	return srv
}

func TestBroadcastAndPresence(t *testing.T) {
	srv := newServer(t)
	asha := dial(t, srv, "asha", "lobby")
	asha.waitFor(t, func(m Message) bool { return m.Type == "join" && m.From == "asha" })
	ravi := dial(t, srv, "ravi", "lobby")

	joined := asha.waitFor(t, func(m Message) bool { return m.Type == "join" && m.From == "ravi" })
	if strings.Join(joined.Online, ",") != "asha,ravi" {
		t.Fatalf("online = %v", joined.Online)
	}

	wsjson.Write(context.Background(), ravi.conn, Message{Text: "  hello  "})
	got := asha.waitFor(t, func(m Message) bool { return m.Type == "message" })
	if got.From != "ravi" || got.Text != "hello" {
		t.Fatalf("got %+v", got)
	}

	ravi.conn.Close(websocket.StatusNormalClosure, "bye")
	asha.waitFor(t, func(m Message) bool { return m.Type == "leave" && m.From == "ravi" })
}

func TestRoomsAreIsolated(t *testing.T) {
	srv := newServer(t)
	red := dial(t, srv, "a", "red")
	blue := dial(t, srv, "b", "blue")
	red.waitFor(t, func(m Message) bool { return m.Type == "join" })
	blue.waitFor(t, func(m Message) bool { return m.Type == "join" })

	wsjson.Write(context.Background(), red.conn, Message{Text: "only red"})
	red.waitFor(t, func(m Message) bool { return m.Type == "message" })
	time.Sleep(50 * time.Millisecond)
	if blue.has(func(m Message) bool { return m.Type == "message" }) {
		t.Fatal("message leaked into another room")
	}
}

func TestInvalidMessagesGetAnErrorButKeepTheConnection(t *testing.T) {
	srv := newServer(t)
	c := dial(t, srv, "c", "lobby")
	c.conn.Write(context.Background(), websocket.MessageText, []byte("not json"))
	c.waitFor(t, func(m Message) bool { return m.Type == "error" && strings.Contains(m.Text, "JSON") })
	wsjson.Write(context.Background(), c.conn, Message{Text: ""})
	c.waitFor(t, func(m Message) bool { return m.Type == "error" && m.Text == "text is required" })

	wsjson.Write(context.Background(), c.conn, Message{Text: "still here"})
	c.waitFor(t, func(m Message) bool { return m.Type == "message" && m.Text == "still here" })
}
