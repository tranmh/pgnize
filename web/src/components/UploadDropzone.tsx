"use client";

import { useRef, useState } from "react";
import { useT } from "@/i18n/I18nProvider";

export interface UploadDropzoneProps {
  // Called with the chosen image file.
  onFile: (file: File) => void;
  disabled?: boolean;
}

// Drag-and-drop / click-to-pick image dropzone with a thumbnail preview.
export default function UploadDropzone({ onFile, disabled }: UploadDropzoneProps) {
  const t = useT();
  const inputRef = useRef<HTMLInputElement>(null);
  const [over, setOver] = useState(false);
  const [previewUrl, setPreviewUrl] = useState<string | null>(null);
  const [name, setName] = useState<string | null>(null);

  const handle = (file: File | undefined | null) => {
    if (!file) return;
    if (!file.type.startsWith("image/")) return;
    setName(file.name);
    setPreviewUrl((prev) => {
      if (prev) URL.revokeObjectURL(prev);
      return URL.createObjectURL(file);
    });
    onFile(file);
  };

  return (
    <div>
      <button
        type="button"
        disabled={disabled}
        onClick={() => inputRef.current?.click()}
        onDragOver={(e) => {
          e.preventDefault();
          setOver(true);
        }}
        onDragLeave={() => setOver(false)}
        onDrop={(e) => {
          e.preventDefault();
          setOver(false);
          handle(e.dataTransfer.files?.[0]);
        }}
        className={[
          "flex w-full flex-col items-center justify-center gap-3 rounded-lg border-2 border-dashed p-8 text-center transition",
          over ? "border-blue-400 bg-blue-50" : "border-gray-300 bg-gray-50",
          disabled ? "cursor-not-allowed opacity-50" : "hover:border-blue-400 hover:bg-blue-50",
        ].join(" ")}
        aria-label={t("dropzone.aria")}
      >
        {previewUrl ? (
          // eslint-disable-next-line @next/next/no-img-element -- local object URL preview
          <img src={previewUrl} alt={t("dropzone.selectedAlt")} className="max-h-56 rounded shadow" />
        ) : (
          <span className="text-4xl">📷</span>
        )}
        <span className="text-sm text-gray-600">
          {name ? (
            <>
              <strong>{name}</strong> — {t("dropzone.changePhoto")}
            </>
          ) : (
            <>
              {t("dropzone.dragHere")}{" "}
              <span className="font-medium text-blue-600">{t("dropzone.browse")}</span>
            </>
          )}
        </span>
      </button>
      <input
        ref={inputRef}
        type="file"
        accept="image/*"
        className="hidden"
        onChange={(e) => handle(e.target.files?.[0])}
      />
    </div>
  );
}
