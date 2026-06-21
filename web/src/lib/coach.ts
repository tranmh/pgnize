// Pure glue between the browser engine and the backend coach.
//
// The engine speaks UCI (e.g. "e2e4", "e7e8q"); the coach prompt reads better in
// SAN ("e4", "e8=Q"). These helpers convert engine output to SAN and assemble the
// wire request. No React, no network — easy to test in isolation.

import { Chess } from "chess.js";
import { sanForDrag, uciToSquares, type EditMove } from "./chess";
import type { Score } from "./engine";
import type { CoachEval, CoachMoveRequest } from "./api-client";
import type { MoveQuality } from "./analysis";

// Convert one UCI move to SAN given the position it is played in.
// IMPORTANT: uciToSquares drops the promotion suffix, so read uci[4] here —
// otherwise every promotion silently becomes a queen.
export function uciToSan(fen: string, uci: string): string | null {
  const sq = uciToSquares(uci);
  if (!sq) return null;
  const promotion = uci.length > 4 ? uci[4] : "q";
  return sanForDrag(fen, sq.from, sq.to, promotion);
}

// Replay a UCI principal variation into a SAN line, starting from `fen`.
// Stops at the first move that does not apply (truncated/garbage PV), and never
// returns more than `limit` moves.
export function pvToSanLine(fen: string, pv: string[], limit = 6): string[] {
  const out: string[] = [];
  let chess: Chess;
  try {
    chess = new Chess(fen);
  } catch {
    return out;
  }
  for (const uci of pv) {
    if (out.length >= limit) break;
    const sq = uciToSquares(uci);
    if (!sq) break;
    const promotion = uci.length > 4 ? uci[4] : "q";
    try {
      const m = chess.move({ from: sq.from, to: sq.to, promotion });
      if (!m) break;
      out.push(m.san);
    } catch {
      break;
    }
  }
  return out;
}

// Collapse a White-POV engine Score into the nullable wire eval shape.
export function scoreToEval(score: Score | undefined | null): CoachEval {
  if (!score) return { cp: null, mate: null };
  return { cp: score.cp, mate: score.mate };
}

// Assemble the per-move coach request from a played move plus engine evals.
// `bestScore` is the engine's evaluation of the position BEFORE the move (its pv
// is the engine's recommended line, the alternative to what was played);
// `afterScore` is the evaluation AFTER the played move.
export function buildCoachMoveRequest(args: {
  move: EditMove;
  bestScore: Score;
  afterScore: Score | undefined;
  quality: MoveQuality;
  gameId?: string;
  ply?: number;
  lang?: string;
}): CoachMoveRequest {
  const { move, bestScore, afterScore, quality, gameId, ply, lang } = args;
  const bestUci = bestScore.bestMove ?? bestScore.pv[0] ?? null;
  const bestSan = bestUci ? uciToSan(move.fenBefore, bestUci) ?? "" : "";
  return {
    gameId,
    ply,
    fen: move.fenBefore,
    side: move.side,
    playedSan: move.san,
    bestSan,
    bestLine: pvToSanLine(move.fenBefore, bestScore.pv),
    evalBefore: scoreToEval(bestScore),
    evalAfter: scoreToEval(afterScore),
    quality: quality ?? "",
    lang,
  };
}
