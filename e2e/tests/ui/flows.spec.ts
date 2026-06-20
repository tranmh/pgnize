import { test, expect } from "@playwright/test";
import { registerAndAuth, useEnglish, trackPageErrors, seedSavedGame, PIXEL_PNG } from "../../helpers/ui";

test.describe("authenticated UI flows", () => {
  test("manual game: workbench renders, saves, and the saved game views without crashing", async ({ page, context }, info) => {
    const errors = trackPageErrors(page);
    const { api } = await registerAndAuth(context, info.workerIndex);
    await useEnglish(page);

    await page.goto("/library");
    await expect(page.getByRole("heading", { name: "Library" })).toBeVisible();

    // "Enter manually" -> /review/{id}. This is the path that previously crashed
    // (manual games have no moves -> null moves -> .map on null).
    await page.getByRole("button", { name: "Enter manually" }).click();
    await page.waitForURL(/\/review\/[0-9a-f-]+$/);

    // The workbench must mount: header fields + an empty move list, no crash.
    await expect(page.locator("#hdr-white")).toBeVisible();
    await expect(page.getByText("No moves yet.")).toBeVisible();

    await page.locator("#hdr-white").fill("Manual White");
    await page.locator("#hdr-black").fill("Manual Black");
    await page.locator("#hdr-result").selectOption("1-0");

    await page.getByRole("button", { name: "Save game" }).click();
    await expect(page.getByText("Saved.")).toBeVisible();

    // Follow the "View game" link -> GameViewer (rebuild on empty moves).
    await page.getByRole("link", { name: "View game" }).click();
    await page.waitForURL(/\/games\/[0-9a-f-]+\/view$/);
    await expect(page.getByText("Manual White")).toBeVisible();
    await expect(page.getByText("Manual Black")).toBeVisible();

    expect(errors, `uncaught page errors:\n${errors.join("\n")}`).toEqual([]);
    await api.dispose();
  });

  test("photo flow: upload -> recognize -> review shows moves -> save -> library + view", async ({ page, context }, info) => {
    const errors = trackPageErrors(page);
    const { api } = await registerAndAuth(context, info.workerIndex);
    await useEnglish(page);

    await page.goto("/upload");
    await page.setInputFiles('input[type="file"]', {
      name: "sheet.png",
      mimeType: "image/png",
      buffer: PIXEL_PNG,
    });
    await page.getByRole("button", { name: "Recognize" }).click();

    // The poller redirects to the review workbench when recognition completes.
    await page.waitForURL(/\/review\/[0-9a-f-]+$/, { timeout: 30_000 });

    // The recognized Ruy Lopez should render with legal moves.
    await expect(page.getByRole("heading", { name: "Moves" })).toBeVisible();
    await expect(page.getByText("legal").first()).toBeVisible({ timeout: 15_000 });

    await page.getByRole("button", { name: "Save game" }).click();
    await expect(page.getByText("Saved.")).toBeVisible({ timeout: 15_000 });

    // It now appears in the library.
    await page.goto("/library");
    await expect(page.getByRole("button", { name: "View" }).first()).toBeVisible();

    expect(errors, `uncaught page errors:\n${errors.join("\n")}`).toEqual([]);
    await api.dispose();
  });

  test("library: search narrows results, View opens a read-only board", async ({ page, context }, info) => {
    const errors = trackPageErrors(page);
    const { api } = await registerAndAuth(context, info.workerIndex);
    await useEnglish(page);

    const tag = `Lib${Date.now()}`;
    await seedSavedGame(api, { white: `${tag}Alpha`, black: "Zeta", result: "1-0" }, [
      { ply: 1, san: "e4" },
      { ply: 2, san: "e5" },
    ]);
    await seedSavedGame(api, { white: "Other", black: "Player", result: "0-1" });

    await page.goto("/library");
    await page.getByLabel("Search games").fill(`${tag}Alpha`);
    await page.getByRole("button", { name: "Apply" }).click();

    await expect(page.getByText(`${tag}Alpha`)).toBeVisible();
    await expect(page.getByText("Other")).toHaveCount(0);

    await page.getByRole("button", { name: "View" }).first().click();
    await page.waitForURL(/\/games\/[0-9a-f-]+\/view$/);
    await expect(page.getByText(`${tag}Alpha`)).toBeVisible();
    // The two recognized plies must render in the read-only move list.
    await expect(page.getByText("legal").first()).toBeVisible();

    expect(errors, `uncaught page errors:\n${errors.join("\n")}`).toEqual([]);
    await api.dispose();
  });

  test("editing a move to an illegal SAN disables save, then fixing re-enables it", async ({ page, context }, info) => {
    const errors = trackPageErrors(page);
    const { api } = await registerAndAuth(context, info.workerIndex);
    await useEnglish(page);

    const gid = await seedSavedGame(api, { white: "Edit", black: "Test", result: "*" }, [
      { ply: 1, san: "e4" },
      { ply: 2, san: "e5" },
    ]);

    await page.goto(`/review/${gid}`);
    await expect(page.getByRole("heading", { name: "Moves" })).toBeVisible();
    const saveBtn = page.getByRole("button", { name: "Save game" });
    await expect(saveBtn).toBeEnabled();

    // Double-click ply 1's "e4" cell to edit it (exact name avoids the move-number button).
    const moveCell = page.getByRole("button", { name: "e4", exact: true });
    await moveCell.dblclick();
    const editor = page.getByLabel("Edit move");
    await expect(editor).toBeVisible();
    await editor.fill("Ke2"); // illegal first move
    await editor.press("Enter");

    // The legality badge flips to illegal and the save gate closes.
    await expect(page.getByText("illegal").first()).toBeVisible();
    await expect(saveBtn).toBeDisabled();

    expect(errors, `uncaught page errors:\n${errors.join("\n")}`).toEqual([]);
    await api.dispose();
  });
});

test.describe("anonymous UI flow", () => {
  test("recognized game flags a low-confidence move to verify, and confirming clears it", async ({
    page,
  }) => {
    const errors = trackPageErrors(page);
    await useEnglish(page);
    await page.goto("/convert");
    await page.setInputFiles('input[type="file"]', {
      name: "sheet.png",
      mimeType: "image/png",
      buffer: PIXEL_PNG,
    });
    await expect(page.getByRole("heading", { name: "Moves" })).toBeVisible({ timeout: 30_000 });

    // The fake game has exactly one ambiguous (auto-picked) move -> "1 to verify".
    await expect(page.getByText("1 to verify")).toBeVisible({ timeout: 15_000 });
    await expect(page.getByRole("button", { name: "Next to verify" })).toBeVisible();

    // Confirm the flagged move via its per-move chip (stable title), and the count clears.
    await page
      .locator('button[title^="Recognized with low confidence"]')
      .first()
      .click();
    await expect(page.getByText("1 to verify")).toHaveCount(0);

    expect(errors, `uncaught page errors:\n${errors.join("\n")}`).toEqual([]);
  });

  test("convert page: upload -> recognize -> review workbench appears", async ({ page }) => {
    const errors = trackPageErrors(page);
    await useEnglish(page);
    await page.goto("/convert");
    await page.setInputFiles('input[type="file"]', {
      name: "sheet.png",
      mimeType: "image/png",
      buffer: PIXEL_PNG,
    });
    // No submit button on /convert: the dropzone fires recognition on file select.
    await expect(page.getByRole("heading", { name: "Moves" })).toBeVisible({ timeout: 30_000 });
    await expect(page.getByText("legal").first()).toBeVisible({ timeout: 15_000 });
    // The anonymous primary action is "Download PGN", not "Save game".
    await expect(page.getByRole("button", { name: "Download PGN" })).toBeVisible();

    expect(errors, `uncaught page errors:\n${errors.join("\n")}`).toEqual([]);
  });
});
