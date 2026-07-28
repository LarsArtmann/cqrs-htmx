import { test } from '@playwright/test';

test('trace data-command-id lifecycle', async ({ page }) => {
  page.on('console', function(msg) { console.log('BROWSER:', msg.text()); });
  page.on('pageerror', function(err) { console.log('PAGEERROR:', err.message); });

  // Trace all htmx events and check data-command-id at each step
  await page.addInitScript(() => {
    ['htmx:beforeRequest', 'htmx:afterRequest', 'htmx:sendError', 'htmx:responseError'].forEach(function(evtName) {
      document.addEventListener(evtName, function(e) {
        var elt = e.detail.elt;
        var target = elt ? elt.closest('[data-command-id]') : null;
        var targetState = elt ? elt.closest('[data-sync-state]') : null;
        console.log(evtName + ': elt=' + (elt ? elt.tagName : 'NULL') +
          ' closestCmdId=' + (target ? target.getAttribute('data-command-id') : 'NONE') +
          ' closestSyncState=' + (targetState ? targetState.getAttribute('data-sync-state') : 'NONE') +
          ' pending=' + (e.detail.requestConfig ? e.detail.requestConfig.headers['X-Command-Id'] : 'no-cfg'));
      });
    });
  });

  await page.goto('/');
  await page.waitForTimeout(2000);

  await page.route('**/api/items', function(route) {
    if (route.request().method() === 'POST') route.abort('failed');
    else route.continue();
  });
  await page.waitForTimeout(300);

  await page.fill('input[name="name"]', 'Trace Test');
  await page.click('button[type="submit"]');
  await page.waitForTimeout(5000);

  const depth = await page.evaluate("(new Promise(function(r){var req=indexedDB.open('cqrshtmx-sync',1);req.onsuccess=function(e){var db=e.target.result;if(!db.objectStoreNames.contains('commands')){r(-1);return;}var tx=db.transaction('commands','readonly');var cr=tx.objectStore('commands').count();cr.onsuccess=function(){r(cr.result)};cr.onerror=function(){r(-2)}};req.onerror=function(){r(-3)}}))");
  console.log('QUEUE_DEPTH:', depth);
});
