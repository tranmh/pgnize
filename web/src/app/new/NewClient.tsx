"use client";

import { useState } from "react";
import Link from "next/link";
import {
  ApiError,
  importGames,
  pasteFen,
  saveGame,
  type GameDraft,
  type Header,
  type MoveInput,
} from "@/lib/api-client";
import { useAuth } from "@/components/AuthProvider";
import AnonymousBanner from "@/components/AnonymousBanner";
import ReviewWorkbench from "@/components/ReviewWorkbench";
import { buildPgn } from "@/lib/pgn-export";
import { downloadText, pgnFilename } from "@/lib/download";
import { useT } from "@/i18n/I18nProvider";

type Mode = "fen" | "import";

// /new: paste a FEN or import PGN / a Lichess study|game, then analyze with the
// browser engine and get LLM coaching — bypassing photo recognition entirely.
// Synchronous endpoints (no job polling): await the draft, then render inline.
export default function NewClient() {
  const t = useT();
  const { user } = useAuth();
  const [mode, setMode] = useState<Mode>("fen");
  const [text, setText] = useState("");
  const [drafts, setDrafts] = useState<GameDraft[] | null>(null);
  const [active, setActive] = useState(0);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function start() {
    const value = text.trim();
    if (!value) return;
    setError(null);
    setSubmitting(true);
    try {
      if (mode === "fen") {
        const draft = await pasteFen({ fen: value });
        setDrafts([draft]);
        setActive(0);
      } else {
        const isUrl = /^https?:\/\//i.test(value);
        const res = await importGames(isUrl ? { url: value } : { pgn: value });
        if (!res.games.length) {
          setError(t("new.errEmpty"));
          return;
        }
        setDrafts(res.games);
        setActive(0);
      }
    } catch (e) {
      setError(
        e instanceof ApiError && e.status === 429
          ? t("new.errRateLimit")
          : e instanceof ApiError && e.status === 400
            ? t("new.errInvalid")
            : e instanceof Error
              ? e.message
              : t("new.errGeneric"),
      );
    } finally {
      setSubmitting(false);
    }
  }

  function reset() {
    setDrafts(null);
    setText("");
    setError(null);
    setActive(0);
  }

  if (drafts) {
    const draft = drafts[active];
    return (
      <div className="flex flex-col gap-6">
        <div>
          <h1 className="text-2xl font-bold">{t("new.title")}</h1>
          <p className="mt-1 text-sm text-gray-500">
            {draft.moves.length === 0
              ? t("new.resultSubtitlePosition")
              : t("new.resultSubtitle")}
          </p>
        </div>

        {!user && <AnonymousBanner />}

        {drafts.length > 1 && (
          <div className="flex flex-wrap gap-2">
            {drafts.map((d, i) => (
              <button
                key={(d.id || "g") + i}
                type="button"
                onClick={() => setActive(i)}
                className={`rounded border px-3 py-1 text-sm ${
                  i === active
                    ? "border-blue-600 bg-blue-600 text-white"
                    : "border-gray-300 bg-white text-gray-700 hover:bg-gray-50"
                }`}
              >
                {t("new.gameLabel", { n: i + 1 })}
              </button>
            ))}
          </div>
        )}

        <NewResult
          key={(draft.id || "draft") + ":" + active}
          draft={draft}
          registered={!!user}
        />

        <button
          type="button"
          onClick={reset}
          className="self-start rounded border border-gray-300 bg-white px-3 py-1 text-sm text-gray-700 hover:bg-gray-50"
        >
          {t("new.startOver")}
        </button>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-6">
      <div>
        <h1 className="text-2xl font-bold">{t("new.title")}</h1>
        <p className="mt-1 text-sm text-gray-500">{t("new.subtitle")}</p>
      </div>

      {!user && <AnonymousBanner />}

      <div className="inline-flex self-start overflow-hidden rounded border border-gray-300 text-sm">
        <button
          type="button"
          onClick={() => setMode("fen")}
          className={`px-3 py-1 ${mode === "fen" ? "bg-blue-600 text-white" : "bg-white text-gray-600 hover:bg-gray-100"}`}
        >
          {t("new.modeFen")}
        </button>
        <button
          type="button"
          onClick={() => setMode("import")}
          className={`px-3 py-1 ${mode === "import" ? "bg-blue-600 text-white" : "bg-white text-gray-600 hover:bg-gray-100"}`}
        >
          {t("new.modeImport")}
        </button>
      </div>

      <textarea
        value={text}
        onChange={(e) => setText(e.target.value)}
        rows={mode === "fen" ? 2 : 8}
        spellCheck={false}
        placeholder={mode === "fen" ? t("new.fenPlaceholder") : t("new.importPlaceholder")}
        aria-label={mode === "fen" ? t("new.fenLabel") : t("new.importLabel")}
        className="w-full rounded border border-gray-300 p-3 font-mono text-sm focus:outline-none focus:ring-2 focus:ring-blue-300"
      />

      {error && (
        <div className="rounded-lg border border-red-300 bg-red-50 p-4">
          <p className="text-sm text-red-600">{error}</p>
        </div>
      )}

      <button
        type="button"
        onClick={start}
        disabled={!text.trim() || submitting}
        className="self-start rounded bg-blue-600 px-5 py-2 text-sm font-semibold text-white hover:bg-blue-700 disabled:bg-gray-300"
      >
        {submitting ? t("new.working") : t("new.submit")}
      </button>

      <p className="text-sm text-gray-500">
        {t("new.promoPrefix")}{" "}
        <Link href="/convert" className="font-medium text-blue-600 underline">
          {t("new.promoConvert")}
        </Link>
      </p>
    </div>
  );
}

// One draft's review surface. Registered owners save to the library; anonymous
// users download a PGN built client-side.
function NewResult({
  draft,
  registered,
}: {
  draft: GameDraft;
  registered: boolean;
}) {
  const t = useT();
  const [saving, setSaving] = useState(false);
  const [savedId, setSavedId] = useState<string | null>(null);
  const [failedAt, setFailedAt] = useState<number | null>(null);

  async function onPrimary(payload: {
    header: Header;
    moves: MoveInput[];
    startFen?: string;
  }) {
    setFailedAt(null);
    if (registered && draft.id) {
      setSaving(true);
      try {
        await saveGame(draft.id, payload);
        setSavedId(draft.id);
      } catch (e) {
        if (e instanceof ApiError && e.failedAt != null) setFailedAt(e.failedAt);
        throw e;
      } finally {
        setSaving(false);
      }
      return;
    }
    const pgn = buildPgn(
      payload.header,
      payload.startFen ?? draft.startFen,
      payload.moves.map((m) => m.san),
    );
    downloadText(
      pgnFilename(payload.header.white, payload.header.black),
      pgn,
    );
  }

  return (
    <ReviewWorkbench
      draft={draft}
      onPrimary={onPrimary}
      primaryLabel={registered ? t("new.save") : t("new.downloadPgn")}
      saving={saving}
      serverFailedAt={failedAt}
      footer={
        savedId ? (
          <p className="rounded border border-green-300 bg-green-50 px-3 py-2 text-sm text-green-700">
            {t("new.saved")}{" "}
            <Link
              href={`/games/${savedId}/view`}
              className="font-medium underline"
            >
              {t("new.viewGame")}
            </Link>
          </p>
        ) : undefined
      }
    />
  );
}
