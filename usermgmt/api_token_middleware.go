package usermgmt

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
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
// the corresponding Bot is stored in the request context.
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
			r = r.WithContext(WithBot(r.Context(), bot))
			next.ServeHTTP(w, r)
		})
	}
}

// RequireBot is HTTP middleware that rejects requests without an authenticated
// Bot in the context. Use after NewAPITokenMiddleware to enforce bot-only routes.
func RequireBot(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := BotFromContext(r.Context()); !ok {
			http.Error(w, "Unauthorized: valid API token required", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
