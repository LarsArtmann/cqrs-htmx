# Status — Browser e2e battery wave 1: DataStar coverage landed, suite flake surfaced

> Session 2026-11-07 08:55–09:13 (continuing the 08-16 session's §f residue under the
> [SUPERB browser-e2e plan](../planning/2026-11-07_08-55_SUPERB-browser-e2e-battery-ci-and-datastar-coverage.md),
> scope-corrected mid-flight to cover ALL sub-projects, not just datastar-demo).
> Trigger: owner pushback — "you focus too much on the datastar-demo! What is with all our other sub projects!"

## §a Fully done

1. **Plan of record written and scope-corrected** — `docs/planning/2026-11-07_08-55_SUPERB-browser-e2e-battery-ci-and-datastar-coverage.md` with the Pareto split (1 %→51 %: the datastar boot spec; 4 %→64 %: + CI browser job; 20 %→80 %: + SSE-contract/multi-tab/firefox/docs), medium tasks M1–M6 with micro tasks F01–F26, mermaid execution graph with kill points, AND the breadth wave M7–M11 (the owner's correction): root HTMX helpers via examples/basic, setup bundle via setup-demo, dashboardui live/replay depth via an e2e/server debug-emit endpoint, health+auditlog mounts, usermgmt session flows — with the architecture rule (shipped examples = webServers; root surfaces/bridges = mounted into the shared e2e/server).
2. **M1 complete: DataStar adapter behavior gate** — `e2e/tests/datastar.spec.ts` (6 specs, serial): boot (the exact `data-on:load` regression assertion), create round-trip (bound signal → @post → elements+signals patches + input cleared), live-feed delivery, live-only honesty (fresh client receives NOTHING from before its connection), multi-tab fan-out, console-error guard. All 6 green in 3.1 s. Wired as the third webServer (:18931) via a new `DATASTAR_DEMO_ADDR` env on the shipped demo (the admin-demo pattern — the artifact under test stays the shipped example).
3. **Two empirically-verified Playwright facts recorded in the spec** (each cost a red run, each now prevents the next): (a) `page.waitForResponse` NEVER fires for an open fetch-streamed SSE response (request event fires; response event does not while the stream is open — probe-verified); (b) `context.setOffline` does NOT sever an ESTABLISHED fetch stream (the "missed" event arrived through the still-open connection). Both documented inline; connect-proofs are event-based with bounded retries instead.
4. **Full battery re-verified with the new webServer entry: 80/82 green** — see §d for the 2.

## §b Partially done

1. **M7 (root HTMX helpers via examples/basic)** — recon complete (showcase markup, handlers, `:8096` hardcoded port, index live-loop with SSE-triggered `htmx.ajax` refresh mapped), spec NOT yet written, `BASIC_DEMO_ADDR` env NOT yet added, webServer :18932 NOT yet wired.
2. **Flake classification** — the 2 sync failures are isolated-green (see §d) but the stabilization task is not done.

## §c Not started (planned, in order)

- M2: CI `e2e` job (bun + Playwright chromium + sibling clone + workspace Go; setup-bun SHA to resolve live).
- M8–M11: setup-demo, dashboardui depth (debug-emit + live row + time-travel + journal replay), health+auditlog mounts, usermgmt session flows.
- M4 firefox (project restructure + local verify; kill point 30 min), M5 docs/ritual, M6 commits+push+push-and-watch.

## §d Totally fucked up (owned)

1. **Full-suite run: 2 sync failures** (`offline reads serve cached events via window.cqrsSync.getEvents()` + `sync:reset on backendId change` — both `getEvents()` → 0, expected ≥2, 15 s poll exhausted). Rerun IN ISOLATION with my config in place: **7/7 green in 17.8 s** → the class is cross-suite interference (both tests sit at the END of the 1.8 m serial run; worker/event-cache startup raced under accumulated load), not a functional break from this session's changes. NOT yet root-caused — routed to §e. Honest caveat: CI never runs this suite (that's M2), so "green in CI" cannot be cited either way.
2. **Two wrong mechanisms burned before the empirical probes**: waitForResponse-based connect proof (structurally impossible for fetch-streamed SSE) and setOffline-based missed-event test (streams survive offline emulation). Each was a plausible-looking choice that only a probe run falsified — the probes are now the spec's documented load-bearing comments.

## §e What we should improve (this session's evidence)

1. **The suite's tail is timing-fragile**: the two heaviest IndexedDB specs fail under full-battery load but pass isolated. Fix candidates, cheapest first: (a) CI retries already config'd (`retries: 1` under CI) — insufficient if the failure is deterministic-under-load; (b) raise the two specs' poll timeouts; (c) investigate the worker startup race (getEvents resolving 0 means the events store read raced the pull completion); (d) split the battery into two Playwright projects (sync-only serial vs rest) so sync never runs starved.
2. **Connection proofs must be event-based in this stack** — no waitForResponse for streams; document once in a shared helper (the datastar spec's `connectStream` should graduate to a shared util when the second SSE spec lands — M9).
3. **The daemon remains a live hazard for in-flight work** (two content-replacement strikes last session): keep the `git show HEAD:<file>` ritual after every daemon commit.

## §f Next up to 50 (ranked)

1. M7: add `BASIC_DEMO_ADDR` env + `root-htmx.spec.ts` (partial-vs-full contract, OOB block, RenderIf via real button click, SSE-triggered `#items` refresh) + webServer :18932.
2. M2: CI `e2e` job — the single highest-leverage change in the plan (4 %→64 %).
3. Flake: rerun the full battery once more to size the sync-tail failure rate; then (b)/(c) from §e.1.
4. M9: e2e/server `POST /api/debug/emit` + dashboardui depth specs (live row append, time-travel slider, journal replay after reconnect — the honest counterpart to datastar's live-only spec).
5. M8: setup-demo webServer (:18933) + session-gate/readiness spec.
6. M10: mount health `NewDashboard` + auditlog viewer into e2e/server + smoke specs.
7. M11: usermgmt logout/`me` posture specs on the loginpage surface.
8. M4: firefox project (move `E2E_BROWSER_PATH` into chromium project; exclude webauthn spec — Playwright WebAuthn is Chromium-only; verify locally; CI-gate on that).
9. M5: e2e/README matrix table + corpus/ledger ritual for this plan+report.
10. M6: logical commits (e2e battery / CI / docs), push via the gotcha-50 protocol if the pre-push hook reds on the documented API-vs-tag class, `.#push-and-watch`.
11. Graduate `connectStream` into a shared helper when M9 lands its first SSE spec.
12. `check-modules`/`check-shellcheck` pass over anything script-adjacent this train touches (none expected — Go/TS/YAML only).

## §g Questions for the owner (cannot be answered from the repo)

1. **Sync-tail stabilization strategy**: accept CI-retries masking (fast, ships M2 sooner) or fix the worker race first (slower, honest-green)? The failure rate under full load is not yet measured (§f.3).
2. **CI e2e budget**: is a ~4–6 min added CI job (three `go run` webServer compiles + browser install) acceptable on every PR, or should the e2e job ride only the release train lane?
3. **Firefox promotion**: include in CI once locally green (adds ~1–2 min + a second engine's flake surface), or keep chromium-only in CI with firefox as a local/nightly lane?

— End of report. Awaiting instructions.
