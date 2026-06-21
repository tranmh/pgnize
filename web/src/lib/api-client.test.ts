import { afterEach, describe, expect, it, vi } from "vitest";
import { convert, scan } from "./api-client";

// Capture the FormData passed to fetch so we can assert how files are encoded.
function stubFetch(): () => FormData {
  const captured: { body?: FormData } = {};
  vi.stubGlobal(
    "fetch",
    vi.fn(async (_url: string, init?: RequestInit) => {
      captured.body = init?.body as FormData;
      return new Response(JSON.stringify({ jobId: "job-1" }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      });
    }),
  );
  return () => captured.body as FormData;
}

afterEach(() => {
  vi.unstubAllGlobals();
});

function file(name: string): File {
  return new File(["x"], name, { type: "image/jpeg" });
}

describe("convert", () => {
  it("appends one 'image' entry per file", async () => {
    const getBody = stubFetch();
    await convert([file("a.jpg"), file("b.jpg")]);
    const images = getBody().getAll("image");
    expect(images).toHaveLength(2);
    expect((images[0] as File).name).toBe("a.jpg");
    expect((images[1] as File).name).toBe("b.jpg");
    expect(getBody().get("backend")).toBeNull();
  });

  it("appends an optional backend field", async () => {
    const getBody = stubFetch();
    await convert([file("a.jpg")], "ollama");
    expect(getBody().getAll("image")).toHaveLength(1);
    expect(getBody().get("backend")).toBe("ollama");
  });
});

describe("scan", () => {
  it("appends one 'image' entry per file", async () => {
    const getBody = stubFetch();
    await scan([file("a.jpg"), file("b.jpg"), file("c.jpg")]);
    expect(getBody().getAll("image")).toHaveLength(3);
  });
});
