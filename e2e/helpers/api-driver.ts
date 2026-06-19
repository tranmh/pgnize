import { APIRequestContext, expect } from "@playwright/test";

// 1x1 PNG; content is irrelevant because the fake recognizer ignores the image.
const PIXEL_PNG = Buffer.from(
  "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+M8AAAMBAQDJ/pLvAAAAAElFTkSuQmCC",
  "base64",
);

export type Draft = {
  id: string;
  status: string;
  header: Record<string, string>;
  startFen: string;
  moves: { ply: number; san: string; isLegal: boolean }[];
  imageUrl: string;
};

// ApiDriver wraps the REST surface. The APIRequestContext carries the session cookie,
// mirroring swiss-manager's ApiDriver strategy.
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

  async pollJob(jobPath: string): Promise<{ status: string; gameId?: string }> {
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
}
