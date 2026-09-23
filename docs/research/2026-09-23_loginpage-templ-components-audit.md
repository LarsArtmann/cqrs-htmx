# loginpage × templ-components — Adoption Audit (mini-report)

**Date:** 2026-09-23 · **Task:** adoption-plan T21 · **Series:** templ-components consumer audits (dashboardui: 2026-09-17, 2026-09-23)
**Verdict: KEEP hand-rolled for now** — adoption is mechanically easy (one 86-line templ file) but the cost is infrastructure + identity, not markup. Revisit triggers in ROADMAP OQ21 are unchanged.

## Census (21.1)

| Metric | Value | Evidence |
| --- | --- | --- |
| templ files | 1 (`page.templ`, 86 lines; generated `_templ.go` committed) | `ls`, `wc -l` |
| templ-components requires | **0** (zero-dep posture) | `go.mod` |
| unique `lp-*` classes | 33 | grep census, non-generated sources |
| interactive hand-rolled surfaces | 6 (2 email inputs, 1 text input, 2 submit buttons, 1 live error region) | grep `lp-input`/`lp-btn`/`lp-error` |
| CSS/JS delivery | `go:embed`-inlined, zero external asset requests (README contract) | `assets.go` |

## The 4 AGENTS.md opportunities, mapped (21.2)

1. **`recipes.AuthLayout`** — the entire page shell. Direct structural replacement, BUT it replaces the module's differentiating identity (favicon-rune brand, custom accent var, footer) with the recipe's standard identity. This is the actual decision, not a markup swap.
2. **`forms.Input`** — the 2 email + 1 display-name inputs (`lp-input`). Drop-in; `BaseProps.ID` must pin `lp-email`/`lp-reg-email`/`lp-reg-name` — login.js targets them.
3. **`feedback.Alert`** — the `#lp-error` live region (`role="alert"`, `aria-live="polite"`, JS-populated). The library Alert is props-driven server render; the JS-populated live region is a different pattern (Alert would need to render the container and let login.js fill it — hook preservation required).
4. **`display.Button`** — the 2 submit buttons (`lp-btn lp-btn-primary`, JS-targeted `lp-login-btn`/`lp-register-btn`). Drop-in with pinned IDs.

## Cost model (why "not now")

- **Bundle infrastructure (~half-day):** compiled Tailwind bundle + `build-loginpage-css` flake app + `check-css-bundles` wiring + embed — every step exists as precedent (adminui/dashboardui) but none exists here.
- **Dependency posture (permanent):** first templ-components require ends the module's "zero external asset requests, self-contained" README contract — its stated differentiator for consumers who vendor it into air-gapped or brand-locked apps.
- **Identity reconciliation + test updates (~half-day):** AuthLayout's brand/identity vs the custom one; golden/handler-test/login.js selector updates.

Total ≈ 1 session for a module whose hand-rolled surface is 33 classes and 86 lines — the swap trades a small, stable, fully-owned surface for a dependency edge and a design-system coupling.

## Verdict (21.3)

**No adoption this train.** The two OQ21 revisit triggers stand: (a) templ-components ships a prebuilt embeddable CSS artifact (kills the bundle-infrastructure cost), or (b) a consumer asks for login-page design parity with adminui (flips the identity trade). Until then, loginpage stays the fleet's zero-dependency control group — which also keeps the audits honest about what the library actually buys.

When either trigger fires, start from §21.2's map (the four swaps are pre-scoped) and follow the dashboardui adoption playbook (`docs/status/archived/2026-09-18_05-42_dashboardui-adoption-run-2-complete.md`).
