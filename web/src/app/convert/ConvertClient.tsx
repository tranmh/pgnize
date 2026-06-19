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

type Stage = "upload" | "processing" | "review" | "error";

export default function ConvertClient() {
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
          setError(e instanceof Error ? e.message : "Could not load the game.");
        });
    } else if (poll.phase === "failed" || poll.phase === "timeout") {
      setStage("error");
      setError(poll.error ?? "Recognition failed.");
    }
  }, [poll.phase, poll.error, jobId, stage]);

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
          ? "Rate limit reached. Please wait a moment and try again."
          : e instanceof Error
            ? e.message
            : "Upload failed.",
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
        setError(e instanceof Error ? e.message : "Export failed.");
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
        <h1 className="text-2xl font-bold">Convert a score sheet</h1>
        <p className="mt-1 text-sm text-gray-500">
          Upload a photo of a handwritten chess score sheet. We&apos;ll read it,
          and you verify the moves before downloading the PGN.
        </p>
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
                ? "Reading handwriting…"
                : "Queued for recognition…"
            }
          />
          <p className="text-xs text-gray-400">This can take up to a few minutes.</p>
        </div>
      )}

      {stage === "review" && !draft && (
        <div className="flex justify-center py-16">
          <Spinner label="Loading recognized game…" />
        </div>
      )}

      {stage === "review" && draft && (
        <ReviewWorkbench
          draft={draft}
          onPrimary={handleExport}
          primaryLabel="Download PGN"
          serverFailedAt={failedAt}
          saving={exporting}
          footer={
            <p className="text-xs text-gray-400">
              Recognition confidence:{" "}
              {Math.round((draft.confidence ?? 0) * 100)}%
            </p>
          }
        />
      )}

      {stage === "error" && (
        <div className="rounded-lg border border-red-300 bg-red-50 p-6">
          <p className="font-medium text-red-700">Something went wrong</p>
          <p className="mt-1 text-sm text-red-600">{error}</p>
          <button
            type="button"
            onClick={reset}
            className="mt-4 rounded border border-red-300 bg-white px-3 py-1 text-sm text-red-700 hover:bg-red-100"
          >
            Try again
          </button>
        </div>
      )}
    </div>
  );
}

function AnonymousBanner() {
  return (
    <div className="rounded-lg border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-800">
      Anonymous conversions are <strong>not saved</strong> to a library.{" "}
      <Link href="/register" className="font-medium underline">
        Create an account
      </Link>{" "}
      or{" "}
      <Link href="/login" className="font-medium underline">
        log in
      </Link>{" "}
      to keep a searchable history of your games.
    </div>
  );
}
