"use client";

import { Chessboard } from "react-chessboard";

export interface BoardProps {
  // FEN of the position to display.
  fen: string;
  orientation?: "white" | "black";
  // Whether the user may drag pieces (false in read-only / blocked states).
  allowDragging?: boolean;
  // Called when the user drops a piece. Return true if the move was accepted
  // (the parent then updates `fen`), false to snap the piece back.
  onMove?: (from: string, to: string) => boolean;
  // Highlight styles keyed by square (e.g. { e4: { background: ... } }).
  squareStyles?: Record<string, React.CSSProperties>;
}

// Thin wrapper over react-chessboard v5's options-based API so the rest of the
// app deals in plain (fen, orientation, onMove) props.
export default function Board({
  fen,
  orientation = "white",
  allowDragging = false,
  onMove,
  squareStyles,
}: BoardProps) {
  return (
    <div className="w-full max-w-[480px]">
      <Chessboard
        options={{
          id: "review-board",
          position: fen,
          boardOrientation: orientation,
          allowDragging: allowDragging && !!onMove,
          showNotation: true,
          animationDurationInMs: 150,
          squareStyles,
          onPieceDrop: ({ sourceSquare, targetSquare }) => {
            if (!onMove || !targetSquare) return false;
            return onMove(sourceSquare, targetSquare);
          },
        }}
      />
    </div>
  );
}
