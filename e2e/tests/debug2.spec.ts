import { test } from '@playwright/test';

test('debug data-command-id placement', async ({ page, context }) => {
  await page.goto('/');
  await page.waitForTimeout(1500);

  await context.setOffline(true);
  await page.route('**/api/items', function(route) {
    if (route.request().method() === 'POST') route.abort('failed');
    else route.continue();
  });
  await page.waitForTimeout(300);

  await page.fill('input[name="name"]', 'Test');
  await page.click('button[type="submit"]');
  await page.waitForTimeout(2000);

  const state = await page.evaluate(() => {
    var all = document.querySelectorAll('*');
    var tagged = [];
    all.forEach(function(el) {
      if (el.getAttribute('data-command-id') || el.getAttribute('data-sync-state') || el.getAttribute('data-sync-queued')) {
        tagged.push({
          tag: el.tagName,
          id: el.id,
          cmdId: el.getAttribute('data-command-id'),
          syncState: el.getAttribute('data-sync-state'),
          syncQueued: el.getAttribute('data-sync-queued'),
        });
      }
    });
    return tagged;
  });
  console.log('TAGGED ELEMENTS:', JSON.stringify(state, null, 2));
  
  // Also check the sendError event detail
  const evtDetail = await page.evaluate(() => {
    // Manually dispatch a test to see what htmx provides
    return typeof htmx !== 'undefined' ? 'htmx loaded' : 'no htmx';
  });
  console.log('HTMX:', evtDetail);
});
