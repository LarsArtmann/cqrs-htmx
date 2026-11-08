import { firefox } from "playwright";

const DEMO = "http://localhost:18923";
const FF = process.env.E2E_FIREFOX_PATH;

const browser = await firefox.launch({
  executablePath: FF,
  env: { LD_LIBRARY_PATH: process.env.E2E_FIREFOX_LIBS },
});
const ctx = await browser.newContext();
const page = await ctx.newPage();
page.on("console", (m) => {
  const t = m.text();
  if (t.includes("sync-worker") || t.includes("[probe]")) console.log(`[page] ${t}`);
});
page.on("pageerror", (e) => console.log(`[pageerror] ${e.message}`));

await page.goto(`${DEMO}/`, { waitUntil: "load" });
await page.waitForTimeout(6000);

const dumpIDB = `(async function() {
  const openReq = indexedDB.open("cqrshtmx-sync");
  const db = await new Promise((res, rej) => { openReq.onsuccess = () => res(openReq.result); openReq.onerror = () => rej(openReq.error); });
  const names = [...db.objectStoreNames];
  const meta = await new Promise((res) => {
    const tx = db.transaction("meta", "readonly");
    const r = tx.objectStore("meta").get("state");
    r.onsuccess = () => res(r.result); r.onerror = () => res("READ-ERR");
  });
  const count = await new Promise((res) => {
    const tx = db.transaction("events", "readonly");
    const r = tx.objectStore("events").count();
    r.onsuccess = () => res(r.result); r.onerror = () => res("READ-ERR");
  });
  db.close();
  return { stores: names, meta, eventCount: count };
})()`;

const before = await page.evaluate(dumpIDB);
console.log(`[probe] BEFORE rotate: ${JSON.stringify(before)}`);
const api = await page.evaluate(`window.cqrsSync.getEvents().then(o => ({count: o.events.length, cursor: o.cursor, backendId: o.backendId}))`);
console.log(`[probe] BEFORE via api: ${JSON.stringify(api)}`);

const rotate = await page.request.post(`${DEMO}/api/debug/rotate-backend`);
console.log(`[probe] rotate status: ${rotate.status()}`);
await page.evaluate(`window.cqrsSync.pull()`);
await page.waitForTimeout(3000);

const after = await page.evaluate(dumpIDB);
console.log(`[probe] AFTER rotate: ${JSON.stringify(after)}`);
const api2 = await page.evaluate(`window.cqrsSync.getEvents().then(o => ({count: o.events.length, cursor: o.cursor, backendId: o.backendId}))`);
console.log(`[probe] AFTER via api: ${JSON.stringify(api2)}`);

await browser.close();
