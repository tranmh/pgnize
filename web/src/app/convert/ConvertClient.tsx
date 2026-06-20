"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import {
  ApiError,
  convert,
  exportConvertPgn,
  getConvertGame,
  getConvertJob,
  type GameDraft,
} from "@/lib/api-client";
import { useJobPoller } from "@/hooks/useJobPoller";
import UploadDropzone from "@/components/UploadDropzone";
import RecognizerSelect from "@/components/RecognizerSelect";
import Spinner from "@/components/Spinner";
import ReviewWorkbench from "@/components/ReviewWorkbench";
import { downloadText, pgnFilename } from "@/lib/download";
import { useT } from "@/i18n/I18nProvider";

type Stage = "upload" | "processing" | "review" | "error";

export default function ConvertClient() {
  const t = useT();
  const [stage, setStage] = useState<Stage>("upload");
  const [jobId, setJobId] = useState<string | null>(null);
  const [draft, setDraft] = useState<GameDraft | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [exporting, setExporting] = useState(false);
  const [failedAt, setFailedAt] = useState<number | null>(null);
  const [backend, setBackend] = useState("");

  const poll = useJobPoller(jobId, getConvertJob);

  // React to the poller reaching a terminal state.
  useEffect(() => {
    if (stage !== "processing") return;
    if (poll.phase === "done" && jobId) {
      setStage("review");
      getConvertGame(jobId)
        .then(setDraft)
        .catch((e) => {
          setStage("error");
          setError(e instanceof Error ? e.message : t("convert.errLoadGame"));
        });
    } else if (poll.phase === "failed" || poll.phase === "timeout") {
      setStage("error");
      setError(
        poll.phase === "timeout"
          ? t("recog.timeout")
          : (poll.error ?? t("recog.failed")),
      );
    }
  }, [poll.phase, poll.error, jobId, stage, t]);

  async function start(file: File) {
    setError(null);
    setFailedAt(null);
    try {
      const { jobId } = await convert(file, backend || undefined);
      setJobId(jobId);
      setStage("processing");
    } catch (e) {
      setStage("error");
      setError(
        e instanceof ApiError && e.status === 429
          ? t("convert.errRateLimit")
          : e instanceof Error
            ? e.message
            : t("convert.errUpload"),
      );
    }
  }

  async function handleExport(payload: {
    header: GameDraft["header"];
    moves: { ply: number; san: string; clockSec?: number | null }[];
  }) {
    if (!jobId) return;
    setExporting(true);
    setFailedAt(null);
    try {
      const pgn = await exportConvertPgn(jobId, {
        header: payload.header,
        moves: payload.moves,
      });
      downloadText(
        pgnFilename(payload.header.white, payload.header.black),
        pgn,
      );
    } catch (e) {
      if (e instanceof ApiError && e.code === "illegal_move") {
        setFailedAt(e.failedAt ?? null);
      } else {
        setError(e instanceof Error ? e.message : t("convert.errExport"));
      }
    } finally {
      setExporting(false);
    }
  }

  function reset() {
    setStage("upload");
    setJobId(null);
    setDraft(null);
    setError(null);
    setFailedAt(null);
  }

  return (
    <div className="flex flex-col gap-6">
      <div>
        <h1 className="text-2xl font-bold">{t("convert.title")}</h1>
        <p className="mt-1 text-sm text-gray-500">{t("convert.subtitle")}</p>
      </div>

      <AnonymousBanner />

      {stage === "upload" && (
        <div className="flex flex-col gap-4">
          <RecognizerSelect value={backend} onChange={setBackend} />
          <UploadDropzone onFile={start} />
        </div>
      )}

      {stage === "processing" && (
        <div className="flex flex-col items-center gap-3 rounded-lg border border-gray-200 bg-white py-16">
          <Spinner
            label={
              poll.status === "running"
                ? t("recog.reading")
                : t("recog.queued")
            }
          />
          <p className="text-xs text-gray-400">{t("convert.takesMinutes")}</p>
        </div>
      )}

      {stage === "review" && !draft && (
        <div className="flex justify-center py-16">
          <Spinner label={t("convert.loadingRecognized")} />
        </div>
      )}

      {stage === "review" && draft && (
        <ReviewWorkbench
          draft={draft}
          onPrimary={handleExport}
          primaryLabel={t("convert.downloadPgn")}
          serverFailedAt={failedAt}
          saving={exporting}
          footer={
            <p className="text-xs text-gray-400">
              {t("convert.confidence", {
                pct: Math.round((draft.confidence ?? 0) * 100),
              })}
            </p>
          }
        />
      )}

      {stage === "error" && (
        <div className="rounded-lg border border-red-300 bg-red-50 p-6">
          <p className="font-medium text-red-700">{t("convert.errTitle")}</p>
          <p className="mt-1 text-sm text-red-600">{error}</p>
          <button
            type="button"
            onClick={reset}
            className="mt-4 rounded border border-red-300 bg-white px-3 py-1 text-sm text-red-700 hover:bg-red-100"
          >
            {t("convert.tryAgain")}
          </button>
        </div>
      )}
    </div>
  );
}

function AnonymousBanner() {
  const t = useT();
  return (
    <div className="rounded-lg border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-800">
      {t("anon.prefix")} <strong>{t("anon.notSaved")}</strong>{" "}
      {t("anon.middle")}{" "}
      <Link href="/register" className="font-medium underline">
        {t("anon.createAccount")}
      </Link>{" "}
      {t("anon.or")}{" "}
      <Link href="/login" className="font-medium underline">
        {t("anon.login")}
      </Link>
      {t("anon.suffix")}
    </div>
  );
}
