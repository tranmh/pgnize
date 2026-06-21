"use client";

import { useEffect, useMemo, useState } from "react";
import { Chess } from "chess.js";
import { useT } from "@/i18n/I18nProvider";
import {
  boardToField,
  buildFen,
  parseBoardField,
  squareToIndex,
  type CastlingRights,
  type FenPiece,
} from "@/lib/fen";
import { STARTING_FEN } from "@/lib/chess";
import Board from "./Board";

export interface PositionEditorProps {
  initialFen: string;
  orientation: "white" | "black";
  onFlip: () => void;
  // Called with the full six-field FEN whenever the position changes.
  onChange?: (fen: string) => void;
  readOnly?: boolean;
}

// A palette tool is either a piece char to stamp, "erase", or null (no tool).
type Tool = FenPiece | "erase";

const WHITE_PIECES: FenPiece[] = ["K", "Q", "R", "B", "N", "P"];
const BLACK_PIECES: FenPiece[] = ["k", "q", "r", "b", "n", "p"];

// Unicode glyphs for the palette buttons (purely cosmetic).
const GLYPH: Record<string, string> = {
  K: "♔", Q: "♕", R: "♖", B: "♗", N: "♘", P: "♙",
  k: "♚", q: "♛", r: "♜", b: "♝", n: "♞", p: "♟",
};

function parseCastling(fen: string): CastlingRights {
  const field = fen.split(" ")[2] ?? "";
  return {
    K: field.includes("K"),
    Q: field.includes("Q"),
    k: field.includes("k"),
    q: field.includes("q"),
  };
}

// PositionEditor lets the user assemble an arbitrary position by hand. It edits
// the board field of the FEN directly (chess.js throws on the illegal
// intermediate positions that occur constantly while editing) and only consults
// chess.js for an advisory legality note on the final composed FEN.
export default function PositionEditor({
  initialFen,
  orientation,
  onFlip,
  onChange,
  readOnly = false,
}: PositionEditorProps) {
  const t = useT();
  // Parse the supplied position once; subsequent edits live in local state.
  const [board, setBoard] = useState<FenPiece[]>(() =>
    parseBoardField(initialFen),
  );
  const [sideToMove, setSideToMove] = useState<"w" | "b">(() =>
    (initialFen.split(" ")[1] === "b" ? "b" : "w"),
  );
  const [castling, setCastling] = useState<CastlingRights>(() =>
    parseCastling(initialFen),
  );
  const [enPassant, setEnPassant] = useState<string>(() => {
    const ep = initialFen.split(" ")[3];
    return ep && ep !== "-" ? ep : "";
  });
  const [tool, setTool] = useState<Tool | null>(null);

  const fen = useMemo(
    () => buildFen({ board, sideToMove, castling, enPassant }),
    [board, sideToMove, castling, enPassant],
  );

  useEffect(() => {
    onChange?.(fen);
    // onChange identity is the caller's concern; we only fire on FEN change.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [fen]);

  // Advisory legality note: chess.js throws on illegal positions; we swallow it
  // and just surface a non-blocking warning.
  const illegal = useMemo(() => {
    try {
      new Chess(fen);
      return false;
    } catch {
      return true;
    }
  }, [fen]);

  const stamp = (square: string) => {
    if (readOnly || tool === null) return;
    const idx = squareToIndex(square);
    if (idx < 0) return;
    setBoard((prev) => {
      const next = [...prev];
      next[idx] = tool === "erase" ? null : tool;
      return next;
    });
  };

  // Dragging relocates a piece. Return false so react-chessboard does not
  // optimistically mutate; the controlled `fen` re-render reflects the change.
  const relocate = (from: string, to: string): boolean => {
    if (readOnly) return false;
    const fromIdx = squareToIndex(from);
    const toIdx = squareToIndex(to);
    if (fromIdx < 0 || toIdx < 0) return false;
    setBoard((prev) => {
      const next = [...prev];
      next[toIdx] = next[fromIdx];
      next[fromIdx] = null;
      return next;
    });
    return false;
  };

  const clearBoard = () => {
    setBoard(parseBoardField(""));
    setCastling({ K: false, Q: false, k: false, q: false });
    setEnPassant("");
  };

  const startingPosition = () => {
    setBoard(parseBoardField(STARTING_FEN));
    setSideToMove("w");
    setCastling({ K: true, Q: true, k: true, q: true });
    setEnPassant("");
  };

  const setRight = (key: keyof CastlingRights, val: boolean) =>
    setCastling((prev) => ({ ...prev, [key]: val }));

  return (
    <div className="flex flex-col gap-4">
      <Board
        fen={fen}
        orientation={orientation}
        allowDragging={!readOnly}
        onMove={relocate}
        onSquareClick={stamp}
      />

      {!readOnly && (
        <>
          <div className="flex flex-col gap-2">
            <span className="text-[11px] font-medium uppercase tracking-wide text-gray-500">
              {t("editor.palette")}
            </span>
            <div className="flex flex-wrap items-center gap-1">
              {WHITE_PIECES.map((p) => (
                <PaletteButton
                  key={p}
                  glyph={GLYPH[p as string]}
                  selected={tool === p}
                  label={p as string}
                  onClick={() => setTool(p)}
                />
              ))}
              {BLACK_PIECES.map((p) => (
                <PaletteButton
                  key={p}
                  glyph={GLYPH[p as string]}
                  selected={tool === p}
                  label={p as string}
                  dark
                  onClick={() => setTool(p)}
                />
              ))}
              <button
                type="button"
                onClick={() => setTool("erase")}
                className={`rounded border px-2 py-1 text-xs ${
                  tool === "erase"
                    ? "border-blue-500 bg-blue-50 text-blue-700"
                    : "border-gray-300 text-gray-600 hover:bg-gray-100"
                }`}
              >
                {t("editor.tool.erase")}
              </button>
            </div>
            <p className="text-xs text-gray-400">{t("editor.placeHint")}</p>
          </div>

          <div className="flex flex-wrap items-center gap-2">
            <button
              type="button"
              onClick={clearBoard}
              className="rounded border border-gray-300 px-3 py-1 text-xs text-gray-700 hover:bg-gray-100"
            >
              {t("editor.clearBoard")}
            </button>
            <button
              type="button"
              onClick={startingPosition}
              className="rounded border border-gray-300 px-3 py-1 text-xs text-gray-700 hover:bg-gray-100"
            >
              {t("editor.startingPosition")}
            </button>
            <button
              type="button"
              onClick={onFlip}
              className="rounded border border-gray-300 px-3 py-1 text-xs text-gray-700 hover:bg-gray-100"
            >
              {t("editor.flip")}
            </button>
          </div>

          <fieldset className="flex flex-wrap items-center gap-4">
            <legend className="text-[11px] font-medium uppercase tracking-wide text-gray-500">
              {t("editor.sideToMove")}
            </legend>
            <label className="flex items-center gap-1 text-sm text-gray-700">
              <input
                type="radio"
                name="side-to-move"
                checked={sideToMove === "w"}
                onChange={() => setSideToMove("w")}
              />
              {t("editor.white")}
            </label>
            <label className="flex items-center gap-1 text-sm text-gray-700">
              <input
                type="radio"
                name="side-to-move"
                checked={sideToMove === "b"}
                onChange={() => setSideToMove("b")}
              />
              {t("editor.black")}
            </label>
          </fieldset>

          <fieldset className="flex flex-wrap items-center gap-4">
            <legend className="text-[11px] font-medium uppercase tracking-wide text-gray-500">
              {t("editor.castling")}
            </legend>
            {(["K", "Q", "k", "q"] as (keyof CastlingRights)[]).map((key) => (
              <label
                key={key}
                className="flex items-center gap-1 text-sm text-gray-700"
              >
                <input
                  type="checkbox"
                  checked={castling[key]}
                  onChange={(e) => setRight(key, e.target.checked)}
                />
                {t(`editor.castling.${key}`)}
              </label>
            ))}
          </fieldset>

          <label className="flex flex-col gap-1">
            <span className="text-[11px] font-medium uppercase tracking-wide text-gray-500">
              {t("editor.enPassant")}
            </span>
            <input
              type="text"
              value={enPassant}
              placeholder={t("editor.enPassant.none")}
              onChange={(e) => setEnPassant(e.target.value.trim())}
              className="w-32 rounded border border-gray-300 px-2 py-1 text-sm focus:border-blue-400 focus:outline-none focus:ring-1 focus:ring-blue-300"
            />
          </label>

          {illegal && (
            <p className="rounded border border-amber-300 bg-amber-50 px-3 py-2 text-sm text-amber-800">
              ⚠ {t("editor.illegalWarning")}
            </p>
          )}

          <p className="break-all font-mono text-xs text-gray-400">
            {boardToField(board)} {sideToMove}
          </p>
        </>
      )}
    </div>
  );
}

function PaletteButton({
  glyph,
  selected,
  label,
  dark,
  onClick,
}: {
  glyph: string;
  selected: boolean;
  label: string;
  dark?: boolean;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-label={label}
      aria-pressed={selected}
      className={`flex h-9 w-9 items-center justify-center rounded border text-2xl leading-none ${
        selected
          ? "border-blue-500 bg-blue-50"
          : "border-gray-300 hover:bg-gray-100"
      } ${dark ? "text-gray-900" : "text-gray-700"}`}
    >
      {glyph}
    </button>
  );
}
