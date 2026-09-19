import { test } from "@playwright/test";

test("probe computed styles", async ({ page }) => {
  await page.goto("/dashboard/events");
  await page.waitForLoadState("networkidle");
  const info = await page.evaluate(() => {
    const span = document.querySelector("span[data-tc-copy-text]");
    if (!span) return "no span";
    const btn = span.closest("[data-tc-copy]");
    const s = getComputedStyle(span);
    const b = getComputedStyle(btn);
    const label = document.querySelector("label[for='filter-type']");
    const l = label ? getComputedStyle(label) : null;
    return {
      spanColor: s.color,
      btnBg: b.backgroundColor,
      btnClass: btn.className.slice(0, 80),
      labelColor: l?.color,
      scheme: matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light",
    };
  });
  console.log(JSON.stringify(info, null, 1));
});
