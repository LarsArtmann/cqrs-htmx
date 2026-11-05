import { chromium } from "@playwright/test";
const browser = await chromium.launch({ executablePath: "/nix/store/fqriy9vihl23l4md56qgiks8z0d2xdlw-chromium-156.0.8078.79/bin/chromium", headless: true });

async function testCase(name, css) {
  const page = await browser.newPage();
  await page.setContent(`<!doctype html><style>
    @supports (anchor-name: --x) {
      [popover][data-tc-anchor] { ${css} }
    }
    #trigger { position: absolute; left: 1000px; top: 20px; width: 100px; height: 30px; }
  </style>
  <button id="trigger" popovertarget="menu">Trigger</button>
  <div id="menu" popover="auto" data-tc-anchor="trigger" data-tc-position="bottom" data-tc-align="end">Menu</div>`);
  await page.click("#trigger");
  await page.waitForTimeout(150);
  const rect = await page.evaluate(() => document.getElementById("menu").getBoundingClientRect().toJSON());
  console.log(name, "-> x:", rect.x.toFixed(0), "y:", rect.y.toFixed(0));
  await page.close();
}

await testCase("plain bottom", "position-area: bottom; inset: auto; margin: 8px;");
await testCase("compound span-inline-start", "position-area: bottom span-inline-start; inset: auto; margin: 8px;");
await testCase("var indirection", "[data-tc-position=bottom][data-tc-align=end] { --tc-popover-area: bottom span-inline-start; } [popover][data-tc-anchor] { position-area: var(--tc-popover-area, bottom); inset: auto; margin: 8px; }");
await browser.close();
