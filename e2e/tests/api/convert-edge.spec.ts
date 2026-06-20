import { test, expect } from "@playwright/test";
import { freshApi } from "../../helpers/api-driver";

const OCTET = 23;
const PIXEL_PNG = Buffer.from(
  "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+M8AAAMBAQDJ/pLvAAAAAElFTkSuQmCC",
  "base64",
);

test.describe("convert endpoint edge cases", () => {
  test("status of an unknown job -> 404", async ({}, info) => {
    const { ctx } = await freshApi(OCTET, info.workerIndex);
    const res = await ctx.get("/api/convert/00000000-0000-0000-0000-000000000000");
    expect(res.status()).toBe(404);
    await ctx.dispose();
  });

  test("game before the job is done -> 404 not ready", async ({}, info) => {
    const { ctx, api } = await freshApi(OCTET, info.workerIndex);
    const jobId = await api.uploadConvert();
    // Immediately (the worker may not have finished): either 404 (not ready) or 200 (done).
    const res = await ctx.get(`/api/convert/${jobId}/game`);
    expect([200, 404]).toContain(res.status());
    await ctx.dispose();
  });

  test("export replays moves and streams a PGN attachment", async ({}, info) => {
    const { ctx, api } = await freshApi(OCTET, info.workerIndex);
    const jobId = await api.uploadConvert();
    const res = await ctx.post(`/api/convert/${jobId}/export`, {
      data: { header: { white: "Anon", black: "Mouse", result: "*" }, moves: [{ ply: 1, san: "e4" }, { ply: 2, san: "e5" }] },
    });
    expect(res.status()).toBe(200);
    expect(res.headers()["content-disposition"]).toContain("attachment");
    const pgn = await res.text();
    expect(pgn).toContain("Anon");
    expect(pgn).toContain("1. e4 e5");
    await ctx.dispose();
  });

  test("export with an illegal move -> 422 failedAt", async ({}, info) => {
    const { ctx, api } = await freshApi(OCTET, info.workerIndex);
    const jobId = await api.uploadConvert();
    const res = await ctx.post(`/api/convert/${jobId}/export`, {
      data: { header: { white: "A", black: "B", result: "*" }, moves: [{ ply: 1, san: "Ke4" }] },
    });
    expect(res.status()).toBe(422);
    const body = await res.json();
    expect(body.error).toBe("illegal_move");
    expect(body.failedAt).toBe(0);
    await ctx.dispose();
  });

  test("export with empty moves yields a valid result-only PGN", async ({}, info) => {
    const { ctx, api } = await freshApi(OCTET, info.workerIndex);
    const jobId = await api.uploadConvert();
    const res = await ctx.post(`/api/convert/${jobId}/export`, {
      data: { header: { white: "A", black: "B", result: "1-0" }, moves: [] },
    });
    expect(res.status()).toBe(200);
    expect(await res.text()).toContain("1-0");
    await ctx.dispose();
  });

  test("export with invalid JSON -> 400", async ({}, info) => {
    const { ctx, api } = await freshApi(OCTET, info.workerIndex);
    const jobId = await api.uploadConvert();
    const res = await ctx.post(`/api/convert/${jobId}/export`, {
      headers: { "Content-Type": "application/json" },
      data: "{bad",
    });
    expect(res.status()).toBe(400);
    await ctx.dispose();
  });
});

test.describe("upload endpoint edge cases", () => {
  test("missing image field -> 400 missing_image", async ({}, info) => {
    const { ctx, api } = await freshApi(OCTET, info.workerIndex);
    await api.registerUnique("noimg");
    const res = await ctx.post("/api/uploads", { multipart: { notimage: "x" } });
    expect(res.status()).toBe(400);
    expect((await res.json()).error).toBe("missing_image");
    await ctx.dispose();
  });

  test("oversized image -> 413 too_large", async ({}, info) => {
    const { ctx, api } = await freshApi(OCTET, info.workerIndex);
    await api.registerUnique("big");
    const big = Buffer.alloc(16 * 1024 * 1024, 7); // 16 MB > the 15 MB default limit
    const res = await ctx.post("/api/uploads", {
      multipart: { image: { name: "big.png", mimeType: "image/png", buffer: big } },
    });
    expect(res.status()).toBe(413);
    await ctx.dispose();
  });

  test("unknown recognition backend -> 400 unknown_backend", async ({}, info) => {
    const { ctx, api } = await freshApi(OCTET, info.workerIndex);
    await api.registerUnique("backend");
    const res = await ctx.post("/api/uploads", {
      multipart: {
        image: { name: "x.png", mimeType: "image/png", buffer: PIXEL_PNG },
        backend: "does-not-exist",
      },
    });
    expect(res.status()).toBe(400);
    expect((await res.json()).error).toBe("unknown_backend");
    await ctx.dispose();
  });
});

test.describe("job ownership isolation", () => {
  test("an account job is invisible on the anonymous convert endpoint", async ({}, info) => {
    const { ctx, api } = await freshApi(OCTET, info.workerIndex);
    await api.registerUnique("jobiso");
    const { jobId } = await api.upload();
    const res = await ctx.get(`/api/convert/${jobId}`);
    expect(res.status(), "account job must 404 on the anon endpoint").toBe(404);
    await ctx.dispose();
  });

  test("an anonymous job is invisible on the account jobs endpoint", async ({}, info) => {
    const anon = await freshApi(OCTET, info.workerIndex);
    const acct = await freshApi(OCTET, info.workerIndex);
    const jobId = await anon.api.uploadConvert();
    await acct.api.registerUnique("acctview");
    const res = await acct.ctx.get(`/api/jobs/${jobId}`);
    expect(res.status(), "anon job must 404 on the account endpoint").toBe(404);
    await anon.ctx.dispose();
    await acct.ctx.dispose();
  });

  test("user B cannot poll user A's job", async ({}, info) => {
    const a = await freshApi(OCTET, info.workerIndex);
    const b = await freshApi(OCTET, info.workerIndex);
    await a.api.registerUnique("jaA");
    await b.api.registerUnique("jaB");
    const { jobId } = await a.api.upload();
    const res = await b.ctx.get(`/api/jobs/${jobId}`);
    expect(res.status()).toBe(404);
    await a.ctx.dispose();
    await b.ctx.dispose();
  });
});

test.describe("image streaming authorization", () => {
  test("account image: owner 200, other user 404, anonymous 404", async ({}, info) => {
    const a = await freshApi(OCTET, info.workerIndex);
    const b = await freshApi(OCTET, info.workerIndex);
    const anon = await freshApi(OCTET, info.workerIndex);
    await a.api.registerUnique("imgA");
    await b.api.registerUnique("imgB");

    const { gameId, draft } = await a.api.uploadAndRecognize();
    expect(draft.imageUrl).not.toBe("");
    const path = draft.imageUrl;

    expect((await a.ctx.get(path)).status(), "owner can view").toBe(200);
    expect((await b.ctx.get(path)).status(), "other user cannot view").toBe(404);
    expect((await anon.ctx.get(path)).status(), "anonymous cannot view account image").toBe(404);
    expect(gameId).toBeTruthy();
    await a.ctx.dispose();
    await b.ctx.dispose();
    await anon.ctx.dispose();
  });

  test("anonymous convert image is served to any caller holding the id", async ({}, info) => {
    const anon = await freshApi(OCTET, info.workerIndex);
    const other = await freshApi(OCTET, info.workerIndex);
    const jobId = await anon.api.uploadConvert();
    const job = await anon.api.pollJob(`/api/convert/${jobId}`);
    expect(job.status).toBe("done");
    const draft = await (await anon.ctx.get(`/api/convert/${jobId}/game`)).json();
    expect(draft.imageUrl).not.toBe("");
    // Anonymous uploads are bearer-by-id: anyone with the URL may fetch.
    expect((await other.ctx.get(draft.imageUrl)).status()).toBe(200);
    await anon.ctx.dispose();
    await other.ctx.dispose();
  });
});
