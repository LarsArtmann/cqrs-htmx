import { chromium } from "@playwright/test";
const browser = await chromium.launch({ executablePath: "/nix/store/fqriy9vihl23l4md56qgiks8z0d2xdlw-chromium-156.0.8078.79/bin/chromium", headless: true });

async function t(name, panelCSS, extraCSS = "") {
  const page = await browser.newPage();
  await page.setContent(`<!doctype html><style>
    @supports (anchor-name: --x) {
      [popover][data-tc-anchor] { ${panelCSS} }
      ${extraCSS}
    }
    #trigger { position: absolute; left: 1000px; top: 20px; width: 100px; height: 30px; }
  </style>
  <button id="trigger" popovertarget="menu">Trigger</button>
  <div id="menu" popover="auto" data-tc-anchor="trigger" data-tc-position="bottom" data-tc-align="end">Menu</div>`);
  await page.click("#trigger");
  await page.waitForTimeout(120);
  const r = await page.evaluate(() => document.getElementById("menu").getBoundingClientRect().toJSON());
  console.log(name.padEnd(34), "x:", r.x.toFixed(0).padStart(5), "right:", r.right.toFixed(0).padStart(5), "y:", r.y.toFixed(0));
  await page.close();
}

const base = "inset: auto; margin: 8px;";
// target: flush-end => right=1100, y=58 (below trigger bottom=50+8)
await t("bottom x-start", `position-area: bottom x-start; ${base}`);
await t("bottom x-end", `position-area: bottom x-end; ${base}`);
await t("bottom span-x", `position-area: bottom span-x; ${base}`);
await t("bottom left", `position-area: bottom left; ${base}`);
await t("bottom right", `position-area: bottom right; ${base}`);
await t("bottom start", `position-area: bottom start; ${base}`);
await t("bottom end", `position-area: bottom end; ${base}`);
await t("bottom + margin-auto trick", `position-area: bottom; ${base} justify-self: anchor-center;`);
await browser.close();
