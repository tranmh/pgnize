"use client";

export interface EngineControlsProps {
  engineOn: boolean;
  onToggleEngine: (on: boolean) => void;
  analyzing: boolean;
  progress: number; // 0..1
  available: boolean;
  hasAnnotations: boolean;
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
  onAnalyze,
  onClear,
}: EngineControlsProps) {
  if (!available) {
    return (
      <p className="text-xs text-gray-400">
        Engine unavailable in this browser.
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
        Engine eval
      </label>

      {analyzing ? (
        <button
          type="button"
          onClick={onClear}
          className="rounded border border-gray-300 px-2.5 py-1 text-xs hover:bg-gray-100"
        >
          Analyzing… {Math.round(progress * 100)}% (stop)
        </button>
      ) : hasAnnotations ? (
        <button
          type="button"
          onClick={onClear}
          className="rounded border border-gray-300 px-2.5 py-1 text-xs hover:bg-gray-100"
        >
          Clear analysis
        </button>
      ) : (
        <button
          type="button"
          onClick={onAnalyze}
          className="rounded border border-gray-300 px-2.5 py-1 text-xs hover:bg-gray-100"
        >
          Analyze game
        </button>
      )}
    </div>
  );
}
