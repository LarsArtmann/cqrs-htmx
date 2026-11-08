import { chromium, firefox } from "playwright";
import fs from "node:fs";

const ADMIN = "http://localhost:18930";
const ENGINE = process.env.PROBE_ENGINE === "firefox" ? firefox : chromium;
const EXE = process.env.E2E_BROWSER_PATH;

const browser = await ENGINE.launch(EXE ? { executablePath: EXE, env: { LD_LIBRARY_PATH: process.env.E2E_FIREFOX_LIBS } } : {});
const ctx = await browser.newContext();
const page = await ctx.newPage();

await page.goto(`${ADMIN}/dev-login`, { waitUntil: "load" });
await page.waitForTimeout(500);

const cookies = await ctx.cookies(ADMIN);
console.log(`[probe] cookies: ${cookies.map((c) => `${c.name}=${c.value.slice(0, 18)}…(path=${c.path})`).join("  ")}`);

const victim = `probe-${Date.now()}@acme.dev`;

// Variant 1: header + cookie value
const r1 = await page.request.post(`${ADMIN}/admin/users`, {
  headers: {
    "Content-Type": "application/x-www-form-urlencoded",
    "X-Csrf-Token": cookies.find((c) => c.name === "csrf_token")?.value ?? "",
  },
  data: `email=${encodeURIComponent(victim)}&display_name=Probe One`,
  maxRedirects: 0,
});
console.log(`[probe] v1 header+cookievalue: ${r1.status()}`);
if (!r1.ok()) console.log(`[probe] v1 body: ${(await r1.text()).slice(0, 400)}`);

// Variant 2: token from the page's meta tag
const meta = await page.evaluate(() => document.querySelector('meta[name="csrf-token"]')?.content ?? "NO-META");
console.log(`[probe] meta token: ${String(meta).slice(0, 30)}…`);
const r2 = await page.request.post(`${ADMIN}/admin/users`, {
  headers: {
    "Content-Type": "application/x-www-form-urlencoded",
    "X-Csrf-Token": meta === "NO-META" ? "" : meta,
  },
  data: `email=${encodeURIComponent("p2" + victim)}&display_name=Probe Two`,
  maxRedirects: 0,
});
console.log(`[probe] v2 meta token: ${r2.status()}`);

// Variant 3: same-origin page fetch (browser-native credentials + nosurf header handling)
const r3 = await page.evaluate(async (email) => {
  const tok = document.querySelector('meta[name="csrf-token"]')?.content ?? "";
  const cookieTok = document.cookie.match(/csrf_token=([^;]+)/)?.[1] ?? "";
  const r = await fetch("/admin/users", {
    method: "POST",
    headers: {
      "Content-Type": "application/x-www-form-urlencoded",
      "X-Csrf-Token": tok || cookieTok,
    },
    body: `email=${encodeURIComponent(email)}&display_name=Probe Three`,
    redirect: "manual",
  });
  return { status: r.status, type: r.type };
}, `p3${victim}`).catch((e) => ({ error: String(e) }));
console.log(`[probe] v3 page-fetch: ${JSON.stringify(r3)}`);

await browser.close();
