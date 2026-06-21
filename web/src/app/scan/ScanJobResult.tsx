"use client";

import { useEffect, useState } from "react";
import {
  exportScanPgn,
  getScanGame,
  getScanJob,
  type GameDraft,
  type Header,
} from "@/lib/api-client";
import { useJobPoller } from "@/hooks/useJobPoller";
import Spinner from "@/components/Spinner";
import PositionReview from "@/components/PositionReview";
import { downloadText, pgnFilename } from "@/lib/download";
import { useT } from "@/i18n/I18nProvider";

// Owns the full lifecycle of ONE position-recognition job: poll → load draft →
// review → export. The parent renders one per job (one in combine mode, a list
// in separate mode), each with its own independent poller.
export default function ScanJobResult({ jobId }: { jobId: string }) {
  const t = useT();
  const [draft, setDraft] = useState<GameDraft | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [exporting, setExporting] = useState(false);

  const poll = useJobPoller(jobId, getScanJob);

  // Load the recognized position once the job reaches "done".
  useEffect(() => {
    if (poll.phase !== "done") return;
    getScanGame(jobId)
      .then(setDraft)
      .catch((e) =>
        setError(e instanceof Error ? e.message : t("scan.errLoadGame")),
      );
  }, [poll.phase, jobId, t]);

  async function handleExport(payload: { header: Header; startFen: string }) {
    setExporting(true);
    try {
      const pgn = await exportScanPgn(jobId, {
        header: payload.header,
        startFen: payload.startFen,
        moves: [],
      });
      downloadText(pgnFilename(payload.header.white, payload.header.black), pgn);
    } catch (e) {
      setError(e instanceof Error ? e.message : t("scan.errExport"));
    } finally {
      setExporting(false);
    }
  }

  if (poll.phase === "failed" || poll.phase === "timeout") {
    return (
      <div className="rounded-lg border border-red-300 bg-red-50 p-6">
        <p className="font-medium text-red-700">{t("scan.errTitle")}</p>
        <p className="mt-1 text-sm text-red-600">
          {poll.phase === "timeout"
            ? t("recog.timeout")
            : (poll.error ?? t("recog.failed"))}
        </p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="rounded-lg border border-red-300 bg-red-50 p-6">
        <p className="font-medium text-red-700">{t("scan.errTitle")}</p>
        <p className="mt-1 text-sm text-red-600">{error}</p>
      </div>
    );
  }

  if (poll.phase !== "done") {
    return (
      <div className="flex flex-col items-center gap-3 rounded-lg border border-gray-200 bg-white py-16">
        <Spinner
          label={poll.status === "running" ? t("recog.reading") : t("recog.queued")}
        />
        <p className="text-xs text-gray-400">{t("scan.takesSeconds")}</p>
      </div>
    );
  }

  if (!draft) {
    return (
      <div className="flex justify-center py-16">
        <Spinner label={t("scan.loadingRecognized")} />
      </div>
    );
  }

  return (
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
  );
}
