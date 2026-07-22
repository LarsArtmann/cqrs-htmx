Extract usermgmt Into a Dedicated Repo? — PRO / CONTRA Analysis cqrs-htmx · brainstorming TL;DR Context PRO CONTRA Matrix Decision Revisit triggers Architecture Decision · Brainstorming 
# Should `usermgmt `become its own repository? 

A structured PRO / CONTRA analysis of extracting the `usermgmt `submodule out of the `cqrs-htmx `monorepo into a
 dedicated repository — grounded in the actual code, dependencies, and release mechanics as
 of v2.4.0. 

Date: 2026-06-17 Baseline: v2.4.0 (commit 7c71a83) Scope: repo extraction (not module split — already done) Verdict: Do not extract now TL;DR 
Keep `usermgmt `in this monorepo. It is already a
 fully independent Go module with its own `go.mod `, its own module path
 ( `github.com/larsartmann/cqrs-htmx/usermgmt/v2 `), and zero import coupling to the root. The module split has already captured
 ~80% of the benefits usually attributed to "extraction." The remaining upside
 (independent release cadence) is achievable through per-module semver tags without splitting the repo — while the costs of splitting (broken atomic
 refactoring across the UserID/Enforcer bridge, duplicated CI/tooling, fragmented docs)
 fall squarely on the project's most valuable activity. 

0 root → usermgmt imports 0 usermgmt → root imports 39 usermgmt prod files 5,708 usermgmt prod LOC 28% commits touch usermgmt 2 live cross-module contracts 
## Context — where things actually stand 

A lot of "extract it!" intuition assumes the candidate is a tangled sub-package that drags
 its host along. That is not the situation here. Before weighing pros and
 cons, the current state does most of the work: 

#### Already a separate Go module 

`usermgmt/go.mod `declares `module github.com/larsartmann/cqrs-htmx/usermgmt/v2 `. A consumer who runs `go get github.com/larsartmann/cqrs-htmx/v2 `does not download usermgmt's dependency tree (go-webauthn, modernc.org/sqlite, pquerna/otp,
 casbin/v3). Dependency isolation is already a solved problem. 

#### Zero mutual imports 

Grepping every root `.go `file: not a single import of `usermgmt `. The reverse is also true. The only references are two doc
 comments in `doc.go:80-82 `. The DAG is clean — confirmed by `docs/modularization/DEPENDENCY_GRAPH.md `. 

#### One bridge, two contracts 

`integration_test/ `(363 LOC, 3 files, its own `go.mod `) is the
 only thing that imports both. It verifies exactly two live contracts: the UserID string bridge ( `usermgmt.NewUserID `↔ `cqrshtmx.ParseUserID `) and the Enforcer adapter ( `authz.AsEnforcer() `satisfying root's `Enforcer `interface). See ADR 0002. 

#### Precedent for independent versioning exists 

The tag `usermgmt/v2.0.0 `already exists on the repo — proving Go's `module/v2 `sub-path scheme works here. Today the project ships a single
 monorepo version, but the mechanism for per-module semver is in place and unused. 

Why this analysis (and not just citing the 2026-05-27 proposal) 
`docs/modularization/PROPOSAL.md `concluded "the existing split is perfect —
 do not extract further." But its data is stale : it sized usermgmt at "9 production files, ~80 exported symbols." The module is now 39 production files, ~104 exported symbols, 5,708 LOC — more than 4×
 larger, with a SQL event store, WebAuthn, TOTP, import/export, and email verification
 added since. The prior decision deserves a fresh look on current data. 

## PRO — arguments for extraction 

These are the strongest honest cases for moving usermgmt to its own repository. Several
 are real — but note how many of them are already delivered by the existing module
 split, which sharply reduces their marginal value. 

#### 1. Independent release cadence 
Moderate · partial already-met 
28% of all commits (144 / 505) touch usermgmt, and recent history is almost entirely
 usermgmt feature work (TOTP, import/export, email verification, SQL event store). Its
 velocity is higher than the root HTTP plumbing. A separate repo lets usermgmt cut
 releases on its own heartbeat instead of waiting for a coordinated monorepo tag. 

Caveat: per-module semver tags (usermgmt/v2.x.y) give the same cadence without leaving
 the repo. Release independence ≠ repo independence. 
#### 2. Standalone identity & discoverability 
Strong · if that is a goal 
usermgmt is a genuinely complete passwordless auth library — WebAuthn/Passkeys,
 event-sourced users, Casbin RBAC, sessions, TOTP MFA, email verification, SQL event
 store. It does not import root and could be adopted by someone who never uses HTMX. A
 dedicated repo + README + godoc URL gives it its own story and search surface. 

This only matters if adoption outside the cqrs-htmx ecosystem is a strategic goal. As
 a private dependency of cqrs-htmx, the benefit is theoretical. 
#### 3. Decoupled go-cqrs-lite upgrades 
Moderate 
Both modules share go-cqrs-lite, go-error-family, and casbin. Today a bump is a
 coordinated commit across root + usermgmt + integration_test (see commit `f131b20 `: v2.3.0 → v2.4.0 across all modules). With separate repos,
 usermgmt could adopt a new go-cqrs-lite major on its own schedule without forcing root
 consumers along. 

#### 4. Focused issue / PR / CI surface 
Weak for solo, moderate for many contributors 
A dedicated repo means a dedicated issue tracker, PR queue, and CI status — useful
 when the two attract different contributor communities. A webauthn specialist
 shouldn't have to triage HTMX CSRF issues and vice versa. 

Commit pattern looks like a solo/small team today. Splitting trackers adds cross-repo
 linking overhead with little triage relief at this scale. 
#### 5. Sharper governance / licensing boundary 
Weak · speculative 
If usermgmt ever moves to a foundation, attracts a different license, or needs a
 security disclosure process distinct from the HTTP layer, a separate repo is the
 natural vehicle. 

#### 6. Smaller clones for single-side contributors 
Negligible 
Total repo is ~27K LOC across all modules. Extracting usermgmt shaves ~13K. Not enough
 to matter for clone or CI time on any modern runner. 

## CONTRA — arguments against extraction 

The costs cluster around one uncomfortable truth: the two modules are not actually independent — they share a small but evolving contract
 surface , and repo extraction makes every change to that surface more expensive. 

#### 1. Atomic cross-seam refactoring breaks 
Strong 
The UserID bridge and the Enforcer adapter are co-designed contracts (ADR
 0002). Changing how `usermgmt.UserID `serializes ripples into `cqrshtmx.ParseUserID `and the integration tests. In the monorepo this is
 one commit, one PR, one CI run, atomic and bisectable. Across repos it becomes: edit
 usermgmt → tag → bump root → bump integration_test → re-tag. The seam is small but it
 is live , and live seams evolve. 

#### 2. Coordinated dependency upgrades get worse 
Strong 
go-cqrs-lite is a shared dependency. Bumping it today is one commit
 ( `f131b20 `) touching all four modules and verified by one CI run. After
 extraction the same change is N cross-repo PRs with a version-ordering constraint
 (root and usermgmt must publish compatible majors before integration_test can
 resolve). You trade one atomic operation for a release choreography. 

#### 3. Bridge testing becomes painful 
Strong 
`integration_test/ `currently pins both modules via local `replace `directives and tests the bridge in one `go test `.
 Post-extraction you must choose: (a) pin to published versions — a slow
 push-tag-bump-test loop that lags real bugs; or (b) keep `replace ../usermgmt `— which only works if contributors clone both repos
 as siblings in a fixed layout, an undocumented and fragile assumption. Neither
 preserves today's "one `git push `tests everything" guarantee. 

#### 4. Independent versioning is a paradox here 
Strong 
The textbook benefit of separate repos is independent versioning. But independent
 versioning only pays off when contracts are stable . The UserID/Enforcer
 bridge is explicitly not v1-frozen (ADR 0002 still calls it a deliberate split). Two
 repos with a moving contract don't release independently — they release in forced lockstep , just with worse tooling. You get the overhead of independent
 versioning without its freedom. 

#### 5. CI / tooling duplication 
Moderate 
One `flake.nix `, one `.github/workflows/ci.yml `, one dependabot
 config, one golangci-lint config currently cover all four modules uniformly (with
 per-module gates: root ≥95%, usermgmt ≥90% coverage). Extraction means duplicating all
 of it and keeping the duplicates in sync — a new source of drift bugs. 

#### 6. Consumer UX & narrative fragmentation 
Moderate 
cqrs-htmx's pitch today is a coherent "batteries-included HTMX + CQRS + auth toolkit."
 A consumer wanting the full stack finds it in one README, one go.work, one example.
 Splitting usermgmt forces consumers to discover, version-match, and integrate two
 libraries — and the root repo suddenly looks thinner and less compelling in isolation. 

#### 7. Documentation drift 
Moderate 
`AGENTS.md `, `DOMAIN_LANGUAGE.md `, `CHANGELOG.md `,
 and the ADRs currently cross-reference root and usermgmt behavior in one place (e.g.
 the UserID bridge is documented from both sides). Splitting fragments this and invites
 the two halves to drift — exactly the failure mode that makes bridges rot. 

#### 8. No automated release pipeline today 
Moderate 
CI builds/tests/lints/sec-scans but does not tag or publish — tagging
 is manual, and even the existing `usermgmt/v2.0.0 `tag is unused. Adding
 repo extraction before having a release pipeline for a single repo is putting
 the cart before the horse: build the versioning muscle in the monorepo first. 

## Net assessment — weighted score 

Scoring each side by realistic weight for a small-team, private-dependency library at
 v2.4.0. Weights reflect impact on day-to-day engineering, not theoretical purity. 

32% 
### Extraction case scores ~3.2 / 10 against current state 

The case for extraction is dominated by benefits the module split already
 delivers. The case against is dominated by real, recurring friction on the
 two live bridge contracts. On current data, extraction is a net negative. 

### Decision pillars 
Dependency isolation KEEP — already solved Consumers of root already skip usermgmt's dep tree. Extraction adds nothing. Independent release cadence ACHIEVABLE IN-PLACE Per-module semver tags give this without leaving the repo. Contract evolution KEEP — needs atomicity UserID + Enforcer bridges are live and co-designed. Atomic commits protect them. Shared-dep upgrades KEEP — needs atomicity go-cqrs-lite bumps touch both; one commit > a release choreography. Standalone identity FAVORS EXTRACTION Only real new benefit — and only if independent adoption is a goal. Tooling overhead KEEP — avoid duplication One flake/CI/lint config beats N synchronized copies. 
## State comparison — current vs extracted, dimension by dimension 
Dimension Monorepo (today) Dedicated repo Winner Root consumers pull usermgmt deps No — separate go.mod (already isolated) No Tie Independent compile / build Yes (per-module apps in flake) Yes Tie Independent versioning possible Yes (usermgmt/v2.x.y tags, Go sub-path scheme) Yes Tie Atomic UserID/Enforcer bridge change 1 commit, 1 CI run Multi-repo choreography + tag ordering Monorepo go-cqrs-lite / casbin major bump 1 commit across all modules N coordinated cross-repo PRs Monorepo Bridge integration test Local replace, 1 `go test `Published-version lag OR sibling-checkout assumption Monorepo CI / lint / flake config 1 copy, uniform gates 2 copies, drift risk Monorepo Docs cross-references (AGENTS, ADR, DOMAIN_LANGUAGE) Single source, consistent Fragmented, drift risk Monorepo Standalone library identity / discoverability Subordinate to cqrs-htmx Own README, godoc, stars, issues Dedicated repo Divergent contributor communities Shared tracker Focused tracker Dedicated repo (if they materialize) Clone size / CI minutes ~27K LOC total ~13K LOC each Negligible 
### Where each side genuinely wins 

#### Monorepo wins (operational) 

 - Atomic bridge refactors 
 - One-commit shared-dep upgrades 
 - Single CI / flake / lint config 
 - Local-replace bridge testing 
 - Unified docs & ADRs 
#### Monorepo wins (already solved) 

 - Dependency isolation (go.mod) 
 - Independent compilability 
 - Per-module versioning mechanism 
 - Per-module coverage gates in CI 
#### Dedicated repo wins (strategic) 

 - Standalone library identity 
 - Own discoverability / godoc URL 
 - Independent governance / license 
 - Focused issue tracker — at scale 
#### Dedicated repo costs 

 - Multi-repo release choreography 
 - Bridge-test friction (replace across repos) 
 - Duplicated & drifting tooling 
 - Fragmented docs / consumer UX 
## Recommendation — the call 
● Verdict: Do not extract now 
### Keep `usermgmt `in the monorepo. Earn independent release cadence via
 per-module tags instead. 

The module split has already banked the structural wins (isolation, independent compile,
 clean DAG). What repo extraction would add — standalone identity — is real but
 speculative for a private dependency, while what it would cost — atomicity on
 the two live bridge contracts and one-shot shared-dependency upgrades — is concrete and
 recurring. The cost/benefit ratio is wrong today . 

If independent releases are the actual goal, do this instead (in priority
 order): 

 - Adopt per-module semver. Cut `usermgmt/v2.4.0 `, `v2 `(root), `integration_test/v2 `tags independently. Go's
 sub-path versioning already supports this and the `usermgmt/v2.0.0 `tag
 proves the mechanism. No repo move required. 
 - Ship an automated release pipeline. Today tagging is manual and there
 is no publish workflow. Build the versioning muscle for one repo before multiplying
 repos — extraction without automation is pure overhead. 
 - Freeze the bridge contracts to v1. Publish the UserID string format
 and the `Enforcer `interface as a stable, versioned surface (new ADR). Once
 the contracts stop moving, the strongest contra argument evaporates and extraction
 becomes low-friction. 
 - Then , if standalone adoption becomes a real goal, extraction is a `git filter-repo `+ tag republish away — cheap, because the module boundary
 is already clean. 
## Revisit triggers — when to reopen this decision 

This is not a "never." It is a "not now." Re-evaluate extraction the moment any of these
 becomes true — each one shifts the cost/benefit balance decisively. 

1 The bridge contracts freeze to a stable v1. UserID serialization and the Enforcer interface get a versioned,
 backwards-compatible guarantee (new ADR). Removes the strongest contra — atomic
 refactoring is no longer needed. 2 usermgmt's dependency tree diverges sharply from root's. e.g. usermgmt adopts a SQL/ORM/migration stack, a templating engine, or an
 observability SDK that root will never use — making the "shared deps" contra flip
 into a "decoupled upgrade" pro. 3 usermgmt attracts independent maintainers or a separate contributor
 community. Distinct ownership justifies a distinct tracker, CI, and review surface — the
 "focused PR queue" pro becomes real rather than theoretical. 4 Release cadence diverges by an order of magnitude. If usermgmt ships weekly while root ships quarterly, coordinated tagging becomes
 genuine drag rather than occasional chore — independent repos start paying for
 themselves. 5 Standalone adoption becomes a strategic goal. A decision to position usermgmt as a general-purpose Go auth library (its own docs
 site, its own stars, consumed without HTMX) — then the identity/discoverability pro
 is the deciding factor. 6 usermgmt needs different governance, licensing, or a security disclosure
 process. e.g. contribution to a foundation, dual-licensing, or a separate CVE pipeline. A
 repo boundary is the natural enforcement point. Safety net 
Because the module boundary is already clean (zero mutual imports, independent go.mod,
 existing `usermgmt/v2 `tag mechanism), extraction remains a reversible, low-risk move at any future point. Nothing about staying in
 the monorepo today forecloses the option. That reversibility is itself a reason to
 defer: pay the cost only when a trigger fires. 

docs/brainstorming/extract-usermgmt-pro-contra.html Baseline v2.4.0 · 2026-06-17 · grounded in code, deps & release mechanics 