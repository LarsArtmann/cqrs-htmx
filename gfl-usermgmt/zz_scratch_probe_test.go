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
	members, err := systemadapter.FindMembershipsByTenant(t.Context(), sys, tenantStream.String())
	t.Logf("probe: by tenant %q -> %d members, err=%v", tenantStream.String(), len(members), err)
	for _, m := range members {
		t.Logf("probe: member %+v", m)
	}
	byActor, err := systemadapter.FindMembershipsByActor(t.Context(), sys, memberActor.String())
	t.Logf("probe: by actor %q -> %d, err=%v", memberActor.String(), len(byActor), err)
	for _, m := range byActor {
		t.Logf("probe: actor-member %+v", m)
	}
}
