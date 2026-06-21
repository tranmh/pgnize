import { describe, it, expect } from "vitest";
import {
  boardToField,
  buildFen,
  indexToSquare,
  parseBoardField,
  squareToIndex,
  type FenPiece,
} from "./fen";

const START = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1";
const EMPTY = "8/8/8/8/8/8/8/8 w - - 0 1";
const SPARSE = "4k3/8/8/8/8/8/8/4K3 w - - 0 1";

describe("parseBoardField / boardToField round-trip", () => {
  it("round-trips the starting position", () => {
    const field = START.split(" ")[0];
    expect(boardToField(parseBoardField(START))).toBe(field);
  });

  it("round-trips the empty board", () => {
    expect(boardToField(parseBoardField(EMPTY))).toBe("8/8/8/8/8/8/8/8");
  });

  it("round-trips a sparse position", () => {
    expect(boardToField(parseBoardField(SPARSE))).toBe("4k3/8/8/8/8/8/8/4K3");
  });

  it("expands digits to nulls (64 cells, mostly empty)", () => {
    const board = parseBoardField(EMPTY);
    expect(board).toHaveLength(64);
    expect(board.every((c) => c === null)).toBe(true);
  });

  it("falls back to an empty board for a malformed field, never throwing", () => {
    expect(boardToField(parseBoardField("not-a-fen"))).toBe(
      "8/8/8/8/8/8/8/8",
    );
    // Too many ranks.
    expect(boardToField(parseBoardField("8/8/8 w - - 0 1"))).toBe(
      "8/8/8/8/8/8/8/8",
    );
    // A rank that doesn't sum to 8.
    expect(boardToField(parseBoardField("9/8/8/8/8/8/8/8 w - - 0 1"))).toBe(
      "8/8/8/8/8/8/8/8",
    );
  });
});

describe("boardToField digit collapsing", () => {
  it("collapses adjacent empty squares into a single digit", () => {
    const board = new Array<FenPiece>(64).fill(null);
    board[0] = "r"; // a8
    board[7] = "r"; // h8
    // Rank 8 should be "r6r"; all other ranks "8".
    expect(boardToField(board)).toBe("r6r/8/8/8/8/8/8/8");
  });
});

describe("buildFen", () => {
  const board = parseBoardField(START);

  it("renders the full starting FEN", () => {
    expect(
      buildFen({
        board,
        sideToMove: "w",
        castling: { K: true, Q: true, k: true, q: true },
        enPassant: "",
      }),
    ).toBe(START);
  });

  it("renders castling as a canonical KQkq subset", () => {
    expect(
      buildFen({
        board,
        sideToMove: "w",
        castling: { K: true, Q: false, k: false, q: true },
        enPassant: "",
      }).split(" ")[2],
    ).toBe("Kq");
  });

  it("renders no castling rights as '-'", () => {
    expect(
      buildFen({
        board,
        sideToMove: "w",
        castling: { K: false, Q: false, k: false, q: false },
        enPassant: "",
      }).split(" ")[2],
    ).toBe("-");
  });

  it("renders an empty en-passant as '-' and a set one verbatim", () => {
    const cast = { K: true, Q: true, k: true, q: true };
    expect(
      buildFen({ board, sideToMove: "w", castling: cast, enPassant: "" }).split(
        " ",
      )[3],
    ).toBe("-");
    expect(
      buildFen({
        board,
        sideToMove: "w",
        castling: cast,
        enPassant: "e3",
      }).split(" ")[3],
    ).toBe("e3");
  });

  it("renders the side to move", () => {
    expect(
      buildFen({
        board,
        sideToMove: "b",
        castling: { K: true, Q: true, k: true, q: true },
        enPassant: "",
      }).split(" ")[1],
    ).toBe("b");
  });

  it("defaults halfmove to 0 and fullmove to 1", () => {
    const parts = buildFen({
      board,
      sideToMove: "w",
      castling: { K: true, Q: true, k: true, q: true },
      enPassant: "",
    }).split(" ");
    expect(parts[4]).toBe("0");
    expect(parts[5]).toBe("1");
  });

  it("builds illegal-but-syntactically-valid positions without throwing", () => {
    // Two white kings.
    const twoKings = new Array<FenPiece>(64).fill(null);
    twoKings[squareToIndex("e1")] = "K";
    twoKings[squareToIndex("e2")] = "K";
    expect(() =>
      buildFen({
        board: twoKings,
        sideToMove: "w",
        castling: { K: false, Q: false, k: false, q: false },
        enPassant: "",
      }),
    ).not.toThrow();

    // A pawn on rank 1 and no kings at all.
    const pawnOnRank1 = new Array<FenPiece>(64).fill(null);
    pawnOnRank1[squareToIndex("a1")] = "P";
    expect(() =>
      buildFen({
        board: pawnOnRank1,
        sideToMove: "w",
        castling: { K: false, Q: false, k: false, q: false },
        enPassant: "",
      }),
    ).not.toThrow();
  });
});

describe("squareToIndex / indexToSquare", () => {
  it("maps a8 to 0 and h1 to 63", () => {
    expect(squareToIndex("a8")).toBe(0);
    expect(squareToIndex("h1")).toBe(63);
  });

  it("is its own inverse across every square", () => {
    for (let i = 0; i < 64; i++) {
      expect(squareToIndex(indexToSquare(i))).toBe(i);
    }
  });

  it("returns -1 / '' for malformed input", () => {
    expect(squareToIndex("z9")).toBe(-1);
    expect(squareToIndex("a")).toBe(-1);
    expect(indexToSquare(-1)).toBe("");
    expect(indexToSquare(64)).toBe("");
  });

  it("locates the kings in the starting position by square", () => {
    const board = parseBoardField(START);
    expect(board[squareToIndex("e1")]).toBe("K");
    expect(board[squareToIndex("e8")]).toBe("k");
  });
});
