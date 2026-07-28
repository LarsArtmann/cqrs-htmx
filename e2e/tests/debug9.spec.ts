import { test } from '@playwright/test';

test('check sendError element attributes', async ({ page, context }) => {
  page.on('console', function(msg) { console.log('BROWSER:', msg.text()); });

  // Add our own sendError listener BEFORE sync-client loads
  await page.addInitScript(() => {
    document.addEventListener('htmx:beforeRequest', function(e) {
      var elt = e.detail.elt;
      console.log('BEFORE_REQ: elt=' + elt.tagName + ' id=' + elt.id + 
        ' hasCmdId=' + (elt.getAttribute('data-command-id') ? 'Y' : 'N') +
        ' closest_sync_target=' + (elt.closest('[data-sync-target]') ? elt.closest('[data-sync-target]').tagName : 'NONE'));
    });
    document.addEventListener('htmx:sendError', function(e) {
      var elt = e.detail.elt;
      var found = elt.closest('[data-command-id]') || elt.closest('[data-sync-state]');
      console.log('SEND_ERROR: elt=' + elt.tagName + ' id=' + elt.id +
        ' closest_cmdId=' + (elt.closest('[data-command-id]') ? elt.closest('[data-command-id]').tagName : 'NONE') +
        ' closest_syncState=' + (elt.closest('[data-sync-state]') ? elt.closest('[data-sync-state]').tagName : 'NONE') +
        ' found=' + (found ? found.tagName : 'NULL') +
        ' foundCmdId=' + (found ? (found.getAttribute('data-command-id') || 'EMPTY') : 'N/A'));
    });
  });

  await page.goto('/');
  await page.waitForTimeout(2000);

  await context.setOffline(true);
  await page.route('**/api/items', function(route) {
    if (route.request().method() === 'POST') route.abort('failed');
    else route.continue();
  });
  await page.waitForTimeout(300);

  await page.fill('input[name="name"]', 'Test');
  await page.click('button[type="submit"]');
  await page.waitForTimeout(3000);
});
