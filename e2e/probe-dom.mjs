import { chromium } from "@playwright/test";
const browser = await chromium.launch({ executablePath: "/nix/store/fqriy9vihl23l4md56qgiks8z0d2xdlw-chromium-156.0.8078.79/bin/chromium", headless: true });
const page = await browser.newPage();
await page.goto("http://localhost:18930/dev-login", { waitUntil: "load" });
await page.waitForTimeout(300);
await page.goto("http://localhost:18930/admin/users", { waitUntil: "load" });
await page.waitForTimeout(400);
const res = await page.evaluate(() => {
  const menu = document.getElementById("admin-user-menu-menu");
  const trigger = document.getElementById("admin-user-menu-button");
  const invokers = [...document.querySelectorAll("[popovertarget]")].map(el => ({
    id: el.id, pt: el.getAttribute("popovertarget"), tag: el.tagName, display: getComputedStyle(el).display,
  }));
  return {
    menuOuterStart: menu.outerHTML.slice(0, 300),
    triggerOuter: trigger.outerHTML.slice(0, 400),
    invokers,
    dupMenuIds: document.querySelectorAll("#admin-user-menu-menu").length,
    dupTriggerIds: document.querySelectorAll("#admin-user-menu-button").length,
  };
});
console.log(JSON.stringify(res, null, 2));
await browser.close();
