import { describe, expect, it, vi } from "vitest";
import { addImage, removeImageAt, submitImages } from "./multi-image";

function file(name: string): File {
  return new File(["x"], name, { type: "image/jpeg" });
}

describe("addImage", () => {
  it("appends to the end without mutating the input", () => {
    const a = file("a.jpg");
    const b = file("b.jpg");
    const start = [a];
    const next = addImage(start, b);
    expect(next).toEqual([a, b]);
    expect(start).toEqual([a]); // unchanged
  });
});

describe("removeImageAt", () => {
  it("drops the file at the given index, keeping the rest in order", () => {
    const a = file("a.jpg");
    const b = file("b.jpg");
    const c = file("c.jpg");
    expect(removeImageAt([a, b, c], 1)).toEqual([a, c]);
  });

  it("is a no-op for an out-of-range index", () => {
    const a = file("a.jpg");
    expect(removeImageAt([a], 5)).toEqual([a]);
  });
});

describe("submitImages", () => {
  it("combine mode sends every picture in one request → one job", async () => {
    const submitOne = vi
      .fn<(images: File[]) => Promise<{ jobId: string }>>()
      .mockResolvedValue({ jobId: "j1" });
    const files = [file("a.jpg"), file("b.jpg")];

    const res = await submitImages(files, "combine", submitOne);

    expect(res).toEqual({ jobIds: ["j1"], rejected: 0 });
    expect(submitOne).toHaveBeenCalledTimes(1);
    expect(submitOne).toHaveBeenCalledWith(files);
  });

  it("separate mode sends one request per picture → one job each", async () => {
    let n = 0;
    const submitOne = vi.fn<(images: File[]) => Promise<{ jobId: string }>>(
      async () => ({ jobId: `j${++n}` }),
    );
    const files = [file("a.jpg"), file("b.jpg"), file("c.jpg")];

    const res = await submitImages(files, "separate", submitOne);

    expect(res.jobIds).toEqual(["j1", "j2", "j3"]);
    expect(res.rejected).toBe(0);
    expect(submitOne).toHaveBeenCalledTimes(3);
    // each call carries exactly one image
    for (const call of submitOne.mock.calls) {
      expect(call[0].length).toBe(1);
    }
  });

  it("a single picture always goes as one request, even in separate mode", async () => {
    const submitOne = vi.fn<(images: File[]) => Promise<{ jobId: string }>>(
      async () => ({ jobId: "j1" }),
    );
    const res = await submitImages([file("a.jpg")], "separate", submitOne);
    expect(res.jobIds).toEqual(["j1"]);
    expect(submitOne).toHaveBeenCalledTimes(1);
  });

  it("separate mode tolerates partial failure, reporting the rejected count", async () => {
    const submitOne = vi
      .fn<(images: File[]) => Promise<{ jobId: string }>>()
      .mockResolvedValueOnce({ jobId: "j1" })
      .mockRejectedValueOnce(new Error("429"))
      .mockResolvedValueOnce({ jobId: "j3" });

    const res = await submitImages(
      [file("a.jpg"), file("b.jpg"), file("c.jpg")],
      "separate",
      submitOne,
    );

    expect(res.jobIds).toEqual(["j1", "j3"]);
    expect(res.rejected).toBe(1);
  });

  it("throws when every picture fails so the caller can surface the error", async () => {
    const boom = new Error("rate limit");
    const submitOne = vi.fn<(images: File[]) => Promise<{ jobId: string }>>(
      async () => {
        throw boom;
      },
    );
    await expect(
      submitImages([file("a.jpg"), file("b.jpg")], "separate", submitOne),
    ).rejects.toBe(boom);
  });
});
