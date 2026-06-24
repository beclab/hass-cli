package client

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/coder/websocket"
)

// wsConn manages a single authenticated Home Assistant WebSocket connection,
// multiplexing id-based request/response calls and subscriptions over it.
type wsConn struct {
	conn   *websocket.Conn
	nextID int64

	mu       sync.Mutex
	pending  map[int64]chan wsMessage
	subs     map[int64]chan wsMessage
	closed   bool
	closeErr error
}

// wsMessage is the envelope used for every frame on the HA WebSocket API.
type wsMessage struct {
	ID      int64           `json:"id,omitempty"`
	Type    string          `json:"type"`
	Success *bool           `json:"success,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Event   json.RawMessage `json:"event,omitempty"`
	Error   *wsError        `json:"error,omitempty"`
	HAVersion string        `json:"ha_version,omitempty"`
}

type wsError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *wsError) Error() string { return fmt.Sprintf("%s: %s", e.Code, e.Message) }

// dialWS opens the connection and performs the auth_required -> auth -> auth_ok
// handshake using a long-lived access token.
func dialWS(ctx context.Context, wsURL, token string) (*wsConn, error) {
	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("websocket dial %s: %w", wsURL, err)
	}
	conn.SetReadLimit(32 << 20)

	w := &wsConn{
		conn:    conn,
		pending: make(map[int64]chan wsMessage),
		subs:    make(map[int64]chan wsMessage),
	}

	var hello wsMessage
	if err := w.read(ctx, &hello); err != nil {
		conn.CloseNow()
		return nil, err
	}
	if hello.Type != "auth_required" {
		conn.CloseNow()
		return nil, fmt.Errorf("unexpected first frame %q (want auth_required)", hello.Type)
	}

	if err := w.write(ctx, map[string]any{"type": "auth", "access_token": token}); err != nil {
		conn.CloseNow()
		return nil, err
	}

	var authResp wsMessage
	if err := w.read(ctx, &authResp); err != nil {
		conn.CloseNow()
		return nil, err
	}
	if authResp.Type != "auth_ok" {
		conn.CloseNow()
		return nil, fmt.Errorf("authentication failed: %s", authResp.Type)
	}

	go w.readLoop()
	return w, nil
}

func (w *wsConn) read(ctx context.Context, v *wsMessage) error {
	_, data, err := w.conn.Read(ctx)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

func (w *wsConn) write(ctx context.Context, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return w.conn.Write(ctx, websocket.MessageText, data)
}

// readLoop fans incoming frames out to the waiting call or subscription channel.
func (w *wsConn) readLoop() {
	for {
		_, data, err := w.conn.Read(context.Background())
		if err != nil {
			w.fail(err)
			return
		}
		var msg wsMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}

		w.mu.Lock()
		if msg.Type == "event" {
			if ch, ok := w.subs[msg.ID]; ok {
				w.mu.Unlock()
				ch <- msg
				continue
			}
		}
		if ch, ok := w.pending[msg.ID]; ok {
			delete(w.pending, msg.ID)
			w.mu.Unlock()
			ch <- msg
			continue
		}
		w.mu.Unlock()
	}
}

func (w *wsConn) fail(err error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return
	}
	w.closed = true
	w.closeErr = err
	for _, ch := range w.pending {
		close(ch)
	}
	for _, ch := range w.subs {
		close(ch)
	}
}

// call sends a command and waits for its single result frame.
func (w *wsConn) call(ctx context.Context, payload map[string]any) (json.RawMessage, error) {
	id := atomic.AddInt64(&w.nextID, 1)
	ch := make(chan wsMessage, 1)

	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return nil, fmt.Errorf("connection closed: %w", w.closeErr)
	}
	w.pending[id] = ch
	w.mu.Unlock()

	payload["id"] = id
	if err := w.write(ctx, payload); err != nil {
		return nil, err
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case msg, ok := <-ch:
		if !ok {
			return nil, fmt.Errorf("connection closed: %w", w.closeErr)
		}
		if msg.Error != nil {
			return nil, msg.Error
		}
		if msg.Success != nil && !*msg.Success {
			return nil, fmt.Errorf("command %q failed", payload["type"])
		}
		return msg.Result, nil
	}
}

// subscribe registers a subscription and streams event payloads to handler
// until ctx is cancelled. It blocks for the lifetime of the subscription.
func (w *wsConn) subscribe(ctx context.Context, payload map[string]any, handler func(json.RawMessage) error) error {
	id := atomic.AddInt64(&w.nextID, 1)
	resultCh := make(chan wsMessage, 1)
	eventCh := make(chan wsMessage, 64)

	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return fmt.Errorf("connection closed: %w", w.closeErr)
	}
	w.pending[id] = resultCh
	w.subs[id] = eventCh
	w.mu.Unlock()

	payload["id"] = id
	if err := w.write(ctx, payload); err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case msg, ok := <-resultCh:
		if !ok {
			return fmt.Errorf("connection closed: %w", w.closeErr)
		}
		if msg.Error != nil {
			return msg.Error
		}
	}

	for {
		select {
		case <-ctx.Done():
			w.unsubscribe(id)
			return ctx.Err()
		case ev, ok := <-eventCh:
			if !ok {
				return fmt.Errorf("connection closed: %w", w.closeErr)
			}
			if err := handler(ev.Event); err != nil {
				w.unsubscribe(id)
				return err
			}
		}
	}
}

func (w *wsConn) unsubscribe(id int64) {
	w.mu.Lock()
	delete(w.subs, id)
	w.mu.Unlock()
	ctx := context.Background()
	uid := atomic.AddInt64(&w.nextID, 1)
	_ = w.write(ctx, map[string]any{"id": uid, "type": "unsubscribe_events", "subscription": id})
}

func (w *wsConn) close() {
	w.mu.Lock()
	closed := w.closed
	w.closed = true
	w.mu.Unlock()
	if !closed {
		_ = w.conn.Close(websocket.StatusNormalClosure, "")
	}
}
