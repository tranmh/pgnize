"use client";

import { useState } from "react";
import { ApiError, convert } from "@/lib/api-client";
import { submitImages, type UploadMode } from "@/lib/multi-image";
import MultiImagePicker from "@/components/MultiImagePicker";
import UploadModeToggle from "@/components/UploadModeToggle";
import RecognizerSelect from "@/components/RecognizerSelect";
import Link from "next/link";
import AnonymousBanner from "@/components/AnonymousBanner";
import ConvertJobResult from "./ConvertJobResult";
import { useT } from "@/i18n/I18nProvider";

export default function ConvertClient() {
  const t = useT();
  const [backend, setBackend] = useState("");
  const [files, setFiles] = useState<File[]>([]);
  const [mode, setMode] = useState<UploadMode>("combine");
  const [jobIds, setJobIds] = useState<string[] | null>(null);
  const [rejected, setRejected] = useState(0);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function start() {
    setError(null);
    setSubmitting(true);
    try {
      const res = await submitImages(files, mode, (images) =>
        convert(images, backend || undefined),
      );
      setRejected(res.rejected);
      setJobIds(res.jobIds);
    } catch (e) {
      setError(
        e instanceof ApiError && e.status === 429
          ? t("convert.errRateLimit")
          : e instanceof Error
            ? e.message
            : t("convert.errUpload"),
      );
    } finally {
      setSubmitting(false);
    }
  }

  function reset() {
    setJobIds(null);
    setFiles([]);
    setMode("combine");
    setRejected(0);
    setError(null);
  }

  const multiple = (jobIds?.length ?? 0) > 1;

  return (
    <div className="flex flex-col gap-6">
      <div>
        <h1 className="text-2xl font-bold">{t("convert.title")}</h1>
        <p className="mt-1 text-sm text-gray-500">{t("convert.subtitle")}</p>
      </div>

      <AnonymousBanner />

      {jobIds === null ? (
        <div className="flex flex-col gap-4">
          <RecognizerSelect value={backend} onChange={setBackend} />
          <MultiImagePicker files={files} onChange={setFiles} disabled={submitting} />
          {files.length > 1 && (
            <UploadModeToggle
              mode={mode}
              onChange={setMode}
              combineLabel={t("convert.modeCombine")}
              separateLabel={t("convert.modeSeparate")}
              disabled={submitting}
            />
          )}
          {error && (
            <div className="rounded-lg border border-red-300 bg-red-50 p-4">
              <p className="text-sm text-red-600">{error}</p>
            </div>
          )}
          <button
            type="button"
            onClick={start}
            disabled={files.length === 0 || submitting}
            className="self-start rounded bg-blue-600 px-5 py-2 text-sm font-semibold text-white hover:bg-blue-700 disabled:bg-gray-300"
          >
            {t("convert.submit")}
          </button>
          <p className="text-sm text-gray-500">
            {t("scan.promoPrefix")}{" "}
            <Link href="/scan" className="font-medium text-blue-600 underline">
              {t("scan.promoLink")}
            </Link>
          </p>
        </div>
      ) : (
        <div className="flex flex-col gap-6">
          {rejected > 0 && (
            <div className="rounded-lg border border-amber-300 bg-amber-50 p-4">
              <p className="text-sm text-amber-700">
                {t("multiupload.someRejected", { n: rejected })}
              </p>
            </div>
          )}
          {jobIds.map((id, i) => (
            <div key={id} className="flex flex-col gap-2">
              {multiple && (
                <h2 className="text-sm font-semibold text-gray-500">
                  {t("convert.resultLabel", { n: i + 1 })}
                </h2>
              )}
              <ConvertJobResult jobId={id} />
            </div>
          ))}
          <button
            type="button"
            onClick={reset}
            className="self-start rounded border border-gray-300 bg-white px-3 py-1 text-sm text-gray-700 hover:bg-gray-50"
          >
            {t("multiupload.startOver")}
          </button>
        </div>
      )}
    </div>
  );
}
