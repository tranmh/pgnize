import { test, expect } from "@playwright/test";
import { useEnglish, trackPageErrors } from "../../helpers/ui";

const SAMPLE_PGN = `[Event "Test"]
[White "Alice"]
[Black "Bob"]
[Result "1-0"]

1. e4 e5 2. Nf3 Nc6 3. Bb5 a6 1-0`;

test.describe("anonymous /new: import → analyze → coach", () => {
  test("paste FEN renders a board with engine eval", async ({ page }) => {
    const errors = trackPageErrors(page);
    await useEnglish(page);
    await page.goto("/new");

    // FEN mode is the default. Paste the starting position and load.
    await page
      .getByLabel("FEN")
      .fill("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1");
    await page.getByRole("button", { name: "Load" }).click();

    // The review workbench mounts with the engine eval control (no moves for a bare FEN).
    await expect(page.getByRole("heading", { name: "Moves" })).toBeVisible({ timeout: 15_000 });

    expect(errors, `uncaught page errors:\n${errors.join("\n")}`).toEqual([]);
  });

  test("import PGN → analyze with the browser engine → Explain a move and coach the game", async ({
    page,
  }) => {
    const errors = trackPageErrors(page);
    await useEnglish(page);
    await page.goto("/new");

    // Switch to import mode and paste a short game.
    await page.getByRole("button", { name: "Import PGN / Lichess" }).click();
    await page.getByLabel("PGN or Lichess URL").fill(SAMPLE_PGN);
    await page.getByRole("button", { name: "Load" }).click();

    // The imported game renders its (server-verified) legal moves.
    await expect(page.getByRole("heading", { name: "Moves" })).toBeVisible({ timeout: 15_000 });
    await expect(page.getByText("legal").first()).toBeVisible({ timeout: 15_000 });

    // Run the browser Stockfish analysis; it must produce per-move annotations.
    await page.getByRole("button", { name: "Analyze game" }).click();

    // Once a move is annotated, its "Explain" button appears. Click it and the coach
    // panel renders prose (the fake coach answers in German — the product default).
    const explain = page.getByRole("button", { name: "Explain" }).first();
    await expect(explain).toBeVisible({ timeout: 45_000 });
    await explain.click();

    await expect(page.getByRole("heading", { name: "Coach", exact: true })).toBeVisible({ timeout: 15_000 });
    await expect(page.getByText(/Die Engine bevorzugt/)).toBeVisible({ timeout: 15_000 });

    // The whole-game summary.
    await page.getByRole("button", { name: "Coach this game" }).click();
    await expect(page.getByText("Game summary")).toBeVisible({ timeout: 15_000 });

    expect(errors, `uncaught page errors:\n${errors.join("\n")}`).toEqual([]);
  });
});
