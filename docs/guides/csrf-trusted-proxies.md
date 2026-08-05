# CSRF TrustedProxies Production Setup

> How to configure `CSRFConfig.TrustedProxies` for common reverse proxy setups.

## Why TrustedProxies Matters

The CSRF middleware needs to know if a request's `Origin`/`Sec-Fetch-Site` header
can be trusted. When traffic flows through a reverse proxy (Nginx, Cloudflare,
Docker), the proxy may strip or rewrite these headers.

Without `TrustedProxies`, the middleware defaults to **allowing plaintext HTTP
requests from loopback only** and logs a warning for other sources. In production,
you must configure `TrustedProxies` so the middleware can correctly identify
same-origin requests behind your proxy.

## Common Setups

### Nginx (single server)

```go
csrf := httputil.CSRFMiddleware(httputil.CSRFConfig{
    TrustedProxies: []string{"127.0.0.1"}, // Nginx on same host
})
```

Nginx config:

```nginx
location / {
    proxy_pass http://localhost:8080;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header Origin $http_origin;  # pass through origin
}
```

### Docker (containerized)

```go
csrf := httputil.CSRFMiddleware(httputil.CSRFConfig{
    TrustedProxies: []string{
        "172.16.0.0/12",  // Docker bridge network
    },
})
```

### Cloudflare (CDN/WAF)

```go
csrf := httputil.CSRFMiddleware(httputil.CSRFConfig{
    TrustedProxies: []string{
        "173.245.48.0/20",
        "103.21.244.0/22",
        "103.22.200.0/22",
        // ... full Cloudflare IP range
        // See: https://www.cloudflare.com/ips/
    },
})
```

### Kubernetes (Ingress Controller)

```go
csrf := httputil.CSRFMiddleware(httputil.CSRFConfig{
    TrustedProxies: []string{
        "10.0.0.0/8",     // Kubernetes pod/service network
    },
})
```

## Verification

After configuring TrustedProxies, verify:

1. Requests from your proxy are NOT flagged as CSRF violations
2. Requests from untrusted IPs ARE still blocked
3. The warning log about empty TrustedProxies no longer appears

## Format

`TrustedProxies` accepts:

- Single IP: `"127.0.0.1"`
- CIDR range: `"10.0.0.0/8"`
- Mix of both: `[]string{"127.0.0.1", "10.0.0.0/8"}`
