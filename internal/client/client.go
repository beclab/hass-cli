// Package client is the unified transport facade. Business methods route to
// REST or WebSocket per capability; callers never choose the transport.
package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/beclab/hass-cli/internal/config"
)

// Client exposes Home Assistant capabilities over a REST + WS facade. The WS
// connection is established lazily on first use.
type Client struct {
	cfg     *config.Config
	rest    *restClient
	timeout time.Duration

	wsMu sync.Mutex
	ws   *wsConn

	supervisorOnce sync.Once
	hasSupervisor  bool
	supervisorErr  error
}

// New builds a client from resolved config. It does not open any connection.
func New(cfg *config.Config) *Client {
	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	return &Client{
		cfg:     cfg,
		timeout: timeout,
		rest:    newRESTClient(cfg.RESTBaseURL(), cfg.Token, cfg.Insecure, timeout),
	}
}

// withTimeout derives a context bounded by the configured timeout, but only
// when the caller's context has no deadline of its own. Streaming callers
// (subscriptions) must not go through here.
func (c *Client) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if c.timeout <= 0 {
		return ctx, func() {}
	}
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, c.timeout)
}

// Close releases the WebSocket connection if one was opened.
func (c *Client) Close() {
	c.wsMu.Lock()
	defer c.wsMu.Unlock()
	if c.ws != nil {
		c.ws.close()
	}
}

// wsConnect returns a live WS connection, dialing lazily. If a prior connection
// failed or was closed, a subsequent call re-dials rather than staying poisoned
// (there is no background auto-reconnect; a dropped stream still surfaces).
func (c *Client) wsConnect(ctx context.Context) (*wsConn, error) {
	c.wsMu.Lock()
	defer c.wsMu.Unlock()
	if c.ws != nil && !c.ws.isClosed() {
		return c.ws, nil
	}
	dialCtx, cancel := c.withTimeout(ctx)
	defer cancel()
	ws, err := dialWS(dialCtx, c.cfg.WebSocketURL(), c.cfg.Token)
	if err != nil {
		return nil, err
	}
	c.ws = ws
	return c.ws, nil
}

// --- generic passthrough -------------------------------------------------

// REST issues a raw REST request (method, path relative to /api).
func (c *Client) REST(ctx context.Context, method, path string, body any) (json.RawMessage, error) {
	return c.rest.do(ctx, method, path, body)
}

// WS issues a raw WebSocket command and returns its result payload. The call is
// bounded by the configured timeout unless the caller already set a deadline.
func (c *Client) WS(ctx context.Context, payload map[string]any) (json.RawMessage, error) {
	conn, err := c.wsConnect(ctx)
	if err != nil {
		return nil, err
	}
	callCtx, cancel := c.withTimeout(ctx)
	defer cancel()
	return conn.call(callCtx, payload)
}

// Subscribe streams events for a WS subscription command until ctx is done.
func (c *Client) Subscribe(ctx context.Context, payload map[string]any, handler func(json.RawMessage) error) error {
	conn, err := c.wsConnect(ctx)
	if err != nil {
		return err
	}
	return conn.subscribe(ctx, payload, handler)
}

// --- core capabilities (REST-preferred) ----------------------------------

// Config returns the running instance configuration.
func (c *Client) Config(ctx context.Context) (json.RawMessage, error) {
	return c.rest.do(ctx, "GET", "config", nil)
}

// States returns all entity states.
func (c *Client) States(ctx context.Context) (json.RawMessage, error) {
	return c.rest.do(ctx, "GET", "states", nil)
}

// State returns one entity's state.
func (c *Client) State(ctx context.Context, entityID string) (json.RawMessage, error) {
	return c.rest.do(ctx, "GET", "states/"+entityID, nil)
}

// SetState overwrites the state machine entry for an entity (does not drive a
// device); mirrors POST /api/states/{entity_id}.
func (c *Client) SetState(ctx context.Context, entityID string, body any) (json.RawMessage, error) {
	return c.rest.do(ctx, "POST", "states/"+entityID, body)
}

// Services returns the service catalog (domain -> services -> fields).
func (c *Client) Services(ctx context.Context) (json.RawMessage, error) {
	return c.rest.do(ctx, "GET", "services", nil)
}

// CallService invokes domain.service with the given data payload.
func (c *Client) CallService(ctx context.Context, domain, service string, data map[string]any) (json.RawMessage, error) {
	return c.rest.do(ctx, "POST", fmt.Sprintf("services/%s/%s", domain, service), data)
}

// FireEvent fires a custom event with optional data.
func (c *Client) FireEvent(ctx context.Context, eventType string, data map[string]any) (json.RawMessage, error) {
	return c.rest.do(ctx, "POST", "events/"+eventType, data)
}

// RenderTemplate renders a Jinja template server-side.
func (c *Client) RenderTemplate(ctx context.Context, tmpl string) (string, error) {
	raw, err := c.rest.do(ctx, "POST", "template", map[string]any{"template": tmpl})
	if err != nil {
		return "", err
	}
	// The template endpoint returns a raw string body (not JSON-wrapped).
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s, nil
	}
	return string(raw), nil
}

// History returns state history; pathSuffix is appended to /api/history/period.
func (c *Client) History(ctx context.Context, pathSuffix string) (json.RawMessage, error) {
	return c.rest.do(ctx, "GET", "history/period"+pathSuffix, nil)
}

// Logbook returns logbook entries; pathSuffix is appended to /api/logbook.
func (c *Client) Logbook(ctx context.Context, pathSuffix string) (json.RawMessage, error) {
	return c.rest.do(ctx, "GET", "logbook"+pathSuffix, nil)
}

// ErrorLog returns the raw error log text.
func (c *Client) ErrorLog(ctx context.Context) (string, error) {
	raw, err := c.rest.do(ctx, "GET", "error_log", nil)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// --- registry capabilities (WS-only) -------------------------------------

// ListRegistry returns entries for a registry, e.g. "area", "device", "entity",
// "floor", "label" via config/<name>_registry/list.
func (c *Client) ListRegistry(ctx context.Context, name string) (json.RawMessage, error) {
	return c.WS(ctx, map[string]any{"type": fmt.Sprintf("config/%s_registry/list", name)})
}

// SystemHealthInfo collects integration system-health data. The
// system_health/info command is subscription-style: HA acknowledges with an
// empty result, then streams an "initial" snapshot, per-key "update" events,
// and a terminating "finish" event. This assembles them into a single object.
func (c *Client) SystemHealthInfo(ctx context.Context) (json.RawMessage, error) {
	conn, err := c.wsConnect(ctx)
	if err != nil {
		return nil, err
	}

	domains := map[string]map[string]any{}
	errFinished := errors.New("finished")

	// system_health/info is semantically one-shot (it terminates on "finish"),
	// so bound it by the configured timeout rather than streaming forever.
	tCtx, tCancel := c.withTimeout(ctx)
	defer tCancel()
	subCtx, cancel := context.WithCancel(tCtx)
	defer cancel()

	subErr := conn.subscribe(subCtx, map[string]any{"type": "system_health/info"}, func(ev json.RawMessage) error {
		var env struct {
			Type    string          `json:"type"`
			Data    json.RawMessage `json:"data"`
			Domain  string          `json:"domain"`
			Key     string          `json:"key"`
			Success *bool           `json:"success"`
			Error   json.RawMessage `json:"error"`
		}
		if err := json.Unmarshal(ev, &env); err != nil {
			return err
		}
		switch env.Type {
		case "initial":
			var snapshot map[string]map[string]any
			if err := json.Unmarshal(env.Data, &snapshot); err != nil {
				return err
			}
			for d, entry := range snapshot {
				domains[d] = entry
			}
		case "update":
			entry := domains[env.Domain]
			if entry == nil {
				entry = map[string]any{}
				domains[env.Domain] = entry
			}
			info, _ := entry["info"].(map[string]any)
			if info == nil {
				info = map[string]any{}
				entry["info"] = info
			}
			if env.Success != nil && *env.Success {
				var v any
				_ = json.Unmarshal(env.Data, &v)
				info[env.Key] = v
			} else {
				var v any
				_ = json.Unmarshal(env.Error, &v)
				info[env.Key] = v
			}
		case "finish":
			return errFinished
		}
		return nil
	})
	if subErr != nil && !errors.Is(subErr, errFinished) {
		return nil, subErr
	}
	return json.Marshal(domains)
}

// --- supervisor (via Core proxy) -----------------------------------------

// HasSupervisor reports whether the instance is a Supervised/HA OS install,
// detected the same way the frontend does: the "hassio" component being loaded.
// The result is cached for the client's lifetime.
func (c *Client) HasSupervisor(ctx context.Context) (bool, error) {
	c.supervisorOnce.Do(func() {
		raw, err := c.rest.do(ctx, "GET", "config", nil)
		if err != nil {
			c.supervisorErr = err
			return
		}
		var cfg struct {
			Components []string `json:"components"`
		}
		if err := json.Unmarshal(raw, &cfg); err != nil {
			c.supervisorErr = err
			return
		}
		for _, comp := range cfg.Components {
			if comp == "hassio" {
				c.hasSupervisor = true
				break
			}
		}
	})
	return c.hasSupervisor, c.supervisorErr
}

// SupervisorAPI proxies a Supervisor endpoint through Core's supervisor/api WS
// command, so a regular admin token reaches /addons, /store, etc. An
// authorization failure is rewrapped with an actionable hint.
func (c *Client) SupervisorAPI(ctx context.Context, method, endpoint string, data map[string]any) (json.RawMessage, error) {
	payload := map[string]any{
		"type":     "supervisor/api",
		"endpoint": endpoint,
		"method":   method,
	}
	if data != nil {
		payload["data"] = data
	}
	raw, err := c.WS(ctx, payload)
	if err != nil && isAuthError(err) {
		return nil, fmt.Errorf("token lacks the admin privileges required for Supervisor access; use an admin long-lived token: %w", err)
	}
	return raw, err
}

// isAuthError matches WS errors that indicate the token is not allowed to run a
// command (unauthorized / admin-required), so callers can hint at a token swap.
func isAuthError(err error) bool {
	var wsErr *wsError
	if errors.As(err, &wsErr) {
		if wsErr.Code == "unauthorized" {
			return true
		}
		msg := strings.ToLower(wsErr.Message)
		return strings.Contains(msg, "unauthorized") || strings.Contains(msg, "admin")
	}
	return false
}
