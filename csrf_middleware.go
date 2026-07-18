package cqrshtmx

import (
	"log/slog"
	"net"
	"net/http"
	"sync"

	"github.com/justinas/nosurf"
)

// CSRFMiddleware returns HTTP middleware that implements double-submit cookie
// CSRF protection with HTMX awareness.
//
// Uses justinas/nosurf internally for:
//   - Cryptographically secure token generation (crypto/rand)
//   - Per-request token masking (BREACH attack mitigation)
//   - Same-origin validation via Origin/Referer/Sec-Fetch-Site headers
//   - Trusted origins support for cross-domain use cases
//
// For GET/HEAD/OPTIONS/TRACE requests, the middleware ensures a CSRF token
// cookie exists and stores the masked token in context for use in templates.
//
// For state-changing methods (POST/PUT/PATCH/DELETE), it validates that the
// request includes a matching token in either:
//   - The X-CSRF-Token header (HTMX default)
//   - A form field named "csrf_token"
//
// Middleware ordering (important):
//
//	handler := cqrshtmx.Chain(
//	    cqrshtmx.CSRFMiddleware(cqrshtmx.CSRFConfig{}),
//	    cqrshtmx.HTMXMiddleware,
//	    app.Middleware(),
//	)(mux)
func CSRFMiddleware(cfg CSRFConfig) func(http.Handler) http.Handler {
	if err := cfg.Validate(); err != nil {
		slog.Error("cqrs-htmx: CSRFConfig validation failed", slog.String("error", err.Error()))
	}

	warnEmptyTrustedProxies(cfg)

	return func(next http.Handler) http.Handler {
		inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if token := nosurf.Token(r); token != "" {
				r = r.WithContext(WithCSRFToken(r.Context(), token))
			}

			next.ServeHTTP(w, r)
		})

		handler := nosurf.New(inner)
		configureNosurfHandler(handler, cfg)

		needsTranslation := cfg.headerName() != defaultCSRFHeaderName ||
			cfg.fieldName() != defaultCSRFFieldName

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			setPlaintextHTTPOrigin(r, cfg)

			// Translate custom header/field names to nosurf defaults.
			if needsTranslation {
				translateCSRFHeaders(r, cfg)
			}

			handler.ServeHTTP(w, r)
		})
	}
}

// warnEmptyTrustedProxies logs a one-time warning if the CSRF config has no
// trusted proxies configured. This moves the warning from the per-request hot
// path (isTrustedProxy) to middleware construction time, preventing log spam
// on every non-loopback HTTP request.
var warnTrustedProxiesOnce sync.Once

func warnEmptyTrustedProxies(cfg CSRFConfig) {
	warnTrustedProxiesOnce.Do(func() {
		if !cfg.AllowPlaintextBypass {
			return
		}

		if len(cfg.TrustedProxies) == 0 && len(cfg.TrustedProxiesCIDR) == 0 {
			slog.Warn(
				"cqrs-htmx: CSRFConfig.AllowPlaintextBypass is enabled with no TrustedProxies — " +
					"ALL non-TLS requests bypass origin validation. Set TrustedProxies or remove " +
					"AllowPlaintextBypass in production",
			)
		}
	})
}

// setPlaintextHTTPOrigin sets the Sec-Fetch-Site header to "same-origin" for
// plain HTTP requests without origin headers. This allows nosurf to skip
// origin validation, matching the behavior of gorilla/csrf's PlaintextHTTPRequest
// for HTTP deployments.
//
// Security: this shortcut ONLY applies when the request comes from a trusted
// proxy (configured via CSRFConfig.TrustedProxies / TrustedProxiesCIDR) or when
// the remote address is localhost. Without this guard, an attacker on a plain
// HTTP deployment could omit Origin/Referer/Sec-Fetch-Site headers and bypass
// nosurf's origin check entirely. Trusting any client without proxy validation
// is a CSRF bypass vulnerability.
func setPlaintextHTTPOrigin(r *http.Request, cfg CSRFConfig) {
	if !shouldBypassPlaintextOrigin(r, cfg) {
		return
	}

	r.Header.Set("Sec-Fetch-Site", "same-origin")
}

// shouldBypassPlaintextOrigin reports whether setPlaintextHTTPOrigin should
// auto-set Sec-Fetch-Site: same-origin for this request. The bypass is only
// granted when the request comes from loopback or a configured trusted proxy.
func shouldBypassPlaintextOrigin(r *http.Request, cfg CSRFConfig) bool {
	if r.TLS != nil {
		return false
	}

	if hasOriginHeader(r) {
		return false
	}

	remoteHost, remoteIP := remoteHostAndIP(r.RemoteAddr)
	if isLoopback(remoteIP) {
		return true
	}

	return isTrustedProxy(remoteHost, remoteIP, r.RemoteAddr, cfg)
}

// hasOriginHeader reports whether the request carries any header that
// identifies its origin (Sec-Fetch-Site, Origin, or Referer).
func hasOriginHeader(r *http.Request) bool {
	return r.Header.Get("Sec-Fetch-Site") != "" ||
		r.Header.Get("Origin") != "" ||
		r.Header.Get("Referer") != ""
}

// remoteHostAndIP splits a RemoteAddr into its host and parsed IP.
// Falls back to the raw address and a nil IP if parsing fails.
func remoteHostAndIP(remoteAddr string) (string, net.IP) {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}

	return host, net.ParseIP(host)
}

// isLoopback reports whether ip is a loopback address (including IPv4 127.0.0.0/8
// and IPv6 ::1).
func isLoopback(ip net.IP) bool {
	return ip != nil && ip.IsLoopback()
}

// isTrustedProxy returns true when remoteHost or remoteAddr matches an entry
// in cfg.TrustedProxies, or remoteIP falls within a cfg.TrustedProxiesCIDR
// network. When no proxies are configured, the result is governed by
// cfg.AllowPlaintextBypass: false (the secure zero-value default) denies the
// bypass to everyone except loopback; true restores the permissive pre-hardening
// behavior. A startup warning for the permissive mode is logged via
// warnEmptyTrustedProxies.
func isTrustedProxy(remoteHost string, remoteIP net.IP, remoteAddr string, cfg CSRFConfig) bool {
	if len(cfg.TrustedProxies) == 0 && len(cfg.TrustedProxiesCIDR) == 0 {
		return cfg.AllowPlaintextBypass
	}

	if remoteIP != nil {
		for _, cidr := range cfg.TrustedProxiesCIDR {
			if cidr.Contains(remoteIP) {
				return true
			}
		}
	}

	for _, trusted := range cfg.TrustedProxies {
		if trusted == remoteHost || trusted == remoteAddr {
			return true
		}
	}

	return false
}

// translateCSRFHeaders maps custom header/field names to nosurf's default
// "X-CSRF-Token" header. nosurf hardcodes its header and field names,
// so we translate before passing the request to nosurf.
func translateCSRFHeaders(r *http.Request, cfg CSRFConfig) {
	if cfg.headerName() != defaultCSRFHeaderName {
		if token := r.Header.Get(cfg.headerName()); token != "" {
			r.Header.Set(defaultCSRFHeaderName, token)

			return
		}
	}

	if cfg.fieldName() != defaultCSRFFieldName {
		if token := r.PostFormValue(cfg.fieldName()); token != "" {
			r.Header.Set(defaultCSRFHeaderName, token)
		}
	}
}

// CSRFResponseHeaderMiddleware returns HTTP middleware that automatically sets
// the X-CSRF-Token response header on every request. This eliminates the need
// for individual handlers to manually call resp.CSRFToken(token).
//
// Place this AFTER CSRFMiddleware in the chain so the token is already in context:
//
//	handler := cqrshtmx.Chain(
//	    cqrshtmx.CSRFMiddleware(cqrshtmx.CSRFConfig{}),
//	    cqrshtmx.CSRFResponseHeaderMiddleware,
//	    cqrshtmx.HTMXMiddleware,
//	    app.Middleware(),
//	)(mux)
//
// The header is only set when a token exists in the request context.
// For HTMX requests, the client reads this header and includes it in
// subsequent requests via hx-headers. For regular requests, the token
// is still available to server-side rendering.
func CSRFResponseHeaderMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if token := csrfTokenFromRequest(r); token != "" {
			w.Header().Set(defaultCSRFHeaderName, token)
		}

		next.ServeHTTP(w, r)
	})
}
