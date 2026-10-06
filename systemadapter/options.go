package systemadapter

import (
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
)

// DomainOption customizes the [system.DomainConfig] returned by
// [DomainConfig].
type DomainOption func(*domainOptions)

type domainOptions struct {
	checkpointStore event.CheckpointStore
	hostOptions     []projectionhost.HostOption
}

// WithCheckpointStore sets the persistent checkpoint store for the system's
// projection host — the declarative twin of the legacy NewProjectionLayer
// option of the same name. Without it, system.New applies its own default:
// checkpoints persist as entries of a system_checkpoints Map collection on
// the deployment-declared engine when one carries the Map ADT (ADR-0142);
// engines without it fall back to an in-memory store (checkpoints lost on
// restart).
//
// Durable checkpoints skip replay of already-processed events after a
// restart — pair them with durable read models (or rebuild on demand),
// because a fresh in-memory read model with an up-to-date checkpoint
// processes nothing new and stays empty.
func WithCheckpointStore(store event.CheckpointStore) DomainOption {
	return func(o *domainOptions) {
		o.checkpointStore = store
	}
}

// WithHostOptions appends projectionhost.HostOptions (batch size, dead-letter
// store, logger, restart policy, metrics, ...) applied when the system
// creates its projection host. Options are appended AFTER systemadapter's
// curated defaults, so passing your own WithDeadLetterStore/WithBatchSize/…
// overrides the corresponding default; the system's own subscriber wiring
// always wins over both.
func WithHostOptions(opts ...projectionhost.HostOption) DomainOption {
	return func(o *domainOptions) {
		o.hostOptions = append(o.hostOptions, opts...)
	}
}
