// Turn engine evaluations into human-friendly per-move annotations
// (blunder / mistake / inaccuracy). Pure functions — no worker, no React.

import { type Score, scoreToCp } from "./engine";
import type { Side } from "./api-client";

export type MoveQuality = "blunder" | "mistake" | "inaccuracy" | null;

export interface MoveAnnotation {
  // Evaluation AFTER this move, from White's point of view.
  score: Score;
  quality: MoveQuality;
}

// Centipawn loss thresholds (loss = how much the mover worsened their own eval).
const BLUNDER = 300;
const MISTAKE = 150;
const INACCURACY = 75;
// Once a side is decided-losing by this much, further "losses" aren't worth nagging.
const DECIDED = 1000;

// Classify a single move given the White-POV eval before and after it.
export function classify(prevCp: number, currCp: number, side: Side): MoveQuality {
  const delta = currCp - prevCp; // change in White's favor
  const loss = side === "white" ? -delta : delta; // cp the mover gave up
  if (loss < INACCURACY) return null;

  // Don't flag a move that was made from an already-hopeless position.
  const moverBefore = side === "white" ? prevCp : -prevCp;
  const moverAfter = side === "white" ? currCp : -currCp;
  if (moverBefore < -DECIDED && moverAfter < -DECIDED) return null;

  if (loss >= BLUNDER) return "blunder";
  if (loss >= MISTAKE) return "mistake";
  return "inaccuracy";
}

export const QUALITY_GLYPH: Record<Exclude<MoveQuality, null>, string> = {
  blunder: "??",
  mistake: "?",
  inaccuracy: "?!",
};

export const QUALITY_LABEL: Record<Exclude<MoveQuality, null>, string> = {
  blunder: "Blunder",
  mistake: "Mistake",
  inaccuracy: "Inaccuracy",
};

// Build per-ply annotations from White-POV evals. `evals[i]` is the eval AFTER
// ply i; `startCp` is the eval of the position before ply 0. Stops at the first
// missing eval (analysis truncated / in progress).
export function annotate(
  startCp: number,
  evals: (Score | undefined)[],
  sides: Side[],
): Record<number, MoveAnnotation> {
  const out: Record<number, MoveAnnotation> = {};
  let prevCp = startCp;
  for (let i = 0; i < evals.length; i++) {
    const score = evals[i];
    if (!score) break;
    const currCp = scoreToCp(score);
    out[i] = { score, quality: classify(prevCp, currCp, sides[i]) };
    prevCp = currCp;
  }
  return out;
}

// Format a White-POV score for display: "+1.3", "-0.5", "M3", "-M2".
export function formatScore(score: Score): string {
  if (score.mate !== null) {
    if (score.mate === 0) return "#";
    return `${score.mate > 0 ? "" : "-"}M${Math.abs(score.mate)}`;
  }
  const pawns = (score.cp ?? 0) / 100;
  const sign = pawns > 0 ? "+" : pawns < 0 ? "" : "";
  return `${sign}${pawns.toFixed(1)}`;
}
