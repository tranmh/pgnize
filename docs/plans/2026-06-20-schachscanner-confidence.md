# Schachscanner-style per-move confidence + recognition-quality upgrades

## Context

The ChessBase *Schachscanner* article describes a photo-to-PGN tool whose headline UX is
**per-move confidence color-coding** — recognized moves are shown green (high confidence) or
yellow (uncertain) so the reviewer knows exactly where to look before saving. The article
stresses that *even legal moves can be wrong* (its example: an ambiguous knight move that the AI
reads as legal but plays with the wrong knight, losing the game), so the reviewer must compare
against the sheet — and the tool directs that attention with color.

PGNize already does the hard parts: async recognition (Ollama + Gemini), German→SAN postprocess,
a server-authoritative review loop, and a split-screen workbench with **legality** badges
(green=legal / red=illegal / amber=ambiguous), "did you mean" suggestions, and disambiguation
pickers. What's missing is a **confidence dimension that is independent of legality**: today a
legal-but-shaky read (auto-corrected by edit-distance, a guessed disambiguation, or a low-quality
scan) renders as solid green, giving the reviewer no signal to double-check it.

This plan adds that confidence dimension end-to-end, plus two recognition-quality fixes the
article calls out (the ambiguous-piece "knight problem" and long-algebraic notation). Scope is
**single-game** (no batch/two-form work) and keeps **both** Ollama and Gemini backends.

## Design overview

Introduce a per-move **confidence** value (0..1), computed server-side in `Reconcile`, persisted,
returned via the API, and rendered in the workbench as a **"verify" (yellow) state for legal
moves below a threshold** — orthogonal to the existing legality badge. The UI gains a
"N moves to verify" chip and next-uncertain navigation; selecting/editing a flagged ply clears it.
Save stays gated on all-legal (unchanged correctness guarantee).

Confidence is deliberately **deterministic**, not model self-reported (the recognizers already
note models "do not reliably self-report" and return a flat 0.5). The flat model number is treated
as "no signal"; confidence is driven by signals we actually trust:

| Situation (per ply)                                   | confidence | UI state |
|-------------------------------------------------------|-----------:|----------|
| placeholder `?` / illegal / blocked downstream        | 0.0        | red / unread (legality badge, as today) |
| guessed disambiguation (new auto-pick, "knight case") | 0.30       | **yellow — verify** |
| auto-corrected via edit-distance (`corrected=true`)   | 0.40       | **yellow — verify** |
| salvaged from truncated JSON (model conf ~0.3)         | 0.30       | **yellow — verify** |
| cleanly validated legal read                          | 0.90       | green |

Threshold: `>= 0.6` → confident (green); legal `< 0.6` → yellow "verify". With Ollama's flat 0.5
every *clean* move still scores 0.90 (not the flat input), so yellow stays a rare, meaningful
signal instead of lighting up the whole game.

## Backend changes (Go)

**Migration** — `migrations/0003_move_confidence.sql` (goose up/down):
`ALTER TABLE moves ADD COLUMN confidence real NOT NULL DEFAULT 1.0;`
Default 1.0 = "verified/confident unless flagged", so manually-entered and saved games need no
extra wiring; only the recognition path writes lower values.

**Domain** — `internal/domain/types.go`: add `Confidence float64 \`json:"confidence"\`` to `Move`.

**Reconcile + confidence** — `internal/recognition/postprocess.go`:
- Set `m.Confidence` for every ply via a small `plyConfidence(...)` helper per the table above.
- **Ambiguous auto-pick (the knight problem):** today an ambiguous read (e.g. "Ne5" with two
  knights) fails `Validate`, `matchLegal` returns a tie → not confident → the move is marked
  illegal and **blocks the rest of the game**. Change this: when the *only* reason for failure is
  ambiguity (the recognized piece+destination matches ≥2 legal moves differing only by origin),
  auto-pick a deterministic default disambiguation, mark `IsLegal=true`, `Corrected=true`,
  `Confidence=0.30`, keep the alternates in `Suggestions`, and **continue** the game. This matches
  the article ("AI makes a logical assumption, user verifies") and keeps the reviewer's view of the
  whole game intact. Add a helper that detects ambiguity from the candidate set (`san` whose
  piece+dest matches multiple legal SANs).
- **Long-algebraic → short normalization:** extend `GermanToSAN` (or a pre-Validate step in
  `Reconcile`) to reduce long notation the model was *not* trained on — `Sf3-e5`/`Sf3xe5` → `Ne5`
  (piece + destination; let `Validate`/ambiguity flow disambiguate), and pawn `e2-e4` → `e4`,
  `e4:d5`/`e4xd5` → `exd5`. Keep promotion/check suffixes. This is the article's explicit
  limitation ("die KI wurde mit der kurzen Notation trainiert").

**Store** — `internal/store/games.go`:
- `insertMoves`: add `confidence` to the INSERT column list and values.
- `GetGame`: add `confidence` to the moves SELECT and `Scan`.
- `SaveGame` path is unchanged in behavior — moves rebuilt from `MoveInput` carry the column
  default (1.0 = verified) since the reviewer has confirmed them.

**Recognizers** — keep both, no API/model changes required for v1 (deterministic confidence does
not need model cooperation). Leave the per-`MoveToken.Confidence` plumbing as-is; `Reconcile`
already receives it and will use it only to detect the salvage/low-signal case. *(Optional follow-up,
not in this plan: extend Gemini's `responseSchema` with a per-cell uncertainty flag to downgrade
clean-but-shaky reads to yellow — Gemini is reliable enough for this; Ollama stays deterministic.)*

**API contract** — `Move` is serialized straight from `domain.Move`, so `confidence` flows out of
`GET /api/games/{id}` automatically once the domain field exists.

## Frontend changes (Next.js/React)

> Note `web/AGENTS.md`: this Next.js has breaking changes — consult `node_modules/next/dist/docs/`
> before writing framework-level code. These changes are component-level.

**Types** — `web/src/lib/api-client.ts`: add `confidence: number` to `Move`.

**Chess model** — `web/src/lib/chess.ts`:
- `EditMove` gains `confidence: number` (carried through `rebuild`, sourced from the API move;
  human-added moves default to 1.0).
- `toEditablePlies` carries `confidence` onto the editable ply.
- Add a derived helper `reviewState(move, confirmed)` → `"illegal" | "unread" | "verify" | "ok"`:
  `verify` when `legality==="legal" && confidence < 0.6 && !confirmed`.

**Move list** — `web/src/components/MoveList.tsx`:
- Add a yellow "verify" treatment (e.g. an amber dot/ring + tooltip) for `verify` plies, distinct
  from the existing legality badge. Reuse the existing suggestion/disambiguation UI — for a flagged
  legal move, surface sibling disambiguations (same piece+destination, other origins) computed from
  `fenBefore` via the existing `legalMovesFrom` + a small same-target filter, so the knight choice
  is one click.
- Selecting or editing a ply marks it confirmed (clears yellow).

**Workbench** — `web/src/components/ReviewWorkbench.tsx`:
- Track `confirmed: Set<number>` of ply indices the reviewer has touched.
- Add a "**N moves to verify**" chip + a "next uncertain" button (jumps `activeIndex` to the next
  `verify` ply) for fast throughput review.
- Keep the save button gated on `allLegal` only; optionally show a soft, non-blocking note when
  unverified moves remain ("review highlighted moves before saving") — mirrors the article's advice,
  does not block.

**i18n** — add keys under `moves.*` / `review.*` in both locale files (the repo has German +
English; `moves.verify`, `moves.verifyHint`, `review.toVerify`, `review.nextUncertain`,
`review.unverifiedNote`).

## Tests (TDD — write failing tests first, then implement)

- **Unit (`make test`, no DB):**
  - `internal/recognition/postprocess_test.go`: long-algebraic normalization cases
    (`Sf3-e5`→`Ne5`, `e2-e4`→`e4`, `e4:d5`→`exd5`); ambiguous auto-pick continues the game and
    sets `Confidence≈0.30`, `Corrected`, populated `Suggestions`; `plyConfidence` table values;
    clean legal read → 0.90; placeholder/illegal → 0.0.
  - Frontend: a `chess.ts` test for `reviewState` (verify vs ok vs illegal) if the repo has unit
    coverage there; otherwise cover via e2e.
- **Integration (`make test-int`, real Postgres):** round-trip a draft with mixed confidences
  through `CreateDraftGame` → `GetGame` and assert per-move `confidence` persists; assert
  `SaveGame` reinsert defaults to 1.0.
- **E2E (`make e2e-ui`, `RECOGNIZER=fake`):** extend `fake.go` (or the fixture) to emit at least
  one ambiguous/low-confidence ply; assert the workbench renders the yellow "verify" state, the
  "N to verify" chip counts it, "next uncertain" jumps to it, and confirming/editing clears it.
  Keep an `api` project assertion that `GET /api/games/{id}` includes `confidence` per move.

## Real-world test fixtures + confidence demonstration (required deliverable)

Environment confirmed: **Ollama is running locally with `minicpm-v`** (the configured default
vision model) — so we can run real handwritten scoresheets through the *actual* recognizer +
`Reconcile` and print genuine per-move confidence. No Gemini key is present; Ollama is the live
backend for the demo.

**Fixtures** — collect into `testdata/scoresheets/*.jpg` at repo root (`testdata/` is ignored by
Go tooling; safe to commit):
1. The ChessBase article images, downloaded exactly as instructed, from
   `https://de.chessbase.com/portals/all/2025/01/schachscanner/` —
   `formulare-screenshot.jpg`, `formulare2.jpg`, `Bericht%20ZugRaster.jpg` (re-encode to `.jpg`).
   Note: `formulare*` are handwriting-style montages and `Bericht ZugRaster` is an app screenshot,
   so they exercise the recognizer/confidence path but are not all clean single-game sheets.
2. **More real-world full-game handwritten scoresheets**: source a handful of openly-available
   handwritten chess scoresheet photos (e.g. public handwritten-scoresheet datasets / open repos),
   download and save as `*.jpg`. Target ~4–6 total fixtures to bound runtime (minicpm-v on CPU is
   ~1–3 min/image). If a cleanly-usable single-game photo can't be reliably sourced for an entry,
   record that in the harness output and proceed with what's available — the deterministic
   confidence logic is fully covered by the unit/integration tests regardless.
   A small `testdata/scoresheets/README.md` records each file's source URL + provenance.
3. A `LICENSE`/provenance note: these are third-party reference images committed only as test
   fixtures; flag the licensing caveat in the fixtures README.

**Harness** — `internal/recognition/realworld_test.go`, env-gated (`RUN_REAL_RECOGNITION=1`, skipped
otherwise so it never runs in `make test`/CI): for each `testdata/scoresheets/*.jpg`, run the real
`Ollama` recognizer → `Reconcile`, then `t.Log` a per-image table:

```
file=formulare2.jpg  header: White=… Black=…  overall=0.50
  ply  side   read→san        legality   conf   state
   1   white  e4 → e4         legal      0.90   ok
   2   black  Sc6 → Nc6       legal      0.90   ok
   3   white  Sf3-e5 → Ne5    legal      0.30   verify (long-notation normalized)
   4   white  Mf3 → Nf3       legal      0.40   verify (auto-corrected)
   …
  summary: 24 plies · 19 ok · 3 verify · 2 illegal · mean conf 0.71
```

The harness asserts invariants (every `confidence ∈ [0,1]`; legal+clean ⇒ ≥0.6; illegal ⇒ 0) and
prints the full table. **Run it at the end and surface the captured output (recognized moves +
confidence per ply + per-file summary) in the final report.**

## Verification

1. `cd chesskit && go test ./...` then `make test` — unit green (confidence + notation logic).
2. `make migrate` then `make test-int` — column + round-trip green.
3. `make e2e-api` and `make e2e-ui` — confidence in payload + yellow-state UX green.
4. `make lint`.
5. `RUN_REAL_RECOGNITION=1 go test ./internal/recognition/ -run RealWorld -v` against the
   committed `testdata/scoresheets/*.jpg` via local Ollama/minicpm-v — **capture and present the
   per-move confidence tables + summaries as the closing deliverable.**
6. Manual: `make dev`, open a recognized draft, confirm legal moves are green, an
   auto-corrected/ambiguous move is yellow with one-click alternates, the "verify" chip counts down
   as you confirm, and save still rejects illegal positions with `failedAt`.

## Execution mode

- **Fully agentic**, in a git **worktree** (not the primary checkout), per project convention.
- Commit this plan to `docs/plans/` as the first step.
- TDD throughout: failing tests first, then implement; run every suite and report results.

## Notes / out of scope

- **Out of scope (confirmed):** batch / multi-sheet upload, two-form long-game stitching, sharing
  between users, library/PGN-export changes, annotations/comments.
- Optional future: Gemini per-cell uncertainty in `responseSchema` to flag clean-but-shaky reads;
  feed it into `plyConfidence` as an additional downgrade input.
