import { firefox } from "playwright";
const DEMO = "http://localhost:18923";
const browser = await firefox.launch({ executablePath: process.env.E2E_FIREFOX_PATH, env: { LD_LIBRARY_PATH: process.env.E2E_FIREFOX_LIBS } });
const ctx = await browser.newContext();
const page = await ctx.newPage();
page.on("console", (m) => { const t = m.text(); if (t.includes("sync-worker") || t.includes("[probe]")) console.log(`[page:${m.type()}] ${t.slice(0,140)}`); });
page.on("pageerror", (e) => console.log(`[pageerror] ${e.message}`));

const dumpIDB = `(async function() {
  const openReq = indexedDB.open("cqrshtmx-sync");
  const db = await new Promise((res, rej) => { openReq.onsuccess = () => res(openReq.result); openReq.onerror = () => rej(openReq.error); });
  const names = [...db.objectStoreNames];
  const meta = await new Promise((res) => { const tx = db.transaction("meta", "readonly"); const r = tx.objectStore("meta").get("state"); r.onsuccess = () => res(r.result); r.onerror = () => res("READ-ERR:" + r.error); });
  const count = await new Promise((res) => { const tx = db.transaction("events", "readonly"); const r = tx.objectStore("events").count(); r.onsuccess = () => res(r.result); r.onerror = () => res("READ-ERR:" + r.error); });
  db.close();
  return { stores: names, meta, eventCount: count };
})()`;

await page.goto(`${DEMO}/`, { waitUntil: "load" });
await page.waitForTimeout(6000);
console.log(`[probe] BEFORE: ${JSON.stringify(await page.evaluate(dumpIDB))}`);
console.log(`[probe] api BEFORE: ${JSON.stringify(await page.evaluate('window.cqrsSync.getEvents().then(o => ({count: o.events.length, cursor: o.cursor, backendId: o.backendId}))'))}`);
const rotate = await page.request.post(`${DEMO}/api/debug/rotate-backend`);
console.log(`[probe] rotate: ${rotate.status()}`);
await page.evaluate('window.cqrsSync.pull()');
await page.waitForTimeout(4000);
console.log(`[probe] AFTER: ${JSON.stringify(await page.evaluate(dumpIDB))}`);
console.log(`[probe] api AFTER: ${JSON.stringify(await page.evaluate('window.cqrsSync.getEvents().then(o => ({count: o.events.length, cursor: o.cursor, backendId: o.backendId}))'))}`);
await browser.close();
