import { test, expect } from "@playwright/test";
import { ApiDriver } from "../../helpers/api-driver";

// Own client IP so register/upload limits don't collide with the convert specs.
test.use({ extraHTTPHeaders: { "X-Forwarded-For": "10.0.3.30" } });

test("account journey: register -> upload -> review -> save -> library -> export", async ({ request }) => {
  const api = new ApiDriver(request);
  await api.register(`acct-${Date.now()}`);

  const { jobId } = await api.upload();
  const job = await api.pollJob(`/api/jobs/${jobId}`);
  expect(job.status).toBe("done");
  expect(job.gameId).toBeTruthy();

  const draft = await api.getGame(job.gameId!);
  expect(draft.moves.every((m) => m.isLegal)).toBeTruthy();
  expect(draft.imageUrl).not.toBe("");

  const moves = draft.moves.map((m, i) => ({ ply: i + 1, san: m.san }));
  const save = await request.patch(`/api/games/${job.gameId}`, {
    data: { header: { white: "Carlsen", black: "Nepo", result: "1-0" }, moves },
  });
  expect(save.ok(), await save.text()).toBeTruthy();

  const list = await (await request.get("/api/games?q=Carlsen")).json();
  expect(list.total).toBe(1);
  expect(list.games[0].white).toBe("Carlsen");

  const pgn = await (await request.get(`/api/games/${job.gameId}/pgn`)).text();
  expect(pgn).toContain("Carlsen");
  expect(pgn).toContain("1-0");
});

test("saving an illegal move is rejected with failedAt", async ({ request }) => {
  const api = new ApiDriver(request);
  await api.register(`illegal-${Date.now()}`);
  const created = await (await request.post("/api/games", { data: { source: "manual" } })).json();

  const res = await request.patch(`/api/games/${created.game.id}`, {
    data: {
      header: { white: "X", black: "Y", result: "*" },
      moves: [
        { ply: 1, san: "e4" },
        { ply: 2, san: "Ke7" }, // illegal: e7 occupied by Black's pawn
      ],
    },
  });
  expect(res.status()).toBe(422);
  const body = await res.json();
  expect(body.error).toBe("illegal_move");
  expect(body.failedAt).toBe(1);
});
