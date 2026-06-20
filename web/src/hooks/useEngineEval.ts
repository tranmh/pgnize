"use client";

import { useEffect, useState } from "react";
import { getEngine, type Score } from "@/lib/engine";

export interface EngineEval {
  score: Score | null;
  // True while a search for the current FEN is still running.
  thinking: boolean;
  // False when the browser can't host the engine worker at all.
  available: boolean;
}

const DEBOUNCE_MS = 250;

// Live, debounced evaluation of a single position. Re-analyzes whenever `fen`
// changes; the previous search is aborted so navigation stays snappy. Updates
// progressively as the engine searches deeper.
export function useEngineEval(
  fen: string | null,
  opts: { depth?: number; enabled?: boolean } = {},
): EngineEval {
  const enabled = opts.enabled ?? true;
  const [score, setScore] = useState<Score | null>(null);
  const [thinking, setThinking] = useState(false);
  const [available, setAvailable] = useState(true);

  useEffect(() => {
    if (!enabled || !fen) {
      setScore(null);
      setThinking(false);
      return;
    }
    const engine = getEngine();
    if (!engine) {
      setAvailable(false);
      return;
    }

    const controller = new AbortController();
    setScore(null);
    setThinking(true);

    const timer = setTimeout(() => {
      engine
        .analyze(fen, {
          depth: opts.depth ?? 14,
          signal: controller.signal,
          onUpdate: (s) => {
            if (!controller.signal.aborted) setScore(s);
          },
        })
        .then((res) => {
          if (!controller.signal.aborted) setScore(res.best);
        })
        .catch(() => undefined)
        .finally(() => {
          if (!controller.signal.aborted) setThinking(false);
        });
    }, DEBOUNCE_MS);

    return () => {
      clearTimeout(timer);
      controller.abort();
    };
  }, [fen, enabled, opts.depth]);

  return { score, thinking, available };
}
