import { firefox } from "playwright";
const DEMO = "http://localhost:18931";
const browser = await firefox.launch({ executablePath: process.env.E2E_FIREFOX_PATH, env: { LD_LIBRARY_PATH: process.env.E2E_FIREFOX_LIBS } });
const ctx = await browser.newContext();
const page = await ctx.newPage();
page.on("pageerror", (e) => console.log(`[t1] ${e.message}`));
await page.goto(`${DEMO}/`, { waitUntil: "load" });
await page.waitForTimeout(2000);
const probePost = (p, l) => p.evaluate(async (l) => { const r = await fetch("/api/todos", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ title: l }) }); return r.status; }, l);
const count = (p) => p.locator("#event-log .event-entry").count();
// prove tab1
for (let i = 0; i < 3; i++) {
  const before = await count(page);
  await probePost(page, `t1-${i}`);
  try { await page.locator("#event-log .event-entry").first().waitFor({ state: "attached", timeout: 4000 });
    if ((await count(page)) > before) { console.log(`[t1] LIVE attempt ${i+1}`); break; } } catch {}
}
await page.close();
await ctx.waitForTimeout?.(0);
// fresh tab AFTER tab1 closed
const t2 = await ctx.newPage();
t2.on("pageerror", (e) => console.log(`[t2] ${e.message}`));
await t2.goto(`${DEMO}/`, { waitUntil: "load" });
await t2.waitForTimeout(2500);
let live = false;
for (let i = 0; i < 5; i++) {
  const before = await count(t2);
  await probePost(t2, `t2-${i}`);
  try { await t2.locator("#event-log .event-entry").first().waitFor({ state: "attached", timeout: 4000 });
    if ((await count(t2)) > before) { console.log(`[t2] LIVE attempt ${i+1}`); live = true; break; } } catch {}
}
console.log(`RESULT t2=${live}`);
await browser.close();
