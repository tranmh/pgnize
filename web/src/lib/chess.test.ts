import { describe, it, expect } from "vitest";
import {
  rankBySimilarity,
  rebuild,
  reviewState,
  siblingDisambiguations,
  STARTING_FEN,
  uciToSquares,
  type EditablePly,
} from "./chess";

function ply(p: Partial<EditablePly>): EditablePly {
  return { san: "", clockSec: null, recognizedText: "", confidence: 1, ...p };
}

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

describe("reviewState", () => {
  it("flags a legal low-confidence move as verify until confirmed", () => {
    const moves = rebuild(STARTING_FEN, [
      ply({ san: "e4", recognizedText: "e4", confidence: 0.3 }),
    ]);
    expect(reviewState(moves[0], false)).toBe("verify");
    expect(reviewState(moves[0], true)).toBe("ok");
  });

  it("keeps a confident legal move as ok", () => {
    const moves = rebuild(STARTING_FEN, [
      ply({ san: "e4", recognizedText: "e4", confidence: 0.9 }),
    ]);
    expect(reviewState(moves[0], false)).toBe("ok");
  });

  it("reports illegal and unread states", () => {
    const moves = rebuild(STARTING_FEN, [
      ply({ san: "e4", recognizedText: "e4", confidence: 0.9 }),
      ply({ san: "Qh6", recognizedText: "Qh6", confidence: 0.9 }), // illegal first move for black
      ply({ san: "?", recognizedText: "?", confidence: 0 }),
    ]);
    expect(reviewState(moves[1], false)).toBe("illegal");
    expect(reviewState(moves[2], false)).toBe("unread");
  });

  it("treats a reviewer-edited (diverged) move as confident", () => {
    // recognizedText differs from san => corrected => full confidence regardless of input.
    const moves = rebuild(STARTING_FEN, [
      ply({ san: "e4", recognizedText: "e5", confidence: 0.2 }),
    ]);
    expect(reviewState(moves[0], false)).toBe("ok");
  });
});

describe("siblingDisambiguations", () => {
  it("offers the other knight to the same square", () => {
    // Knights on b1 and f3 both reach d2 (after 1.Nf3 d5 2.g3 e6 3.Bg2 Nf6 4.d3 Be7).
    const fen = "rnbqk2r/ppp1bppp/4pn2/3p4/8/3P1NP1/PPP1PPBP/RNBQK2R w KQkq - 1 5";
    const sibs = siblingDisambiguations(fen, "Nbd2");
    expect(sibs.sort()).toEqual(["Nbd2", "Nfd2"]);
  });

  it("returns nothing for an unambiguous move", () => {
    expect(siblingDisambiguations(STARTING_FEN, "e4")).toEqual([]);
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
