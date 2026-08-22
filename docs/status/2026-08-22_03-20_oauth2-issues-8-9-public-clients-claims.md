# Status: OAuth2 issues #8 + #9 — public PKCE clients, preferred_username, raw ID token

**Date:** 2026-08-22 03:20 · **Session scope:** `gh issue view 8/9` → implement both in `usermgmt/oauth2` → verify → document
**State at session end:** working tree clean (auto-git daemon absorbed everything), all module-level gates green, work NOT pushed, issues NOT closed, features NOT tagged/released.

---

## a) FULLY DONE

### Issue #8 — public (secret-less) PKCE clients
- `ClientType` typed string enum (`provider.go:24-38`): `ClientTypeConfidential` (default, zero value = old behavior) and `ClientTypePublic`.
- `ProviderConfig.ClientType` field + `Validate()` rework (`provider.go:63-93`): empty defaults to confidential, unknown values rejected (`oauth2.invalid_client_type`), `ClientSecret` required **only** for confidential clients. Fully backward compatible — every existing config keeps its exact old validation outcome (pinned by `TestProviderConfig_Validate_ConfidentialDefaultRequiresSecret`).
- PKCE S256 remains unconditional (it always was — `BeginLogin` always sends `S256ChallengeOption`), which is exactly what makes public clients safe per RFC 7636 / OAuth 2.1.
- `initProvider` needed zero changes: `oauth2.Config` with empty `ClientSecret` is the correct public-client wire shape.
- New tests: `Validate_PublicClientNoSecret`, `Validate_PublicClientWithSecret`, `Validate_ConfidentialDefaultRequiresSecret`, `Validate_InvalidClientType`, `BeginLogin_PublicClient` (auth URL carries `client_id` + `code_challenge`, no secret needed).

### Issue #9 — claims exposure (BOTH proposed options shipped)
- **Narrow:** `PreferredUsername` added to the internal `userInfo` JSON seam (`preferred_username,omitempty`, additive) — extracted from the OIDC ID-token claims (`extractFromIDToken`) AND the non-OIDC userinfo endpoint (`fetchUserInfo`), with GitHub-style `login` fallback (verified by `FinishLogin_PureOAuth2_GitHubLoginFallback` assertion).
- **General:** new exported `Provider.FinishLoginWithToken(ctx, provider, code, verifier) ([]byte, string, error)` — identical userInfo JSON plus the **verified** raw ID token (post-`verifier.Verify`), so consumers (dnsblockd's audit log / header rendering) can read provider-specific claims without a library round-trip or re-verification. Non-OIDC providers return `""` (pinned by test).
- `FinishLogin`'s signature, the `OAuth2Provider` interface, and the usermgmt-side JSON contract are **unchanged** — usermgmt's `OAuth2UserInfo` simply ignores the extra key (documented on `userInfo` as an intentional additive divergence).
- New tests: `FinishLoginWithToken_OIDC` (JWT 3-part shape + claims), `FinishLoginWithToken_PureOAuth2` (empty token + claim), `FinishLoginWithToken_UnknownProvider` (error path).

### Verification (module-scoped, hermetic `GOWORK=off`, `/tmp` caches)
- `go build` OK · `go vet` OK (compiles tests) · `go test -count=1 -race` OK (29 test funcs, was 21) · `golangci-lint run` **0 issues** · coverage **88.4%** (gate 80%, was 88.3%) — all new public paths covered.
- Downstream sanity: full `usermgmt` test suite green (20.7s); `integration_test` module builds hermetically and OAuth2 tests pass (against published oauth2 v4.7.0 — interface unchanged, as designed).

### Docs
- `usermgmt/oauth2/CHANGELOG.md`: Unreleased → Added (3 entries) + consumer note updated ("both resolved").
- Root `CHANGELOG.md`: one Added entry summarizing both issues.
- `docs/adr/0014-oauth2-oidc-integration.md`: both "Open upstream considerations" marked **RESOLVED 2026-08-22** with the mechanism.
- `FEATURES.md`: OAuth2 row updated (public clients, claims, 29 tests).
- `AGENTS.md`: oauth2 module bullet extended; coverage number 88.3→88.4; **cache gotcha rewritten** (see e-4).
- Commits (daemon-authored): `953393e4`, `ce050eec`, `0a4edcbf`.

---

## b) PARTIALLY DONE

1. **Full-repo verification NOT run** — only oauth2/usermgmt/integration_test(OAuth2 subset) were verified. NOT run: `nix run .#build` (26 modules), `.#test`, `.#lint` (15 modules), `.#coverage-gate`, `.#check-modules`, `.#check-cqrs-lint` (oauth2 is in the cqrs-lint module list), `.#check-codegen`, `.#test-fuzz`, `.#test-flake`. Blocker: `/mnt/buildcache` is fully dead (`df` → "No such device"), so every nix gate fails at cache init. I proved the `/tmp` env override works for raw go/golangci commands but did **not** retry the nix apps with `goEnv`-equivalent overrides — unproven whether the flake apps can be pointed at `/tmp` without edits.
2. **Test-depth gap (own choice, honestly flagged):** public-client coverage stops at `BeginLogin`. The **token exchange** for a public client (empty `ClientSecret` → x/oauth2 must omit `client_secret`/Basic auth) is exercised only implicitly, not by an explicit `FinishLogin` test with a secret-rejecting fake server. I reasoned it through but did not prove it on the wire.
3. **`login` → `PreferredUsername` fallback is test-documented but not godoc-documented** — a consumer cannot distinguish "provider sent the claim" from "we fell back to `login`". Should be one sentence on `FinishLogin`.

## c) NOT STARTED

1. **GitHub issue hygiene:** no comment or close on #8/#9 (needs commit refs + version).
2. **Release/publish:** features are unreleased — consumers resolve `oauth2/v4.7.0` from the proxy; no `oauth2/v4.8.x` tag cut (family-version coordination open; buildflow enforces one coordinated family version, and `totp` also sits at v4.7.0).
3. **dnsblockd follow-up** (other repo): once tagged, it can drop its confidential-client secret and read `preferred_username`/raw token.
4. **Promoting `preferred_username` into core** — not added to `identitymodel.OAuth2UserInfo`, `ExternalAccount`, commands, or events (deliberate minimal-scope choice, but it's a real open design decision → question 1).
5. **cqrs-lint** run on the touched module.

## d) TOTALLY FUCKED UP

Nothing destructive, nothing left broken — but three classes of sloppiness this session, all self-caught:
1. **Careless command/edit attempts** (5 instances): typo'd paths (`cqrs--htmx`), mangled env vars (`GOLINT_CACHE`, `GANGWAY=off`), a corrupt `old_string` with a stray `​` character, "AD 0014" typo, "userm"/"usermymgmt" typos. Every one failed loudly and was fixed — but each burned a round trip that exact-match discipline would have saved.
2. **Formatted after testing, not before** — gofumpt flagged `provider.go` only after the first green test run; correct order is fmt → build → test.
3. **Mid-session git confusion** — `git diff` "lost" my changes because the daemon had committed them (`953393e4`/`ce050eec`); I briefly doubted HEAD's content before checking `git show HEAD:`. The AGENTS.md warning about concurrent committers exists precisely for this; consult `git log` before concluding anything from `git diff` next time.

## e) WHAT WE SHOULD IMPROVE (session-level lessons)

1. **Verify the wire behavior you claim:** "public clients work" is only fully proven by an exchange test without a secret. Reasoning about x/oauth2 internals is not evidence.
2. **The Sollbruchstelle comment drifted while I edited it:** "tags kept identical to OAuth2UserInfo on purpose" is now "identical plus one additive field" — I softened the comment instead of escalating the divergence decision to the user. Contract seams deserve explicit sign-off, not prose patches.
3. **Run the module's OWN gate battery from the start** (gofumpt, vet, lint, race, coverage) as one scripted sequence instead of piecemeal invocations — half my wasted round trips were re-running tools I'd already run with wrong env.
4. **`/mnt/buildcache` situation worsened and the documented workaround rotted:** the mount is now gone entirely AND `~/.cache/go-build` is a read-only nix symlink, so the AGENTS.md `$HOME`-based workaround fails with "file exists". Fixed the gotcha in AGENTS.md this session (→ `/tmp` caches); the disk itself still needs hardware attention, and until then every nix gate is effectively unusable.
5. **Issue-to-code loop lacked the closing arc:** implement + verify + document, but no issue comment. The reporter (dnsblockd integration) gets no signal that their two issues are resolved.

## f) NEXT (Pareto-ordered)

1. User gate: push master (`0a4edcbf`.. plus concurrent `b0caabc0` toolchain bump).
2. Comment + close #8 and #9 with commit refs and usage snippets (after push).
3. Decide tag strategy: `oauth2/v4.8.x` alone vs. next family train (see question 2) — note `totp` is also at v4.7.0 and buildflow enforces one family version.
4. Add the public-client **exchange** test (fake server that rejects `client_secret` presence; assert 200 exchange) — closes verification gap b-2.
5. Godoc the `login` fallback on `FinishLogin` (one sentence).
6. Run `nix run .#check-cqrs-lint` (or direct cqrs-lint with `/tmp` caches) on oauth2.
7. Re-run the full nix gate battery once `/mnt/buildcache` is fixed (or wire `/tmp` overrides into the flake `goEnv` temporarily): build, test, lint, coverage-gate, check-modules, check-codegen, fuzz, flake.
8. Decide `identitymodel.OAuth2UserInfo.PreferredUsername` promotion (question 1) — if yes: identity-model field + usermgmt alias + FoldUser/read-model plumbing + tests.
9. Optional (if 8 = yes): persist the handle on `ExternalAccountCore` (event payload `omitempty` = backward-compatible schema addition; no version bump needed for additive JSON, but decide upcaster policy anyway).
10. dnsblockd (other repo): bump to the new oauth2 tag, switch Pocket ID client to public if desired, render `preferred_username` from `FinishLoginWithToken` JSON/token.
11. Extend `setup/config.go`'s oauth2 usage comment with `ClientTypePublic` example.
12. Consider an ADR-0014 addendum or short guide snippet showing all three new capabilities (they're currently godoc+CHANGELOG only).
13. Hardware: replace/repair the sda1 disk behind `/mnt/buildcache`; until then consider switching devShell/flake `goEnv` to `/tmp` or nvme paths so CI-equivalent gates run at all.
14. `docs/status` harvesting: sweep this file + the two new oauth2 CHANGELOG entries into TODO_LIST/CHANGELOG conventions if the docs-health pass requires it.

## g) QUESTIONS (cannot decide myself)

1. **Should `preferred_username` be promoted into core identity-model** (`identitymodel.OAuth2UserInfo` + optionally persisted on `ExternalAccount`/link events), or remain an oauth2-module-only additive JSON field? Promotion is the "real" fix for usermgmt-native consumers; staying module-only keeps the event schema frozen. Both defensible — your call.
2. **Tag timing for the release:** cut `usermgmt/oauth2/v4.8.x` now (dnsblockd can consume it immediately; likely needs a coordinated `totp` no-op tag too, per the one-family-version rule), or hold for the next full family train (which also carries the setup/DEV-only replaces stripping)?
3. **Issue closure mechanics:** want me to comment on #8/#9 myself (with commit refs + usage snippet) and close them, or do you prefer closing them yourself after reviewing the diff? (Push is user-gated either way.)

---

*Verification at close: oauth2 build/vet/race/lint(0 issues)/coverage 88.4% green · usermgmt suite green · integration_test OAuth2 green · tree clean at `0a4edcbf`.*
