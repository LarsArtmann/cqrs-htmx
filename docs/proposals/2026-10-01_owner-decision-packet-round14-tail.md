# Owner decision packet — round-14 tail (2026-10-01)

Nine decisions, each with the evidence and a recommendation. Items 1–2 are the PapDashboard architectural pair; item 9 is the ready-to-send reply that references them.

## 1. OQ23 — split `setup` into `setup/core` + identity wiring (ROADMAP 23)

**GO, conditional on the consumer's flip trigger.** Acceptance criteria are already recorded verbatim from PapDashboard feedback (their go.mod gains `setup/core` but NOT `usermgmt`/`identity-model`; a `setup.New`-equivalent boots with zero identity goroutines). Cost: one new module + panel-injection seams (~1–2 days incl. tests, following the module-per-boundary family pattern). Recommendation: commit to the split when PapDashboard's SaaS/multi-user flip trigger approaches — not before (YAGNI today: their recorded current mode is single-operator/key-auth, where the split buys nothing). The design question (new module vs subpackage) resolves to NEW MODULE — build-tag-free, dependency-direction-clean, consistent with the 28-module family.

## 2. OQ24 — pluggable SSE envelope/encoder (ROADMAP 24)

**NO-GO today.** No consumer is blocked (PapDashboard serves their own stream via the Path 0 building blocks); the golden-pinned envelope is a FEATURE for default-path consumers, and a seam that weakens the pin would trade a real guarantee for a hypothetical. Revisit only when a second consumer needs a different FROZEN wire shape — then design the seam so default-path bytes stay pinned. Cost if ever built: ~2–3 days (design + golden-test preservation + migration doc).

## 3. OQ26 — fleet go-directive policy

**KEEP the patch pin (`go 1.27.1`) everywhere.** The fleet re-pin settled this on 2026-10-01; patch-pinning is what makes the 16 `.golangci.yml` `run.go` values, the go.work floor, and the toolchain resolution agree. The residual `go-line-flipflop` warnings are historical churn noise (the settling is done), and published modules keep minor-only floors in their PUBLIC go.mod (already the practice). Close as decided.

## 4. `examples/datastar-demo` — KEEP as-is (confirm)

**KEEP.** Evidence stands: it is the only runnable proof of the datastar adapter + `broadcast` wiring; its cost is one examples module. No rebrand, no removal.

## 5. loginpage OQ21 — templ-components adoption

**DEFER until the next loginpage change** (same posture as SidebarNav): the page is hand-rolled `lp-*` CSS by design (documented), works, and has no open defect. Adoption (recipes.AuthLayout, forms.Input, feedback.Alert, display.Button) is a bounded ~2 h task WHEN touched next, not before.

## 6. rawIDToken suppress-with-reason — confirm-and-close (round-14 g1)

**EXECUTED per the recorded decision** (17:01 session): the trio at `usermgmt/oauth2/provider.go` carries `//nolint:erraudit // LIVE-SECRET: id_token is a live credential` + an `extractFromIDToken` doc comment stating the secrecy contract. 0 erraudit criticals verified across all 28 modules since. Tick to close; the precedent (LIVE-SECRET class) is documented in gotcha 8. One adjacent observation for the security review: `exchangeAndExtractUser` attaches `pkce_verifier` (a credential-class value) via `WithContext` on the token-exchange error — single-use and short-lived, but it IS echoable context; recommend the same LIVE-SECRET treatment in the next touch of that function.

## 7. `/mnt/buildcache` reclaim

**Urgency DOWN, decision still owed.** Post-reboot the disk sits at 72% (59G free) — the 99% crisis self-resolved. The structural candidates remain (rust/ 155G, sccache/ 20G — other ecosystems' caches, not this repo's to delete). Recommendation: authorize a one-time `cargo cache` prune + sccache gc at the next maintenance window; until then the existing watch (`df -h` before gate batteries) suffices.

## 8. T08 — system cqrs-lint binary swap (fleet)

**All local verification is DONE** (rules diff = +1 rule C043 only; strict root walk 0×C040, was 21 phantoms; 14/14 gate modules green under the new binary). The remaining step is fleet-level and deliberately yours: bump the cqrs-lint source pin in the fleet config (the pin that builds `/run/current-system/sw/bin/cqrs-lint`, currently `3756eb4`) to go-cqrs-lite `b06ac8add5e8`+ and switch. After the swap: re-run the rules-diff ritual here (expect clean — done once already), expect B024 to re-appear at `setup/setup.go:139` / `usermgmt/es_setup.go:221` (documented FP — re-add the suppressions with the applyBusMiddleware reason per the triage note), and delete the gotcha-13 caveat.

## 9. PapDashboard reply — READY TO SEND

Verified today against proxy.golang.org: `usermgmt/v4 v4.13.0` and `setup/v4 v4.13.2` both resolve — the codec-migration adoption prerequisite is LIVE. Draft reply:

> The v4.13.0 family train shipped 2026-10-01: usermgmt v4.13.0 carries the go-codec migration you recorded as your adoption prerequisite (verified resolving from proxy.golang.org today), and setup v4.13.2 stabilizes `RunWithAppkit` + the ownership seams. Your two architectural asks are tracked with your acceptance criteria verbatim: the setup/core split (#23 in our ROADMAP — acceptance: your go.mod gains setup/core without usermgmt/identity-model; zero identity goroutines at boot) and the SSE envelope seam (#24 — we'd design it so default-path wire bytes stay golden-pinned). Neither blocks your Path 0 usage today. Tell us when the SaaS/multi-user flip trigger approaches and #23 goes to the front of the queue.

--
💘 Generated with Crush
