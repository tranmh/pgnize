import { test, expect } from "@playwright/test";
import { ApiDriver } from "../../helpers/api-driver";

test.describe("anonymous convert happy path", () => {
  // Distinct client IP so the rate-limit test below can't starve this one.
  test.use({ extraHTTPHeaders: { "X-Forwarded-For": "10.0.1.10" } });

  test("upload -> recognize -> review -> export PGN", async ({ request }) => {
    const api = new ApiDriver(request);

    const jobId = await api.uploadConvert();
    const job = await api.pollJob(`/api/convert/${jobId}`);
    expect(job.status).toBe("done");

    const gameRes = await request.get(`/api/convert/${jobId}/game`);
    expect(gameRes.ok()).toBeTruthy();
    const draft = await gameRes.json();
    expect(draft.moves.length).toBeGreaterThan(0);
    expect(draft.moves[0].isLegal).toBeTruthy();

    const moves = draft.moves.map((m: any, i: number) => ({ ply: i + 1, san: m.san }));
    const exp = await request.post(`/api/convert/${jobId}/export`, {
      data: { header: { white: "Anon", black: "Mouse", result: "*" }, moves },
    });
    expect(exp.ok()).toBeTruthy();
    const pgn = await exp.text();
    expect(pgn).toContain("Anon");
    expect(pgn).toContain("1. e4");
  });
});

test.describe("anonymous convert rate limiting", () => {
  test.use({ extraHTTPHeaders: { "X-Forwarded-For": "10.0.2.20" } });

  test("trips a 429 within the hourly budget", async ({ request }) => {
    const api = new ApiDriver(request);
    let limited = false;
    for (let i = 0; i < 12; i++) {
      try {
        await api.uploadConvert();
      } catch {
        limited = true;
        break;
      }
    }
    expect(limited, "expected a 429 within 12 anonymous converts").toBeTruthy();
  });
});
