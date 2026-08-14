package health

import (
	"context"
	"fmt"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	errorfamily "github.com/larsartmann/go-error-family"
	gohealth "github.com/larsartmann/go-health"
	"github.com/samber/do/v2"
)

// ProjectionStatusProvider reports live projection health. It is an alias of
// cqrshtmx.ProjectionStatusProvider, so *usermgmt.Service and
// *usermgmt.EventSourcedSetup satisfy it directly.
type ProjectionStatusProvider = cqrshtmx.ProjectionStatusProvider

// Projection worker states (mirror projectionhost.WorkerState statuses;
// see cqrshtmx.ProjectionReadinessCheck for the readiness semantics).
const (
	statusLive    = "live"
	statusStopped = "stopped"
	statusFailed  = "failed"
)

// NewProbe builds a go-health [gohealth.Probe] with one named check per
// projection of the provider. Check names are the projection names
// ("user-read-model", "casbin-projection", ...), so
// [gohealth.WithCriticalServices] can reference them directly.
//
// opts are passed through to go-health (WithVersion, WithRefreshInterval,
// WithCriticalServices, ...); the bridge adds its own recorder on top.
//
// The returned probe answers liveness, readiness, and startup from
// [gohealth.DefaultRoutes] after RegisterRoutes.
func NewProbe(provider ProjectionStatusProvider, opts ...gohealth.Option) (*gohealth.Probe, error) {
	if provider == nil {
		return nil, errorfamily.NewRejection("cqrshtmx.health.no_provider",
			"health: provider must not be nil (pass *usermgmt.Service or *usermgmt.EventSourcedSetup)")
	}

	recorder := Recorder(provider)
	all := append([]gohealth.Option{gohealth.WithHealthRecorder(recorder)}, opts...)

	return gohealth.New(do.New(), all...), nil
}

// Recorder returns a go-health recorder that reports one named check per
// projection, merged with the injector's own service checks. Use it when
// composing a [gohealth.Probe] yourself (e.g. with a populated samber/do
// injector) so application services and projections are checked in one batch:
//
//	probe := gohealth.New(injector, gohealth.WithHealthRecorder(health.Recorder(svc)))
func Recorder(provider ProjectionStatusProvider) gohealth.HealthRecorder {
	return projectionRecorder{provider: provider}
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
		for name, err := range injector.HealthCheckWithContext(ctx) {
			results[name] = err
		}
	}

	for _, entry := range r.provider.ProjectionStatuses() {
		results[entry.Name] = checkError(entry)
	}

	return results
}

// checkError maps a projection worker state to a health result, mirroring
// cqrshtmx.ProjectionReadinessCheck semantics: live/stopped are healthy,
// drain states are transient (still catching up), failed is an
// infrastructure error carrying the last error message.
func checkError(entry cqrshtmx.ProjectionStatusEntry) error {
	switch entry.Status {
	case statusLive, statusStopped:
		return nil
	case statusFailed:
		return errorfamily.NewInfrastructure("cqrshtmx.health.projection_failed",
			fmt.Sprintf("projection %q failed: %s", entry.Name, entry.LastError))
	default:
		return errorfamily.NewTransient("cqrshtmx.health.projection_draining",
			fmt.Sprintf("projection %q is %q (catching up)", entry.Name, entry.Status))
	}
}
