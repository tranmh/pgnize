"use client";

import { useEffect, useState } from "react";
import {
  ApiError,
  exportConvertPgn,
  getConvertGame,
  getConvertJob,
  type GameDraft,
} from "@/lib/api-client";
import { useJobPoller } from "@/hooks/useJobPoller";
import Spinner from "@/components/Spinner";
import ReviewWorkbench from "@/components/ReviewWorkbench";
import { downloadText, pgnFilename } from "@/lib/download";
import { useT } from "@/i18n/I18nProvider";

// Owns the full lifecycle of ONE recognition job: poll → load draft → review →
// export. The parent renders one of these per job, so combine mode shows a single
// block and separate mode shows a list — each with its own independent poller.
export default function ConvertJobResult({ jobId }: { jobId: string }) {
  const t = useT();
  const [draft, setDraft] = useState<GameDraft | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [exporting, setExporting] = useState(false);
  const [failedAt, setFailedAt] = useState<number | null>(null);

  const poll = useJobPoller(jobId, getConvertJob);

  // Load the recognized game once the job reaches "done".
  useEffect(() => {
    if (poll.phase !== "done") return;
    getConvertGame(jobId)
      .then(setDraft)
      .catch((e) =>
        setError(e instanceof Error ? e.message : t("convert.errLoadGame")),
      );
  }, [poll.phase, jobId, t]);

  async function handleExport(payload: {
    header: GameDraft["header"];
    moves: { ply: number; san: string; clockSec?: number | null }[];
  }) {
    setExporting(true);
    setFailedAt(null);
    try {
      const pgn = await exportConvertPgn(jobId, {
        header: payload.header,
        moves: payload.moves,
      });
      downloadText(pgnFilename(payload.header.white, payload.header.black), pgn);
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

  if (poll.phase === "failed" || poll.phase === "timeout") {
    return (
      <div className="rounded-lg border border-red-300 bg-red-50 p-6">
        <p className="font-medium text-red-700">{t("convert.errTitle")}</p>
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
        <p className="font-medium text-red-700">{t("convert.errTitle")}</p>
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
        <p className="text-xs text-gray-400">{t("convert.takesMinutes")}</p>
      </div>
    );
  }

  if (!draft) {
    return (
      <div className="flex justify-center py-16">
        <Spinner label={t("convert.loadingRecognized")} />
      </div>
    );
  }

  return (
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
  );
}
