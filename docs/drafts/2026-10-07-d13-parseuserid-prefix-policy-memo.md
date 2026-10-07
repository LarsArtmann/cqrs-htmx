# D13 Decision Memo — `ParseUserID` brand-prefix strip: generic vs whitelist

**Status:** DECISION INPUT for the owner (TODO_LIST D13) · 2026-10-07
**Subject:** `identity-model/id.go` `strippedBrandPrefix` (added `eba45c80`) — `ParseUserID` drops everything before the FIRST colon before the strict ULID parse.
**Evidence pins:** `identity-model/model_test.go` `TestUserID_ParseBrandPrefix_Policy` (6 subtests, green — added this memo's session; before it the strip had ZERO test coverage).

## The two policies

**A. Generic strip (CURRENT).** `x:<ulid>` → `<ulid>` for ANY `x`.

- For: format-proof for every future branded-id brand (the go-cqrs-lite `id` train brands freely: `StreamMarker:`, and the same law applies to any marker). No new brand ever requires an identity-model release. Matches the upstream `ParseStreamID` round-trip law (the prefix carries no identity).
- Against: a security smell — a hostile or buggy caller can smuggle `AnythingWeNeverApproved:<ulid>` and it parses. The brand name is security theater if any brand is accepted. Empty brand (`:<ulid>`) also parses (documented quirk pin).
- Blast radius of a hostile input: LOW. The result is still a strictly-valid ULID that must ALSO satisfy every downstream ownership check (session/tenant/membership lookups). The strip cannot forge an identity that wasn't already a valid ULID; it only tolerates decorated spellings of one. The attack class "SHA-256-hashed wrong-but-valid user" (gotcha 16) lives in `NewUserID`/`SyntheticUserID`, NOT here — `ParseUserID` never hashes.

**B. Known-brand whitelist.** Only `StreamMarker:` (and explicitly added brands) is stripped.

- For: honest strictness; unknown decorations become Rejections, surfacing caller bugs early.
- Against: every upstream brand addition breaks parsing of decorated ids until identity-model adds the brand (a cross-repo release-train coupling for a cosmetic prefix); the failure mode is a confusing Rejection on a legitimate id spelling. Also: the strip's PURPOSE is future-proofing against the in-flight id train — a whitelist partially defeats it.

## Recommendation

**Keep A (generic), keep the pins.** The strip only normalizes decoration on an id that must still parse as a strict ULID; no additional identity is reachable through an unapproved brand name. The real hardening target is the deprecated `NewUserID` hash path (gotcha 16), not this tolerance. Revisit only if a surface emerges where brand names carry authorization meaning.

**Owner call requested:** confirm A (then strike TODO_LIST D13 citing this memo + the pins), or request B (then the pins' `unknown brand`/`leading colon` subtests flip to rejection expectations).
