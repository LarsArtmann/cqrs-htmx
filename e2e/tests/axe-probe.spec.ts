import { test } from "@playwright/test";
import AxeBuilder from "@axe-core/playwright";

test("probe", async ({ page }) => {
  await page.goto("/dashboard/");
  await page.waitForLoadState("networkidle");
  const results = await new AxeBuilder({ page })
    .withTags(["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"])
    .analyze();
  for (const v of results.violations) {
    if (!["critical", "serious"].includes(v.impact ?? "")) continue;
    console.log(`VIOLATION ${v.id} (${v.impact})`);
    for (const n of v.nodes) console.log(`  target: ${n.target.join(" ")} | html: ${n.html.slice(0, 120)}`);
  }
});
