"use client";

import { useT } from "@/i18n/I18nProvider";

export interface EngineControlsProps {
  engineOn: boolean;
  onToggleEngine: (on: boolean) => void;
  analyzing: boolean;
  progress: number; // 0..1
  available: boolean;
  hasAnnotations: boolean;
  // Whether a full-game analysis is meaningful (false for a single pasted position).
  canAnalyze?: boolean;
  onAnalyze: () => void;
  onClear: () => void;
}

// Toolbar for the in-browser engine: toggle live evaluation and run (or clear)
// a full-game analysis. Degrades gracefully when no engine worker is available.
export default function EngineControls({
  engineOn,
  onToggleEngine,
  analyzing,
  progress,
  available,
  hasAnnotations,
  canAnalyze = true,
  onAnalyze,
  onClear,
}: EngineControlsProps) {
  const t = useT();
  if (!available) {
    return (
      <p className="text-xs text-gray-400">
        {t("engine.unavailable")}
      </p>
    );
  }

  return (
    <div className="flex flex-wrap items-center gap-3 text-sm">
      <label className="flex items-center gap-1.5 text-gray-600">
        <input
          type="checkbox"
          checked={engineOn}
          onChange={(e) => onToggleEngine(e.target.checked)}
        />
        {t("engine.eval")}
      </label>

      {analyzing ? (
        <button
          type="button"
          onClick={onClear}
          className="rounded border border-gray-300 px-2.5 py-1 text-xs hover:bg-gray-100"
        >
          {t("engine.analyzing", { pct: Math.round(progress * 100) })}
        </button>
      ) : hasAnnotations ? (
        <button
          type="button"
          onClick={onClear}
          className="rounded border border-gray-300 px-2.5 py-1 text-xs hover:bg-gray-100"
        >
          {t("engine.clear")}
        </button>
      ) : canAnalyze ? (
        <button
          type="button"
          onClick={onAnalyze}
          className="rounded border border-gray-300 px-2.5 py-1 text-xs hover:bg-gray-100"
        >
          {t("engine.analyze")}
        </button>
      ) : null}
    </div>
  );
}
