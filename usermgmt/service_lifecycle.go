package usermgmt

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

func (s *Service) wireLockoutEviction() {
	if s.lockout == nil {
		return
	}
	evictor, ok := s.lockout.(interface{ EvictStale() int })
	if !ok {
		return
	}
	s.stopLockoutEviction = startPeriodicEviction(evictor.EvictStale, lockoutEvictionInterval)
}

// Authz returns the underlying authorization engine for direct policy queries.
func (s *Service) Authz() *Authz { return s.authz }

// Stop gracefully shuts down background resources associated with the Service,
// such as the WebAuthn session eviction goroutine. It is safe to call multiple
// times and is a no-op when no background resources are running.
//
// Stop does NOT close the event bus or event store. Use [Service.Close] for
// full lifecycle shutdown.
func (s *Service) Stop() {
	s.stopEvictions()
}

// stopEvictions stops all background eviction goroutines. Idempotent.
func (s *Service) stopEvictions() {
	for _, stop := range []*func(){
		&s.stopWebAuthnEviction,
		&s.stopVerificationEviction,
		&s.stopPendingTOTPEviction,
		&s.stopOAuth2Eviction,
		&s.stopLockoutEviction,
	} {
		if *stop != nil {
			(*stop)()
			*stop = nil
		}
	}
}

// Close performs a full lifecycle shutdown: stops all background eviction
// goroutines, then closes the event bus and event store (if they implement
// io.Closer). It is safe to call multiple times.
//
// For context-bounded shutdown prefer [Service.GracefulClose].
func (s *Service) Close() error {
	s.stopEvictions()
	return s.closeInfra()
}

// GracefulClose is identical to [Service.Close] but bounded by ctx.
// If the context expires before resources are closed, the context error is
// returned alongside any resource errors.
func (s *Service) GracefulClose(ctx context.Context) error {
	s.stopEvictions()
	if err := s.closeInfra(); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return errorfamily.WrapTransient(
			err,
			"usermgmt.service.graceful_close",
			"context cancelled during graceful close",
		)
	}
	return nil
}

// closeInfra closes the bus and store if they implement io.Closer.
// Returns the first error encountered.
func (s *Service) closeInfra() error {
	if s.dispatcher != nil {
		if err := s.dispatcher.Close(); err != nil {
			return errorfamily.WrapTransient(err, "usermgmt.service.close_dispatcher", "close command dispatcher")
		}
	}
	stopProjectionHost(s.projectionHost, "usermgmt.Service", "usermgmt.service.stop_projections")
	if c, ok := s.bus.(interface{ Close() error }); ok {
		if err := c.Close(); err != nil {
			return errorfamily.WrapTransient(err, "usermgmt.service.close_bus", "close event bus")
		}
	}
	if c, ok := s.store.(interface{ Close() error }); ok {
		if err := c.Close(); err != nil {
			return errorfamily.WrapTransient(err, "usermgmt.service.close_store", "close event store")
		}
	}
	return nil
}

// ReadModel returns the user read model for direct queries.
func (s *Service) ReadModel() *UserReadModel { return s.readModel }

// AuditLog returns the configured audit log, or nil if not configured.
func (s *Service) AuditLog() *AuditLog { return s.auditLog }

// ProjectionHost returns the internal projection host that manages read-model
// and Casbin projections. Returns nil if the service was constructed without
// event-sourced setup.
//
// Consumers building observability dashboards (e.g. dashboardui) can use this
// to wire projection health panels without constructing a separate host.
func (s *Service) ProjectionHost() *projectionhost.Host { return s.projectionHost }

// Journal returns the event store backing the service's aggregates.
//
// Composition seam: consumers wiring their own projections, read models, SSE
// endpoints, or dashboards on top of an already-built Service need the journal
// it persists to — without this accessor they must keep the ServiceConfig
// value around and hope it stays in sync. The Service retains ownership (see
// [Service.Close] for lifecycle).
func (s *Service) Journal() event.Store { return s.store }

// EventBus returns the bus the service publishes committed events to.
//
// Composition seam: subscribing to live domain events (e.g. bridging them to
// an SSE broadcaster) requires the exact bus instance the service publishes
// to. The Service retains ownership (see [Service.Close] for lifecycle).
func (s *Service) EventBus() event.Bus { return s.bus }

func (s *Service) emailFromEvent(evt event.Event) string {
	user, ok := s.readModel.FindByID(evt.StreamID())
	if !ok {
		return ""
	}
	return user.Email
}
