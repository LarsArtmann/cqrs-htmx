import { test, expect, Page } from "@playwright/test";

/**
 * F#32 browser gate: the user-detail credential-restore card (F170, T23
 * hand-off) — the card adminui/users.templ renders under
 * `templ credentialRestoreCard` (title "Restore a passkey from backup",
 * paste-textarea `#credential-backup-json` name=backup, submit button
 * "Restore credential" POSTing `{BasePath}/users/{id}/credentials/restore`,
 * danger data-confirm carrying the escalation copy).
 *
 * WHAT THE BATTERY CAN HONESTLY PROVE (negative/gating path):
 * The card is gated on DEPLOYMENT capability, not per-user credential state:
 * `d.AuthMethods.HasWebAuthn` is `svc.HasWebAuthn()` — "a WebAuthn provider is
 * configured" (adminui/models.go:312, usermgmt/service_auth_methods.go). The
 * same probe hides the whole passkey surface: the Passkeys card, the
 * restore card, and the "Credentials" stat tile (users.templ ~600-660).
 *
 * No battery server pairs the panel with a WebAuthn provider, so the card can
 * NEVER render here:
 *  - admin-demo (:18930) mounts the panel with a TOTP provider ONLY
 *    (examples/admin-demo/main.go: ServiceConfig{AuditLog, TOTP, TokenPepper}
 *    — no WebAuthn field);
 *  - e2e/server (:18923) configures a REAL WebAuthn provider (server/login.go)
 *    but mounts no adminui panel at all.
 *
 * The demo DOES hold a genuine WebAuthnCredential in its store for
 * erin@acme.dev (examples/admin-demo/seed.go: seedEquippedUser →
 * svc.AddCredential, id "admin-demo-passkey-erin-01"), so the second test
 * below pins the sharpest honest fact available: a user whose STORE carries a
 * passkey still gets NO restore card, because the deployment cannot serve
 * WebAuthn — the probe hides dead surfaces even when data exists ("dead cards
 * never render", users.templ). A fresh self-seeded user (the first test)
 * proves the same gate on an account no other session touches, with the
 * TOTP definition row as the live positive control proving the capability
 * probe itself works (the demo HAS TOTP, so "TOTP MFA" renders).
 *
 * WHY THE POSITIVE PATH (card renders + escalation copy + textarea/submit
 * wiring + a real POST restore) IS NOT ASSERTED: it needs a battery server
 * that mounts the panel WITH `WebAuthn:` configured — a change to
 * examples/admin-demo/main.go or e2e/server, both outside the e2e specs'
 * scope. Playwright's CDP virtual authenticator (the working pattern lives in
 * loginpage-webauthn.spec.ts enableVirtualAuthenticator) could drive a genuine
 * enrollment once such a server exists, and the restore route itself needs no
 * browser ceremony at all (adminui/handler_credential_restore.go unmarshals
 * the backup JSON and calls Service.AddCredential directly). We do NOT fake
 * the card by injecting its markup into the DOM — a fake positive would test
 * nothing the panel ships.
 *
 * Same recipe as admin-csrf.spec.ts: SERIAL mode, dev-login beforeEach,
 * gotcha-66 browser env from the runner. Gotcha 59: erin@acme.dev is a SHARED
 * seed — assert only facts no other session can change (email presence,
 * deployment-gate absences, the demo's static TOTP provider); the fresh user
 * is self-seeded per run.
 */
test.describe.configure({ mode: "serial" });

const ADMIN = "http://localhost:18930";

test.beforeEach(async ({ page }) => {
  await page.goto(`${ADMIN}/dev-login`, { waitUntil: "load" });
  await page.waitForTimeout(500);
});

/** The request CSRF token lives in the layout's meta tag. */
async function csrf(page: Page): Promise<string> {
  const token = await page.evaluate(() =>
    document.querySelector('meta[name="csrf-token"]')?.getAttribute("content"),
  );
  expect(token, "panel layout should render the csrf meta").toBeTruthy();
  return token ?? "";
}

/** Form-POST a panel endpoint with the CSRF header; returns the response. */
async function postForm(
  page: Page,
  path: string,
  fields: Record<string, string>,
): Promise<{ ok: boolean; status: number; location: string }> {
  const token = await csrf(page);
  const body = Object.entries(fields)
    .map(([k, v]) => `${k}=${encodeURIComponent(v)}`)
    .join("&");
  const res = await page.request.post(`${ADMIN}${path}`, {
    headers: {
      "Content-Type": "application/x-www-form-urlencoded",
      "X-Csrf-Token": token,
    },
    data: body,
  });
  return { ok: res.ok(), status: res.status(), location: res.url() };
}

/**
 * Navigate from the users list to a user's detail page by row content, the
 * way a real admin would (search view is page-independent under pagination).
 */
async function openUserDetail(page: Page, email: string): Promise<void> {
  await page.goto(`${ADMIN}/admin/users?q=${encodeURIComponent(email)}`, {
    waitUntil: "load",
  });
  await page.waitForTimeout(400);
  const row = page.locator("#users-table tbody tr", { hasText: email });
  await expect(row, `user ${email} should be findable via the search view`).toHaveCount(1);
  await row.locator("a").first().click();
  await page.waitForLoadState("load");
  await page.waitForTimeout(400);
  await expect(page.locator("h1, h2").first()).toContainText(email);
}

/** The restore card, when the deployment can serve it, renders exactly these. */
const RESTORE_FORM = 'form[action$="/credentials/restore"]';

test("credential restore: no card on the panel that lacks a WebAuthn provider (fresh user)", async ({
  page,
}) => {
  const stamp = Date.now();
  const email = `e2e-restore-${stamp}@acme.dev`;
  const created = await postForm(page, "/admin/users", {
    email,
    display_name: "E2E Restore Gate",
  });
  expect(created.ok, `seed user: ${created.status}`).toBeTruthy();

  await openUserDetail(page, email);

  // Positive control — the capability probe is LIVE on this page: the demo
  // configures a TOTP provider, so the Details definition list carries the
  // "TOTP MFA" row (adminui/models.go userDetailDefinitionItems). If this
  // ever disappears, the absences below are no longer evidence of gating.
  await expect(page.getByText("TOTP MFA").first()).toBeVisible();

  // The gated surfaces: no restore card (form + title), no Passkeys card
  // (its empty-state copy), no "Credentials" stat tile. Absence is scoped to
  // the exact card fixtures, not whole-page text, so unrelated copy (e.g. the
  // cannot-sign-in banner's "no passkey" wording) cannot false-negative it.
  await expect(page.locator(RESTORE_FORM)).toHaveCount(0);
  await expect(page.getByText("Restore a passkey from backup")).toHaveCount(0);
  await expect(page.getByText("No passkeys enrolled")).toHaveCount(0);
  await expect(page.getByText("Credentials", { exact: true })).toHaveCount(0);

  // And the page is otherwise healthy: the details card rendered.
  await expect(page.getByText("User ID").first()).toBeVisible();
});

test("credential restore: a stored passkey does NOT summon the card — the gate is deployment capability", async ({
  page,
}) => {
  // erin@acme.dev is the demo's fully-equipped seed: her aggregate carries a
  // real WebAuthnCredential (seed.go AddCredential). If the gate were
  // per-user credential state, her page would render the card — it must not,
  // because :18930 has no WebAuthn provider.
  await openUserDetail(page, "erin@acme.dev");

  await expect(page.getByText("TOTP MFA").first()).toBeVisible();
  await expect(page.locator(RESTORE_FORM)).toHaveCount(0);
  await expect(page.getByText("Restore a passkey from backup")).toHaveCount(0);
  await expect(page.getByText("No passkeys enrolled")).toHaveCount(0);
  await expect(page.getByText("Credentials", { exact: true })).toHaveCount(0);
});
