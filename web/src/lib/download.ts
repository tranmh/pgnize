// Trigger a browser download of text content as a file.
export function downloadText(filename: string, content: string) {
  const blob = new Blob([content], { type: "text/plain;charset=utf-8" });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  a.remove();
  URL.revokeObjectURL(url);
}

// Build a sensible PGN filename from header-ish fields.
export function pgnFilename(white?: string, black?: string): string {
  const slug = (s?: string) =>
    (s || "").trim().replace(/[^\w-]+/g, "_").slice(0, 40) || "game";
  return `${slug(white)}_vs_${slug(black)}.pgn`;
}
