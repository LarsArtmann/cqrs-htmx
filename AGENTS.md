# cqrs-htmx

> **THIS IS A LIBRARY/SDK — NOT AN APPLICATION.** Consumers import this package into THEIR projects.

Go library that makes it easy to use go-cqrs-lite with HTMX, templ, and Casbin authorization. Framework-agnostic — works with `net/http`, Gin, Chi, etc.

## Quick Reference

| Item     | Value                                                                     |
| -------- | ------------------------------------------------------------------------- |
| Language | Go 1.26.4 (`GOEXPERIMENT=jsonv2` required)                                |
| Module   | `github.com/larsartmann/cqrs-htmx/v4`                                     |
| Test     | `nix run .#test` or `GOEXPERIMENT=jsonv2 go test ./... -count=1 -race`    |
| Build    | `nix run .#build` or `GOEXPERIMENT=jsonv2 go build ./...`                 |
| Lint     | `nix run .#lint` or `GOEXPERIMENT=jsonv2 golangci-lint run`               |
| Coverage | `nix run .#coverage` / `nix run .#coverage-gate` (root 90%, usermgmt 75%) |
| Fmt      | `nix fmt`                                                                 |
| DevShell | `nix develop`                                                             |

## Architecture

**Multi-module Go workspace.** 12 independent Go modules under one `go.work`:

- **Root** (`cqrs-htmx/v4`): Core library — HTTP handler builder, HTMX/SSE/WS helpers, authz (Casbin), CSRF, rate limiting, security headers, error mapping, pagination
- **usermgmt** (`usermgmt/v4`): Event-sourced CQRS user management (aggregates: User, Membership, Tenant, Bot). Passwordless — auth via WebAuthn/TOTP/OAuth2 behind interfaces
- **usermgmt/totp**, **usermgmt/webauthn**, **usermgmt/oauth2**: Independent auth strategy modules — satisfy `usermgmt` interfaces via structural typing
- **adminui** (`adminui/v4`): Ready-made admin dashboard (templ + HTMX)
- **loginpage** (`loginpage/v4`): Ready-made passwordless login page
- **integration_test**: Cross-module bridge tests
- **examples/**: `basic`, `datastar-demo`, `catalog-demo`, `admin-demo`

**Dependency direction:** Root → usermgmt is zero imports (clean boundary). Auth strategies → root/usermgmt via interfaces only. adminui/loginpage → root + usermgmt. Nothing depends on adminui or loginpage.

**Key dependencies:** go-cqrs-lite v4 (CQRS/event sourcing), casbin/v3 (authz), justinas/nosurf (CSRF), go-error-family (error classification), go-branded-id (typed IDs), a-h/templ (HTML templating), ginkgo/gomega (BDD testing).

## Key Patterns

- **Library principle:** Never enforce defaults consumers might disagree with (no mandatory CSP/HSTS/CSRF — all opt-in)
- **No stdlib error constructors:** `errors.New`/`fmt.Errorf` banned in non-test code. Use `event.New*/Wrap*/Wrapf/Newf` from go-cqrs-lite.
- **Error families → HTTP status:** Rejection(400), Conflict(409), Transient(503), Corruption(500), Infrastructure(503)
- **Duck-typing for external libs:** `Enforcer` (← casbin), `TemplComponent` (← templ), auth provider interfaces — consumers import what they need
- **authMode enum:** `authNone`/`authRequired`/`authAuthorized` — impossible states unrepresentable
- **templ `_templ.go` committed:** Generated files are committed so consumers run no codegen
- **Coverage gate:** Root ≥90%, usermgmt ≥74% (CI-enforced)

## Gotchas

- **`GOEXPERIMENT=jsonv2` is mandatory:** Build fails without it. Never add depguard rules banning `encoding/json/v2`.
- **`ActorID` differs by module:** Root's is `brandid.ID` (use `NewActorID("...")`). usermgmt's is a kind-discriminated struct (use `NewActorID(kind, raw)`).
- **CSRF middleware ordering:** `Chain(CSRFMiddleware, HTMXMiddleware, app.Middleware())` — CSRF first.
- **`HandlerConfig.Secure` is `*bool`:** nil defaults to true. Use `new(bool)` for false.
- **Registration is email-only:** No password field. Auth is exclusively via WebAuthn passkeys.
- **UserID types are aliased:** Root defines `type UserID = id.UserID` and usermgmt aliases the same type — they interoperate directly, no bridging needed.
- **Module path casing:** go-cqrs-lite uses lowercase `github.com/larsartmann/go-cqrs-lite`.
- **go-cqrs-lite v4.0.0 publishing bug:** go.mod files reference internal siblings with zero pseudo-versions. Consumers must explicitly `go get` ALL transitive go-cqrs-lite modules.
- **GOWORK=off for submodules:** `go.work` covers workspace; use `GOWORK=off` for per-module go.mod commands.
