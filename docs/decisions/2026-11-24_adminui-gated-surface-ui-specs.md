# Adminui gated-surface UI specs — T31 (F221–F224), PAPER ONLY

**Date:** 2026-11-24 · **Status:** DRAFT — design specs only. NOTHING here is implemented; every surface renders only the day its owner gate opens.
**Provenance:** fine tasks F221–F224 of T31 in [`docs/planning/2026-11-24_06-31_SUPERB-adminui-next-wave-pareto-plan.md`](../planning/2026-11-24_06-31_SUPERB-adminui-next-wave-pareto-plan.md) (Table B, §T31).
**Ground truth:** [`docs/decisions/2026-11-24_adminui-d1-d3-and-upstream-asks.md`](2026-11-24_adminui-d1-d3-and-upstream-asks.md) — the T02 decision packet. Its API sketches and its verified evidence are the ONLY APIs these specs consume. Where a surface needs something the packet does not sketch, the spec says so and refuses to invent it.
**Hard rule:** every external dependency is labeled `gated: needs <D1/D2/D3/ask A–H/M2M row>`. A spec section is not implementable until every label in its "Data needs" is discharged by the owner.

---

## 0. Panel conventions every spec assumes (verified in the working tree, 2026-11-24)

- **Components:** `display.Card`, `display.PageHeader` + `navigation.Breadcrumbs`, `display.Table`, `display.DefinitionList`, `display.StatCard`, `display.Grid`, `display.EmptyState` (adminui wrapper `empty()`), `display.Button`, `display.Badge`/`statusBadge` (`adminui/components.templ:18-37`), `display.CopyButton` (wrapper `copyableID`, `components.templ:99-104`), `display.Modal`, `display.Dropdown`, `forms.Input`, `dangerZoneHeader` (`components.templ:106-111`).
- **Confirm modal:** ONE shared native `<dialog>` (`#admin-confirm-modal`, rendered once by `pageFoot`, `layout.templ:221-234`). Buttons carry `data-confirm` (body text) + `data-confirm-title`; `admin.js` intercepts `htmx:confirm`, fills the dialog, and issues the original htmx request only on Confirm (`adminui/assets/admin.js:56-99`). Cancel, backdrop, Escape close without acting; focus trap and focus restore are native `<dialog>` behavior. Reason-bearing destructive actions (impersonation, revoke-all, TOTP bypass) need an input INSIDE the flow, so those specs use dedicated inline form cards (the `profileEditCard` precedent, `users.templ:457-499`) plus the shared modal for the final yes/no — the shared modal stays text-only.
- **Toasts:** handlers set `HX-Trigger: adminui:toast` via `triggerToast` (`adminui/render.go:46-59`); `admin.js` bridges it to `tcShowToast` (`admin.js:39-54`). Kinds: `ok` / `err` / `warn` / `info`.
- **CSRF:** `<meta name="csrf-token">` in the head (`layout.templ:38-41`, fed by `pageData.CSRFMeta`); `admin.js` attaches `X-CSRF-Token` to every htmx request at `htmx:configRequest` (`admin.js:10-17`). Every new POST route is covered automatically; no per-form token fields.
- **htmx discipline (M058):** content refreshes are targeted swaps (`hx-post`/`hx-target="this"`, `hx-disabled-elt="this"`); anything that changes identity or session — impersonation start/exit, anything that flips the session cookie — is a FULL navigation (plain form POST + redirect), the same honesty rule `listPager` applies to links (`components.templ:132-136`). No `hx-boost`.
- **Routes:** every new route is a `routeSpec` row in `buildRouteTable` (`adminui/routes.go:109-114`) — `method`, `pattern`, `panel`, `guarded`, `mode` (`routeModeSuperAdmin` for all four specs' writes unless stated), `write: true`, and a one-line `why`. The manifest golden `TestRoutes_ManifestGolden` (`adminui/routes_manifest_test.go:31`) changes in the same commit, per its own comment. A row that forgets its mode is served by NO mode — absence fails loudly.
- **Badge grammar:** dots encode STATE (`statusBadge`); accent badges without dots encode ACTIONS (`components.templ:26-29`). Impersonation/lockout states get dots; "impersonation started" audit chips do not.
- **Honesty rules inherited from T01:** dead surfaces never render (a card for a capability the deployment cannot serve is a lie, `users.templ:396-400`); the audit window is dashed, not fabricated (`users.templ:252-262`); banner strips use the amber family `bg-amber-50 dark:bg-amber-900/20` (`users.templ:360-368`); text sits on `gray-600`, not `gray-500` (4.5:1 contrast, axe sweep 2026-11-14).
- **a11y baseline:** WCAG 2.5.3 Label in Name on any trigger with visible text (the avatar+email Dropdown precedent, `layout.templ:169-178`); `role="status" aria-live="polite"` for one-shot announcements; `required` + visible `Label` on every mandatory field; keyboard-first (native `<dialog>` gives Esc + focus trap for free).

---

## Spec 1 — F221: Impersonation kit (banner, exit, audit chip, self-refusal, origin disclosure)

**Gate summary:** `gated: needs D1 owner tick + opt-in config knob` — and NOTHING else. `BeginImpersonation`/`EndImpersonation` are published usermgmt v4.20.0; D1 is a panel-surface + guardrail decision only (packet §3 D1, evidence `usermgmt/service_impersonation.go:26/:109`). This is why sessions+impersonation leads D3's ordering.

### 1.1 Config gate

`Config.ImpersonationEnabled` (default **false**). When false: the two routes below are not registered, the action card, banner, menu row, and audit chips for impersonation never render — the T01 dead-surface discipline. When true, the panel behaves exactly as speced. The knob is the whole D1 decision made tangible: one opt-in, no API change.

### 1.2 Routes (routeSpec rows)

| Method | Pattern | Panel | Mode | Write | `why` |
| --- | --- | --- | --- | --- | --- |
| POST | `/users/{id}/impersonate` | Users | SuperAdmin | yes | "start an impersonation session; super_admin is a Service guard, the route is super-admin panel only" |
| POST | `/impersonation/exit` | Users | SuperAdmin | yes | "end the admin's impersonation session; the original session is untouched" |

Both are full-navigation POSTs (see §0). Manifest golden updated in the same commit.

### 1.3 Wireframe — action card on user detail

Placed after `tenantAccessCard`, before `userTimelineCard` — privileged, but NOT in the danger zone: impersonation is reversible and audited, and parking it next to irreversible deletes inflates both.

```
┌ Act as this user ──────────────────────────────────────────────────┐
│ Open a signed-in session as ada@example.com to reproduce an issue │
│ or audit their access. Your own session stays signed in; the      │
│ panel shows a banner until you exit. The reason is recorded.      │
│                                                                    │
│  Reason (required)                                                │
│  ┌──────────────────────────────────────────────┐                 │
│  │ ticket 4821 — reported passkey loop          │                 │
│  └──────────────────────────────────────────────┘                 │
│                             [ Act as ada@example.com ]            │
└────────────────────────────────────────────────────────────────────┘
```

- The card renders for super-admins only, and never for the signed-in admin's own detail page — `userDetailData.IsSelf` already exists (`adminui/handler_users.go:190`) and hides it (self-impersonation refusal is a SERVICE rejection, `usermgmt/service_impersonation.go:36-41`; the panel refuses to offer it too).
- The submit button is a plain form POST (full navigation): `POST /users/{id}/impersonate` with the reason field. The shared confirm modal intercepts it via `data-confirm` — body: "Act as **ada@example.com**? You take on their permissions. Every write is attributed to them; the audit log names both of you. Your own session stays signed in." Title: "Begin impersonation".

### 1.4 Wireframe — persistent banner while impersonating

Rendered by `Layout` as the FIRST element inside the content shell, above `PageHeader`, on every page (desktop + mobile), for the whole session lifetime:

```
┌────────────────────────────────────────────────────────────────────┐
│ ● ACTING AS ada@example.com — impersonation by you, reason         │
│   "ticket 4821 — reported passkey loop". Writes are attributed to  │
│   the target; the audit log names both.            [ Exit ]        │
└────────────────────────────────────────────────────────────────────┘
```

- Visual: `bg-gray-900 text-white` strip with an accent-colored dot — deliberately NOT the amber warning family: impersonation is an identity state, not a danger, but it must be unmissable and unmistakable against the normal chrome. `Exit` is a plain form button POSTing `/impersonation/exit` (full navigation), `data-confirm`: "End impersonation? You return to your own session — ada@example.com's session ends." Title: "Exit impersonation".
- Session-origin disclosure is baked into the banner copy: it names the target, the initiating admin (when another admin is the initiator — a resumed admin sees "impersonation by ops@…"), and the reason, which is exactly the typed `Impersonation{By, Reason, At}` origin (`identity-model/session.go:32-39`).
- The top-bar user Dropdown gains one item above "Sign out" while impersonating: "Exit impersonation (acting as ada@…)" — the mobile escape hatch and the Label-in-Name-safe affordance (`layout.templ` Dropdown pattern).

### 1.5 Interaction contract

**States**
1. *Opt-in off* → no card, no banner, no routes. Nothing.
2. *Super-admin, target detail* → card renders; reason empty; button enabled (server re-checks).
3. *Impersonation active* → banner on every page; user menu carries the exit item; the impersonated session behaves as the target user everywhere else in the panel (the panel is mode-scoped super-admin, but the SESSION is the target's — writes dispatch as the target; that is the point and the risk the banner names).
4. *Exited* → full-navigation redirect back to the originating user detail page + toast `ok` "Impersonation ended".

**Refusals (each an honest toast or error page, never a silent noop)**
- *Self-impersonation* (raced past the hidden card): Service rejection `usermgmt.impersonation.self_impersonation` → toast `err` "You cannot impersonate yourself."
- *Empty reason*: client `required` on the field; Service rejection `usermgmt.impersonation.reason_required` → toast `err` "An audit reason is required to impersonate."
- *Insufficient privileges*: route posture answers first (super-admin panel only); Service guard `usermgmt.impersonation.insufficient_privileges` is the backstop.
- *Target gone*: `usermgmt.impersonation.target_not_found` → the standard "User not found" error page (`writeErrorPage`, `handler_users.go` pattern).
- *Exit with no impersonation session* (double-exit, expired session): Service rejection `usermgmt.impersonation.not_impersonation` → toast `warn` "No active impersonation session — you are signed in as yourself." and redirect to the dashboard. `EndImpersonation` deletes ONLY impersonation-origin sessions and journals the end with cause `SessionEndImpersonationEnded` (`usermgmt/service_impersonation.go:109-130`) — the admin's original session is untouched, so no re-login ever happens.

**Empty states:** none needed — every state above is either a card, a banner, or an absence.

**Audit chip (the plan's item 75 "privilege audit chip")**
- Add two entries to `auditLabels` (`adminui/audit_labels.go`): `impersonation_begun` → "Impersonation started", `impersonation_ended` → "Impersonation ended". Both already flow into the audit log (impersonation is a session/auth event class; the audited set is `allUserEventTypes + allAuthEventTypes`, packet §2 ask H). Neither is in the label map today — the chip is raw-action fallback text until this lands.
- Chip rendering in the user-detail timeline and audit table: dotless `badge()` (an action, not a state), label + `· by <admin email> · "<reason>"`. Tone via `auditActionBadge`: falls through to `BadgePrimary` (accent) — correct, these are mutations, and they must NOT read as errors.

### 1.6 a11y notes

- Banner is `role="status" aria-live="polite"`: announces once on appearance (begin redirect, exit redirect) without shouting over subsequent navigation. It is the first element in content DOM order; `Exit` is the first interactive control after the skip link while impersonating.
- Banner text contrast: white on `gray-900` — clears AAA; the accent dot is decorative (`aria-hidden`).
- The action card's reason field: visible `Label` "Reason (required)", `required`, `HelpText` "Recorded in the audit log next to your name." Confirm modal + reason form: native `<dialog>` focus trap, focus restore to the trigger on close (existing behavior, `admin.js:56-99`).
- Dropdown exit item label contains the full visible text ("Exit impersonation (acting as ada@example.com)") per the Label-in-Name rule that the first axe sweep enforced on the user menu (`layout.templ:169-178`).

### 1.7 Data needs

| Need | Source | Gate |
| --- | --- | --- |
| Start impersonation | `BeginImpersonation(ctx, callerID, targetID, reason) (*Session, error)` — published v4.20.0, guards verified (packet §3 D1) | `gated: needs D1 tick + opt-in knob` (no API ask) |
| End impersonation | `EndImpersonation(ctx, token) error` — published; impersonation-origin-only, cause journaled | `gated: needs D1 tick + opt-in knob` |
| Banner/session awareness | panel-side: the session middleware type-asserts `session.Origin.(Impersonation)` (same pattern as `usermgmt/service_impersonation.go:115`) and feeds `pageData` an `Impersonation` struct (target email, admin email, reason) | `gated: needs D1 tick` — panel work only |
| Audit chip labels | `AuditLog` entries `impersonation_begun`/`impersonation_ended` already captured; two `auditLabels` rows are panel-side | `gated: needs D1 tick` |
| Sessions-list origin disclosure | ask B's `SessionView` origin kind + impersonation actor (Spec 3 consumes the same field) | `gated: needs ask B` — Spec 1 renders without it; the sessions card adds the disclosure row when B lands |

### 1.8 Build order (what lands first on D1 tick)

1. `Config.ImpersonationEnabled` + route rows + manifest golden (the gate itself; everything else is unreachable without it).
2. Action card + reason form + confirm copy + refusals (begin path).
3. Banner + Dropdown exit item + exit route (the persistent state).
4. Audit labels + chips (read-only polish; the entries already exist).

---

## Spec 2 — F222: Bots pack (list, register + show-once reveal, rotate, delete)

**Gate summary:** `gated: needs D2 tick + D3 ordering + M2M owner-decisions row` (packet §3 D3: the TODO_LIST "M2M owner decisions" D1–D4 row must tick before any bot panel work — its own "do NOT start W1" rule — and bots run SECOND behind sessions+impersonation). Within that: the LIST additionally `gated: needs ask E`; register/delete use published API.

### 2.0 New panel

New const `panelBots = "Bots"`, sidebar nav item (super-admin mode only — bots are platform-scoped), plus the route rows below. The manifest golden and the T27 route↔nav↔href surface gate both change with the same commit discipline.

### 2.2 Routes (routeSpec rows)

| Method | Pattern | Panel | Mode | Write | `why` |
| --- | --- | --- | --- | --- | --- |
| GET | `/bots` | Bots | SuperAdmin | no | "enumerate registered bots; the Service cannot list bots until ask E" |
| POST | `/bots` | Bots | SuperAdmin | yes | "register a bot and reveal its one-time API token (published RegisterBot)" |
| POST | `/bots/{id}/rotate` | Bots | SuperAdmin | yes | "rotate a bot token by composing published RegisterBot + DeleteBot; no RotateBot API exists" |
| POST | `/bots/{id}/delete` | Bots | SuperAdmin | yes | "permanently delete a bot; every caller with its token is refused immediately" |

`GET /bots` renders ONLY when ask E has landed (it has no data source otherwise); the three writes are published-API and land with the wave. If the owner grants the wave but not ask E, the panel ships the register/delete capability with a stub list card that says so honestly (state 1 below) — never a fake table.

### 2.3 Wireframe — bots list

```
Tenants · Bots                                                        [ Register bot ]

┌ Bots ──────────────────────────────────────────────────────────────────────────┐
│ NAME            OWNER              SCOPES                 REGISTERED           │
│ ci-deployer     ops@example.com    events:write, read     2 days ago  [Rotate] │
│                                                                              [Delete] │
│ billing-sync   finance@example.com events:read            3 weeks ago [Rotate] │
│                                                                              [Delete] │
└────────────────────────────────────────────────────────────────────────────────┘
```

- `display.Table` inside a `display.Card`, `empty()` for the zero state: "No bots yet — Register a bot to issue a scoped API token for machine callers." (search/filter deliberately out of scope until the list is big enough to need it; `listNote`/`listPager` only if ask E's sketch grows pagination).
- Columns come from the bot read model (`Name`, `OwnerID`, `Scopes`, registration time) — `gated: needs ask E`, which is read-model only and carries NO token material (packet §2 ask E). Token hashes never render.
- Row actions are sm `display.Button` components with `hx-post` + `data-confirm`/`data-confirm-title` + `hx-disabled-elt="this"` (the danger-zone button pattern, `users.templ:416-427`).

### 2.4 Wireframe — register flow with SHOW-ONCE token reveal

Register is a dedicated page (`GET /bots/new` implied by the Register CTA — add the route row; same panel/mode, read posture), because the reveal must not be a lost toast or a swapped-out partial:

```
┌ Register bot ──────────────────────────────────────────────────────┐
│  Name (required)                                                  │
│  ┌──────────────────────────────────────────────┐                 │
│  │ ci-deployer                                  │                 │
│  └──────────────────────────────────────────────┘                 │
│  Owner (required)                                                 │
│  [ user/email select — the memberEmailOptions pattern ]           │
│  Scopes                                                           │
│  ( ) events:write   ( ) events:read   ( ) …  ← checkbox group     │
│                                     [ Register bot ] [ Cancel ]   │
└────────────────────────────────────────────────────────────────────┘
        │ POST /bots (published RegisterBot)
        ▼  full-navigation render of the ONE-TIME response
┌ Token created — copy it now ───────────────────────────────────────┐
│                                                                    │
│  This token is shown ONCE. The panel stores only a hash; neither   │
│  the panel nor the service can show it again.                      │
│                                                                    │
│  ┌────────────────────────────────────────────────┐  [ Copy ]     │
│  │ bt_9f3e…  (full 256-bit token, mono, readonly) │               │
│  └────────────────────────────────────────────────┘               │
│                                                                    │
│              [ I stored it — hide forever ]                        │
└────────────────────────────────────────────────────────────────────┘
```

- This matches the Service contract exactly: `RegisterBotResult{Bot, Token}` — "The raw token is returned in the result and NEVER persisted or shown again" (`usermgmt/service_bot.go:20-31`). The reveal is a server-rendered response page, not client state: re-render, reload, or back-navigation shows the bot row, never the token.
- `Copy` is `display.CopyButton` (the `copyableID` component carries exactly this affordance today).
- "I stored it — hide forever" is the ONLY deliberate dismissal (navigates to `/bots`). Leaving any other way loses the token forever — the card says so, and that is honest, because the service already behaves that way.
- Success toast AFTER dismissal: `ok` "Bot ci-deployer registered".

### 2.5 Interaction contract

**States**
1. *Wave not granted / ask E missing* → `/bots` renders a `display.Card` with honest copy: "Bot listing needs usermgmt API ask E (bot enumeration) — the panel cannot show registered bots yet. Registration and deletion work once the M2M owner decisions row ticks." Register/delete CTAs render per the M2M row, not per ask E.
2. *Pepper not configured*: `RegisterBot` rejects with `usermgmt.bot.pepper_not_configured` (`service_bot.go:36-40`) — the panel probes `ServiceConfig.TokenPepper` presence like T01 probes WebAuthn/OAuth2, and when absent hides the Register CTA and renders: "Token signing is not configured on this deployment (TokenPepper). Bots cannot be registered here." Dead surface never renders.
3. *List populated* → table as wireframed; Rotate/Delete per row.
4. *Reveal pending* → the one-time card; all other navigation loses the token (disclosed on the card).

**Refusals**
- Missing name / owner: client `required` + Service rejections `usermgmt.bot.name_required` / `usermgmt.bot.id_required` → toast `err`.
- Registration refused by decider (duplicate identity etc.): `writeActionError` toast with the Service message.
- Pepper unconfigured: see state 2 — pre-empted, plus the toast if raced.

**Confirms (the escalation language names blast radius)**
- *Rotate token*: "Rotate the token for **ci-deployer**? This registers a replacement token and deletes this bot. Every caller using the current token stops working the moment the old bot is deleted — there is no grace period. Move consumers to the new token before confirming." Title: "Rotate bot token".
- *Delete*: "Delete bot **ci-deployer**? It has N scopes and its token authenticates its callers today. Every caller is refused IMMEDIATELY and the bot cannot be restored." Title: "Delete bot". This is the escalation confirm required by the task: name, scope count, owner, immediate-effect sentence.
- *Honest rotate framing*: **no RotateBot API exists and none is sketched in the packet** — the rotate flow is a composition of published `RegisterBot` (new bot + fresh token) → reveal card → `DeleteBot` (old bot). The confirm copy above says exactly that in user terms. A scoped rotate with token history would be a NEW upstream ask; it is out of scope here and must not be implied by the UI.
- *Audit honesty on delete*: bot lifecycle events are NOT in the panel audit vocabulary today — the audited set is `allUserEventTypes + allAuthEventTypes` (packet §2 ask H evidence), and bots are named in neither ask H option. The delete confirmation carries one disclosure line: "Bot lifecycle events are not in the audit log yet." The reason travels on the `DeleteBot` command only.

**Empty states:** zero bots → `empty()` as wireframed; ask-E-missing → state 1 card.

### 2.6 a11y notes

- Reveal card: on render, focus moves to the token `<input readonly>` (focusable by default) so keyboard users can Ctrl/Cmd-A + copy without tabbing; heading is `tabindex="-1"` for context; the Copy button's accessible name is "Copy API token".
- "I stored it" is a real `<button>` with explicit text — the state transition (token permanently hidden) is stated in the label, not an icon.
- Row actions: `aria-label="Rotate token for ci-deployer"` / `"Delete bot ci-deployer"` (table-cell buttons otherwise read as bare "Rotate"/"Delete" three times over).
- Scopes checkbox group reuses the `roleCheckboxGroup` pattern: `role="group"` + `aria-label` (`components.templ:53-64`).
- Register page: visible labels + `required` on name and owner.

### 2.7 Data needs

| Need | Source | Gate |
| --- | --- | --- |
| Bot list | packet ask E sketch: `Bots(ctx) ([]*Bot, error)` (optionally `BotsForOwner`); scope the ask as "export `FindByOwner` + add read-model `All`" per the packet's partial-coverage note | `gated: needs ask E` (+ D2/D3 + M2M row for the wave) |
| Register | `RegisterBot(ctx, RegisterBotRequest) (*RegisterBotResult, error)` — published; requires `ServiceConfig.TokenPepper` | published — wave-gated: `needs D2 + D3 + M2M row` |
| Delete | `DeleteBot(ctx, botID, reason) error` — published | published — wave-gated: `needs D2 + D3 + M2M row` |
| Rotate | composed Register + Delete (§2.5); NOT a packet ask | published — wave-gated, same as above |
| Audit visibility for bots | not covered by ask H option (a)/(b) as sketched | out of scope — disclose honestly in copy |

### 2.8 Build order (what lands first on the gates opening)

1. Panel + nav + route rows + manifest golden (skeleton; honest stub list per state 1).
2. Register flow + show-once reveal + pepper probe (published API — first real capability).
3. Delete with escalation confirm (published API).
4. List table — the last piece, `gated: needs ask E` landing in the same tag train.
5. Rotate flow last (composition; pure UI over 2+3).

---

## Spec 3 — F223: Sessions list, revoke, MFA bypass, lockout badge

**Gate summary:** sessions card `gated: needs ask B`; revoke-all `gated: needs ask A`; MFA bypass `gated: needs ask C`; lockout badge `gated: needs ask D`. Each piece renders only when its ask lands (D3 puts this wave FIRST, ahead of bots).

### 3.1 Routes (routeSpec rows)

| Method | Pattern | Panel | Mode | Write | `why` |
| --- | --- | --- | --- | --- | --- |
| GET | `/users/{id}/sessions` | Users | SuperAdmin | no | "enumerate a user's sessions for the detail card; no enumeration exists until ask B" |
| POST | `/users/{id}/sessions/revoke-all` | Users | SuperAdmin | yes | "sign the user out everywhere; thin surface over the published DeleteByUserID store path (ask A)" |
| POST | `/users/{id}/sessions/revoke` | Users | SuperAdmin | yes | "revoke ONE session by token prefix; needs a single-session Service method not sketched in the packet — raise alongside ask A" |
| POST | `/users/{id}/totp/disable` | Users | SuperAdmin | yes | "admin TOTP bypass with mandatory reason + audit event (ask C)" |

Lockout badge adds NO route — ask D is a read (`LockoutStatus`) consumed while rendering `userDetail`. The GET sessions route may instead be an embedded render-time call like other cards; the route exists so the card can refresh as a targeted partial (`hx-get`) without reloading the page — same pattern as `/partials/stats`.

### 3.2 Wireframe — sessions card on user detail

Placed after `tenantAccessCard`, before the timeline card:

```
┌ Sessions ────────────────────────────────────────────────────────┐
│                                                                    │
│  ORIGIN                          CREATED      EXPIRES     TOKEN    │
│  (●) direct sign-in              2 h ago      tomorrow    9f3e…   │
│  (●) impersonation by ops@…      1 h ago      in 59 min   b712…   │
│        "ticket 4821 — reported passkey loop"                       │
│                                                                    │
│  Signed-in sessions are the only sign-in state. Revoking all      │
│  signs the user out of every device immediately.                 │
│                                                                    │
│  Reason (required)                                                 │
│  ┌──────────────────────────────────────────────┐                 │
│  └──────────────────────────────────────────────┘                 │
│                          [ Sign out everywhere ] (danger, sm)     │
└────────────────────────────────────────────────────────────────────┘
```

- **Device column honesty:** the task brief asks for device/origin/created/expires. The `Session` model carries `Token, UserID, ActorID, Origin, CreatedAt, ExpiresAt` (`identity-model/session.go:16-22`) and ask B's sketch returns **origin kind (incl. impersonation origin + actor)** — there is NO user-agent/device field in the model or the ask. The spec renders the ORIGIN column as the honest substitute ("direct sign-in" / "impersonation by …") and does NOT invent a device column. If the owner wants device labels, the ask-B conversation must extend `SessionView` with a UA-captured field — flagged there, never assumed here.
- **Token column:** ask B's redaction rule — token PREFIX only, never the raw token (packet ask B sketch). Rendered as mono text; `copyableID` is pointless for a redacted prefix and is not used.
- **Impersonation-origin rows** (ask B: origin carries the actor) render the reason as the second line — the session-origin disclosure Spec 1 promised.
- **Expires:** `display.RelativeTime` for both created/expires; an expiry inside the next hour gets a `statusBadge("expires soon", BadgeWarning)` next to the time (a state → dot).
- **Revoke-all** is the ask-A call: `RevokeAllSessions(ctx, adminID, userID, reason)` — super-admin-gated, non-empty reason (packet ask A sketch). Hence the inline reason field + the shared confirm modal on submit (`data-confirm`: "Sign **ada@example.com** out of EVERYWHERE? N active session(s) end immediately and they must sign in again. The reason is recorded."). The store path is already published and journaled — one `SessionEnded` batch entry (`usermgmt/session_audit.go:100`, called on the delete cascade at `service_register.go:173`) — so the audit chip for the batch is ONE "Session ended" entry, and the toast after success is `ok` "Signed out everywhere (N sessions)".
- **Revoke-one (per-row)** is `gated: needs an ask-A extension`. The packet sketches only `RevokeAllSessions`; `SessionStore.Delete` exists but no Service surface exposes single-session revocation (packet ask B "Missing" note covers the interface). Until the owner extends ask A with e.g. `RevokeSession(ctx, adminID, token, reason)`, the per-row action **does not render** — a dead button over an unlanded API is the exact lie T01 removed. The card footer may carry the honest sentence: "Individual session revocation needs a Service method — tracked with ask A."
- **Empty state:** zero sessions → `empty("session", "No active sessions", "The user is signed out everywhere.", "", "")` — true after a revoke-all, and for never-signed-in users.
- **Impersonation banner interplay:** while the ADMIN is impersonating (Spec 1) and views another user's sessions card, revoke-all still targets that user — the banner already discloses the acting-as state; no extra copy.

### 3.3 Wireframe — MFA bypass (ask C)

Added to the user-detail security area, next to where `totpBadge` renders (badges cluster in the table rows stays as T01 shipped it — `statusBadge("MFA", BadgeInfo)` only when enabled, `users.templ:244-246`):

```
┌ Two-factor authentication ────────────────────────────────────────┐
│  (●) TOTP enabled since 3 Nov 2026                                 │
│                                                                    │
│  Admin bypass removes this user's TOTP without their code. It     │
│  exists for locked-out legitimate owners after enrollment-link    │
│  rescue failed. The action writes an audited entry naming YOU,    │
│  the target, and the reason — permanently.                        │
│  (Supervised enrollment-link rescue is the first-line path; see   │
│   docs/decisions/2026-11-24_adminui-supervised-enrollment-note.)  │
│                                                                    │
│  Reason (required)                                                 │
│  ┌──────────────────────────────────────────────┐                 │
│  └──────────────────────────────────────────────┘                 │
│                [ Disable TOTP for ada@example.com ] (danger, sm)  │
└────────────────────────────────────────────────────────────────────┘
```

- Card renders ONLY when `d.User.TOTPEnabled` — a bypass control on a user without TOTP is a dead surface.
- The bypass consumes ask C's sketch: `AdminDisableTOTP(ctx, adminID, userID, reason)` — super-admin-gated, reason MANDATORY, dispatching the existing disable path and emitting a new audited `AdminTOTPDisabled`-class event naming admin, target, reason (packet ask C). The ONLY disable path today requires the user's own valid code (`usermgmt/totp.go:117-156`, "prevent MFA stripping via session hijack") — this card is the deliberate, audited exception the owner is being asked to authorize.
- Confirm (`data-confirm` on the submit): "Disable TOTP for **ada@example.com** WITHOUT their code? They keep any remaining sign-in methods (passkeys, linked provider). An audited entry names you and this reason, permanently." Title: "Admin TOTP bypass".
- Refusals: empty reason → client `required` + Service rejection → toast `err` "A reason is mandatory for an admin TOTP bypass."; non-super-admin → route posture. Post-success: toast `ok` "TOTP disabled for ada@example.com — audited."; the badges cluster flips to the neutral "disabled" state AND the user-detail timeline gains the new action's audit chip (label row to add when ask C's vocabulary lands: `admin_totp_disabled` → "MFA disabled by admin" — the existing `totp_disabled` label "MFA disabled" stays for self-service; the distinction is the disclosure).
- Honest limit: the panel cannot distinguish post-hoc "why" beyond the audit entry — the badge after bypass reads plain "disabled"; the AUDIT chip carries who/why. That asymmetry is deliberate and disclosed here.

### 3.4 Wireframe — lockout badge treatment (ask D)

Consistent with T01's honest auth-state surface: same amber banner family as "Cannot sign in" (`users.templ:360-368`), rendered ABOVE it when both apply (a locked user usually also cannot sign in — the banners stack, lockout first as the more immediate cause):

```
┌────────────────────────────────────────────────────────────────────┐
│ (● locked)  Sign-ins are locked until 14:05 UTC (25 min).          │
│ 7 of 10 allowed failed attempts used. Further failures extend     │
│ the window. Successful sign-in resets immediately.                │
└────────────────────────────────────────────────────────────────────┘
```

- Data: ask D sketch `LockoutStatus{Locked, LockedUntil, FailedAttempts, MaxAttempts}` (`gated: needs ask D`). All four fields render; `LockedUntil` as absolute time + relative parenthetical; `FailedAttempts/MaxAttempts` as the fraction. No data the sketch does not return — no "device", no IP (the lockout machinery is attempt-count based, packet ask D evidence).
- Badges cluster (users list rows and detail): `statusBadge("locked", display.BadgeError)` (state → dot) alongside verified/MFA. **Detail page only** until ask D grows a batched query: rendering it per directory row is one `LockoutStatus` call per user — the exact N+1 ask G exists to kill, so the spec refuses it for the list. If the owner wants directory badges, that is an explicit ask-D extension (batched status for a page of users).
- **No "clear lockout" button.** Ask D's sketch is introspection-only; the lockout store has `Reset` internally but no Service surface and no owner decision on admin unlock. A dead or presumptuous button is worse than absence — the banner copy explains the reset condition instead. If the owner extends ask D with an admin unlock, it slots into this banner as a reason-bearing action (same pattern as revoke-all) — noted, not designed, until it is an ask.
- Service-truth note for implementers (not UI copy): the default `AccountLockout` is in-memory and lost on restart (packet ask D evidence). Render ONLY what `LockoutStatus` returns at render time; never cache lockout state client-side — the honest dash wins over a stale "locked".
- Empty/zero state: not locked → no banner, no badge; the "Cannot sign in" banner (T01) covers the no-credentials case. The two banners are independent truths, not merged.

### 3.5 Interaction contract (sessions card, consolidated)

**States:** ask B missing → card does not render (honest stub sentence only if the wave ships ahead of the tag train: "Session listing needs usermgmt ask B."); populated → table + revoke-all; zero → `empty()`; post-revoke-all → table swaps to `empty()` via targeted htmx swap (`hx-target="#sessions-card"`), toast `ok`.

**Refusals:** empty reason (revoke-all or bypass) → client `required` + server rejection toast; target deleted mid-view → `writeErrorPage` "User not found"; wrong-mode caller → route posture 403.

**a11y:** table headers real `<th>`s (`display.Table`); per-row danger buttons carry `aria-label` with the user's email; reason fields visible-label + `required` + HelpText "Recorded in the audit log."; banner is `role="status" aria-live="polite"`; the "expires soon" badge is text, not color-only; focus returns to the sessions card heading after a targeted revoke swap (`hx-swap-oob` not needed — set `tabindex="-1"` on the card title and let the swap target preserve it).

### 3.6 Data needs

| Need | Source | Gate |
| --- | --- | --- |
| Session rows | packet ask B sketch: `SessionsForUser(ctx, userID) ([]SessionView, error)` — prefix, created/expires, origin kind + impersonation actor; new store query + redaction rule | `gated: needs ask B` |
| Revoke-all | packet ask A sketch: `RevokeAllSessions(ctx, adminID, userID, reason)` — thin Service surface over the published, journaled `DeleteByUserID` | `gated: needs ask A` |
| Revoke-one | NOT sketched in the packet — raise as ask-A extension during filing | `gated: needs ask A extension` (button hidden until then) |
| TOTP bypass | packet ask C sketch: `AdminDisableTOTP(ctx, adminID, userID, reason)` + audited event; new API unavoidable (only self-code path today) | `gated: needs ask C` |
| Lockout badge | packet ask D sketch: `LockoutStatus` — the largest design surface (interface extension vs event-sourced is the owner's call) | `gated: needs ask D` |
| Audit labels | `session_ended` exists today; `admin_totp_disabled`-class label lands with ask C's vocabulary | panel-side, with ask C |

### 3.7 Build order

1. Ask A lands → revoke-all card + reason form + confirm (thinnest ask, biggest rescue value; the packet calls it the cheapest of the five).
2. Ask B lands → sessions table + origin disclosure + impersonation rows (Spec 1's disclosure completes here).
3. Ask C lands → MFA bypass card + audit label.
4. Ask D lands → lockout banner + detail badge (last; largest design choice).

---

## Spec 4 — F224: Tenant rename, batched member counts, tenant-attribution

**Gate summary:** rename `gated: needs ask F`; batched counts `gated: needs ask G`; attribution `gated: needs ask H + the owner's option (a)/(b) pick`.

### 4.1 Routes (routeSpec rows)

| Method | Pattern | Panel | Mode | Write | `why` |
| --- | --- | --- | --- | --- | --- |
| POST | `/tenants/{id}/rename` | Tenants | SuperAdmin | yes | "tenant display-name change; wires the existing identity-model RenameTenant vocabulary (ask F) — three glue points are the whole ask" |

Batched counts and attribution add NO write route: counts replace the handler's count source; attribution is read-side (filter parameter + a read card).

### 4.2 Rename (ask F)

Packet evidence: the vocabulary is COMPLETE in identity-model — `RenameTenantCmd`, `EventTenantRenamed`, fold branch, posture row, event catalog (`identity-model/commands.go:377-390`, `fold.go:386-387`, `command_postures.go:88-89`) — the ask is three usermgmt glue points (decider branch, dispatch registration, Service method). The packet calls it the cheapest ask after A.

Wireframe — inline form inside the tenant-detail "Details" card (the `profileEditCard` precedent; `DefinitionList` keeps the read-only rows):

```
┌ Details ──────────────────────────────────────────────────────────┐
│  Slug          acme                                                │
│  Members       12                                                  │
│  Status        (●) active                                          │
│  Created       2 Sep 2026                                          │
│  ─────────────────────────────────────────────────────────────    │
│  Display name                                                      │
│  ┌──────────────────────────────────────────────┐                 │
│  │ Acme Robotics                                │                 │
│  └──────────────────────────────────────────────┘                 │
│  [ Save display name ]                                             │
└────────────────────────────────────────────────────────────────────┘
```

- Contract: `hx-post` to `/tenants/{id}/rename`, `hx-target="this"` targeted swap of the Details card; `data-confirm`: "Rename this tenant? The display name changes everywhere it renders — the slug **acme** and all membership links stay." Title: "Rename tenant".
- Refusals: blank/whitespace name → client `required` + server rejection toast; unchanged name → success no-op toast `info` "Display name unchanged." (Service-side identical-value handling is ask-F glue detail; the panel treats a no-diff POST as an early return before dispatching).
- Post-success: toast `ok` "Tenant renamed."; the tenants LIST row (name + slug two-liner, `tenants.templ:136-143`) reflects it on next render.
- **Audit honesty (copy on the confirm):** `EventTenantRenamed` exists on the tenant stream, but tenant events are NOT in the panel audit vocabulary today (audited set is user/session-only — packet ask H). The confirm carries: "This change is recorded on the tenant's stream; it will not appear in the audit log until tenant events are audited (ask H)." If ask H option (b) lands, the new `TenantRenamed` audit rows get an `auditLabels` entry ("Tenant renamed") and the caveat disappears — the confirm copy is conditional on H's absence.

### 4.3 Batched member counts (ask G)

- **Zero visual change by design.** The tenants-list "Members" column (`tenants.templ:110`) keeps its exact rendering; the tenants detail "Members" definition row keeps its count.
- The CHANGE is the data source: today every rendered count is one `TenantMembers` call per tenant — the structural N+1 (`adminui/handler_tenants.go:27`, packet ask G "Exists/Missing"). With ask G, the handler makes ONE `TenantMemberCounts(ctx, visibleTenantIDs)` call per render (the sketch's own contract: "the tenants page calls it once per render with the visible tenant set") and feeds `MemberCount` from the returned map.
- The page-window scoping matters: `listNote`/`listPager` already bound the visible set, so the batch is page-sized, not table-sized.
- Honest edge: a tenant whose membership stream changed between `AllTenants` and the counts call may show a count off by one until next render — same eventual-consistency the per-tenant call had; no new lie.
- Contract with the T30 render benchmark: when ask G lands, the tenants-list bench (if added) should show the per-render query count drop from N+1 to 2 (`AllTenants` + `TenantMemberCounts`). That is the verify line for the ask; the UI never changes.

### 4.4 Tenant attribution (ask H) — UI for BOTH owner options

The packet leaves option (a) widen `AuditEntry` with `TenantID` + `EntriesForTenant` vs option (b) widen the audited event set as the owner's call. The UI consequences differ materially, so both are speced:

**Option (a) — tenant-filterable audit (the fuller surface):**
- Tenant detail gains an "Activity" card above the members card: `display.Table` reusing the audit page's exact row rendering (`auditLabel` + `auditActionBadge` + `RelativeTime`), fed by `EntriesForTenant(tenantID, limit)`. Empty state: `empty("audit", "No audited activity for this tenant", "Tenant-attributed entries appear here once members or lifecycle actions are audited.", "", "")` — the honest dash, never a fabricated window.
- The global audit page gains a tenant filter chip row beside the existing search (the tenants page lifecycle-chip pattern, `tenants.templ:60-70`): chips render the tenants present in the current window, `?tenant=<id>` is a shareable URL state like the audit search today.
- Membership entries become attributable: today they aggregate on membership streams with only an implicit tenant (packet ask H) — option (a)'s `TenantID` makes "show this tenant's membership changes" a real filter, which is the T104 unblocking the packet names.

**Option (b) — event-set widening only:**
- Tenant lifecycle actions (`CreateTenant`/`SuspendTenant`/`DeleteTenant`/`RenameTenant`) appear in the GLOBAL audit with the tenant named in `Detail`; `auditLabels` grows the matching rows ("Tenant created", …) and `auditActionBadge` maps destructive tenant actions (`TenantDeleted`) to `BadgeError`.
- **The tenant-detail Activity card does NOT land under (b) alone.** There is no `TenantID` field, so a per-tenant view would be a Detail-text scrape — the packet records that exact shortcut (T104) as "unimplementable … reverted rather than shipped as a dead-end link". Spec honors the precedent: (b) without (a) means global rows only, no tenant card, no tenant chip.
- Owner may grant both (a)+(b): lifecycle rows then ALSO carry `TenantID`, so they appear both globally and in the tenant card/chip — the (a) surface simply includes them.

### 4.5 Interaction contract (spec 4, consolidated)

- **States:** rename form always present on super-admin tenant detail (posture-gated); counts column identical pre/post ask G (only the handler changes); attribution card/chips render only under option (a).
- **Refusals:** rename rejected by decider (unknown tenant, suspended-state rules if any — glue detail of ask F) → `writeActionError` toast; no other new refusals.
- **Confirms:** only rename (§4.2 copy with the ask-H caveat).
- **Empty states:** attribution card under (a) as §4.4; the tenants list empty states are already shipped.
- **a11y:** rename input visible-label "Display name" + HelpText "Shown across the panel; the slug stays."; tenant chips are real links with `aria-current="true"` on the active chip (match the lifecycle-chip treatment); the Activity card is the same audit table markup — no new table a11y surface.

### 4.6 Data needs

| Need | Source | Gate |
| --- | --- | --- |
| Rename | packet ask F sketch: `RenameTenant(ctx, tenantID, displayName)` dispatching `NewRenameTenantCmd` — vocabulary complete, 3 glue points missing | `gated: needs ask F` |
| Batched counts | packet ask G sketch: `TenantMemberCounts(ctx, tenantIDs) (map[TenantID]int, error)` — additive read-model query | `gated: needs ask G` |
| Attribution (a) | packet ask H option (a): `AuditEntry.TenantID` (omitzero) + `EntriesForTenant` | `gated: needs ask H + option (a)` |
| Attribution (b) | packet ask H option (b): widen the audited event set to tenant lifecycle actions | `gated: needs ask H + option (b)`; tenant card needs (a) regardless |
| Count column rendering | existing `MemberCount` field/row — unchanged | none |

### 4.7 Build order

1. Ask F lands → rename form (cheapest new capability; pure wiring per the packet).
2. Ask G lands → handler swap to `TenantMemberCounts` (invisible to users, visible to the benchmark).
3. Ask H decision + option pick → (b) labels/badge rows if chosen; (a) Activity card + audit tenant chips if chosen; both if granted.

---

## 5. Links

**Ground-truth packet:** [`docs/decisions/2026-11-24_adminui-d1-d3-and-upstream-asks.md`](2026-11-24_adminui-d1-d3-and-upstream-asks.md) (T02, F011–F016). This file is the PANEL HALF of that packet: the packet holds the decision + the ask sketches and their verified evidence; this file holds the wireframes, interaction contracts, and build orders for the surfaces those decisions unblock. Neither implements anything; both are inert until the owner ticks D1/D2/D3 and the per-ask approvals. The packet links back to this file (one pointer line at the end).

**Which spec consumes which packet item:**

| Packet item | Spec section | Surface |
| --- | --- | --- |
| D1 (impersonation GO, opt-in knob; no API ask) | Spec 1 | impersonate card, banner, exit, audit chips, origin disclosure |
| Ask A — `RevokeAllSessions` | Spec 3 §3.2 | revoke-all + reason (cheapest ask; leads D3 wave 1) |
| Ask A extension — single-session revoke (NOT sketched; raise when filing A) | Spec 3 §3.2 | per-row revoke (hidden until then) |
| Ask B — `SessionsForUser` | Spec 3 §3.2 (+ Spec 1 §1.7 disclosure row) | sessions table, origin/impersonation rows, token prefix |
| Ask C — `AdminDisableTOTP` | Spec 3 §3.3 | MFA bypass card, mandatory reason, audit disclosure |
| Ask D — `LockoutStatus` | Spec 3 §3.4 | lockout banner + detail badge (no unlock button — not an ask) |
| Ask E — bot enumeration ("export FindByOwner + add All") | Spec 2 §2.3 | bots list |
| Ask F — `RenameTenant` (wire existing vocabulary) | Spec 4 §4.2 | tenant display-name form |
| Ask G — `TenantMemberCounts` | Spec 4 §4.3 | zero-UI handler swap for the Members column |
| Ask H — tenant attribution, options (a)/(b) | Spec 4 §4.4 | tenant Activity card + audit tenant chips ((a) only), global lifecycle rows + labels ((b)) |
| D2/D3 + M2M owner-decisions row (bots wave ordering + second gate) | Spec 2 (gate summary) | whole bots panel |
| Packet ask H evidence (audited set is user/session-only) | Spec 2 §2.5 + Spec 4 §4.2 | bot/rename audit-honesty copy |

**Cross-wave build order (D3's answer, made concrete):** wave 1 = Spec 1 (D1-only, no API) + Spec 3 pieces as asks A→B→C→D land; wave 2 = Spec 2 behind the M2M row with the list last (ask E); Spec 4 pieces ride whichever tag train carries F/G/H. Plan-row cross-references: item 6 (sign out everywhere) → Spec 3 §3.2; item 42 (rename) → Spec 4 §4.2; item 43 (tenant attribution) → Spec 4 §4.4; item 74 (self-edit re-auth) → NOT covered by these four specs — it needs a session API the packet does not sketch (its plan row already says "needs session API — ask"); item 75 (privilege audit chip) → Spec 1 §1.5; item 77 (bot show-once contract) → Spec 2 §2.4.
