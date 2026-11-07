// Empirical check: does `data-on:load` fire on <ul>/<div> under the
// embedded DataStar v1.0.3 client? Records whether the initial
// GET /api/todos and the SSE GET /api/events ever fire, and whether
// #todo-list gets populated without manual interaction.
import { test, expect } from "@playwright/test";

test("data-on:load firing check", async ({ page }) => {
  const fired = { todos: 0, events: 0, anyFetch: [] as string[] };
  page.on("request", (req) => {
    const u = req.url();
    if (u.includes("/api/todos")) fired.todos++;
    if (u.includes("/api/events")) fired.events++;
    if (u.includes("/api/")) fired.anyFetch.push(u.replace("http://localhost:8095", ""));
  });
  await page.goto("http://localhost:8095/", { waitUntil: "networkidle" });
  await page.waitForTimeout(2500);
  const listChildren = await page.locator("#todo-list").evaluate((el) => el.children.length);
  const statsHtml = await page.locator("#stats").evaluate((el) => el.innerHTML.trim());
  console.log(JSON.stringify({ fired, listChildren, statsLen: statsHtml.length }, null, 2));
  // Also check whether datastar signals initialized (data-signals present + $title works)
  const hasSignals = await page.evaluate(() => {
    const el = document.querySelector("[data-signals]");
    return el !== null;
  });
  console.log("hasSignals:", hasSignals);
});
