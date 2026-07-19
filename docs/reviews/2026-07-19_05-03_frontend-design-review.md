# Frontend Design Review — adminui + loginpage

**Date:** 2026-07-19 | **Scope:** Default UI shipped by the library for consumers to embed.

## The brief (clarified)

These components have an unusual design constraint: they are **library defaults that consumers will rebrand**, not end products. Distinctive visual personality is _wrong_ here — consumers would have to fight it. The right brief is: quiet, professional, accessible, easy to skin via CSS variables, structurally correct.

Evaluated against that brief, both components largely succeed. This review notes what's working and the small set of changes that would raise the floor without overstepping the library-default role.

## adminui (templ + Tailwind v4)

### What's good

- **Theming via CSS variables:** `--accent`, `--sidebar-bg`, `--surface` are all consumer-overridable via `Config.AccentColor`. The injection point `<style>:root{--accent:{p.Accent}}</style>` is the right shape.
- **Responsive layout:** `grid-cols-[248px_1fr] max-md:grid-cols-1` + mobile hamburger (`admin-toggle`) + scrim (`admin-scrim`) + transform-based slide-in. Correct and complete.
- **Semantic HTML:** `<aside>`, `<nav>`, `<header>`, `<main>` used appropriately. ARIA-label on the menu button.
- **SSE sync indicator:** honest-UI pattern — only renders if `p.SSEURL != ""`. Pairs with `data-sync-status` attribute that JS can drive.
- **Subtle brand mark:** "cqrs-htmx admin" in the sidebar footer at 55% opacity — quiet, doesn't compete with the consumer's brand.
- **Backdrop blur header** with `color-mix` for translucency — modern, cross-browser.

### Improvement candidates (judgment-applied)

| Priority | Issue                                                                                                                                                                                                  | Fix                                                                                                      |
| -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | -------------------------------------------------------------------------------------------------------- |
| Medium   | **No visible focus states** in the layout file. Keyboard users navigating the sidebar cannot see where they are.                                                                                       | Add `focus-visible:outline-2 outline-offset-2 outline-[var(--accent)]` to nav links.                     |
| Medium   | **No `prefers-reduced-motion` guard** on the sidebar slide-in transition (`max-md:transition-transform`).                                                                                              | Wrap in `motion-safe:` Tailwind variant or add a media query in `admin-tw.css`.                          |
| Low      | The `style="background:var(--sidebar-bg)"` (and similar inline styles) work but inline styles have higher specificity than utility classes — consumers trying to override via stylesheet may struggle. | Move defaults into `admin-tw.css` via `:root` variables. Inline styles only for the consumer-set values. |
| Low      | The accent color is injected via string concatenation (`<style>:root{--accent:{p.Accent}}</style>`). Works for trusted input but bypasses CSS validation.                                              | Already documented as consumer-trusted; acceptable.                                                      |

### What NOT to change

- The color palette is intentionally neutral (grays + accent) because consumers need to rebrand. Adding more personality would be Verschlimmbesserung.
- Tailwind v4 + compiled `admin-tw.css` is correct for a library (no runtime build needed for consumers).

## loginpage (self-contained CSS)

### What's good

- **CSS variables scoped with `--lp-` prefix** — won't collide with consumer styles even when inlined. Good namespacing discipline.
- **`noindex` meta** — correct for a login page (don't index the auth surface).
- **`color-scheme: light dark`** + explicit light/dark variables in CSS.
- **Accessibility primitives:** `role="alert" aria-live="polite"` on the error region; `autocomplete="email"` on inputs; `autofocus` on the primary email field.
- **Progressive disclosure:** register form hidden behind a toggle, so the primary "Sign in with passkey" gets visual priority.
- **No-auth state:** when neither WebAuthn nor OAuth2 is configured, shows a clear message with muted explanation. Honest empty state.
- **Inline CSS/JS** as a fallback (`p.inlineCSS`, `p.inlineJS`) when `CSSPath` is empty — the page works even if the consumer doesn't serve the CSS asset.

### Improvement candidates

| Priority | Issue                                                                                                                                                              | Fix                                                                                                                             |
| -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------- |
| Medium   | The first-rune favicon (`firstRune(p.Brand)`) is clever for ASCII brands but produces broken glyphs for emoji/multi-byte brand names.                              | Validate or document; consider an SVG fallback when the rune is not printable.                                                  |
| Medium   | **No visible focus ring** on the form inputs and buttons. `:focus-visible` outline missing from `login.css`.                                                       | Add `input:focus-visible, button:focus-visible, a:focus-visible { outline: 2px solid var(--lp-accent); outline-offset: 2px; }`. |
| Low      | Error styling (`--lp-error-bg: #fef2f2`) is hardcoded to light mode. In dark `color-scheme`, the error region will look washed out.                                | Add `@media (prefers-color-scheme: dark)` overrides for the error region, or use `color-mix` against the surface variable.      |
| Low      | `--lp-radius: 14px` is good for the card but inputs inherit `--lp-radius` too. Inputs typically want a smaller radius (4-8px) to read as form fields, not buttons. | Split into `--lp-card-radius` and `--lp-input-radius`.                                                                          |
| Low      | The "or" divider between passkey and OAuth2 sections uses a centered text-over-line pattern. Works but is a generic SSO look. Acceptable for a library default.    | Leave as-is — consumers will restyle.                                                                                           |

### What NOT to change

- The single-column card layout is correct for a login page. Don't add a hero image, marketing copy, or split-screen layout — those would be wrong for a library default that consumers embed at arbitrary paths.

## Copy review (per skill: "Words appear to make it easier to use")

### loginpage

- **"Sign in with passkey"** — accurate, action-named. Good.
- **"Don't have an account? Create one"** — conversational, active. Good.
- **"Your browser does not support passkeys. Use a modern browser (Chrome, Firefox, Safari, or Edge) or try SSO if available."** — specific, actionable, tells the user how to fix. Good.
- **"No authentication method is configured. Set up WebAuthn or OAuth2 in your ServiceConfig to enable login."** — this message is for the developer, not the end-user. It will be seen by end-users if the deployer forgot to configure auth. Consider rephrasing to "Login is not yet available. Please contact the site administrator." with the developer-facing detail in HTML comment or server log.

### adminui

- **"cqrs-htmx admin"** footer — quiet brand mark, fine.
- **"Synced"** in the SSE bar — calm, present tense. The sync dot is a stronger signal than the text.
- **"Sign out"** — action-named. Better than "Logout" or "Log out" for the consistent verb-noun pattern.

## Overall verdict

**adminui: 7.5/10** — solid library default. Add focus states and reduced-motion guards; leave the rest alone.
**loginpage: 8/10** — better accessibility primitives out of the box. Fix the focus ring, split the radius variable, rephrase the dev-facing message.

Neither should pursue visual distinction — that would be wrong for a library default. Restraint is the correct mode here.
