# Real-world score-sheet fixtures

Third-party images of handwritten/printed chess score sheets, committed **only** as test
fixtures for the recognition + per-move-confidence demo
(`internal/recognition/realworld_test.go`, gated by `RUN_REAL_RECOGNITION=1`). They are not part
of the shipped product. Each remains under its original license/rights holder — see provenance
below. If any rights holder objects, remove the file; the deterministic confidence logic is fully
covered by the unit/integration tests and does not depend on these images.

## Provenance

### ChessBase article ("Der Schachscanner – Partieneingabe leicht gemacht")
Source: https://de.chessbase.com/post/der-schachscanner-partieneingabe-leicht-gemacht
Images under https://de.chessbase.com/portals/all/2025/01/schachscanner/ — © ChessBase.

| local file | source file | notes |
|---|---|---|
| `cb_formulare-screenshot.jpg` | `formulare-screenshot.jpg` | montage of handwriting styles (not a single game) |
| `cb_formulare2.jpg` | `formulare2.jpg` | handwriting-style examples |
| `cb_Bericht_ZugRaster.jpg` | `Bericht ZugRaster.jpg` | app move-grid screenshot (printed) |

### Wikimedia Commons — Category:Chess score sheets
Source category: https://commons.wikimedia.org/wiki/Category:Chess_score_sheets
Downloaded via `Special:FilePath/<file>`. See each file's Commons page for its exact license.

| local file | Commons file |
|---|---|
| `wiki_chess_score_sheet.jpg` | File:Chess Score Sheet.jpg |
| `wiki_fischer_score_card.jpg` | File:Fischer Score Card.jpg |
| `wiki_eisenberg_capablanca.jpg` | File:Planilha Eisenberg e Capablanca.jpg |
| `wiki_carlsen.jpg` | File:Planilla Carlsen.jpg |
| `wiki_anotacion_027.jpg` | File:Anotación 027.jpg |

## Output

The harness writes a per-move confidence table per image (recognized text → SAN, legality,
confidence 0–1, and ok/verify/illegal state), plus per-fixture and total wall-clock timing:

- `RESULTS.txt` — Ollama `minicpm-v` run (`TestRealWorldConfidence`).
- `RESULTS_GEMINI.txt` — Gemini `gemini-2.5-flash` run (`TestRealWorldConfidenceGemini`,
  `RUN_REAL_RECOGNITION=1 GEMINI_API_KEY=… go test ./internal/recognition/ -run RealWorldConfidenceGemini -v`).

## Interpreting the recorded `RESULTS.txt`

The committed run used the local **`minicpm-v`** Ollama model on CPU — a small model that reads
real handwritten German score sheets poorly: it mostly emits `?` (illegible) or hallucinated
illegal moves, and two large scans truncated mid-JSON. The point this demonstrates is the
**correctness guarantee**, not model accuracy: every unreliable read is surfaced as `illegal`
(confidence `0.00`), so nothing bad reaches a saved PGN — exactly the review-loop invariant. The
full spectrum of states (green `ok` / yellow `verify` / red `illegal`) with real confidence
scores is exercised deterministically by the `fake` recognizer in the unit and e2e suites
(the ambiguous `Nd2` → `Nbd2` auto-pick at confidence `0.30` → `verify`).

## Ollama vs Gemini (`RESULTS_GEMINI.txt`)

Same 8 fixtures, same Reconcile/confidence logic — only the backend differs. Gemini Flash is
dramatically stronger at actually reading the moves and never crashed/truncated:

| fixture | Ollama (legal/total) | Gemini (legal/total) | Gemini time |
|---|---|---|---|
| `cb_Bericht_ZugRaster.jpg` | 0 / 32 | **34 / 116** | 19.4s |
| `cb_formulare-screenshot.jpg` | 0 / 38 | **14 / 14** (perfect) | 19.3s |
| `cb_formulare2.jpg` | 2 / 42 | **11 / 56** | 16.1s |
| `wiki_anotacion_027.jpg` | JSON-truncation error | 0 / 80 (illegible to both) | 7.4s |
| `wiki_carlsen.jpg` | 0 / 56 | 2 / 2 | 22.3s |
| `wiki_chess_score_sheet.jpg` | JSON-truncation error | 0 / 0 (empty) | 3.8s |
| `wiki_eisenberg_capablanca.jpg` | 0 / 44 | 0 / 58 (descriptive notation) | 13.0s |
| `wiki_fischer_score_card.jpg` | 0 / 38 | 0 / 2 (descriptive notation) | 19.8s |
| **total legal reads** | **2** | **61** | **2m1s wall** |

Two caveats on Gemini's remaining `illegal` rows — these are *not* misreads:

1. **Descriptive notation.** `wiki_eisenberg_capablanca` / `wiki_fischer` are old English/Spanish
   descriptive sheets (`P-K4`, `S-KB3`, `Q x P`). Gemini transcribed them faithfully, but
   `postprocess.go` only normalizes German SAN (S→N, L→B), so descriptive moves can't be made
   legal and are correctly surfaced as `illegal`. Supporting descriptive notation would be a
   separate postprocess feature.
2. **Move-order divergence.** On the long `cb_Bericht_ZugRaster` sheet the first 34 plies are a
   spot-on Ruy Lopez, then the read diverges (two-column ordering) and every later ply fails
   legality from that point — again surfaced, never saved.

The takeaway is unchanged and reinforced: regardless of backend strength, the review loop only
ever lets legal, reconciled moves through — Gemini just produces far more green `ok` rows to
start from.
