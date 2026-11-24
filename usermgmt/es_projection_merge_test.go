package usermgmt

import (
	"slices"
	"testing"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// The projections around the user read model must follow a merge too: the
// casbin projection retires the merged identity's grants and clears a merged
// tenant's domain, the bot read model moves transferred bots to their new
// owner, and the audit log labels both merge events.

// groupRowsFor returns the group-policy rows whose subject slot is subject.
func groupRowsFor(t *testing.T, authz *Authz, subject string) [][]string {
	t.Helper()
	rows, err := authz.GroupPolicies()
	if err != nil {
		t.Fatalf("GroupPolicies: %v", err)
	}
	var out [][]string
	for _, row := range rows {
		if len(row) > 0 && row[0] == subject {
			out = append(out, row)
		}
	}
	return out
}

// groupRowsInDomain returns the group-policy rows whose domain slot is domain.
func groupRowsInDomain(t *testing.T, authz *Authz, domain string) [][]string {
	t.Helper()
	rows, err := authz.GroupPolicies()
	if err != nil {
		t.Fatalf("GroupPolicies: %v", err)
	}
	var out [][]string
	for _, row := range rows {
		if len(row) > 2 && row[2] == domain {
			out = append(out, row)
		}
	}
	return out
}

func TestCasbinProjection_UserMergedIntoRemovesEveryGrant(t *testing.T) {
	authz, err := NewAuthz()
	if err != nil {
		t.Fatalf("NewAuthz: %v", err)
	}
	proj, err := NewCasbinProjection(authz)
	if err != nil {
		t.Fatalf("NewCasbinProjection: %v", err)
	}

	loser := newUserStream(t)
	survivor := newUserStream(t)
	member := newUserStream(t)
	tenant := NewTenantID("01JXCASBINMERGE00000000000A")
	actor := ActorIDFromUser(member.userID())

	// Grants the projection itself derives: the loser's own registration
	// roles (subject = raw stream ID) and a tenant membership (subject =
	// actor spelling). The merge retires only the raw-subject rows; the
	// membership rows are the service's RemoveMember commands' business.
	registered := makeUserEvent(t, loser.aggID, eventUserRegistered, 1, UserRegisteredPayload{
		SchemaVersion: currentSchemaVersion,
		Email:         "loser@example.com",
		Roles:         []Role{RoleUser, RoleViewer},
	})
	memberAdded := makeEventFor(t, eventMemberAdded, 1, deriveMembershipID(actor, tenant),
		aggregateTypeMembership, MemberAddedPayload{
			SchemaVersion: currentSchemaVersion,
			ActorKind:     actor.Kind().String(),
			ActorID:       actor.String(),
			TenantID:      tenant.Get(),
			Roles:         []Role{RoleUser},
		})
	if err := proj.Handle(t.Context(), registered); err != nil {
		t.Fatalf("Handle UserRegistered: %v", err)
	}
	if err := proj.Handle(t.Context(), memberAdded); err != nil {
		t.Fatalf("Handle MemberAdded: %v", err)
	}
	if got := len(groupRowsFor(t, authz, loser.aggID.Get())); got != 2 {
		t.Fatalf("loser raw-subject grants before merge = %d, want 2", got)
	}

	if err := proj.Handle(t.Context(), loser.mergedInto(survivor.aggID)); err != nil {
		t.Fatalf("Handle UserMergedInto: %v", err)
	}

	if got := groupRowsFor(t, authz, loser.aggID.Get()); len(got) != 0 {
		t.Errorf("loser raw-subject grants after merge = %v, want none", got)
	}
	if got := len(groupRowsFor(t, authz, actor.String())); got != 1 {
		t.Errorf("member actor grants after merge = %d, want 1 (removed by RemoveMember, not by the retirement)", got)
	}

	// Re-delivery stays the no-op the at-least-once drain needs.
	if err := proj.Handle(t.Context(), loser.mergedInto(survivor.aggID)); err != nil {
		t.Fatalf("re-delivered Handle UserMergedInto: %v", err)
	}
	if got := groupRowsFor(t, authz, loser.aggID.Get()); len(got) != 0 {
		t.Errorf("loser grants re-appeared after re-delivery: %v", got)
	}
}

func TestCasbinProjection_TenantMergedIntoClearsDomain(t *testing.T) {
	authz, err := NewAuthz()
	if err != nil {
		t.Fatalf("NewAuthz: %v", err)
	}
	proj, err := NewCasbinProjection(authz)
	if err != nil {
		t.Fatalf("NewCasbinProjection: %v", err)
	}

	loserRaw := "01JXTENANTMERGE000000000000B"
	loserTenant := NewTenantID(loserRaw)
	survivorRaw := "01JXTENANTMERGE0000000000C"
	mergedAwayStream, err := id.ParseStreamID(loserRaw)
	if err != nil {
		t.Fatalf("ParseStreamID: %v", err)
	}
	actor := ActorIDFromUser(NewUserID("01JXCASBINACTOR0000000000D"))

	member := makeEventFor(t, eventMemberAdded, 1, deriveMembershipID(actor, loserTenant),
		aggregateTypeMembership, MemberAddedPayload{
			SchemaVersion: currentSchemaVersion,
			ActorKind:     actor.Kind().String(),
			ActorID:       actor.String(),
			TenantID:      loserTenant.Get(),
			Roles:         []Role{RoleAdmin},
		})
	if err := proj.Handle(t.Context(), member); err != nil {
		t.Fatalf("Handle MemberAdded: %v", err)
	}
	// The domain holds the member's grant plus the mirrored role-hierarchy
	// edges; what matters is that the member has exactly one grant there.
	if got := len(groupRowsFor(t, authz, actor.String())); got != 1 {
		t.Fatalf("member grants before merge = %d, want 1", got)
	}

	mergedInto := makeEventFor(t, eventTenantMergedInto, 1, mergedAwayStream,
		aggregateTypeTenant, identitymodel.TenantMergedIntoPayload{
			SchemaVersion: currentSchemaVersion,
			SurvivorID:    NewTenantID(survivorRaw),
			Reason:        "dedupe",
		})
	if err := proj.Handle(t.Context(), mergedInto); err != nil {
		t.Fatalf("Handle TenantMergedInto: %v", err)
	}

	if got := groupRowsInDomain(t, authz, loserRaw); len(got) != 0 {
		t.Errorf("domain grants after tenant merge = %v, want none", got)
	}
	if got := groupRowsFor(t, authz, actor.String()); len(got) != 0 {
		t.Errorf("member grants survived the tenant merge: %v", got)
	}
}

func TestBotReadModel_TransferMovesTheBot(t *testing.T) {
	m := NewBotReadModel()
	botStream := id.NewStreamID()
	oldOwner := NewUserID("01JXBOTOWNEROLD000000000000E")
	newOwner := NewUserID("01JXBOTOWNERNEW000000000000F")

	registered := makeEventFor(t, eventBotRegistered, 1, botStream, aggregateTypeBot,
		BotRegisteredPayload{
			SchemaVersion: currentSchemaVersion,
			Name:          "merger",
			OwnerID:       oldOwner,
			TokenHash:     []byte("hash-merger"),
			Scopes:        []string{"read"},
		})
	if err := m.Handle(t.Context(), registered); err != nil {
		t.Fatalf("Handle BotRegistered: %v", err)
	}

	transferred := makeEventFor(t, eventBotTransferred, 2, botStream, aggregateTypeBot,
		identitymodel.BotTransferredPayload{
			SchemaVersion: currentSchemaVersion,
			OwnerID:       newOwner,
			Reason:        "merge",
		})
	if err := m.Handle(t.Context(), transferred); err != nil {
		t.Fatalf("Handle BotTransferred: %v", err)
	}

	bot, ok := m.FindByID(botStream)
	if !ok {
		t.Fatal("bot missing after transfer")
	}
	if bot.OwnerID != newOwner {
		t.Errorf("owner after transfer = %s, want %s", bot.OwnerID.Get().String(), newOwner.Get().String())
	}
	if string(bot.TokenHash) != "hash-merger" {
		t.Errorf("token hash changed on transfer: %q", string(bot.TokenHash))
	}
	if got := m.FindByOwner(oldOwner); len(got) != 0 {
		t.Errorf("FindByOwner(old) = %d bots after transfer, want 0", len(got))
	}
	if got := m.FindByOwner(newOwner); len(got) != 1 {
		t.Errorf("FindByOwner(new) = %d bots after transfer, want 1", len(got))
	}
	if _, ok := m.FindByTokenHash([]byte("hash-merger")); !ok {
		t.Error("FindByTokenHash lost the bot after transfer")
	}

	// A transfer for a stream with no row is a no-op, and re-delivery keeps
	// the last owner.
	unknown := makeEventFor(t, eventBotTransferred, 1, id.NewStreamID(), aggregateTypeBot,
		identitymodel.BotTransferredPayload{SchemaVersion: currentSchemaVersion, OwnerID: oldOwner})
	if err := m.Handle(t.Context(), unknown); err != nil {
		t.Fatalf("Handle BotTransferred for unknown bot: %v", err)
	}
	if err := m.Handle(t.Context(), transferred); err != nil {
		t.Fatalf("re-delivered Handle BotTransferred: %v", err)
	}
	if bot, ok := m.FindByID(botStream); !ok || bot.OwnerID != newOwner {
		t.Error("re-delivered transfer moved the bot away from its owner")
	}
}

func TestSQLBotReadModel_TransferSurvivesRestart(t *testing.T) {
	db := newSQLTestDB(t)
	m, err := NewSQLiteBotReadModel(db)
	if err != nil {
		t.Fatalf("NewSQLiteBotReadModel: %v", err)
	}
	botStream := id.NewStreamID()
	oldOwner := NewUserID("01JXBOTOWNEROLD000000000000E")
	newOwner := NewUserID("01JXBOTOWNERNEW000000000000F")

	for _, evt := range []event.Event{
		makeEventFor(t, eventBotRegistered, 1, botStream, aggregateTypeBot, BotRegisteredPayload{
			SchemaVersion: currentSchemaVersion,
			Name:          "restart-merger",
			OwnerID:       oldOwner,
			TokenHash:     []byte("hash-restart"),
		}),
		makeEventFor(t, eventBotTransferred, 2, botStream, aggregateTypeBot, identitymodel.BotTransferredPayload{
			SchemaVersion: currentSchemaVersion,
			OwnerID:       newOwner,
			Reason:        "merge",
		}),
	} {
		if err := m.Handle(t.Context(), evt); err != nil {
			t.Fatalf("Handle %s: %v", evt.Type(), err)
		}
	}

	fresh, err := NewSQLiteBotReadModel(db)
	if err != nil {
		t.Fatalf("NewSQLiteBotReadModel (fresh): %v", err)
	}
	if err := fresh.Hydrate(t.Context()); err != nil {
		t.Fatalf("Hydrate: %v", err)
	}

	bot, ok := fresh.FindByID(botStream)
	if !ok {
		t.Fatal("bot missing after hydrate")
	}
	if bot.OwnerID != newOwner {
		t.Errorf("owner after hydrate = %s, want %s", bot.OwnerID.Get().String(), newOwner.Get().String())
	}
	if got := fresh.FindByOwner(oldOwner); len(got) != 0 {
		t.Errorf("FindByOwner(old) after hydrate = %d bots, want 0", len(got))
	}
	if _, ok := fresh.FindByTokenHash([]byte("hash-restart")); !ok {
		t.Error("FindByTokenHash lost the bot after hydrate")
	}
}

func TestAuditLog_LabelsMergeEvents(t *testing.T) {
	if got := auditActionFor(eventUserMerged); got != AuditActionUserMerged {
		t.Errorf("auditActionFor(UserMerged) = %q, want %q", got, AuditActionUserMerged)
	}
	if got := auditActionFor(eventUserMergedInto); got != AuditActionUserMergedInto {
		t.Errorf("auditActionFor(UserMergedInto) = %q, want %q", got, AuditActionUserMergedInto)
	}
	if !slices.Contains(allAuditedEventTypes, eventUserMerged) ||
		!slices.Contains(allAuditedEventTypes, eventUserMergedInto) {
		t.Error("merge events missing from the audited event types")
	}
}
