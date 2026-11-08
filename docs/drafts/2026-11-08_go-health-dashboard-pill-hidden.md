# Title: connection pill shows all three states at once — Tailwind display classes defeat the `hidden` attribute

## Problem

The dashboard header's connection pill renders "Live", "Reconnecting", and "Offline" simultaneously in a real browser. The three spans toggle via the `hidden` attribute (`setPillState` in `page_scripts.templ`), but each span also carries Tailwind display classes (`inline-flex min-w-24 ...`):

- `page_scripts.templ` (v0.13.2 and current master): `<span id="conn-state-reconnecting" hidden class="inline-flex min-w-24 ...">`

An author-level `display` declaration beats the UA stylesheet's `[hidden] { display: none }` regardless of specificity, so the attribute never hides anything as long as any Tailwind display class is present. `setPillState` flips `el.hidden`, the computed style stays `inline-flex`.

Measured (Chromium 156, v0.13.2 page served as-is):

```js
> getComputedStyle(document.getElementById("conn-state-reconnecting")).display
"flex"          // hidden attribute = true
```

Playwright's `toBeHidden()` fails on the element while it carries `hidden=""` — which is how it surfaced: an e2e guard of the pill cannot assert visibility.

## Fix options

1. Ship a `[hidden] { display: none !important; }` rule in the page's own styles (one line, fixes every future `hidden` usage too).
2. Toggle a Tailwind `hidden` class instead of the attribute in `setPillState`.
3. Drop the display classes from the three spans and style a wrapper.

Option 1 is the smallest and covers the `#health-filter`/`data-filter-empty` usages I can see in the templates as well.

Found while wiring `cqrshealth.NewDashboard` into a Playwright battery (cqrs-htmx's e2e suite asserts the pill's attribute state machine with a citation today).

💘 Generated with Crush

Assisted-by: Crush:glm-5.3
