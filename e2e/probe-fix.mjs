import { chromium } from "@playwright/test";
const browser = await chromium.launch({ executablePath: "/nix/store/fqriy9vihl23l4md56qgiks8z0d2xdlw-chromium-156.0.8078.79/bin/chromium", headless: true });
const page = await browser.newPage();
await page.setContent(`<!doctype html><style>
  @supports (anchor-name: --x) {
    [popover][data-tc-anchor] { position-area: var(--tc-popover-area, bottom); inset: auto; margin: 8px; position-try-fallbacks: flip-block, flip-inline; }
    [data-tc-position="bottom"] { --tc-popover-area: bottom; }
    [popover][data-tc-anchor][data-tc-align="start"] { justify-self: start; }
    [popover][data-tc-anchor][data-tc-align="end"] { justify-self: end; }
  }
  #trigger { position: absolute; left: 1000px; top: 20px; width: 100px; height: 30px; }
</style>
<button id="trigger" popovertarget="menu">Trigger</button>
<div id="menu" popover="auto" data-tc-anchor="trigger" data-tc-position="bottom" data-tc-align="end">Menu</div>`);
await page.click("#trigger");
await page.waitForTimeout(150);
const res = await page.evaluate(() => {
  const m = document.getElementById("menu");
  const r = m.getBoundingClientRect();
  const t = document.getElementById("trigger").getBoundingClientRect();
  return { menu: { x: r.x, right: r.right, y: r.y }, trigger: { x: t.x, right: t.right, bottom: t.bottom },
    verdict: Math.abs(r.right - t.right) < 2 && Math.abs(r.y - t.bottom - 8) < 2 ? "FLUSH-END-CORRECT" : "MISS" };
});
console.log(JSON.stringify(res, null, 2));
await browser.close();
