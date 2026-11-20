package legalparty

import errorfamily "github.com/larsartmann/go-error-family"

var (
	// ErrBotActorForbidden is returned when a bot (or unattributed) actor
	// tries to open, update, or close a legal party. Actors are not
	// billable parties, and a machine credential must never be the
	// responsible party for billing master data.
	ErrBotActorForbidden = errorfamily.NewRejection("legalparty.actor.bot_forbidden",
		"legal parties are maintained by human, system, or service actors; a bot or unattributed actor cannot open, update, or close one")
	// ErrLegalPartyNotFound is returned when a legal party command targets
	// a stream that was never opened.
	ErrLegalPartyNotFound = errorfamily.NewRejection("legalparty.legal_party.not_found",
		"legal party does not exist")
	// ErrLegalPartyClosed is returned when a closed (terminal, retained)
	// legal party is updated, or closed a second time.
	ErrLegalPartyClosed = errorfamily.NewConflict("legalparty.legal_party.closed",
		"legal party is closed and can no longer be changed")
	// ErrRegistryNotFound is returned by registry adapters when no registry
	// entry matches the query.
	ErrRegistryNotFound = errorfamily.NewRejection("legalparty.registry.not_found",
		"no registry entry matches the reference")
	// ErrRegistryUnavailable is returned by registry adapters when the
	// registry cannot be reached or answered with a server-side failure.
	ErrRegistryUnavailable = errorfamily.NewTransient("legalparty.registry.unavailable",
		"registry lookup failed; try again later")
)
