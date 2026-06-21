// Build a downloadable PGN from a header + start FEN + SAN list, client-side.
//
// Used by the anonymous /new flow, which has no persisted game row and therefore
// no server export endpoint. chess.js handles move numbering and the SetUp/FEN
// tags for non-standard start positions. The moves are already legal SAN
// (validated by the server on the way in), but we stop defensively on the first
// move chess.js rejects so a download never throws.

import { Chess } from "chess.js";
import { STARTING_FEN } from "./chess";
import type { Header } from "./api-client";

export function buildPgn(
  header: Header,
  startFen: string,
  sans: string[],
): string {
  const fen = startFen || STARTING_FEN;
  const chess = fen === STARTING_FEN ? new Chess() : new Chess(fen);
  chess.setHeader("Event", header.event || "?");
  chess.setHeader("Site", header.site || "?");
  chess.setHeader("Date", header.date || "????.??.??");
  chess.setHeader("Round", header.round || "-");
  chess.setHeader("White", header.white || "?");
  chess.setHeader("Black", header.black || "?");
  chess.setHeader("Result", header.result || "*");
  if (header.board) chess.setHeader("Board", header.board);
  for (const san of sans) {
    try {
      chess.move(san);
    } catch {
      break;
    }
  }
  return chess.pgn();
}
