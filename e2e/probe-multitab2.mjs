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
    console.log(`[${tag}:${m.type()}] ${m.text().slice(0, 160)}`);
  });
  page.on("pageerror", (e) => console.log(`[${tag}] pageerror: ${e.message}`));
  page.on("requestfailed", (r) => console.log(`[${tag}] REQ-FAIL ${r.method()} ${r.url()} :: ${r.failure()?.errorText}`));
};

const withFetchLog = (page, tag) =>
  page.addInitScript((t) => {
    const orig = window.fetch;
    window.fetch = async (...args) => {
      const url = String(args[0]?.url ?? args[0]);
      const res = await orig(...args);
      if (url.includes("/api/events")) {
        console.log(`[probe] ${t} /api/events -> ${res.status} @${performance.now() | 0}`);
      }
      return res;
    };
    // Count DOM entries every second
    setInterval(() => {
      const n = document.querySelectorAll("#event-log .event-entry").length;
      (window.__counts ??= {})[t] = n;
    }, 1000);
  }, tag);

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

const connectStream = async (page, tag, label, attempts = 5) => {
  for (let i = 0; i < attempts; i++) {
    const before = await eventCount(page);
    const status = await probePost(page, `${label}-${i}`);
    console.log(`[${tag}] ${label}-${i} post=${status} before=${before}`);
    try {
      await page.locator("#event-log .event-entry").first().waitFor({ state: "attached", timeout: 4000 });
      const after = await eventCount(page);
      if (after > before) {
        console.log(`[${tag}] ${label} LIVE (attempt ${i + 1}): ${before} -> ${after}`);
        return true;
      }
    } catch { /* not yet */ }
  }
  console.log(`[${tag}] ${label} NEVER LIVE`);
  return false;
};

// === EXACT SPEC SEQUENCE ===
const tab1 = await ctx.newPage();
instrument(tab1, "tab1");
withFetchLog(tab1, "tab1");
await tab1.goto(`${DEMO}/`, { waitUntil: "load" });
await tab1.waitForTimeout(2000);

const tab2 = await ctx.newPage();
instrument(tab2, "tab2");
withFetchLog(tab2, "tab2");
await tab2.goto(`${DEMO}/`, { waitUntil: "load" });
await tab2.waitForTimeout(2000);

console.log("=== tab1 connectStream (tab1 front) ===");
const t1 = await connectStream(tab1, "tab1", "tab1-probe");

console.log("=== tab2 bringToFront + reload ===");
await tab2.bringToFront();
await tab2.reload({ waitUntil: "load" });
await tab2.waitForTimeout(2000);
const t2 = await connectStream(tab2, "tab2", "tab2-probe");

console.log(`=== RESULT tab1=${t1} tab2=${t2} ===`);

// Diagnostics: what does tab2's API view see?
const diag = await tab2.evaluate(() => ({
  online: navigator.onLine,
  visibility: document.visibilityState,
  hasCqrsSync: typeof window.cqrsSync,
}));
console.log(`[tab2 diag] ${JSON.stringify(diag)}`);
const counts = await tab2.evaluate(() => window.__counts);
console.log(`[tab2 counts over time] ${JSON.stringify(counts)}`);

await browser.close();
