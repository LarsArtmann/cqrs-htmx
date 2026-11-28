package usermgmt

import (
	"context"
	"log/slog"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// BotService owns the bot aggregate: registration, deletion, lookup, token
// resolution, and the bot side of the user lifecycle (deleting or merging a
// user re-owns its bots). It is one of the focused sub-services ADR-0038
// splits out of [Service], which embeds it by value, so every method stays
// reachable as svc.RegisterBot(...) through promotion while new code may
// take the focused service directly (svc.BotService).
type BotService struct {
	botReadModel *BotReadModel
	tokenPepper  TokenPepper
	dispatcher   dispatcher
	logger       *slog.Logger
}

// RegisterBotRequest holds the parameters for registering a new bot.
// The raw API token is returned to the caller ONCE; only the HMAC hash is persisted.
type RegisterBotRequest struct {
	ID      BotID    `json:"id"`
	Name    string   `json:"name"`
	OwnerID UserID   `json:"owner_id"`
	Scopes  []string `json:"scopes"`
}

// RegisterBotResult contains the bot identity and the raw API token (shown once).
type RegisterBotResult struct {
	Bot   *Bot   `json:"bot"`
	Token string `json:"token"`
}

// TokenPepper is the server-side secret used for HMAC-SHA256 token hashing.
// It must be set before registering or authenticating bots, via
// ServiceConfig.TokenPepper. There is no setter; the pepper is read from the
// config exactly once at service construction, into the BotService.
type TokenPepper = []byte

// RegisterBot creates a new bot, generates a 256-bit API token, and dispatches
// the BotRegistered event with the HMAC hash of the token.
// The raw token is returned in the result and NEVER persisted or shown again.
func (s *BotService) RegisterBot(ctx context.Context, req RegisterBotRequest) (*RegisterBotResult, error) {
	if s.tokenPepper == nil {
		return nil, errorfamily.NewRejection(
			"usermgmt.bot.pepper_not_configured",
			"token pepper must be configured before registering bots (ServiceConfig.TokenPepper)",
		)
	}
	if req.ID.IsZero() {
		return nil, errorfamily.NewRejection("usermgmt.bot.id_required", "bot ID is required")
	}
	if req.Name == "" {
		return nil, errorfamily.NewRejection("usermgmt.bot.name_required", "bot name is required")
	}

	token, err := GenerateToken()
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err, "usermgmt.bot.token_generation_failed", "generate API token")
	}

	tokenHash := HashToken(token, s.tokenPepper)

	aggID, err := aggIDFromBot(req.ID)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err, "usermgmt.bot.id_conversion_failed", "convert bot ID")
	}

	err = s.dispatcher.Dispatch(ctx, NewRegisterBotCmd(
		aggID, req.Name, req.OwnerID, tokenHash, req.Scopes,
	))
	if err != nil {
		return nil, err //nolint:wrapcheck // decider returns typed domain errors
	}

	bot, ok := s.botReadModel.FindByID(aggID)
	if !ok {
		return nil, errorfamily.NewTransient("usermgmt.bot.read_model_missing", "bot not in read model after register")
	}
	return &RegisterBotResult{Bot: bot, Token: token}, nil
}

// DeleteBot permanently deletes a bot.
func (s *BotService) DeleteBot(ctx context.Context, botID BotID, reason string) error {
	aggID, err := aggIDFromBot(botID)
	if err != nil {
		return errorfamily.WrapInfrastructure(err, "usermgmt.bot.id_conversion_failed", "convert bot ID").
			WithContextAny("bot_id", botID)
	}
	return s.dispatcher.Dispatch( //nolint:wrapcheck // decider returns typed domain errors
		ctx,
		NewDeleteBotCmd(aggID, reason),
	)
}

// GetBot retrieves a bot by ID from the read model.
func (s *BotService) GetBot(_ context.Context, id BotID) (*Bot, error) {
	aggID, err := aggIDFromBot(id)
	if err != nil {
		return nil, errorfamily.WrapInfrastructure(err, "usermgmt.bot.id_conversion_failed", "convert bot ID")
	}
	bot, ok := s.botReadModel.FindByID(aggID)
	if !ok {
		return nil, errorfamily.NewRejection("usermgmt.bot.not_found", "bot not found")
	}
	return bot, nil
}

// ResolveBotByToken resolves a raw API token to a Bot identity.
// Used by API token authentication middleware.
func (s *BotService) ResolveBotByToken(token string) (*Bot, bool) {
	if s.tokenPepper == nil || token == "" {
		return nil, false
	}
	hash := HashToken(token, s.tokenPepper)
	return s.botReadModel.FindByTokenHash(hash)
}

// deleteBotsForUserBestEffort deletes all bots owned by a user.
// Errors are logged but not returned — the user is already deleted.
func (s *BotService) deleteBotsForUserBestEffort(ctx context.Context, userID UserID, reason string) {
	bots := s.botReadModel.FindByOwner(userID)
	for _, bot := range bots {
		aggID, err := aggIDFromBot(bot.ID)
		if err != nil {
			s.logger.Warn(
				"usermgmt: failed to convert bot ID on user deletion",
				"user_id", userID.Get(),
				"bot_id", bot.ID.Get(),
				"error", err,
			)
			continue
		}
		if err := s.dispatcher.Dispatch(ctx, NewDeleteBotCmd(aggID, reason)); err != nil {
			s.logger.Warn(
				"usermgmt: failed to delete bot on user deletion",
				"user_id", userID.Get(),
				"bot_id", bot.ID.Get(),
				"error", err,
			)
		}
	}
}

// transferBotsToSurvivor re-owns every bot the loser owns. An unknown
// owner on the bot side answers the decider's own typed error; a bot already
// owned by the survivor is a decider no-op.
func (s *BotService) transferBotsToSurvivor(ctx context.Context, survivorID, loserID UserID, reason string) error {
	for _, bot := range s.botReadModel.FindByOwner(loserID) {
		aggID, err := aggIDFromBot(bot.ID)
		if err != nil {
			return err
		}
		if err := s.dispatcher.Dispatch(ctx, identitymodel.NewTransferBotCmd(aggID, survivorID, reason)); err != nil {
			return classifyDispatchError(err, loserID,
				"bot_id", bot.ID.Get(),
				"survivor_id", survivorID.Get().String(),
			)
		}
	}
	return nil
}

func aggIDFromBot(botID BotID) (id.StreamID, error) {
	aggID, err := id.ParseStreamID(botID.Get())
	if err != nil {
		return id.StreamID{}, errorfamily.Wrapf(
			err,
			event.Infrastructure,
			"usermgmt.invalid_bot_id",
			"invalid BotID for AggregateID conversion",
		)
	}
	return aggID, nil
}
