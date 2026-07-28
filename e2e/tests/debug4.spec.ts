import { test } from '@playwright/test';

test('check sendError detail and SharedWorker', async ({ page, context }) => {
  await page.goto('/');
  await page.waitForTimeout(1500);

  // Inject a debug listener BEFORE the form submit
  await page.evaluate(() => {
    document.addEventListener('htmx:sendError', function(e) {
      var cfg = (e.detail && e.detail.requestConfig) || null;
      console.log('SENDERROR_DETAIL: ' + JSON.stringify({
        hasDetail: !!e.detail,
        hasRequestConfig: !!cfg,
        cfgVerb: cfg ? cfg.verb : 'none',
        cfgPath: cfg ? cfg.path : 'none',
        cfgParameters: cfg ? JSON.stringify(cfg.parameters) : 'none',
        cfgHeaders: cfg ? JSON.stringify(cfg.headers) : 'none',
        eltTag: e.detail.elt ? e.detail.elt.tagName : 'none',
      }));
    });
    
    // Check SharedWorker availability
    console.log('SHARED_WORKER: ' + (typeof SharedWorker !== 'undefined' ? 'available' : 'NOT available'));
  });

  await context.setOffline(true);
  await page.route('**/api/items', function(route) {
    if (route.request().method() === 'POST') route.abort('failed');
    else route.continue();
  });
  await page.waitForTimeout(300);

  page.on('console', function(msg) { 
    var t = msg.text();
    if (t.includes('SENDERROR') || t.includes('SHARED') || t.includes('sync')) {
      console.log('BROWSER:', t);
    }
  });

  await page.fill('input[name="name"]', 'Test');
  await page.click('button[type="submit"]');
  await page.waitForTimeout(3000);
});
