"use client";

import { useT } from "@/i18n/I18nProvider";

// Coach toolbar button ("Coach this game" / "Coach this position"). Hidden until `visible`
// (e.g. the engine has produced annotations, or the engine is available for a position);
// shows a busy label while the coach is thinking.
export default function CoachButton({
  label,
  visible,
  loading,
  onClick,
}: {
  label: string;
  visible: boolean;
  loading: boolean;
  onClick: () => void;
}) {
  const t = useT();
  if (!visible) return null;
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={loading}
      className="rounded border border-indigo-300 bg-indigo-50 px-2.5 py-1 text-xs font-medium text-indigo-700 hover:bg-indigo-100 disabled:opacity-50"
    >
      {loading ? t("coach.thinking") : label}
    </button>
  );
}
