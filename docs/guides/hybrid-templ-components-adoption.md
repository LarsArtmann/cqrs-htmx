# Hybrid templ-components Adoption Guide

How to adopt [templ-components](https://github.com/larsartmann/templ-components)
inside a module that renders with Go string builders (`strings.Builder` +
`fmt.Fprintf`) — without converting to templ. This is the pattern dashboardui
adopted across 15 capabilities (scored ~85/100 in the 2026-09-17 audit's
rubric), and the pitfalls it hit so you don't have to.

## The core pattern

templ components are just `templ.Component` values with a
`Render(ctx, *strings.Builder) error` method. You can call them from plain Go:

```go
func badgeHTML(ctx context.Context, text string, t display.BadgeType) string {
	var b strings.Builder

	props := display.BadgeProps{
		BaseProps: utils.BaseProps{Nonce: nonceFromCtx(ctx)},
		Text:      text,
		Type:      t,
		Dot:       true,
	}
	_ = display.Badge(props).Render(ctx, &b)

	return b.String()
}
```

Wire the request context through EVERY render helper (`renderLayout(ctx, ...)`
closures, helper functions taking `ctx` first). The library reads the CSP
nonce from the context (`templ.GetNonce`); dropping the context silently
produces nonce-less scripts that die under a strict CSP. `contextcheck` linter
enforces the threading.

## Pitfalls learned the hard way

### 1. templ `{children...}` is empty in hybrid renders

`display.Grid` and other components that take `templ.Component` children render
NOTHING when you build them standalone — children-slots expect templ's runtime
child capture, which string-builder invocation doesn't provide. Components that
take DATA props (strings, enums, slices) work fine. `display.Table` accepts
rows as data AND raw-body HTML (`templ.Raw`), both hybrid-safe.

### 2. Class families live in Go sources, not only `.templ`

Tailwind's scanner must see the class literals. The library ships them in
`*_templ.go` generated code AND `*_go.go` class-map files (e.g. errorpage's
`styles.go` emits runtime class strings). If your CSS build copies only
`.templ` files, you will miss entire utility families and ship a broken
bundle. dashboardui's flake app `build-dashboardui-css` scans `*.templ` +
`*_go.go` + errorpage `styles.go` and canary-fails on missing families.

### 3. Rebuild the CSS bundle in the same change as any family bump

The compiled bundle captures the class set of ONE library version. dashboardui
v4.10.0 shipped with a v1.17.0-era bundle after the v1.18.0 bump landed —
consumers got a visually degraded UI. Rule: `nix run .#build-dashboardui-css`
in the same commit as the bump, then re-run the screenshot pass.

### 4. String-contains tests cannot see rendering bugs

A page-level `strings.Contains(body, "42")` passes while the page also renders
`%!(EXTRA string=/dashboard)` two lines above. The screenshot pass caught a
Sprintf extra-argument leak that dozens of string tests missed. Guards that
actually work:

- render EVERY route and assert no `%!(` fmt markers plus a 200 status
  (`fmt_markers_test.go` in dashboardui);
- assert values INSIDE stable elements (`ValueID` hooks like
  `#stat-total-events`), not page-wide;
- run the Playwright screenshot + axe sweeps for what only a browser sees.

### 5. Library overrides fight your CSS

The library's components carry their own Tailwind classes; your module's
unlayered custom CSS (table cell colors, `.mono`) beats them in the cascade
and can push text under WCAG 4.5:1. Assert contrast with the axe sweep
instead of eyeballing, and fix at the token level (`--muted`) or with one
explicit override rule, not per-component forks.

### 6. nolint directives must fit on one line

golines (max length 120) will re-wrap a line and DETACH a trailing
`//nolint:<linter>` directive when the line with the reason exceeds the
limit - the directive stops applying and the lint failure returns. Keep the
reason short, or put the directive on its own line above.

## Context threading contract

- Render closures receive `ctx` from the HTTP request (`r.Context()`) and
  pass it to every component render.
- Nonce: `layout.Base`/the shell places the per-request nonce on the context;
  scripts emitted by `feedback.ToastContainer`, `htmx.GlobalErrorHandling`,
  `display.CopyButton` pick it up automatically.
- Never construct a `context.Background()` inside a render helper.

## Checklist for adopting one more capability

1. Add the component call through a thin helper (`badgeHTML`, `statCardHTML`, ...).
2. Rebuild the CSS bundle; check canaries pass.
3. Update/extend goldens (`go test ./... -update`, review the diff).
4. Add/adjust element-scoped assertions (ValueIDs, not page-wide Contains).
5. Run the screenshot + axe sweeps; fix findings at the token level.
6. If the family version changed: bump requires in the same change.
