"use client";

import { useEffect, useRef, useState } from "react";
import type { JobState } from "@/lib/api-client";

const POLL_INTERVAL_MS = 1500;
// ~5 minutes cap, per the contract.
const MAX_DURATION_MS = 5 * 60 * 1000;

export type PollPhase = "idle" | "polling" | "done" | "failed" | "timeout";

export interface JobPollerResult {
  phase: PollPhase;
  status: JobState["status"] | null;
  gameId: string | null;
  error: string | null;
}

// Polls a job endpoint every 1.5s until it reports done/failed or the ~5min cap
// is hit. Pass `null` as jobId to stay idle. `fetchJob` is the endpoint-specific
// getter (getJob or getConvertJob).
export function useJobPoller(
  jobId: string | null,
  fetchJob: (jobId: string) => Promise<JobState>,
): JobPollerResult {
  const [result, setResult] = useState<JobPollerResult>({
    phase: jobId ? "polling" : "idle",
    status: null,
    gameId: null,
    error: null,
  });

  // Keep the latest fetcher without re-subscribing the effect on every render.
  const fetchRef = useRef(fetchJob);
  fetchRef.current = fetchJob;

  useEffect(() => {
    if (!jobId) {
      setResult({ phase: "idle", status: null, gameId: null, error: null });
      return;
    }

    let cancelled = false;
    const startedAt = Date.now();
    setResult({ phase: "polling", status: "queued", gameId: null, error: null });

    let timer: ReturnType<typeof setTimeout>;

    const tick = async () => {
      if (cancelled) return;

      if (Date.now() - startedAt > MAX_DURATION_MS) {
        setResult((r) => ({
          ...r,
          phase: "timeout",
          error: "Timed out waiting for recognition to finish.",
        }));
        return;
      }

      try {
        const state = await fetchRef.current(jobId);
        if (cancelled) return;

        if (state.status === "done") {
          setResult({
            phase: "done",
            status: "done",
            gameId: state.gameId ?? null,
            error: null,
          });
          return;
        }
        if (state.status === "failed") {
          setResult({
            phase: "failed",
            status: "failed",
            gameId: state.gameId ?? null,
            error: state.error ?? "Recognition failed.",
          });
          return;
        }
        // queued | running -> keep polling
        setResult({
          phase: "polling",
          status: state.status,
          gameId: state.gameId ?? null,
          error: null,
        });
        timer = setTimeout(tick, POLL_INTERVAL_MS);
      } catch (e) {
        if (cancelled) return;
        // Transient errors: keep polling until the cap.
        setResult((r) => ({
          ...r,
          phase: "polling",
          error: e instanceof Error ? e.message : "Network error",
        }));
        timer = setTimeout(tick, POLL_INTERVAL_MS);
      }
    };

    // Kick off immediately.
    tick();

    return () => {
      cancelled = true;
      clearTimeout(timer);
    };
  }, [jobId]);

  return result;
}
