"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { useT } from "@/i18n/I18nProvider";

export interface UploadDropzoneProps {
  // Called with the chosen image file.
  onFile: (file: File) => void;
  disabled?: boolean;
}

type Mode = "idle" | "camera" | "preview";
type Facing = "environment" | "user";

// Camera-first image picker: opens the device camera in-app (PWA-friendly) with a
// live preview + capture, and keeps drag-and-drop / file picking as a fallback.
export default function UploadDropzone({ onFile, disabled }: UploadDropzoneProps) {
  const t = useT();
  const inputRef = useRef<HTMLInputElement>(null);
  const videoRef = useRef<HTMLVideoElement>(null);
  const streamRef = useRef<MediaStream | null>(null);

  const [mode, setMode] = useState<Mode>("idle");
  const [over, setOver] = useState(false);
  const [previewUrl, setPreviewUrl] = useState<string | null>(null);
  const [name, setName] = useState<string | null>(null);
  const [facing, setFacing] = useState<Facing>("environment");
  const [cameraError, setCameraError] = useState<string | null>(null);
  const [starting, setStarting] = useState(false);

  // getUserMedia needs a secure context; gate the in-app camera UI on availability.
  const cameraSupported =
    typeof navigator !== "undefined" && !!navigator.mediaDevices?.getUserMedia;

  const stopStream = useCallback(() => {
    if (streamRef.current) {
      for (const track of streamRef.current.getTracks()) track.stop();
      streamRef.current = null;
    }
    if (videoRef.current) videoRef.current.srcObject = null;
  }, []);

  const accept = useCallback(
    (file: File | undefined | null) => {
      if (!file) return;
      if (!file.type.startsWith("image/")) return;
      setName(file.name);
      setPreviewUrl((prev) => {
        if (prev) URL.revokeObjectURL(prev);
        return URL.createObjectURL(file);
      });
      setMode("preview");
      onFile(file);
    },
    [onFile],
  );

  const startCamera = useCallback(
    async (which: Facing) => {
      if (!cameraSupported) return;
      setCameraError(null);
      setStarting(true);
      stopStream();
      setMode("camera");
      try {
        const stream = await navigator.mediaDevices.getUserMedia({
          video: { facingMode: { ideal: which } },
          audio: false,
        });
        streamRef.current = stream;
        setFacing(which);
        if (videoRef.current) {
          videoRef.current.srcObject = stream;
          // iOS Safari needs an explicit play() after assigning the stream.
          await videoRef.current.play().catch(() => undefined);
        }
      } catch {
        stopStream();
        setMode("idle");
        setCameraError(t("dropzone.cameraError"));
      } finally {
        setStarting(false);
      }
    },
    [cameraSupported, stopStream, t],
  );

  const capture = useCallback(() => {
    const video = videoRef.current;
    if (!video || !video.videoWidth) return;
    const canvas = document.createElement("canvas");
    canvas.width = video.videoWidth;
    canvas.height = video.videoHeight;
    const ctx = canvas.getContext("2d");
    if (!ctx) return;
    ctx.drawImage(video, 0, 0, canvas.width, canvas.height);
    canvas.toBlob(
      (blob) => {
        if (!blob) return;
        accept(new File([blob], `photo-${Date.now()}.jpg`, { type: "image/jpeg" }));
        stopStream();
      },
      "image/jpeg",
      0.92,
    );
  }, [accept, stopStream]);

  const cancelCamera = useCallback(() => {
    stopStream();
    setMode(previewUrl ? "preview" : "idle");
  }, [stopStream, previewUrl]);

  // Clean up the camera stream and any object URL on unmount.
  useEffect(() => {
    return () => {
      stopStream();
      setPreviewUrl((prev) => {
        if (prev) URL.revokeObjectURL(prev);
        return null;
      });
    };
  }, [stopStream]);

  // ---- Live camera view ----------------------------------------------------
  if (mode === "camera") {
    return (
      <div className="flex flex-col gap-3">
        <div className="relative overflow-hidden rounded-lg border border-gray-300 bg-black">
          <video
            ref={videoRef}
            playsInline
            muted
            autoPlay
            className="h-auto max-h-[60vh] w-full object-contain"
          />
          {starting && (
            <div className="absolute inset-0 flex items-center justify-center text-sm text-white/80">
              {t("dropzone.cameraStarting")}
            </div>
          )}
        </div>
        <div className="flex items-center justify-center gap-2">
          <button
            type="button"
            onClick={capture}
            disabled={starting}
            className="rounded bg-blue-600 px-5 py-2 text-sm font-semibold text-white hover:bg-blue-700 disabled:bg-gray-300"
          >
            {t("dropzone.capture")}
          </button>
          <button
            type="button"
            onClick={() => startCamera(facing === "environment" ? "user" : "environment")}
            disabled={starting}
            className="rounded border border-gray-300 bg-white px-3 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 disabled:opacity-50"
          >
            {t("dropzone.switchCamera")}
          </button>
          <button
            type="button"
            onClick={cancelCamera}
            className="rounded border border-gray-300 bg-white px-3 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50"
          >
            {t("dropzone.cancel")}
          </button>
        </div>
      </div>
    );
  }

  // ---- Idle / preview view -------------------------------------------------
  return (
    <div className="flex flex-col gap-3">
      <button
        type="button"
        disabled={disabled}
        onClick={() => {
          if (cameraSupported) {
            void startCamera(facing);
          } else {
            inputRef.current?.click();
          }
        }}
        onDragOver={(e) => {
          e.preventDefault();
          setOver(true);
        }}
        onDragLeave={() => setOver(false)}
        onDrop={(e) => {
          e.preventDefault();
          setOver(false);
          accept(e.dataTransfer.files?.[0]);
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
              <strong>{name}</strong> — {t("dropzone.retake")}
            </>
          ) : cameraSupported ? (
            <span className="font-medium text-blue-600">{t("dropzone.takePhoto")}</span>
          ) : (
            <>
              {t("dropzone.dragHere")}{" "}
              <span className="font-medium text-blue-600">{t("dropzone.browse")}</span>
            </>
          )}
        </span>
      </button>

      {/* Fallback: choose an existing image from the device. */}
      <button
        type="button"
        disabled={disabled}
        onClick={() => inputRef.current?.click()}
        className="self-center text-xs font-medium text-gray-500 underline hover:text-gray-700 disabled:opacity-50"
      >
        {t("dropzone.orUpload")}
      </button>

      {cameraError && <p className="text-center text-sm text-red-600">{cameraError}</p>}

      <input
        ref={inputRef}
        type="file"
        accept="image/*"
        className="hidden"
        onChange={(e) => accept(e.target.files?.[0])}
      />
    </div>
  );
}
