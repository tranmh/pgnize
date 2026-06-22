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

interface SpeechSettingsState {
  enabled: boolean;
  source: SpeechSource;
  setEnabled: (v: boolean) => void;
  setSource: (s: SpeechSource) => void;
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

  useEffect(() => {
    const storedEnabled = localStorage.getItem(ENABLED_KEY);
    if (storedEnabled === "true" || storedEnabled === "false") {
      setEnabledState(storedEnabled === "true");
    }
    const storedSource = localStorage.getItem(SOURCE_KEY);
    if (storedSource === "server" || storedSource === "browser") {
      setSourceState(storedSource);
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

  return (
    <SpeechSettingsContext.Provider
      value={{ enabled, source, setEnabled, setSource }}
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
