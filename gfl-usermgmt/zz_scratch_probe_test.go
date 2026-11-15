package gflusermgmt

import (
	"testing"

	systemadapter "github.com/larsartmann/cqrs-htmx/systemadapter/v4"
	"github.com/larsartmann/go-cqrs-lite/systemscenario/v4"
)

func TestScratchProbe(t *testing.T) {
	sc := systemscenario.System(t, t.Context(), systemadapter.DomainConfig(), systemadapter.RecommendedMemoryDeployment())
	sc.Given(CreatedTenant(sc, "acme")...).
		Events(MemberOfTenant(sc, "admin")...)

	sys := sc.System()
	for _, w := range sys.ProjectionHost().Status() {
		t.Logf("probe: worker name=%q checkpoint=%q processed=%d status=%s", w.Name, w.Checkpoint, w.Processed, w.Status)
	}
	members, err := systemadapter.FindMembershipsByTenant(t.Context(), sys, tenantStream.String())
	t.Logf("probe: by tenant %q -> %d members, err=%v", tenantStream.String(), len(members), err)
	for _, m := range members {
		t.Logf("probe: member %+v", m)
	}
	_, _ = sys, members
	sc.When(DeleteTenant("shutdown")).Await().Then("TenantDeleted")
}
