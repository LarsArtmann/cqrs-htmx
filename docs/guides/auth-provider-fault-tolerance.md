# Auth Provider Fault Tolerance

> How to handle transient and permanent failures from OAuth2, WebAuthn, and TOTP providers.

---

## The Problem

cqrs-htmx delegates authentication to external systems via interfaces:

- `WebAuthnProvider` — passkey registration and login ceremonies
- `OAuth2Provider` — OAuth2/OIDC login flows
- `TOTPProvider` — time-based one-time password verification

When these external systems fail (Google's OAuth endpoint is down, WebAuthn library throws), the library returns a `Transient` error (HTTP 503). But without a circuit breaker, every request queues a retry against a dead provider, wasting resources and increasing latency.

---

## The Library's Stance

cqrs-htmx correctly **does not force a circuit breaker**. Consumers have different requirements:

- Some want fail-fast (circuit breaker)
- Some want retry-with-backoff
- Some want custom fallback logic

The library returns `Transient` errors for provider failures. What you do with them is your call.

---

## Recommended Pattern: Circuit Breaker Wrapper

Wrap your auth provider implementation with a circuit breaker. The most popular Go library is [gobreaker](https://github.com/sony/gobreaker):

### Example: Wrapping an OAuth2 Provider

```go
import "github.com/sony/gobreaker"

type circuitBreakerOAuth2 struct {
    inner usermgmt.OAuth2Provider
    cb    *gobreaker.CircuitBreaker
}

func NewCircuitBreakerOAuth2(inner usermgmt.OAuth2Provider) usermgmt.OAuth2Provider {
    return &circuitBreakerOAuth2{
        inner: inner,
        cb: gobreaker.NewCircuitBreaker(gobreaker.Settings{
            Name:        "oauth2-provider",
            MaxRequests: 5,               // half-open: allow 5 test requests
            Interval:    60 * time.Second, // reset error count every 60s
            Timeout:     30 * time.Second, // open -> half-open after 30s
            ReadyToTrip: func(c gobreaker.Counts) bool {
                return c.ConsecutiveFailures > 5
            },
        }),
    }
}

func (c *circuitBreakerOAuth2) Exchange(ctx context.Context, code string) (*usermgmt.OAuth2UserInfo, error) {
    result, err := c.cb.Execute(func() (any, error) {
        return c.inner.Exchange(ctx, code)
    })
    if err != nil {
        return nil, err
    }
    return result.(*usermgmt.OAuth2UserInfo), nil
}
```

### Example: Wrapping a WebAuthn Provider

```go
type circuitBreakerWebAuthn struct {
    inner usermgmt.WebAuthnProvider
    cb    *gobreaker.CircuitBreaker
}

func NewCircuitBreakerWebAuthn(inner usermgmt.WebAuthnProvider) usermgmt.WebAuthnProvider {
    return &circuitBreakerWebAuthn{
        inner: inner,
        cb: gobreaker.NewCircuitBreaker(gobreaker.Settings{
            Name:    "webauthn-provider",
            Timeout: 30 * time.Second,
            ReadyToTrip: func(c gobreaker.Counts) bool {
                return c.ConsecutiveFailures > 3
            },
        }),
    }
}
```

---

## Transient vs Permanent Failures

Not all provider failures should trip the circuit breaker:

| Failure Type        | Trip Circuit? | Example                       |
| ------------------- | ------------- | ----------------------------- |
| Network timeout     | Yes           | OAuth provider is down        |
| Rate limited (429)  | Yes           | Too many requests to provider |
| Invalid credentials | No            | User entered wrong code       |
| User cancelled      | No            | User closed WebAuthn prompt   |
| Invalid grant       | No            | OAuth code expired            |

Distinguish by error type: `Transient` errors should count toward the circuit; `Rejection` errors should not.

```go
ReadyToTrip: func(c gobreaker.Counts) bool {
    // Only trip on transient failures, not user errors
    return c.ConsecutiveFailures > 5
},
OnStateChange: func(name string, from, to gobreaker.State) {
    slog.Warn("circuit breaker state change",
        "provider", name,
        "from", from.String(),
        "to", to.String())
},
```

---

## See Also

- Service Setup — How to inject auth providers into the Service (see `docs/guides/provider-implementation.md`)
- Error Mapping — How `Transient` errors map to HTTP 503 (see AGENTS.md "Error families → HTTP status")
