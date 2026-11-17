# PR: fix(templates): position-area alignment bands use x-axis spellings

**Branch:** `fix/popover-position-area-chromium` (templ-components checkout `0f977eaf`)
**Status:** prepared, NOT sent — owner-gated (K3)

## Why

Every popover rendered with a `data-tc-align` variant parks at its static
position on Chromium instead of its band. The declarative anchor-positioning
layer sets `--tc-popover-area` to compound values like `bottom
span-inline-start` — and Chromium rejects those keywords. Because the value
reaches `position-area` through `var()`, the failure is silent and total:
invalid var() substitution resolves at computed-value time to the INITIAL
value, not to a cascade fallback, so even the base `bottom` band never
applies. First seen in the cqrs-htmx adminui consumer, where the user-menu
dropdown rendered at the viewport corner on every engine with anchor
positioning (>= 125).

## Evidence (Chromium 152.0.7977.82, headless, computed styles)

| declaration                                             | computed `position-area` | verdict                           |
| ------------------------------------------------------- | ------------------------ | --------------------------------- |
| `bottom span-inline-start` (the shipped band)           | `none`                   | REJECTED — drops to initial       |
| `bottom span-inline-end`                                | `none`                   | REJECTED                          |
| `bottom span-x-start` (the fix)                         | `span-x-start bottom`    | accepted                          |
| `bottom x-start` / `bottom right` / `bottom span-right` | accepted                 | alternative spellings work too    |
| `bottom` (single-keyword band)                          | `bottom`                 | accepted — unaffected by this bug |

Second finding pinned by the same probe: `anchor()` functions cannot see the
implicit `popovertarget` invoker as anchor — `top: anchor(bottom)` computes
`0px` on an implicitly-anchored popover, `129px` (the real invoker bottom) once
an explicit `anchor-name`/`position-anchor` pair is declared. The library
layer only uses `position-area` (implicit anchors work there), so this is a
consumer-side caveat, not a library bug; recorded here because the cqrs-htmx
consumer's flush-end refinement needed it.

## What changed

- `templates/custom.css`: the four alignment-variant rules switch
  `span-inline-start`/`span-inline-end` to `span-x-start`/`span-x-end`
  (`:152-164`); the section comment records the engine verification and the
  computed-value-time drop mechanism.
- `cmd/tc/_sources/starter/custom.css`: same change (starter copy).

## Test plan

- [x] Headless computed-style probe (the table above) against
      `templates/custom.css` before and after: `none` → `span-x-start bottom`.
- [x] `go test ./display/...` (includes the tooltip test that reads
      `custom.css` directly) and `go test ./cmd/tc/...` green.
- [x] Consumer receipt: the cqrs-htmx adminui dropdown e2e (user menu opens
      anchored after the same correction was vendored there).

## Non-goals

No JS changes: the `tcPopoverPosition` fallback path (engines without anchor
positioning) is untouched and correct. RTL note: `x-start`/`x-end` are
physical-axis spellings; in RTL contexts the alignment side mirrors compared
to the old (never-rendered) intent. If logical spelling matters upstream, the
accepted alternative is `x-start`/`x-end` per probe — both physical.

---

💘 Generated with Crush
