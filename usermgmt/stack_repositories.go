package usermgmt

import (
	"github.com/larsartmann/go-cqrs-lite/decider/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/stack/v4"
	errorfamily "github.com/larsartmann/go-error-family"
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
// error is returned so the caller never sees a half-built bundle. snap
// optionally enables aggregate snapshotting (see SnapshotConfig); a zero-value
// snap leaves repositories in full-replay mode.
func buildStackRepositories(bundle *stack.Bundle, snap SnapshotConfig) (*aggregateRepositories, error) {
	user, err := stack.Repository(bundle, UserDecider(), snapshotOptions[UserState](snap)...)
	if err != nil {
		_ = bundle.Close()
		return nil, errorfamily.WrapTransient(err, "internal", "create user repository")
	}
	membership, err := stack.Repository(bundle, MembershipDecider(), snapshotOptions[MembershipState](snap)...)
	if err != nil {
		_ = bundle.Close()
		return nil, errorfamily.WrapTransient(err, "internal", "create membership repository")
	}
	tenant, err := stack.Repository(bundle, TenantDecider(), snapshotOptions[TenantState](snap)...)
	if err != nil {
		_ = bundle.Close()
		return nil, errorfamily.WrapTransient(err, "internal", "create tenant repository")
	}
	bot, err := stack.Repository(bundle, BotDecider(), snapshotOptions[BotState](snap)...)
	if err != nil {
		_ = bundle.Close()
		return nil, errorfamily.WrapTransient(err, "internal", "create bot repository")
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
// than receiving a stack Bundle. snap optionally enables aggregate
// snapshotting (see SnapshotConfig); a zero-value snap leaves repositories in
// full-replay mode.
func buildDeciderRepositories(
	store event.Store, bus event.Publisher, closeOnErr func(), snap SnapshotConfig,
) (*aggregateRepositories, error) {
	user, err := decider.NewRepository( //cqrs-lint:ignore(A017) snapshot is opt-in via SnapshotConfig; zero-value = full-replay mode
		store,
		bus,
		UserDecider(),
		snapshotOptions[UserState](
			snap,
		)...)
	if err != nil {
		closeOnErr()
		return nil, errorfamily.NewTransient("internal", "create decider repository").WithCause(err)
	}
	membership, err := decider.NewRepository( //cqrs-lint:ignore(A017) snapshot is opt-in via SnapshotConfig
		store,
		bus,
		MembershipDecider(),
		snapshotOptions[MembershipState](snap)...)
	if err != nil {
		closeOnErr()
		return nil, errorfamily.NewTransient("internal", "create membership decider repository").WithCause(err)
	}
	tenant, err := decider.NewRepository( //cqrs-lint:ignore(A017) snapshot is opt-in via SnapshotConfig
		store,
		bus,
		TenantDecider(),
		snapshotOptions[TenantState](snap)...)
	if err != nil {
		closeOnErr()
		return nil, errorfamily.NewTransient("internal", "create tenant decider repository").WithCause(err)
	}
	bot, err := decider.NewRepository( //cqrs-lint:ignore(A017) snapshot is opt-in via SnapshotConfig
		store,
		bus,
		BotDecider(),
		snapshotOptions[BotState](snap)...)
	if err != nil {
		closeOnErr()
		return nil, errorfamily.NewTransient("internal", "create bot decider repository").WithCause(err)
	}
	return &aggregateRepositories{
		User:       user,
		Membership: membership,
		Tenant:     tenant,
		Bot:        bot,
	}, nil
}
