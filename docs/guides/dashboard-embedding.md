# Embedding the Dashboard Inside Your App

> **Audience:** consumers whose app already owns the page shell (nav, CSS,
> theme) and want the dashboard panels to render inside it, instead of the
> dashboard serving its own standalone pages. The mode matrix and the
> one-liners live in [`dashboardui/README.md` § Integration Modes](../../dashboardui/README.md);
> this guide is the working manual for the Embedded mode seam.

## The seam: `Config.Layout`

Without `Layout`, the dashboard renders its own page shell (sidebar, theme
toggle, document). With it, YOUR function renders the document and the
dashboard hands you the page metadata plus the ready-rendered content:

```go
type LayoutFunc func(meta PageMeta, content templ.Component) templ.Component

cfg.Layout = func(meta dashboardui.PageMeta, content templ.Component) templ.Component {
    return myShell(myPage{Title: meta.FullTitle, User: currentUser}, content)
}
```

The shell owns the ENTIRE document (html/head/body) and must:

- link `meta.CSSURLs` (or vendor equivalents) or the panels render unstyled;
- load `meta.ScriptURLs` (htmx.js, dashboard.js) — skip htmx.js if your shell
  already loads its own (never load htmx twice);
- emit `meta.Nonce` on any script tags when a nonce CSP is in play;
- include a `feedback.ToastContainer` — write actions surface toasts via the
  `dashboardui:toast` HX-Trigger event.

`PageMeta` additionally carries `Title`/`FullTitle`/`Brand`, `BasePath`,
`AccentColor`, `ReadOnly`, `Capabilities`, and `Nav` — the navigation model
the built-in sidebar would have rendered, capability items first, consumer
extras last.

## `NavLink` and the `Verbatim` contract

`PageMeta.Nav` is `[]NavLink`. Every entry carries `Label`, `Href`, `Icon`,
and `Active` (the dashboard resolves active pages itself — an entry is active
when the request path is the page it links to).

The one field a custom layout MUST respect is `Verbatim`:

- **Built-in entries are `Verbatim == false`** — their `Href` is relative to
  the dashboard's own mount and a shell may rebase or prefix it.
- **`Config.ExtraNav` entries are `Verbatim == true`** — the consumer
  declared them with absolute hrefs that render EXACTLY as written, never
  BasePath-prefixed. A sibling surface mounted next door stays one click
  away; prefixing would silently break it.

```go
for _, link := range meta.Nav {
    href := link.Href
    if !link.Verbatim {
        href = path.Join(meta.BasePath, href) // your shell's choice
    }
    // render <a href=... class=active?>…
}
```

If your shell ignores `Verbatim`, extra entries still WORK in the built-in
sidebar — but every consumer that mounts the shell elsewhere sees broken
links. The escape assertion (`TestExtraNav_EscapesLabelAndHref`) and the
verbatim proof (`TestExtraNav_RendersInBuiltInSidebarWithVerbatimHref`) pin
both halves in-tree.

## Consumer nav: `Config.ExtraNav`

`ExtraNav []NavEntry` appends your entries after the capability items:

```go
cfg.ExtraNav = []dashboardui.NavEntry{
    {Label: "Admin", Href: "/admin/", Icon: icons.ShieldCheck},
    {Label: "Docs", Href: "https://docs.example.com"}, // Icon defaults to icons.Link
}
```

- Hrefs render verbatim in the built-in sidebar and arrive `Verbatim: true`
  in `PageMeta.Nav` for custom layouts.
- An entry pointing AT a dashboard page (e.g. `/cqrs/events` under
  `BasePath: /cqrs`) gets `Active` on that page exactly like a built-in item;
  sibling-surface entries are never active.
- Labels and hrefs are templ-escaped (attribute context) — but treat them as
  developer config, not user input.

## HTMX partials bypass the shell

Panel refreshes are HTMX partial requests; they swap fragments and never
re-render your document, so `Layout` is only invoked on full-page visits. No
shell work is needed for live updates — the panels manage their own polling
and SSE subscriptions. The two exceptions that always keep the built-in
shell: the error page and the 404 page (a broken document must not depend on
a possibly-broken consumer shell).

## CSP and theme notes

- The embedded panels carry no inline handlers or inline styles (CSP-safe
  under a nonce policy); your shell provides the nonce if it uses one.
- Dark mode is class-driven (`html.dark`) in the built-in shell. In Embedded
  mode YOUR document owns the `<html>` element — apply the same class
  contract (or your own theming) so the panels' `dark:` variants match your
  chrome.

## Related

- [`dashboardui/README.md` § Integration Modes](../../dashboardui/README.md) — the mode matrix and `Autodetect`.
- [ADR-0056](../adr/0056-frontend-sync-protocol.md) — the frontend sync protocol the panels' live updates ride.
- `setup.Config.DashboardLayout` — the `setup` bundle's pass-through for this same seam.
