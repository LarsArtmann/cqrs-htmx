package cqrshtmx

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/httputil"
)

//nolint:gochecknoglobals // context key singletons; one instance per key is standard.
var (
	ipAddressKeyInstance = contextKey[event.IPAddress]{name: "ip_address"}
	userAgentKeyInstance = contextKey[event.UserAgent]{name: "user_agent"}
	clientIDKeyInstance  = contextKey[id.ClientID]{name: "client_id"}
)

// WithIPAddress stores the client IP address in the context.
// Set automatically by [ContextEnrichmentMiddleware]; consumers only need
// this to inject an IP from a non-HTTP source (e.g., a queue consumer).
func WithIPAddress(ctx context.Context, ip event.IPAddress) context.Context {
	return ipAddressKeyInstance.WithValue(ctx, ip)
}

// IPAddressFromContext retrieves the client IP stored by [WithIPAddress]
// (normally captured from the request by [ContextEnrichmentMiddleware]).
// Returns the zero value if no IP is present.
func IPAddressFromContext(ctx context.Context) event.IPAddress {
	v, _ := ipAddressKeyInstance.FromContext(ctx)

	return v
}

// WithUserAgent stores the client User-Agent in the context.
// Set automatically by [ContextEnrichmentMiddleware]; consumers only need
// this to inject one from a non-HTTP source.
func WithUserAgent(ctx context.Context, ua event.UserAgent) context.Context {
	return userAgentKeyInstance.WithValue(ctx, ua)
}

// UserAgentFromContext retrieves the User-Agent stored by [WithUserAgent]
// (normally captured from the request by [ContextEnrichmentMiddleware]).
// Returns the zero value if no User-Agent is present.
func UserAgentFromContext(ctx context.Context) event.UserAgent {
	v, _ := userAgentKeyInstance.FromContext(ctx)

	return v
}

// HeaderClientID is the HTTP header carrying the persistent client/device ID
// for offline-first attribution. The browser sync client (sync-client.js)
// stamps it on every mutation with a localStorage-persisted ULID; the value
// is attribution metadata, not authentication — it is trivially spoofable.
// ("X-Client-Id" is the canonical Go header spelling; net/http canonicalizes
// on Get/Set, so the wire bytes are identical either way.)
const HeaderClientID = "X-Client-Id"

// WithClientID stores the client device ID in the context.
// Set automatically by [ContextEnrichmentMiddleware] from the
// [HeaderClientID] request header; consumers only need this to inject an ID
// from a non-HTTP source (e.g., a queue consumer replaying offline work).
func WithClientID(ctx context.Context, clientID id.ClientID) context.Context {
	return clientIDKeyInstance.WithValue(ctx, clientID)
}

// ClientIDFromContext retrieves the client device ID stored by
// [WithClientID] (normally captured from the [HeaderClientID] header).
// Returns the zero value if no client ID is present.
func ClientIDFromContext(ctx context.Context) id.ClientID {
	v, _ := clientIDKeyInstance.FromContext(ctx)

	return v
}

// enrichClientMetadata returns ctx with the client IP address and User-Agent
// captured from the request, for propagation into event metadata by
// [EventOptionsFromContext].
//
// The IP is extracted with [httputil.ClientIP] (X-Forwarded-For first entry,
// then X-Real-IP, then RemoteAddr — only trustworthy behind a proxy that
// overwrites those headers). Values that fail event.ParseIPAddress (e.g., a
// non-IP RemoteAddr) are logged at debug level and dropped, matching the
// invalid-header handling of the correlation-ID enrichment. An empty
// User-Agent is simply omitted.
//
// Privacy note: IP addresses are personal data under GDPR. They are stored in
// the request context only and reach durable storage solely when events are
// built with [EventOptionsFromContext] (the App dispatch path does this).
// Consumers who must not persist client IPs have two escape hatches: skip
// [ContextEnrichmentMiddleware] entirely (build the middleware chain without
// app.Middleware — request/correlation IDs then become your responsibility),
// or shadow the captured value with a zero one before dispatch
// (WithIPAddress(ctx, "") overwrites the earlier capture; zero values are
// never propagated).
func enrichClientMetadata(ctx context.Context, r *http.Request) context.Context {
	if clientIP, err := event.ParseIPAddress(httputil.ClientIP(r)); err != nil {
		slog.Debug(
			"cqrs-htmx: unparsable client IP",
			slog.String("remote_addr", r.RemoteAddr),
			slog.String("error", err.Error()),
		)
	} else if !clientIP.IsZero() {
		ctx = WithIPAddress(ctx, clientIP)
	}

	if ua := event.NewUserAgent(r.UserAgent()); !ua.IsZero() {
		ctx = WithUserAgent(ctx, ua)
	}

	// Offline-first attribution: the sync client stamps a persistent ULID on
	// mutations. Non-ULID values are logged at debug level and dropped,
	// matching the invalid-header handling of the other enrichments.
	if raw := r.Header.Get(HeaderClientID); raw != "" {
		if clientID, err := id.ParseClientID(raw); err != nil {
			slog.Debug(
				"cqrs-htmx: invalid client ID header",
				slog.String("header", HeaderClientID),
				slog.String("error", err.Error()),
			)
		} else {
			ctx = WithClientID(ctx, clientID)
		}
	}

	return ctx
}
