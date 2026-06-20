"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { ApiError, getJob, upload } from "@/lib/api-client";
import { useJobPoller } from "@/hooks/useJobPoller";
import { useAuth } from "@/components/AuthProvider";
import UploadDropzone from "@/components/UploadDropzone";
import RecognizerSelect from "@/components/RecognizerSelect";
import Spinner from "@/components/Spinner";
import { useT } from "@/i18n/I18nProvider";

export default function UploadPage() {
  const t = useT();
  const router = useRouter();
  const { user, loading: authLoading } = useAuth();

  const [file, setFile] = useState<File | null>(null);
  const [consent, setConsent] = useState(false);
  const [backend, setBackend] = useState("");
  const [jobId, setJobId] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!authLoading && !user) router.replace("/login");
  }, [authLoading, user, router]);

  const poll = useJobPoller(jobId, getJob);

  useEffect(() => {
    if (poll.phase === "done" && poll.gameId) {
      router.replace(`/review/${poll.gameId}`);
    } else if (poll.phase === "failed" || poll.phase === "timeout") {
      setError(
        poll.phase === "timeout"
          ? t("recog.timeout")
          : (poll.error ?? t("recog.failed")),
      );
      setJobId(null);
    }
  }, [poll.phase, poll.gameId, poll.error, router, t]);

  async function submit() {
    if (!file) return;
    setSubmitting(true);
    setError(null);
    try {
      const { jobId } = await upload(file, consent, backend || undefined);
      setJobId(jobId);
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

  const processing = !!jobId;

  return (
    <div className="mx-auto flex max-w-xl flex-col gap-6">
      <div>
        <h1 className="text-2xl font-bold">{t("upload.title")}</h1>
        <p className="mt-1 text-sm text-gray-500">{t("upload.subtitle")}</p>
      </div>

      {processing ? (
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
      ) : (
        <>
          <UploadDropzone onFile={setFile} disabled={submitting} />

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
            disabled={!file || submitting}
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
