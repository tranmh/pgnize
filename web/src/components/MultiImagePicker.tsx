"use client";

import { useEffect, useMemo } from "react";
import UploadDropzone from "@/components/UploadDropzone";
import { addImage, removeImageAt } from "@/lib/multi-image";
import { useT } from "@/i18n/I18nProvider";

export interface MultiImagePickerProps {
  files: File[];
  onChange: (files: File[]) => void;
  disabled?: boolean;
}

// Lets the user collect one or more images (pages / angles) before submitting
// them together. Shows a thumbnail strip with per-image removal, and reuses
// UploadDropzone (in reset-after-pick mode) as the "add another" control.
export default function MultiImagePicker({
  files,
  onChange,
  disabled,
}: MultiImagePickerProps) {
  const t = useT();

  // One object URL per file, recomputed whenever the file list changes. The
  // cleanup effect revokes exactly the URLs created here, so we never leak and
  // never revoke a URL still in use.
  const urls = useMemo(() => files.map((f) => URL.createObjectURL(f)), [files]);
  useEffect(() => {
    return () => {
      for (const url of urls) URL.revokeObjectURL(url);
    };
  }, [urls]);

  return (
    <div className="flex flex-col gap-3">
      {files.length > 0 && (
        <ul className="grid grid-cols-3 gap-3 sm:grid-cols-4">
          {files.map((file, idx) => (
            <li
              key={`${file.name}-${idx}`}
              className="relative overflow-hidden rounded-lg border border-gray-300 bg-gray-50"
            >
              {/* eslint-disable-next-line @next/next/no-img-element -- local object URL preview */}
              <img
                src={urls[idx]}
                alt={t("multiupload.imageLabel", { n: idx + 1 })}
                className="h-28 w-full object-cover"
              />
              <span className="absolute bottom-1 left-1 rounded bg-black/60 px-1.5 py-0.5 text-xs font-medium text-white">
                {t("multiupload.imageLabel", { n: idx + 1 })}
              </span>
              <button
                type="button"
                disabled={disabled}
                onClick={() => onChange(removeImageAt(files, idx))}
                aria-label={t("multiupload.remove")}
                title={t("multiupload.remove")}
                className="absolute right-1 top-1 flex h-6 w-6 items-center justify-center rounded-full bg-black/60 text-sm font-bold text-white hover:bg-black/80 disabled:opacity-50"
              >
                ×
              </button>
            </li>
          ))}
        </ul>
      )}

      <UploadDropzone
        resetAfterPick
        onFile={(f) => onChange(addImage(files, f))}
        disabled={disabled}
      />

      <p className="text-xs text-gray-500">{t("multiupload.hint")}</p>
    </div>
  );
}
