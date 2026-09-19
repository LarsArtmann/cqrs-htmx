import { test } from "@playwright/test";
import AxeBuilder from "@axe-core/playwright";

for (const p of ["events", "commands", "dead-letters", "time-travel"]) {
  test(`probe ${p}`, async ({ page }) => {
    await page.goto(`/dashboard/${p}`);
    await page.waitForLoadState("networkidle");
    const results = await new AxeBuilder({ page })
      .withTags(["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"])
      .analyze();
    for (const v of results.violations) {
      if (!["critical", "serious"].includes(v.impact ?? "")) continue;
      console.log(`VIOLATION ${v.id} (${v.impact}) on ${p}`);
      const seen = new Set<string>();
      for (const n of v.nodes) {
        const t = n.target.join(" ");
        if (!seen.has(t)) { seen.add(t); console.log(`  ${t}`); }
      }
    }
  });
}
