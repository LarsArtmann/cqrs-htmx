# usermgmt/webauthn

WebAuthn/passkey authentication for [usermgmt](../README.md), built on
[go-webauthn]. An independent Go module — import it only if you want
passkeys; core usermgmt carries zero auth dependencies.

```go
import usermgmtwebauthn "github.com/larsartmann/cqrs-htmx/usermgmt/webauthn/v4"

passkeys, err := usermgmtwebauthn.New(usermgmtwebauthn.Config{
    RPID:      "example.com",
    RPOrigins: []string{"https://example.com"},
})
if err != nil {
    log.Fatal(err)
}
svc, err := usermgmt.NewService(usermgmt.ServiceConfig{
    WebAuthn: passkeys,
})
```

Inject the provider as `ServiceConfig.WebAuthn` (the interface lives in
usermgmt; this package is the reference implementation). Passkeys are the
default passwordless flow; enrollment ceremonies are owner-session-gated
(ADR-0055). Config pointers and the full auth-strategy matrix:
[usermgmt/README.md](../README.md).

[go-webauthn]: https://github.com/go-webauthn/webauthn
