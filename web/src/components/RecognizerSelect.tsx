"use client";

import { useEffect, useState } from "react";
import { fetchRecognizers, type RecognizerInfo } from "@/lib/api-client";
import { useT } from "@/i18n/I18nProvider";

// RecognizerSelect lets the user pick which engine reads the score sheet. It self-loads
// the advertised backends and renders nothing when only one (or none) is available, so the
// control disappears in single-backend deployments (e.g. CI's fake recognizer).
export default function RecognizerSelect({
  value,
  onChange,
  disabled,
}: {
  value: string;
  onChange: (key: string) => void;
  disabled?: boolean;
}) {
  const t = useT();
  const [options, setOptions] = useState<RecognizerInfo[]>([]);

  useEffect(() => {
    let active = true;
    fetchRecognizers()
      .then((r) => {
        if (!active) return;
        setOptions(r.recognizers);
        if (!value) {
          const def =
            r.recognizers.find((o) => o.default) ?? r.recognizers[0];
          if (def) onChange(def.key);
        }
      })
      .catch(() => {
        /* leave the picker hidden; the server default still applies */
      });
    return () => {
      active = false;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  if (options.length <= 1) return null;

  return (
    <label className="flex flex-col gap-1 text-sm">
      <span className="font-medium text-gray-700">{t("recognizer.label")}</span>
      <select
        value={value}
        disabled={disabled}
        onChange={(e) => onChange(e.target.value)}
        className="rounded border border-gray-300 px-2 py-1.5 text-sm disabled:bg-gray-100"
      >
        {options.map((o) => (
          <option key={o.key} value={o.key}>
            {o.label}
          </option>
        ))}
      </select>
    </label>
  );
}
