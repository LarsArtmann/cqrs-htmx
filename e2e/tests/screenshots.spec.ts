import { test } from "@playwright/test";

/**
 * Dashboard screenshot harness (Run-3 N5): captures every dashboard page in
 * light, dark (prefers-color-scheme emulation), and mobile (375x812) so the
 * templ-components adoption is actually SEEN, not just string-asserted.
 *
 * Output: docs/screenshots/{light,dark,mobile}/dashboard-<page>.png
 * Run: pnpm dlx playwright test screenshots.spec.ts
 */

const PAGES = [
  "overview",
  "events",
  "aggregates",
  "commands",
  "queries",
  "projections",
  "dead-letters",
  "time-travel",
  "snapshots",
];

const MODES = [
  { name: "light", colorScheme: "light" as const, viewport: { width: 1280, height: 900 } },
  { name: "dark", colorScheme: "dark" as const, viewport: { width: 1280, height: 900 } },
  { name: "mobile", colorScheme: "light" as const, viewport: { width: 375, height: 812 } },
];

for (const mode of MODES) {
  test.describe(`dashboard screenshots (${mode.name})`, () => {
    test.use({ colorScheme: mode.colorScheme, viewport: mode.viewport });

    for (const pageName of PAGES) {
      test(`captures ${pageName}`, async ({ page }) => {
        const url = pageName === "overview" ? "/dashboard/" : `/dashboard/${pageName}`;
        const resp = await page.goto(url);
        if (resp?.status() !== 200) {
          test.skip(true, `${url} unavailable (${resp?.status()})`);
          return;
        }

        await page.waitForLoadState("networkidle");
        await page.screenshot({
          path: `../docs/screenshots/${mode.name}/dashboard-${pageName}.png`,
          fullPage: true,
        });
      });
    }
  });
}
