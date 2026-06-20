import { describe, it, expect } from "vitest";
import { classify, annotate, formatScore } from "./analysis";
import type { Score } from "./engine";

const score = (over: Partial<Score> = {}): Score => ({
  cp: 0,
  mate: null,
  depth: 1,
  pv: [],
  bestMove: null,
  ...over,
});

describe("classify (White-POV cp before/after)", () => {
  it("flags a White blunder when White's eval craters", () => {
    expect(classify(20, -320, "white")).toBe("blunder");
  });

  it("flags a Black blunder when White's eval jumps after Black's move", () => {
    expect(classify(-20, 320, "black")).toBe("blunder");
  });

  it("returns null for a near-best move", () => {
    expect(classify(20, 10, "white")).toBeNull();
  });

  it("grades inaccuracies and mistakes by loss size", () => {
    expect(classify(0, -90, "white")).toBe("inaccuracy");
    expect(classify(0, -200, "white")).toBe("mistake");
  });

  it("does not nag when the side was already hopelessly lost", () => {
    expect(classify(-1500, -1700, "white")).toBeNull();
  });
});

describe("annotate", () => {
  it("classifies each ply and stops at the first missing eval", () => {
    const ann = annotate(
      0,
      [score({ cp: -90 }), undefined],
      ["white", "black"],
    );
    expect(ann[0].quality).toBe("inaccuracy");
    expect(ann[1]).toBeUndefined();
  });
});

describe("formatScore", () => {
  it("formats centipawns with a sign", () => {
    expect(formatScore(score({ cp: 134 }))).toBe("+1.3");
    expect(formatScore(score({ cp: -50 }))).toBe("-0.5");
  });

  it("formats mates", () => {
    expect(formatScore(score({ cp: null, mate: 3 }))).toBe("M3");
    expect(formatScore(score({ cp: null, mate: -2 }))).toBe("-M2");
  });
});
