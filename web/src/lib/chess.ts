// Client-side ADVISORY chess logic via chess.js.
//
// The server is authoritative on save/export (it replays through chesskit).
// Everything here is purely to give the reviewer live feedback: legality
// badges, legal-move dropdowns, board jumps, and drag-to-move SAN.

import { Chess } from "chess.js";
import type { Move, Side } from "./api-client";

export const STARTING_FEN =
  "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1";

export type Legality = "legal" | "illegal" | "ambiguous";

// A reviewer-editable ply. `san` may be a real move, the "?" placeholder, or "".
export interface EditMove {
  san: string;
  clockSec: number | null;
  recognizedText: string;
  // computed:
  side: Side;
  fenBefore: string;
  fenAfter: string; // equals fenBefore when not legal
  legality: Legality;
  // true once it diverges from the recognized text
  corrected: boolean;
  // when legality === "ambiguous", the candidate SANs sharing the input
  ambiguousOptions: string[];
}

export const PLACEHOLDER = "?";

function sideForPly(index: number, startFen: string): Side {
  // Whose move it is depends on the side to move in startFen plus the ply index.
  const startToMove = startFen.split(" ")[1] === "b" ? 1 : 0;
  return (index + startToMove) % 2 === 0 ? "white" : "black";
}

// Returns true if `san` is the placeholder or an empty/unknown token.
export function isPlaceholder(san: string): boolean {
  const t = san.trim();
  return t === "" || t === PLACEHOLDER;
}

// All legal SANs from a given FEN. Empty array if the FEN itself is unusable.
export function legalMovesFrom(fen: string): string[] {
  try {
    const c = new Chess(fen);
    return c.moves();
  } catch {
    return [];
  }
}

// Try to apply a single SAN to a FEN. Detects illegal vs ambiguous inputs.
function applyOne(
  fen: string,
  san: string,
): { ok: true; fenAfter: string } | { ok: false; ambiguous: string[] } {
  const c = new Chess(fen);
  try {
    c.move(san);
    return { ok: true, fenAfter: c.fen() };
  } catch {
    // chess.js throws on both illegal and ambiguous. Distinguish ambiguity by
    // finding legal moves whose SAN (stripped of disambiguation/check marks)
    // matches the user's input.
    const stripped = stripSan(san);
    if (stripped) {
      const candidates = c
        .moves()
        .filter((m) => stripSan(m) === stripped || looselyMatches(m, san));
      if (candidates.length > 1) {
        return { ok: false, ambiguous: candidates };
      }
    }
    return { ok: false, ambiguous: [] };
  }
}

// Remove check/mate marks and source-square disambiguation so "Nbd2" and
// "Nfd2" both reduce to a comparable core ("Nd2").
function stripSan(san: string): string {
  return san
    .replace(/[+#]/g, "")
    .replace(/=([QRBN])/, "")
    .replace(/^([KQRBN])[a-h1-8]?x?/, "$1")
    .replace(/x/g, "")
    .trim();
}

function looselyMatches(legal: string, input: string): boolean {
  const a = stripSan(legal);
  const b = stripSan(input);
  return a.length > 0 && a === b;
}

// Rebuild the full computed move list from editable plies. Each ply is
// validated from the prior ply's resulting position; once a ply is illegal,
// all downstream plies are marked illegal too (downstream is blocked in the UI).
export function rebuild(
  startFen: string,
  plies: { san: string; clockSec: number | null; recognizedText: string }[],
): EditMove[] {
  const out: EditMove[] = [];
  let fen = startFen || STARTING_FEN;
  let blocked = false;

  plies.forEach((p, i) => {
    const side = sideForPly(i, startFen || STARTING_FEN);
    const corrected = p.san.trim() !== p.recognizedText.trim();

    if (isPlaceholder(p.san)) {
      // A placeholder is "ambiguous" (amber) — known to be incomplete, and it
      // blocks everything after it because we can't advance the position.
      out.push({
        san: PLACEHOLDER,
        clockSec: p.clockSec,
        recognizedText: p.recognizedText,
        side,
        fenBefore: fen,
        fenAfter: fen,
        legality: "ambiguous",
        corrected,
        ambiguousOptions: [],
      });
      blocked = true;
      return;
    }

    if (blocked) {
      out.push({
        san: p.san,
        clockSec: p.clockSec,
        recognizedText: p.recognizedText,
        side,
        fenBefore: fen,
        fenAfter: fen,
        legality: "illegal",
        corrected,
        ambiguousOptions: [],
      });
      return;
    }

    const res = applyOne(fen, p.san);
    if (res.ok) {
      out.push({
        san: p.san,
        clockSec: p.clockSec,
        recognizedText: p.recognizedText,
        side,
        fenBefore: fen,
        fenAfter: res.fenAfter,
        legality: "legal",
        corrected,
        ambiguousOptions: [],
      });
      fen = res.fenAfter;
    } else if (res.ambiguous.length > 1) {
      out.push({
        san: p.san,
        clockSec: p.clockSec,
        recognizedText: p.recognizedText,
        side,
        fenBefore: fen,
        fenAfter: fen,
        legality: "ambiguous",
        corrected,
        ambiguousOptions: res.ambiguous,
      });
      blocked = true;
    } else {
      out.push({
        san: p.san,
        clockSec: p.clockSec,
        recognizedText: p.recognizedText,
        side,
        fenBefore: fen,
        fenAfter: fen,
        legality: "illegal",
        corrected,
        ambiguousOptions: [],
      });
      blocked = true;
    }
  });

  return out;
}

// Given a FEN and a drag (from/to squares, optional promotion), return the SAN
// of the resulting move, or null if illegal.
export function sanForDrag(
  fen: string,
  from: string,
  to: string,
  promotion: string = "q",
): string | null {
  try {
    const c = new Chess(fen);
    const m = c.move({ from, to, promotion });
    return m ? m.san : null;
  } catch {
    return null;
  }
}

// The position from which a drag should be interpreted is the FEN before the
// first non-legal ply, i.e. the last reachable position. The board "tail" lets
// the reviewer keep adding moves at the end.
export function tailFen(moves: EditMove[], startFen: string): string {
  for (let i = moves.length - 1; i >= 0; i--) {
    if (moves[i].legality === "legal") return moves[i].fenAfter;
  }
  return startFen || STARTING_FEN;
}

// True when every ply is legal (game may be empty). Truncation is handled by
// the caller deleting downstream plies, so "all legal" is the save gate.
export function allLegal(moves: EditMove[]): boolean {
  return moves.every((m) => m.legality === "legal");
}

// Convert API moves into editable plies (drop computed fields; keep san/clock/text).
export function toEditablePlies(moves: Move[]): {
  san: string;
  clockSec: number | null;
  recognizedText: string;
}[] {
  return moves.map((m) => ({
    san: m.san,
    clockSec: m.clockSec,
    recognizedText: m.recognizedText,
  }));
}
