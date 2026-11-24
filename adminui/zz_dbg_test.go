package adminui

import (
	"context"
	"testing"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
)

func TestDbgAudit(t *testing.T) {
	panel := newManifestPanel(t, Config{})
	svc := panel.config.Service
	tenant := mustCreateTenant(t, svc, "dbg")
	u := mustRegister(t, svc, "o1", "o1@dbg.dev")
	mustAddMember(t, svc, u.User.ID, tenant.ID, identitymodel.RoleOwner)
	user := mustUser(t, "admin@example.com")
	ctx := cqrshtmx.WithActorID(context.Background(), identitymodel.ActorIDFromUser(user.ID))
	if err := svc.DeleteTenant(ctx, tenant.ID, "x"); err != nil {
		t.Fatal(err)
	}
	al := svc.AuditLog()
	t.Logf("count=%d", al.Count())
	stream, _ := id.ParseStreamID(tenant.ID.Get())
	for _, e := range al.EntriesFor(stream) {
		t.Logf("entry type=%v action=%s", e.EventType, e.Action)
	}
	for _, e := range al.Entries() {
		t.Logf("all: %v %s %s", e.EventType, e.Action, e.AggregateID.String())
	}
}
