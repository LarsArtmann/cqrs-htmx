# middleware-showcase

Every [httputil](https://github.com/larsartmann/httputil) HTTP middleware in
one runnable, validated `MiddlewareStack`: recovery, security headers,
metrics, CORS, client IP, rate limiting, compression, and ETag. Complements
[middleware-demo](../middleware-demo/README.md), which shows go-cqrs-lite
*dispatch* middleware (retry, circuit breaker) instead.

```bash
GOEXPERIMENT=jsonv2 go run .   # http://localhost:8099
```
