package usermgmt

import (
	"github.com/larsartmann/go-cqrs-lite/codec/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

func decideLinkExternalAccount(
	aggID id.StreamID,
	provider, subject, email, displayName string,
) func(UserState, event.Version) ([]event.Event, error) {
	return func(state UserState, version event.Version) ([]event.Event, error) {
		if err := requireExists(state, "link_external_account"); err != nil {
			return nil, err
		}
		if state.Deleted {
			return nil, errorfamily.NewRejection("usermgmt.link_external_account.deleted",
				"cannot link external account to deleted user")
		}
		if provider == "" || subject == "" {
			return nil, errorfamily.NewRejection("usermgmt.link_external_account.invalid",
				"provider and subject are required")
		}
		for _, ea := range state.ExternalAccounts {
			if ea.Provider == provider && ea.Subject == subject {
				return nil, errorfamily.NewConflict("usermgmt.external_account_already_linked",
					"external account already linked to this user")
			}
		}
		evt, err := event.New(
			eventExternalAccountLinked, aggID, aggregateTypeUser, version.Increment(),
			ExternalAccountLinkedPayload{
				SchemaVersion: currentSchemaVersion,
				ExternalAccountCore: ExternalAccountCore{
					Provider:    provider,
					Subject:     subject,
					Email:       email,
					DisplayName: displayName,
				},
			},
			event.WithCodec(codec.JSONCodec{}),
		)
		if err != nil {
			return nil, errorfamily.WrapInfrastructure(
				err,
				"usermgmt.link_external_account.event_failed",
				"create ExternalAccountLinked event",
			)
		}
		return []event.Event{evt}, nil
	}
}

func decideUnlinkExternalAccount(
	aggID id.StreamID,
	provider, subject string,
) func(UserState, event.Version) ([]event.Event, error) {
	return func(state UserState, version event.Version) ([]event.Event, error) {
		if err := requireExists(state, "unlink_external_account"); err != nil {
			return nil, err
		}
		if state.Deleted {
			return nil, errorfamily.NewRejection("usermgmt.unlink_external_account.deleted",
				"cannot unlink external account from deleted user")
		}
		found := false
		for _, ea := range state.ExternalAccounts {
			if ea.Provider == provider && ea.Subject == subject {
				found = true
				break
			}
		}
		if !found {
			return nil, errorfamily.NewRejection("usermgmt.external_account_not_found",
				"external account not linked to this user")
		}
		// Last-auth-method guard: reject if removing this would leave the user
		// with zero WebAuthn credentials and zero other external accounts.
		if len(state.Credentials) == 0 && len(state.ExternalAccounts) <= 1 {
			return nil, errorfamily.NewRejection("usermgmt.unlink_external_account.last_auth_method",
				"cannot remove the last authentication method")
		}
		evt, err := event.New(
			eventExternalAccountUnlinked, aggID, aggregateTypeUser, version.Increment(),
			ExternalAccountUnlinkedPayload{
				SchemaVersion: currentSchemaVersion,
				Provider:      provider,
				Subject:       subject,
			},
			event.WithCodec(codec.JSONCodec{}),
		)
		if err != nil {
			return nil, errorfamily.WrapInfrastructure(
				err,
				"usermgmt.unlink_external_account.event_failed",
				"create ExternalAccountUnlinked event",
			)
		}
		return []event.Event{evt}, nil
	}
}
