"use client";

import { useMemo, useState } from "react";
import type { GameDraft } from "@/lib/api-client";
import {
  rebuild,
  STARTING_FEN,
  toEditablePlies,
  type EditMove,
} from "@/lib/chess";
import { useGameAnalysis } from "@/hooks/useGameAnalysis";
import EngineBoard from "./EngineBoard";
import EngineControls from "./EngineControls";
import MoveList from "./MoveList";

const noop = () => {};

// A dead-simple, read-only board to watch a game: step through the moves, flip
// the board, and optionally run the engine. No editing, no photo, no save.
export default function GameViewer({ draft }: { draft: GameDraft }) {
  const startFen = draft.startFen || STARTING_FEN;
  const plies = useMemo(() => toEditablePlies(draft.moves), [draft.moves]);
  const moves: EditMove[] = useMemo(
    () => rebuild(startFen, plies),
    [startFen, plies],
  );

  const [activeIndex, setActiveIndex] = useState<number | null>(null);
  const [engineOn, setEngineOn] = useState(true);
  const [flip, setFlip] = useState(false);

  const analysis = useGameAnalysis(startFen, moves);

  const boardFen =
    activeIndex === null ? startFen : moves[activeIndex]?.fenAfter ?? startFen;

  const base: "white" | "black" =
    startFen.split(" ")[1] === "b" ? "black" : "white";
  const orientation = flip ? (base === "white" ? "black" : "white") : base;

  const h = draft.header;
  const subtitle = [
    h.event,
    h.round && `Round ${h.round}`,
    h.board && `Board ${h.board}`,
    h.date,
  ]
    .filter(Boolean)
    .join(" · ");

  return (
    <div className="flex flex-col gap-4">
      <div className="rounded-lg border border-gray-200 bg-white px-4 py-3">
        <div className="text-lg font-medium">
          {h.white || "?"} <span className="text-gray-400">vs</span>{" "}
          {h.black || "?"}{" "}
          <span className="ml-2 font-mono text-sm text-gray-500">{h.result}</span>
        </div>
        {subtitle && <div className="text-sm text-gray-500">{subtitle}</div>}
      </div>

      <section className="flex flex-col gap-4 rounded-lg border border-gray-200 bg-white p-4">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <button
            type="button"
            onClick={() => setFlip((f) => !f)}
            className="rounded border border-gray-300 px-2.5 py-1 text-xs hover:bg-gray-100"
          >
            Flip board
          </button>
          <EngineControls
            engineOn={engineOn}
            onToggleEngine={setEngineOn}
            analyzing={analysis.analyzing}
            progress={analysis.progress}
            available={analysis.available}
            hasAnnotations={Object.keys(analysis.annotations).length > 0}
            onAnalyze={analysis.run}
            onClear={analysis.clear}
          />
        </div>

        <EngineBoard
          fen={boardFen}
          orientation={orientation}
          count={moves.length}
          activeIndex={activeIndex}
          onSelectIndex={setActiveIndex}
          keyboard
          engine={engineOn}
          caption="Use ◀ ▶ or the arrow keys to step through the game."
        />

        <MoveList
          moves={moves}
          activeIndex={activeIndex}
          onSelect={setActiveIndex}
          onEditSan={noop}
          onInsertAfter={noop}
          onDelete={noop}
          onPlaceholder={noop}
          onTruncate={noop}
          readOnly
          annotations={analysis.annotations}
        />
      </section>
    </div>
  );
}
