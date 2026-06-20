import { test, expect } from "@playwright/test";
import { freshApi, ApiDriver } from "../../helpers/api-driver";
import type { APIRequestContext } from "@playwright/test";

const OCTET = 22;

// saveGame creates+saves a manual game with the given header and returns its id.
async function saveGame(
  ctx: APIRequestContext,
  api: ApiDriver,
  header: Record<string, string>,
  moves: { ply: number; san: string }[] = [],
) {
  const gid = await api.createManual();
  const res = await ctx.patch(`/api/games/${gid}`, { data: { header, moves } });
  expect(res.status(), await res.text()).toBe(200);
  return gid;
}

test.describe("library listing, search and filters", () => {
  test("search q matches white, black, and event substrings", async ({}, info) => {
    const { ctx, api } = await freshApi(OCTET, info.workerIndex);
    await api.registerUnique("search");
    const tag = `S${Date.now()}`;
    await saveGame(ctx, api, { white: `${tag}White`, black: "Zzz", result: "*" });
    await saveGame(ctx, api, { white: "Yyy", black: `${tag}Black`, result: "*" });
    await saveGame(ctx, api, { white: "Aaa", black: "Bbb", event: `${tag}Event`, result: "*" });

    const all = await (await ctx.get(`/api/games?q=${tag}`)).json();
    expect(all.total, "q should match white, black and event").toBe(3);
    await ctx.dispose();
  });

  test("player filter matches either color; event filter matches event only", async ({}, info) => {
    const { ctx, api } = await freshApi(OCTET, info.workerIndex);
    await api.registerUnique("filter");
    const tag = `F${Date.now()}`;
    await saveGame(ctx, api, { white: `${tag}Magnus`, black: "Other", event: "X", result: "*" });
    await saveGame(ctx, api, { white: "Other", black: `${tag}Magnus`, event: "Y", result: "*" });
    await saveGame(ctx, api, { white: "Nobody", black: "Nobody", event: `${tag}Open`, result: "*" });

    const byPlayer = await (await ctx.get(`/api/games?player=${tag}Magnus`)).json();
    expect(byPlayer.total, "player filter should match white OR black").toBe(2);

    const byEvent = await (await ctx.get(`/api/games?event=${tag}Open`)).json();
    expect(byEvent.total).toBe(1);
    expect(byEvent.games[0].event).toBe(`${tag}Open`);
    await ctx.dispose();
  });

  test("date from/to filter on event_date (PGN dotted format)", async ({}, info) => {
    const { ctx, api } = await freshApi(OCTET, info.workerIndex);
    await api.registerUnique("date");
    const tag = `D${Date.now()}`;
    await saveGame(ctx, api, { white: `${tag}A`, black: "x", date: "2020.05.05", result: "*" });
    await saveGame(ctx, api, { white: `${tag}B`, black: "x", date: "2026.05.05", result: "*" });

    const r = await (await ctx.get(`/api/games?q=${tag}&from=2025.01.01&to=2027.01.01`)).json();
    expect(r.total, "only the 2026 game falls in [2025,2027]").toBe(1);
    expect(r.games[0].white).toBe(`${tag}B`);
    await ctx.dispose();
  });

  test("pagination: total is the full count; pages slice correctly", async ({}, info) => {
    const { ctx, api } = await freshApi(OCTET, info.workerIndex);
    await api.registerUnique("page");
    const tag = `P${Date.now()}`;
    for (let i = 0; i < 5; i++) {
      await saveGame(ctx, api, { white: `${tag}`, black: `n${i}`, result: "*" });
    }
    const p1 = await (await ctx.get(`/api/games?q=${tag}&page=1&pageSize=2`)).json();
    expect(p1.total).toBe(5);
    expect(p1.games).toHaveLength(2);
    expect(p1.page).toBe(1);
    expect(p1.pageSize).toBe(2);

    const p3 = await (await ctx.get(`/api/games?q=${tag}&page=3&pageSize=2`)).json();
    expect(p3.total).toBe(5);
    expect(p3.games, "last page holds the remaining 1").toHaveLength(1);

    // Out-of-range page returns an empty slice but the correct total.
    const p9 = await (await ctx.get(`/api/games?q=${tag}&page=9&pageSize=2`)).json();
    expect(p9.total).toBe(5);
    expect(p9.games).toHaveLength(0);
    await ctx.dispose();
  });

  test("garbage page/pageSize fall back to sane defaults", async ({}, info) => {
    const { ctx, api } = await freshApi(OCTET, info.workerIndex);
    await api.registerUnique("pagedef");
    const r = await (await ctx.get(`/api/games?page=-3&pageSize=0`)).json();
    expect(r.page).toBe(1);
    expect(r.pageSize).toBe(20);
    await ctx.dispose();
  });
});

test.describe("players autocomplete", () => {
  test("saving a game upserts both players into the caller's pool", async ({}, info) => {
    const { ctx, api } = await freshApi(OCTET, info.workerIndex);
    await api.registerUnique("players");
    const tag = `Pl${Date.now()}`;
    await saveGame(ctx, api, { white: `${tag}, Magnus`, black: `${tag}, Hikaru`, result: "*" });

    const res = await ctx.get(`/api/players?q=${tag}`);
    expect(res.status()).toBe(200);
    const players = (await res.json()).players;
    expect(players.length, "both white and black should be searchable").toBeGreaterThanOrEqual(2);
    await ctx.dispose();
  });

  test("players are scoped to the owning user", async ({}, info) => {
    const a = await freshApi(OCTET, info.workerIndex);
    const b = await freshApi(OCTET, info.workerIndex);
    await a.api.registerUnique("plA");
    await b.api.registerUnique("plB");
    const tag = `Scope${Date.now()}`;
    await saveGame(a.ctx, a.api, { white: `${tag}Secret`, black: "x", result: "*" });

    const bPlayers = (await (await b.ctx.get(`/api/players?q=${tag}`)).json()).players;
    expect(bPlayers, "user B must not see user A's players").toHaveLength(0);
    await a.ctx.dispose();
    await b.ctx.dispose();
  });
});

test.describe("export bundle", () => {
  test("concatenates the PGNs of the requested owned games", async ({}, info) => {
    const { ctx, api } = await freshApi(OCTET, info.workerIndex);
    await api.registerUnique("bundle");
    const tag = `B${Date.now()}`;
    const g1 = await saveGame(ctx, api, { white: `${tag}One`, black: "x", result: "1-0" }, [{ ply: 1, san: "e4" }]);
    const g2 = await saveGame(ctx, api, { white: `${tag}Two`, black: "y", result: "0-1" }, [{ ply: 1, san: "d4" }]);

    const res = await ctx.post("/api/games/export", { data: { ids: [g1, g2] } });
    expect(res.status()).toBe(200);
    const pgn = await res.text();
    expect(pgn).toContain(`${tag}One`);
    expect(pgn).toContain(`${tag}Two`);
    await ctx.dispose();
  });

  test("empty ids -> 400", async ({}, info) => {
    const { ctx, api } = await freshApi(OCTET, info.workerIndex);
    await api.registerUnique("bundle0");
    const res = await ctx.post("/api/games/export", { data: { ids: [] } });
    expect(res.status()).toBe(400);
    await ctx.dispose();
  });

  test("cannot export another user's game via the bundle endpoint", async ({}, info) => {
    const a = await freshApi(OCTET, info.workerIndex);
    const b = await freshApi(OCTET, info.workerIndex);
    await a.api.registerUnique("bunA");
    await b.api.registerUnique("bunB");
    const tag = `Leak${Date.now()}`;
    const aGame = await saveGame(a.ctx, a.api, { white: `${tag}Private`, black: "x", result: "1-0" }, [{ ply: 1, san: "e4" }]);

    const res = await b.ctx.post("/api/games/export", { data: { ids: [aGame] } });
    const pgn = await res.text();
    expect(pgn, "B must not receive A's PGN").not.toContain(`${tag}Private`);
    await a.ctx.dispose();
    await b.ctx.dispose();
  });
});
