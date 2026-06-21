import { describe, it, expect } from "vitest";
import { uciToSan, pvToSanLine, scoreToEval, buildCoachMoveRequest } from "./coach";
import { STARTING_FEN, type EditMove } from "./chess";
import type { Score } from "./engine";

const score = (over: Partial<Score> = {}): Score => ({
  cp: 0,
  mate: null,
  depth: 12,
  pv: [],
  bestMove: null,
  ...over,
});

describe("uciToSan", () => {
  it("converts a capture", () => {
    // After 1.e4 d5, white to move: exd5 is a capture.
    const fen = "rnbqkbnr/ppp1pppp/8/3p4/4P3/8/PPPP1PPP/RNBQKBNR w KQkq d6 0 2";
    expect(uciToSan(fen, "e4d5")).toBe("exd5");
  });

  it("converts kingside castling", () => {
    const fen = "rnbqk2r/pppp1ppp/5n2/2b1p3/2B1P3/5N2/PPPP1PPP/RNBQK2R w KQkq - 4 4";
    expect(uciToSan(fen, "e1g1")).toBe("O-O");
  });

  it("converts a promotion, reading the suffix piece (not defaulting to queen)", () => {
    const fen = "8/4P3/8/8/8/8/k7/4K3 w - - 0 1";
    expect(uciToSan(fen, "e7e8q")).toBe("e8=Q");
    expect(uciToSan(fen, "e7e8n")).toBe("e8=N");
  });

  it("returns null for an illegal/garbage move", () => {
    expect(uciToSan(STARTING_FEN, "e2e5")).toBeNull();
    expect(uciToSan(STARTING_FEN, "zz")).toBeNull();
  });
});

describe("pvToSanLine", () => {
  it("replays a UCI principal variation into SAN", () => {
    expect(pvToSanLine(STARTING_FEN, ["e2e4", "e7e5", "g1f3"])).toEqual([
      "e4",
      "e5",
      "Nf3",
    ]);
  });

  it("stops at the first move that does not apply", () => {
    expect(pvToSanLine(STARTING_FEN, ["e2e4", "e2e4"])).toEqual(["e4"]);
  });

  it("honors the limit", () => {
    expect(pvToSanLine(STARTING_FEN, ["e2e4", "e7e5", "g1f3"], 2)).toEqual([
      "e4",
      "e5",
    ]);
  });
});

describe("scoreToEval", () => {
  it("maps a White-POV score to the wire shape", () => {
    expect(scoreToEval(score({ cp: 134 }))).toEqual({ cp: 134, mate: null });
    expect(scoreToEval(score({ cp: null, mate: -2 }))).toEqual({
      cp: null,
      mate: -2,
    });
  });

  it("is null/null for a missing score", () => {
    expect(scoreToEval(undefined)).toEqual({ cp: null, mate: null });
  });
});

describe("buildCoachMoveRequest", () => {
  const move: EditMove = {
    san: "Nf5",
    clockSec: null,
    recognizedText: "",
    confidence: 1,
    side: "white",
    fenBefore: STARTING_FEN,
    fenAfter: STARTING_FEN,
    legality: "legal",
    corrected: false,
    ambiguousOptions: [],
  };

  it("assembles the wire request, converting the engine PV to SAN", () => {
    const best = score({ cp: 30, pv: ["e2e4", "e7e5"], bestMove: "e2e4" });
    const after = score({ cp: -260 });
    const req = buildCoachMoveRequest({
      move,
      bestScore: best,
      afterScore: after,
      quality: "mistake",
      gameId: "g1",
      ply: 0,
      lang: "de",
    });
    expect(req).toEqual({
      gameId: "g1",
      ply: 0,
      fen: STARTING_FEN,
      side: "white",
      playedSan: "Nf5",
      bestSan: "e4",
      bestLine: ["e4", "e5"],
      evalBefore: { cp: 30, mate: null },
      evalAfter: { cp: -260, mate: null },
      quality: "mistake",
      lang: "de",
    });
  });

  it("tolerates a missing best move / after score", () => {
    const req = buildCoachMoveRequest({
      move,
      bestScore: score(),
      afterScore: undefined,
      quality: null,
    });
    expect(req.bestSan).toBe("");
    expect(req.bestLine).toEqual([]);
    expect(req.quality).toBe("");
    expect(req.evalAfter).toEqual({ cp: null, mate: null });
  });
});
