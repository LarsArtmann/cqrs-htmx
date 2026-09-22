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
