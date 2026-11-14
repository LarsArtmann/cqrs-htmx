package adminui

import (
	"context"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// addedByIndex resolves, per member row, WHOSE dispatch created the
// membership. A membership lives on its own stream (one per actor+tenant
// pair); the MemberAdded event is the FIRST event on it, and its metadata
// ActorID records the initiator — the same actor chain the audit trail
// rides (go-cqrs-lite's ActorEnricher, wired by usermgmt's NewService with
// zero consumer code). The cost is one bounded stream Load per visible row:
// a page of members pays page-sized reads, never a tenant-wide scan.
func (h *Handler) addedByIndex(
	ctx context.Context, tenantID identitymodel.TenantID, members []memberRow,
) map[string]string {
	addedBy := make(map[string]string, len(members))
	for _, m := range members {
		stream := identitymodel.DeriveMembershipID(m.Actor, tenantID)
		events, err := h.config.Service.Journal().Load(
			ctx, id.NewStreamRef(identitymodel.AggregateTypeMembership, stream),
		)
		if err != nil || len(events) == 0 {
			continue // unknown stream or store hiccup: the cell stays honest
		}
		actor := events[0].Metadata().ActorID
		if actor.IsZero() {
			continue // pre-actor-chain event: no attribution recorded
		}
		addedBy[m.Actor.PrefixedString()] = h.actorDisplayName(actor)
	}
	return addedBy
}

// actorDisplayName renders the attribution as a person (email) when the
// directory resolves the actor, falling back to the prefixed actor id for
// bot/system initiators — the same rule the member rows follow.
func (h *Handler) actorDisplayName(actor identitymodel.ActorID) string {
	if uid, ok := identitymodel.ActorIDAsUserID(actor); ok {
		if u, found := h.lookupUser(uid); found && u.Email != "" {
			return u.Email
		}
	}
	return actor.PrefixedString()
}
