package usermgmt

import (
	"context"
	"encoding/json/v2"
	"log/slog"
	"net/http"
	"strings"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	"github.com/larsartmann/cqrs-htmx/v4/httperr"
)

// botContextKeyType is the context key for the authenticated Bot.
type botContextKeyType struct{}

var botContextKey botContextKeyType //nolint:gochecknoglobals // sentinel type for context keys

// WithBot stores the authenticated Bot in the context.
func WithBot(ctx context.Context, bot *Bot) context.Context {
	return context.WithValue(ctx, botContextKey, bot)
}

// BotFromContext retrieves the authenticated Bot from the context.
// Returns the bot and true if found, or nil and false otherwise.
func BotFromContext(ctx context.Context) (*Bot, bool) {
	if ctx == nil {
		return nil, false
	}
	bot, ok := ctx.Value(botContextKey).(*Bot)
	return bot, ok
}

// NewAPITokenMiddleware returns HTTP middleware that authenticates requests
// via Bearer token against registered bots. When a valid API token is found,
// the corresponding Bot is stored in the request context AND the bot's machine
// actor is bridged into the actor chain, so every dispatch from a
// token-authenticated request is attributed to `bot:<id>` — never anonymous.
// An actor a consumer set upstream is never overridden.
//
// Requests without a Bearer token pass through unmodified — this middleware
// does not block unauthenticated requests. Chain it with session middleware
// or a RequireBot middleware for enforcement.
func NewAPITokenMiddleware(service *Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if !strings.HasPrefix(auth, bearerPrefix) {
				next.ServeHTTP(w, r)
				return
			}

			token := strings.TrimPrefix(auth, bearerPrefix)
			bot, ok := service.ResolveBotByToken(token)
			if !ok {
				next.ServeHTTP(w, r)
				return
			}

			slog.DebugContext(
				r.Context(), "bot authenticated via API token",
				"bot_id", bot.ID.Get(),
				"bot_name", bot.Name,
			)
			ctx := r.Context()
			if cqrshtmx.ActorIDFromContext(ctx).IsZero() {
				ctx = cqrshtmx.WithActorID(ctx, ActorIDFromBot(bot.ID))
			}
			r = r.WithContext(WithBot(ctx, bot))
			next.ServeHTTP(w, r)
		})
	}
}

// contentTypeProblem matches the RFC 9457 content type the root httperr
// package writes; redeclared because internal/hx is not importable here.
const contentTypeProblem = "application/problem+json; charset=utf-8"

// RequireBot is HTTP middleware that rejects requests without an authenticated
// Bot in the context. Use after NewAPITokenMiddleware to enforce bot-only routes.
//
// The rejection is the shared RFC 9457 problem document (application/problem+json)
// with the stable shared denial code "unauthorized" — the same document every
// other cqrs-htmx surface emits, because the consumer of a machine gate is a
// program that branches on the code, not a human reading prose.
func RequireBot(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := BotFromContext(r.Context()); !ok {
			status := httperr.MapError(identitymodel.ErrUnauthorized)
			w.Header().Set("Content-Type", contentTypeProblem)
			w.WriteHeader(status)
			_ = json.MarshalWrite(w, httperr.NewStructuredError(identitymodel.ErrUnauthorized, r))
			return
		}
		next.ServeHTTP(w, r)
	})
}
