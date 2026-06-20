// Typed fetch wrapper for the pgnize REST API.
//
// This file is the CONTRACT BOUNDARY. The types below mirror
// docs/api-contract.md exactly. The Go backend (internal/httpapi) and this file
// must both conform to the contract. Keep these shapes precise.
//
// All requests go to `/api/*`; in dev Next.js rewrites that to PGNIZE_API_URL.
// Auth is a session cookie, so every request uses `credentials: 'include'`.

// ---------------------------------------------------------------------------
// Shared JSON shapes
// ---------------------------------------------------------------------------

export type Result = "1-0" | "0-1" | "1/2-1/2" | "*";
export type Side = "white" | "black";
export type GameSource = "recognized" | "manual";
export type GameStatus = "draft" | "reviewing" | "saved";

export interface Header {
  white: string;
  black: string;
  event: string;
  site: string;
  date: string; // "YYYY.MM.DD" or ""
  round: string;
  board: string;
  result: Result;
}

export interface Move {
  ply: number;
  side: Side;
  san: string;
  fenAfter: string;
  clockSec: number | null;
  isLegal: boolean;
  recognizedText: string;
  corrected: boolean;
  // Deterministic recognition confidence (0..1), independent of legality. A legal move below
  // the review threshold is surfaced as a "verify" (yellow) state. 1.0 for human-entered moves.
  confidence: number;
}

export interface GameDraft {
  id: string;
  source: GameSource;
  status: GameStatus;
  header: Header;
  startFen: string;
  moves: Move[];
  imageUrl: string | null;
  confidence: number;
}

export interface GameSummary {
  id: string;
  white: string;
  black: string;
  event: string;
  date: string;
  result: Result;
  moveCount: number;
  savedAt: string | null; // RFC3339 or null
}

export interface User {
  id: string;
  name: string;
  email: string;
}

export interface Player {
  id: string;
  fullName: string;
  club: string;
  fideId: string;
}

export type JobStatus = "queued" | "running" | "done" | "failed";

export interface JobState {
  status: JobStatus;
  gameId?: string;
  error?: string;
}

// Payload accepted by save/export endpoints. Only the editable subset of a Move
// is sent back; the server recomputes fenAfter / isLegal authoritatively.
export interface MoveInput {
  ply: number;
  san: string;
  clockSec?: number | null;
}

export interface SavePayload {
  header: Header;
  moves: MoveInput[];
  startFen?: string;
}

// ---------------------------------------------------------------------------
// Error handling
// ---------------------------------------------------------------------------

export interface ApiErrorBody {
  error: string;
  message?: string;
  // present only for error === "illegal_move": 0-based ply index that failed.
  failedAt?: number;
}

export class ApiError extends Error {
  readonly status: number;
  readonly code: string;
  readonly failedAt?: number;

  constructor(status: number, body: ApiErrorBody | null, fallback: string) {
    super(body?.message || body?.error || fallback);
    this.name = "ApiError";
    this.status = status;
    this.code = body?.error ?? "http_error";
    this.failedAt = body?.failedAt;
  }
}

// ---------------------------------------------------------------------------
// Low-level request helpers
// ---------------------------------------------------------------------------

const BASE = "/api";

async function parseError(res: Response, fallback: string): Promise<ApiError> {
  let body: ApiErrorBody | null = null;
  try {
    body = (await res.json()) as ApiErrorBody;
  } catch {
    body = null;
  }
  return new ApiError(res.status, body, fallback);
}

async function requestJson<T>(
  path: string,
  init: RequestInit = {},
): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    credentials: "include",
    ...init,
    headers: {
      Accept: "application/json",
      ...(init.body && !(init.body instanceof FormData)
        ? { "Content-Type": "application/json" }
        : {}),
      ...(init.headers ?? {}),
    },
  });

  if (!res.ok) {
    throw await parseError(res, `Request failed (${res.status})`);
  }

  // 204 No Content
  if (res.status === 204) {
    return undefined as T;
  }
  return (await res.json()) as T;
}

async function requestText(
  path: string,
  init: RequestInit = {},
): Promise<string> {
  const res = await fetch(`${BASE}${path}`, {
    credentials: "include",
    ...init,
    headers: {
      Accept: "text/plain",
      ...(init.body && !(init.body instanceof FormData)
        ? { "Content-Type": "application/json" }
        : {}),
      ...(init.headers ?? {}),
    },
  });
  if (!res.ok) {
    throw await parseError(res, `Request failed (${res.status})`);
  }
  return res.text();
}

function jsonBody(value: unknown): string {
  return JSON.stringify(value);
}

// ---------------------------------------------------------------------------
// Auth
// ---------------------------------------------------------------------------

export function register(input: {
  name: string;
  email: string;
  password: string;
}): Promise<{ user: User }> {
  return requestJson("/auth/register", {
    method: "POST",
    body: jsonBody(input),
  });
}

export function login(input: {
  email: string;
  password: string;
}): Promise<{ user: User }> {
  return requestJson("/auth/login", {
    method: "POST",
    body: jsonBody(input),
  });
}

export function logout(): Promise<void> {
  return requestJson("/auth/logout", { method: "POST" });
}

export function me(): Promise<{ user: User }> {
  return requestJson("/auth/me", { method: "GET" });
}

// ---------------------------------------------------------------------------
// Anonymous convert
// ---------------------------------------------------------------------------

export function convert(
  image: File,
  backend?: string,
): Promise<{ jobId: string }> {
  const fd = new FormData();
  fd.append("image", image);
  if (backend) {
    fd.append("backend", backend);
  }
  return requestJson("/convert", { method: "POST", body: fd });
}

export function getConvertJob(jobId: string): Promise<JobState> {
  return requestJson(`/convert/${encodeURIComponent(jobId)}`, {
    method: "GET",
  });
}

export function getConvertGame(jobId: string): Promise<GameDraft> {
  return requestJson(`/convert/${encodeURIComponent(jobId)}/game`, {
    method: "GET",
  });
}

export function exportConvertPgn(
  jobId: string,
  payload: Pick<SavePayload, "header" | "moves">,
): Promise<string> {
  return requestText(`/convert/${encodeURIComponent(jobId)}/export`, {
    method: "POST",
    body: jsonBody(payload),
  });
}

// ---------------------------------------------------------------------------
// Account: upload -> job -> review -> save
// ---------------------------------------------------------------------------

export function upload(
  image: File,
  consentTraining: boolean,
  backend?: string,
): Promise<{ uploadId: string; jobId: string }> {
  const fd = new FormData();
  fd.append("image", image);
  if (consentTraining) {
    fd.append("consentTraining", "true");
  }
  if (backend) {
    fd.append("backend", backend);
  }
  return requestJson("/uploads", { method: "POST", body: fd });
}

// ---------------------------------------------------------------------------
// Recognition backends (which engine reads the score sheet)
// ---------------------------------------------------------------------------

export interface RecognizerInfo {
  key: string;
  label: string;
  default: boolean;
}

export function fetchRecognizers(): Promise<{
  recognizers: RecognizerInfo[];
  default: string;
}> {
  return requestJson("/recognizers", { method: "GET" });
}

export function getJob(jobId: string): Promise<JobState> {
  return requestJson(`/jobs/${encodeURIComponent(jobId)}`, { method: "GET" });
}

export function createManualGame(): Promise<{ game: GameDraft }> {
  return requestJson("/games", {
    method: "POST",
    body: jsonBody({ source: "manual" }),
  });
}

export function getGame(id: string): Promise<GameDraft> {
  return requestJson(`/games/${encodeURIComponent(id)}`, { method: "GET" });
}

export function saveGame(
  id: string,
  payload: SavePayload,
): Promise<{ game: GameDraft }> {
  return requestJson(`/games/${encodeURIComponent(id)}`, {
    method: "PATCH",
    body: jsonBody(payload),
  });
}

export function deleteGame(id: string): Promise<void> {
  return requestJson(`/games/${encodeURIComponent(id)}`, { method: "DELETE" });
}

// ---------------------------------------------------------------------------
// Library
// ---------------------------------------------------------------------------

export interface ListGamesParams {
  q?: string;
  player?: string;
  event?: string;
  from?: string;
  to?: string;
  page?: number;
  pageSize?: number;
}

export interface ListGamesResponse {
  games: GameSummary[];
  total: number;
  page: number;
  pageSize: number;
}

export function listGames(
  params: ListGamesParams = {},
): Promise<ListGamesResponse> {
  const qs = new URLSearchParams();
  if (params.q) qs.set("q", params.q);
  if (params.player) qs.set("player", params.player);
  if (params.event) qs.set("event", params.event);
  if (params.from) qs.set("from", params.from);
  if (params.to) qs.set("to", params.to);
  if (params.page) qs.set("page", String(params.page));
  if (params.pageSize) qs.set("pageSize", String(params.pageSize));
  const suffix = qs.toString() ? `?${qs.toString()}` : "";
  return requestJson(`/games${suffix}`, { method: "GET" });
}

export function getGamePgn(id: string): Promise<string> {
  return requestText(`/games/${encodeURIComponent(id)}/pgn`, { method: "GET" });
}

export function exportGamesBundle(ids: string[]): Promise<string> {
  return requestText("/games/export", {
    method: "POST",
    body: jsonBody({ ids }),
  });
}

// ---------------------------------------------------------------------------
// Players (autocomplete)
// ---------------------------------------------------------------------------

export function searchPlayers(q: string): Promise<{ players: Player[] }> {
  const suffix = q ? `?q=${encodeURIComponent(q)}` : "";
  return requestJson(`/players${suffix}`, { method: "GET" });
}
