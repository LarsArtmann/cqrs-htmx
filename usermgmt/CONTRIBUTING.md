# Contributing to usermgmt

The user management submodule — a passwordless, event-sourced CQRS user management system.

## Module-Specific Setup

This is a **separate Go module** (`github.com/larsartmann/cqrs-htmx/usermgmt/v4`). It has its own `go.mod` and must be tested/built with `GOWORK=off`:

```bash
cd usermgmt
GOWORK=off GONOSUMCHECK='github.com/larsartmann/*' go build ./...
GOWORK=off GONOSUMCHECK='github.com/larsartmann/*' go test ./... -count=1 -race
GOWORK=off golangci-lint run
```

Or via Nix (sets `GOWORK=off` automatically):

```bash
nix run .#test-usermgmt
nix run .#lint   # covers root + usermgmt + adminui
```

## Architecture — Event-Sourced CQRS

All state changes are **events** persisted to an event store. State is reconstructed by folding events.

- **20 commands** embed `*command.BasicCommand` (ADR-0032) — structurally guarantees unique command IDs
- **21 event types** across 4 aggregates: User, Membership, Tenant, Bot
- **Pure domain layer**: `fold*()` reconstructs state, `decide*()` validates guards and emits events
- **Write path**: Service → CommandDispatcher → DeciderRepository.Execute (load → fold → decide → save → publish)
- **Read path**: Service queries `UserReadModel` projection
- **Authentication**: Passwordless — WebAuthn (passkeys), OAuth2/OIDC, TOTP MFA

### The fold/decide pattern

```go
// foldUser reconstructs state from events (pure, no I/O)
func foldUser(state UserState, evt event.Event) (UserState, error) { ... }

// decideRegisterUser validates guards and returns an event-producing closure
func decideRegisterUser(aggID id.AggregateID, email, display string, roles []string) decideFunc { ... }
```

### Testing the domain layer

Use the **scenario/v3 BDD DSL** for decider tests:

```go
scenario.Given[*RegisterUserCmd, UserState](t, foldUser, UserState{}).
    When(cmd, decide).
    Then(eventUserRegistered)
```

See `es_scenario_test.go` for patterns. Table-driven tests with standard `testing` for service/integration tests.

## Error Handling

**NO stdlib error constructors** — `errors.New`, `fmt.Errorf`, `errors.Join` are banned. Enforced by `nix run .#errorfamily`. Use `event.New*/Wrap*` from `go-cqrs-lite/event/v3`:

```go
return event.NewRejection("usermgmt.bad_input", "email is required")
return event.NewConflict("usermgmt.user_already_exists", "email taken")
```

## Key Dependencies

- **go-webauthn v0.17.4** — WebAuthn/Passkey ceremonies
- **casbin/casbin/v3** — RBAC authorization (via `Authz` wrapper)
- **pquerna/otp** — TOTP MFA (RFC 6238)
- **coreos/go-oidc + golang.org/x/oauth2** — OAuth2/OIDC social login
- **go-branded-id** — `UserID`, `TenantID`, `BotID` branded types

## PR Checklist

- [ ] `GOWORK=off go test ./... -count=1 -race` passes
- [ ] `GOWORK=off golangci-lint run` reports 0 issues
- [ ] No stdlib error constructors
- [ ] New commands embed `*command.BasicCommand`
- [ ] New events have corresponding `fold` + `decide` + projection handling
