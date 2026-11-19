package usermgmt

import (
	"context"
	"errors"
	"strings"
	"time"

	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/middleware/v4"
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
const IdempotencyMetadataKey command.MetadataKey = "cqrs-htmx.dev/idempotency-key"

// idempotencyMemoryStoreSweep bounds how often the default in-memory store
// reclaims expired keys.
const idempotencyMemoryStoreSweep = time.Minute

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

// defaultCommandIdempotency builds the default-mounted replay guard: keyed
// by the actor-scoped Idempotency-Key metadata, pass-through when the key is
// absent, duplicate sentinel re-familied to Conflict so HTTP consumers
// answer 409 (cqrshtmx.MapError falls open to 503 for family-less errors —
// a replay is a client error, not an availability incident). The guard sits
// directly inside the audit chain: replays short-circuit before lifecycle
// and consumer middleware run, so a deduped retry burns no retry budget and
// emits no lifecycle events. The store claims the key whether the command
// succeeds or fails (upstream CheckAndRecord contract) — a consumer needing
// record-after-success semantics runs the HTTP-layer recipe instead
// (examples/admin-demo ackMiddleware) or opts out via
// ServiceConfig.DisableCommandIdempotency.
func defaultCommandIdempotency(store idempotency.Store) command.Middleware {
	if store == nil {
		//nolint:staticcheck // SA1019: MemoryStore is deprecated for production; the default is the dev/test class — inject ServiceConfig.IdempotencyStore for multi-instance deployments.
		store = idempotency.NewMemoryStore(idempotencyMemoryStoreSweep)
	}
	dedup := middleware.CommandIdempotency(store, DefaultCommandIdempotencyTTL, idempotencyKeyFromMetadata)
	return func(next command.Handler) command.Handler {
		guarded := dedup(next)
		return func(ctx context.Context, cmd command.Command) error {
			err := guarded(ctx, cmd)
			if errors.Is(err, idempotency.ErrDuplicate) {
				return errorfamily.NewConflict(
					"usermgmt.command.replayed",
					"duplicate request: a request with this Idempotency-Key was already accepted within the replay window",
				)
			}
			return err
		}
	}
}
