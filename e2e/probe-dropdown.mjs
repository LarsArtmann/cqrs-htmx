import { chromium } from "@playwright/test";

const browser = await chromium.launch({
  executablePath: process.env.E2E_BROWSER_PATH || undefined,
  headless: true,
});
const page = await browser.newPage();
await page.goto("http://localhost:18930/dev-login", { waitUntil: "load" });
await page.waitForTimeout(300);
await page.goto("http://localhost:18930/admin/users", { waitUntil: "load" });
await page.waitForTimeout(400);

const probe = await page.evaluate(() => {
  const menu = document.getElementById("admin-user-menu-menu");
  const trigger = document.getElementById("admin-user-menu-button");
  if (!menu || !trigger) return { error: "missing elements", menu: !!menu, trigger: !!trigger };
  const cs = getComputedStyle(menu);
  return {
    supportsAnchor: CSS.supports("anchor-name: --x"),
    area: cs.getPropertyValue("position-area"),
    varArea: cs.getPropertyValue("--tc-popover-area"),
    position: cs.position,
    inset: cs.inset,
    margin: cs.margin,
    menuRect: menu.getBoundingClientRect().toJSON(),
    triggerRect: trigger.getBoundingClientRect().toJSON(),
  };
});
console.log("BEFORE OPEN:", JSON.stringify(probe, null, 2));

await page.click("#admin-user-menu-button");
await page.waitForTimeout(300);

const after = await page.evaluate(() => {
  const menu = document.getElementById("admin-user-menu-menu");
  const cs = getComputedStyle(menu);
  return {
    open: menu.matches(":popover-open"),
    area: cs.getPropertyValue("position-area"),
    inset: cs.inset,
    left: cs.left,
    top: cs.top,
    menuRect: menu.getBoundingClientRect().toJSON(),
  };
});
console.log("AFTER OPEN:", JSON.stringify(after, null, 2));

await browser.close();
