"use client";

import { useEffect } from "react";
import { useI18n } from "@/i18n/I18nProvider";
import { useSpeechSettings } from "@/i18n/SpeechSettingsProvider";
import { useSpeech } from "@/hooks/useSpeech";
import type { CoachState } from "@/hooks/useCoach";

// Logical keys for the dedupe guard: positions and game summaries use fixed
// sentinels; per-move uses the active ply index.
const KEY_GAME = -1;
const KEY_POSITION = -2;

// Renders the LLM coach's prose: the explanation for the selected move and/or
// the whole-game summary. Hidden until there is something to show. When speech
// is enabled, the currently visible text is auto-spoken aloud.
export default function CoachPanel({
  coach,
  activeIndex,
}: {
  coach: CoachState;
  activeIndex: number | null;
}) {
  const { t, locale } = useI18n();
  const settings = useSpeechSettings();
  const speech = useSpeech();
  const moveText = activeIndex !== null ? coach.byPly[activeIndex] : undefined;
  const busy = coach.loadingPly !== null;

  // Auto-speak the single visible coach text. Precedence mirrors the render
  // below: per-move (keyed by ply) → position (-2) → game (-1). Re-runs when the
  // active text changes; useSpeech's lastSpokenKey guard prevents double-speak.
  useEffect(() => {
    if (!settings.enabled) return;
    if (moveText && activeIndex !== null) {
      speech.speak(activeIndex, moveText, locale);
    } else if (coach.positionText) {
      speech.speak(KEY_POSITION, coach.positionText, locale);
    } else if (coach.gameText) {
      speech.speak(KEY_GAME, coach.gameText, locale);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [moveText, coach.positionText, coach.gameText, activeIndex, settings.enabled, locale]);

  if (!moveText && !coach.gameText && !coach.positionText && !coach.error && !busy)
    return null;

  const hasSpeakable = Boolean(moveText || coach.positionText || coach.gameText);

  return (
    <div className="flex flex-col gap-2 rounded-lg border border-indigo-200 bg-indigo-50 p-4">
      <div className="flex items-center justify-between gap-2">
        <h3 className="text-xs font-semibold uppercase tracking-wide text-indigo-500">
          {t("coach.title")}
        </h3>
        {settings.enabled && hasSpeakable && (
          <div className="flex items-center gap-1">
            <button
              type="button"
              data-testid="coach-speak-toggle"
              onClick={() => {
                if (speech.speaking) {
                  speech.stop();
                } else {
                  speech.replay();
                }
              }}
              title={speech.speaking ? t("coach.stop") : t("coach.play")}
              className="rounded border border-indigo-300 bg-white px-2 py-1 text-xs text-indigo-700 hover:bg-indigo-100"
            >
              <span aria-hidden>{speech.speaking ? "⏹" : "▶"}</span>
              <span className="sr-only">
                {speech.speaking ? t("coach.stop") : t("coach.play")}
              </span>
            </button>
            <button
              type="button"
              data-testid="coach-speak-replay"
              onClick={() => speech.replay()}
              title={t("coach.replay")}
              className="rounded border border-indigo-300 bg-white px-2 py-1 text-xs text-indigo-700 hover:bg-indigo-100"
            >
              <span aria-hidden>↻</span>
              <span className="sr-only">{t("coach.replay")}</span>
            </button>
          </div>
        )}
      </div>
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
