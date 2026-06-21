import { test, expect } from "@playwright/test";
import { ApiDriver } from "../../helpers/api-driver";

const START = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1";
const SAMPLE_PGN = `[Event "Test"]
[White "Alice"]
[Black "Bob"]
[Result "1-0"]

1. e4 e5 2. Nf3 Nc6 3. Bb5 a6 1-0
`;

test.describe("paste FEN + import (anonymous)", () => {
  // Distinct client IP so per-IP rate limits never bleed across describes.
  test.use({ extraHTTPHeaders: { "X-Forwarded-For": "10.0.7.10" } });

  test("paste FEN returns a position draft", async ({ request }) => {
    const res = await request.post("/api/positions", { data: { fen: START } });
    expect(res.ok(), await res.text()).toBeTruthy();
    const draft = await res.json();
    expect(draft.startFen).toContain("rnbqkbnr");
    expect(Array.isArray(draft.moves)).toBeTruthy();
  });

  test("illegal FEN is rejected", async ({ request }) => {
    const res = await request.post("/api/positions", { data: { fen: "not-a-fen" } });
    expect(res.status()).toBe(400);
  });

  test("import raw PGN returns a verified draft", async ({ request }) => {
    const res = await request.post("/api/import", { data: { pgn: SAMPLE_PGN } });
    expect(res.ok(), await res.text()).toBeTruthy();
    const { games } = await res.json();
    expect(games.length).toBe(1);
    expect(games[0].moves.length).toBe(6);
    expect(games[0].moves.every((m: any) => m.isLegal)).toBeTruthy();
  });
});

test.describe("coach move (anonymous, stateless)", () => {
  test.use({ extraHTTPHeaders: { "X-Forwarded-For": "10.0.7.20" } });

  test("returns prose and is not cached without a gameId", async ({ request }) => {
    const res = await request.post("/api/coach/move", {
      data: {
        fen: START, side: "white", playedSan: "e4", bestSan: "d4",
        evalBefore: { cp: 20 }, evalAfter: { cp: 15 },
      },
    });
    expect(res.ok(), await res.text()).toBeTruthy();
    const body = await res.json();
    expect(body.text.length).toBeGreaterThan(0);
    expect(body.cached).toBeFalsy();
  });
});

test.describe("coach move caching (registered)", () => {
  test.use({ extraHTTPHeaders: { "X-Forwarded-For": "10.0.7.30" } });

  test("second identical call is served from cache", async ({ request }) => {
    const api = new ApiDriver(request);
    await api.registerUnique("coach");

    const posRes = await request.post("/api/positions", { data: { fen: START } });
    expect(posRes.ok(), await posRes.text()).toBeTruthy();
    const draft = await posRes.json();
    expect(draft.id, "logged-in paste FEN should persist a draft id").toBeTruthy();

    const payload = {
      gameId: draft.id, ply: 0, fen: START, side: "white",
      playedSan: "e4", bestSan: "d4", evalBefore: { cp: 20 }, evalAfter: { cp: 15 },
    };
    const first = await (await request.post("/api/coach/move", { data: payload })).json();
    expect(first.cached).toBeFalsy();
    const second = await (await request.post("/api/coach/move", { data: payload })).json();
    expect(second.cached).toBeTruthy();
    expect(second.text).toBe(first.text);
  });
});
