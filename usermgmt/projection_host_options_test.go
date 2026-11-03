package usermgmt

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
)

// R8: ServiceConfig.ProjectionHostOptions threads consumer projection-host
// options through the shared factory, appended after the curated defaults
// (last-wins). The nil default must keep today's chain byte-identical.

func TestProjectionHostOptions_ThreadsThroughFactory(t *testing.T) {
	t.Parallel()

	// Options applied through the seam: build succeeds and projections
	// register/start — an option the host rejected would fail NewService.
	svc := newTestServiceWithConfig(t, ServiceConfig{
		ProjectionHostOptions: []projectionhost.HostOption{
			projectionhost.WithCheckpointEvery(500),
			projectionhost.WithBatchSize(512),
		},
	})
	defer svc.Close() //nolint:errcheck // test cleanup

	userID := GenerateUserID()
	aggID, err := aggIDFromUser(userID)
	if err != nil {
		t.Fatalf("aggIDFromUser: %v", err)
	}
	if err := svc.dispatcher.Dispatch(
		t.Context(),
		NewRegisterUserCmd(aggID, "host-opts@example.com", "HostOpts", nil),
	); err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if _, ok := svc.readModel.FindByEmail("host-opts@example.com"); !ok {
		t.Error("read model missed the event under consumer host options")
	}
}

func TestProjectionHostOptions_NilKeepsDefaultChain(t *testing.T) {
	t.Parallel()

	svc := newTestServiceWithConfig(t, ServiceConfig{})
	defer svc.Close() //nolint:errcheck // test cleanup

	userID := GenerateUserID()
	aggID, err := aggIDFromUser(userID)
	if err != nil {
		t.Fatalf("aggIDFromUser: %v", err)
	}
	if err := svc.dispatcher.Dispatch(
		t.Context(),
		NewRegisterUserCmd(aggID, "default-chain@example.com", "Default", nil),
	); err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if _, ok := svc.readModel.FindByEmail("default-chain@example.com"); !ok {
		t.Error("nil ProjectionHostOptions changed the default projection chain")
	}
}
