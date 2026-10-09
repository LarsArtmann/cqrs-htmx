import { test, expect } from "@playwright/test";

// TEMPORARY debug spec — delete after T01 triage.

test("debug: instrument real worker boot errors", async ({ page }) => {
  await page.goto("/");
  const result = await page.evaluate(`(async function() {
    var src = await fetch('/sync-worker.js').then(function(r) { return r.text(); });
    var prefix = "var __bootError = null;" +
      "self.addEventListener('error', function(e) {" +
      "  if (!__bootError) __bootError = String(e.message) + ' @' + (e.filename||'?') + ':' + (e.lineno||'?') + ':' + (e.colno||'?');" +
      "});";
    var suffix = "(function() {" +
      "  var _orig = self.onconnect;" +
      "  self.onconnect = function(e) {" +
      "    var port = e.ports[0];" +
      "    port.addEventListener('message', function(ev) {" +
      "      if (ev.data && ev.data.type === '__bootprobe') {" +
      "        port.postMessage({ type: '__bootreply', error: __bootError, onconnectAssigned: typeof _orig === 'function' });" +
      "      }" +
      "    });" +
      "    if (_orig) _orig(e);" +
      "  };" +
      "})();";
    var blob = new Blob([prefix + src + suffix], { type: 'application/javascript' });
    var w = new SharedWorker(URL.createObjectURL(blob));
    return await new Promise(function(resolve) {
      var out = { replies: [] };
      var t1 = setTimeout(function() { resolve({ timeout: true, out: out }); }, 6000);
      w.port.onmessage = function(e) {
        if (e.data && e.data.type === '__bootreply') { clearTimeout(t1); resolve(e.data); }
      };
      w.port.postMessage({ type: 'hello', tabId: 'probe-tab' });
      w.port.postMessage({ type: '__bootprobe' });
      setTimeout(function() { w.port.postMessage({ type: '__bootprobe' }); }, 1500);
    });
  })()`);
  console.log("=== BOOT PROBE ===", JSON.stringify(result));
  expect(true).toBe(true);
});
