import { chromium } from 'playwright';
import { pathToFileURL } from 'url';
import { resolve } from 'path';

const html = resolve('/home/lars/projects/cqrs-htmx/docs/research/2026-11-17_docs-marketing-skill-fleet-audit.html');
const out = '/home/lars/projects/cqrs-htmx/docs/research/2026-11-17_docs-marketing-skill-fleet-audit.render.png';

const browser = await chromium.launch();
const page = await browser.newPage({ viewport: { width: 1440, height: 900 } });
await page.goto(pathToFileURL(html).href, { waitUntil: 'networkidle' });

const probe = await page.evaluate(() => {
  const ids = ['review-audiblesync','review-dnsblockd','review-PapDashboard','review-CV','review-cv','review-cqrs-htmx'];
  return ids.map(id => ({ id, found: !!document.getElementById(id) }));
});
console.log('anchor probe:', JSON.stringify(probe));

await page.screenshot({ path: out, fullPage: true });
const h = await page.evaluate(() => document.body.scrollHeight);
console.log('fullPage height:', h);
await browser.close();
