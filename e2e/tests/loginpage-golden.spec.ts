import { expect, test } from "@playwright/test";
import { readFileSync, writeFileSync } from "node:fs";
import { resolve } from "node:path";

/**
 * loginpage full-page render golden (plan W10 / D11 / S57): pins the
 * complete rendered DOM of GET /login — structure, stable lp-* IDs, the
 * loginpage-config JSON contract, and the inline WebAuthn JS presence.
 *
 * Nonce- and CSRF-insensitive by construction: both values are normalized
 * before comparison (the server pins a fixed nonce for determinism, but the
 * golden must not depend on it — a consumer with real per-request nonces
 * renders the same page).
 *
 * Regenerate after an INTENTIONAL loginpage render change:
 *   WRITE_GOLDEN=1 npx playwright test loginpage-golden
 * then diff the file by hand before committing.
 */

const GOLDEN_PATH = resolve(__dirname, "__snapshots__/loginpage-render.golden.html");

test("login page full render golden", async ({ page }) => {
  await page.goto("/login");

  // The registration section is hidden on load (RegisterFirst=false) — the
  // golden pins the DEFAULT view; the toggle behavior is covered by the
  // WebAuthn spec.
  await expect(page.locator("#lp-login-section")).toBeVisible();

  const html = await page.evaluate(() => document.documentElement.outerHTML);

  const normalized = html
    .replace(/nonce="[^"]*"/g, 'nonce="<NONCE>"')
    .replace(/name="csrf_token" value="[^"]*"/g, 'name="csrf_token" value="<CSRF>"')
    .replace(/name="csrf-token" content="[^"]*"/g, 'name="csrf-token" content="<CSRF>"');

  if (process.env.WRITE_GOLDEN === "1") {
    writeFileSync(GOLDEN_PATH, normalized + "\n");
    test.skip(true, "golden written — review the diff and commit");
  }

  const golden = readFileSync(GOLDEN_PATH, "utf8");
  expect(normalized + "\n").toBe(golden);
});
