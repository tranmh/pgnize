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

The harness writes a per-move confidence table per image to `RESULTS.txt` in this directory
(recognized text → SAN, legality, confidence 0–1, and ok/verify/illegal state).
