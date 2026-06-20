"use client";

import { useEffect, useMemo, useState } from "react";
import { useT } from "@/i18n/I18nProvider";
import type { GameDraft, Header, MoveInput } from "@/lib/api-client";
import {
  allLegal,
  PLACEHOLDER,
  rebuild,
  reviewState,
  sanForDrag,
  STARTING_FEN,
  tailFen,
  toEditablePlies,
  type EditMove,
} from "@/lib/chess";
import { useGameAnalysis } from "@/hooks/useGameAnalysis";
import EngineBoard from "./EngineBoard";
import EngineControls from "./EngineControls";
import MoveList from "./MoveList";
import HeaderFields from "./HeaderFields";
import PhotoViewer from "./PhotoViewer";

type Ply = {
  san: string;
  clockSec: number | null;
  recognizedText: string;
  confidence: number;
};

export interface ReviewWorkbenchProps {
  draft: GameDraft;
  // The primary action: save (account flow) or export PGN (anonymous flow).
  // Receives the editable payload; should throw ApiError on failure so we can
  // surface failedAt highlighting.
  onPrimary: (payload: {
    header: Header;
    moves: MoveInput[];
    startFen?: string;
  }) => Promise<void>;
  primaryLabel: string;
  // Banner / status content rendered above the workbench (e.g. anonymous note).
  banner?: React.ReactNode;
  // Index of a ply the server reported illegal (from 422 failedAt), to highlight.
  serverFailedAt?: number | null;
  saving?: boolean;
  // Post-success element (e.g. "saved — link to library").
  footer?: React.ReactNode;
  readOnly?: boolean;
}

export default function ReviewWorkbench({
  draft,
  onPrimary,
  primaryLabel,
  banner,
  serverFailedAt,
  saving,
  footer,
  readOnly = false,
}: ReviewWorkbenchProps) {
  const t = useT();
  const startFen = draft.startFen || STARTING_FEN;
  const [header, setHeader] = useState<Header>(draft.header);
  const [plies, setPlies] = useState<Ply[]>(toEditablePlies(draft.moves));
  const [activeIndex, setActiveIndex] = useState<number | null>(null);
  // Ply indices the reviewer has confirmed (clears the yellow "verify" highlight).
  const [confirmed, setConfirmed] = useState<Set<number>>(new Set());
  // Toggle a clean read-only "view" of the game vs. the editing surface.
  const [viewMode, setViewMode] = useState(false);
  const [engineOn, setEngineOn] = useState(true);

  // Recompute legality whenever the plies change.
  const moves: EditMove[] = useMemo(
    () => rebuild(startFen, plies),
    [startFen, plies],
  );

  // Indices still needing verification: a legal move read with low confidence the reviewer
  // hasn't yet confirmed. Drives the "N to verify" chip and the next-uncertain jump.
  const verifyIndices = useMemo(
    () => moves.flatMap((m, i) => (reviewState(m, confirmed.has(i)) === "verify" ? [i] : [])),
    [moves, confirmed],
  );

  const confirm = (i: number) =>
    setConfirmed((prev) => (prev.has(i) ? prev : new Set(prev).add(i)));

  const jumpToNextUnverified = () => {
    const from = activeIndex ?? -1;
    const next = verifyIndices.find((i) => i > from) ?? verifyIndices[0];
    if (next !== undefined) setActiveIndex(next);
  };

  // Whole-game engine analysis (eval + blunder/mistake/inaccuracy per move).
  const analysis = useGameAnalysis(startFen, moves);

  // Any edit invalidates prior analysis (ply indices and positions shift).
  useEffect(() => {
    analysis.clear();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [plies]);

  const boardFen =
    activeIndex === null
      ? startFen
      : moves[activeIndex]?.fenAfter ?? startFen;

  // For appending new moves by dragging: use the position at the end of the
  // legal prefix when nothing is selected, else the active ply's position.
  const dragFen =
    activeIndex === null ? tailFen(moves, startFen) : boardFen;

  const ro = readOnly || viewMode;
  const canEdit = !ro && !saving;

  const handleDrag = (from: string, to: string): boolean => {
    if (!canEdit) return false;
    const san = sanForDrag(dragFen, from, to);
    if (!san) return false;

    // If a ply is selected, replace it; otherwise append.
    if (activeIndex !== null) {
      replaceSan(activeIndex, san);
    } else {
      // Append after the legal prefix.
      const insertAt = lastLegalIndex(moves);
      const next = [...plies];
      next.splice(insertAt + 1, 0, { san, clockSec: null, recognizedText: "", confidence: 1 });
      setPlies(next);
      setActiveIndex(insertAt + 1);
    }
    return true;
  };

  const replaceSan = (index: number, san: string) => {
    setPlies((prev) => {
      const next = [...prev];
      next[index] = { ...next[index], san };
      return next;
    });
  };

  const insertAfter = (index: number) => {
    setPlies((prev) => {
      const next = [...prev];
      next.splice(index + 1, 0, {
        san: PLACEHOLDER,
        clockSec: null,
        recognizedText: "",
        confidence: 1,
      });
      return next;
    });
  };

  const remove = (index: number) => {
    setPlies((prev) => prev.filter((_, i) => i !== index));
    setActiveIndex(null);
  };

  const placeholder = (index: number) => replaceSan(index, PLACEHOLDER);

  const truncate = (index: number) => {
    setPlies((prev) => prev.slice(0, index));
    setActiveIndex(null);
  };

  const orientation: "white" | "black" =
    startFen.split(" ")[1] === "b" ? "black" : "white";

  // Save gate: empty is allowed; otherwise every ply must be legal.
  const saveEnabled = canEdit && allLegal(moves);

  const buildPayload = () => ({
    header,
    startFen: draft.startFen || undefined,
    moves: moves.map<MoveInput>((m, i) => ({
      ply: i + 1,
      san: m.san,
      clockSec: m.clockSec,
    })),
  });

  // Highlight a server-reported failed ply on the board square set.
  const squareStyles = useMemo(() => {
    if (serverFailedAt == null) return undefined;
    return {} as Record<string, React.CSSProperties>;
  }, [serverFailedAt]);

  return (
    <div className="flex flex-col gap-4">
      {banner}

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        {/* Left: photo (hidden for manual entries with no image) */}
        {draft.imageUrl ? (
          <section className="min-h-[420px] rounded-lg border border-gray-200 bg-white p-3">
            <PhotoViewer src={draft.imageUrl} />
          </section>
        ) : (
          <section className="flex min-h-[420px] items-center justify-center rounded-lg border border-dashed border-gray-200 bg-gray-50 p-3 text-sm text-gray-400">
            {t("review.manualNoPhoto")}
          </section>
        )}

        {/* Right: board + header + move list */}
        <section className="flex flex-col gap-4 rounded-lg border border-gray-200 bg-white p-4">
          {!readOnly && (
            <div className="flex items-center justify-between gap-2">
              <div className="inline-flex overflow-hidden rounded border border-gray-300 text-xs">
                <button
                  type="button"
                  onClick={() => setViewMode(false)}
                  className={`px-3 py-1 ${!viewMode ? "bg-blue-600 text-white" : "bg-white text-gray-600 hover:bg-gray-100"}`}
                >
                  {t("review.edit")}
                </button>
                <button
                  type="button"
                  onClick={() => setViewMode(true)}
                  className={`px-3 py-1 ${viewMode ? "bg-blue-600 text-white" : "bg-white text-gray-600 hover:bg-gray-100"}`}
                >
                  {t("review.view")}
                </button>
              </div>
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
          )}

          <EngineBoard
            fen={boardFen}
            orientation={orientation}
            allowDragging={canEdit}
            onMove={handleDrag}
            squareStyles={squareStyles}
            count={moves.length}
            activeIndex={activeIndex}
            onSelectIndex={setActiveIndex}
            keyboard
            engine={engineOn}
            caption={ro ? t("review.captionView") : t("review.captionEdit")}
          />

          <HeaderFields header={header} onChange={setHeader} readOnly={ro} />

          {verifyIndices.length > 0 && (
            <div className="flex items-center gap-2">
              <span className="inline-flex items-center gap-1 rounded-full border border-amber-300 bg-amber-50 px-2.5 py-0.5 text-xs font-medium text-amber-800">
                ⚠ {t("review.toVerify", { n: verifyIndices.length })}
              </span>
              {!ro && (
                <button
                  type="button"
                  onClick={jumpToNextUnverified}
                  className="rounded border border-amber-300 px-2 py-0.5 text-xs text-amber-700 hover:bg-amber-50"
                >
                  {t("review.nextUncertain")} →
                </button>
              )}
            </div>
          )}

          <MoveList
            moves={moves}
            activeIndex={activeIndex}
            onSelect={setActiveIndex}
            onEditSan={(i, san) => replaceSan(i, san)}
            onInsertAfter={insertAfter}
            onDelete={remove}
            onPlaceholder={placeholder}
            onTruncate={truncate}
            readOnly={ro}
            annotations={analysis.annotations}
            confirmed={confirmed}
            onConfirm={confirm}
          />

          {serverFailedAt != null && (
            <p className="rounded border border-red-300 bg-red-50 px-3 py-2 text-sm text-red-700">
              {t("review.serverRejected", { n: serverFailedAt + 1 })}
            </p>
          )}

          {footer}

          {!ro && (
            <div className="flex items-center gap-3">
              <button
                type="button"
                disabled={!saveEnabled}
                onClick={() => onPrimary(buildPayload())}
                className="rounded bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:cursor-not-allowed disabled:bg-gray-300"
              >
                {saving ? t("review.working") : primaryLabel}
              </button>
              {!allLegal(moves) ? (
                <span className="text-xs text-amber-600">
                  {t("review.resolveIllegal")}
                </span>
              ) : (
                verifyIndices.length > 0 && (
                  <span className="text-xs text-amber-600">
                    {t("review.unverifiedNote")}
                  </span>
                )
              )}
            </div>
          )}
        </section>
      </div>
    </div>
  );
}

function lastLegalIndex(moves: EditMove[]): number {
  let idx = -1;
  for (let i = 0; i < moves.length; i++) {
    if (moves[i].legality === "legal") idx = i;
    else break;
  }
  return idx;
}
