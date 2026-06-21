"use client";

import { useState } from "react";
import { useT } from "@/i18n/I18nProvider";
import type { GameDraft, Header, MoveInput } from "@/lib/api-client";
import { STARTING_FEN } from "@/lib/chess";
import PositionEditor from "./PositionEditor";
import HeaderFields from "./HeaderFields";
import PhotoViewer from "./PhotoViewer";

export interface PositionReviewProps {
  draft: GameDraft;
  // The primary action: export PGN (anonymous) or save (account). Receives the
  // edited position in `startFen` with an always-empty `moves` array.
  onPrimary: (payload: {
    header: Header;
    startFen: string;
    moves: MoveInput[];
  }) => void;
  primaryLabel: string;
  // Post-action element (e.g. "saved" confirmation or a confidence note).
  footer?: React.ReactNode;
  saving?: boolean;
}

// PositionReview is the shared review shell for the board-photo flow (anonymous
// scan + account scan/review). It owns the current FEN + header state and hands
// the edited position back through onPrimary. Moves are always [].
export default function PositionReview({
  draft,
  onPrimary,
  primaryLabel,
  footer,
  saving,
}: PositionReviewProps) {
  const t = useT();
  const [header, setHeader] = useState<Header>(draft.header);
  const [fen, setFen] = useState<string>(draft.startFen || STARTING_FEN);
  const [orientation, setOrientation] = useState<"white" | "black">("white");

  return (
    <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
      {/* Left: photo (hidden when there is no source image) */}
      {draft.imageUrl ? (
        <section className="min-h-[420px] rounded-lg border border-gray-200 bg-white p-3">
          <PhotoViewer src={draft.imageUrl} />
        </section>
      ) : (
        <section className="flex min-h-[420px] items-center justify-center rounded-lg border border-dashed border-gray-200 bg-gray-50 p-3 text-sm text-gray-400">
          {t("review.manualNoPhoto")}
        </section>
      )}

      {/* Right: editor + header + primary action */}
      <section className="flex flex-col gap-4 rounded-lg border border-gray-200 bg-white p-4">
        <PositionEditor
          initialFen={draft.startFen || STARTING_FEN}
          orientation={orientation}
          onFlip={() =>
            setOrientation((o) => (o === "white" ? "black" : "white"))
          }
          onChange={setFen}
        />

        <HeaderFields header={header} onChange={setHeader} />

        {footer}

        <div className="flex items-center gap-3">
          <button
            type="button"
            disabled={saving}
            onClick={() => onPrimary({ header, startFen: fen, moves: [] })}
            className="rounded bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:cursor-not-allowed disabled:bg-gray-300"
          >
            {saving ? t("review.working") : primaryLabel}
          </button>
        </div>
      </section>
    </div>
  );
}
