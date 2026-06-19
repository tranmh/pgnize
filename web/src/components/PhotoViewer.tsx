"use client";

import { useRef, useState } from "react";

export interface PhotoViewerProps {
  src: string;
  alt?: string;
}

// Lightweight pan/zoom viewer for the score-sheet photo. Wheel/buttons zoom;
// drag pans. No external deps.
export default function PhotoViewer({ src, alt = "Score sheet" }: PhotoViewerProps) {
  const [scale, setScale] = useState(1);
  const [offset, setOffset] = useState({ x: 0, y: 0 });
  const dragging = useRef<{ x: number; y: number } | null>(null);

  const clampScale = (s: number) => Math.min(6, Math.max(0.5, s));

  const zoomBy = (delta: number) => setScale((s) => clampScale(s + delta));
  const reset = () => {
    setScale(1);
    setOffset({ x: 0, y: 0 });
  };

  return (
    <div className="flex h-full flex-col">
      <div className="flex items-center gap-2 border-b border-gray-200 pb-2">
        <span className="text-sm font-semibold uppercase tracking-wide text-gray-500">
          Photo
        </span>
        <div className="ml-auto flex items-center gap-1">
          <ZoomButton label="−" title="Zoom out" onClick={() => zoomBy(-0.25)} />
          <span className="w-12 text-center text-xs text-gray-500">
            {Math.round(scale * 100)}%
          </span>
          <ZoomButton label="+" title="Zoom in" onClick={() => zoomBy(0.25)} />
          <ZoomButton label="⤢" title="Reset view" onClick={reset} />
        </div>
      </div>

      <div
        className="relative mt-2 flex-1 cursor-grab overflow-hidden rounded bg-gray-100 active:cursor-grabbing"
        onWheel={(e) => {
          e.preventDefault();
          zoomBy(e.deltaY > 0 ? -0.2 : 0.2);
        }}
        onMouseDown={(e) => {
          dragging.current = { x: e.clientX - offset.x, y: e.clientY - offset.y };
        }}
        onMouseMove={(e) => {
          if (!dragging.current) return;
          setOffset({
            x: e.clientX - dragging.current.x,
            y: e.clientY - dragging.current.y,
          });
        }}
        onMouseUp={() => (dragging.current = null)}
        onMouseLeave={() => (dragging.current = null)}
      >
        {/* eslint-disable-next-line @next/next/no-img-element -- presigned external URL, dynamic, no optimization wanted */}
        <img
          src={src}
          alt={alt}
          draggable={false}
          className="absolute left-1/2 top-1/2 max-w-none select-none"
          style={{
            transform: `translate(-50%, -50%) translate(${offset.x}px, ${offset.y}px) scale(${scale})`,
            transformOrigin: "center",
          }}
        />
      </div>
    </div>
  );
}

function ZoomButton({
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
      className="h-7 w-7 rounded border border-gray-300 text-sm hover:bg-gray-100"
    >
      {label}
    </button>
  );
}
