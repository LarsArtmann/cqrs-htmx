import { firefox } from "playwright";

const DEMO = "http://localhost:18931";
const FF = process.env.E2E_FIREFOX_PATH;

const browser = await firefox.launch({
  executablePath: FF,
  env: { LD_LIBRARY_PATH: process.env.E2E_FIREFOX_LIBS },
});
const ctx = await browser.newContext();

const instrument = (page, tag) => {
  page.on("console", (m) => {
    const t = m.text();
    if (t.includes("[probe]")) console.log(`[${tag}] ${t}`);
  });
  page.on("pageerror", (e) => console.log(`[${tag}] pageerror: ${e.message}`));
};

const probePost = (page, label) =>
  page.evaluate(async (l) => {
    const r = await fetch("/api/todos", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ title: l }),
    });
    return r.status;
  }, label);

const eventCount = (page) => page.locator("#event-log .event-entry").count();

const connectStream = async (page, tag, label) => {
  for (let attempt = 0; attempt < 3; attempt++) {
    const before = await eventCount(page);
    const status = await probePost(page, `${label}-${attempt}`);
    console.log(`[${tag}] probe ${label}-${attempt} post=${status} before=${before}`);
    try {
      await page
        .locator("#event-log .event-entry")
        .first()
        .waitFor({ state: "attached", timeout: 4000 });
      const after = await eventCount(page);
      if (after > before) {
        console.log(`[${tag}] LIVE after ${attempt + 1} attempt(s): ${before} -> ${after}`);
        return true;
      }
    } catch {
      console.log(`[${tag}] attempt ${attempt}: no delivery`);
    }
  }
  return false;
};

const tab1 = await ctx.newPage();
instrument(tab1, "tab1");
await tab1.goto(`${DEMO}/`, { waitUntil: "load" });
await tab1.waitForTimeout(2000);

const tab2 = await ctx.newPage();
instrument(tab2, "tab2");
await tab2.goto(`${DEMO}/`, { waitUntil: "load" });
await tab2.waitForTimeout(2000);

console.log("=== PHASE 1: tab2 BACKGROUND (never brought to front) ===");
const liveBackground = await connectStream(tab2, "tab2", "bg-probe");

console.log(`=== PHASE 2: bring tab2 to FRONT, probe again ===`);
await tab2.bringToFront();
await tab2.waitForTimeout(1000);
const liveFront = await connectStream(tab2, "tab2", "front-probe");

console.log(`=== SUMMARY: background=${liveBackground} front=${liveFront} ===`);

await browser.close();
