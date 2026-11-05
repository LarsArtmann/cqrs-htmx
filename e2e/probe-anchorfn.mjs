import { chromium } from "@playwright/test";
const browser = await chromium.launch({ executablePath: "/nix/store/fqriy9vihl23l4md56qgiks8z0d2xdlw-chromium-156.0.8078.79/bin/chromium", headless: true });

async function t(name, css) {
  const page = await browser.newPage();
  await page.setContent(`<!doctype html><style>
    @supports (anchor-name: --x) { [popover][data-tc-anchor] { ${css} } }
    #trigger { position: absolute; left: 1000px; top: 20px; width: 100px; height: 30px; }
  </style>
  <button id="trigger" popovertarget="menu">Trigger</button>
  <div id="menu" popover="auto" data-tc-anchor="trigger" data-tc-position="bottom" data-tc-align="end">Menu</div>`);
  await page.click("#trigger");
  await page.waitForTimeout(120);
  const r = await page.evaluate(() => document.getElementById("menu").getBoundingClientRect().toJSON());
  console.log(name.padEnd(40), "x:", r.x.toFixed(0).padStart(5), "right:", r.right.toFixed(0).padStart(5), "y:", r.y.toFixed(0));
  await page.close();
}

// target flush-end: right=1100, y=58
await t("anchor() end: top+right", "position-area: none; inset: auto; margin: 8px; top: anchor(bottom); right: anchor(right);");
await t("anchor() start: top+left", "position-area: none; inset: auto; margin: 8px; top: anchor(bottom); left: anchor(left);");
await t("anchor() + try-fallbacks", "position-area: none; inset: auto; margin: 8px; top: anchor(bottom); right: anchor(right); position-try-fallbacks: flip-block, flip-inline;");
await browser.close();
