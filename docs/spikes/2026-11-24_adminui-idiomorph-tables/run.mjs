// T24 F176 spike runner — Playwright drives the scratch HTML page headless.
import { chromium } from "playwright-core";
import { writeFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";

const dir = dirname(fileURLToPath(import.meta.url));
const executablePath = process.env.E2E_BROWSER_PATH || undefined;
const browser = await chromium.launch({ executablePath, args: ["--no-sandbox"] });
const page = await browser.newPage();
page.on("console", (m) => console.log("[console]", m.text()));
page.on("pageerror", (e) => { console.error("[pageerror]", e.message); process.exitCode = 1; });
await page.goto("file://" + join(dir, "index.html"));
await page.waitForFunction(() => window.__SPIKE_DONE === true, { timeout: 20000 });
const results = await page.evaluate(() => document.getElementById("result").textContent);
writeFileSync(join(dir, "RESULTS.json"), results);
console.log(results);
await browser.close();
