"use client";

import { use, useEffect, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import {
  ApiError,
  getGame,
  saveGame,
  type GameDraft,
  type Header,
  type MoveInput,
} from "@/lib/api-client";
import { useAuth } from "@/components/AuthProvider";
import { useT } from "@/i18n/I18nProvider";
import ReviewWorkbench from "@/components/ReviewWorkbench";
import Spinner from "@/components/Spinner";

export default function ReviewPage({
  params,
}: {
  params: Promise<{ gameId: string }>;
}) {
  const t = useT();
  const { gameId } = use(params);
  const router = useRouter();
  const { user, loading: authLoading } = useAuth();

  const [draft, setDraft] = useState<GameDraft | null>(null);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);

  const [saving, setSaving] = useState(false);
  const [failedAt, setFailedAt] = useState<number | null>(null);
  const [saveError, setSaveError] = useState<string | null>(null);
  const [saved, setSaved] = useState(false);

  useEffect(() => {
    if (!authLoading && !user) router.replace("/login");
  }, [authLoading, user, router]);

  useEffect(() => {
    if (!user) return;
    let cancelled = false;
    setLoading(true);
    getGame(gameId)
      .then((g) => {
        if (!cancelled) setDraft(g);
      })
      .catch((e) => {
        if (!cancelled)
          setLoadError(e instanceof Error ? e.message : t("reviewPage.loadError"));
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [gameId, user]);

  async function handleSave(payload: {
    header: Header;
    moves: MoveInput[];
    startFen?: string;
  }) {
    setSaving(true);
    setFailedAt(null);
    setSaveError(null);
    setSaved(false);
    try {
      const { game } = await saveGame(gameId, payload);
      setDraft(game);
      setSaved(true);
    } catch (e) {
      if (e instanceof ApiError && e.code === "illegal_move") {
        setFailedAt(e.failedAt ?? null);
      } else {
        setSaveError(e instanceof Error ? e.message : t("reviewPage.saveFailed"));
      }
    } finally {
      setSaving(false);
    }
  }

  if (authLoading || (!user && !loadError)) {
    return (
      <div className="flex justify-center py-16">
        <Spinner label={t("common.loading")} />
      </div>
    );
  }

  if (loading) {
    return (
      <div className="flex justify-center py-16">
        <Spinner label={t("common.loadingGame")} />
      </div>
    );
  }

  if (loadError || !draft) {
    return (
      <div className="rounded-lg border border-red-300 bg-red-50 p-6">
        <p className="font-medium text-red-700">{t("reviewPage.couldNotLoad")}</p>
        <p className="mt-1 text-sm text-red-600">{loadError}</p>
        <Link
          href="/library"
          className="mt-4 inline-block rounded border border-red-300 bg-white px-3 py-1 text-sm text-red-700 hover:bg-red-100"
        >
          {t("common.backToLibrary")}
        </Link>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center gap-3">
        <h1 className="text-2xl font-bold">{t("reviewPage.title")}</h1>
        <span className="rounded-full bg-gray-200 px-2 py-0.5 text-xs uppercase text-gray-600">
          {draft.status}
        </span>
        <Link href="/library" className="ml-auto text-sm text-blue-600 underline">
          {t("common.backToLibrary")}
        </Link>
      </div>

      <ReviewWorkbench
        draft={draft}
        onPrimary={handleSave}
        primaryLabel={t("reviewPage.saveGame")}
        serverFailedAt={failedAt}
        saving={saving}
        footer={
          <div className="flex flex-col gap-1">
            {saveError && (
              <p className="rounded border border-red-300 bg-red-50 px-3 py-2 text-sm text-red-700">
                {saveError}
              </p>
            )}
            {saved && (
              <p className="rounded border border-green-300 bg-green-50 px-3 py-2 text-sm text-green-700">
                {t("reviewPage.saved")}{" "}
                <Link
                  href={`/games/${gameId}/view`}
                  className="font-medium underline"
                >
                  {t("reviewPage.viewGame")}
                </Link>{" "}
                {t("anon.or")}{" "}
                <Link href="/library" className="font-medium underline">
                  {t("reviewPage.goToLibrary")}
                </Link>
                .
              </p>
            )}
          </div>
        }
      />
    </div>
  );
}
