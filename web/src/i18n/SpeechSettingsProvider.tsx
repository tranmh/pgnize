"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useState,
} from "react";
import type { SpeechSource } from "@/lib/tts";

const ENABLED_KEY = "pgnize.tts.enabled";
const SOURCE_KEY = "pgnize.tts.source";
const STT_SOURCE_KEY = "pgnize.stt.source";

// Speech-to-text source for the conversational coach's voice input. "server"
// records audio and uploads it (preferred); "browser" uses the Web Speech API.
export type SttSource = "server" | "browser";

interface SpeechSettingsState {
  enabled: boolean;
  source: SpeechSource;
  sttSource: SttSource;
  setEnabled: (v: boolean) => void;
  setSource: (s: SpeechSource) => void;
  setSttSource: (s: SttSource) => void;
}

const SpeechSettingsContext = createContext<SpeechSettingsState | null>(null);

// Mirrors I18nProvider: always start at the defaults so server and first client
// render agree (no hydration mismatch), then apply any stored preference after
// mount.
export function SpeechSettingsProvider({
  children,
}: {
  children: React.ReactNode;
}) {
  const [enabled, setEnabledState] = useState<boolean>(true);
  const [source, setSourceState] = useState<SpeechSource>("server");
  const [sttSource, setSttSourceState] = useState<SttSource>("server");

  useEffect(() => {
    const storedEnabled = localStorage.getItem(ENABLED_KEY);
    if (storedEnabled === "true" || storedEnabled === "false") {
      setEnabledState(storedEnabled === "true");
    }
    const storedSource = localStorage.getItem(SOURCE_KEY);
    if (storedSource === "server" || storedSource === "browser") {
      setSourceState(storedSource);
    }
    const storedStt = localStorage.getItem(STT_SOURCE_KEY);
    if (storedStt === "server" || storedStt === "browser") {
      setSttSourceState(storedStt);
    }
  }, []);

  const setEnabled = useCallback((v: boolean) => {
    setEnabledState(v);
    try {
      localStorage.setItem(ENABLED_KEY, v ? "true" : "false");
    } catch {
      /* storage may be unavailable; in-memory setting still applies */
    }
  }, []);

  const setSource = useCallback((s: SpeechSource) => {
    setSourceState(s);
    try {
      localStorage.setItem(SOURCE_KEY, s);
    } catch {
      /* storage may be unavailable; in-memory setting still applies */
    }
  }, []);

  const setSttSource = useCallback((s: SttSource) => {
    setSttSourceState(s);
    try {
      localStorage.setItem(STT_SOURCE_KEY, s);
    } catch {
      /* storage may be unavailable; in-memory setting still applies */
    }
  }, []);

  return (
    <SpeechSettingsContext.Provider
      value={{ enabled, source, sttSource, setEnabled, setSource, setSttSource }}
    >
      {children}
    </SpeechSettingsContext.Provider>
  );
}

export function useSpeechSettings(): SpeechSettingsState {
  const ctx = useContext(SpeechSettingsContext);
  if (!ctx)
    throw new Error(
      "useSpeechSettings must be used within SpeechSettingsProvider",
    );
  return ctx;
}
