import { test, expect } from "@playwright/test";
import { trackPageErrors } from "../../helpers/ui";

// Browser smoke for the key routes. Deeper review-workbench interaction is exercised
// against the real DOM in CI where the web server is guaranteed up; these checks keep
// the suite resilient to component-level markup changes.
const routes = ["/", "/convert", "/scan", "/login", "/register", "/library", "/upload"];

for (const route of routes) {
  test(`renders ${route} without crashing`, async ({ page }) => {
    const errors = trackPageErrors(page);
    const resp = await page.goto(route);
    expect(resp?.status(), `navigation to ${route}`).toBeLessThan(500);
    // The app shell must mount some visible content.
    await expect(page.locator("body")).not.toBeEmpty();
    // Give the client time to hydrate; a hydration mismatch or any uncaught
    // exception surfaces as a `pageerror` (this is what "without crashing" means
    // for a client component — an HTTP 200 alone does not prove the page is sound).
    await page.waitForLoadState("networkidle").catch(() => {});
    await page.waitForTimeout(1000);
    expect(errors, `uncaught page errors on ${route}:\n${errors.join("\n")}`).toEqual([]);
  });
}

test("convert page exposes an image upload control", async ({ page }) => {
  await page.goto("/convert");
  // A file input (possibly hidden behind a dropzone) must exist for photo capture.
  await expect(page.locator('input[type="file"]')).toHaveCount(1, { timeout: 10_000 });
});
