# loginpage v4.14.0 — pre-staged release notes (next train)

> Pre-staged per TODO row 98 (round-36 §f24): the §g3 train-timing decision
> converts to execution in one step. Verified 2026-11-07: loginpage did NOT
> ride the v4.17.0 family train (last tag `loginpage/v4.13.0`, 2026-10-14);
> the dashboardui W14 half of the row DID ride it (`dashboardui/v4.17.0`) and
> is therefore not re-drafted here. Every delta below has a receipt in
> [`loginpage/CHANGELOG.md`](../loginpage/CHANGELOG.md) § [Unreleased].

## v4.14.0 — ready when the train rolls

**New**

- **`Config.NoWebAuthn`** — force-hide the passkey form on THIS page while the
  shared `*usermgmt.Service` keeps WebAuthn for other surfaces. With
  `AutoRedirect` + a sole provider, the page becomes a pure OAuth2 entry point
  (302 straight to the begin URL). Mutually exclusive with `NoOAuth2`
  (rejected at `New`).
- **`Handler.Routes()`** — the page's route-posture manifest (authorization
  L4): one public `GET /` row, 405 for other methods.

**Changed**

- **`Config.AuthPrefix` is validated at `New`** — must be empty or
  root-relative (leading `/`, no `//host`); bare-relative prefixes mis-resolve
  against the page URL. Previously silently accepted.

**Fixed**

- **WebAuthn ceremonies work against real usermgmt responses** — the embedded
  JS now unwraps the `{"publicKey": {...}}` envelope the real
  `/auth/webauthn/*/begin` endpoints return (bare objects keep working);
  previously every real ceremony crashed in the browser. Found by the page's
  first browser-level proof (`e2e/tests/loginpage-webauthn.spec.ts`, Playwright
  virtual authenticator).
- **`NewPageData` populates Nonce, Theme, CSRF, and the error message** from
  the request — embedded forms no longer break under nonce-CSP or lose SSR
  theming.
- **Favicon renders for hex accents** — the inline SVG data-URI is
  percent-encoded (the raw `#` truncated every hex accent color at the
  fragment boundary).

## Train recipe (unchanged rhythm)

```console
nix run .#wave-tag -- loginpage v4.14.0          # dry-run, then --execute
bash scripts/tools/verify-remote-tags.sh 'loginpage/v4.14.0'
```

Sweep expectation: `setup` requires loginpage (bump to the new tag rides the
wave-tag sweep); nothing else in-tree pins it directly.
