// Package client is the unified transport facade. Business methods route to
// REST or WebSocket per capability; callers never choose the transport.
package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/bytetrade/hass-cli/internal/config"
)

// Client exposes Home Assistant capabilities over a REST + WS facade. The WS
// connection is established lazily on first use.
type Client struct {
	cfg  *config.Config
	rest *restClient

	wsOnce sync.Once
	ws     *wsConn
	wsErr  error
}

// New builds a client from resolved config. It does not open any connection.
func New(cfg *config.Config) *Client {
	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	return &Client{
		cfg:  cfg,
		rest: newRESTClient(cfg.RESTBaseURL(), cfg.Token, cfg.Insecure, timeout),
	}
}

// Close releases the WebSocket connection if one was opened.
func (c *Client) Close() {
	if c.ws != nil {
		c.ws.close()
	}
}

func (c *Client) wsConnect(ctx context.Context) (*wsConn, error) {
	c.wsOnce.Do(func() {
		c.ws, c.wsErr = dialWS(ctx, c.cfg.WebSocketURL(), c.cfg.Token)
	})
	return c.ws, c.wsErr
}

// --- generic passthrough -------------------------------------------------

// REST issues a raw REST request (method, path relative to /api).
func (c *Client) REST(ctx context.Context, method, path string, body any) (json.RawMessage, error) {
	return c.rest.do(ctx, method, path, body)
}

// WS issues a raw WebSocket command and returns its result payload.
func (c *Client) WS(ctx context.Context, payload map[string]any) (json.RawMessage, error) {
	conn, err := c.wsConnect(ctx)
	if err != nil {
		return nil, err
	}
	return conn.call(ctx, payload)
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

	subCtx, cancel := context.WithCancel(ctx)
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

// SupervisorAPI proxies a Supervisor endpoint through Core's supervisor/api WS
// command, so a regular admin token reaches /addons, /store, etc.
func (c *Client) SupervisorAPI(ctx context.Context, method, endpoint string, data map[string]any) (json.RawMessage, error) {
	payload := map[string]any{
		"type":     "supervisor/api",
		"endpoint": endpoint,
		"method":   method,
	}
	if data != nil {
		payload["data"] = data
	}
	return c.WS(ctx, payload)
}
