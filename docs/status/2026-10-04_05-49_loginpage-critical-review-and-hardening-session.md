# loginpage Critical Review & Hardening — Session Status

**Date:** 2026-10-04 05:49 CEST · **Scope:** loginpage module critical review (single-session snapshot)
**Session type:** module-scoped review + fix + verify · **Reports in series:** `docs/reviews/2026-10-04_05-45_loginpage-critical-review.html`

> **Format note:** status-report skill is HTML-canonical; this report is `.md` per the
> explicit path demand in the dispatch prompt. One-off override, not propagated.

---

## a) FULLY DONE

| # | Work | Evidence |
| - | ---- | -------- |
| 1 | Full file-by-file review of all 17 loginpage files (handler, config, util, page.templ, generated _templ.go, CSS, JS, 43 pre-existing tests, README, doc.go, CHANGELOG, go.mod, .golangci.yml) | this session |
| 2 | Prior-art check before reviewing (gotcha 21): read `docs/research/2026-09-23_loginpage-templ-components-audit.md`, continued its series instead of starting blind | session discovery step |
| 3 | Contract verification loginpage ↔ usermgmt: all 5 endpoint URLs, `session_key` → `?user_id` finish params, `credential_name`, `RegisterResponse{user.id,session}` — every URL/param the inline JS builds matches the server routes | usermgmt/http.go:201-204, webauthn_http.go, webauthn_service.go:71 |
| 4 | **CRITICAL fix — script breakout**: `encoding/json/v2` does NOT HTML-escape (verified empirically with a live repro), so `Config.CredentialName`/`AuthPrefix` containing `</script>` could terminate the `application/json` script block rendered via `templ.Raw`. Fixed with `scriptSafeJSON` (`\u003c`-style escaping, safe because `<>&` only occur inside JSON string literals) | loginpage/handler.go:158-166, applied at :205 |
| 5 | **HIGH fix — `<style>` injection**: `AccentColor` was embedded unescaped in the raw `<style>` block (page.templ:16) and the SVG favicon; `validateAccentColor` now rejects `< > & " ' \` backtick at construction (rgb()/color-mix()/var() still accepted — "any CSS color" contract preserved) | loginpage/util.go:21, wired in config.go:163 |
| 6 | **HIGH fix — executable CSSPath schemes**: `javascript:alert(1)` rendered as a clickable href (templ escapes attributes, does not sanitize schemes); `validateStylesheetURL` now accepts only root-relative or absolute http(s) URLs | loginpage/util.go:33, wired in config.go:166 |
| 7 | **DOC split-brain fix**: `Config.OAuth2Buttons` field doc claimed "Empty slice hides all OAuth2 buttons" — behavior was (and is) auto-detect on empty. Comment now describes reality | loginpage/config.go |
| 8 | 6 regression tests added (49 total in handler_test.go): malicious accent ×5 payloads, 5 valid color grammars, 3 malicious CSSPaths, 4 valid paths, configJSON escaping assertion, rendered-body injection assertion | loginpage/handler_test.go tail |
| 9 | Verification battery: `go test -count=1` green, `-race` green, `go vet` clean, `gofmt` clean, per-module `golangci-lint run` 0 issues, `GOWORK=off go build` OK, scoped `nix run .#fmt` 0 changes | session verification steps |
| 10 | loginpage CHANGELOG `[Unreleased]` Fixed section with per-fix rationale and honest framing (consumer-controlled config, defense-in-depth) | loginpage/CHANGELOG.md |
| 11 | HTML review report written from the report-kit template (not hand-transcribed): `docs/reviews/2026-10-04_05-45_loginpage-critical-review.html` — stat cards, 4 issue cards, strengths, issue table, verified-claims table | docs/reviews/ |
| 12 | TODO_LIST follow-up item filed next to the existing adoption owner-call | TODO_LIST.md:52-53 |
| 13 | Coverage-safe conclusion on templ-components adoption: confirmed the 2026-09-23 "keep hand-rolled" verdict was still standing at review time; did not re-litigate it | audit doc §21.3 |

## b) PARTIALLY DONE

1. **Defense-in-depth hardening is partial by design**: AccentColor uses a charset blacklist, not a positive color grammar (deliberate — keeps `var()`/`color-mix()` working); the favicon data-URI still emits non-ASCII brand initials un-percent-encoded (works in browsers, not strictly URL-clean).
2. **My 05:45 review report is already partially stale**: row 6 ("CSP-nonce — Open") was implemented by the concurrent session within ~20 minutes of being filed; the "zero-dependency posture" strength is now obsolete (templ-components references are in page_templ.go at HEAD). Needs an inline ANNOTATE pass, not a rewrite.
3. **Review scope**: loginpage-in-isolation was complete, but I did not review the setup→loginpage mounting contract (setup mounts all three panels) — out of the requested scope, noting as the natural next ring.
4. **The hardening's consumer impact is documented but not yet release-bundled**: config values that previously rendered (weirdly) now fail construction — a behavior change sitting in `[Unreleased]`, awaiting a train/patch decision.

## c) NOT STARTED

1. **`NoOAuth2`-style force-hide option** — the only surviving filed follow-up (b was done by the sibling session); currently impossible to hide auto-detected providers.
2. **Release bundling** of the hardening fixes (PUBLISHED module code touched → per gotcha 8, bundles with the next train or an explicit patch cut; owner decision).
3. **loginpage JS testing** — login.js (~296 lines of WebAuthn ceremony logic) has zero automated tests; contract was verified by reading, not execution.
4. **E2E round-trip** — no test drives a real WebAuthn ceremony through loginpage's rendered page against a usermgmt service.
5. **Coverage-gate re-run for loginpage post-adoption** — the sibling session's templ-components adoption changes the render path; the module's coverage gate hasn't been re-validated this session.
6. **Research-file annotation** — the 2026-09-23 audit's verdict ("keep hand-rolled") is now factually overturned at HEAD; the series-hygiene outcome-note hasn't been written yet.
7. TOTP second-factor UI (README-documented future work, untouched, unrelated to this session's findings).

## d) TOTALLY FUCKED UP (own failures, honestly)

1. **Three consecutive edit-tool blunders** (trailing-newline mismatch class): two no-op/malformed edits on handler.go that briefly merged adjacent lines, and one on handler_test.go that merged a function signature with its body (`func TestRenderPage_RenderError(t *testing.T) {	data := ...`). All caught by immediate re-view before any test run, but each was a wasted round trip and the signature merge would have shipped as a syntax error had I batched edits without re-reading.
2. **One fabricated-format edit attempt** (multiedit param type error) before landing the sanitizer — pure tool-handling noise, zero impact, but it validates the "re-view after every edit" discipline I nearly skipped.
3. **Unverified statistic in a shipped report**: the 05:45 HTML report claims "40 tests covering rendering states…" — the actual pre-existing count was 43 (49 after my 6). A report about precision containing an uncounted number is exactly the verify-external-claims class this repo polices.
4. **Commingled attribution**: the auto-commit daemon swept the concurrent session's page.templ rewrite into commit 6a773451 together with MY review report + TODO_LIST item. History now shows "my" docs and "their" adoption in one heuristic commit. Not repairable without history surgery (not worth it); noted so future sessions don't misattribute the adoption to this session.
5. **Report-incoherence at write time**: my report praises the zero-dependency differentiator while the tree simultaneously carried an in-flight adoption that destroys it. Point-in-time snapshots are allowed to go stale later — but I noticed the foreign diff DURING the session and shipped the report without a coherence caveat. Fixed only now, by this status report.
6. **Read-but-not-audited files**: `.golangci.yml` (163 lines) was opened for inventory but never critically reviewed; coverage.out existence was noted without checking whether the module's gate actually passes. A "full review" that skips config files is a partial review wearing a full-review name.

## e) WHAT WE SHOULD IMPROVE

1. **Edit discipline**: for multi-hunk Go edits, prefer `lsp_replace_symbol` / whole-block replaces over line-anchored `edit` calls — the three blunders were all the same trailing-newline mistake.
2. **Count before you claim**: test counts, file counts, issue counts in shipped reports must come from `grep -c`, not memory.
3. **Foreign-diff protocol earlier**: on the FIRST sign of a concurrent session (page.templ appearing modified), immediately diff + `git log` instead of proceeding to format/verify — would have surfaced the adoption 15 minutes earlier and avoided the stale-strength report text.
4. **Review checklists should include config/docs**: .golangci.yml, README security claims, and CHANGELOG accuracy belong in the per-file pass, not afterthoughts.
5. **JS deserves at least smoke tests**: the module's riskiest logic (Base64URL round-trip, ceremony serialization) is the only logic with zero test coverage — Go-side tests cannot catch a JS regression.
6. **Percent-encode the favicon data-URI** while keeping the firstRune letter filter — closes the non-ASCII nit without behavior change.

## f) Up to 50 things to get done next (brainstorm — most are ROADMAP fuel; HARVEST routes them)

**Immediately actionable (this repo, small):**
1. Annotate the 05:45 review report inline (CSP-nonce row → Fixed by sibling session; zero-dep strength → superseded by adoption).
2. Annotate `docs/research/2026-09-23_loginpage-templ-components-audit.md` with an outcome-note: verdict overturned at HEAD (gotcha-21 series hygiene).
3. Verify the sibling session's adoption end-state: `nix run .#check-codegen`, `.#check-css-bundles` (+ new `build-loginpage-css` app if the pattern followed adminui/dashboardui), `.#coverage-gate` for loginpage, full `.#test` battery.
4. Audit the adoption for the AGENTS-mandated contract: `BaseProps.ID` pinning of every DOM hook login.js targets (`lp-email`, `lp-login-btn`, `lp-error`, …) — a renamed hook breaks the ceremony silently.
5. README truth-pass post-adoption: "Zero external asset requests", "No Tailwind dependency", "Self-contained" claims — all now false or conditional; rewrite or scope them.
6. AGENTS.md loginpage bullet still says "hand-rolled `lp-*` CSS, no templ-components" — STALE at HEAD; update.
7. CHANGELOG coherence pass: merge my Fixed section with the sibling's NonceFromRequest entry; add the behavior-change callout (AccentColor/CSSPath now fail construction).
8. Decide + implement `NoOAuth2` (the one surviving filed follow-up).
9. Decide release vehicle: next family train vs out-of-train loginpage patch for the injection hardening.
10. Re-run `nix run .#erraudit-inventory` + `check-cqrs-lint` scoped to loginpage post-adoption (new sibling code unaudited by the gates this session).
11. Update the `Config` table in README (NonceFromRequest row exists; check AccentColor/CSSPath rows now document validation).

**Test-depth (high value, medium effort):**
12. JS smoke tests for login.js (Base64URL round-trip property test; serializeAssertion/serializeAttestation shape goldens).
13. Playwright E2E: full WebAuthn ceremony through the rendered page (repo already carries Playwright infra + browser-cache fallback).
14. Full-page golden test (current suite is `strings.Contains`-style; a golden pins the whole render and catches accidental structure drift).
15. Fuzz/property test `scriptSafeJSON` against adversarial config strings.
16. HEAD-request test asserting no body render (semantics polish).
17. Cover `New()`'s no-auth slog.Warn path with a log-capture assertion.
18. `TestMount` conflict matrix: method-specific root pattern, subtree pattern, duplicate registration panic surface.

**Hardening depth:**
19. Positive AccentColor grammar (allowlist) behind an opt-in strict flag — default keeps "any CSS color".
20. Percent-encode favicon data-URI components.
21. AuthPrefix URL-validity validation (currently only trailing-slash trim; it flows into the JSON blob and button hrefs).
22. Fail-fast `Redirect` validation option (currently silently sanitized via SafeRedirectPath — fine, but an owner-visible error may beat silent correction).
23. Investigate `encoding/json/v2` escaping options upstream (replace the hand-rolled replacer with a stdlib flag if one exists; otherwise propose it upstream).
24. Document the CSP story end-to-end in README (NonceFromRequest + validation semantics + example middleware chain).
25. `role="alert"` vs `aria-live="polite"` — write the decision down (kept deliberately; one sentence in README a11y notes prevents re-litigation).

**Consistency / architecture:**
26. Review setup→loginpage mounting contract (the un-reviewed ring): path validation interplay, `/` ownership, session-middleware ordering claims.
27. Dark-mode parity question: loginpage is `prefers-color-scheme`-driven; adminui/dashboardui moved to class-driven (2026-09-23) — decide whether loginpage follows or stays media-driven deliberately.
28. Audit loginpage's `.golangci.yml` against the fleet config (163 lines, never reviewed).
29. `Mount` API: method-specific pattern option to kill the documented "GET /{$}" footgun (v5 candidate — breaking).
30. Skip body render for HEAD requests.
31. Consider exposing the favicon as an overridable component (Option B consumers currently get the built-in SVG unconditionally in `Page`).
32. login.js modernization (const/arrow) — zero-dep posture kept; cosmetic; ROADMAP fuel only.
33. i18n hooks for hardcoded English strings — ROADMAP fuel, likely YAGNI.
34. Example wiring: confirm setup-demo exercises loginpage with WebAuthn+OAuth2 so the no-auth fallback and both sections are demo-visible.

**Process / hygiene:**
35. Run `nix run .#preflight-tree-check` before the next batch step in this tree (concurrent-session velocity is high right now).
36. Cross-link the 05:45 report from `docs/reviews/README.md` if that index tracks module reviews.
37. Verify CHANGELOG [Unreleased] has no duplication between my entry and the sibling's (both wrote to it within minutes).
38. Once the adoption settles: re-run the full `nix run .#test-all` battery (28 modules incl. e2e/examples) at a quiet window.
39. `coverage.out` on disk is untracked build residue — confirm no gate expects it committed (checked: git ls-files empty — fine; noting to prevent future confusion).
40. Session-lesson candidate for project memory: "jsonv2 does not HTML-escape — any raw script embedding needs scriptSafeJSON-style escaping" (generalizes beyond loginpage; belongs in AGENTS.md gotchas if a second consumer appears).

## g) Questions I cannot figure out myself

1. **Is the loginpage templ-components adoption owner-approved?** It landed via a concurrent session that also implemented my CSP-nonce follow-up — coordinated behavior — but the 2026-09-23 audit explicitly said "no adoption until an OQ21 trigger fires". If the trigger fired, I annotate the audit as overturned-by-decision; if not, this is a rogue parallel workstream that should be flagged for rollback. Which is it?
2. **Release vehicle for the injection hardening?** The fixes change behavior (previously-rendering config values now fail construction) in a PUBLISHED module. Ride the next family train, or cut an out-of-train loginpage patch (v4.12.2-class) so consumers get the hardening before the next train?
3. **Is "zero external asset requests / self-contained" still a SUPPORTED consumer contract after the adoption?** The 2026-09-23 audit records it as the differentiator for air-gapped/brand-locked consumers. If that promise is retired, the README rewrite is honest deletion; if it must survive, the adoption needs an opt-in/submodule shape instead of the baked-in dependency I see at HEAD.

---

*Point-in-time snapshot 2026-10-04 05:49 CEST. All claims verified against source or marked as observation. Test counts via `grep -c "^func Test"`; commit attribution via `git show --stat`.*
