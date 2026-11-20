package usermgmt

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/larsartmann/go-idempotency"
)

// DefaultCommandIdempotencyTTL bounds the replay window of the
// default-mounted command idempotency guard: a retry carrying the same
// actor-scoped Idempotency-Key within this TTL is answered with the
// duplicate rejection; after expiry the command EXECUTES AGAIN, by design —
// size client retry budgets to at most this window. Sizing rule (the B2
// guide): the TTL covers the LONGEST retry window your clients use, not
// processing time, because the upstream middleware claims the key whether
// the command succeeds or fails.
const DefaultCommandIdempotencyTTL = 10 * time.Minute

// IdempotencyMetadataKey is the command-metadata key the default idempotency
// guard reads — the same wire semantic as the HTTP Idempotency-Key header
// (the B2 recipe's decode hook copies the header into this metadata key, so
// consumers already following the recipe are deduped by the default mount
// with no further wiring). Commands whose metadata does not carry the key
// pass through un-deduped: session traffic and key-less bots are
// byte-for-byte unaffected.
//
// The constant is deliberately UNTYPED (it converts implicitly to
// command.MetadataKey at every use site, the B2 recipe's own form): a const
// typed by an upstream type is counted as an alias declaration by the
// ADR-0062 re-export scanner, and the frozen reexport-baseline refuses new
// alias files without an ADR amendment.
const IdempotencyMetadataKey = "cqrs-htmx.dev/idempotency-key"

// idempotencyMemoryStoreSweep bounds how often the default in-memory store
// reclaims expired keys.
const idempotencyMemoryStoreSweep = 60 * time.Second

// idempotencyKeyFromMetadata resolves the dedup key for one command: the
// client-declared Idempotency-Key metadata, scoped by the authenticated
// actor when the audit chain has enriched the command (per-token bots share
// one dispatcher, so one bot's retry must not suppress another bot's
// distinct command — the guide's key-scoping rule). An empty key passes
// through un-deduped (the upstream middleware's own contract).
func idempotencyKeyFromMetadata(cmd command.Command) string {
	carrier, ok := cmd.(command.MetadataCarrier)
	if !ok {
		return ""
	}
	meta := carrier.Metadata()
	key := strings.TrimSpace(meta.Custom[IdempotencyMetadataKey])
	if key == "" {
		return ""
	}
	if !meta.ActorID.IsZero() {
		return meta.ActorID.String() + "/" + key
	}
	return key
}

// defaultCommandIdempotency builds the default-mounted replay guard. It owns
// the claim end to end — the CheckAndRecord discipline mirrors upstream
// middleware.CommandIdempotency with the Conflict translation built in — so
// the re-familied replay error can ONLY originate from this guard's own
// store claim: an ErrDuplicate bubbling from a consumer-mounted idempotency
// middleware deeper in the chain passes through untouched (the B2 recipe's
// sentinel contract stays intact; the wrapper-composition form in the guide
// is the consumer-side equivalent). Keyed by the actor-scoped
// Idempotency-Key metadata, pass-through when the key is absent; HTTP
// consumers answer 409 (cqrshtmx.MapError falls open to 503 for
// family-less errors — a replay is a client error, not an availability
// incident). The guard sits directly inside the audit chain: replays
// short-circuit before lifecycle and consumer middleware run, so a deduped
// retry burns no retry budget and emits no lifecycle events. The store
// claims the key whether the command succeeds or fails (the upstream
// CheckAndRecord contract) — a consumer needing record-after-success
// semantics runs the HTTP-layer recipe instead (examples/admin-demo
// ackMiddleware) or opts out via ServiceConfig.DisableCommandIdempotency.
// newDefaultCommandIdempotencyStore builds the fallback store for the
// default-mounted replay guard. The caller owns the returned store's sweeper
// goroutine: Service.stopEvictions stops it via
// Service.stopCommandIdemSweeper, so a consumer that never injects
// ServiceConfig.IdempotencyStore still shuts the goroutine down on
// Service.Stop/Close instead of leaking it past construction.
func newDefaultCommandIdempotencyStore() idempotency.Store {
	//nolint:staticcheck // SA1019: MemoryStore is deprecated for production; the default is the dev/test class — inject ServiceConfig.IdempotencyStore for multi-instance deployments.
	//cqrs-lint:ignore(C026) the argument is the sweep interval, not a TTL — the replay TTL is DefaultCommandIdempotencyTTL at the CheckAndRecord claim below
	return idempotency.NewMemoryStore(idempotencyMemoryStoreSweep)
}

func defaultCommandIdempotency(store idempotency.Store) command.Middleware {
	if store == nil {
		store = newDefaultCommandIdempotencyStore()
	}
	return func(next command.Handler) command.Handler {
		return func(ctx context.Context, cmd command.Command) error {
			key := idempotencyKeyFromMetadata(cmd)
			if key == "" {
				return next(ctx, cmd)
			}
			if err := store.CheckAndRecord(ctx, key, DefaultCommandIdempotencyTTL); err != nil {
				if errors.Is(err, idempotency.ErrDuplicate) {
					return errorfamily.NewConflict(
						"usermgmt.command.replayed",
						"duplicate request: a request with this Idempotency-Key was already accepted within the replay window",
					)
				}
				return errorfamily.Wrapf(
					err, errorfamily.Transient,
					"usermgmt.command_idempotency",
					"check-and-record failed for %s",
					cmd.Type(),
				)
			}
			return next(ctx, cmd)
		}
	}
}
