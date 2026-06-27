"use client";

import { useEffect, useRef, useState } from "react";
import { useI18n } from "@/i18n/I18nProvider";
import { useSpeechSettings } from "@/i18n/SpeechSettingsProvider";
import { useSpeech } from "@/hooks/useSpeech";
import { useChatCoach } from "@/hooks/useChatCoach";
import { useVoiceInput } from "@/hooks/useVoiceInput";
import type { Side } from "@/lib/api-client";

// ChatCoachPanel is the conversational coach: a collapsible thread where the user
// asks about the current position by typing or speaking, and the coach answers
// (server-side Stockfish + LLM). The latest answer is auto-spoken via the shared
// TTS controller, so it never overlaps the one-shot CoachPanel's speech.
export default function ChatCoachPanel({
  fen,
  side,
  gameId,
  ply,
  defaultOpen = false,
}: {
  fen: string;
  side: Side;
  gameId?: string;
  ply?: number | null;
  defaultOpen?: boolean;
}) {
  const { t, locale } = useI18n();
  const settings = useSpeechSettings();
  const speech = useSpeech();
  const chat = useChatCoach({ fen, side, gameId, ply, lang: locale });

  const [open, setOpen] = useState(defaultOpen);
  const [input, setInput] = useState("");
  const listRef = useRef<HTMLDivElement | null>(null);

  const voice = useVoiceInput({
    sttSource: settings.sttSource,
    lang: locale,
    onAudio: (blob) => void chat.sendAudio(blob),
    onPartialTranscript: (text) => setInput(text),
    onFinalTranscript: (text) => setInput(text),
  });

  // Auto-speak only the latest coach message. The chat:<id> key never collides
  // with CoachPanel's numeric keys, and the shared controller stops any prior
  // utterance — so chat and one-shot coaching can't speak over each other.
  useEffect(() => {
    if (!settings.enabled) return;
    const last = chat.messages[chat.messages.length - 1];
    if (last?.role === "coach" && last.text) {
      speech.speak(`chat:${last.id}`, last.text, locale);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [chat.messages, settings.enabled, locale]);

  // Keep the newest message in view.
  useEffect(() => {
    listRef.current?.scrollTo({ top: listRef.current.scrollHeight });
  }, [chat.messages.length, chat.inFlight]);

  const submit = () => {
    const q = input.trim();
    if (!q || chat.inFlight) return;
    setInput("");
    void chat.sendText(q);
  };

  if (!open) {
    return (
      <button
        type="button"
        data-testid="chat-coach-toggle"
        onClick={() => setOpen(true)}
        className="self-start rounded-lg border border-indigo-300 bg-white px-3 py-2 text-sm font-medium text-indigo-700 hover:bg-indigo-50"
      >
        💬 {t("chat.open")}
      </button>
    );
  }

  return (
    <div className="flex flex-col gap-2 rounded-lg border border-indigo-200 bg-indigo-50 p-4">
      <div className="flex items-center justify-between gap-2">
        <h3 className="text-xs font-semibold uppercase tracking-wide text-indigo-500">
          {t("chat.title")}
        </h3>
        <button
          type="button"
          data-testid="chat-coach-close"
          onClick={() => setOpen(false)}
          className="rounded border border-indigo-300 bg-white px-2 py-1 text-xs text-indigo-700 hover:bg-indigo-100"
        >
          {t("chat.close")}
        </button>
      </div>

      <div
        ref={listRef}
        data-testid="chat-coach-messages"
        className="flex max-h-72 flex-col gap-2 overflow-y-auto"
      >
        {chat.messages.length === 0 && !chat.inFlight && (
          <p className="text-sm text-indigo-400">{t("chat.empty")}</p>
        )}
        {chat.messages.map((m) =>
          m.role === "user" ? (
            <div key={m.id} className="self-end max-w-[85%]">
              <p
                data-testid="chat-msg-user"
                className="whitespace-pre-wrap rounded-lg rounded-br-none bg-white px-3 py-2 text-sm text-slate-800 shadow-sm"
              >
                {m.text}
              </p>
            </div>
          ) : (
            <div key={m.id} className="self-start max-w-[85%]">
              <p
                data-testid="chat-msg-coach"
                className="whitespace-pre-wrap rounded-lg rounded-bl-none bg-indigo-100 px-3 py-2 text-sm text-indigo-900"
              >
                {m.text}
              </p>
              {settings.enabled && (
                <button
                  type="button"
                  data-testid="chat-msg-replay"
                  onClick={() => speech.speak(`chat-replay:${m.id}:${Date.now()}`, m.text, locale)}
                  title={t("coach.replay")}
                  className="mt-1 rounded border border-indigo-300 bg-white px-2 py-0.5 text-xs text-indigo-700 hover:bg-indigo-100"
                >
                  <span aria-hidden>▶</span>
                  <span className="sr-only">{t("coach.replay")}</span>
                </button>
              )}
            </div>
          ),
        )}
        {chat.inFlight && (
          <p data-testid="chat-coach-thinking" className="text-sm text-indigo-500">
            {t("chat.thinking")}
          </p>
        )}
      </div>

      {chat.error && (
        <p data-testid="chat-coach-error" className="text-sm text-red-600">
          {t("chat.error")}
        </p>
      )}
      {voice.error && <p className="text-sm text-red-600">{t(voice.error)}</p>}
      {voice.active && (
        <p className="text-sm text-indigo-500">
          {voice.recording ? t("chat.mic.recording") : t("chat.mic.listening")}
        </p>
      )}

      <div className="flex items-center gap-2">
        <input
          data-testid="chat-coach-input"
          type="text"
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter") submit();
          }}
          placeholder={t("chat.placeholder")}
          className="flex-1 rounded border border-indigo-300 bg-white px-3 py-2 text-sm text-slate-800 focus:border-indigo-500 focus:outline-none"
        />
        <button
          type="button"
          data-testid="chat-coach-mic"
          disabled={voice.available === "none"}
          title={
            voice.available === "none"
              ? t("chat.mic.unavailable")
              : voice.active
                ? t("chat.mic.stop")
                : t("chat.mic.start")
          }
          onClick={() => (voice.active ? voice.stop() : voice.start())}
          className={`rounded border px-3 py-2 text-sm ${
            voice.active
              ? "border-red-400 bg-red-50 text-red-700"
              : "border-indigo-300 bg-white text-indigo-700 hover:bg-indigo-100"
          } disabled:cursor-not-allowed disabled:opacity-40`}
        >
          <span aria-hidden>{voice.active ? "⏹" : "🎤"}</span>
          <span className="sr-only">{voice.active ? t("chat.mic.stop") : t("chat.mic.start")}</span>
        </button>
        <button
          type="button"
          data-testid="chat-coach-send"
          disabled={chat.inFlight || input.trim() === ""}
          onClick={submit}
          className="rounded bg-indigo-600 px-3 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:cursor-not-allowed disabled:opacity-40"
        >
          {t("chat.send")}
        </button>
      </div>
    </div>
  );
}
