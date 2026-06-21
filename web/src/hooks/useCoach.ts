"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { getEngine } from "@/lib/engine";
import type { GameAnalysis } from "@/hooks/useGameAnalysis";
import type { EditMove } from "@/lib/chess";
import type { Header } from "@/lib/api-client";
import {
  coachGame as apiCoachGame,
  coachMove as apiCoachMove,
  type CoachGameMove,
} from "@/lib/api-client";
import { buildCoachMoveRequest, scoreToEval } from "@/lib/coach";

export interface CoachState {
  // Per-ply coaching prose, keyed by ply index.
  byPly: Record<number, string>;
  // Whole-game summary, once requested.
  gameText: string | null;
  // Ply index currently being coached, -1 for the whole-game summary, else null.
  loadingPly: number | null;
  error: string | null;
  // Explain a single ply (must already be engine-analyzed so the eval/quality exist).
  coachMove: (ply: number) => Promise<void>;
  // Summarize the whole game.
  coachGame: () => Promise<void>;
  clear: () => void;
}

// useCoach turns the existing engine analysis into LLM coaching prose.
//
// The browser engine (useGameAnalysis) only retains the eval AFTER each move —
// its pv is the continuation, NOT the alternative the engine would have played.
// So coachMove runs a fresh, on-demand search at the pre-move position to get the
// engine's recommended line, then sends both evals to the backend coach.
export function useCoach(
  startFen: string,
  moves: EditMove[],
  analysis: GameAnalysis,
  header: Header,
  gameId?: string,
  lang?: string,
  depth = 14,
): CoachState {
  const [byPly, setByPly] = useState<Record<number, string>>({});
  const [gameText, setGameText] = useState<string | null>(null);
  const [loadingPly, setLoadingPly] = useState<number | null>(null);
  const [error, setError] = useState<string | null>(null);
  const abortRef = useRef<AbortController | null>(null);

  const clear = useCallback(() => {
    abortRef.current?.abort();
    abortRef.current = null;
    setByPly({});
    setGameText(null);
    setLoadingPly(null);
    setError(null);
  }, []);

  // Coaching references specific plies/positions; any edit shifts them, so drop it.
  useEffect(() => {
    clear();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [moves]);

  const coachMove = useCallback(
    async (ply: number) => {
      const move = moves[ply];
      if (!move || move.legality !== "legal") return;
      const engine = getEngine();
      if (!engine) {
        setError("engine_unavailable");
        return;
      }
      abortRef.current?.abort();
      const controller = new AbortController();
      abortRef.current = controller;
      setError(null);
      setLoadingPly(ply);
      try {
        // Fresh search at the pre-move position → engine's best line + pre-move eval.
        const { best } = await engine.analyze(move.fenBefore, {
          multipv: 1,
          depth,
          signal: controller.signal,
        });
        if (controller.signal.aborted) return;
        const annotation = analysis.annotations[ply];
        const req = buildCoachMoveRequest({
          move,
          bestScore: best,
          afterScore: annotation?.score,
          quality: annotation?.quality ?? null,
          gameId,
          ply,
          lang,
        });
        const res = await apiCoachMove(req);
        if (controller.signal.aborted) return;
        setByPly((prev) => ({ ...prev, [ply]: res.text }));
      } catch (e) {
        if (!controller.signal.aborted) {
          setError(e instanceof Error ? e.message : "coach_failed");
        }
      } finally {
        if (abortRef.current === controller) setLoadingPly(null);
      }
    },
    [moves, analysis, gameId, lang, depth],
  );

  const coachGame = useCallback(async () => {
    // Only the legal prefix can be summarized.
    const legal: EditMove[] = [];
    for (const m of moves) {
      if (m.legality === "legal") legal.push(m);
      else break;
    }
    if (legal.length === 0) return;
    abortRef.current?.abort();
    const controller = new AbortController();
    abortRef.current = controller;
    setError(null);
    setLoadingPly(-1);
    try {
      const gameMoves: CoachGameMove[] = legal.map((m, i) => ({
        san: m.san,
        side: m.side,
        evalAfter: scoreToEval(analysis.annotations[i]?.score),
        quality: analysis.annotations[i]?.quality ?? "",
      }));
      const res = await apiCoachGame({
        gameId,
        startFen,
        header,
        moves: gameMoves,
        lang,
      });
      if (controller.signal.aborted) return;
      setGameText(res.text);
    } catch (e) {
      if (!controller.signal.aborted) {
        setError(e instanceof Error ? e.message : "coach_failed");
      }
    } finally {
      if (abortRef.current === controller) setLoadingPly(null);
    }
  }, [moves, analysis, startFen, header, gameId, lang]);

  return { byPly, gameText, loadingPly, error, coachMove, coachGame, clear };
}
