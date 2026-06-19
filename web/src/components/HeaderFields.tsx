"use client";

import type { Header, Result } from "@/lib/api-client";
import PlayerAutocomplete from "./PlayerAutocomplete";

export interface HeaderFieldsProps {
  header: Header;
  onChange: (header: Header) => void;
  readOnly?: boolean;
}

const RESULTS: Result[] = ["1-0", "0-1", "1/2-1/2", "*"];

// Editable PGN header. White/Black use player autocomplete; the rest are plain
// inputs. `date` is free text (contract format YYYY.MM.DD).
export default function HeaderFields({
  header,
  onChange,
  readOnly = false,
}: HeaderFieldsProps) {
  const set = <K extends keyof Header>(key: K, val: Header[K]) =>
    onChange({ ...header, [key]: val });

  return (
    <div className="grid grid-cols-2 gap-x-4 gap-y-3">
      <Field label="White" htmlFor="hdr-white">
        <PlayerAutocomplete
          id="hdr-white"
          aria-label="White player"
          value={header.white}
          onChange={(v) => set("white", v)}
          placeholder="White player"
        />
      </Field>
      <Field label="Black" htmlFor="hdr-black">
        <PlayerAutocomplete
          id="hdr-black"
          aria-label="Black player"
          value={header.black}
          onChange={(v) => set("black", v)}
          placeholder="Black player"
        />
      </Field>
      <Field label="Event" htmlFor="hdr-event">
        <Text id="hdr-event" value={header.event} onChange={(v) => set("event", v)} readOnly={readOnly} />
      </Field>
      <Field label="Site" htmlFor="hdr-site">
        <Text id="hdr-site" value={header.site} onChange={(v) => set("site", v)} readOnly={readOnly} />
      </Field>
      <Field label="Date (YYYY.MM.DD)" htmlFor="hdr-date">
        <Text id="hdr-date" value={header.date} onChange={(v) => set("date", v)} readOnly={readOnly} placeholder="2026.06.19" />
      </Field>
      <Field label="Round" htmlFor="hdr-round">
        <Text id="hdr-round" value={header.round} onChange={(v) => set("round", v)} readOnly={readOnly} />
      </Field>
      <Field label="Board" htmlFor="hdr-board">
        <Text id="hdr-board" value={header.board} onChange={(v) => set("board", v)} readOnly={readOnly} />
      </Field>
      <Field label="Result" htmlFor="hdr-result">
        <select
          id="hdr-result"
          value={header.result}
          disabled={readOnly}
          onChange={(e) => set("result", e.target.value as Result)}
          className="w-full rounded border border-gray-300 px-2 py-1 text-sm focus:border-blue-400 focus:outline-none focus:ring-1 focus:ring-blue-300 disabled:bg-gray-50"
          aria-label="Result"
        >
          {RESULTS.map((r) => (
            <option key={r} value={r}>
              {r}
            </option>
          ))}
        </select>
      </Field>
    </div>
  );
}

function Field({
  label,
  htmlFor,
  children,
}: {
  label: string;
  htmlFor: string;
  children: React.ReactNode;
}) {
  return (
    <label htmlFor={htmlFor} className="flex flex-col gap-1">
      <span className="text-[11px] font-medium uppercase tracking-wide text-gray-500">
        {label}
      </span>
      {children}
    </label>
  );
}

function Text({
  id,
  value,
  onChange,
  readOnly,
  placeholder,
}: {
  id: string;
  value: string;
  onChange: (v: string) => void;
  readOnly?: boolean;
  placeholder?: string;
}) {
  return (
    <input
      id={id}
      type="text"
      value={value}
      readOnly={readOnly}
      placeholder={placeholder}
      onChange={(e) => onChange(e.target.value)}
      className="w-full rounded border border-gray-300 px-2 py-1 text-sm focus:border-blue-400 focus:outline-none focus:ring-1 focus:ring-blue-300 read-only:bg-gray-50"
    />
  );
}
