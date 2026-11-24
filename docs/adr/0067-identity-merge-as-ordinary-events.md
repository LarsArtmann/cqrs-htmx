# ADR 0067: Identity Merge as Ordinary Events

**Status:** Accepted
**Date:** 2026-11-24
**Driver (owner):** GitHub issues #46 (user merge — the live deployment had
nine user rows for one human) and #47 (tenant rename/merge).
**Related:** [ADR 0060](0060-session-lifecycle-audit.md) (the audit-chain
precedent for forensics), [ADR 0062](0062-direct-imports-only.md) (the
no-new-aliases rule the merge symbols follow),
[plan of record](../planning/2026-11-24_06-30_SUPERB-identity-merge-open-issue-wave-pareto-plan.md).

## Context

Users end up duplicated through no fault of the model: a typo'd email, an
OAuth2 variant address, a second device that could not find the first
account. The live deployment of the admin panel had **nine rows for one
human** — only one of them carrying the 6,932-visit history. Deleting eight
of them would have destroyed attribution (and tombstoning them would have
orphaned their credentials). Tenants duplicate the same way ("Acme Corp" vs
"acme corp "), and renaming one meant delete-and-recreate because the
display name was only set at creation.

The naive design — a `MergedUsers` event carrying both identities plus
merge-specific handling in every projection — was rejected before this ADR:
it makes every read model learn the merge concept, forever.

## Decision

**Adopted assets are ordinary lifecycle events.** A merge is NOT a special
event shape; the survivor's stream simply receives the authenticators the
loser held — `CredentialAdded`, `ExternalAccountLinked`, `TOTPEnabled`,
`EmailVerified` (the last only on exact email equality, D12) — followed by
one `UserMerged` event recording which losers were absorbed. Every existing
projection adopts a merged user's assets with zero merge-specific code.

**Losers are tombstoned, never deleted.** One `UserMergedInto` event per
loser, dispatched LAST. The read models keep a pointer (`MergeTarget`) and
delete the live row; lookups follow the pointer; the SQL layer keeps a soft
tombstone (the pointer inside the existing Data blob — no DDL) and rebuilds
it on hydrate. Terminal identities answer typed conflicts naming the real
survivor (`usermgmt.merge_users.survivor_merged` with `merged_into`
context, D13); a deleted loser is a Conflict, not a Rejection, because
"already in a terminal state" is what conflicts mean in this module (D14).

**The merge is resumable, every step idempotent.** `Service.MergeUsers`
orders its steps: atomic survivor adoption first, membership role-set union
(survivor add/update, then loser removal — D7: the union, so no role is ever
lost), bot transfer, session revocation, loser tombstones last. A call that
fails halfway is simply made again: already-adopted assets are recognised
(and losers already absorbed are skipped via the pointer chain walk).

**Tenants rename in place and merge the same way.** `TenantRenamed` moves
the label; the tenant ID never moves (it is the stream identity, the Casbin
domain, and the key of every tenant-scoped row — which is exactly why
delete-and-recreate was wrong). `TenantMerged`/`TenantMergedInto` mirror
the user pair, with the roster moving through ordinary membership commands.
`NormalizeTenantName` (trim + lowercase) is the comparison the duplicates
report uses; uniqueness on create is an opt-in knob
(`ServiceConfig.UniqueTenantNames`), because a library never enforces a
default a consumer might disagree with.

**Bots follow their owner.** `BotTransferred` moves ownership, keeping the
bot's ID and token hash — integrations keep working across a user merge.

**The declarative path wires the same vocabulary.** systemadapter registers
all six commands, decodes all events, and its projections follow the
semantics (loser rows and policy entries removed, rename/transfer applied)
— but the orchestration (roster moves, bot transfer before the tombstone)
stays a consumer concern there, exactly as the service makes it ours on the
default path.

## Consequences

- Every projection that understands the ordinary lifecycle events understands
  merges; only the pointer semantics (`Retired`, `MergeTarget`) are new.
- Third-party external-account conflicts are refused BEFORE dispatch (naming
  both rows), because the keep-first index would otherwise silently resolve
  adoption ambiguously.
- Duplicate users can no longer be born through `Register` (the email guard
  canonicalizes), so email clusters in the duplicates report imply legacy
  data or direct-dispatch history — the report reads the read model, not the
  guard.
- The identity-model count moved to 33 events / 27 commands; the bijection,
  domain-counts, catalog, decoder, and schema gates all pin the new totals.
