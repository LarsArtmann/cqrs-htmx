package usermgmt

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

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
// It must be set before registering or authenticating bots. Set via
// ServiceConfig.TokenPepper or by calling Service.SetTokenPepper.
type TokenPepper = []byte

// RegisterBot creates a new bot, generates a 256-bit API token, and dispatches
// the BotRegistered event with the HMAC hash of the token.
// The raw token is returned in the result and NEVER persisted or shown again.
func (s *Service) RegisterBot(ctx context.Context, req RegisterBotRequest) (*RegisterBotResult, error) {
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
func (s *Service) DeleteBot(ctx context.Context, botID BotID, reason string) error {
	aggID, err := aggIDFromBot(botID)
	if err != nil {
		return errorfamily.WrapInfrastructure(err, "usermgmt.bot.id_conversion_failed", "convert bot ID")
	}
	return s.dispatcher.Dispatch( //nolint:wrapcheck // decider returns typed domain errors
		ctx,
		NewDeleteBotCmd(aggID, reason),
	)
}

// GetBot retrieves a bot by ID from the read model.
func (s *Service) GetBot(_ context.Context, id BotID) (*Bot, error) {
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
func (s *Service) ResolveBotByToken(token string) (*Bot, bool) {
	if s.tokenPepper == nil || token == "" {
		return nil, false
	}
	hash := HashToken(token, s.tokenPepper)
	return s.botReadModel.FindByTokenHash(hash)
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
