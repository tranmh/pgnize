# Anonymous multi-image upload (Partieformular + Stellung scannen)

## Goal

In anonymous mode, let a user upload **two or more** pictures per submission for
both tools — "Partieformular umwandeln" (`/convert`, scoresheet → PGN) and
"Brettstellung scannen" (`/scan`, board photo → position). Today each tool
accepts exactly one image end-to-end.

## Decisions (from the user)

- **Both tools** get multi-image support; "second" means *two or more*.
- The extra images may be **more pages/angles of the same game/position** *or*
  effectively another — so all images of one submission are sent to the
  recognizer **together as a single multi-image request** producing **one
  combined result**. (Scoresheet: pages concatenated into one move list.
  Position: angles fused into one best board.)
- **Optional**: one image still works exactly as before (plus one submit click).
- **Rate limit unchanged**: one *submission* = one rate-limit unit regardless of
  image count ⇒ multi-image must be **one job**, never N jobs.

## Architecture: one job → many uploads, recognizer gets all images

### Data model (`migrations/0005_job_images.sql`)

- Keep `recognition_jobs.upload_id` as the **first / primary** image (NOT NULL,
  unchanged) — preserves `games.upload_id`, the review thumbnail, the account
  flow, and `ClaimNextJob`.
- New table for the **extra** images (idx ≥ 1):
  ```sql
  CREATE TABLE job_images (
      job_id    uuid NOT NULL REFERENCES recognition_jobs(id) ON DELETE CASCADE,
      upload_id uuid NOT NULL REFERENCES uploads(id) ON DELETE CASCADE,
      idx       int  NOT NULL,
      PRIMARY KEY (job_id, idx)
  );
  ```
  Single-image jobs have zero `job_images` rows — fully backward compatible.

### Store (`internal/store/jobs.go`)

- `CreateJob(... , extraUploadIDs []string)` — append param; insert the job then
  the `job_images` rows in one transaction. Existing callers pass `nil`.
- `JobExtraStorageKeys(ctx, jobID) ([]string, error)` — extra images' storage
  keys ordered by `idx`.

### Recognizer (`internal/recognition/recognizer.go` + gemini/ollama)

- Add `type ImageBlob struct { Data []byte; MimeType string }`.
- Add `Extra []ImageBlob` to `ScoreSheetInput` and `PositionInput`.
  **No change** to the `Recognize` / `RecognizePosition` signatures → `poseval`
  and all existing single-image tests are unaffected (`Extra` nil).
- Gemini: append one `geminiPart{InlineData}` per `Extra` (after the primary).
- Ollama: append each `Extra` (downscaled, base64) to the `Images` array.
- Fake: ignores images (already does).

### Worker (`internal/jobs/pipeline.go`)

- Replace `loadImage` with `loadImages` → primary (`job.StorageKey`) + each
  extra (`JobExtraStorageKeys` → `Storage.Get`). Build `Extra []ImageBlob`.
- Pass `Extra` into `ScoreSheetInput` / `PositionInput`.

### HTTP (`internal/httpapi/handlers_upload.go`, `_convert.go`, `_scan.go`)

- Refactor the per-file store logic into `storeFileHeader(...)`; add
  `storeImages(w, r, owner) ([]string, bool)` that iterates
  `r.MultipartForm.File["image"]` (≥1 required, cap at `maxImagesPerJob = 5`,
  each within `UploadMaxBytes`).
- `handleConvert` / `handleScan`: `ids := storeImages(...)`; `ids[0]` is the
  job's `upload_id`, `ids[1:]` are extras → `CreateJob(..., ids[1:])`.
  Rate-limit call unchanged (one per request). Response unchanged: `{jobId}`.
- `handleUpload` (account) stays single-image (`storeImage` unchanged).

### API client (`web/src/lib/api-client.ts`)

- `convert(images: File[], backend?)` / `scan(images: File[], backend?)` —
  append each file under the **same** `"image"` field. Response unchanged.

### Frontend UI

- `UploadDropzone`: add optional `resetAfterPick?: boolean`. When true, after a
  pick/capture it fires `onFile` then resets to idle so it can capture the next
  image. Default false → account upload page (`/upload`) unchanged.
- New `MultiImagePicker` (controlled, `{ files, onChange, disabled }`): renders a
  thumbnail strip (index badge + remove ×) and an `UploadDropzone resetAfterPick`
  as the "add another picture" control; manages object-URL lifecycle.
- `ConvertClient` / `ScanClient`: state `files: File[]`; upload stage shows
  `RecognizerSelect` + `MultiImagePicker` + a primary **Convert/Scan** button
  (disabled when `files.length === 0`); `start(files)` calls `convert/scan`.
  `reset()` clears `files`. (New: explicit submit step — one extra click for the
  single-image case, clearer for multi.)

### i18n (`web/src/i18n/messages.ts`) — add to BOTH `en` and `de`

`multiupload.add`, `multiupload.remove`, `multiupload.imageLabel` (`{n}`),
`multiupload.hint` ("Add more pages or angles — optional"), `multiupload.empty`,
`convert.submit` ("Convert"), `scan.submit` ("Scan").

## UPDATE — two selectable modes (combine vs separate)

The user wants to handle BOTH "same game/position over multiple pictures" AND
"separate games/positions in one upload". Auto-detection isn't reliable, so the
upload screen carries a **mode toggle**:

- **Combine** ("one game" / "one position"): all pictures → ONE request
  (`/convert` or `/scan` with N `"image"` parts) → one job → one combined result.
  This is the `job_images` backend path. One rate-limit unit per submission.
- **Separate** ("separate games" / "separate positions"): the frontend fires
  ONE request PER picture → N jobs → N results rendered as a list. **No backend
  change** — reuses the existing single-image endpoint. N rate-limit units.

### Frontend refactor for N results

- Extract per-job result components: `ConvertJobResult` (owns its `useJobPoller`
  → `getConvertGame` → `ReviewWorkbench` → `exportConvertPgn`) and
  `ScanJobResult` (poller → `getScanGame` → `PositionReview` → `exportScanPgn`).
- `ConvertClient`/`ScanClient` keep `files: File[]`, a `mode: "combine"|"separate"`,
  and `jobIds: string[]`. On submit:
  - combine → `convert(files)` / `scan(files)` → `[jobId]`
  - separate → `Promise.all(files.map(f => convert([f])))` → `jobIds`
  Then render one result block per `jobId` (combine ⇒ exactly one).
- Mode toggle UI sits in the upload stage under `MultiImagePicker`, only shown
  when `files.length > 1` (a single picture has no mode to choose).

### Extra i18n (both en + de)

`multiupload.modePrompt` ("These pictures are:"),
`convert.modeCombine` ("One game (extra pages)"), `convert.modeSeparate`
("Separate games"), `scan.modeCombine` ("One position (extra angles)"),
`scan.modeSeparate` ("Separate positions").

## UPDATE — registered (account) flow gets the same support

Applied multi-image + the two modes to the logged-in `/upload` flow too:

- **Backend**: `handleUpload` now uses `storeImages` (1..5 `"image"` fields) and
  passes `ids[1:]` as extras to `CreateJob` — combine mode is one job with extras
  (same as anon). `storeImage` (single-file wrapper) removed as now-unused.
- **API client**: `upload(images: File[], consent, backend?, kind?)`.
- **UI** (`web/src/app/upload/page.tsx`): `MultiImagePicker` + `UploadModeToggle`
  (labels reuse `convert.*`/`scan.*` mode keys by the selected `kind`). One job
  (single image OR combine) keeps the existing **auto-redirect** to the review
  screen; a **separate** submission of N pictures lists each as a row with a
  "Review →" link to its own `/review/[gameId]` or `/scan/review/[gameId]`
  (new `UploadJobRow` component, new keys `upload.recognized`/`upload.reviewLink`).
- **Test**: `TestAccountUploadMultiImage` (integration) — a 3-image `/uploads`
  submission creates one job with 2 extras.

The per-game save-to-library review screens are unchanged and reused as-is.

## Out of scope

- Review screen still shows the **primary** image only (combined result is what
  matters). Multi-image gallery in review is a later nice-to-have.
- Account `/upload` flow stays single-image.

## Tests (TDD — run them)

- Go unit: recognizer includes `Extra` images in Gemini/Ollama request bodies
  (httptest); `storeImages` min/max/limit; pipeline `loadImages` order; `Process`
  forwards `Extra` (spy recognizer).
- Go integration (DB): `CreateJob` with extras writes `job_images`;
  `JobExtraStorageKeys` ordering.
- Web unit (vitest): `convert`/`scan` append N `"image"` parts;
  `MultiImagePicker` add/remove → `onChange`.
- Gates: `make test-go`, `make lint`, `make build-web` (and `make test-int` if a
  test DB is available).
```
