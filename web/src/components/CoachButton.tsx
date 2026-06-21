"use client";

import { useT } from "@/i18n/I18nProvider";

// "Coach this game" toolbar button. Enabled once the engine has produced
// annotations (the coach needs the evals); shows a busy label while thinking.
export default function CoachButton({
  hasAnnotations,
  loading,
  onClick,
}: {
  hasAnnotations: boolean;
  loading: boolean;
  onClick: () => void;
}) {
  const t = useT();
  if (!hasAnnotations) return null;
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={loading}
      className="rounded border border-indigo-300 bg-indigo-50 px-2.5 py-1 text-xs font-medium text-indigo-700 hover:bg-indigo-100 disabled:opacity-50"
    >
      {loading ? t("coach.thinking") : t("coach.coachGame")}
    </button>
  );
}
