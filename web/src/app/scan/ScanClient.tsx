"use client";

import { useEffect, useState } from "react";
import {
  ApiError,
  exportScanPgn,
  getScanGame,
  getScanJob,
  scan,
  type GameDraft,
  type Header,
} from "@/lib/api-client";
import { useJobPoller } from "@/hooks/useJobPoller";
import UploadDropzone from "@/components/UploadDropzone";
import RecognizerSelect from "@/components/RecognizerSelect";
import Spinner from "@/components/Spinner";
import PositionReview from "@/components/PositionReview";
import AnonymousBanner from "@/components/AnonymousBanner";
import { downloadText, pgnFilename } from "@/lib/download";
import { useT } from "@/i18n/I18nProvider";

type Stage = "upload" | "processing" | "review" | "error";

export default function ScanClient() {
  const t = useT();
  const [stage, setStage] = useState<Stage>("upload");
  const [jobId, setJobId] = useState<string | null>(null);
  const [draft, setDraft] = useState<GameDraft | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [exporting, setExporting] = useState(false);
  const [backend, setBackend] = useState("");

  const poll = useJobPoller(jobId, getScanJob);

  // React to the poller reaching a terminal state.
  useEffect(() => {
    if (stage !== "processing") return;
    if (poll.phase === "done" && jobId) {
      setStage("review");
      getScanGame(jobId)
        .then(setDraft)
        .catch((e) => {
          setStage("error");
          setError(e instanceof Error ? e.message : t("scan.errLoadGame"));
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
    try {
      const { jobId } = await scan(file, backend || undefined);
      setJobId(jobId);
      setStage("processing");
    } catch (e) {
      setStage("error");
      setError(
        e instanceof ApiError && e.status === 429
          ? t("scan.errRateLimit")
          : e instanceof Error
            ? e.message
            : t("scan.errUpload"),
      );
    }
  }

  async function handleExport(payload: { header: Header; startFen: string }) {
    if (!jobId) return;
    setExporting(true);
    try {
      const pgn = await exportScanPgn(jobId, {
        header: payload.header,
        startFen: payload.startFen,
        moves: [],
      });
      downloadText(
        pgnFilename(payload.header.white, payload.header.black),
        pgn,
      );
    } catch (e) {
      setError(e instanceof Error ? e.message : t("scan.errExport"));
    } finally {
      setExporting(false);
    }
  }

  function reset() {
    setStage("upload");
    setJobId(null);
    setDraft(null);
    setError(null);
  }

  return (
    <div className="flex flex-col gap-6">
      <div>
        <h1 className="text-2xl font-bold">{t("scan.title")}</h1>
        <p className="mt-1 text-sm text-gray-500">{t("scan.subtitle")}</p>
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
          <p className="text-xs text-gray-400">{t("scan.takesSeconds")}</p>
        </div>
      )}

      {stage === "review" && !draft && (
        <div className="flex justify-center py-16">
          <Spinner label={t("scan.loadingRecognized")} />
        </div>
      )}

      {stage === "review" && draft && (
        <PositionReview
          draft={draft}
          onPrimary={handleExport}
          primaryLabel={t("scan.downloadPgn")}
          saving={exporting}
          footer={
            <p className="text-xs text-gray-400">
              {t("scan.confidence", {
                pct: Math.round((draft.confidence ?? 0) * 100),
              })}
            </p>
          }
        />
      )}

      {stage === "error" && (
        <div className="rounded-lg border border-red-300 bg-red-50 p-6">
          <p className="font-medium text-red-700">{t("scan.errTitle")}</p>
          <p className="mt-1 text-sm text-red-600">{error}</p>
          <button
            type="button"
            onClick={reset}
            className="mt-4 rounded border border-red-300 bg-white px-3 py-1 text-sm text-red-700 hover:bg-red-100"
          >
            {t("scan.tryAgain")}
          </button>
        </div>
      )}
    </div>
  );
}
