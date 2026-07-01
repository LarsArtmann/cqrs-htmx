# usermgmt — Event-Sourced User Management for cqrs-htmx

## Description

Passwordless, event-sourced user management with WebAuthn/Passkey authentication,
RBAC authorization via Casbin, and opt-in event signing/encryption.

## Installation

```bash
go get github.com/larsartmann/cqrs-htmx/usermgmt/v4
```

## Event Signing & Encryption (Opt-in)

Signing and encryption are **zero-dependency by default** — consumers who don't
need crypto pull nothing extra. To enable, import `go-cqrs-lite/signing/v3` and/or
`go-cqrs-lite/encryption/v3` and wire them via `ServiceConfig`:

### Encryption-at-rest + bus-level signing (recommended)

```go
import (
    "github.com/larsartmann/go-cqrs-lite/encryption/v3"
    "github.com/larsartmann/go-cqrs-lite/signing/v3"
)

signer, _ := signing.NewHMAC(signingKey)       // 32+ bytes
cipher, _ := encryption.NewAES256GCM(encKey)   // exactly 32 bytes

svc, _ := usermgmt.NewService(usermgmt.ServiceConfig{
    StoreWrapper: func(s event.Store) (event.Store, error) {
        return encryption.NewEncryptedStore(s, cipher)
    },
    PublishMiddleware: []event.PublishMiddleware{signing.SignMiddleware(signer)},
    HandlerMiddleware: []event.Middleware{signing.RequireSignatureMiddleware(signer)},
})
```

### Asymmetric signing (Ed25519)

```go
pubKey, privKey, _ := signing.GenerateEd25519KeyPair()
signer, _ := signing.NewEd25519(privKey)
verifier, _ := signing.NewEd25519Verifier(pubKey)

svc, _ := usermgmt.NewService(usermgmt.ServiceConfig{
    PublishMiddleware: []event.PublishMiddleware{signing.SignMiddleware(signer)},
    HandlerMiddleware: []event.Middleware{signing.RequireSignatureMiddleware(verifier)},
})
```

See `docs/adr/0011-event-signing-encryption.md` for the full decision record.

## Development

```bash
# Build the project
nix run .#build

# Run tests
nix run .#test

# Run linter
nix run .#lint
```

## License

See [LICENSE](LICENSE) file for details.
