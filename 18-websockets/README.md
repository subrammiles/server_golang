# 18: Realtime with WebSockets

HTTP is request/response. A **WebSocket** is a long-lived, **two-way** connection (an upgraded HTTP request),
so the server can **push**: chat, notifications, live dashboards, multiplayer games.

## Concepts
- `websocket.Accept` upgrades an HTTP request (`github.com/coder/websocket`; `gorilla/websocket` is the other common choice).
  `OriginPatterns` controls which web pages may connect (cross-site WebSocket hijacking protection).
- **Goroutines per connection**: one **read loop**, and one **write loop** that is the only writer
  (WebSocket connections don't allow concurrent writes).
- **Hub**: a mutex-guarded set of clients, **rooms**, **presence** lists, broadcast.
- **Buffered outbox per client** + non-blocking send: one slow client can't stall everyone.
- **Input limits** (`SetReadLimit`, text length) and validation. Invalid JSON gets an error, and the connection stays.
- **Heartbeats** (`Ping`): drop clients whose network vanished without a close.
- **Context** cancels both loops together when either side ends.
- **Scaling out**: each instance knows only its own sockets. With several instances, relay messages through Redis
  pub/sub or a broker. **Server-Sent Events** are a simpler option when only the server pushes.
- **Testing**: `httptest.NewServer` + `websocket.Dial`, with real connections and no browser.

## Run
```bash
go run ./18-websockets/cmd/server      # open http://localhost:8080 in two tabs
go test -v -race ./18-websockets
```
