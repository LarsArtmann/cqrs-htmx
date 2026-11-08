import { firefox } from "playwright";

const DEMO = "http://localhost:18933";
const FF = process.env.E2E_FIREFOX_PATH;

const browser = await firefox.launch({
  executablePath: FF,
  env: { LD_LIBRARY_PATH: process.env.E2E_FIREFOX_LIBS },
});
const ctx = await browser.newContext();
const page = await ctx.newPage();

page.on("console", (m) => {
  const t = m.text();
  if (t.includes("[probe]") || m.type() === "error") console.log(`[${m.type()}] ${t}`);
});
page.on("pageerror", (e) => console.log(`[pageerror] ${e.message}`));
page.on("request", (r) => {
  if (r.url().includes("/ds/events")) console.log(`[pw-request] ${r.method()} ${r.url()} hdrs=${JSON.stringify(r.headers())}`);
});

await page.goto(`${DEMO}/dev-login`, { waitUntil: "load" });

await page.addInitScript(() => {
  window.__probe = { fetches: [], chunks: [], span: [] };
  const orig = window.fetch;
  window.fetch = async (...args) => {
    const url = String(args[0]?.url ?? args[0]);
    const res = await orig(...args);
    if (url.includes("/ds/events")) {
      const init = args[1] ?? {};
      window.__probe.fetches.push({ t: performance.now() | 0, status: res.status, ct: res.headers.get("content-type"), method: init.method ?? "GET" });
      console.log(`[probe] FETCH /ds/events -> ${res.status} (${res.headers.get("content-type")}) @${performance.now() | 0}`);
      if (res.body) {
        const [toSdk, toLog] = res.body.tee();
        toLog.getReader().read().then(function pump({ done, value }) {
          if (done) { console.log("[probe] LOG-BRANCH done"); return; }
          const text = new TextDecoder().decode(value);
          window.__probe.chunks.push({ t: performance.now() | 0, text });
          console.log(`[probe] CHUNK @${performance.now() | 0} ${JSON.stringify(text)}`);
          return toLog.getReader().read().then(pump);
        }).catch((e) => console.log(`[probe] LOG-ERR ${e}`));
        return new Response(toSdk, { headers: res.headers, status: res.status, statusText: res.statusText });
      }
    }
    return res;
  };
  setInterval(() => {
    const el = document.querySelector("[data-text='$broadcasts']");
    if (el) window.__probe.span.push({ t: performance.now() | 0, v: el.textContent });
  }, 500);
});

await page.goto(`${DEMO}/ds-demo`, { waitUntil: "load" });
await page.waitForTimeout(3000);
console.log(`[probe] initial: ${await page.locator("[data-text='$broadcasts']").textContent()}`);

const post = () =>
  page.evaluate(async () => {
    const r = await fetch("/broadcast", { method: "POST" });
    return r.status;
  });
console.log(`[probe] post1 status: ${await post()}`);
await page.waitForTimeout(2500);
console.log(`[probe] after post1: ${await page.locator("[data-text='$broadcasts']").textContent()}`);
console.log(`[probe] post2 status: ${await post()}`);
await page.waitForTimeout(2500);
console.log(`[probe] after post2: ${await page.locator("[data-text='$broadcasts']").textContent()}`);
await page.waitForTimeout(3000);
console.log(`[probe] final: ${await page.locator("[data-text='$broadcasts']").textContent()}`);

const probe = await page.evaluate(() => window.__probe);
console.log(`[probe] /ds/events connections: ${probe.fetches.length}`);
probe.fetches.forEach((f) => console.log(`  fetch @${f.t} ${f.method} -> ${f.status}`));

await browser.close();
