"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { useSpeechSettings } from "@/i18n/SpeechSettingsProvider";
import { getSpeechController } from "@/lib/tts";

// A speak request is identified by a stable `key` (e.g. the active ply index,
// -2 for a position, -1 for a game summary). The dedupe guard ensures the same
// (key) is not auto-spoken twice across re-renders. We key on identity only —
// when the underlying text for that key changes, callers pass a new key OR
// clear it, so distinct content always speaks.
export function shouldSpeak(
  lastKey: string | number | null,
  key: string | number,
): boolean {
  return lastKey !== key;
}

export interface SpeechApi {
  speaking: boolean;
  // Speak `text` for the logical slot `key`. No-op if disabled or already spoken
  // for this key.
  speak: (key: string | number, text: string, lang: string) => void;
  stop: () => void;
  // Re-speak the last spoken text, ignoring the dedupe guard.
  replay: () => void;
}

export function useSpeech(): SpeechApi {
  const { enabled, source } = useSpeechSettings();
  const [speaking, setSpeaking] = useState(false);
  const lastSpokenKeyRef = useRef<string | number | null>(null);
  const lastSpokenRef = useRef<{ text: string; lang: string } | null>(null);

  const doSpeak = useCallback(
    (text: string, lang: string) => {
      lastSpokenRef.current = { text, lang };
      // Drive `speaking` from the REAL playback lifecycle: true when audio
      // actually starts, false on natural end / stop() / error. The speak()
      // promise resolves as soon as playback STARTS, so we must not key off it.
      void getSpeechController().speak(text, lang, source, {
        onStart: () => setSpeaking(true),
        onEnd: () => setSpeaking(false),
      });
    },
    [source],
  );

  const speak = useCallback(
    (key: string | number, text: string, lang: string) => {
      if (!enabled || !text) return;
      if (!shouldSpeak(lastSpokenKeyRef.current, key)) return;
      lastSpokenKeyRef.current = key;
      doSpeak(text, lang);
    },
    [enabled, doSpeak],
  );

  const stop = useCallback(() => {
    getSpeechController().stop();
    setSpeaking(false);
  }, []);

  const replay = useCallback(() => {
    const last = lastSpokenRef.current;
    if (!last) return;
    doSpeak(last.text, last.lang);
  }, [doSpeak]);

  // Stop any active speech when the consumer unmounts.
  useEffect(() => {
    return () => {
      getSpeechController().stop();
    };
  }, []);

  return { speaking, speak, stop, replay };
}
