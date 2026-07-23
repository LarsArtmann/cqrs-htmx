package usermgmt

import (
	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

type BotState = identitymodel.BotState

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
		return state, errorfamily.NewRejection(
			"usermgmt.bot.unknown_event",
			"foldBot received unknown event type: "+string(evt.Type()),
		)
	}

	return next, nil
}
