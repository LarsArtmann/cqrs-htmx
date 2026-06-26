package usermgmt

import (
	"github.com/larsartmann/go-cqrs-lite/decider/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/stack/v3"
)

// aggregateRepositories groups the four decider repositories that every
// usermgmt stack preset wires up (User, Membership, Tenant, Bot). Centralised
// so that adding a new aggregate means changing one struct instead of every
// stack-setup file.
type aggregateRepositories struct {
	User       *decider.Repository[UserState]
	Membership *decider.Repository[MembershipState]
	Tenant     *decider.Repository[TenantState]
	Bot        *decider.Repository[BotState]
}

// buildStackRepositories creates the four aggregate repositories from a
// go-cqrs-lite stack Bundle. On failure, the bundle is closed before the
// error is returned so the caller never sees a half-built bundle.
func buildStackRepositories(bundle *stack.Bundle) (*aggregateRepositories, error) {
	user, err := stack.Repository(bundle, UserDecider())
	if err != nil {
		_ = bundle.Close()
		return nil, event.WrapTransient(err, "internal", "create user repository")
	}
	membership, err := stack.Repository(bundle, MembershipDecider())
	if err != nil {
		_ = bundle.Close()
		return nil, event.WrapTransient(err, "internal", "create membership repository")
	}
	tenant, err := stack.Repository(bundle, TenantDecider())
	if err != nil {
		_ = bundle.Close()
		return nil, event.WrapTransient(err, "internal", "create tenant repository")
	}
	bot, err := stack.Repository(bundle, BotDecider())
	if err != nil {
		_ = bundle.Close()
		return nil, event.WrapTransient(err, "internal", "create bot repository")
	}
	return &aggregateRepositories{
		User:       user,
		Membership: membership,
		Tenant:     tenant,
		Bot:        bot,
	}, nil
}

// buildDeciderRepositories creates the four aggregate repositories from a raw
// store + bus pair. closeOnErr is invoked if any repo creation fails so the
// caller can release partially-built infrastructure (the bus, for example).
// Used by NewEventSourcedSetup, which composes its own store + bus rather
// than receiving a stack Bundle.
func buildDeciderRepositories(
	store event.Store, bus event.Publisher, closeOnErr func(),
) (*aggregateRepositories, error) {
	user, err := decider.NewRepository(store, bus, UserDecider())
	if err != nil {
		closeOnErr()
		return nil, event.NewTransient("internal", "create decider repository").WithCause(err)
	}
	membership, err := decider.NewRepository(store, bus, MembershipDecider())
	if err != nil {
		closeOnErr()
		return nil, event.NewTransient("internal", "create membership decider repository").WithCause(err)
	}
	tenant, err := decider.NewRepository(store, bus, TenantDecider())
	if err != nil {
		closeOnErr()
		return nil, event.NewTransient("internal", "create tenant decider repository").WithCause(err)
	}
	bot, err := decider.NewRepository(store, bus, BotDecider())
	if err != nil {
		closeOnErr()
		return nil, event.NewTransient("internal", "create bot decider repository").WithCause(err)
	}
	return &aggregateRepositories{
		User:       user,
		Membership: membership,
		Tenant:     tenant,
		Bot:        bot,
	}, nil
}
