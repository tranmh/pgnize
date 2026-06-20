"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useState,
} from "react";
import {
  DEFAULT_LOCALE,
  messages,
  type Locale,
} from "./messages";

const STORAGE_KEY = "pgnize.locale";

type Vars = Record<string, string | number>;

interface I18nState {
  locale: Locale;
  setLocale: (l: Locale) => void;
  t: (key: string, vars?: Vars) => string;
}

const I18nContext = createContext<I18nState | null>(null);

function interpolate(template: string, vars?: Vars): string {
  if (!vars) return template;
  return template.replace(/\{(\w+)\}/g, (m, k) =>
    k in vars ? String(vars[k]) : m,
  );
}

export function I18nProvider({ children }: { children: React.ReactNode }) {
  // Always start at the default so server and first client render agree (no
  // hydration mismatch); a stored preference is applied after mount.
  const [locale, setLocaleState] = useState<Locale>(DEFAULT_LOCALE);

  useEffect(() => {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (stored === "de" || stored === "en") setLocaleState(stored);
  }, []);

  useEffect(() => {
    document.documentElement.lang = locale;
  }, [locale]);

  const setLocale = useCallback((l: Locale) => {
    setLocaleState(l);
    try {
      localStorage.setItem(STORAGE_KEY, l);
    } catch {
      /* storage may be unavailable; in-memory locale still applies */
    }
  }, []);

  const t = useCallback(
    (key: string, vars?: Vars) => {
      const value =
        messages[locale]?.[key] ?? messages[DEFAULT_LOCALE][key] ?? key;
      return interpolate(value, vars);
    },
    [locale],
  );

  return (
    <I18nContext.Provider value={{ locale, setLocale, t }}>
      {children}
    </I18nContext.Provider>
  );
}

export function useI18n(): I18nState {
  const ctx = useContext(I18nContext);
  if (!ctx) throw new Error("useI18n must be used within I18nProvider");
  return ctx;
}

// Convenience hook for components that only need the translate function.
export function useT(): I18nState["t"] {
  return useI18n().t;
}
