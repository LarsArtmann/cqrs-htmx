package usermgmt

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// mergeTestUserID builds a distinct, fixed UserID for a duplicate row.
func mergeTestUserID(n int) UserID {
	digits := []byte("0123456789ABCDEF")
	return NewUserID("01M46DUP000000000000000" + string(digits[n]))
}

func registerMergeTestUser(t *testing.T, svc *Service, n int, email string) UserID {
	t.Helper()
	resp, err := svc.Register(context.Background(), RegisterRequest{
		ID:    mergeTestUserID(n),
		Email: email,
	})
	if err != nil {
		t.Fatalf("Register user %d: %v", n, err)
	}
	return resp.User.ID
}

func linkMergeTestAccount(t *testing.T, svc *Service, userID UserID, provider, subject string) {
	t.Helper()
	aggID, err := aggIDFromUser(userID)
	if err != nil {
		t.Fatalf("aggIDFromUser: %v", err)
	}
	if err := svc.dispatcher.Dispatch(context.Background(),
		identitymodel.NewLinkExternalAccountCmd(aggID, provider, subject, "", "")); err != nil {
		t.Fatalf("LinkExternalAccount: %v", err)
	}
}

// issue46Seed captures everything the assertions need about the seeded
// nine-row, one-human world of the issue #46 scenario.
type issue46Seed struct {
	survivor     UserID
	dups         [8]UserID
	tenant       TenantID
	credID       []byte
	botID        BotID
	botToken     string
	sessionToken string
}

// seedIssue46Duplicates builds the live deployment's shape (issue #46): one
// human with nine rows, each duplicate carrying part of the human's history.
func seedIssue46Duplicates(t *testing.T, svc *Service) *issue46Seed {
	t.Helper()
	ctx := context.Background()

	const humanEmail = "one.human@example.com"
	seed := &issue46Seed{}
	seed.survivor = registerMergeTestUser(t, svc, 0, humanEmail)
	for i := range seed.dups {
		// Every duplicate under a different email spelling — that is how the
		// live rows came to exist past the duplicate-email guard.
		seed.dups[i] = registerMergeTestUser(t, svc, i+1,
			"one.human+dup"+string(rune('1'+i))+"@example.com")
	}

	seed.credID = []byte{9, 9}
	if err := svc.AddCredential(ctx, seed.dups[0], WebAuthnCredential{
		ID: seed.credID, PublicKey: []byte{1, 2, 3}, AttestationType: "none", Name: "loser key",
	}); err != nil {
		t.Fatalf("AddCredential: %v", err)
	}
	linkMergeTestAccount(t, svc, seed.dups[1], "google", "subj-dup-2")

	tenant, err := svc.CreateTenant(ctx, CreateTenantRequest{
		ID:   NewTenantID("01M46TENANT0000000000000X1"),
		Name: "tenant-x",
	})
	if err != nil {
		t.Fatalf("CreateTenant: %v", err)
	}
	seed.tenant = tenant.ID
	if err := svc.AddMember(ctx, ActorIDFromUser(seed.survivor), tenant.ID, []Role{RoleUser}); err != nil {
		t.Fatalf("AddMember survivor: %v", err)
	}
	if err := svc.AddMember(ctx, ActorIDFromUser(seed.dups[2]), tenant.ID, []Role{RoleAdmin}); err != nil {
		t.Fatalf("AddMember loser: %v", err)
	}

	botResult, err := svc.RegisterBot(ctx, RegisterBotRequest{
		ID: NewBotID("01M46BOT000000000000000B1"), Name: "loser-bot", OwnerID: seed.dups[3],
	})
	if err != nil {
		t.Fatalf("RegisterBot: %v", err)
	}
	seed.botID, seed.botToken = botResult.Bot.ID, botResult.Token

	loserSession, err := identitymodel.NewSession(seed.dups[4], time.Hour)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	if err := svc.sessions.Create(ctx, loserSession); err != nil {
		t.Fatalf("sessions.Create: %v", err)
	}
	seed.sessionToken = loserSession.Token

	// The visits-analog: give one duplicate a long event history.
	for i := range 5 {
		if err := svc.ChangeDisplayName(ctx, seed.dups[5], "visit-"+string(rune('a'+i))); err != nil {
			t.Fatalf("ChangeDisplayName %d: %v", i, err)
		}
	}
	return seed
}

// TestService_MergeUsers_Issue46NineRowsOneHuman is the wave's first kill
// switch: the live deployment's shape (issue #46 — nine user rows for one
// human, each row with its own history) collapses into one live identity
// whose assets, memberships, bots and attribution survive.
func TestService_MergeUsers_Issue46NineRowsOneHuman(t *testing.T) {
	svc := setupTestService(t)
	ctx := context.Background()
	seed := seedIssue46Duplicates(t, svc)

	if err := svc.MergeUsers(ctx, seed.survivor, seed.dups[:], "issue #46: nine rows, one human"); err != nil {
		t.Fatalf("MergeUsers: %v", err)
	}
	assertIssue46OneLiveIdentity(t, svc, seed)
	assertIssue46AssetsMoved(t, svc, ctx, seed)
	assertIssue46RetiredWithAttribution(t, svc, ctx, seed)
}

func assertIssue46OneLiveIdentity(t *testing.T, svc *Service, seed *issue46Seed) {
	t.Helper()
	if got := svc.readModel.Count(); got != 1 {
		t.Fatalf("live user count = %d, want 1", got)
	}
	live, ok := svc.readModel.FindByEmail("one.human@example.com")
	if !ok || live.ID.Get().String() != seed.survivor.Get().String() {
		t.Fatalf("FindByEmail = %v,%v; want survivor %s", live, ok, seed.survivor.Get())
	}
	survivorAgg, err := aggIDFromUser(seed.survivor)
	if err != nil {
		t.Fatalf("aggIDFromUser survivor: %v", err)
	}
	state, _, err := svc.repository.LoadRef(context.Background(),
		id.NewStreamRef(aggregateTypeUser, survivorAgg))
	if err != nil {
		t.Fatalf("LoadRef survivor: %v", err)
	}
	if !state.HasCredential(seed.credID) {
		t.Error("survivor did not adopt the loser's credential")
	}
	foundAccount := false
	for _, ea := range state.ExternalAccounts {
		if ea.Provider == "google" && ea.Subject == "subj-dup-2" {
			foundAccount = true
		}
	}
	if !foundAccount {
		t.Error("survivor did not adopt the loser's external account")
	}
	if len(state.MergedUserIDs) != len(seed.dups) {
		t.Fatalf("MergedUserIDs = %d entries, want %d", len(state.MergedUserIDs), len(seed.dups))
	}
}

func assertIssue46AssetsMoved(t *testing.T, svc *Service, ctx context.Context, seed *issue46Seed) {
	t.Helper()
	var survivorRoles []Role
	for _, mem := range svc.membershipReadModel.FindByActor(seed.survivor.Get().String()) {
		if mem.TenantID.Get() == seed.tenant.Get() {
			survivorRoles = mem.Roles
		}
	}
	if !roleSetHas(survivorRoles, RoleUser) || !roleSetHas(survivorRoles, RoleAdmin) {
		t.Errorf("survivor tenant roles = %v, want the union [user admin]", survivorRoles)
	}
	if rows := svc.membershipReadModel.FindByActor(seed.dups[2].Get().String()); len(rows) != 0 {
		t.Errorf("loser memberships = %d rows, want 0", len(rows))
	}

	if bots := svc.botReadModel.FindByOwner(seed.dups[3]); len(bots) != 0 {
		t.Errorf("loser still owns %d bots", len(bots))
	}
	owned := svc.botReadModel.FindByOwner(seed.survivor)
	if len(owned) != 1 || owned[0].ID.Get() != seed.botID.Get() {
		t.Fatalf("survivor owns %v, want the transferred bot", owned)
	}
	if resolved, ok := svc.ResolveBotByToken(seed.botToken); !ok || resolved.ID.Get() != seed.botID.Get() {
		t.Error("transferred bot no longer authenticates with its original token")
	}

	if _, err := svc.sessions.Find(ctx, seed.sessionToken); !errors.Is(err, ErrSessionNotFound) {
		t.Errorf("loser session Find err = %v, want ErrSessionNotFound", err)
	}
}

func assertIssue46RetiredWithAttribution(t *testing.T, svc *Service, ctx context.Context, seed *issue46Seed) {
	t.Helper()
	survivorAgg, err := aggIDFromUser(seed.survivor)
	if err != nil {
		t.Fatalf("aggIDFromUser survivor: %v", err)
	}
	for i, dup := range seed.dups {
		aggID, err := aggIDFromUser(dup)
		if err != nil {
			t.Fatalf("aggIDFromUser dup %d: %v", i, err)
		}
		if !svc.readModel.Retired(aggID) {
			t.Errorf("duplicate %d is not retired", i)
		}
		target, merged := svc.readModel.MergeTarget(aggID)
		if !merged || target.Get() != survivorAgg.Get() {
			t.Errorf("duplicate %d MergeTarget = %v,%v; want %s", i, target, merged, survivorAgg.Get())
		}
	}

	// Attribution survives: the visited duplicate's journal is intact and the
	// retirement is its LAST event (task 57: tombstone last).
	visitedAgg, err := aggIDFromUser(seed.dups[5])
	if err != nil {
		t.Fatalf("aggIDFromUser visited: %v", err)
	}
	journal, ok := svc.store.(event.Journal)
	if !ok {
		t.Fatalf("event store %T does not expose a journal", svc.store)
	}
	all, err := journal.ReadAll(ctx)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	var visitedEvents []event.Event
	for _, evt := range all {
		if evt.StreamID().Get() == visitedAgg.Get() {
			visitedEvents = append(visitedEvents, evt)
		}
	}
	if len(visitedEvents) < 7 {
		t.Fatalf("visited stream has %d events, want at least 7 (history preserved)", len(visitedEvents))
	}
	if last := visitedEvents[len(visitedEvents)-1]; last.Type() != eventUserMergedInto {
		t.Errorf("last event on loser stream = %v, want UserMergedInto (tombstone last)", last.Type())
	}
}

func roleSetHas(roles []Role, want Role) bool {
	return slices.Contains(roles, want)
}

// TestService_MergeUsers_IdempotentRerun proves the resumable contract: a
// second call with the same arguments answers nil and changes nothing.
func TestService_MergeUsers_IdempotentRerun(t *testing.T) {
	svc := setupTestService(t)
	ctx := context.Background()

	survivor := registerMergeTestUser(t, svc, 0, "keep@example.com")
	loser := registerMergeTestUser(t, svc, 1, "keep.dup@example.com")
	if err := svc.MergeUsers(ctx, survivor, []UserID{loser}, "rerun"); err != nil {
		t.Fatalf("first MergeUsers: %v", err)
	}
	survivorAgg, err := aggIDFromUser(survivor)
	if err != nil {
		t.Fatalf("aggIDFromUser: %v", err)
	}
	_, versionBefore, err := svc.repository.LoadRef(ctx, id.NewStreamRef(aggregateTypeUser, survivorAgg))
	if err != nil {
		t.Fatalf("LoadRef: %v", err)
	}

	if err := svc.MergeUsers(ctx, survivor, []UserID{loser}, "rerun"); err != nil {
		t.Fatalf("re-run MergeUsers: %v", err)
	}
	_, versionAfter, err := svc.repository.LoadRef(ctx, id.NewStreamRef(aggregateTypeUser, survivorAgg))
	if err != nil {
		t.Fatalf("LoadRef after: %v", err)
	}
	if versionAfter != versionBefore {
		t.Errorf("survivor version moved %v -> %v on an idempotent re-run", versionBefore, versionAfter)
	}
	if got := svc.readModel.Count(); got != 1 {
		t.Errorf("live user count = %d, want 1", got)
	}
}

// TestService_MergeUsers_InputRejections covers the malformed-input class
// (D14: rejections are reserved for malformed input).
func TestService_MergeUsers_InputRejections(t *testing.T) {
	svc := setupTestService(t)
	ctx := context.Background()

	survivor := registerMergeTestUser(t, svc, 0, "valid@example.com")
	loser := registerMergeTestUser(t, svc, 1, "other@example.com")

	cases := []struct {
		name   string
		surv   UserID
		losers []UserID
		code   string
	}{
		{"survivor required", UserID{}, []UserID{loser}, "usermgmt.merge_users.survivor_required"},
		{"no losers", survivor, nil, "usermgmt.merge_users.no_losers"},
		{"self merge", survivor, []UserID{survivor}, "usermgmt.merge_users.self_merge"},
		{"duplicate loser", survivor, []UserID{loser, loser}, "usermgmt.merge_users.duplicate_loser"},
		{
			"loser not found",
			survivor,
			[]UserID{NewUserID("01M46GHOST0000000000000Z9")},
			"usermgmt.merge_users.loser_not_found",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ef := requireDomainError(t, svc.MergeUsers(ctx, tc.surv, tc.losers, "x"), errorfamily.Rejection, tc.code)
			if tc.name == "loser not found" && ef.ErrorContext()["loser_id"] == "" {
				t.Error("loser_not_found does not name the loser")
			}
		})
	}
}

// TestService_MergeUsers_TerminalConflicts covers D13 and D14: merged-away
// and deleted identities answer conflicts that name the truth.
func TestService_MergeUsers_TerminalConflicts(t *testing.T) {
	svc := setupTestService(t)
	ctx := context.Background()

	a := registerMergeTestUser(t, svc, 0, "a@example.com")
	b := registerMergeTestUser(t, svc, 1, "b@example.com")
	c := registerMergeTestUser(t, svc, 2, "c@example.com")
	d := registerMergeTestUser(t, svc, 3, "d@example.com")
	e := registerMergeTestUser(t, svc, 4, "e@example.com")

	if err := svc.MergeUsers(ctx, a, []UserID{b}, "chain"); err != nil {
		t.Fatalf("merge b into a: %v", err)
	}

	t.Run("survivor merged names the real survivor", func(t *testing.T) {
		ef := requireDomainError(t, svc.MergeUsers(ctx, b, []UserID{c}, "x"),
			errorfamily.Conflict, "usermgmt.merge_users.survivor_merged")
		if ef.ErrorContext()["merged_into"] != a.Get().String() {
			t.Errorf("merged_into = %q, want %q", ef.ErrorContext()["merged_into"], a.Get())
		}
	})

	t.Run("loser merged elsewhere is a conflict", func(t *testing.T) {
		ef := requireDomainError(t, svc.MergeUsers(ctx, c, []UserID{b}, "x"),
			errorfamily.Conflict, "usermgmt.merge_users.loser_merged")
		if ef.ErrorContext()["merged_into"] != a.Get().String() {
			t.Errorf("merged_into = %q, want %q", ef.ErrorContext()["merged_into"], a.Get())
		}
	})

	t.Run("loser already in this survivor is skipped", func(t *testing.T) {
		if err := svc.MergeUsers(ctx, a, []UserID{b}, "again"); err != nil {
			t.Errorf("re-merge of absorbed loser: %v", err)
		}
	})

	t.Run("deleted loser is a conflict not a rejection", func(t *testing.T) {
		if err := svc.DeleteUser(ctx, d, "gone"); err != nil {
			t.Fatalf("DeleteUser: %v", err)
		}
		ef := requireDomainError(t, svc.MergeUsers(ctx, a, []UserID{d}, "x"),
			errorfamily.Conflict, "usermgmt.merge_user_into.deleted")
		if ef.ErrorContext()["loser_id"] != d.Get().String() {
			t.Errorf("loser_id = %q, want %q", ef.ErrorContext()["loser_id"], d.Get())
		}
	})

	t.Run("deleted survivor is refused", func(t *testing.T) {
		if err := svc.DeleteUser(ctx, e, "gone"); err != nil {
			t.Fatalf("DeleteUser: %v", err)
		}
		_ = requireDomainError(t, svc.MergeUsers(ctx, e, []UserID{c}, "x"),
			errorfamily.Rejection, "usermgmt.merge_users.deleted")
	})
}

// TestService_MergeUsers_ExternalAccountConflict proves the adoption refuses
// to steal an external account another live user holds (task 51).
func TestService_MergeUsers_ExternalAccountConflict(t *testing.T) {
	svc := setupTestService(t)
	ctx := context.Background()

	survivor := registerMergeTestUser(t, svc, 0, "s@example.com")
	loser := registerMergeTestUser(t, svc, 1, "l@example.com")
	third := registerMergeTestUser(t, svc, 2, "t@example.com")

	// The third party links FIRST: the keep-first read-model index then
	// resolves the shared account to the third user, and the loser's own link
	// (allowed by the decider, which only checks the user's own accounts)
	// would have its adoption steal the entry.
	linkMergeTestAccount(t, svc, third, "github", "shared-subject")
	linkMergeTestAccount(t, svc, loser, "github", "shared-subject")

	ef := requireDomainError(t, svc.MergeUsers(ctx, survivor, []UserID{loser}, "x"),
		errorfamily.Conflict, "usermgmt.merge_users.external_account_conflict")
	ctxMap := ef.ErrorContext()
	if ctxMap["loser_id"] != loser.Get().String() {
		t.Errorf("loser_id = %q, want %q", ctxMap["loser_id"], loser.Get())
	}
	if ctxMap["holder_id"] != third.Get().String() {
		t.Errorf("holder_id = %q, want %q", ctxMap["holder_id"], third.Get())
	}
	if ctxMap["provider"] != "github" || ctxMap["subject"] != "shared-subject" {
		t.Errorf("context = %v, want provider and subject named", ctxMap)
	}

	// Unlink resolves it: after the third user releases the account the merge
	// goes through (a credential keeps the unlink from stripping its last
	// authentication method).
	if err := svc.AddCredential(ctx, third, WebAuthnCredential{
		ID: []byte{7}, PublicKey: []byte{1}, AttestationType: "none",
	}); err != nil {
		t.Fatalf("AddCredential third: %v", err)
	}
	thirdAgg, err := aggIDFromUser(third)
	if err != nil {
		t.Fatalf("aggIDFromUser: %v", err)
	}
	if err := svc.dispatcher.Dispatch(ctx,
		identitymodel.NewUnlinkExternalAccountCmd(thirdAgg, "github", "shared-subject")); err != nil {
		t.Fatalf("UnlinkExternalAccount: %v", err)
	}
	if err := svc.MergeUsers(ctx, survivor, []UserID{loser}, "x"); err != nil {
		t.Fatalf("MergeUsers after unlink: %v", err)
	}
}

// TestService_MergeUsers_MembershipUnionKeepsBothDirections checks the D7
// union across two tenants where each side holds a different role.
func TestService_MergeUsers_MembershipUnionKeepsBothDirections(t *testing.T) {
	svc := setupTestService(t)
	ctx := context.Background()

	survivor := registerMergeTestUser(t, svc, 0, "u@example.com")
	loser := registerMergeTestUser(t, svc, 1, "u.dup@example.com")

	tenantA, err := svc.CreateTenant(ctx, CreateTenantRequest{ID: NewTenantID("01M46TENANT00000000000UN1A"), Name: "a"})
	if err != nil {
		t.Fatalf("CreateTenant A: %v", err)
	}
	tenantB, err := svc.CreateTenant(ctx, CreateTenantRequest{ID: NewTenantID("01M46TENANT00000000000UN1B"), Name: "b"})
	if err != nil {
		t.Fatalf("CreateTenant B: %v", err)
	}
	if err := svc.AddMember(ctx, ActorIDFromUser(survivor), tenantA.ID, []Role{RoleUser}); err != nil {
		t.Fatalf("AddMember A: %v", err)
	}
	if err := svc.AddMember(ctx, ActorIDFromUser(loser), tenantA.ID, []Role{RoleAdmin}); err != nil {
		t.Fatalf("AddMember A loser: %v", err)
	}
	if err := svc.AddMember(ctx, ActorIDFromUser(loser), tenantB.ID, []Role{RoleViewer}); err != nil {
		t.Fatalf("AddMember B loser: %v", err)
	}

	if err := svc.MergeUsers(ctx, survivor, []UserID{loser}, "union"); err != nil {
		t.Fatalf("MergeUsers: %v", err)
	}

	rolesByTenant := map[string][]Role{}
	for _, mem := range svc.membershipReadModel.FindByActor(survivor.Get().String()) {
		rolesByTenant[mem.TenantID.Get()] = mem.Roles
	}
	if !roleSetHas(rolesByTenant[tenantA.ID.Get()], RoleUser) ||
		!roleSetHas(rolesByTenant[tenantA.ID.Get()], RoleAdmin) {
		t.Errorf("tenant A roles = %v, want union [user admin]", rolesByTenant[tenantA.ID.Get()])
	}
	if !roleSetHas(rolesByTenant[tenantB.ID.Get()], RoleViewer) {
		t.Errorf("tenant B roles = %v, want the loser's viewer role inherited", rolesByTenant[tenantB.ID.Get()])
	}
	if rows := svc.membershipReadModel.FindByActor(loser.Get().String()); len(rows) != 0 {
		t.Errorf("loser memberships = %d rows, want 0", len(rows))
	}
}
