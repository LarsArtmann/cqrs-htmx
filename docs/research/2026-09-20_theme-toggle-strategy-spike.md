# Theme Toggle Strategy Spike (adminui) — M088

**Date:** 2026-09-20 · **Status:** spike complete, awaiting M089 sign-off
**Question:** how should adminui gain a user-controllable theme toggle — Tailwind class-strategy migration, or a toggle over the existing `@theme` token-flip?

## Current state (what exists today)

`adminui/tailwind.css` implements dark mode exclusively through **CSS custom-property tokens flipped by `prefers-color-scheme`**:

- One token set (`--bg`, `--surface`, `--text`, `--muted`, `--border`, `--sidebar-*`, …) with light defaults on `:root`, dark values inside `@media (prefers-color-scheme: dark)`.
- A Tailwind v4 `@theme` block maps the gray scale onto the tokens (`--color-gray-500: var(--muted)` … `--color-gray-900: var(--text)`), so library components written as `text-gray-700` automatically follow adminui's palette.
- Two bridge rules patch library expectations: `.bg-white { background-color: var(--surface) }` and the dark-mode `--color-gray-800/900` re-pin (library uses those grays as dark *surfaces* via `dark:` variants; the `@theme` mapping would otherwise paint them light).
- adminui's own templates use **zero** `dark:` variants in audit/dashboard/layout/tenants/users (5 total in components/members) — the tokens do all the work.

Consequence: **no user toggle is possible today.** The theme strictly follows the OS preference; a user on a light OS stuck in a bright room cannot darken the panel, and vice versa.

## Option A — class-strategy migration (`dark:` variants + `.dark` class)

Mechanism: declare `@custom-variant dark (&:where(.dark, .dark *))`, strip the `@theme` gray mapping + token flip, and add explicit `dark:` overrides wherever tokens currently resolve (every `text-gray-*`, `bg-*`, `border-*` consumer across all 8 templ files plus the bridge rules).

- ✅ Industry-default pattern; matches how templ-components itself is authored — future library components work with zero bridge maintenance.
- ✅ Per-component dark divergence becomes expressible (e.g. dimming one card differently).
- ❌ **Large diff**: the token indirection is adminui's entire styling architecture (AGENTS documents it as a deliberate unique look). Migration rewrites ~every class-bearing line in 8 templ files + a tailwind.css overhaul.
- ❌ **High regression risk**: missed spots are invisible until a dark screenshot gate catches them; the existing 20-shot light/dark baselines all churn once.
- ❌ No user-visible gain beyond what Option B delivers (the toggle itself).

Estimated cost: 1–2 sessions plus a full re-baselining of the visual gate.

## Option B — keep token-flip, toggle via `data-theme` (recommended)

Mechanism: duplicate the dark token block under `:root[data-theme="dark"]`, keep the media query as the **first-visit default**, and add:

1. A small header `display.Button` (ghost variant) with sun/moon icon.
2. ~15 lines of CSP-safe JS in admin.js: toggle `data-theme` on `<html>`, persist to `localStorage`, three-state logic (`auto | light | dark`, unset = follow system).
3. A ~5-line nonce'd inline script in `layout.Base`'s head (or ` templ.Raw` in the layout pre-body) applying the stored choice before first paint — the ONLY inline-script addition, same nonce mechanism the modal/toast scripts already use, so no CSP posture change.

- ✅ **~30 lines total**, zero changes to any templ component class, zero churn of the 20-shot visual baselines (auto mode renders byte-identically to today).
- ✅ Achieves the actual goal of M089: a user-controllable theme.
- ✅ The library's `dark:` variants keep working exactly as today via the existing gray-800/900 pin — nothing about the bridge changes.
- ❌ adminui stays token-driven: per-component dark divergence remains expressed as new tokens, not `dark:` classes (this is the documented architecture, and has not been a limitation in practice).
- ❌ `localStorage` needs a cookie-ish fallback if consumers want the choice server-side per session (not requested; noted for completeness).

## Recommendation

**Option B.** The token architecture is a deliberate, documented strength of adminui's theming (one seam, library bridge included), and the migration cost of Option A buys no user-visible capability that B lacks. Revisit class-strategy only if a concrete need for per-component dark divergence appears.

## M089 — decision required (sign-off gate)

Per the remediation plan, no theme code ships before the user picks:

- [ ] **A** — class-strategy migration (bigger diff, industry-default)
- [ ] **B** — token-flip + `data-theme` toggle (~30 lines, recommended)
- [ ] **defer** — no toggle this train; system preference only (status quo)
