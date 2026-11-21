import { test, expect } from "@playwright/test";

// THROWAWAY diagnostic probe for the QUERY-ladder regression (delete after).
const ADMIN = "http://localhost:18930";

test("probe: extension state + real request methods", async ({ page }) => {
  const consoleErrors: string[] = [];
  page.on("console", (m) => {
    if (m.type() === "error") consoleErrors.push(m.text());
  });
  page.on("pageerror", (e) => consoleErrors.push(`pageerror: ${e.message}`));

  const methods: string[] = [];
  page.on("request", (r) => {
    if (r.url().includes("/users/search")) methods.push(`${r.method()} ${r.url()}`);
  });

  await page.goto(`${ADMIN}/admin/users`, { waitUntil: "load" });
  await page.waitForTimeout(1200); // generous: let every script settle

  const probe = await page.evaluate(() => {
    const form = document.querySelector('input[name="q"]')?.closest("form");
    const reg = window.htmx
      ? Object.keys((window.htmx as any).extensionRegistry || (window.htmx as any).definedExtensions || {})
      : ["NO HTMX"];
    const extScript = Array.from(document.querySelectorAll("script[src]")).map((s) =>
      (s as HTMLScriptElement).getAttribute("src"),
    );
    return {
      formAttrs: form ? form.outerHTML.slice(0, 600) : "NO FORM",
      extensions: reg,
      scriptSrcs: extScript,
      hxExtInherited: form?.getAttribute("hx-ext"),
    };
  });
  console.log("PROBE-FORM:", probe.formAttrs);
  console.log("PROBE-EXTENSIONS:", JSON.stringify(probe.extensions));
  console.log("PROBE-SCRIPTS:", JSON.stringify(probe.scriptSrcs));
  console.log("PROBE-HXEXT:", probe.hxExtInherited);

  const input = page.locator('input[name="q"]');
  await input.fill("team1");
  await input.press("Enter");
  await page.waitForTimeout(1500);
  console.log("PROBE-METHODS:", JSON.stringify(methods));
  console.log("PROBE-CONSOLE-ERRORS:", JSON.stringify(consoleErrors));
  expect(true).toBe(true);
});
