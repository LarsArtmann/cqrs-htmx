package usermgmt

import (
	"github.com/larsartmann/go-cqrs-lite/event/v3"
)

// BotState is the aggregate state for a Bot (machine actor), reconstructed
// by folding events.
type BotState struct {
	Name      string
	OwnerID   string
	TokenHash []byte
	Scopes    []string
	Deleted   bool
}

// Exists reports whether the bot has been registered (has at least one event).
func (s BotState) Exists() bool {
	return s.Name != "" && !s.Deleted
}

// foldBot applies an event to the current BotState, returning the new state.
func foldBot(state BotState, evt event.Event) (BotState, error) {
	next := state

	switch evt.Type() {
	case eventBotRegistered:
		p, err := unmarshalPayload[BotRegisteredPayload](evt)
		if err != nil {
			return state, err
		}
		scopes := make([]string, len(p.Scopes))
		copy(scopes, p.Scopes)
		next.Name = p.Name
		next.OwnerID = p.OwnerID
		next.TokenHash = p.TokenHash
		next.Scopes = scopes
		next.Deleted = false

	case eventBotDeleted:
		next.Deleted = true

	default:
		return state, event.NewRejection(
			"usermgmt.bot.unknown_event",
			"foldBot received unknown event type: "+string(evt.Type()),
		)
	}

	return next, nil
}
