import { test, expect } from "@playwright/test";
import { freshApi } from "../../helpers/api-driver";

const OCTET = 21;

test.describe("manual game lifecycle", () => {
  test("create returns moves: [] (never null) and source manual", async ({}, info) => {
    const { ctx, api } = await freshApi(OCTET, info.workerIndex);
    await api.registerUnique("man");
    const res = await ctx.post("/api/games", { data: { source: "manual" } });
    expect(res.status()).toBe(201);
    const body = await res.json();
    expect(body.game.moves, "moves must be an array, never null").toEqual([]);
    expect(body.game.source).toBe("manual");
    expect(body.game.result || body.game.header?.result).toBeTruthy();
    // And a fresh GET also returns an array.
    const got = await api.getGame(body.game.id);
    expect(Array.isArray(got.moves)).toBeTruthy();
    await ctx.dispose();
  });

  test("save legal moves, reload shows FENs + isLegal, PGN is canonical", async ({}, info) => {
    const { ctx, api } = await freshApi(OCTET, info.workerIndex);
    await api.registerUnique("save");
    const gid = await api.createManual();
    const save = await ctx.patch(`/api/games/${gid}`, {
      data: {
        header: { white: "Carlsen, M", black: "Nepo, I", result: "1-0", event: "Test" },
        moves: [
          { ply: 1, san: "e4" },
          { ply: 2, san: "e5" },
          { ply: 3, san: "Nf3" },
          { ply: 4, san: "Nc6" },
        ],
      },
    });
    expect(save.status(), await save.text()).toBe(200);

    const got = await api.getGame(gid);
    expect(got.status).toBe("saved");
    expect(got.moves).toHaveLength(4);
    expect(got.moves.every((m) => m.isLegal)).toBeTruthy();
    expect(got.moves[0].fenAfter ?? got.moves[0]["fenAfter"]).toBeTruthy();
    expect(got.moves[0].side).toBe("white");
    expect(got.moves[1].side).toBe("black");

    const pgn = await (await ctx.get(`/api/games/${gid}/pgn`)).text();
    expect(pgn).toContain("Carlsen, M");
    expect(pgn).toContain("1. e4 e5 2. Nf3 Nc6");
    expect(pgn.trimEnd().endsWith("1-0"), `PGN should end with the result:\n${pgn}`).toBeTruthy();
    await ctx.dispose();
  });

  test("clockSec round-trips through save + reload", async ({}, info) => {
    const { ctx, api } = await freshApi(OCTET, info.workerIndex);
    await api.registerUnique("clock");
    const gid = await api.createManual();
    await ctx.patch(`/api/games/${gid}`, {
      data: {
        header: { white: "W", black: "B", result: "*" },
        moves: [
          { ply: 1, san: "e4", clockSec: 300 },
          { ply: 2, san: "e5", clockSec: 290 },
        ],
      },
    });
    const got = await api.getGame(gid);
    expect(got.moves.map((m) => m.clockSec)).toEqual([300, 290]);
    await ctx.dispose();
  });

  test("save with empty moves persists a header-only game with a result", async ({}, info) => {
    const { ctx, api } = await freshApi(OCTET, info.workerIndex);
    await api.registerUnique("empty");
    const gid = await api.createManual();
    const save = await ctx.patch(`/api/games/${gid}`, {
      data: { header: { white: "Alice", black: "Bob", result: "1/2-1/2" }, moves: [] },
    });
    expect(save.status()).toBe(200);
    const pgn = await (await ctx.get(`/api/games/${gid}/pgn`)).text();
    expect(pgn).toContain("Alice");
    expect(pgn).toContain("1/2-1/2");
    await ctx.dispose();
  });

  test("custom startFen is preserved and emits SetUp/FEN tags", async ({}, info) => {
    const { ctx, api } = await freshApi(OCTET, info.workerIndex);
    await api.registerUnique("fen");
    const gid = await api.createManual();
    const fen = "r1bqk2r/pppp1Qpp/2n2n2/2b1p3/2B1P3/8/PPPP1PPP/RNB1K1NR b KQkq - 0 4";
    const save = await ctx.patch(`/api/games/${gid}`, {
      data: { header: { white: "W", black: "B", result: "1-0" }, startFen: fen, moves: [] },
    });
    expect(save.status(), await save.text()).toBe(200);
    const got = await api.getGame(gid);
    expect(got.startFen).toBe(fen);
    const pgn = await (await ctx.get(`/api/games/${gid}/pgn`)).text();
    expect(pgn).toContain(`[FEN "${fen}"]`);
    expect(pgn).toContain('[SetUp "1"]');
    await ctx.dispose();
  });

  test("re-saving with fewer moves replaces the move list", async ({}, info) => {
    const { ctx, api } = await freshApi(OCTET, info.workerIndex);
    await api.registerUnique("reshrink");
    const gid = await api.createManual();
    await ctx.patch(`/api/games/${gid}`, {
      data: {
        header: { white: "W", black: "B", result: "*" },
        moves: [
          { ply: 1, san: "e4" },
          { ply: 2, san: "e5" },
          { ply: 3, san: "Nf3" },
          { ply: 4, san: "Nc6" },
        ],
      },
    });
    expect((await api.getGame(gid)).moves).toHaveLength(4);
    await ctx.patch(`/api/games/${gid}`, {
      data: { header: { white: "W", black: "B", result: "*" }, moves: [{ ply: 1, san: "d4" }] },
    });
    const got = await api.getGame(gid);
    expect(got.moves).toHaveLength(1);
    expect(got.moves[0].san).toBe("d4");
    await ctx.dispose();
  });

  test("delete removes the game (get/pgn become 404, gone from library)", async ({}, info) => {
    const { ctx, api } = await freshApi(OCTET, info.workerIndex);
    await api.registerUnique("del");
    const marker = `Del${Date.now()}`;
    const gid = await api.createManual();
    await ctx.patch(`/api/games/${gid}`, {
      data: { header: { white: marker, black: "Z", result: "*" }, moves: [] },
    });
    expect((await ctx.delete(`/api/games/${gid}`)).status()).toBe(204);
    expect((await ctx.get(`/api/games/${gid}`)).status()).toBe(404);
    expect((await ctx.get(`/api/games/${gid}/pgn`)).status()).toBe(404);
    const list = await (await ctx.get(`/api/games?q=${marker}`)).json();
    expect(list.total).toBe(0);
    await ctx.dispose();
  });
});

test.describe("server-authoritative move validation", () => {
  test("first move illegal -> 422 failedAt 0", async ({}, info) => {
    const { ctx, api } = await freshApi(OCTET, info.workerIndex);
    await api.registerUnique("ill0");
    const gid = await api.createManual();
    const res = await ctx.patch(`/api/games/${gid}`, {
      data: { header: { white: "X", black: "Y", result: "*" }, moves: [{ ply: 1, san: "e5" }] },
    });
    expect(res.status()).toBe(422);
    const body = await res.json();
    expect(body.error).toBe("illegal_move");
    expect(body.failedAt).toBe(0);
    await ctx.dispose();
  });

  test("a later illegal move reports its index in failedAt", async ({}, info) => {
    const { ctx, api } = await freshApi(OCTET, info.workerIndex);
    await api.registerUnique("ill2");
    const gid = await api.createManual();
    const res = await ctx.patch(`/api/games/${gid}`, {
      data: {
        header: { white: "X", black: "Y", result: "*" },
        moves: [
          { ply: 1, san: "e4" },
          { ply: 2, san: "e5" },
          { ply: 3, san: "Qh6" }, // illegal: queen cannot reach h6
        ],
      },
    });
    expect(res.status()).toBe(422);
    expect((await res.json()).failedAt).toBe(2);
    await ctx.dispose();
  });

  test("a rejected save does not mutate the stored game", async ({}, info) => {
    const { ctx, api } = await freshApi(OCTET, info.workerIndex);
    await api.registerUnique("illnomut");
    const gid = await api.createManual();
    // First, a good save.
    await ctx.patch(`/api/games/${gid}`, {
      data: { header: { white: "Good", black: "State", result: "*" }, moves: [{ ply: 1, san: "e4" }] },
    });
    // Then a bad save that must be rejected.
    const bad = await ctx.patch(`/api/games/${gid}`, {
      data: { header: { white: "Bad", black: "State", result: "*" }, moves: [{ ply: 1, san: "Ke3" }] },
    });
    expect(bad.status()).toBe(422);
    // The stored game must still reflect the last *successful* save.
    const got = await api.getGame(gid);
    expect(got.header.white).toBe("Good");
    expect(got.moves).toHaveLength(1);
    expect(got.moves[0].san).toBe("e4");
    await ctx.dispose();
  });
});

test.describe("recognized game review + save (feedback path)", () => {
  test("upload, recognize, save the reviewed moves -> 200 and library + PGN", async ({}, info) => {
    const { ctx, api } = await freshApi(OCTET, info.workerIndex);
    await api.registerUnique("rec");
    const { gameId, draft } = await api.uploadAndRecognize();
    expect(draft.source).toBe("recognized");
    expect(draft.moves.length).toBeGreaterThan(0);
    // Castling SAN from the recognizer must be replayable on export.
    expect(draft.moves.some((m) => m.san === "O-O")).toBeTruthy();

    const moves = draft.moves.map((m, i) => ({ ply: i + 1, san: m.san }));
    const save = await ctx.patch(`/api/games/${gameId}`, {
      data: { header: { white: "Rec White", black: "Rec Black", result: "1-0" }, moves },
    });
    expect(save.status(), await save.text()).toBe(200);

    const pgn = await (await ctx.get(`/api/games/${gameId}/pgn`)).text();
    expect(pgn).toContain("O-O");
    expect(pgn).toContain("Rec White");
    await ctx.dispose();
  });
});
