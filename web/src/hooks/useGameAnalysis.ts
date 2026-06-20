"use client";

import { useCallback, useRef, useState } from "react";
import {
  analyzePositions,
  getEngine,
  scoreToCp,
  type Score,
} from "@/lib/engine";
import { annotate, type MoveAnnotation } from "@/lib/analysis";
import type { EditMove } from "@/lib/chess";

export interface GameAnalysis {
  annotations: Record<number, MoveAnnotation>;
  analyzing: boolean;
  progress: number; // 0..1
  available: boolean;
  run: () => void;
  clear: () => void;
}

// Walk the legal prefix of a game with the engine, producing per-move
// annotations (eval + blunder/mistake/inaccuracy). Sequential and abortable;
// annotations stream in as each position finishes.
export function useGameAnalysis(
  startFen: string,
  moves: EditMove[],
  depth = 12,
): GameAnalysis {
  const [annotations, setAnnotations] = useState<Record<number, MoveAnnotation>>(
    {},
  );
  const [analyzing, setAnalyzing] = useState(false);
  const [progress, setProgress] = useState(0);
  const abortRef = useRef<AbortController | null>(null);

  const available = typeof window !== "undefined" && !!getEngine();

  const clear = useCallback(() => {
    abortRef.current?.abort();
    abortRef.current = null;
    setAnnotations({});
    setProgress(0);
    setAnalyzing(false);
  }, []);

  const run = useCallback(async () => {
    // Only the legal prefix can be replayed and evaluated.
    const legal: EditMove[] = [];
    for (const m of moves) {
      if (m.legality === "legal") legal.push(m);
      else break;
    }
    if (legal.length === 0) return;

    abortRef.current?.abort();
    const controller = new AbortController();
    abortRef.current = controller;
    setAnalyzing(true);
    setProgress(0);
    setAnnotations({});

    const sides = legal.map((m) => m.side);
    // Evaluate the start position followed by each resulting position.
    const fens = [startFen, ...legal.map((m) => m.fenAfter)];
    const evals: (Score | undefined)[] = new Array(legal.length).fill(undefined);
    let startCp = 0;

    await analyzePositions(fens, {
      depth,
      signal: controller.signal,
      onEach: (i, score) => {
        if (i === 0) startCp = scoreToCp(score);
        else evals[i - 1] = score;
        setProgress((i + 1) / fens.length);
        setAnnotations(annotate(startCp, evals, sides));
      },
    });

    if (!controller.signal.aborted) setAnalyzing(false);
  }, [startFen, moves, depth]);

  return { annotations, analyzing, progress, available, run, clear };
}
