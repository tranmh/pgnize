"use client";

import { useState } from "react";
import type { EditMove, Legality } from "@/lib/chess";
import { legalMovesFrom, PLACEHOLDER } from "@/lib/chess";

export interface MoveListProps {
  moves: EditMove[];
  // Index of the ply currently shown on the board (or null for start position).
  activeIndex: number | null;
  onSelect: (index: number | null) => void;
  onEditSan: (index: number, san: string) => void;
  onInsertAfter: (index: number) => void;
  onDelete: (index: number) => void;
  onPlaceholder: (index: number) => void;
  onTruncate: (index: number) => void;
  readOnly?: boolean;
}

function badgeClasses(legality: Legality): string {
  switch (legality) {
    case "legal":
      return "bg-green-100 text-green-800 border-green-300";
    case "illegal":
      return "bg-red-100 text-red-800 border-red-300";
    case "ambiguous":
      return "bg-amber-100 text-amber-800 border-amber-300";
  }
}

function badgeLabel(legality: Legality): string {
  switch (legality) {
    case "legal":
      return "legal";
    case "illegal":
      return "illegal";
    case "ambiguous":
      return "ambiguous";
  }
}

function moveNumber(side: EditMove["side"], index: number): string {
  // Display 1-based full-move numbers. White ply N -> move ceil((N+1)/2).
  const moveNo = Math.floor(index / 2) + 1;
  return side === "white" ? `${moveNo}.` : `${moveNo}...`;
}

export default function MoveList({
  moves,
  activeIndex,
  onSelect,
  onEditSan,
  onInsertAfter,
  onDelete,
  onPlaceholder,
  onTruncate,
  readOnly = false,
}: MoveListProps) {
  return (
    <div className="flex flex-col">
      <div className="flex items-center justify-between border-b border-gray-200 pb-2">
        <h2 className="text-sm font-semibold uppercase tracking-wide text-gray-500">
          Moves
        </h2>
        <button
          type="button"
          className="rounded px-2 py-1 text-xs text-gray-600 hover:bg-gray-100 disabled:opacity-40"
          onClick={() => onSelect(null)}
          aria-label="Jump to starting position"
        >
          ⏮ start
        </button>
      </div>

      <ol className="divide-y divide-gray-100">
        {moves.length === 0 && (
          <li className="py-6 text-center text-sm text-gray-400">
            No moves yet.
            {!readOnly && " Add a move by dragging a piece on the board."}
          </li>
        )}
        {moves.map((m, i) => (
          <MoveRow
            key={i}
            index={i}
            move={m}
            active={activeIndex === i}
            // Downstream of an illegal/ambiguous ply: greyed + blocked.
            blocked={isBlockedDownstream(moves, i)}
            readOnly={readOnly}
            onSelect={onSelect}
            onEditSan={onEditSan}
            onInsertAfter={onInsertAfter}
            onDelete={onDelete}
            onPlaceholder={onPlaceholder}
            onTruncate={onTruncate}
          />
        ))}
      </ol>
    </div>
  );
}

// A ply is "blocked downstream" when an earlier ply is not legal.
function isBlockedDownstream(moves: EditMove[], index: number): boolean {
  for (let i = 0; i < index; i++) {
    if (moves[i].legality !== "legal") return true;
  }
  return false;
}

interface MoveRowProps {
  index: number;
  move: EditMove;
  active: boolean;
  blocked: boolean;
  readOnly: boolean;
  onSelect: (index: number | null) => void;
  onEditSan: (index: number, san: string) => void;
  onInsertAfter: (index: number) => void;
  onDelete: (index: number) => void;
  onPlaceholder: (index: number) => void;
  onTruncate: (index: number) => void;
}

function MoveRow({
  index,
  move,
  active,
  blocked,
  readOnly,
  onSelect,
  onEditSan,
  onInsertAfter,
  onDelete,
  onPlaceholder,
  onTruncate,
}: MoveRowProps) {
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState(move.san);

  const startEdit = () => {
    if (readOnly) return;
    setDraft(move.san === PLACEHOLDER ? "" : move.san);
    setEditing(true);
  };

  const commit = () => {
    setEditing(false);
    if (draft.trim() !== move.san.trim()) {
      onEditSan(index, draft.trim());
    }
  };

  // Correction dropdown: legal SANs from the position before this ply.
  const legalOptions =
    move.legality !== "legal" ? legalMovesFrom(move.fenBefore) : [];
  const ambiguousOptions = move.ambiguousOptions;

  return (
    <li
      className={[
        "flex flex-col gap-1 px-1 py-2",
        blocked ? "opacity-50" : "",
        active ? "bg-blue-50" : "",
      ].join(" ")}
    >
      <div className="flex items-center gap-2">
        <button
          type="button"
          className="w-12 shrink-0 text-right font-mono text-xs text-gray-500 hover:text-gray-800"
          onClick={() => onSelect(index)}
          aria-label={`Show position after ${moveNumber(move.side, index)} ${move.san}`}
        >
          {moveNumber(move.side, index)}
        </button>

        {editing ? (
          <input
            autoFocus
            value={draft}
            onChange={(e) => setDraft(e.target.value)}
            onBlur={commit}
            onKeyDown={(e) => {
              if (e.key === "Enter") commit();
              if (e.key === "Escape") setEditing(false);
            }}
            placeholder="SAN, e.g. Nf3"
            className="w-24 rounded border border-blue-400 px-2 py-1 font-mono text-sm focus:outline-none focus:ring-2 focus:ring-blue-300"
            aria-label="Edit move SAN"
          />
        ) : (
          <button
            type="button"
            onClick={() => (active ? startEdit() : onSelect(index))}
            onDoubleClick={startEdit}
            className="min-w-[3.5rem] rounded px-2 py-1 text-left font-mono text-sm hover:bg-gray-100"
            title={readOnly ? undefined : "Click to view, again to edit"}
          >
            {move.san || PLACEHOLDER}
          </button>
        )}

        <span
          className={`rounded border px-1.5 py-0.5 text-[10px] font-medium uppercase ${badgeClasses(move.legality)}`}
        >
          {badgeLabel(move.legality)}
        </span>

        {!readOnly && (
          <div className="ml-auto flex items-center gap-1 text-gray-400">
            <RowAction label="?" title="Mark as unreadable placeholder" onClick={() => onPlaceholder(index)} />
            <RowAction label="+" title="Insert a move after this one" onClick={() => onInsertAfter(index)} />
            <RowAction label="✂" title="Truncate game here (delete this and all later moves)" onClick={() => onTruncate(index)} />
            <RowAction label="🗑" title="Delete this move" onClick={() => onDelete(index)} />
          </div>
        )}
      </div>

      {move.recognizedText && move.recognizedText !== move.san && (
        <div className="pl-14 text-[11px] text-gray-400">
          read as “{move.recognizedText}”
          {move.corrected && <span className="ml-1 text-blue-500">(corrected)</span>}
        </div>
      )}

      {/* Ambiguous: offer the disambiguated candidates. */}
      {!readOnly && move.legality === "ambiguous" && ambiguousOptions.length > 1 && (
        <CorrectionPicker
          label="Disambiguate"
          options={ambiguousOptions}
          onPick={(san) => onEditSan(index, san)}
        />
      )}

      {/* Illegal: offer the legal moves for this position. */}
      {!readOnly &&
        move.legality === "illegal" &&
        !blocked &&
        legalOptions.length > 0 && (
          <CorrectionPicker
            label="Pick a legal move"
            options={legalOptions}
            onPick={(san) => onEditSan(index, san)}
          />
        )}
    </li>
  );
}

function RowAction({
  label,
  title,
  onClick,
}: {
  label: string;
  title: string;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      title={title}
      aria-label={title}
      onClick={onClick}
      className="h-6 w-6 rounded text-xs hover:bg-gray-200 hover:text-gray-700"
    >
      {label}
    </button>
  );
}

function CorrectionPicker({
  label,
  options,
  onPick,
}: {
  label: string;
  options: string[];
  onPick: (san: string) => void;
}) {
  return (
    <div className="pl-14">
      <label className="flex items-center gap-2 text-[11px] text-gray-500">
        {label}:
        <select
          className="rounded border border-gray-300 px-1 py-0.5 font-mono text-xs"
          defaultValue=""
          onChange={(e) => {
            if (e.target.value) onPick(e.target.value);
          }}
          aria-label={label}
        >
          <option value="" disabled>
            choose…
          </option>
          {options.map((o) => (
            <option key={o} value={o}>
              {o}
            </option>
          ))}
        </select>
      </label>
    </div>
  );
}
