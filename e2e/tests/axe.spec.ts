import { test, expect } from "@playwright/test";
import AxeBuilder from "@axe-core/playwright";

/**
 * Accessibility sweep over the dashboard pages (Run-3 N9). Audits the real
 * rendered DOM with axe-core; fails on new critical/serious violations.
 * Known/accepted findings are listed in ACCEPTED with a reason so the sweep
 * stays a regression gate rather than a zero-tolerance wall.
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

// Accepted violations: id(s) + why we ship them today.
const ACCEPTED = new Set<string>([
  // (none yet - first sweep baseline; populate with reasons, not silently)
]);

for (const pageName of PAGES) {
  test(`axe sweep: ${pageName}`, async ({ page }) => {
    const url = pageName === "overview" ? "/dashboard/" : `/dashboard/${pageName}`;
    await page.goto(url);
    await page.waitForLoadState("networkidle");

    const results = await new AxeBuilder({ page })
      .withTags(["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"])
      .analyze();

    const serious = results.violations.filter((v) =>
      ["critical", "serious"].includes(v.impact ?? ""),
    );

    const unexpected = serious.filter((v) => !ACCEPTED.has(v.id));
    if (unexpected.length > 0) {
      const summary = unexpected
        .map(
          (v) =>
            `${v.id} (${v.impact}): ${v.nodes.length} node(s), e.g. ${v.nodes[0]?.target.join(" ")}`,
        )
        .join("; ");
      test.info().annotations.push({
        type: "axe findings",
        description: summary,
      });
    }

    expect(unexpected, `axe critical/serious violations on ${pageName}`).toEqual([]);
    void ACCEPTED; // keep the set referenced for future entries
  });
}

/**
 * Dark-mode sweep (round-12 post-theme follow-up): the class-driven theme
 * flips the token block under html.dark, so contrast findings differ per
 * mode. Forces dark through ThemeScript's own storage key, asserts the
 * class landed pre-paint plus the toggle's synced aria-checked state, then
 * runs the same critical/serious axe gate.
 */
for (const pageName of PAGES) {
  test(`axe sweep (dark): ${pageName}`, async ({ page }) => {
    const url = pageName === "overview" ? "/dashboard/" : `/dashboard/${pageName}`;
    await page.addInitScript(() => {
      try {
        localStorage.setItem("theme", "dark");
      } catch {
        /* storage blocked — sweep still verifies via the class assertion */
      }
    });
    await page.goto(url);
    await page.waitForLoadState("networkidle");

    await expect(page.locator("html")).toHaveClass(/dark/);
    const toggle = page.locator("[data-theme-toggle]").first();
    if ((await toggle.count()) > 0) {
      await expect(toggle).toHaveAttribute("aria-checked", "true");
    }

    const results = await new AxeBuilder({ page })
      .withTags(["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"])
      .analyze();

    const serious = results.violations.filter((v) =>
      ["critical", "serious"].includes(v.impact ?? ""),
    );
    const unexpected = serious.filter((v) => !ACCEPTED.has(v.id));
    expect(unexpected, `axe dark-mode violations on ${pageName}`).toEqual([]);
  });
}

/**
 * Toggle contract: clicking the switch flips html.dark, persists the choice
 * under ThemeScript's 'theme' key, and keeps aria-checked in sync. Runs with
 * an emulated light scheme so the starting state is deterministic.
 */
test("theme toggle flips dark mode and aria-checked", async ({ page }) => {
  await page.emulateMedia({ colorScheme: "light" });
  await page.goto("/dashboard/");
  await page.waitForLoadState("networkidle");

  const toggle = page.locator("[data-theme-toggle]").first();
  await expect(toggle).toHaveAttribute("aria-checked", "false");

  await toggle.click();
  await expect(page.locator("html")).toHaveClass(/dark/);
  await expect(toggle).toHaveAttribute("aria-checked", "true");
  expect(await page.evaluate(() => localStorage.getItem("theme"))).toBe("dark");

  await toggle.click();
  await expect(page.locator("html")).not.toHaveClass(/dark/);
  await expect(toggle).toHaveAttribute("aria-checked", "false");
  expect(await page.evaluate(() => localStorage.getItem("theme"))).toBe("light");
});
