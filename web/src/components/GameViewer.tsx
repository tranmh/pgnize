"use client";

import { useMemo, useState } from "react";
import { useT } from "@/i18n/I18nProvider";
import type { GameDraft } from "@/lib/api-client";
import {
  rebuild,
  STARTING_FEN,
  toEditablePlies,
  type EditMove,
} from "@/lib/chess";
import { useGameAnalysis } from "@/hooks/useGameAnalysis";
import { useCoach } from "@/hooks/useCoach";
import { useAuth } from "./AuthProvider";
import EngineBoard from "./EngineBoard";
import EngineControls from "./EngineControls";
import CoachButton from "./CoachButton";
import CoachPanel from "./CoachPanel";
import MoveList from "./MoveList";

const noop = () => {};

// A dead-simple, read-only board to watch a game: step through the moves, flip
// the board, and optionally run the engine. No editing, no photo, no save.
export default function GameViewer({ draft }: { draft: GameDraft }) {
  const t = useT();
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
  const { user } = useAuth();
  const coachGameId = user && draft.id ? draft.id : undefined;
  const coach = useCoach(startFen, moves, analysis, draft.header, coachGameId);
  const hasAnnotations = Object.keys(analysis.annotations).length > 0;

  const boardFen =
    activeIndex === null ? startFen : moves[activeIndex]?.fenAfter ?? startFen;

  const base: "white" | "black" =
    startFen.split(" ")[1] === "b" ? "black" : "white";
  const orientation = flip ? (base === "white" ? "black" : "white") : base;

  const h = draft.header;
  const subtitle = [
    h.event,
    h.round && t("viewer.round", { n: h.round }),
    h.board && t("viewer.board", { n: h.board }),
    h.date,
  ]
    .filter(Boolean)
    .join(" · ");

  return (
    <div className="flex flex-col gap-4">
      <div className="rounded-lg border border-gray-200 bg-white px-4 py-3">
        <div className="text-lg font-medium">
          {h.white || "?"} <span className="text-gray-400">{t("viewer.vs")}</span>{" "}
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
            {t("viewer.flip")}
          </button>
          <div className="flex flex-wrap items-center gap-3">
            <EngineControls
              engineOn={engineOn}
              onToggleEngine={setEngineOn}
              analyzing={analysis.analyzing}
              progress={analysis.progress}
              available={analysis.available}
              hasAnnotations={hasAnnotations}
              onAnalyze={analysis.run}
              onClear={analysis.clear}
            />
            <CoachButton
              hasAnnotations={hasAnnotations}
              loading={coach.loadingPly === -1}
              onClick={coach.coachGame}
            />
          </div>
        </div>

        <EngineBoard
          fen={boardFen}
          orientation={orientation}
          count={moves.length}
          activeIndex={activeIndex}
          onSelectIndex={setActiveIndex}
          keyboard
          engine={engineOn}
          caption={t("viewer.stepCaption")}
        />

        <CoachPanel coach={coach} activeIndex={activeIndex} />

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
          onExplain={(i) => {
            // Select the ply so the coach panel (keyed on activeIndex) shows its prose.
            setActiveIndex(i);
            coach.coachMove(i);
          }}
          coaching={coach.byPly}
        />
      </section>
    </div>
  );
}
