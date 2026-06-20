"use client";

import { useEffect } from "react";

export interface MoveNavProps {
  // Currently shown ply index, or null for the starting position.
  index: number | null;
  // Number of plies in the game.
  count: number;
  onChange: (index: number | null) => void;
  // Bind ←/→ (and Home/End) to step through moves. Off by default so multiple
  // boards on a page don't fight over the keyboard.
  keyboard?: boolean;
}

// First / previous / next / last controls for stepping through a game. `index`
// of null is the starting position; 0..count-1 are positions after each ply.
export default function MoveNav({
  index,
  count,
  onChange,
  keyboard = false,
}: MoveNavProps) {
  const cur = index ?? -1;
  const atStart = cur <= -1;
  const atEnd = cur >= count - 1;

  const go = (next: number) => {
    const clamped = Math.max(-1, Math.min(count - 1, next));
    onChange(clamped < 0 ? null : clamped);
  };

  useEffect(() => {
    if (!keyboard) return;
    const onKey = (e: KeyboardEvent) => {
      // Don't hijack typing in inputs (e.g. the SAN editor or header fields).
      const el = e.target as HTMLElement | null;
      if (el && /^(INPUT|TEXTAREA|SELECT)$/.test(el.tagName)) return;
      if (e.key === "ArrowLeft") {
        e.preventDefault();
        go(cur - 1);
      } else if (e.key === "ArrowRight") {
        e.preventDefault();
        go(cur + 1);
      } else if (e.key === "Home") {
        e.preventDefault();
        onChange(null);
      } else if (e.key === "End") {
        e.preventDefault();
        go(count - 1);
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [keyboard, cur, count]);

  return (
    <div className="flex items-center gap-1">
      <NavButton label="⏮" title="Start position" disabled={atStart} onClick={() => onChange(null)} />
      <NavButton label="◀" title="Previous move" disabled={atStart} onClick={() => go(cur - 1)} />
      <NavButton label="▶" title="Next move" disabled={atEnd} onClick={() => go(cur + 1)} />
      <NavButton label="⏭" title="Last move" disabled={atEnd} onClick={() => go(count - 1)} />
    </div>
  );
}

function NavButton({
  label,
  title,
  disabled,
  onClick,
}: {
  label: string;
  title: string;
  disabled: boolean;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      title={title}
      aria-label={title}
      disabled={disabled}
      onClick={onClick}
      className="rounded border border-gray-300 px-2.5 py-1 text-sm hover:bg-gray-100 disabled:cursor-not-allowed disabled:opacity-40"
    >
      {label}
    </button>
  );
}
