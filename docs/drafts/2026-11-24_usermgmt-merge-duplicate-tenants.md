# usermgmt: no tenant merge or rename — delete is terminal per stream, duplicates are creatable

## Why

Tenant identity is a caller-supplied string (`usermgmt.CreateTenantRequest.ID`, `usermgmt/service_tenant.go:18`) and `decideCreateTenant` only checks the target stream's own state (`usermgmt/es_tenant_decide.go:24`), so two tenants can legally carry the same name under different IDs. There is no merge command and no rename command, and `DeleteTenant` (`usermgmt/service_tenant.go:77`) does not release the name: `FoldTenant` keeps `Name` on `EventTenantDeleted` (`identity-model/fold.go:329`) while `decideCreateTenant` rejects any create whose ID already has a name. Deleting a mistyped tenant therefore leaves its name permanently unusable, and the route's own `why` string admits the cost: "deleting tenants destroys their history" (`adminui/routes.go:398`).

## Symptom (verified against the current tree)

- **Duplicate name, different ID is allowed.** `usermgmt.DecideCreateTenant(id.NewStreamID(), "acme", "Acme")` against a zero state returns 1 event and `err=<nil>`. Two `acme` tenants, two rosters, two suspend/delete lifecycles, no link between them.
- **Delete-then-recreate is impossible.** Folding a create+delete state (`TenantState{Name: "acme", Deleted: true, DeleteReason: "typo"}`) and calling `DecideCreateTenant(sameStream, "acme", "Acme")` returns `[conflict:usermgmt.tenant.already_exists] tenant with this ID already exists`.
- **The panel makes this the normal path.** `adminui/handler_tenants.go:73` derives the stream ID from the submitted name (`identitymodel.NewTenantID(name)`), and `NewTenantID` is a raw brand wrapper with no casefold (`identity-model/id.go:100`). `Acme` and `acme` are two tenants, and once a name is deleted it can never be created again from the panel.
- **The admin surface has no consolidation at all.** Tenant routes are create / suspend / reactivate / delete (`adminui/routes.go:365-401`); the detail page offers Suspend and Delete. No rename, no merge.

## Root cause (source)

The tenant aggregate exposes exactly four commands — `CreateTenant`, `SuspendTenant`, `ReactivateTenant`, `DeleteTenant` (`usermgmt/service_tenant.go:18-98`). There is no `TenantRenamed` and no `TenantMerged` event in `identity-model`, and no equivalent service method. Plan-wise, rename is already recorded as an open ask: TODO item (h)(3), "tenant rename — display name is written once at creation, no rename command". Merge is the missing partner of delete, and recreation is the missing partner of both.

The fold even carries the intent that recreation should work — "Clear tombstone/suspension reasons too: a recreated tenant must not inherit the previous incarnation's reasons" (`identity-model/fold.go:311-315`) — but no create can ever reach that branch for an existing stream. The two halves disagree.

## Design

- [ ] `RenameTenant(ctx, tenantID, displayName)` dispatching `TenantRenamed`; the ID stays, the human label moves. This is the typo fix that does not destroy a roster.
- [ ] `MergeTenants(ctx, survivor TenantID, losers []TenantID, reason string)` dispatching `TenantMerged`: memberships move to the survivor through the existing `AddMember`/`RemoveMember` commands (union per actor, highest role wins), each loser stream gets a tombstone carrying `MergedInto`, and the survivor's name/display name stay canonical.
- [ ] Publish `TenantMerged` so consumers can re-point tenant-keyed rows, the same way `UserMerged` serves app-owned `user_id` columns.
- [ ] Optional normalized-name uniqueness on `CreateTenant` (trim + casefold), because the panel already treats the name as the key; today the collision is invisible until two rows exist.
- [ ] Admin UI: a merge action on the tenant detail page and a duplicates report (normalized name collisions listing every matching ID), plus a create form that names the existing tenant instead of a bare `usermgmt.tenant.already_exists`.
- [ ] Decide the recreate rule and make the two halves agree: either allow `CreateTenant` on a tombstoned stream (what the fold comment promises) or delete the comment and surface the terminal state in the panel.

## To verify

- Decider test: create → delete → create on the same stream returns the documented outcome, and two distinct streams with the same normalized name are rejected (or flagged) once the uniqueness option is on.
- Merge test: two tenants with overlapping memberships, merge → one live tenant, one tombstone with `MergedInto`, membership union on the survivor, one `TenantMerged` event in the journal.
- Panel test: `/tenants/{id}` renders the merge action, `POST /tenants/{id}/merge` moves the roster, and `/tenants` lists exactly one surviving row per normalized name.

💘 Generated with Crush

Assisted-by: Crush:hf:deepseek-ai/DeepSeek-V4.1-Flash-1017
