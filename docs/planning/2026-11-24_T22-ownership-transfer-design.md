# T22 — Ownership transfer design (F162)

Status: implemented 2026-11-24 · Owner: adminui tenants surface (T21+T22 agent)

## Decision

Ownership transfer is TWO durable commands sequenced by the adminui HANDLER under
ONE confirm. No new usermgmt API (gotcha 60: owner-gated asks stay unsigned).
Both commands are already published:

- `Service.GrantRoles(ctx, actorID, tenantID, roles)` — usermgmt/service_grants.go:25.
  Grants EXACTLY `roles` (AddMember, then UpdateMemberRoles on conflict), so the
  handler passes the target's existing roles ∪ {owner} — a plain "grant owner" with
  only `[owner]` would silently strip the target's other roles.
- Demotion = `Service.UpdateMemberRoles(ctx, prior, tenantID, rolesMinusOwner)` —
  `RevokeRoles` (service_grants.go:45) removes the WHOLE membership, which is not a
  demotion. Only when the prior owner holds NO other role does the handler use
  `RevokeRoles`, and the outcome says the membership was removed, not merely demoted.

## Sequence

1. Resolve target (by email) and the prior owner (the tenant's owner-role member
   other than the target) from the roster snapshot.
2. Refusals BEFORE any dispatch (nothing lands on a refused transfer):
   - target not a member → refuse ("add them first");
   - target already owner (this covers self-transfer: the current owner naming
     themselves) → refuse;
   - ambiguous prior: more than one other owner → refuse (panel cannot pick);
   - no other owner exists → grant-only, with an honest note that nobody was demoted.
3. Grant owner to target (durable, journaled).
4. Demote prior: UpdateMemberRoles(prior, roles − owner) or RevokeRoles when that
   set is empty.

## Mid-sequence failure honesty (F164)

Steps 3 and 4 are independent dispatches; the handler never pretends step 4
happened when it failed. The outcome toast says exactly what landed:

- full success: `Ownership transferred to <target>. <prior> demoted.`
- grant failed: `Ownership transfer failed: <reason>. Nothing changed.`
- demote failed: `Owner granted to <target>, but demotion of <prior> failed:
  <reason>. <prior> STILL HOLDS the owner role.`

The tenant is never left ownerless by this flow: the grant happens first, so a
failed demotion leaves TWO owners (safe direction), never zero.

## Why not one command

An atomic TransferOwnership command is an upstream API ask (owner-gated, D2
family). Two public commands under handler sequencing is the honest v1: each
step is journaled, and the failure message names the partial state.
