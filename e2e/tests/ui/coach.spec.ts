import { test, expect } from "@playwright/test";
import { useEnglish, trackPageErrors } from "../../helpers/ui";

const SAMPLE_PGN = `[Event "Test"]
[White "Alice"]
[Black "Bob"]
[Result "1-0"]

1. e4 e5 2. Nf3 Nc6 3. Bb5 a6 1-0`;

test.describe("anonymous /new: import → analyze → coach", () => {
  test("paste a position → Coach this position → prose (no moves needed)", async ({ page }) => {
    const errors = trackPageErrors(page);
    await useEnglish(page);
    await page.goto("/new");

    // A real mid-game position (the user's pasted FEN), which has no move list.
    await page
      .getByLabel("FEN")
      .fill("1r6/5pp1/R1R4p/1r1pP3/2pkQPP1/7P/1P6/2K5 w - - 0 41");
    await page.getByRole("button", { name: "Load" }).click();

    await expect(page.getByRole("heading", { name: "Moves" })).toBeVisible({ timeout: 15_000 });

    // The "Coach this position" affordance must exist for a no-moves draft and produce prose.
    const coachPos = page.getByRole("button", { name: "Coach this position" });
    await expect(coachPos).toBeVisible({ timeout: 15_000 });
    await coachPos.click();

    const prose = page.getByTestId("coach-position-text");
    await expect(prose).toBeVisible({ timeout: 45_000 });
    expect((await prose.innerText()).trim().length).toBeGreaterThan(10);
    await expect(page.getByText("Coaching failed. Please try again.")).toHaveCount(0);

    // Speech controls render alongside the visible coach text.
    await expect(page.getByTestId("coach-speak-toggle")).toBeVisible();
    await expect(page.getByTestId("coach-speak-replay")).toBeVisible();

    expect(errors, `uncaught page errors:\n${errors.join("\n")}`).toEqual([]);
  });

  test("TTS: global toggle renders and auto-speak fires a /coach/speak request", async ({
    page,
  }) => {
    const errors = trackPageErrors(page);
    await useEnglish(page);

    // NOTE: the app registers a service worker, and SW-issued fetches bypass
    // Playwright's page/context `route` interception — so a `route` counter is
    // non-deterministic (it can stay 0 while the request truly fires). Instead
    // we prove auto-speak via `page.waitForRequest` (which observes the network
    // regardless of who issues it) and let the real fake TTS backend serve the
    // synthesize + audio responses. This is both deterministic and a truer e2e.

    await page.goto("/new");

    // The global speech toggle lives in the nav (server source by default).
    await expect(page.getByTestId("speech-toggle")).toBeVisible();

    await page
      .getByLabel("FEN")
      .fill("1r6/5pp1/R1R4p/1r1pP3/2pkQPP1/7P/1P6/2K5 w - - 0 41");
    await page.getByRole("button", { name: "Load" }).click();

    await expect(page.getByRole("heading", { name: "Moves" })).toBeVisible({ timeout: 15_000 });

    const coachPos = page.getByRole("button", { name: "Coach this position" });
    await expect(coachPos).toBeVisible({ timeout: 15_000 });

    const speakReq = page.waitForRequest("**/api/coach/speak", { timeout: 45_000 });
    await coachPos.click();

    // Prose appears, then auto-speak fires a POST /coach/speak request.
    await expect(page.getByTestId("coach-position-text")).toBeVisible({ timeout: 45_000 });
    const req = await speakReq;
    expect(req.method()).toBe("POST");

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
