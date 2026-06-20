"use client";

import { useI18n } from "./I18nProvider";
import { LOCALES, LOCALE_LABELS } from "./messages";

// A compact DE/EN segmented toggle for the nav. German is the default; the choice
// persists to localStorage via the provider.
export default function LanguageSwitcher() {
  const { locale, setLocale } = useI18n();

  return (
    <div
      className="inline-flex overflow-hidden rounded border border-gray-300 text-xs"
      role="group"
      aria-label="Language"
    >
      {LOCALES.map((l) => (
        <button
          key={l}
          type="button"
          onClick={() => setLocale(l)}
          aria-pressed={locale === l}
          className={`px-2 py-1 font-medium ${
            locale === l
              ? "bg-blue-600 text-white"
              : "bg-white text-gray-600 hover:bg-gray-100"
          }`}
        >
          {LOCALE_LABELS[l]}
        </button>
      ))}
    </div>
  );
}
