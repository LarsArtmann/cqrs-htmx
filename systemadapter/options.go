package systemadapter

import (
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
)

// DomainOption customizes the [system.DomainConfig] returned by
// [DomainConfig]. It is the same underlying option type as
// [ProjectionLayerOption], so the checkpoint/dead-letter vocabulary works at
// both call sites.
type DomainOption = func(*adapterOptions)

// adapterOptions carries the projection-infrastructure knobs shared by the
// declarative path ([DomainConfig]) and the legacy [NewProjectionLayer].
type adapterOptions struct {
	checkpointStore event.CheckpointStore
	deadLetterStore projectionhost.DeadLetterStore
	hostOptions     []projectionhost.HostOption
}

// WithCheckpointStore sets the persistent checkpoint store for the projection
// host. Without it, an in-memory store is used and every projection replays
// the full journal on start — acceptable for demos and tests, not for
// production.
//
// On the declarative path this becomes system.DomainConfig.CheckpointStore;
// when left nil there, system.New applies its own default (checkpoints as
// entries of a system_checkpoints Map collection on the deployment-declared
// engine when one carries the Map ADT, ADR-0142).
//
// Durable checkpoints skip replay of already-processed events after a
// restart — pair them with durable read models (or rebuild on demand),
// because a fresh in-memory read model with an up-to-date checkpoint
// processes nothing new and stays empty.
func WithCheckpointStore(store event.CheckpointStore) DomainOption {
	return func(o *adapterOptions) {
		o.checkpointStore = store
	}
}

// WithHostOptions appends projectionhost.HostOptions (batch size, logger,
// restart policy, metrics, ...) applied when the declarative system creates
// its projection host. Options are appended AFTER systemadapter's curated
// defaults, so passing your own WithBatchSize/WithBackoff/... overrides the
// corresponding default; the system's own subscriber wiring always wins over
// both. Legacy NewProjectionLayer ignores this option — it builds its own
// host with fixed tuning.
func WithHostOptions(opts ...projectionhost.HostOption) DomainOption {
	return func(o *adapterOptions) {
		o.hostOptions = append(o.hostOptions, opts...)
	}
}
