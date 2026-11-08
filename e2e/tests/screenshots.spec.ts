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

        // A single non-200 under machine load (sibling sessions compiling)
        // is transient — retry a few times before skipping so the harness
        // only skips when the page is REALLY unavailable. The old
        // single-shot skip masked the axe findings for weeks (2026-11-08).
        let resp = await page.goto(url);
        for (let attempt = 0; attempt < 3 && resp?.status() !== 200; attempt++) {
          await page.waitForTimeout(1_000);
          resp = await page.goto(url);
        }
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
