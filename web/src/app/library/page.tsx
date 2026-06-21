"use client";

import { useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import {
  createManualGame,
  exportGamesBundle,
  getGamePgn,
  listGames,
  type GameSummary,
  type ListGamesParams,
} from "@/lib/api-client";
import { useAuth } from "@/components/AuthProvider";
import { useT } from "@/i18n/I18nProvider";
import Spinner from "@/components/Spinner";
import { downloadText, pgnFilename } from "@/lib/download";

const PAGE_SIZE = 20;

export default function LibraryPage() {
  const t = useT();
  const router = useRouter();
  const { user, loading: authLoading } = useAuth();

  const [filters, setFilters] = useState<ListGamesParams>({ page: 1 });
  const [games, setGames] = useState<GameSummary[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selected, setSelected] = useState<Set<string>>(new Set());

  // Redirect anonymous users to login.
  useEffect(() => {
    if (!authLoading && !user) router.replace("/login");
  }, [authLoading, user, router]);

  const load = useCallback(async (params: ListGamesParams) => {
    setLoading(true);
    setError(null);
    try {
      const res = await listGames({ ...params, pageSize: PAGE_SIZE });
      setGames(res.games);
      setTotal(res.total);
      setPage(res.page);
    } catch (e) {
      setError(e instanceof Error ? e.message : t("library.errLoad"));
    } finally {
      setLoading(false);
    }
  }, [t]);

  useEffect(() => {
    if (user) void load(filters);
  }, [user, filters, load]);

  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));

  const toggleSelect = (id: string) =>
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });

  async function downloadOne(id: string, white: string, black: string) {
    try {
      const pgn = await getGamePgn(id);
      downloadText(pgnFilename(white, black), pgn);
    } catch (e) {
      setError(e instanceof Error ? e.message : t("library.errDownload"));
    }
  }

  async function downloadBundle() {
    if (selected.size === 0) return;
    try {
      const pgn = await exportGamesBundle([...selected]);
      downloadText(`pgnize_bundle_${selected.size}_games.pgn`, pgn);
    } catch (e) {
      setError(e instanceof Error ? e.message : t("library.errBundle"));
    }
  }

  async function newManual() {
    try {
      const { game } = await createManualGame();
      router.push(`/review/${game.id}`);
    } catch (e) {
      setError(e instanceof Error ? e.message : t("library.errDraft"));
    }
  }

  if (authLoading || !user) {
    return (
      <div className="flex justify-center py-16">
        <Spinner label={t("common.loading")} />
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-5">
      <div className="flex flex-wrap items-center gap-3">
        <h1 className="text-2xl font-bold">{t("library.title")}</h1>
        <div className="ml-auto flex gap-2">
          <button
            type="button"
            onClick={() => router.push("/upload")}
            className="rounded bg-blue-600 px-3 py-2 text-sm font-medium text-white hover:bg-blue-700"
          >
            {t("library.newFromPhoto")}
          </button>
          <button
            type="button"
            onClick={newManual}
            className="rounded border border-gray-300 px-3 py-2 text-sm hover:bg-gray-100"
          >
            {t("library.enterManually")}
          </button>
        </div>
      </div>

      <SearchFilters
        onApply={(f) => setFilters({ ...f, page: 1 })}
      />

      {selected.size > 0 && (
        <div className="flex items-center gap-3 rounded border border-blue-200 bg-blue-50 px-3 py-2 text-sm">
          <span>{t("library.selected", { n: selected.size })}</span>
          <button
            type="button"
            onClick={downloadBundle}
            className="rounded bg-blue-600 px-3 py-1 text-white hover:bg-blue-700"
          >
            {t("library.downloadBundle")}
          </button>
          <button
            type="button"
            onClick={() => setSelected(new Set())}
            className="text-gray-500 underline"
          >
            {t("library.clear")}
          </button>
        </div>
      )}

      {error && (
        <p className="rounded border border-red-300 bg-red-50 px-3 py-2 text-sm text-red-700">
          {error}
        </p>
      )}

      {loading ? (
        <div className="flex justify-center py-16">
          <Spinner label={t("library.loadingGames")} />
        </div>
      ) : games.length === 0 ? (
        <div className="rounded-lg border border-dashed border-gray-300 bg-white py-16 text-center text-gray-500">
          {t("library.empty")}
        </div>
      ) : (
        <div className="overflow-x-auto rounded-lg border border-gray-200 bg-white">
          <table className="w-full text-sm">
            <thead className="border-b border-gray-200 text-left text-xs uppercase tracking-wide text-gray-500">
              <tr>
                <th className="w-8 px-3 py-2" />
                <th className="px-3 py-2">{t("library.colWhite")}</th>
                <th className="px-3 py-2">{t("library.colBlack")}</th>
                <th className="px-3 py-2">{t("library.colEvent")}</th>
                <th className="px-3 py-2">{t("library.colDate")}</th>
                <th className="px-3 py-2">{t("library.colResult")}</th>
                <th className="px-3 py-2">{t("library.colMoves")}</th>
                <th className="px-3 py-2 text-right">{t("library.colActions")}</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100">
              {games.map((g) => (
                <tr key={g.id} className="hover:bg-gray-50">
                  <td className="px-3 py-2">
                    <input
                      type="checkbox"
                      checked={selected.has(g.id)}
                      onChange={() => toggleSelect(g.id)}
                      aria-label={t("library.selectAria", { white: g.white, black: g.black })}
                    />
                  </td>
                  <td className="px-3 py-2 font-medium">{g.white || "—"}</td>
                  <td className="px-3 py-2 font-medium">{g.black || "—"}</td>
                  <td className="px-3 py-2 text-gray-600">{g.event || "—"}</td>
                  <td className="px-3 py-2 text-gray-600">{g.date || "—"}</td>
                  <td className="px-3 py-2 font-mono">{g.result}</td>
                  <td className="px-3 py-2 text-gray-600">{g.moveCount}</td>
                  <td className="px-3 py-2">
                    <div className="flex justify-end gap-2">
                      <button
                        type="button"
                        onClick={() => router.push(`/games/${g.id}/view`)}
                        className="rounded border border-gray-300 px-2 py-1 text-xs hover:bg-gray-100"
                      >
                        {t("library.view")}
                      </button>
                      <button
                        type="button"
                        onClick={() => router.push(`/review/${g.id}`)}
                        className="rounded border border-gray-300 px-2 py-1 text-xs hover:bg-gray-100"
                      >
                        {t("library.open")}
                      </button>
                      <button
                        type="button"
                        onClick={() => downloadOne(g.id, g.white, g.black)}
                        className="rounded border border-gray-300 px-2 py-1 text-xs hover:bg-gray-100"
                      >
                        {t("library.pgn")}
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {totalPages > 1 && (
        <div className="flex items-center justify-center gap-3 text-sm">
          <button
            type="button"
            disabled={page <= 1}
            onClick={() => setFilters((f) => ({ ...f, page: page - 1 }))}
            className="rounded border border-gray-300 px-3 py-1 disabled:opacity-40"
          >
            {t("library.previous")}
          </button>
          <span className="text-gray-500">
            {t("library.pageOf", { page, total: totalPages })}
          </span>
          <button
            type="button"
            disabled={page >= totalPages}
            onClick={() => setFilters((f) => ({ ...f, page: page + 1 }))}
            className="rounded border border-gray-300 px-3 py-1 disabled:opacity-40"
          >
            {t("library.next")}
          </button>
        </div>
      )}
    </div>
  );
}

function SearchFilters({
  onApply,
}: {
  onApply: (f: ListGamesParams) => void;
}) {
  const t = useT();
  const [q, setQ] = useState("");
  const [player, setPlayer] = useState("");
  const [event, setEvent] = useState("");
  const [from, setFrom] = useState("");
  const [to, setTo] = useState("");

  return (
    <form
      onSubmit={(e) => {
        e.preventDefault();
        onApply({
          q: q || undefined,
          player: player || undefined,
          event: event || undefined,
          from: from || undefined,
          to: to || undefined,
        });
      }}
      className="grid grid-cols-2 gap-3 rounded-lg border border-gray-200 bg-white p-3 md:grid-cols-6"
    >
      <input
        value={q}
        onChange={(e) => setQ(e.target.value)}
        placeholder={t("library.searchPlaceholder")}
        aria-label={t("library.searchAria")}
        className="col-span-2 rounded border border-gray-300 px-2 py-1 text-sm md:col-span-2"
      />
      <input
        value={player}
        onChange={(e) => setPlayer(e.target.value)}
        placeholder={t("library.playerPlaceholder")}
        aria-label={t("library.playerAria")}
        className="rounded border border-gray-300 px-2 py-1 text-sm"
      />
      <input
        value={event}
        onChange={(e) => setEvent(e.target.value)}
        placeholder={t("library.eventPlaceholder")}
        aria-label={t("library.eventAria")}
        className="rounded border border-gray-300 px-2 py-1 text-sm"
      />
      <input
        value={from}
        onChange={(e) => setFrom(e.target.value)}
        placeholder={t("library.fromPlaceholder")}
        aria-label={t("library.fromAria")}
        className="rounded border border-gray-300 px-2 py-1 text-sm"
      />
      <input
        value={to}
        onChange={(e) => setTo(e.target.value)}
        placeholder={t("library.toPlaceholder")}
        aria-label={t("library.toAria")}
        className="rounded border border-gray-300 px-2 py-1 text-sm"
      />
      <button
        type="submit"
        className="col-span-2 rounded bg-gray-800 px-3 py-1 text-sm text-white hover:bg-gray-900 md:col-span-6 md:w-32"
      >
        {t("library.apply")}
      </button>
    </form>
  );
}
