"use client";

import { useT } from "@/i18n/I18nProvider";
import { useSpeechSettings } from "@/i18n/SpeechSettingsProvider";
import { getSpeechController, type SpeechSource } from "@/lib/tts";

// Global speech control for the nav: a mute/unmute toggle plus a source select
// (server cloud TTS vs. browser Web Speech). Persists via SpeechSettingsProvider.
export default function SpeechToggle() {
  const t = useT();
  const { enabled, source, setEnabled, setSource } = useSpeechSettings();

  return (
    <div
      data-testid="speech-toggle"
      className="inline-flex items-center gap-2 text-xs"
    >
      <button
        type="button"
        onClick={() => {
          const next = !enabled;
          if (!next) getSpeechController().stop();
          setEnabled(next);
        }}
        aria-pressed={enabled}
        title={enabled ? t("tts.on") : t("tts.off")}
        className={`rounded border px-2 py-1 font-medium ${
          enabled
            ? "border-blue-600 bg-blue-600 text-white"
            : "border-gray-300 bg-white text-gray-600 hover:bg-gray-100"
        }`}
      >
        <span aria-hidden>{enabled ? "🔊" : "🔇"}</span>
        <span className="sr-only">{enabled ? t("tts.on") : t("tts.off")}</span>
      </button>
      <select
        value={source}
        onChange={(e) => setSource(e.target.value as SpeechSource)}
        disabled={!enabled}
        aria-label={t("tts.sourceLabel")}
        className="rounded border border-gray-300 bg-white px-1 py-1 text-gray-700 disabled:opacity-50"
      >
        <option value="server">{t("tts.source.server")}</option>
        <option value="browser">{t("tts.source.browser")}</option>
      </select>
    </div>
  );
}
