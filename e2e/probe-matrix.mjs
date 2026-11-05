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
  // trigger: x=1000..1100, flush-end target: right=1100; centered target: x~1024
  console.log(name.padEnd(42), "x:", r.x.toFixed(0).padStart(5), "right:", r.right.toFixed(0).padStart(5), "y:", r.y.toFixed(0));
  await page.close();
}

const base = "inset: auto; margin: 8px;";
await t("1. area:bottom (control)", `position-area: bottom; ${base}`);
await t("2. bottom + justify-self:end", `position-area: bottom; justify-self: end; ${base}`);
await t("3. bottom + justify-self:start", `position-area: bottom; justify-self: start; ${base}`);
await t("4. area: bottom end", `position-area: bottom end; ${base}`);
await t("5. area: bottom start", `position-area: bottom start; ${base}`);
await t("6. area: bottom span-inline-start (library)", `position-area: bottom span-inline-start; ${base}`);
await t("7. area: bottom span-inline-end", `position-area: bottom span-inline-end; ${base}`);
await t("8. bottom + justify-self:safe end", `position-area: bottom; justify-self: safe end; ${base}`);
await browser.close();
