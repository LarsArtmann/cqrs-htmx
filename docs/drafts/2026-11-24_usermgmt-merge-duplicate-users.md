# usermgmt: no merge for duplicate user identities — delete is the only exit

## Why

An installation can accumulate several user rows for the same human. `usermgmt` has no way to consolidate them: `DeleteUser` (`usermgmt/service_misc.go:100`) is the only exit, and it throws away the loser's credentials, external accounts, TOTP secret and memberships. The admin panel carries the same gap — the user detail page has `POST /users/{id}/delete` (`adminui/routes.go:250`) and no merge action.

## Symptom (live deployment, 2026-11-24)

`https://history.home.lan/admin/users` reports **9 users** on a single-human household. All nine rows are the same identity:

| user id (ULID)               | created_at           | visits | agent tokens |
| ---------------------------- | -------------------- | ------ | ------------ |
| `01M000JA6P0VR4Q1BNEPSJN3ME` | 2026-08-14T11:29:55Z | 6 932  | 1            |
| `01M08ZZEPDHTQ0KF6XBES2S494` | 2026-08-17T23:12:47Z | 0      | 0            |
| `01M1PA7XNN30GHP0XK47B42R96` | 2026-09-04T13:38:00Z | 0      | 0            |
| `01M1Q2G7YV6QQ2MDYS79WS67F2` | 2026-09-04T20:41:58Z | 0      | 0            |
| `01M1Q8VCPZJJ6XVQ4EQX8D4ATA` | 2026-09-04T22:32:55Z | 0      | 0            |
| `01M1QR8W2GZBPBJ9D7F2V91DF4` | 2026-09-05T03:02:25Z | 0      | 0            |
| `01M2N48TX0YVG84AGWTWCC8W9N` | 2026-09-16T12:50:05Z | 0      | 0            |
| `01M2Q8Q2Q57QN9SDQ6CC1D3BN9` | 2026-09-17T08:46:15Z | 0      | 0            |
| `01M2X007JZB9Y1WDYF35AD5GSD` | 2026-09-19T14:09:56Z | 0      | 0            |

All nine share `lars@larsartmann.cloud` / "Lars Artmann" (one email, nine streams). The eight newer rows own zero visits, zero tokens, zero bookmarks/downloads/omnibox rows — every row in every user-scoped table is under `01M000JA6…` or the anonymous zero id. They are OAuth2 auto-provisioning duplicates from before the email-match fix (same class the keep-first dedupe in `492e473e` addressed).

`MAX_USERS=1` now closes registration (verified in the live unit: `MAX_USERS=1`), so the residue cannot be replaced or re-registered away — only deleted or merged, and only one of those is lossless.

## Root cause (source)

- `rg -n -i merge usermgmt adminui identity-model` finds no command, event, or route. The aggregate surface is register / delete / change-email / change-display-name / credentials / TOTP / external link-unlink / platform roles / bots.
- `usermgmt/service_register.go:93` refuses a new registration whose email the read model already knows, but that guard is a read-model check at one instant; the nine rows predate it and the OAuth2 path that created them.
- Consumers hand-roll the consolidation. `browser-history/api/merge_duplicate_users_cli.go:30-46` re-points the app's own event payloads and read-model rows onto one survivor, then states why it cannot finish the job: "The command NEVER writes to users_view or usermgmt tables: account rows are event-sourced, so a read-model tombstone would be resurrected by the next usermgmt drain. Deleting the losing account afterwards is an owner step through the admin panel". That is today's whole workflow: one bespoke CLI per consumer, then N manual row deletions.

## Design

- [ ] `MergeUsers(ctx, survivor UserID, losers []UserID, reason string)` on `usermgmt.Service`, dispatching `UserMerged` on the survivor stream plus a tombstone event on each loser stream that carries `MergedInto`.
- [ ] Preserve on the survivor: credentials (dedupe by credential id), external accounts (provider + subject; a genuine conflict is a typed rejection naming both rows), TOTP (keep the survivor's secret, adopt the loser's only when the survivor has none), `EmailVerified`.
- [ ] Move memberships through the existing `AddMember`/`RemoveMember` commands so the membership journal stays the record; union per tenant with the highest role winning.
- [ ] Re-point bots (`Bot.OwnerID`) and revoke the losers' sessions.
- [ ] Read model: `MergedInto` on `users_view`, and `FindByEmail`/`FindBySubject` follow the pointer, so a merged id resolves instead of depending on map-scan order.
- [ ] Publish `UserMerged` so consumers can re-point their own `user_id` columns. The browser-history CLI is exactly that re-pointing, written by hand because there is no event to subscribe to.
- [ ] Admin UI: `POST /users/{id}/merge` with a survivor picker, plus a duplicates report (same email or same external subject) listing groups. Keep delete; add merge as the lossless option.

## To verify

- `rg -n -i merge usermgmt adminui identity-model` returns the new command, event, read-model field and route.
- A test seeds two users with one email and separate credentials and external accounts, merges them, and asserts: one live row, the loser row carries `MergedInto`, both credentials sit on the survivor, `users_view` count drops by one, one `UserMerged` event in the journal.
- Against the live case: merge the eight zombie rows into `01M000JA6P0VR4Q1BNEPSJN3ME`; `/admin/users` shows 1 and the dashboard still renders all 6,932 visits.

💘 Generated with Crush

Assisted-by: Crush:hf:deepseek-ai/DeepSeek-V4.1-Flash-1017
