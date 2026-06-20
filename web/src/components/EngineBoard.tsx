"use client";

import { useEngineEval } from "@/hooks/useEngineEval";
import { uciToSquares } from "@/lib/chess";
import Board from "./Board";
import EvalBar from "./EvalBar";
import MoveNav from "./MoveNav";

export interface EngineBoardProps {
  // Position currently shown on the board.
  fen: string;
  orientation?: "white" | "black";
  // Editing affordances (omitted in read-only viewers).
  allowDragging?: boolean;
  onMove?: (from: string, to: string) => boolean;
  squareStyles?: Record<string, React.CSSProperties>;
  // Navigation through the game.
  count: number;
  activeIndex: number | null;
  onSelectIndex: (index: number | null) => void;
  keyboard?: boolean;
  // Whether to run the engine for live eval + best-move arrow.
  engine?: boolean;
  caption?: React.ReactNode;
}

// The board, its evaluation bar, the engine's best-move arrow, and the
// first/prev/next/last navigation — bundled so the edit workbench and the
// read-only viewer present an identical board experience.
export default function EngineBoard({
  fen,
  orientation = "white",
  allowDragging = false,
  onMove,
  squareStyles,
  count,
  activeIndex,
  onSelectIndex,
  keyboard = false,
  engine = true,
  caption,
}: EngineBoardProps) {
  const { score, thinking, available } = useEngineEval(fen, { enabled: engine });

  const arrows =
    engine && score?.bestMove
      ? (() => {
          const sq = uciToSquares(score.bestMove);
          return sq ? [{ from: sq.from, to: sq.to }] : undefined;
        })()
      : undefined;

  return (
    <div className="flex flex-col items-center gap-2">
      <div className="flex items-stretch justify-center gap-2">
        {engine && available && (
          <EvalBar score={score} thinking={thinking} height={480} />
        )}
        <Board
          fen={fen}
          orientation={orientation}
          allowDragging={allowDragging}
          onMove={onMove}
          squareStyles={squareStyles}
          arrows={arrows}
        />
      </div>

      <MoveNav
        index={activeIndex}
        count={count}
        onChange={onSelectIndex}
        keyboard={keyboard}
      />

      {caption && <div className="text-xs text-gray-400">{caption}</div>}
    </div>
  );
}
