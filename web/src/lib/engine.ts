// Browser-side Stockfish (WASM) wrapper.
//
// Loads the single-threaded "lite" Stockfish.js build as a Web Worker and
// speaks UCI to it. Single-threaded is deliberate: it needs no
// cross-origin-isolation (COOP/COEP) headers, so it cannot break score-sheet
// images served from object storage.
//
// This engine is ADVISORY only — like everything else on the client it never
// decides correctness. The server (chesskit) stays authoritative on save.
//
// The line parsers (`parseInfo`, `parseBestMove`) are pure and exported so they
// can be reasoned about and tested in isolation from the worker.

export const ENGINE_SCRIPT_URL = "/engine/stockfish-18-lite-single.js";

// A normalized evaluation, always from WHITE's point of view.
//   cp:   centipawns (positive = good for White), null when a mate is seen.
//   mate: signed moves-to-mate (positive = White mates), null otherwise.
export interface Score {
  cp: number | null;
  mate: number | null;
  depth: number;
  // Principal variation in UCI long algebraic (e.g. ["e2e4", "e7e5"]).
  pv: string[];
  // First PV move in UCI (e.g. "e2e4", "e7e8q"), or null at terminal nodes.
  bestMove: string | null;
}

export const EMPTY_SCORE: Score = {
  cp: 0,
  mate: null,
  depth: 0,
  pv: [],
  bestMove: null,
};

// Raw, side-to-move-relative info parsed straight off a UCI `info` line.
export interface ParsedInfo {
  depth: number;
  multipv: number;
  // Exactly one of cp / mate is set (side-to-move relative).
  cp: number | null;
  mate: number | null;
  pv: string[];
}

// Parse a UCI `info` line. Returns null for info lines without a score+pv
// (e.g. "info depth 1 currmove ..."). Score is RELATIVE TO THE SIDE TO MOVE,
// exactly as Stockfish reports it — callers normalize to White via `toScore`.
export function parseInfo(line: string): ParsedInfo | null {
  if (!line.startsWith("info ")) return null;
  const tokens = line.split(/\s+/);

  let depth = 0;
  let multipv = 1;
  let cp: number | null = null;
  let mate: number | null = null;
  let pv: string[] = [];

  for (let i = 1; i < tokens.length; i++) {
    const t = tokens[i];
    if (t === "depth") {
      depth = Number(tokens[++i]) || 0;
    } else if (t === "multipv") {
      multipv = Number(tokens[++i]) || 1;
    } else if (t === "score") {
      const kind = tokens[++i];
      const value = Number(tokens[++i]);
      if (kind === "cp") cp = Number.isFinite(value) ? value : null;
      else if (kind === "mate") mate = Number.isFinite(value) ? value : null;
    } else if (t === "pv") {
      pv = tokens.slice(i + 1);
      break; // pv is always last
    }
  }

  if (cp === null && mate === null) return null;
  return { depth, multipv, cp, mate, pv };
}

// Parse a UCI `bestmove` line. Returns the move in UCI, or null for
// "bestmove (none)" (terminal position) / non-bestmove lines.
// Distinguish "not a bestmove line" (undefined) from "no move" (null).
export function parseBestMove(line: string): string | null | undefined {
  if (!line.startsWith("bestmove")) return undefined;
  const move = line.split(/\s+/)[1];
  if (!move || move === "(none)") return null;
  return move;
}

// Convert a side-to-move-relative ParsedInfo into a White-POV Score.
export function toScore(info: ParsedInfo, blackToMove: boolean): Score {
  const sign = blackToMove ? -1 : 1;
  return {
    cp: info.cp === null ? null : info.cp * sign,
    mate: info.mate === null ? null : info.mate * sign,
    depth: info.depth,
    pv: info.pv,
    bestMove: info.pv[0] ?? null,
  };
}

// Collapse a Score to a single comparable centipawn number (White POV).
// Mates map to large magnitudes so they always dominate plain cp values.
export function scoreToCp(score: Score): number {
  if (score.mate !== null) {
    const MATE = 100000;
    return score.mate > 0 ? MATE - score.mate : -MATE - score.mate;
  }
  return score.cp ?? 0;
}

export interface AnalyzeOptions {
  depth?: number;
  movetime?: number;
  multipv?: number;
  // Fired on each deeper main-line (multipv 1) update — for live eval UIs.
  onUpdate?: (best: Score) => void;
  // Abort a running/queued analysis (the worker is told to `stop`).
  signal?: AbortSignal;
}

export interface AnalyzeResult {
  best: Score;
  // One entry per multipv line, ordered best-first. lines[0] === best.
  lines: Score[];
}

type LineListener = (line: string) => void;

// Thin UCI client over a Stockfish Web Worker. One search runs at a time;
// concurrent `analyze` calls are serialized, and starting a new one aborts the
// search in flight so live-eval UIs stay responsive.
export class Engine {
  private worker: Worker;
  private listeners = new Set<LineListener>();
  private ready: Promise<void>;
  private chain: Promise<unknown> = Promise.resolve();

  constructor(scriptUrl: string = ENGINE_SCRIPT_URL) {
    this.worker = new Worker(scriptUrl);
    this.worker.onmessage = (e: MessageEvent) => {
      const line = typeof e.data === "string" ? e.data : e.data?.data;
      if (typeof line === "string") {
        for (const l of this.listeners) l(line);
      }
    };
    this.ready = this.handshake();
  }

  private handshake(): Promise<void> {
    return new Promise((resolve) => {
      const onLine = (line: string) => {
        if (line.startsWith("uciok")) {
          this.listeners.delete(onLine);
          resolve();
        }
      };
      this.listeners.add(onLine);
      this.send("uci");
    });
  }

  private send(cmd: string) {
    this.worker.postMessage(cmd);
  }

  // Analyze a FEN. Serialized behind any in-flight analysis; the previous
  // search is asked to stop first so this one can begin promptly.
  analyze(fen: string, opts: AnalyzeOptions = {}): Promise<AnalyzeResult> {
    // Tell whatever is running to wrap up; our queued job runs after it.
    this.send("stop");
    const run = this.chain.then(() => this.runAnalysis(fen, opts));
    // Keep the chain alive regardless of individual failures/aborts.
    this.chain = run.catch(() => undefined);
    return run;
  }

  private async runAnalysis(
    fen: string,
    opts: AnalyzeOptions,
  ): Promise<AnalyzeResult> {
    await this.ready;
    if (opts.signal?.aborted) return { best: EMPTY_SCORE, lines: [] };

    const blackToMove = fen.split(" ")[1] === "b";
    const multipv = Math.max(1, opts.multipv ?? 1);
    this.send(`setoption name MultiPV value ${multipv}`);
    this.send(`position fen ${fen}`);

    return new Promise<AnalyzeResult>((resolve) => {
      const byLine = new Map<number, ParsedInfo>();

      const cleanup = () => {
        this.listeners.delete(onLine);
        opts.signal?.removeEventListener("abort", onAbort);
      };

      const finish = () => {
        cleanup();
        const ordered = [...byLine.entries()]
          .sort((a, b) => a[0] - b[0])
          .map(([, info]) => toScore(info, blackToMove));
        resolve({ best: ordered[0] ?? EMPTY_SCORE, lines: ordered });
      };

      const onLine = (line: string) => {
        const info = parseInfo(line);
        if (info && info.pv.length) {
          byLine.set(info.multipv, info);
          if (info.multipv === 1 && opts.onUpdate) {
            opts.onUpdate(toScore(info, blackToMove));
          }
          return;
        }
        if (parseBestMove(line) !== undefined) finish();
      };

      const onAbort = () => this.send("stop");

      this.listeners.add(onLine);
      opts.signal?.addEventListener("abort", onAbort);

      const go = opts.movetime
        ? `go movetime ${opts.movetime}`
        : `go depth ${opts.depth ?? 14}`;
      this.send(go);
    });
  }

  terminate() {
    this.listeners.clear();
    this.worker.terminate();
  }
}

// App-wide singleton — one worker is plenty and avoids re-downloading the WASM.
let shared: Engine | null = null;

export function getEngine(): Engine | null {
  if (typeof window === "undefined" || typeof Worker === "undefined") {
    return null;
  }
  if (!shared) shared = new Engine();
  return shared;
}

// Sequentially evaluate a list of positions (White-POV). Calls `onEach` as each
// completes so callers can render progress. Respects an optional AbortSignal.
export async function analyzePositions(
  fens: string[],
  opts: { depth?: number; onEach?: (index: number, score: Score) => void; signal?: AbortSignal } = {},
): Promise<Score[]> {
  const engine = getEngine();
  const out: Score[] = [];
  if (!engine) return out;
  for (let i = 0; i < fens.length; i++) {
    if (opts.signal?.aborted) break;
    const { best } = await engine.analyze(fens[i], {
      depth: opts.depth ?? 12,
      signal: opts.signal,
    });
    out[i] = best;
    opts.onEach?.(i, best);
  }
  return out;
}
