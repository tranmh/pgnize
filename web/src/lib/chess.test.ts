import { describe, it, expect } from "vitest";
import { rankBySimilarity, uciToSquares } from "./chess";

describe("rankBySimilarity", () => {
  it("ranks the move closest to the recognized text first", () => {
    // "Mf3" is a likely misread of "Nf3".
    expect(rankBySimilarity(["e4", "Nf3", "Qh5", "O-O"], "Mf3")[0]).toBe("Nf3");
  });

  it("prefers the exact disambiguated match", () => {
    expect(rankBySimilarity(["Nbd2", "Nfd2", "Ne2"], "Nfd2")[0]).toBe("Nfd2");
  });

  it("preserves the original order when there is nothing to match", () => {
    expect(rankBySimilarity(["a3", "b3"], "")).toEqual(["a3", "b3"]);
  });

  it("ignores check/capture decoration when comparing", () => {
    expect(rankBySimilarity(["Qxh7", "Qh5", "a4"], "Qh7")[0]).toBe("Qxh7");
  });
});

describe("uciToSquares", () => {
  it("splits a basic move", () => {
    expect(uciToSquares("e2e4")).toEqual({ from: "e2", to: "e4" });
  });

  it("drops the promotion suffix", () => {
    expect(uciToSquares("e7e8q")).toEqual({ from: "e7", to: "e8" });
  });

  it("returns null for malformed input", () => {
    expect(uciToSquares("x")).toBeNull();
  });
});
