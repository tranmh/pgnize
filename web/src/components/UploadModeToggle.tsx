"use client";

import { useT } from "@/i18n/I18nProvider";

// How a multi-picture submission is turned into results.
//  - "combine":  all pictures → one job → one combined game/position
//  - "separate": one job per picture → one result each
export type UploadMode = "combine" | "separate";

export interface UploadModeToggleProps {
  mode: UploadMode;
  onChange: (mode: UploadMode) => void;
  // Tool-specific labels (e.g. "One game" vs "One position").
  combineLabel: string;
  separateLabel: string;
  disabled?: boolean;
}

// A small radio group letting the user say whether the pictures belong to one
// game/position (extra pages or angles) or are separate items. Only meaningful
// with more than one picture, so callers render it conditionally.
export default function UploadModeToggle({
  mode,
  onChange,
  combineLabel,
  separateLabel,
  disabled,
}: UploadModeToggleProps) {
  const t = useT();
  return (
    <fieldset className="flex flex-col gap-2" disabled={disabled}>
      <legend className="text-sm font-medium text-gray-700">
        {t("multiupload.modePrompt")}
      </legend>
      <div className="flex flex-col gap-2 sm:flex-row sm:gap-4">
        {(
          [
            ["combine", combineLabel],
            ["separate", separateLabel],
          ] as const
        ).map(([value, label]) => (
          <label
            key={value}
            className="inline-flex cursor-pointer items-center gap-2 text-sm text-gray-700"
          >
            <input
              type="radio"
              name="upload-mode"
              value={value}
              checked={mode === value}
              onChange={() => onChange(value)}
              className="h-4 w-4 text-blue-600"
            />
            {label}
          </label>
        ))}
      </div>
    </fieldset>
  );
}
