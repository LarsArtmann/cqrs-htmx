import { test } from '@playwright/test';

test('check IDB directly from worker context', async ({ page }) => {
  page.on('console', function(msg) { console.log('BROWSER:', msg.text()); });

  await page.goto('/');
  await page.waitForTimeout(2000);

  // Route block for POST
  await page.route('**/api/items', function(route) {
    if (route.request().method() === 'POST') route.abort('failed');
    else route.continue();
  });
  await page.waitForTimeout(300);

  // Submit and wait
  await page.fill('input[name="name"]', 'Test Direct');
  await page.click('button[type="submit"]');
  await page.waitForTimeout(5000);

  // Check IDB from the page context
  const depth = await page.evaluate("(new Promise(function(r){var req=indexedDB.open('cqrshtmx-sync',1);req.onupgradeneeded=function(e){var db=e.target.result;if(!db.objectStoreNames.contains('commands'))db.createObjectStore('commands',{keyPath:'commandId'})};req.onsuccess=function(e){var db=e.target.result;if(!db.objectStoreNames.contains('commands')){db.close();r(-1);return;}var tx=db.transaction('commands','readonly');var cr=tx.objectStore('commands').count();cr.onsuccess=function(){db.close();r(cr.result)};cr.onerror=function(){db.close();r(-2)}};req.onerror=function(){r(-3)}}))");
  console.log('PAGE_IDB_DEPTH:', depth);

  // Also check with getAll to see entries
  const entries = await page.evaluate("(new Promise(function(r){var req=indexedDB.open('cqrshtmx-sync',1);req.onsuccess=function(e){var db=e.target.result;if(!db.objectStoreNames.contains('commands')){r([]);return;}var tx=db.transaction('commands','readonly');var ga=tx.objectStore('commands').getAll();ga.onsuccess=function(){r(ga.result||[])};ga.onerror=function(){r([])}};req.onerror=function(){r([])}}))");
  console.log('PAGE_IDB_ENTRIES:', JSON.stringify(entries));
});
