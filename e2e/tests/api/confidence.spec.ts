import { test, expect } from "@playwright/test";
import { ApiDriver } from "../../helpers/api-driver";

// The fake recognizer (RECOGNIZER=fake) yields a game whose 5th half-move is an ambiguous
// "Nc3" (both knights reach c3). The pipeline auto-picks a disambiguation and flags it with
// low confidence, while clean moves are confident. This verifies the confidence contract
// end-to-end through HTTP + DB + Reconcile.
test.describe("per-move confidence", () => {
  test.use({ extraHTTPHeaders: { "X-Forwarded-For": "10.0.3.30" } });

  test("recognized moves carry confidence; ambiguous move is flagged for verify", async ({
    request,
  }) => {
    const api = new ApiDriver(request);
    const jobId = await api.uploadConvert();
    const job = await api.pollJob(`/api/convert/${jobId}`);
    expect(job.status).toBe("done");

    const res = await request.get(`/api/convert/${jobId}/game`);
    expect(res.ok()).toBeTruthy();
    const draft = await res.json();

    // Every move has a confidence in [0, 1].
    for (const m of draft.moves) {
      expect(typeof m.confidence).toBe("number");
      expect(m.confidence).toBeGreaterThanOrEqual(0);
      expect(m.confidence).toBeLessThanOrEqual(1);
    }

    // Clean legal moves are confident (>= 0.6 threshold).
    const clean = draft.moves.filter((m: any) => m.isLegal && !m.corrected);
    expect(clean.length).toBeGreaterThan(0);
    for (const m of clean) expect(m.confidence).toBeGreaterThanOrEqual(0.6);

    // Exactly the auto-picked ambiguous move is legal-but-low-confidence ("verify").
    const verify = draft.moves.filter(
      (m: any) => m.isLegal && m.confidence < 0.6,
    );
    expect(verify).toHaveLength(1);
    expect(verify[0].corrected).toBe(true);
    expect(["Nbd2", "Nfd2"]).toContain(verify[0].san);
  });
});
