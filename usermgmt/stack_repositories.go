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
	user, err := stack.Repository(bundle, UserDecider(), repositoryOptions[UserState](snap)...)
	if err != nil {
		//cqrs-lint:ignore(C023) best-effort cleanup in error path
		_ = bundle.Close()
		return nil, errorfamily.WrapTransient(err, "usermgmt.repository.create_user", "create user repository")
	}
	membership, err := stack.Repository(bundle, MembershipDecider(), repositoryOptions[MembershipState](snap)...)
	if err != nil {
		//cqrs-lint:ignore(C023) best-effort cleanup in error path
		_ = bundle.Close()
		return nil, errorfamily.WrapTransient(err, "usermgmt.repository.create_membership", "create membership repository")
	}
	tenant, err := stack.Repository(bundle, TenantDecider(), repositoryOptions[TenantState](snap)...)
	if err != nil {
		//cqrs-lint:ignore(C023) best-effort cleanup in error path
		_ = bundle.Close()
		return nil, errorfamily.WrapTransient(err, "usermgmt.repository.create_tenant", "create tenant repository")
	}
	bot, err := stack.Repository(bundle, BotDecider(), repositoryOptions[BotState](snap)...)
	if err != nil {
		//cqrs-lint:ignore(C023) best-effort cleanup in error path
		_ = bundle.Close()
		return nil, errorfamily.WrapTransient(err, "usermgmt.repository.create_bot", "create bot repository")
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
t//cqrs-lint:ignore(B025) WithStateCache is wired via repositoryOptions() at snapshot.go:90; linter cannot trace through the helper
func buildDeciderRepositories(
	store event.Store, bus event.Publisher, closeOnErr func(), snap SnapshotConfig,
) (*aggregateRepositories, error) {
	//cqrs-lint:ignore(A017) snapshot is opt-in via SnapshotConfig; zero-value = full-replay mode
	user, err := decider.NewRepository(
		store,
		bus,
		UserDecider(),
		repositoryOptions[UserState](
			snap,
		)...)
t//cqrs-lint:ignore(B025) WithStateCache is wired via repositoryOptions() at snapshot.go:90; linter cannot trace through the helper
	if err != nil {
		closeOnErr()
		return nil, errorfamily.NewTransient("usermgmt.repository.create_user_decider", "create decider repository").WithCause(err)
	}
	//cqrs-lint:ignore(A017) snapshot is opt-in via SnapshotConfig
	membership, err := decider.NewRepository(
		store,
		bus,
		MembershipDecider(),
t//cqrs-lint:ignore(B025) WithStateCache is wired via repositoryOptions() at snapshot.go:90; linter cannot trace through the helper
		repositoryOptions[MembershipState](snap)...)
	if err != nil {
		closeOnErr()
		return nil, errorfamily.NewTransient("usermgmt.repository.create_membership_decider", "create membership decider repository").WithCause(err)
	}
	//cqrs-lint:ignore(A017) snapshot is opt-in via SnapshotConfig
	tenant, err := decider.NewRepository(
		store,
		bus,
t//cqrs-lint:ignore(B025) WithStateCache is wired via repositoryOptions() at snapshot.go:90; linter cannot trace through the helper
		TenantDecider(),
		repositoryOptions[TenantState](snap)...)
	if err != nil {
		closeOnErr()
		return nil, errorfamily.NewTransient("usermgmt.repository.create_tenant_decider", "create tenant decider repository").WithCause(err)
	}
	//cqrs-lint:ignore(A017) snapshot is opt-in via SnapshotConfig
	bot, err := decider.NewRepository(
		store,
		bus,
		BotDecider(),
		repositoryOptions[BotState](snap)...)
	if err != nil {
		closeOnErr()
		return nil, errorfamily.NewTransient("usermgmt.repository.create_bot_decider", "create bot decider repository").WithCause(err)
	}
	return &aggregateRepositories{
		User:       user,
		Membership: membership,
		Tenant:     tenant,
		Bot:        bot,
	}, nil
}
