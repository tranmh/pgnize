"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { ApiError, getJob, upload, type UploadKind } from "@/lib/api-client";
import { submitImages, type UploadMode } from "@/lib/multi-image";
import { useJobPoller } from "@/hooks/useJobPoller";
import { useAuth } from "@/components/AuthProvider";
import MultiImagePicker from "@/components/MultiImagePicker";
import UploadModeToggle from "@/components/UploadModeToggle";
import RecognizerSelect from "@/components/RecognizerSelect";
import Spinner from "@/components/Spinner";
import UploadJobRow from "./UploadJobRow";
import { useT } from "@/i18n/I18nProvider";

export default function UploadPage() {
  const t = useT();
  const router = useRouter();
  const { user, loading: authLoading } = useAuth();

  const [files, setFiles] = useState<File[]>([]);
  const [consent, setConsent] = useState(false);
  const [backend, setBackend] = useState("");
  const [kind, setKind] = useState<UploadKind>("scoresheet");
  const [mode, setMode] = useState<UploadMode>("combine");
  // A single recognized item auto-redirects to its review screen (jobId); a
  // "separate" submission of several pictures lists them instead (jobIds).
  const [jobId, setJobId] = useState<string | null>(null);
  const [jobIds, setJobIds] = useState<string[] | null>(null);
  const [rejected, setRejected] = useState(0);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!authLoading && !user) router.replace("/login");
  }, [authLoading, user, router]);

  const poll = useJobPoller(jobId, getJob);

  useEffect(() => {
    if (poll.phase === "done" && poll.gameId) {
      // Board positions land in the dedicated position-editor review screen.
      router.replace(
        kind === "position"
          ? `/scan/review/${poll.gameId}`
          : `/review/${poll.gameId}`,
      );
    } else if (poll.phase === "failed" || poll.phase === "timeout") {
      setError(
        poll.phase === "timeout"
          ? t("recog.timeout")
          : (poll.error ?? t("recog.failed")),
      );
      setJobId(null);
    }
  }, [poll.phase, poll.gameId, poll.error, router, t, kind]);

  async function submit() {
    if (files.length === 0) return;
    setSubmitting(true);
    setError(null);
    try {
      const res = await submitImages(files, mode, (images) =>
        upload(
          images,
          consent,
          backend || undefined,
          kind === "position" ? "position" : undefined,
        ),
      );
      // One job → auto-redirect; several → show a review list.
      if (res.jobIds.length === 1) {
        setJobId(res.jobIds[0]);
      } else {
        setRejected(res.rejected);
        setJobIds(res.jobIds);
      }
    } catch (e) {
      setError(
        e instanceof ApiError && e.status === 429
          ? t("upload.errRateLimit")
          : e instanceof Error
            ? e.message
            : t("upload.errGeneric"),
      );
    } finally {
      setSubmitting(false);
    }
  }

  if (authLoading || !user) {
    return (
      <div className="flex justify-center py-16">
        <Spinner label={t("common.loading")} />
      </div>
    );
  }

  const redirecting = !!jobId;
  const listing = !!jobIds;

  return (
    <div className="mx-auto flex max-w-xl flex-col gap-6">
      <div>
        <h1 className="text-2xl font-bold">{t("upload.title")}</h1>
        <p className="mt-1 text-sm text-gray-500">{t("upload.subtitle")}</p>
      </div>

      {redirecting ? (
        <div className="flex flex-col items-center gap-3 rounded-lg border border-gray-200 bg-white py-16">
          <Spinner
            label={
              poll.status === "running"
                ? t("recog.reading")
                : t("recog.queued")
            }
          />
          <p className="text-xs text-gray-400">{t("upload.autoRedirect")}</p>
        </div>
      ) : listing ? (
        <div className="flex flex-col gap-4">
          {rejected > 0 && (
            <p className="text-sm text-amber-700">
              {t("multiupload.someRejected", { n: rejected })}
            </p>
          )}
          <p className="text-sm text-gray-600">
            {t("upload.recognized", { n: jobIds!.length })}
          </p>
          <ul className="flex flex-col gap-3">
            {jobIds!.map((id, i) => (
              <li
                key={id}
                className="flex items-center justify-between gap-3 rounded-lg border border-gray-200 bg-white px-4 py-3"
              >
                <span className="text-sm font-semibold text-gray-500">
                  {t(
                    kind === "position"
                      ? "scan.resultLabel"
                      : "convert.resultLabel",
                    { n: i + 1 },
                  )}
                </span>
                <UploadJobRow jobId={id} kind={kind} />
              </li>
            ))}
          </ul>
        </div>
      ) : (
        <>
          <fieldset className="flex flex-col gap-2">
            <legend className="text-[11px] font-medium uppercase tracking-wide text-gray-500">
              {t("upload.kind.label")}
            </legend>
            <div className="inline-flex overflow-hidden rounded border border-gray-300 text-sm">
              <button
                type="button"
                onClick={() => setKind("scoresheet")}
                disabled={submitting}
                className={`px-3 py-1 ${kind === "scoresheet" ? "bg-blue-600 text-white" : "bg-white text-gray-600 hover:bg-gray-100"}`}
              >
                {t("upload.kind.scoresheet")}
              </button>
              <button
                type="button"
                onClick={() => setKind("position")}
                disabled={submitting}
                className={`px-3 py-1 ${kind === "position" ? "bg-blue-600 text-white" : "bg-white text-gray-600 hover:bg-gray-100"}`}
              >
                {t("upload.kind.scan")}
              </button>
            </div>
          </fieldset>

          <MultiImagePicker files={files} onChange={setFiles} disabled={submitting} />

          {files.length > 1 && (
            <UploadModeToggle
              mode={mode}
              onChange={setMode}
              combineLabel={t(
                kind === "position" ? "scan.modeCombine" : "convert.modeCombine",
              )}
              separateLabel={t(
                kind === "position"
                  ? "scan.modeSeparate"
                  : "convert.modeSeparate",
              )}
              disabled={submitting}
            />
          )}

          <RecognizerSelect
            value={backend}
            onChange={setBackend}
            disabled={submitting}
          />

          <label className="flex items-start gap-2 text-sm text-gray-600">
            <input
              type="checkbox"
              checked={consent}
              onChange={(e) => setConsent(e.target.checked)}
              className="mt-0.5"
            />
            <span>{t("upload.consent")}</span>
          </label>

          {error && <p className="text-sm text-red-600">{error}</p>}

          <button
            type="button"
            disabled={files.length === 0 || submitting}
            onClick={submit}
            className="rounded bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:bg-gray-300"
          >
            {submitting ? t("upload.submitting") : t("upload.submit")}
          </button>
        </>
      )}
    </div>
  );
}
