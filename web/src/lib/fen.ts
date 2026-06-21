// Pure FEN board-field manipulation for the position editor.
//
// chess.js THROWS on the illegal intermediate positions (two kings, a pawn on
// rank 1, no king at all) that occur constantly while editing a position by
// hand. So the editor never feeds an intermediate position to chess.js: it
// manipulates the board field of the FEN string directly through this model and
// uses chess.js only for an advisory (try/catch) legality note at the very end.
//
// The board array is 64 cells, rank-major, with index 0 = a8 and index 63 = h1
// (the same order a FEN board field is written in). Each cell is a single piece
// character (white pieces uppercase "KQRBNP", black lowercase "kqrbnp") or null
// for an empty square.

// A single-character chess piece in FEN notation, or null for an empty square.
export type FenPiece =
  | "K" | "Q" | "R" | "B" | "N" | "P"
  | "k" | "q" | "r" | "b" | "n" | "p"
  | null;

export interface CastlingRights {
  K: boolean;
  Q: boolean;
  k: boolean;
  q: boolean;
}

export interface BuildFenInput {
  board: FenPiece[];
  sideToMove: "w" | "b";
  castling: CastlingRights;
  // Empty string renders as "-".
  enPassant: string;
  halfmove?: number;
  fullmove?: number;
}

const PIECE_CHARS = new Set([
  "K", "Q", "R", "B", "N", "P",
  "k", "q", "r", "b", "n", "p",
]);

const FILES = "abcdefgh";

// An empty 64-cell board. Returned as a fresh array each call so callers never
// share mutable state.
function emptyBoard(): FenPiece[] {
  return new Array<FenPiece>(64).fill(null);
}

// parseBoardField expands the board field of a FEN (the part before the first
// space) into a 64-length, rank-major array (index 0 = a8 … 63 = h1). Digits
// expand to that many nulls. A malformed field falls back to an empty board;
// this never throws.
export function parseBoardField(fen: string): FenPiece[] {
  const board = emptyBoard();
  if (!fen) return board;

  const field = fen.split(" ")[0] ?? "";
  const ranks = field.split("/");
  if (ranks.length !== 8) return emptyBoard();

  for (let r = 0; r < 8; r++) {
    let file = 0;
    for (const ch of ranks[r]) {
      if (file > 8) return emptyBoard();
      if (ch >= "1" && ch <= "8") {
        file += Number(ch);
      } else if (PIECE_CHARS.has(ch)) {
        if (file >= 8) return emptyBoard();
        board[r * 8 + file] = ch as FenPiece;
        file += 1;
      } else {
        return emptyBoard();
      }
    }
    if (file !== 8) return emptyBoard();
  }

  return board;
}

// boardToField is the inverse of parseBoardField: it collapses runs of empty
// squares into digits and joins the eight ranks with "/".
export function boardToField(board: FenPiece[]): string {
  const ranks: string[] = [];
  for (let r = 0; r < 8; r++) {
    let rank = "";
    let empty = 0;
    for (let f = 0; f < 8; f++) {
      const piece = board[r * 8 + f] ?? null;
      if (piece === null) {
        empty += 1;
      } else {
        if (empty > 0) {
          rank += String(empty);
          empty = 0;
        }
        rank += piece;
      }
    }
    if (empty > 0) rank += String(empty);
    ranks.push(rank);
  }
  return ranks.join("/");
}

// buildFen assembles a full six-field FEN from the editable model. It performs
// no legality checks — the position may be syntactically valid yet illegal (two
// white kings, a pawn on rank 1); it still builds without throwing.
export function buildFen({
  board,
  sideToMove,
  castling,
  enPassant,
  halfmove = 0,
  fullmove = 1,
}: BuildFenInput): string {
  const field = boardToField(board);
  let rights = "";
  if (castling.K) rights += "K";
  if (castling.Q) rights += "Q";
  if (castling.k) rights += "k";
  if (castling.q) rights += "q";
  const castlingField = rights || "-";
  const ep = enPassant.trim() || "-";
  return `${field} ${sideToMove} ${castlingField} ${ep} ${halfmove} ${fullmove}`;
}

// squareToIndex maps an algebraic square ("a8") to a board index (a8 = 0).
// Returns -1 for a malformed square.
export function squareToIndex(square: string): number {
  if (!square || square.length !== 2) return -1;
  const file = FILES.indexOf(square[0]);
  const rank = Number(square[1]);
  if (file < 0 || !(rank >= 1 && rank <= 8)) return -1;
  // Rank 8 is row 0, rank 1 is row 7.
  return (8 - rank) * 8 + file;
}

// indexToSquare maps a board index (a8 = 0) back to an algebraic square.
// Returns "" for an out-of-range index.
export function indexToSquare(index: number): string {
  if (index < 0 || index > 63) return "";
  const file = index % 8;
  const row = Math.floor(index / 8);
  const rank = 8 - row;
  return `${FILES[file]}${rank}`;
}
