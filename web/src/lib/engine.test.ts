import { describe, it, expect } from "vitest";
import {
  parseInfo,
  parseBestMove,
  toScore,
  scoreToCp,
  type ParsedInfo,
  type Score,
} from "./engine";

const score = (over: Partial<Score> = {}): Score => ({
  cp: 0,
  mate: null,
  depth: 1,
  pv: [],
  bestMove: null,
  ...over,
});

describe("parseInfo", () => {
  it("parses a centipawn line with depth, multipv and pv", () => {
    const info = parseInfo(
      "info depth 12 seldepth 18 multipv 1 score cp 34 nodes 1000 pv e2e4 e7e5 g1f3",
    );
    expect(info).toEqual<ParsedInfo>({
      depth: 12,
      multipv: 1,
      cp: 34,
      mate: null,
      pv: ["e2e4", "e7e5", "g1f3"],
    });
  });

  it("parses a mate score and a non-default multipv", () => {
    const info = parseInfo("info depth 9 multipv 2 score mate -3 pv d1h5 g6h5");
    expect(info?.multipv).toBe(2);
    expect(info?.cp).toBeNull();
    expect(info?.mate).toBe(-3);
    expect(info?.pv[0]).toBe("d1h5");
  });

  it("returns null for info lines without a score+pv", () => {
    expect(parseInfo("info depth 1 currmove e2e4 currmovenumber 1")).toBeNull();
  });

  it("returns null for non-info lines", () => {
    expect(parseInfo("bestmove e2e4")).toBeNull();
  });
});

describe("parseBestMove", () => {
  it("extracts the move", () => {
    expect(parseBestMove("bestmove e2e4 ponder e7e5")).toBe("e2e4");
  });

  it("returns null for a terminal position", () => {
    expect(parseBestMove("bestmove (none)")).toBeNull();
  });

  it("returns undefined for non-bestmove lines", () => {
    expect(parseBestMove("info depth 1")).toBeUndefined();
  });
});

describe("toScore (normalize to White's perspective)", () => {
  const cpInfo: ParsedInfo = { depth: 12, multipv: 1, cp: 34, mate: null, pv: ["e2e4"] };
  const mateInfo: ParsedInfo = { depth: 9, multipv: 1, cp: null, mate: -3, pv: ["d1h5"] };

  it("keeps the sign when White is to move", () => {
    expect(toScore(cpInfo, false).cp).toBe(34);
  });

  it("negates when Black is to move", () => {
    expect(toScore(cpInfo, true).cp).toBe(-34);
  });

  it("flips a mate score to White's perspective", () => {
    // Black to move, mate -3 (bad for Black) => White mates in 3.
    expect(toScore(mateInfo, true).mate).toBe(3);
  });

  it("exposes the first pv move as bestMove", () => {
    expect(toScore(cpInfo, false).bestMove).toBe("e2e4");
  });
});

describe("scoreToCp", () => {
  it("passes centipawns through", () => {
    expect(scoreToCp(score({ cp: 50 }))).toBe(50);
  });

  it("maps mates to magnitudes that dominate any cp value", () => {
    expect(scoreToCp(score({ cp: null, mate: 2 }))).toBeGreaterThan(50000);
    expect(scoreToCp(score({ cp: null, mate: -2 }))).toBeLessThan(-50000);
  });

  it("ranks a faster mate above a slower one", () => {
    expect(scoreToCp(score({ cp: null, mate: 1 }))).toBeGreaterThan(
      scoreToCp(score({ cp: null, mate: 5 })),
    );
  });
});
