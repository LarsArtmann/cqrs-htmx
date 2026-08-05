# Security

## Overview

cqrs-htmx provides defense-in-depth security for CQRS + HTMX Go applications:

- **CSRF Protection**: Double-submit cookie pattern via justinas/nosurf with origin validation
- **Authorization**: Casbin RBAC/ABAC with HTMX-aware redirect handling
- **Rate Limiting**: Token-bucket per-key limiter with 429 responses
- **Security Headers**: Content-Type nosniff, X-Frame-Options, XSS-Protection
- **Open Redirect Protection**: URL sanitization on all redirects
- **Strong Identity**: ULID-backed UserID and CorrelationID types

---

## CSRF Protection

### How It Works

1. **Token Generation**: cryptographically secure random tokens via `crypto/rand`
2. **Cookie Storage**: token stored in a cookie (not HttpOnly — readable by JS for double-submit)
3. **Origin Validation**: validates `Origin` and `Referer` headers to prevent cross-site requests
4. **Validation**: state-changing methods (POST/PUT/PATCH/DELETE) require matching token in header or form field

### Configuration

```go
csrfMW := httputil.CSRFMiddleware(httputil.CSRFConfig{
    Secure:        true,                     // HTTPS only
    SameSite:      http.SameSiteStrictMode,
    TrustedOrigins: []string{"https://example.com"},
})
```

**Critical settings:**

| Setting                          | Risk if Wrong                                                     |
| -------------------------------- | ----------------------------------------------------------------- |
| `Secure=false` + HTTPS site      | Browser rejects cookie (if site is HTTPS)                         |
| `SameSite=None` + `Secure=false` | Browser rejects cookie (modern browsers require Secure with None) |

---

## Authorization

### Casbin Integration

```go
enforcer, _ := casbin.NewEnforcer("model.conf", "policy.csv")
app, _ := cqrshtmx.New(cqrshtmx.Config{
    Enforcer:        enforcer,
    UserIDExtractor: extractUserIDFromJWT,
})

// Require specific permission
app.Command("DeleteGame", cqrshtmx.Authorize("game", "delete"), decodeDeleteGame())

// Require authentication only
app.Query("GetProfile", cqrshtmx.RequireAuth(), decodeProfileQuery())
```

### Auth Error Handling

- **HTMX requests**: returns `HX-Redirect: /login` (303 See Other) instead of error body
- **Regular requests**: returns 401/403 with plain text or JSON error

---

## Rate Limiting

```go
middleware := cqrshtmx.RateLimiterMiddleware(cqrshtmx.RateLimiterConfig{
    Limit:        100,
    Window:       time.Minute,
    Burst:        20,
    KeyExtractor: cqrshtmx.KeyExtractorFromRemoteAddr(),
})
```

**Known limitation**: The internal per-key limiter map grows unbounded. For deployments with many unique keys (e.g., per-IP limiting on public-facing services), entries are evicted after 10 minutes of inactivity (TTL-based cleanup). For extremely high-cardinality key spaces, consider wrapping with an external rate limiter (Redis, etc.).

---

## Responsible Disclosure

If you discover a security vulnerability in cqrs-htmx:

1. **DO NOT** open a public issue
2. Email the maintainer directly at `security@lars.software`
3. Include:
   - Description of the vulnerability
   - Steps to reproduce
   - Impact assessment
   - Suggested fix (if any)

You can expect:

- Acknowledgment within 48 hours
- Initial assessment within 7 days
- Fix release within 30 days for critical issues

---

## Hardening Checklist

- [ ] Set `CSRFConfig.Secure=true` for HTTPS deployments
- [ ] Use `SameSiteStrictMode` or `SameSiteLaxMode` (not `None` unless cross-domain)
- [ ] Configure rate limiting for all public endpoints
- [ ] Use `SecurityHeadersMiddleware` on all responses
- [ ] Run `govulncheck` regularly: `go install golang.org/x/vuln/cmd/govulncheck@latest && govulncheck ./...`

---

## Known Limitations

1. **Rate limiter memory**: per-key map has TTL eviction (10 min) but can still grow large under attack
2. **Reverse proxy scheme detection**: nosurf validates Origin headers; plain HTTP behind TLS-terminating proxies may need `TrustedOrigins` configuration
3. **UserID format**: strongly typed to ULID; consumers using UUID or integer IDs must adapt

---

## Dependencies Security

| Dependency             | Purpose         | Risk Level                 |
| ---------------------- | --------------- | -------------------------- |
| justinas/nosurf        | CSRF protection | Low (simple, well-audited) |
| casbin/casbin/v3       | Authorization   | Low (actively maintained)  |
| go-error-family        | Error handling  | Low (maintained)           |
| golang.org/x/time/rate | Rate limiting   | Low (Go x/ repo)           |

Run `govulncheck ./...` regularly to detect known vulnerabilities in dependencies.
