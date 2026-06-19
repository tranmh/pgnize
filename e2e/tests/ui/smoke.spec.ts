import { test, expect } from "@playwright/test";

// Browser smoke for the key routes. Deeper review-workbench interaction is exercised
// against the real DOM in CI where the web server is guaranteed up; these checks keep
// the suite resilient to component-level markup changes.
const routes = ["/", "/convert", "/login", "/register", "/library", "/upload"];

for (const route of routes) {
  test(`renders ${route} without crashing`, async ({ page }) => {
    const resp = await page.goto(route);
    expect(resp?.status(), `navigation to ${route}`).toBeLessThan(500);
    // The app shell must mount some visible content.
    await expect(page.locator("body")).not.toBeEmpty();
  });
}

test("convert page exposes an image upload control", async ({ page }) => {
  await page.goto("/convert");
  // A file input (possibly hidden behind a dropzone) must exist for photo capture.
  await expect(page.locator('input[type="file"]')).toHaveCount(1, { timeout: 10_000 });
});
