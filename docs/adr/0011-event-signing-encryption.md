# ADR 0011: Opt-in Event Signing & Encryption

## Status: Accepted

## Context

`go-cqrs-lite` ships two companion security modules:

- **`signing/v2`** — HMAC-SHA256 and Ed25519 event signatures via
  `SignMiddleware` (publish-side) and `VerifyMiddleware` / `RequireSignatureMiddleware`
  (handle-side).
- **`encryption/v2`** — AES-256-GCM and XChaCha20-Poly1305 authenticated
  encryption via `EncryptMiddleware` / `DecryptMiddleware` (bus-level) and
  `NewEncryptedStore` (store-level transparent encryption-at-rest).

`usermgmt` owns the event-sourced infrastructure (event store + bus) and creates
in-memory defaults inside `NewService`. Consumers had no seam to inject these
security layers: they would have to bypass `NewService` entirely and re-wire the
store, bus, repository, and projections themselves — losing the "easy" promise of
cqrs-htmx.

The question: how to make signing/encryption **easy opt-in** without forcing a
dependency on every consumer.

## Decision

Add three dependency-free fields to `ServiceConfig`, applied at the correct
points in `NewService`'s setup sequence:

| Field               | Type                                     | Applied via                           | Typical use                                                |
| ------------------- | ---------------------------------------- | ------------------------------------- | ---------------------------------------------------------- |
| `StoreWrapper`      | `func(event.Store) (event.Store, error)` | Before journal detection + repository | `encryption.NewEncryptedStore` (encryption-at-rest)        |
| `PublishMiddleware` | `[]event.PublishMiddleware`              | `bus.UsePublish`                      | `signing.SignMiddleware`, `encryption.EncryptMiddleware`   |
| `HandlerMiddleware` | `[]event.Middleware`                     | `bus.Use`                             | `signing.VerifyMiddleware`, `encryption.DecryptMiddleware` |

### Why dependency-free seams (not hard imports)

1. **True opt-in** — consumers who don't need crypto pull zero new dependencies.
   `signing/v2` is stdlib-only, but `encryption/v2` brings `golang.org/x/crypto`.
   Keeping the seam generic means the cost is paid only by those who opt in.
2. **Consistent with existing patterns** — cqrs-htmx already duck-types
   `casbin.Enforcer` via the local `Enforcer` interface and `templ.Component` via
   `TemplComponent`. The middleware slices follow the same seam philosophy.
3. **Strategy-agnostic** — works with HMAC, Ed25519, AES-GCM, XChaCha20, or any
   future/custom implementation. The library imposes no algorithm choice.

### Ordering guarantees

`NewService` applies the hooks in this order:

1. **StoreWrapper** — wraps the store _before_ journal detection and repository
   creation. This ensures persisted events are protected on write and unwrapped
   on load. The wrapper must implement `event.Journal` when the inner store does
   (for projection catch-up); `NewEncryptedStore` already does.
2. **PublishMiddleware + HandlerMiddleware** — applied to the bus _before_
   projections subscribe. The memory bus rebuilds its middleware chain on each
   `Use`/`UsePublish`, so middleware wraps all current and future handlers
   regardless of subscription timing — but applying before subscription is
   cleaner and matches the bus lifecycle.

### Recommended patterns

**Store-level encryption + bus-level signing (recommended for usermgmt):**

```go
signer, _ := signing.NewHMAC(key)
cipher, _ := encryption.NewAES256GCM(encKey)

svc, _ := usermgmt.NewService(usermgmt.ServiceConfig{
    StoreWrapper: func(s event.Store) (event.Store, error) {
        return encryption.NewEncryptedStore(s, cipher)
    },
    PublishMiddleware: []event.PublishMiddleware{signing.SignMiddleware(signer)},
    HandlerMiddleware: []event.Middleware{signing.RequireSignatureMiddleware(signer)},
})
```

Store-level encryption keeps in-process projections seeing plaintext (they
subscribe to the bus, which carries plaintext), while persisted events are
ciphertext. Signing on the bus provides tamper detection in transit.

**Bus-level sign+encrypt (encryption-in-transit):**

```go
PublishMiddleware: []event.PublishMiddleware{
    signing.SignMiddleware(signer),       // sign plaintext first
    encryption.EncryptMiddleware(enc),     // then encrypt
},
HandlerMiddleware: []event.Middleware{
    encryption.DecryptMiddleware(enc),     // decrypt first
    signing.RequireSignatureMiddleware(signer), // then verify
},
```

Middleware runs outermost-first (registration order). For sign+encrypt, sign
the plaintext _before_ encrypting; for decrypt+verify, decrypt _before_ verifying.

## Consequences

**Positive:**

- Zero new dependencies for consumers who don't use crypto.
- One-line opt-in for the most common patterns.
- Correct middleware/store ordering is handled by `NewService`, not the consumer.
- Works with any signing/encryption strategy, including custom implementations.

**Negative:**

- Consumers must import `signing/v2` / `encryption/v2` themselves and construct
  the primitives. This is intentional (opt-in) but slightly more boilerplate
  than a "batteries" API that accepted raw keys.
- Store-level signing is not directly available (the signing module provides bus
  middleware, not a store wrapper). Consumers needing signed-at-rest events can
  compose a custom `StoreWrapper`.

**Root module (`cqrs-htmx`):** The root `App` does not own a bus or store, so no
integration is added there. Consumers using the root module without `usermgmt`
wire their own bus/store and apply signing/encryption directly via go-cqrs-lite.

## Verification

- 6 unit tests in `usermgmt/service_security_test.go` verify the hooks are
  applied, ordered correctly, and that projections survive.
- 3 integration tests in `integration_test/signing_encryption_test.go` exercise
  the real `signing/v2` + `encryption/v2` modules end-to-end: encryption-at-rest
  (ciphertext verified in raw store), decrypt-on-load (second Service
  reconstructs state), bus-level sign+encrypt, and Casbin projection survival.
