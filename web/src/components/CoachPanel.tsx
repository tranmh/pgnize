"use client";

import { useT } from "@/i18n/I18nProvider";
import type { CoachState } from "@/hooks/useCoach";

// Renders the LLM coach's prose: the explanation for the selected move and/or
// the whole-game summary. Hidden until there is something to show.
export default function CoachPanel({
  coach,
  activeIndex,
}: {
  coach: CoachState;
  activeIndex: number | null;
}) {
  const t = useT();
  const moveText = activeIndex !== null ? coach.byPly[activeIndex] : undefined;
  const busy = coach.loadingPly !== null;

  if (!moveText && !coach.gameText && !coach.positionText && !coach.error && !busy)
    return null;

  return (
    <div className="flex flex-col gap-2 rounded-lg border border-indigo-200 bg-indigo-50 p-4">
      <h3 className="text-xs font-semibold uppercase tracking-wide text-indigo-500">
        {t("coach.title")}
      </h3>
      {busy && <p className="text-sm text-indigo-500">{t("coach.thinking")}</p>}
      {coach.error && (
        <p className="text-sm text-red-600">{t("coach.error")}</p>
      )}
      {moveText && (
        <p data-testid="coach-move-text" className="whitespace-pre-wrap text-sm text-indigo-900">{moveText}</p>
      )}
      {coach.positionText && (
        <p data-testid="coach-position-text" className="whitespace-pre-wrap text-sm text-indigo-900">
          {coach.positionText}
        </p>
      )}
      {coach.gameText && (
        <div className="flex flex-col gap-1">
          <p className="text-xs font-semibold text-indigo-500">
            {t("coach.gameSummary")}
          </p>
          <p data-testid="coach-game-text" className="whitespace-pre-wrap text-sm text-indigo-900">
            {coach.gameText}
          </p>
        </div>
      )}
    </div>
  );
}
