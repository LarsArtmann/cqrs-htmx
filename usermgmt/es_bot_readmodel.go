package usermgmt

import (
	"context"
	"sync"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/projection/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// Bot is the read-model representation of a bot (machine actor).
type Bot struct {
	ID        BotID    `json:"id"`
	Name      string   `json:"name"`
	OwnerID   UserID   `json:"owner_id"`
	TokenHash []byte   `json:"token_hash"`
	Scopes    []string `json:"scopes"`
	Deleted   bool     `json:"deleted"`
}

// BotReadModel is the projection-side store for bots.
// It indexes bots by aggregate ID for lookup by BotID, and provides
// FindByTokenHash for API token authentication middleware.
type BotReadModel struct {
	mu          sync.RWMutex
	bots        map[id.AggregateID]*Bot
	byTokenHash map[string]*Bot
}

// NewBotReadModel creates an empty BotReadModel.
func NewBotReadModel() *BotReadModel {
	return &BotReadModel{ //nolint:exhaustruct // mu starts zero-valued
		bots:        make(map[id.AggregateID]*Bot),
		byTokenHash: make(map[string]*Bot),
	}
}

func (m *BotReadModel) Name() string { return "bot-read-model" }

func (m *BotReadModel) EventTypes() []event.Type { return allBotEventTypes }

func (m *BotReadModel) Handle(_ context.Context, evt event.Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	aggID := evt.AggregateID()

	switch evt.Type() {
	case eventBotRegistered:
		p, err := unmarshalPayload[BotRegisteredPayload](evt)
		if err != nil {
			return errorfamily.WrapCorruption(
				err, "usermgmt.bot_readmodel.decode_failed",
				"decode BotRegistered in read model",
			)
		}
		scopes := make([]string, len(p.Scopes))
		copy(scopes, p.Scopes)
		tokenHashStr := string(p.TokenHash)
		bot := &Bot{
			ID:        NewBotID(aggID.String()),
			Name:      p.Name,
			OwnerID:   p.OwnerID,
			TokenHash: p.TokenHash,
			Scopes:    scopes,
			Deleted:   false,
		}
		m.bots[aggID] = bot
		m.byTokenHash[tokenHashStr] = bot

	case eventBotDeleted:
		if bot, ok := m.botsDelete(aggID); ok {
			delete(m.byTokenHash, string(bot.TokenHash))
		}

	default:
		return errorfamily.NewRejection(
			"usermgmt.bot_readmodel.unknown_event",
			"bot read model received unknown event type: "+string(evt.Type()),
		)
	}

	return nil
}

// botsDelete removes a bot from the bots map and returns it if found.
func (m *BotReadModel) botsDelete(aggID id.AggregateID) (*Bot, bool) {
	bot, ok := m.bots[aggID]
	if !ok {
		return nil, false
	}
	delete(m.bots, aggID)
	return bot, true
}

// FindByID returns the bot for the given aggregate ID.
func (m *BotReadModel) FindByID(aggID id.AggregateID) (*Bot, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	bot, ok := m.bots[aggID]
	return bot, ok
}

// FindByTokenHash returns the bot associated with the given token hash.
// Used by API token middleware to resolve a bearer token to a bot identity.
func (m *BotReadModel) FindByTokenHash(hash []byte) (*Bot, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	bot, ok := m.byTokenHash[string(hash)]
	return bot, ok
}

var _ projection.Projection = (*BotReadModel)(nil)
