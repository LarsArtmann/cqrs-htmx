package cqrshtmx

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
)

// WriteJSON encodes v as JSON and writes it to w with the given HTTP status code
// and Content-Type: application/json header. Returns any encoding error.
func WriteJSON(w http.ResponseWriter, status int, v any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		return fmt.Errorf("encode JSON response: %w", err)
	}
	return nil
}

// ClientIP extracts the client IP address from the request using the following
// precedence: X-Forwarded-For (first entry), X-Real-IP, then RemoteAddr with
// net.SplitHostPort. This handles requests behind reverse proxies (nginx,
// Cloudflare, etc.) and direct connections.
//
// Warning: This function trusts X-Forwarded-For and X-Real-IP headers without
// validation. In deployments without a trusted reverse proxy, clients can spoof
// these headers to bypass IP-based rate limiting or logging. Only use behind a
// proxy that strips/overwrites these headers.
func ClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if ips := strings.Split(xff, ","); len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
