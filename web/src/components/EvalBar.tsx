"use client";

import { formatScore } from "@/lib/analysis";
import { type Score, scoreToCp } from "@/lib/engine";

export interface EvalBarProps {
  score: Score | null;
  thinking?: boolean;
  // Match the board height so the bar lines up next to it.
  height?: number;
}

// Map a White-POV centipawn eval to White's share of the bar (0..1) via a
// gentle sigmoid, so small edges are visible but big ones saturate.
function whiteFraction(score: Score): number {
  if (score.mate !== null) return score.mate > 0 ? 1 : 0;
  const cp = score.cp ?? 0;
  return 1 / (1 + Math.pow(10, -cp / 400));
}

// A vertical evaluation bar (White on the bottom) with a numeric readout.
// Renders a neutral placeholder until the engine produces a first score.
export default function EvalBar({ score, thinking, height = 480 }: EvalBarProps) {
  const frac = score ? whiteFraction(score) : 0.5;
  const label = score ? formatScore(score) : thinking ? "…" : "—";
  const whiteAhead = score ? scoreToCp(score) >= 0 : true;

  return (
    <div className="flex flex-col items-center gap-1" aria-hidden>
      <div
        className="relative w-5 overflow-hidden rounded bg-neutral-800"
        style={{ height }}
        title={score ? `Evaluation: ${label}` : "Engine evaluation"}
      >
        {/* White's portion grows from the bottom. */}
        <div
          className="absolute bottom-0 left-0 w-full bg-neutral-100 transition-[height] duration-200"
          style={{ height: `${Math.round(frac * 100)}%` }}
        />
        {/* Midline at 0.00 */}
        <div className="absolute left-0 top-1/2 h-px w-full bg-neutral-500/60" />
      </div>
      <span
        className={`font-mono text-xs tabular-nums ${
          whiteAhead ? "text-neutral-800" : "text-neutral-500"
        }`}
      >
        {label}
      </span>
    </div>
  );
}
