import { describe, it, expect } from "vitest";
import { buildPgn } from "./pgn-export";
import { STARTING_FEN } from "./chess";
import type { Header } from "./api-client";

const header = (over: Partial<Header> = {}): Header => ({
  white: "Doe, John",
  black: "Roe, Jane",
  event: "Club",
  site: "",
  date: "2026.06.22",
  round: "",
  board: "",
  result: "*",
  ...over,
});

describe("buildPgn", () => {
  it("writes tags and numbered movetext from the start position", () => {
    const pgn = buildPgn(header(), STARTING_FEN, ["e4", "e5", "Nf3"]);
    expect(pgn).toContain('[White "Doe, John"]');
    expect(pgn).toContain('[Black "Roe, Jane"]');
    expect(pgn).toContain("1. e4 e5 2. Nf3");
  });

  it("emits SetUp/FEN tags for a non-standard start position", () => {
    const fen = "4k3/8/8/8/8/8/8/4K2R w K - 0 1";
    const pgn = buildPgn(header({ result: "1-0" }), fen, []);
    expect(pgn).toContain('[SetUp "1"]');
    expect(pgn).toContain(`[FEN "${fen}"]`);
    expect(pgn).toContain("1-0");
  });

  it("stops at the first illegal SAN instead of throwing", () => {
    const pgn = buildPgn(header(), STARTING_FEN, ["e4", "Qh5??garbage"]);
    expect(pgn).toContain("1. e4");
  });
});
