"use client";

import { useEffect, useRef, useState } from "react";
import { searchPlayers, type Player } from "@/lib/api-client";

export interface PlayerAutocompleteProps {
  id?: string;
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  "aria-label"?: string;
}

// Free-text input with a suggestion dropdown sourced from GET /players?q=.
// The user may type any name (the value is free text); suggestions are a
// convenience drawn from their saved pool.
export default function PlayerAutocomplete({
  id,
  value,
  onChange,
  placeholder,
  "aria-label": ariaLabel,
}: PlayerAutocompleteProps) {
  const [suggestions, setSuggestions] = useState<Player[]>([]);
  const [open, setOpen] = useState(false);
  const boxRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const q = value.trim();
    if (q.length < 2) {
      setSuggestions([]);
      return;
    }
    let cancelled = false;
    const handle = setTimeout(async () => {
      try {
        const { players } = await searchPlayers(q);
        if (!cancelled) setSuggestions(players);
      } catch {
        if (!cancelled) setSuggestions([]);
      }
    }, 200);
    return () => {
      cancelled = true;
      clearTimeout(handle);
    };
  }, [value]);

  // Close on outside click.
  useEffect(() => {
    const onDoc = (e: MouseEvent) => {
      if (boxRef.current && !boxRef.current.contains(e.target as Node)) {
        setOpen(false);
      }
    };
    document.addEventListener("mousedown", onDoc);
    return () => document.removeEventListener("mousedown", onDoc);
  }, []);

  return (
    <div ref={boxRef} className="relative">
      <input
        id={id}
        type="text"
        value={value}
        placeholder={placeholder}
        aria-label={ariaLabel}
        autoComplete="off"
        onChange={(e) => {
          onChange(e.target.value);
          setOpen(true);
        }}
        onFocus={() => setOpen(true)}
        className="w-full rounded border border-gray-300 px-2 py-1 text-sm focus:border-blue-400 focus:outline-none focus:ring-1 focus:ring-blue-300"
      />
      {open && suggestions.length > 0 && (
        <ul className="absolute z-20 mt-1 max-h-48 w-full overflow-auto rounded border border-gray-200 bg-white shadow-lg">
          {suggestions.map((p) => (
            <li key={p.id}>
              <button
                type="button"
                className="flex w-full flex-col px-2 py-1 text-left text-sm hover:bg-blue-50"
                onClick={() => {
                  onChange(p.fullName);
                  setOpen(false);
                }}
              >
                <span>{p.fullName}</span>
                {(p.club || p.fideId) && (
                  <span className="text-[11px] text-gray-400">
                    {[p.club, p.fideId && `FIDE ${p.fideId}`]
                      .filter(Boolean)
                      .join(" · ")}
                  </span>
                )}
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
