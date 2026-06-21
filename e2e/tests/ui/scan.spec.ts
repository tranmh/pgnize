import { test, expect } from "@playwright/test";
import { useEnglish, trackPageErrors, PIXEL_PNG } from "../../helpers/ui";

test.describe("anonymous scan flow", () => {
  test("scan page: upload -> recognize -> position editor mounts, edit, download PGN", async ({
    page,
  }) => {
    const errors = trackPageErrors(page);
    await useEnglish(page);
    await page.goto("/scan");

    await page.setInputFiles('input[type="file"]', {
      name: "board.png",
      mimeType: "image/png",
      buffer: PIXEL_PNG,
    });
    // The multi-image picker collects the file; recognition fires on submit.
    await page.getByRole("button", { name: "Scan" }).click();

    // The position editor must mount: palette, board, castling checkboxes, and
    // the side-to-move radios are all visible.
    await expect(page.getByText("Pieces")).toBeVisible({ timeout: 30_000 });
    await expect(page.locator('input[type="checkbox"]')).toHaveCount(4);
    await expect(page.getByLabel("White player")).toBeVisible();
    // Four castling checkboxes + side-to-move radios.
    await expect(page.getByRole("radio")).toHaveCount(2);
    await expect(
      page.getByRole("button", { name: "Starting position" }),
    ).toBeVisible();

    // Pick a palette piece, then click a board square — must not crash.
    await page.getByRole("button", { name: "K", exact: true }).click();
    // react-chessboard renders square targets with data-square attributes.
    await page.locator('[data-square="e4"]').first().click();

    // The anonymous primary action downloads the PGN.
    const downloadButton = page.getByRole("button", { name: "Download PGN" });
    await expect(downloadButton).toBeVisible();
    const download = page.waitForEvent("download");
    await downloadButton.click();
    await download;

    expect(errors, `uncaught page errors:\n${errors.join("\n")}`).toEqual([]);
  });
});
