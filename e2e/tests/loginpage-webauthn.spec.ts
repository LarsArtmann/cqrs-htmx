import { expect, test, type CDPSession, type Page } from "@playwright/test";

/**
 * loginpage WebAuthn E2E (plan W10 / D11 / S57) — the only browser-level
 * proof that the rendered login page drives the full ceremonies:
 *
 *   register → registerBegin → navigator.credentials.create → registerFinish
 *   login    → loginBegin    → navigator.credentials.get    → loginFinish
 *
 * The ceremonies run against a REAL go-webauthn server (e2e/server login.go)
 * through Playwright's CDP virtual authenticator, so the attestation and
 * assertion bytes are genuine cryptography, not fixtures. The ADR-0055
 * register-issues-cookie dependency is exercised exactly as in production:
 * every fetch rides credentials: "same-origin".
 */

/** Attach a CTAP2 platform virtual authenticator to the page. */
async function enableVirtualAuthenticator(page: Page): Promise<CDPSession> {
  const cdp = await page.context().newCDPSession(page);
  await cdp.send("WebAuthn.enable");
  await cdp.send("WebAuthn.addVirtualAuthenticator", {
    options: {
      protocol: "ctap2",
      transport: "internal",
      hasResidentKey: true,
      hasUserVerification: true,
      isUserVerified: true,
      automaticPresenceSimulation: true,
    },
  });
  return cdp;
}

test.describe("loginpage WebAuthn", () => {
  test("register then login through the rendered page", async ({ page }) => {
    await enableVirtualAuthenticator(page);

    const email = `w10-${Date.now()}@example.com`;

    // --- Registration: the page's "Create one" toggle reveals the form ---
    await page.goto("/login");
    // The CSRF meta only exists on /login pages — capture the masked token
    // while we are here; it stays valid for the whole cookie session.
    const csrf = await page.locator('meta[name="csrf-token"]').getAttribute("content");

    await page.locator("#lp-show-register").click();
    await expect(page.locator("#lp-register-section")).not.toHaveClass(/hidden/);

    await page.locator("#lp-reg-email").fill(email);
    await page.locator("#lp-reg-name").fill("W10 E2E");
    await page.locator("#lp-register-btn").click();

    // Success redirects to the session-gated landing page, which renders
    // the email from the REAL session cookie — not just a URL change.
    await expect(page.locator("#post-login-email")).toHaveText(email, { timeout: 30_000 });
    await expect(page).toHaveURL(/\/post-login$/);

    // --- Logout (clears the register-issued session so login is real) ---
    await page.request.post("/auth/logout", {
      headers: { "X-CSRF-Token": csrf ?? "" },
    });
    await page.goto("/post-login");
    await expect(page).toHaveURL(/\/login$/); // session gone → bounced to /login

    // --- Login with the credential the registration created ---
    await page.locator("#lp-email").fill(email);
    await page.locator("#lp-login-btn").click();
    await expect(page.locator("#post-login-email")).toHaveText(email, { timeout: 30_000 });
  });

  test("registration errors surface in the alert region", async ({ page }) => {
    await enableVirtualAuthenticator(page);

    await page.goto("/login");
    await page.locator("#lp-show-register").click();
    // Duplicate email: user already exists from the previous test's
    // registration is not guaranteed (test isolation), so register a user
    // through the API first to force the conflict deterministically.
    const email = `dup-${Date.now()}@example.com`;
    const csrf = await page.locator('meta[name="csrf-token"]').getAttribute("content");
    await page.request.post("/auth/register", {
      headers: { "X-CSRF-Token": csrf ?? "" },
      data: { email, display_name: "Dup" },
    });

    await page.locator("#lp-reg-email").fill(email);
    await page.locator("#lp-reg-name").fill("Dup");
    await page.locator("#lp-register-btn").click();

    await expect(page.locator("#lp-error")).not.toHaveClass(/hidden/);
    await expect(page.locator("#lp-error")).toContainText(/.+/);
    await expect(page).toHaveURL(/\/login$/); // no redirect on failure
  });
});
