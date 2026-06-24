package client

import (
	"context"
	"errors"
	"net"
	"net/url"
	"strings"
)

// FriendlyMessage turns a transport error into actionable, human-readable text.
// It augments HTTP, WebSocket, and connectivity failures with a short hint while
// preserving the underlying detail. Unrecognized errors are returned verbatim.
func FriendlyMessage(err error) string {
	if err == nil {
		return ""
	}

	var httpErr *HTTPError
	if errors.As(err, &httpErr) {
		switch {
		case httpErr.Status == 401 || httpErr.Status == 403:
			return httpErr.Error() + " (token rejected or lacks permission; check HASS_TOKEN is a valid admin long-lived token)"
		case httpErr.Status == 404:
			return httpErr.Error() + " (not found; check the path or resource exists)"
		case httpErr.Status >= 500:
			return httpErr.Error() + " (Home Assistant internal error; check the HA logs)"
		default:
			return httpErr.Error()
		}
	}

	if isAuthError(err) {
		return err.Error() + " (token rejected or lacks permission; use an admin long-lived token)"
	}

	if isConnectivityError(err) {
		return err.Error() + " (cannot reach HASS_SERVER; check the address and that Home Assistant is online)"
	}

	return err.Error()
}

// isConnectivityError reports whether err looks like a network/timeout failure
// rather than an application-level error from Home Assistant.
func isConnectivityError(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return true
	}
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return true
	}
	msg := strings.ToLower(err.Error())
	for _, frag := range []string{"connection refused", "no such host", "deadline exceeded", "websocket dial", "i/o timeout", "network is unreachable"} {
		if strings.Contains(msg, frag) {
			return true
		}
	}
	return false
}
