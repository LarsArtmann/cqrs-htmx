import { firefox } from "playwright";

const DEMO = "http://localhost:18933";
const FF = process.env.E2E_FIREFOX_PATH;

const browser = await firefox.launch({
  executablePath: FF,
  env: { LD_LIBRARY_PATH: process.env.E2E_FIREFOX_LIBS },
});
const ctx = await browser.newContext();
const page = await ctx.newPage();

page.on("console", (m) => console.log(`[console:${m.type()}] ${m.text()}`));
page.on("pageerror", (e) => console.log(`[pageerror] ${e.message}`));

await page.goto(`${DEMO}/dev-login`, { waitUntil: "load" });

await page.addInitScript(() => {
  window.__probe = { fetches: [], chunks: [], errors: [] };
  const orig = window.fetch;
  window.fetch = async (...args) => {
    const url = String(args[0]?.url ?? args[0]);
    if (url.includes("/ds/events")) {
      window.__probe.fetches.push({ t: performance.now(), url, headers: args[1]?.headers ?? null });
      console.log(`[probe] FETCH /ds/events #${window.__probe.fetches.length} @${performance.now() | 0}`);
    }
    const res = await orig(...args);
    if (url.includes("/ds/events") && res.body) {
      const reader = res.body.getReader().getReader ? res.body.getReader() : res.body.getReader();
      const stream = new ReadableStream({
        start(ctrl) {
          const pump = async () => {
            try {
              for (;;) {
                const { done, value } = await reader.read();
                if (done) { ctrl.close(); return; }
                const text = new TextDecoder().decode(value);
                window.__probe.chunks.push({ t: performance.now() | 0, text });
                console.log(`[probe] CHUNK @${performance.now() | 0} ${JSON.stringify(text)}`);
                ctrl.enqueue(value);
              }
            } catch (e) {
              window.__probe.errors.push(String(e));
              console.log(`[probe] READ-ERR ${e}`);
            }
          };
          pump();
        },
      });
      return new Response(stream, { headers: res.headers, status: res.status });
    }
    return res;
  };
});

await page.goto(`${DEMO}/ds-demo`, { waitUntil: "load" });
const counter = page.locator("[data-text='$broadcasts']");
await page.waitForTimeout(3000);
console.log(`[probe] initial span text: ${await counter.textContent()}`);

// One broadcast at a time, sampling the span between posts.
await fetch(DEMO + "/broadcast", { method: "POST" }).catch(() => {});
await page.waitForTimeout(2500);
console.log(`[probe] after post1: ${await counter.textContent()}`);
await fetch(DEMO + "/broadcast", { method: "POST" }).catch(() => {});
await page.waitForTimeout(2500);
console.log(`[probe] after post2: ${await counter.textContent()}`);
await page.waitForTimeout(3000);
console.log(`[probe] final: ${await counter.textContent()}`);

const probe = await page.evaluate(() => window.__probe);
console.log(`[probe] total /ds/events fetches: ${probe.fetches.length}`);
for (const f of probe.fetches) console.log(`  fetch @${f.t | 0}`);
console.log(`[probe] errors: ${JSON.stringify(probe.errors)}`);

await browser.close();
