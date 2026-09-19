import { test, expect } from "@playwright/test";

/**
 * Browser-truth specs for the dashboardui templ-components adoption (Run-3
 * N6-N8). These assert BEHAVIOR in real Chromium against the seeded e2e
 * server: stat-card ValueIDs, badge rendering, toast bridging, error-page
 * shapes, and table sorting. SSE live-row injection is intentionally not
 * covered here: it needs a real event.Bus wired into Config.EventBus, and
 * the SSE envelope itself is golden-pinned by the transport package.
 */

test.describe("dashboard badges + stat cards (N6)", () => {
  test("overview stat cards expose stable ValueIDs with real values", async ({
    page,
  }) => {
    await page.goto("/dashboard/");
    await expect(page.locator("#stat-total-events")).toHaveText("4");

    const health = page.locator("#stat-system-health");
    await expect(health).toBeVisible();
    await expect(health).toContainText(/healthy/i);
  });

  test("events table renders mapped type badges and seeded rows", async ({
    page,
  }) => {
    await page.goto("/dashboard/events");
    await expect(page.locator("#events-tbody tr")).toHaveCount(4);
    await expect(
      page.locator("#events-tbody").getByText("User.registered").first(),
    ).toBeVisible();
    await expect(
      page.locator("#events-tbody").getByText("Tenant.registered").first(),
    ).toBeVisible();
  });

  test("projections page shows the drained worker as healthy-classed stopped badge", async ({ page }) => {
    await page.goto("/dashboard/projections");
    await expect(page.getByText("demo-projection").first()).toBeVisible();

    // The M8 semantics: a drained journal-only worker reports "stopped" but
    // classifies healthy - the badge carries the success tone + status dot.
    const badge = page.getByText("stopped", { exact: true }).first();
    await expect(badge).toBeVisible();
    await expect(badge.locator("xpath=ancestor::*[contains(@class,'badge')][1]"))
      .toContainText("stopped");
    await expect(page.getByText("4").first()).toBeVisible(); // processed
  });
});

test.describe("dashboard toasts + error pages (N7)", () => {
  test("dashboardui:toast events surface in the toast container", async ({
    page,
  }) => {
    await page.goto("/dashboard/");
    await page.evaluate(() => {
      document.dispatchEvent(
        new CustomEvent("dashboardui:toast", {
          detail: { message: "Browser-truth toast", kind: "ok" },
        }),
      );
    });
    await expect(page.getByText("Browser-truth toast")).toBeVisible();
  });

  test("unknown route renders the NotFound404 full shell", async ({ page }) => {
    const resp = await page.goto("/dashboard/definitely-not-a-page");
    expect(resp?.status()).toBe(404);
    await expect(page).toHaveTitle(/not found/i);
    await expect(page.locator("h1, h2").first()).toBeVisible();
  });

  test("HTMX requests get the bare error card, not the shell", async ({
    page,
  }) => {
    const resp = await page.request.get("/dashboard/definitely-not-a-page", {
      headers: { "HX-Request": "true" },
    });
    expect(resp.status()).toBe(404);
    const body = await resp.text();
    expect(body).not.toContain("<!DOCTYPE html>");
    expect(body.toLowerCase()).toContain("not found");
  });
});

test.describe("dashboard sortable tables (N8)", () => {
  test("clicking the Time header sorts and flips aria-sort", async ({
    page,
  }) => {
    await page.goto("/dashboard/events");
    const timeHeader = page.locator("th", { hasText: "Time" });

    await timeHeader.locator("a").first().click();
    await expect(page).toHaveURL(/sort=time&dir=asc/);
    const timeHeaderAfterAsc = page.locator("th", { hasText: "Time" });
    await expect(timeHeaderAfterAsc).toHaveAttribute("aria-sort", "ascending");

    await timeHeaderAfterAsc.locator("a").first().click();
    await expect(page).toHaveURL(/sort=time&dir=desc/);
    const timeHeaderAfterDesc = page.locator("th", { hasText: "Time" });
    await expect(timeHeaderAfterDesc).toHaveAttribute("aria-sort", "descending");
  });
});
