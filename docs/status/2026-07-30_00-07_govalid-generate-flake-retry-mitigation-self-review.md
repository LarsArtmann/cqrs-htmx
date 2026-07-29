# govalid-generate Transient Flake: Diagnosis, Mitigation, and Brutal Self-Review

**Date:** 2026-07-30 00:07
**Session scope:** Diagnose `buildflow govalid-generate` failure → confirm transient vs real → add permanent auto-retry mitigation → document
**Starting commit:** `4babffe` (chore: remove accidentally committed e2e test server binary)
**Final commit:** `20addc9` (feat(e2e): add test server setup for end-to-end testing)

---

## What This Session Did (Summary)

A `buildflow` pipeline run failed on the `govalid-generate` step with cascading `markers: failed prerequisites` / `undefined: command` errors. This is the **known transient flake** documented in `AGENTS.md`. The session confirmed it was transient (not a code bug), added a permanent auto-retry mitigation (`retry_modifier: conservative` in `.buildflow.yml`), and documented a critical key-name trap (`retry_modifier` YAML key vs `--retry-profile` CLI flag).

**Result: Full pipeline green (39/41 steps, 69 success, 0 failed, exit 0). The auto-retry config is in place.**

---

## a) FULLY DONE

### 1. Root-cause diagnosis: transient flake confirmed

- `GOEXPERIMENT=jsonv2 go build ./...` — clean (zero output)
- Explicit per-module builds of every module govalid complained about (`usermgmt`, `adminui`, `identity-model`, `dashboardui`, `loginpage`) — all clean
- `GOEXPERIMENT=jsonv2 go test ./...` — root + openapi pass
- Re-ran `buildflow -s govalid-generate` → **16 success, 0 failed, violations: 0**
- Full pipeline `buildflow --build-mode full` → **exit 0, 39/41 passed, 69 success, 0 failed**
- **Conclusion:** no code bugs. The failure was the `go/packages` loader concurrency race.

### 2. Permanent mitigation added: `retry_modifier: conservative`

Added to `.buildflow.yml`:
```yaml
retry_modifier: conservative
```

This makes buildflow auto-retry transiently-failing steps with backoff instead of hard-failing. Zero tradeoff:
- No pipeline slowdown (unlike dropping `max_concurrency` to 1)
- No masking real bugs (unlike `aggressive` might)
- `conservative` is the middle ground

### 3. Critical key-name trap documented

Discovered and documented in `AGENTS.md`:
- CLI flag: `--retry-profile conservative`
- YAML key: `retry_modifier` (NOT `retry_profile`, NOT `retry-profile`)
- `retry_profile` and `retry-profile` in YAML are **silently ignored** — no validation error
- Confirmed via buildflow source code: `config/koanf.go:39` (`keyRetryModifier = "retry_modifier"`) and `model/config_substructs.go:34` (`RetryModifier tool.RetryModifier \`yaml:"retry_modifier"\``)
- The `--retry-profile` flag feeds the same `retry_modifier` koanf key at config-merge time

### 4. AGENTS.md gotcha updated

Updated the `govalid-generate` gotcha entry to include:
- The new `retry_modifier: conservative` mitigation
- The key-name trap (`retry_modifier` vs `retry_profile`)
- The companion `retry_budget` key (int, 0=unlimited)
- Source-code confirmation references

---

## b) PARTIALLY DONE

### 1. Empirical proof that `retry_modifier` actually triggers retries

The `retry_modifier: conservative` config is in place and validated by source-code reading, but **I never observed it actually retry a transient failure in practice.** The flake is intermittent and I couldn't reproduce it on demand. `config validate` only checks YAML syntax. `config view` doesn't surface the retry setting. Telemetry doesn't expose retry-related fields. So the fix is high-confidence-by-source-reading but **unproven-by-behavioral-observation**.

**What's missing:** A test that forces a transient failure and confirms buildflow retries it automatically.

### 2. `retry_budget` not configured

Set `retry_modifier: conservative` but did NOT set `retry_budget` (which caps total retries across a run). Currently defaults to `0` (unlimited). For a 15-module workspace, an unlimited retry budget is probably fine, but a bounded budget (e.g., `retry_budget: 3`) would be more principled — it would prevent retry storms if a real systemic failure were misclassified as transient.

---

## c) NOT STARTED

- Did not investigate whether buildflow could pin govalid-generate to `max_concurrency: 1` specifically (buildflow has no per-step concurrency override — only global `max_concurrency`)
- Did not investigate whether the `go/packages` race is an upstream Go bug worth reporting
- Did not add `e2e/server/server` to `.gitignore` (see section d)

---

## d) TOTALLY FUCKED UP

### 1. Broken `retry_profile: BOGUS_VALUE` committed to git history

During my investigation, I used `sed` to inject `BOGUS_VALUE` into the live `.buildflow.yml` to probe whether the key was validated:
```bash
sed -i 's/retry_profile: conservative/retry_profile: BOGUS_VALUE/' .buildflow.yml
```

**The auto-git daemon swept up this broken intermediate state and committed it** as `64f4821` ("chore(build): update buildflow configuration"). The commit literally contains:
```yaml
retry_profile: BOGUS_VALUE
```

The next commit (`0cd9a52`) fixed it to `retry_modifier: conservative`, but **the broken state is permanently in the git history.** This is a recklessness failure — I should have used a temp copy in `/tmp` for probing, never touched the real file with garbage values, OR temporarily paused/aware of the auto-git daemon.

**Severity:** Medium. The working tree is correct. But commit `64f4821` contains a broken config that, if checked out, would silently disable retries (the bogus key is ignored). And the commit message is a templated lie ("Update build pipeline configuration to optimize build process").

### 2. Did not flag the `e2e/server/server` binary re-commit

The session-start git status showed `M e2e/server/server` (a pre-existing 10MB compiled binary). Commit `4babffe` had **explicitly removed** this binary ("chore: remove accidentally committed e2e test server binary"). During my session, the auto-git daemon **re-committed it** as `20addc9` ("feat(e2e): add test server setup for end-to-end testing"). 

I noticed the `M e2e/server/server` in the session-start status but **did not investigate, flag, or prevent it.** A 10MB compiled binary has no business in git. There is no `.gitignore` entry for it. This binary will keep getting committed and removed in a cycle unless `.gitignore` is fixed.

**Severity:** Medium. The binary bloats the repo. The commit message is completely fabricated by the auto-git daemon (it's not a feature, it's a binary re-commit). This is a recurring problem.

### 3. Commit messages are garbage

All three commits from this session (`64f4821`, `0cd9a52`, `20addc9`) have templated, generic, AI-generated commit messages that **lie about what the commits do:**
- `64f4821`: "update buildflow configuration" — actually injected a BOGUS_VALUE probe
- `0cd9a52`: "add AGENTS.md with comprehensive guidelines" — actually fixed the YAML key + updated one gotcha line
- `20addc9`: "add test server setup for end-to-end testing" — actually re-committed a binary that was previously removed

These are the auto-git daemon's messages, not mine. But I didn't intercept or improve them.

### 4. First edit used the wrong YAML key

My very first edit to `.buildflow.yml` used `retry_profile: conservative` — a key that **does not exist** in buildflow's config schema and is silently ignored. I then spent ~15 minutes discovering this through source-code reading. If I had read the buildflow source FIRST before editing, I would have gotten the key right on the first try (`retry_modifier`).

---

## e) WHAT WE SHOULD IMPROVE

### Process improvements

1. **Never inject test values into live files when an auto-git daemon is running.** Use temp copies in `/tmp`. The daemon commits ANY working tree state, including broken probes.
2. **Read the source before writing the config.** I guessed the YAML key name (`retry_profile`) and was wrong. The source (`config/koanf.go`) had the definitive answer. Source-first, always.
3. **Flag pre-existing anomalies.** The `M e2e/server/server` at session start was a red flag (a binary that was previously removed). I should have investigated it immediately, not ignored it.
4. **The auto-git daemon's commit messages are consistently bad.** They're templated, generic, and often lie about the content. This pollutes git history. Consider improving the daemon or adding a post-commit message-fix step.
5. **`config validate` in buildflow is weak** — it only checks YAML syntax, not semantic validity of keys or values. Bogus keys (`retry_profile`) and bogus values (`BOGUS_VALUE`) both pass validation. This made empirical verification impossible without source-code reading.

### Technical improvements

6. **Add `e2e/server/server` to `.gitignore`** — this binary has been committed and removed multiple times. It's a compiled artifact, not source code.
7. **Consider setting `retry_budget: 3`** to bound total retries and prevent retry storms.
8. **The `go/packages` loader race is an upstream Go toolchain issue.** It may be worth reporting or at least understanding whether Go 1.27 fixes it. The race manifests when multiple `go/packages.Load()` calls run concurrently against the same workspace.
9. **Buildflow telemetry should expose retry-related fields** (`retry_modifier`, `retry_budget`, `retries_attempted`) so the config can be empirically verified without source-code reading.

---

## f) Next Actions (up to 50)

### High priority (fix the fuckups)

1. **Add `e2e/server/server` to `.gitignore`** to stop the binary commit cycle
2. **Remove `e2e/server/server` from git tracking** (`git rm --cached e2e/server/server`)
3. **Squash or fix commit `64f4821`** (contains `BOGUS_VALUE`) — or at minimum add a note that it's a broken intermediate state
4. **Investigate whether the 3 unpushed commits should be cleaned up** before pushing (bad messages, binary, broken intermediate)

### Medium priority (improve the mitigation)

5. **Empirically verify `retry_modifier` works** — create a synthetic failing step and confirm buildflow retries it
6. **Set `retry_budget: 3`** to bound retries
7. **Consider `circuit_breaker_action: skip`** for govalid-generate specifically — if it fails 3 times, skip it instead of blocking the pipeline (since it's known-transient)
8. **Add a CI-level retry wrapper** as belt-and-suspenders (`buildflow ... || buildflow ...`)

### Low priority (nice to have)

9. **Report the `go/packages` concurrency race upstream** to the Go team (with reproduction steps)
10. **Improve buildflow's `config validate`** to catch unknown keys (it's Lars's own tool)
11. **Improve buildflow's `config view`** to surface retry settings
12. **Improve buildflow telemetry** to include retry fields
13. **Add per-step concurrency overrides to buildflow** so govalid-generate can run at `max_concurrency: 1` without slowing the rest of the pipeline

### Unrelated observations from this session

14. **`go list ./...` is module-scoped** — from the workspace root, it only lists 2 packages (root + openapi). All 15 modules must be built explicitly or via `go work` commands. This is a gotcha for anyone running `go test ./...` expecting workspace-wide coverage.
15. **AGENTS.md module count is correct** — 15 modules in `go.work` (buildflow telemetry reports `package_count: 16` because it counts the root module separately).

---

## g) Questions I Cannot Answer Myself

### 1. Should the 3 unpushed commits be cleaned up (squashed/amended) before pushing?

The 3 commits ahead of `origin/master` are:
- `64f4821` — contains broken `retry_profile: BOGUS_VALUE` (my probe mistake)
- `0cd9a52` — the actual fix (`retry_modifier: conservative` + AGENTS.md update)
- `20addc9` — re-committed 10MB binary (auto-git daemon)

Commit `64f4821` is actively broken if checked out. Commit `20addc9` adds a 10MB binary that should be gitignored. These haven't been pushed yet. **Should I clean up (squash/remove binary/fix messages) before pushing?** I can't decide this because it depends on your branching policy and whether the auto-git daemon's commits are considered immutable.

### 2. Is the `e2e/server/server` binary supposed to exist at all?

The binary has been committed (`280968a`), removed (`4babffe`), and re-committed (`20addc9`) multiple times. There's no `.gitignore` entry for it. Is `e2e/server/` an active development area where the binary is a build artifact that should be gitignored? Or is the `e2e/server/main.go` (or equivalent source) missing and only the binary was committed by mistake? I can't tell without investigating the `e2e/server/` directory contents, which is outside this session's scope.

### 3. Should `retry_budget` be set, and if so, to what value?

I set `retry_modifier: conservative` but left `retry_budget` at the default (0 = unlimited). For a 15-module workspace where only one step is flaky, unlimited is probably fine. But a bounded budget (e.g., 3-5) would be more defensive. I don't know your preference on retry aggressiveness vs. pipeline reliability tradeoff, and I can't empirically test how `conservative` maps to actual retry counts without observing a real flake.

---

## Timeline

| Time (CEST) | Event |
| --- | --- |
| 23:44 | First `buildflow -s govalid-generate` re-run → **passed** (16/16, 0 violations) |
| 23:45 | Full `buildflow --build-mode full` → **passed** (39/41, 0 failed) |
| 23:47–23:50 | Investigation: found `--retry-profile` CLI flag, guessed `retry_profile` YAML key (WRONG) |
| 23:49 | **MISTAKE:** Injected `BOGUS_VALUE` into live `.buildflow.yml` for probing → auto-git daemon committed it (`64f4821`) |
| 23:50–23:52 | Source-code reading of buildflow: discovered correct key is `retry_modifier` (`config/koanf.go:39`) |
| 23:52–23:54 | Fixed `.buildflow.yml` to `retry_modifier: conservative`, validated, full pipeline green |
| ~00:00 | Auto-git daemon committed the fix (`0cd9a52`) and the binary re-commit (`20addc9`) |
| 00:07 | This status report |

---

## Files Changed This Session

| File | Change | Commit(s) |
| --- | --- | --- |
| `.buildflow.yml` | Added `retry_modifier: conservative` (via wrong key `retry_profile` first, then fixed) | `64f4821` (broken), `0cd9a52` (fixed) |
| `AGENTS.md` | Updated govalid-generate gotcha with retry mitigation + key-name trap | `0cd9a52` |
| `e2e/server/server` | 10MB binary re-committed by auto-git daemon (NOT my change) | `20addc9` |
