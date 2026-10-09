import { test, expect } from "@playwright/test";

// TEMPORARY debug spec — delete after T01 triage.

test("debug: probe worker + DB + 404s", async ({ page, context }) => {
  const logs: string[] = [];
  page.on("console", (m) => logs.push(`[page:${m.type()}] ${m.text()}`));
  page.on("pageerror", (e) => logs.push(`[pageerror] ${e.message}`));
  page.on("response", (r) => {
    if (r.status() >= 400) logs.push(`[http ${r.status()}] ${r.url()}`);
  });
  page.on("requestfailed", (r) => logs.push(`[reqfail] ${r.url()} ${r.failure()?.errorText}`));

  await page.goto("/");
  await expect(page.locator("[data-sync-status]")).toContainText(
    /Connected|Synced|All changes saved/i,
    { timeout: 15000 },
  );
  await page.waitForTimeout(1500);

  const probe = await page.evaluate(`(async function() {
    var dbs = [];
    if (indexedDB.databases) {
      try { dbs = (await indexedDB.databases()).map(function(d) { return d.name + "(v" + (d.version || "?") + ")"; }); } catch (e) { dbs = ["err " + e]; }
    }
    var clientVersion = window.cqrsSync && window.cqrsSync.version;
    var events = null;
    if (window.cqrsSync && window.cqrsSync.getEvents) {
      try { events = JSON.stringify(await window.cqrsSync.getEvents()).slice(0, 200); } catch (e) { events = "err " + e; }
    }
    return { dbs: dbs, clientVersion: clientVersion, events: events, onLine: navigator.onLine };
  })()`);
  console.log("=== PROBE (after load, before submit) ===", JSON.stringify(probe));

  await context.setOffline(true);
  await page.waitForTimeout(300);
  await page.fill('input[name="name"]', "Debug Item 2");
  await page.click('button[type="submit"]');
  await page.waitForTimeout(2000);

  const probe2 = await page.evaluate(`(async function() {
    var dbs = [];
    if (indexedDB.databases) {
      try { dbs = (await indexedDB.databases()).map(function(d) { return d.name + "(v" + (d.version || "?") + ")"; }); } catch (e) { dbs = ["err " + e]; }
    }
    var out = {};
    try {
      out.state = await new Promise(function(resolve) {
        var req = indexedDB.open('cqrshtmx-sync');
        req.onsuccess = function(e) {
          var db = e.target.result;
          out.version = db.version;
          out.stores = Array.from(db.objectStoreNames);
          if (db.objectStoreNames.contains('commands')) {
            var tx = db.transaction('commands', 'readonly');
            var c = tx.objectStore('commands').count();
            c.onsuccess = function() { db.close(); resolve({count: c.result}); };
          } else { db.close(); resolve({count: -1}); }
        };
        req.onerror = function() { resolve({error: String(req.error)}); };
      });
    } catch (e) { out.state = "err " + e; }
    return out;
  })()`);
  console.log("=== PROBE2 (after offline submit) ===", JSON.stringify(probe2));
  console.log("=== CONSOLE/HTTP LOGS ===");
  for (const l of logs) console.log(l);
});
