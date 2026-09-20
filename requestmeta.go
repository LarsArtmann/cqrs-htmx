package cqrshtmx

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/httputil"
)

//nolint:gochecknoglobals // context key singletons; one instance per key is standard.
var (
	ipAddressKeyInstance = contextKey[event.IPAddress]{name: "ip_address"}
	userAgentKeyInstance = contextKey[event.UserAgent]{name: "user_agent"}
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
	if ip, err := event.ParseIPAddress(httputil.ClientIP(r)); err != nil {
		slog.Debug(
			"cqrs-htmx: unparsable client IP",
			slog.String("remote_addr", r.RemoteAddr),
			slog.String("error", err.Error()),
		)
	} else if !ip.IsZero() {
		ctx = WithIPAddress(ctx, ip)
	}

	if ua := event.NewUserAgent(r.UserAgent()); !ua.IsZero() {
		ctx = WithUserAgent(ctx, ua)
	}

	return ctx
}
