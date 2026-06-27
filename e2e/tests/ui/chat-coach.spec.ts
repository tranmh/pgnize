import { test, expect, Page } from "@playwright/test";
import { useEnglish, trackPageErrors } from "../../helpers/ui";

const MIDGAME = "1r6/5pp1/R1R4p/1r1pP3/2pkQPP1/7P/1P6/2K5 w - - 0 41";

// Load a pasted position into the review workbench and open the chat panel.
async function openChat(page: Page) {
  await page.getByLabel("FEN").fill(MIDGAME);
  await page.getByRole("button", { name: "Load" }).click();
  await expect(page.getByRole("heading", { name: "Moves" })).toBeVisible({ timeout: 15_000 });
  await page.getByTestId("chat-coach-toggle").click();
  await expect(page.getByTestId("chat-coach-input")).toBeVisible();
}

test.describe("conversational coach (UI)", () => {
  test("typed multi-turn Q&A renders user + coach bubbles", async ({ page }) => {
    const errors = trackPageErrors(page);
    await useEnglish(page);
    await page.goto("/new");
    await openChat(page);

    await page.getByTestId("chat-coach-input").fill("What is the best move?");
    await page.getByTestId("chat-coach-send").click();

    await expect(page.getByTestId("chat-msg-user").first()).toContainText("best move", { timeout: 10_000 });
    const coach = page.getByTestId("chat-msg-coach").first();
    await expect(coach).toBeVisible({ timeout: 45_000 });
    expect((await coach.innerText()).trim().length).toBeGreaterThan(5);
    await expect(page.getByTestId("chat-coach-error")).toHaveCount(0);

    // Second turn proves multi-turn: a growing thread.
    await page.getByTestId("chat-coach-input").fill("And why?");
    await page.getByTestId("chat-coach-send").click();
    await expect.poll(async () => page.getByTestId("chat-msg-coach").count(), { timeout: 45_000 }).toBeGreaterThanOrEqual(2);

    expect(errors, `uncaught page errors:\n${errors.join("\n")}`).toEqual([]);
  });

  test("coach answer auto-speaks via /coach/speak", async ({ page }) => {
    await useEnglish(page);
    await page.goto("/new");
    await openChat(page);

    const speakReq = page.waitForRequest("**/api/coach/speak", { timeout: 45_000 });
    await page.getByTestId("chat-coach-input").fill("Best move?");
    await page.getByTestId("chat-coach-send").click();
    await expect(page.getByTestId("chat-msg-coach").first()).toBeVisible({ timeout: 45_000 });
    const req = await speakReq;
    expect(req.method()).toBe("POST");
  });

  test("browser STT (injected) drops a transcript into the input, then sends", async ({ page }) => {
    await useEnglish(page);
    // Force browser STT + inject a fake SpeechRecognition that fires a canned final result.
    await page.addInitScript(() => {
      try {
        localStorage.setItem("pgnize.stt.source", "browser");
      } catch {
        /* ignore */
      }
      class FakeRecognition {
        lang = "";
        interimResults = false;
        continuous = false;
        onresult: ((e: unknown) => void) | null = null;
        onerror: ((e: unknown) => void) | null = null;
        onend: (() => void) | null = null;
        start() {
          // Fire a final transcript synchronously, then end.
          this.onresult?.({
            resultIndex: 0,
            results: [{ 0: { transcript: "Is there a mate combination" }, isFinal: true }],
          });
          this.onend?.();
        }
        stop() {
          this.onend?.();
        }
        abort() {}
      }
      (window as unknown as { SpeechRecognition: unknown }).SpeechRecognition = FakeRecognition;
    });

    await page.goto("/new");
    await openChat(page);

    await page.getByTestId("chat-coach-mic").click();
    await expect(page.getByTestId("chat-coach-input")).toHaveValue(/mate combination/i, {
      timeout: 10_000,
    });
    await page.getByTestId("chat-coach-send").click();
    await expect(page.getByTestId("chat-msg-coach").first()).toBeVisible({ timeout: 45_000 });
  });

  test("mic is disabled when no voice input is available", async ({ page }) => {
    await useEnglish(page);
    // Remove both voice paths before any app code runs.
    await page.addInitScript(() => {
      try {
        delete (window as unknown as Record<string, unknown>).SpeechRecognition;
        delete (window as unknown as Record<string, unknown>).webkitSpeechRecognition;
        delete (window as unknown as Record<string, unknown>).MediaRecorder;
      } catch {
        /* ignore */
      }
      Object.defineProperty(navigator, "mediaDevices", { value: undefined, configurable: true });
    });

    await page.goto("/new");
    await openChat(page);

    await expect(page.getByTestId("chat-coach-mic")).toBeDisabled();
    // Text still works.
    await page.getByTestId("chat-coach-input").fill("Best move?");
    await page.getByTestId("chat-coach-send").click();
    await expect(page.getByTestId("chat-msg-coach").first()).toBeVisible({ timeout: 45_000 });
  });
});
