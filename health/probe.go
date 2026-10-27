package health

import (
	"context"
	"fmt"
	"maps"
	"time"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	errorfamily "github.com/larsartmann/go-error-family"
	gohealth "github.com/larsartmann/go-health"
	"github.com/samber/do/v2"
)

// ProjectionStatusProvider reports live projection health. It is an alias of
// cqrshtmx.ProjectionStatusProvider, so *usermgmt.Service and
// *usermgmt.EventSourcedSetup satisfy it directly.
type ProjectionStatusProvider = cqrshtmx.ProjectionStatusProvider

// Projection worker states come from the root module
// (cqrshtmx.ProjectionStatusLive/Stopped/Failed, published in root v4.14.0);
// see cqrshtmx.ProjectionReadinessCheck for the readiness semantics.

// NewProbe builds a go-health [gohealth.Probe] with one named check per
// projection of the provider. Check names are the projection names
// ("user-read-model", "casbin-projection", ...), so
// [gohealth.WithCriticalServices] can reference them directly.
//
// The probe needs no injector: batches come straight from the provider via
// [gohealth.NewWithDetailedCheck], and every projection check carries the
// shared ProjectionStatuses() read time as its duration, so dashboards render
// duration_ns out of the box.
//
// opts are passed through to go-health (WithVersion, WithRefreshInterval,
// WithCriticalServices, ...). [gohealth.WithHealthRecorder] has no effect
// here — the provider owns the batch. To compose additional recorders (an
// audit-log plugin, injector service checks), build the probe yourself with
// [RecorderChain]:
//
//	probe := gohealth.New(injector, gohealth.WithHealthRecorder(
//	    health.RecorderChain(health.Recorder(svc), auditPlugin),
//	))
//
// The returned probe answers liveness, readiness, and startup from
// [gohealth.DefaultRoutes] after RegisterRoutes.
func NewProbe(provider ProjectionStatusProvider, opts ...gohealth.Option) (*gohealth.Probe, error) {
	if provider == nil {
		return nil, errorfamily.NewRejection("cqrshtmx.health.no_provider",
			"health: provider must not be nil (pass *usermgmt.Service or *usermgmt.EventSourcedSetup)")
	}

	recorder := projectionRecorder{provider: provider}

	return gohealth.NewWithDetailedCheck(
		func(ctx context.Context) map[string]gohealth.CheckDetail {
			return recorder.RecordDetailedHealthCheckWithContext(ctx, nil)
		},
		opts...,
	), nil
}

// Recorder returns a go-health recorder that reports one named check per
// projection, merged with the injector's own service checks. Use it when
// composing a [gohealth.Probe] yourself (e.g. with a populated samber/do
// injector) so application services and projections are checked in one batch:
//
//	probe := gohealth.New(injector, gohealth.WithHealthRecorder(health.Recorder(svc)))
//
// The returned recorder also implements [gohealth.DetailedHealthRecorder]:
// projection checks carry the shared ProjectionStatuses() read time as their
// duration, so duration_ns reaches the wire on this path too.
func Recorder(provider ProjectionStatusProvider) gohealth.HealthRecorder {
	if provider == nil {
		// Mirror NewProbe's fail-fast: without the guard the nil dereference
		// surfaces only when the first health batch runs, far from the cause.
		panic("cqrs-htmx/health: Recorder requires a non-nil provider " +
			"(pass *usermgmt.Service or *usermgmt.EventSourcedSetup)")
	}

	return projectionRecorder{provider: provider}
}

// RecorderChain composes several [gohealth.HealthRecorder]s into one, in
// order: every batch runs each recorder and merges the results. Later
// recorders win name collisions, so put the surface that should own a shared
// name last. Nil recorders are skipped; a chain of only nils reports no
// checks.
//
// Use it to make "one probe" composition literal, e.g. projection health plus
// the samber-do-auditlog plugin (which satisfies HealthRecorder implicitly):
//
//	probe := gohealth.New(injector, gohealth.WithHealthRecorder(
//	    health.RecorderChain(health.Recorder(svc), auditPlugin),
//	))
//
// Deprecated: use [gohealth.ChainRecorders] directly. It is the same merge —
// this function now forwards to it — but the upstream chain always satisfies
// [gohealth.DetailedHealthRecorder], so per-check durations from detailed
// members survive the merge and reach the wire. This wrapper used to drop
// them: it implemented only the plain interface, so a probe holding it never
// took the detailed path and every projection duration was lost.
func RecorderChain(recorders ...gohealth.HealthRecorder) gohealth.HealthRecorder {
	return gohealth.ChainRecorders(recorders...)
}

type projectionRecorder struct {
	provider ProjectionStatusProvider
}

// RecordHealthCheckWithContext merges the injector's service checks with the
// provider's projection checks. Projection names take precedence on collision.
func (r projectionRecorder) RecordHealthCheckWithContext(
	ctx context.Context,
	injector do.Injector,
) map[string]error {
	results := make(map[string]error)

	if injector != nil {
		maps.Copy(results, injector.HealthCheckWithContext(ctx))
	}

	for _, entry := range r.provider.ProjectionStatuses() {
		results[entry.Name] = checkError(entry)
	}

	return results
}

// RecordDetailedHealthCheckWithContext is the metadata-rich variant used by
// probes that support [gohealth.DetailedHealthRecorder]: every projection
// check carries the shared ProjectionStatuses() read time as its
// [gohealth.CheckDetail.Duration]. ProjectionStatuses() is one batch read, so
// the batch time is the honest per-check measurement. Injector service checks
// merge in without durations; samber/do reports their outcomes only.
func (r projectionRecorder) RecordDetailedHealthCheckWithContext(
	ctx context.Context,
	injector do.Injector,
) map[string]gohealth.CheckDetail {
	start := time.Now()
	entries := r.provider.ProjectionStatuses()
	readDuration := time.Since(start)

	results := make(map[string]gohealth.CheckDetail, len(entries))

	if injector != nil {
		for name, err := range injector.HealthCheckWithContext(ctx) {
			results[name] = gohealth.CheckDetail{Err: err, Duration: 0}
		}
	}

	for _, entry := range entries {
		results[entry.Name] = gohealth.CheckDetail{
			Err:      checkError(entry),
			Duration: readDuration,
		}
	}

	return results
}

// checkError maps a projection worker state to a health result, mirroring
// cqrshtmx.ProjectionReadinessCheck semantics: live/stopped are healthy,
// drain states are transient (still catching up), failed is an
// infrastructure error carrying the last error message.
func checkError(entry cqrshtmx.ProjectionStatusEntry) error {
	switch entry.Status {
	case cqrshtmx.ProjectionStatusLive, cqrshtmx.ProjectionStatusStopped:
		return nil

	case cqrshtmx.ProjectionStatusFailed:
		return errorfamily.NewInfrastructure("cqrshtmx.health.projection_failed",
			fmt.Sprintf("projection %q failed: %s", entry.Name, entry.LastError))
	case cqrshtmx.ProjectionStatusIdle, cqrshtmx.ProjectionStatusRunning, cqrshtmx.ProjectionStatusBackoff:
		return errorfamily.NewTransient("cqrshtmx.health.projection_draining",
			fmt.Sprintf("projection %q is %q (catching up)", entry.Name, entry.Status))
	default:
		// Unknown statuses stay transient (not-ready, retry) but are worded
		// honestly — calling an unrecognized state "catching up" would be a
		// guess presented as a diagnosis.
		return errorfamily.NewTransient("cqrshtmx.health.projection_unknown_status",
			fmt.Sprintf("projection %q reported unrecognized status %q (treated as not-ready)",
				entry.Name, entry.Status))
	}
}
