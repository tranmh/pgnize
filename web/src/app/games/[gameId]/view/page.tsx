"use client";

import { use, useEffect, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { getGame, type GameDraft } from "@/lib/api-client";
import { useAuth } from "@/components/AuthProvider";
import { useT } from "@/i18n/I18nProvider";
import GameViewer from "@/components/GameViewer";
import Spinner from "@/components/Spinner";

export default function GameViewPage({
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

  if (authLoading || (!user && !loadError) || loading) {
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
        <h1 className="text-2xl font-bold">{t("viewPage.title")}</h1>
        <div className="ml-auto flex items-center gap-3 text-sm">
          <Link href={`/review/${gameId}`} className="text-blue-600 underline">
            {t("viewPage.edit")}
          </Link>
          <Link href="/library" className="text-blue-600 underline">
            {t("common.backToLibrary")}
          </Link>
        </div>
      </div>

      <GameViewer draft={draft} />
    </div>
  );
}
