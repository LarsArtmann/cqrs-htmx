import { chromium } from "@playwright/test";

const browser = await chromium.launch({
  executablePath: "/nix/store/fqriy9vihl23l4md56qgiks8z0d2xdlw-chromium-156.0.8078.79/bin/chromium",
  headless: true,
});
const page = await browser.newPage();
await page.setContent(`<!doctype html>
<style>
  @supports (anchor-name: --x) {
    [popover][data-tc-anchor] {
      position-area: bottom;
      inset: auto;
      margin: 8px;
    }
  }
  #trigger { position: absolute; left: 1000px; top: 20px; width: 100px; height: 30px; }
</style>
<button id="trigger" popovertarget="menu">Trigger</button>
<div id="menu" popover="auto" data-tc-anchor="trigger">Menu content</div>
<script>
  window.results = {};
</script>`);
await page.click("#trigger");
await page.waitForTimeout(200);
const res = await page.evaluate(() => {
  const menu = document.getElementById("menu");
  return {
    open: menu.matches(":popover-open"),
    rect: menu.getBoundingClientRect().toJSON(),
    note: "trigger at x=1000: if implicit anchor works, menu.x should be ~1000; if viewport fallback, ~8",
  };
});
console.log(JSON.stringify(res, null, 2));
await browser.close();
