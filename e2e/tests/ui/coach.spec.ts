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

    await page
      .getByLabel("FEN")
      .fill("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1");
    await page.getByRole("button", { name: "Load" }).click();

    await expect(page.getByRole("heading", { name: "Moves" })).toBeVisible({ timeout: 15_000 });
    expect(errors, `uncaught page errors:\n${errors.join("\n")}`).toEqual([]);
  });

  test("import PGN → analyze → Explain every move (no coach errors) → coach the game", async ({
    page,
  }) => {
    const errors = trackPageErrors(page);
    await useEnglish(page);
    await page.goto("/new");

    await page.getByRole("button", { name: "Import PGN / Lichess" }).click();
    await page.getByLabel("PGN or Lichess URL").fill(SAMPLE_PGN);
    await page.getByRole("button", { name: "Load" }).click();

    await expect(page.getByRole("heading", { name: "Moves" })).toBeVisible({ timeout: 15_000 });
    await expect(page.getByText("legal").first()).toBeVisible({ timeout: 15_000 });

    // Browser Stockfish analysis must produce per-move annotations.
    await page.getByRole("button", { name: "Analyze game" }).click();

    // Explain EVERY move. Each must yield prose, never the "Coaching failed" error —
    // this is the regression net for the empty-best-move / mid-analysis race.
    const explainButtons = page.getByRole("button", { name: "Explain" });
    await expect(explainButtons.first()).toBeVisible({ timeout: 45_000 });

    // Wait for the full legal prefix (6 plies) to be analyzed so all Explain buttons exist.
    await expect.poll(async () => explainButtons.count(), { timeout: 45_000 }).toBeGreaterThanOrEqual(6);
    const count = await explainButtons.count();

    for (let i = 0; i < count; i++) {
      await explainButtons.nth(i).click();
      // Backend-agnostic: assert real prose appears (works for fake + gemini), and the
      // coach error never shows.
      const prose = page.getByTestId("coach-move-text");
      await expect(prose).toBeVisible({ timeout: 30_000 });
      expect((await prose.innerText()).trim().length, `move ${i} prose`).toBeGreaterThan(10);
      await expect(page.getByText("Coaching failed. Please try again.")).toHaveCount(0);
    }

    // Whole-game summary.
    await page.getByRole("button", { name: "Coach this game" }).click();
    const summary = page.getByTestId("coach-game-text");
    await expect(summary).toBeVisible({ timeout: 30_000 });
    expect((await summary.innerText()).trim().length).toBeGreaterThan(10);

    expect(errors, `uncaught page errors:\n${errors.join("\n")}`).toEqual([]);
  });
});
