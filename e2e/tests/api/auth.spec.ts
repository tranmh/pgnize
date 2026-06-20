import { test, expect } from "@playwright/test";
import { freshApi } from "../../helpers/api-driver";

// Each test gets its own client IP (fileOctet 20) so register/login rate limits don't collide.
const OCTET = 20;

test.describe("auth: registration validation", () => {
  test("rejects a short password with 400", async ({}, info) => {
    const { ctx } = await freshApi(OCTET, info.workerIndex);
    const res = await ctx.post("/api/auth/register", {
      data: { name: "Short", email: `short-${Date.now()}@e.com`, password: "1234567" },
    });
    expect(res.status()).toBe(400);
    expect((await res.json()).error).toBe("invalid_input");
    await ctx.dispose();
  });

  test("rejects a blank name with 400", async ({}, info) => {
    const { ctx } = await freshApi(OCTET, info.workerIndex);
    const res = await ctx.post("/api/auth/register", {
      data: { name: "   ", email: `blank-${Date.now()}@e.com`, password: "password12" },
    });
    expect(res.status()).toBe(400);
    await ctx.dispose();
  });

  test("rejects a missing email with 400", async ({}, info) => {
    const { ctx } = await freshApi(OCTET, info.workerIndex);
    const res = await ctx.post("/api/auth/register", {
      data: { name: "NoEmail", password: "password12" },
    });
    expect(res.status()).toBe(400);
    await ctx.dispose();
  });

  test("rejects invalid JSON with 400", async ({}, info) => {
    const { ctx } = await freshApi(OCTET, info.workerIndex);
    const res = await ctx.post("/api/auth/register", {
      headers: { "Content-Type": "application/json" },
      data: "{not json",
    });
    expect(res.status()).toBe(400);
    await ctx.dispose();
  });

  test("duplicate email returns 409 email_taken", async ({}, info) => {
    const { ctx, api } = await freshApi(OCTET, info.workerIndex);
    const email = await api.registerUnique("dup");
    const res = await ctx.post("/api/auth/register", {
      data: { name: "Dup2", email, password: "password12" },
    });
    expect(res.status()).toBe(409);
    expect((await res.json()).error).toBe("email_taken");
    await ctx.dispose();
  });

  test("email is treated case-insensitively (UPPER == lower)", async ({}, info) => {
    const { ctx } = await freshApi(OCTET, info.workerIndex);
    const base = `mixed-${Date.now()}@e.com`;
    const r1 = await ctx.post("/api/auth/register", {
      data: { name: "Mixed", email: base.toUpperCase(), password: "password12" },
    });
    expect(r1.status()).toBe(201);
    // Registering the lowercase variant must collide.
    const r2 = await ctx.post("/api/auth/register", {
      data: { name: "Mixed2", email: base.toLowerCase(), password: "password12" },
    });
    expect(r2.status(), "lower-case variant of an existing email must be taken").toBe(409);
    await ctx.dispose();
  });
});

test.describe("auth: login + sessions", () => {
  test("login with the registered (lower-cased) email succeeds", async ({}, info) => {
    const { ctx } = await freshApi(OCTET, info.workerIndex);
    const email = `login-${Date.now()}@e.com`;
    await ctx.post("/api/auth/register", { data: { name: "L", email: email.toUpperCase(), password: "password12" } });
    // Log out the session the registration created, then log back in.
    await ctx.post("/api/auth/logout");
    const res = await ctx.post("/api/auth/login", { data: { email: email.toLowerCase(), password: "password12" } });
    expect(res.status(), await res.text()).toBe(200);
    await ctx.dispose();
  });

  test("wrong password returns 401", async ({}, info) => {
    const { ctx, api } = await freshApi(OCTET, info.workerIndex);
    const email = await api.registerUnique("wp");
    await ctx.post("/api/auth/logout");
    const res = await ctx.post("/api/auth/login", { data: { email, password: "wrongpassword" } });
    expect(res.status()).toBe(401);
    expect((await res.json()).error).toBe("invalid_credentials");
    await ctx.dispose();
  });

  test("unknown account returns 401", async ({}, info) => {
    const { ctx } = await freshApi(OCTET, info.workerIndex);
    const res = await ctx.post("/api/auth/login", {
      data: { email: `ghost-${Date.now()}@e.com`, password: "password12" },
    });
    expect(res.status()).toBe(401);
    await ctx.dispose();
  });

  test("/auth/me is 401 anonymous and 200 after login", async ({}, info) => {
    const { ctx, api } = await freshApi(OCTET, info.workerIndex);
    const anon = await ctx.get("/api/auth/me");
    expect(anon.status()).toBe(401);
    await api.registerUnique("me");
    const me = await ctx.get("/api/auth/me");
    expect(me.status()).toBe(200);
    expect((await me.json()).user.email).toContain("me-");
    await ctx.dispose();
  });

  test("logout clears the session: protected routes become 401", async ({}, info) => {
    const { ctx, api } = await freshApi(OCTET, info.workerIndex);
    await api.registerUnique("lo");
    expect((await ctx.get("/api/games")).status()).toBe(200);
    const out = await ctx.post("/api/auth/logout");
    expect(out.status()).toBe(204);
    expect((await ctx.get("/api/games")).status()).toBe(401);
    expect((await ctx.get("/api/auth/me")).status()).toBe(401);
    await ctx.dispose();
  });
});

test.describe("authorization: protected routes reject anonymous", () => {
  const protectedGets = ["/api/games", "/api/players", "/api/jobs/whatever"];
  for (const path of protectedGets) {
    test(`GET ${path} anonymous -> 401`, async ({}, info) => {
      const { ctx } = await freshApi(OCTET, info.workerIndex);
      const res = await ctx.get(path);
      expect(res.status()).toBe(401);
      await ctx.dispose();
    });
  }

  test("POST /api/uploads anonymous -> 401", async ({}, info) => {
    const { ctx } = await freshApi(OCTET, info.workerIndex);
    const res = await ctx.post("/api/uploads", {
      multipart: { image: { name: "x.png", mimeType: "image/png", buffer: Buffer.from([1, 2, 3]) } },
    });
    expect(res.status()).toBe(401);
    await ctx.dispose();
  });

  test("POST /api/games anonymous -> 401", async ({}, info) => {
    const { ctx } = await freshApi(OCTET, info.workerIndex);
    const res = await ctx.post("/api/games", { data: { source: "manual" } });
    expect(res.status()).toBe(401);
    await ctx.dispose();
  });
});

test.describe("authorization: cross-user isolation", () => {
  test("user B cannot read, save, delete, or export user A's game", async ({}, info) => {
    const a = await freshApi(OCTET, info.workerIndex);
    const b = await freshApi(OCTET, info.workerIndex);
    await a.api.registerUnique("ownerA");
    await b.api.registerUnique("ownerB");

    const gid = await a.api.createManual();
    await a.ctx.patch(`/api/games/${gid}`, {
      data: { header: { white: "Secret", black: "Game", result: "*" }, moves: [{ ply: 1, san: "e4" }] },
    });

    // All of B's attempts must look like the game simply doesn't exist (404, not 403).
    expect((await b.ctx.get(`/api/games/${gid}`)).status()).toBe(404);
    expect((await b.ctx.patch(`/api/games/${gid}`, { data: { header: { white: "x", black: "y", result: "*" }, moves: [] } })).status()).toBe(404);
    expect((await b.ctx.get(`/api/games/${gid}/pgn`)).status()).toBe(404);
    expect((await b.ctx.delete(`/api/games/${gid}`)).status()).toBe(404);

    // And A's game is untouched.
    expect((await a.ctx.get(`/api/games/${gid}`)).status()).toBe(200);
    await a.ctx.dispose();
    await b.ctx.dispose();
  });

  test("library lists only the caller's own saved games", async ({}, info) => {
    const a = await freshApi(OCTET, info.workerIndex);
    const b = await freshApi(OCTET, info.workerIndex);
    await a.api.registerUnique("libA");
    await b.api.registerUnique("libB");

    const marker = `Iso${Date.now()}`;
    const gid = await a.api.createManual();
    await a.ctx.patch(`/api/games/${gid}`, {
      data: { header: { white: marker, black: "Z", result: "*" }, moves: [] },
    });

    const aList = await (await a.ctx.get(`/api/games?q=${marker}`)).json();
    expect(aList.total).toBe(1);
    const bList = await (await b.ctx.get(`/api/games?q=${marker}`)).json();
    expect(bList.total, "user B must not see user A's game").toBe(0);
    await a.ctx.dispose();
    await b.ctx.dispose();
  });
});
