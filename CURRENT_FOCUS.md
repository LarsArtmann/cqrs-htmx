# Current focus

> One file, one thread. Every session starts here and in [AGENTS.md](AGENTS.md).
> Rewrite this file when the focus changes; never append history to it (that is what
> [CHANGELOG.md](CHANGELOG.md) and `docs/status/` are for).

**As of:** 2026-11-14 (afternoon, adminui next-wave close-out) · **Owner:** Lars · **Module count:** 30 (`go.work`)

## The active thread

**The adminui next-wave plan executed M00–M17 — receipts in the plan's §9; the tree waits on the owner packets and the exit report.**
The 2026-11-14 SUPERB adminui plan ([`docs/planning/2026-11-14_08-45_SUPERB-adminui-tenant-user-management-next-wave-pareto-plan.md`](docs/planning/2026-11-14_08-45_SUPERB-adminui-tenant-user-management-next-wave-pareto-plan.md))
landed everything owner-independent: **M09** header tenant switcher (super-admin, live tenants,
current marked), **M10** list polish (status filter + member sort + blast-radius confirms; the
per-tenant audit shortcut reverted honestly — audit entries carry no tenant attribution, ask (h)(5)),
**M06** bulk user import/export (literal routes, size-capped, header-guarded), **M07** last-seen,
**M08** member attribution, **M11** a11y sweep with the first findings fixed (Label-in-Name, aria-current
chips, named icon buttons, chip contrast) and the keyboard path pinned, **M12** gotcha 60 + README truth
+ admin screenshot harness, **M13/M15/M16** verification + decision packet, **M14** archive pass (tail
14→3, corpus re-trued), **M17** family-train alignment (loginpage cyclop + admin-demo nolint fixed,
catalog v4.15.0 swept, **strict gate 0 lag**, catalog-demo's A25 red healed by the bump).
Browser proof: the admin spec family (behavior + tenant-access + a11y) is 18/18 chromium green on the
settled tree; chromium full battery green + ran-count green; firefox stays ADVISORY (66
`setDownloadOptions` protocol failures = Playwright↔Nix-firefox binary mismatch, environment class).

Three decision packets now sit one tick from GO, all with recommendations pre-staged:

- **adminui next-wave D1–D3** ([`docs/proposals/2026-11-14_adminui-next-wave-owner-decisions-d1-d3.md`](docs/proposals/2026-11-14_adminui-next-wave-owner-decisions-d1-d3.md)) —
  impersonation GO behind opt-in config (erratum: usermgmt v4.20.0 already publishes the API — no tag
  needed) / batch the five usermgmt asks of TODO item (h) into ONE tag train / sessions+impersonation
  before bots.
- **M2M D1–D4** ([`docs/proposals/2026-11-07_m2m-owner-decisions-d1-d4.md`](docs/proposals/2026-11-07_m2m-owner-decisions-d1-d4.md)) —
  unchanged; one word per row converts W1/W2 to GO.
- **usermgmt → system/metaengine D1–D5** ([`docs/proposals/2026-11-07_usermgmt-system-metaengine-migration-decisions-d1-d5.md`](docs/proposals/2026-11-07_usermgmt-system-metaengine-migration-decisions-d1-d5.md)) —
  defaults adopted; W1–W2 probes stay GO without further owner input.

Known standing reds, all owner-tracked or self-healing: docs-gates audit-all (drain-or-re-pin, TODO P2
top row), docs-gates freshness (greens when this session's exit report lands), firefox advisory engine.
The 2026-11-08 blanket-directive receipts are fully drained; nothing from it remains actionable.
