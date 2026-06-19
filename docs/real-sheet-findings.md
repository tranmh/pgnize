# Real-world recognition test — findings

Exercised the **real** local VLM recognizer (`internal/recognition/ollama.go`) against
real German/historical chess score-sheet photos pulled from the public web, on a
**CPU-only** box. Goal: find bugs and measure performance. Harness:
`internal/recognition/ollama_real_test.go` (build tag `ollama`, opt-in).

## Test corpus

9 images in `testdata/real-sheets/` (see `SOURCES.md` for URLs + licenses):
5 handwritten (historical: Capablanca, Fischer, Carlsen photos), 1 didactic filled,
2 **blank German Partieformulare** (exact target layout), 1 Polish blank. Sizes range
from 572×787 up to 4928×3264 (16 MP, 4 MB).

> Caveat: no **openly-licensed photo of a filled-in German** Partieformular exists that we
> could find — the handwritten opens are historical/English-notation; the genuinely German
> sheets available openly are blank templates. Good enough to shake out pipeline bugs and
> measure latency; not a recognition-accuracy benchmark.

## Bugs found & fixed

| # | Severity | Bug | Fix |
|---|----------|-----|-----|
| 1 | **Critical** | `format: <json-schema>` constrained decoding + **no `num_predict` cap** caused runaway generation. minicpm-v never returned within the 10-min HTTP timeout on a single 572×787 image — the recognizer was effectively unusable. | Switched to `format: "json"` (simple JSON mode) and added a `num_predict` cap. Direct A/B: full-schema ran to 512 tokens in **160 s** (and unbounded otherwise); `json` mode returned valid JSON and **stopped naturally at 120 tokens in 37 s**. |
| 2 | **High** | Model returns the `"no"` move-number field inconsistently as an int *or* a string (`1` vs `"1"`). The rigid `No int` struct field made `json.Unmarshal` fail the **entire** result → whole job failed. | Dropped `No` from the decode struct (we key off `white`/`black` text and derive ply ourselves). |
| 3 | **High** | Hardcoded 10-min client timeout — both far too long for UX and the cause of #1 hiding. | Configurable timeout (default 5 min) + the token cap keeps real latency to ~2–3 min worst case. |
| 4 | **Medium** | No image downscaling. 16 MP / 4 MB photos were sent at full resolution (slow upload + the model tiles huge images). | Added `downscale()` — longest edge capped at 1600 px, re-encoded JPEG q85, decode-error-safe (returns original). |
| 5 | **Low (robustness)** | On invalid model JSON the recognizer returned a bare error, losing the raw text. | Now returns the raw response in `RawJSON` alongside the error so a job records *something* for debugging/review. |
| 6 | **Medium** | The `num_predict` cap can truncate the JSON mid-array → `unexpected end of JSON input` → the **entire** transcription is lost (observed on 2 of 3 dense sheets). | Added `salvageMoves()`: on decode failure it extracts the complete `{...}` move objects already emitted (scoped to inside the `"moves"` array so the header's white/black player names aren't mistaken for moves) and keeps the partial transcription. Unit-tested. |

## Performance (CPU, minicpm-v 8B Q4, ~3 tok/s)

- Model load into RAM (first call): one-time ~seconds (5.5 GB resident).
- `prompt_eval` for one sheet image: ~3 s (≈350 image tokens — the model down-samples internally).
- **Generation dominates**: latency ≈ `num_predict × ~0.3 s`. Hence the 512-token cap.
Latency after the fixes (minicpm-v, `num_predict=512`, downscale≤1600px), 3-image run,
**avg 3m08s/image, no timeouts** (vs. the previous 10-min hard timeout):

| image | dimensions | latency | outcome |
|-------|-----------|---------|---------|
| sheet-01.jpg | 572×787 | 3m32s | JSON truncated at the cap → recoverable via salvage (#6) |
| sheet-02.jpg | 1708×1200 | 2m04s | ✅ parsed; header read correctly (`Event=Olympiade, Site=Siegen, Date=September 1970`), a few legible moves, rest `?` |
| sheet-03.jpg | 4928×3264 → 1600 | 3m48s | JSON truncated at the cap → recoverable via salvage (#6) |

(The salvage fix #6 was added after this measurement run; it is unit-tested and converts the
two "truncated → total loss" cases into partial transcriptions.)

### moondream (1.7 B) comparison
~49 s/image but output is hallucinated ("White king, Black queen…") — too weak to read
handwritten chess notation. Useful only as a fast smoke recognizer.

## Conclusions

- **CPU is viable only with a small token cap and a fast/forgiving model.** minicpm-v is the
  best small option but still ~2–3 min/sheet on CPU — which is exactly why the architecture
  runs recognition as an **async job** with status polling and why the **manual review loop**
  carries the correctness guarantee (model output on these hard historical sheets is poor).
- For production accuracy, a GPU (or a hosted vision API behind the same `Recognizer`
  interface) is recommended; the interface makes that a config swap.
- The fixes above make the local recognizer actually usable on CPU instead of timing out.
