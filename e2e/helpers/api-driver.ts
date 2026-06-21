import { APIRequestContext, expect, request as pwRequest } from "@playwright/test";

// 1x1 PNG; content is irrelevant because the fake recognizer ignores the image.
const PIXEL_PNG = Buffer.from(
  "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+M8AAAMBAQDJ/pLvAAAAAElFTkSuQmCC",
  "base64",
);

export type Draft = {
  id: string;
  status: string;
  source: string;
  header: Record<string, string>;
  startFen: string;
  moves: {
    ply: number;
    side: string;
    san: string;
    isLegal: boolean;
    clockSec: number | null;
    confidence: number;
    corrected: boolean;
    recognizedText: string;
  }[];
  imageUrl: string;
};

// ApiDriver wraps the REST surface. The APIRequestContext carries the session cookie,
// mirroring OpenPairing.org's ApiDriver strategy.
export class ApiDriver {
  constructor(private request: APIRequestContext) {}

  async register(suffix: string) {
    const email = `e2e+${suffix}@example.com`;
    const res = await this.request.post("/api/auth/register", {
      data: { name: `E2E ${suffix}`, email, password: "password1234" },
    });
    expect(res.status(), await res.text()).toBe(201);
    return email;
  }

  // registerUnique registers a brand-new account with a collision-proof email and
  // returns it. Use when a test needs its own account on a shared dev database.
  async registerUnique(prefix = "user") {
    const email = `${prefix}-${rand()}@example.com`;
    const res = await this.request.post("/api/auth/register", {
      data: { name: prefix, email, password: "password1234" },
    });
    expect(res.status(), await res.text()).toBe(201);
    return email;
  }

  async uploadConvert() {
    const res = await this.request.post("/api/convert", {
      multipart: { image: { name: "sheet.png", mimeType: "image/png", buffer: PIXEL_PNG } },
    });
    expect(res.status(), await res.text()).toBe(202);
    return (await res.json()).jobId as string;
  }

  async upload() {
    const res = await this.request.post("/api/uploads", {
      multipart: { image: { name: "sheet.png", mimeType: "image/png", buffer: PIXEL_PNG } },
    });
    expect(res.status(), await res.text()).toBe(202);
    return (await res.json()) as { jobId: string; uploadId: string };
  }

  async pollJob(jobPath: string): Promise<{ status: string; gameId?: string; error?: string }> {
    for (let i = 0; i < 100; i++) {
      const res = await this.request.get(jobPath);
      const body = await res.json();
      if (body.status === "done" || body.status === "failed") return body;
      await new Promise((r) => setTimeout(r, 300));
    }
    throw new Error(`job ${jobPath} never finished`);
  }

  async getGame(id: string): Promise<Draft> {
    const res = await this.request.get(`/api/games/${id}`);
    expect(res.ok(), await res.text()).toBeTruthy();
    return res.json();
  }

  // createManual creates an empty draft owned by the current session and returns its id.
  async createManual(): Promise<string> {
    const res = await this.request.post("/api/games", { data: { source: "manual" } });
    expect(res.status(), await res.text()).toBe(201);
    return (await res.json()).game.id as string;
  }

  // uploadAndRecognize runs the full account upload->recognize flow and returns the
  // recognized draft for the current (authenticated) session.
  async uploadAndRecognize(): Promise<{ gameId: string; draft: Draft }> {
    const { jobId } = await this.upload();
    const job = await this.pollJob(`/api/jobs/${jobId}`);
    expect(job.status).toBe("done");
    expect(job.gameId).toBeTruthy();
    return { gameId: job.gameId!, draft: await this.getGame(job.gameId!) };
  }
}

function rand() {
  return Math.random().toString(36).slice(2, 10) + Date.now().toString(36);
}

const API_BASE = process.env.PGNIZE_API_BASE || "http://localhost:8080";

// freshApi returns a request context bound to a unique client IP, so per-IP rate limits
// never bleed across tests. fileOctet namespaces a spec file; workerIndex + a per-worker
// counter keep the IP unique across parallel Playwright workers.
let ipCounter = 0;
export async function freshApi(fileOctet: number, workerIndex: number) {
  ipCounter += 1;
  const ip = `10.${fileOctet}.${workerIndex % 250}.${(ipCounter % 250) + 1}`;
  const ctx = await pwRequest.newContext({
    baseURL: API_BASE,
    extraHTTPHeaders: { "X-Forwarded-For": ip },
  });
  return { ctx, api: new ApiDriver(ctx), ip };
}
