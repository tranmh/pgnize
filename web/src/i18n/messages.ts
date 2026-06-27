// Translation catalog. German ("de") is the default UI language; English ("en")
// is the alternative, toggled via the language switcher in the nav. Keys are flat,
// dot-namespaced strings. {placeholders} are substituted at render time by `t()`.
//
// To add a string: add the same key to BOTH dictionaries. A key missing from the
// active locale falls back to German, then to the raw key.

export type Locale = "de" | "en";

export const LOCALES: Locale[] = ["de", "en"];
export const DEFAULT_LOCALE: Locale = "de";

export const LOCALE_LABELS: Record<Locale, string> = {
  de: "DE",
  en: "EN",
};

type Catalog = Record<string, string>;

const en: Catalog = {
  // Common
  "common.email": "Email",
  "common.password": "Password",
  "common.name": "Name",
  "common.loading": "Loading…",
  "common.loadingGame": "Loading game…",
  "common.backToLibrary": "Back to library",

  // Nav
  "nav.convert": "Convert",
  "nav.scan": "Scan position",
  "nav.new": "Analyze & coach",
  "nav.library": "Library",
  "nav.signOut": "Sign out",
  "nav.login": "Log in",
  "nav.register": "Register",

  // Landing page (marketing)
  "landing.hero.eyebrow": "Score sheet & board → digital chess",
  "landing.hero.title": "Turn a photo of your game into PGN",
  "landing.hero.subtitle":
    "Snap a handwritten score sheet or a board position. PGNize reads it, you verify every move, and you download clean, engine-checked PGN. No typing.",
  "landing.hero.ctaConvert": "Convert a score sheet",
  "landing.hero.ctaScan": "Scan a position",
  "landing.hero.ctaCoach": "Analyze & get coaching",
  "landing.hero.free": "Free to try — no account needed.",

  // Landing: feature 1 (score sheet → PGN)
  "landing.f1.tag": "Handwritten score sheet → PGN",
  "landing.f1.title": "Read a whole game from one photo",
  "landing.f1.body":
    "Upload a photo of a German Partieformular (or any score sheet). PGNize transcribes the moves and replays them through a real chess engine, so what you download is guaranteed legal.",
  "landing.f1.inputCaption": "Photo of a handwritten score sheet",
  "landing.f1.outputCaption": "Verified, downloadable PGN",
  "landing.f1.cta": "Try it with your own sheet →",

  // Landing: feature 2 (board photo → position)
  "landing.f2.tag": "Board photo → editable position",
  "landing.f2.title": "Capture any position in seconds",
  "landing.f2.body":
    "Point your camera at a physical board or a screen. PGNize recognizes the position as FEN and hands you an editable board to fix anything before exporting.",
  "landing.f2.inputCaption": "Photo of a real board",
  "landing.f2.outputCaption": "Recognized position (FEN)",
  "landing.f2.cta": "Try it with your own board →",

  // Landing: how it works
  "landing.how.title": "How it works",
  "landing.how.step1.title": "1 · Snap a photo",
  "landing.how.step1.body":
    "A score sheet for a full game, or a board for a single position. Phone photos are fine.",
  "landing.how.step2.title": "2 · We recognize it",
  "landing.how.step2.body":
    "Moves and pieces are read automatically and checked against the rules of chess.",
  "landing.how.step3.title": "3 · You verify & download",
  "landing.how.step3.body":
    "Review highlighted uncertain moves, correct anything, and export clean PGN.",

  // Landing: example/credit labels
  "landing.exampleBadge": "Real example",
  "landing.input": "Input",
  "landing.output": "Output",
  "landing.boardCredit":
    "Board photo: Chess Recognition Dataset (ChessReD), CC BY-NC-SA 4.0.",

  // Login
  "login.title": "Log in",
  "login.submit": "Log in",
  "login.submitting": "Signing in…",
  "login.errInvalid": "Incorrect email or password.",
  "login.errGeneric": "Login failed.",
  "login.noAccount": "No account?",

  // Register
  "register.title": "Create an account",
  "register.submit": "Register",
  "register.submitting": "Creating…",
  "register.errConflict": "That email is already registered.",
  "register.errGeneric": "Registration failed.",
  "register.haveAccount": "Already have an account?",

  // Upload page
  "upload.title": "New game from photo",
  "upload.subtitle":
    "Upload a score-sheet photo. After recognition you'll review and verify the moves before saving to your library.",
  "upload.autoRedirect": "You'll be taken to the review screen automatically.",
  "upload.consent":
    "Allow this image to improve recognition. Your photo and corrected transcription may be used to train the model. You can leave this unchecked; non-consented uploads are deleted automatically.",
  "upload.submit": "Recognize",
  "upload.submitting": "Uploading…",
  "upload.errRateLimit": "Rate limit reached. Please wait and try again.",
  "upload.errGeneric": "Upload failed.",
  "upload.kind.label": "What's in the photo?",
  "upload.kind.scoresheet": "Scoresheet (moves)",
  "upload.kind.scan": "Board position",
  "upload.recognized": "Recognized {n} item(s). Open each to review and save.",
  "upload.reviewLink": "Review →",

  // Convert (anonymous) flow
  "convert.submit": "Convert",
  "convert.modeCombine": "One game (extra pages)",
  "convert.modeSeparate": "Separate games",
  "convert.resultLabel": "Game {n}",
  "convert.title": "Convert a score sheet",
  "convert.subtitle":
    "Upload a photo of a handwritten chess score sheet. We'll read it, and you verify the moves before downloading the PGN.",
  "convert.takesMinutes": "This can take up to a few minutes.",
  "convert.loadingRecognized": "Loading recognized game…",
  "convert.downloadPgn": "Download PGN",
  "convert.confidence": "Recognition confidence: {pct}%",
  "convert.errTitle": "Something went wrong",
  "convert.tryAgain": "Try again",
  "convert.errRateLimit": "Rate limit reached. Please wait a moment and try again.",
  "convert.errLoadGame": "Could not load the game.",
  "convert.errExport": "Export failed.",
  "convert.errUpload": "Upload failed.",

  // Scan (anonymous board photo -> position) flow
  "scan.submit": "Scan",
  "scan.modeCombine": "One position (extra angles)",
  "scan.modeSeparate": "Separate positions",
  "scan.resultLabel": "Position {n}",
  "scan.title": "Scan a board position",
  "scan.subtitle":
    "Upload a photo of a chess board. We'll recognize the position; you correct it on the editable board before downloading the PGN.",
  "scan.takesSeconds": "This usually takes a few seconds.",
  "scan.loadingRecognized": "Loading recognized position…",
  "scan.downloadPgn": "Download PGN",
  "scan.confidence": "Recognition confidence: {pct}%",
  "scan.errTitle": "Something went wrong",
  "scan.tryAgain": "Try again",
  "scan.errRateLimit": "Rate limit reached. Please wait a moment and try again.",
  "scan.errLoadGame": "Could not load the position.",
  "scan.errExport": "Export failed.",
  "scan.errUpload": "Upload failed.",
  "scan.promoPrefix": "Have a photo of a board instead of a score sheet?",
  "scan.promoLink": "Scan a position →",

  // Position editor
  "editor.palette": "Pieces",
  "editor.tool.erase": "Erase",
  "editor.clearBoard": "Clear board",
  "editor.startingPosition": "Starting position",
  "editor.flip": "Flip board",
  "editor.sideToMove": "Side to move",
  "editor.white": "White",
  "editor.black": "Black",
  "editor.castling": "Castling",
  "editor.castling.K": "White O-O",
  "editor.castling.Q": "White O-O-O",
  "editor.castling.k": "Black O-O",
  "editor.castling.q": "Black O-O-O",
  "editor.enPassant": "En passant",
  "editor.enPassant.none": "none",
  "editor.illegalWarning":
    "This position may be illegal. The PGN will still be created, but check it carefully.",
  "editor.placeHint":
    "Pick a piece, then click a square to place it. Drag a piece to move it.",

  // Anonymous banner
  "anon.prefix": "Anonymous conversions are",
  "anon.notSaved": "not saved",
  "anon.middle": "to a library.",
  "anon.createAccount": "Create an account",
  "anon.or": "or",
  "anon.login": "log in",
  "anon.suffix": "to keep a searchable history of your games.",

  // Recognition status / errors (shared)
  "recog.reading": "Reading handwriting…",
  "recog.queued": "Queued for recognition…",
  "recog.failed": "Recognition failed.",
  "recog.timeout": "Timed out waiting for recognition to finish.",

  // Recognizer select
  "recognizer.label": "Recognition engine",

  // Upload dropzone
  "dropzone.changePhoto": "click to choose a different photo",
  "dropzone.dragHere": "Drag a photo of the score sheet here, or",
  "dropzone.browse": "browse",
  "dropzone.aria": "Upload a score-sheet photo",
  "dropzone.selectedAlt": "Selected score sheet",
  "dropzone.takePhoto": "Tap to take a photo",
  "dropzone.capture": "Take photo",
  "dropzone.cancel": "Cancel",
  "dropzone.retake": "tap to retake",
  "dropzone.switchCamera": "Flip",
  "dropzone.cameraStarting": "Starting camera…",
  "dropzone.cameraError":
    "Could not access the camera. Please allow camera access or upload a file instead.",
  "dropzone.orUpload": "or upload a file instead",

  // Multi-image picker
  "multiupload.add": "Add another picture",
  "multiupload.remove": "Remove",
  "multiupload.imageLabel": "Image {n}",
  "multiupload.hint": "Add more pages or angles — optional.",
  "multiupload.modePrompt": "These pictures are:",
  "multiupload.startOver": "Start over",
  "multiupload.someRejected":
    "{n} picture(s) were not accepted (rate limit). Showing the rest.",

  // Library
  "library.title": "Library",
  "library.newFromPhoto": "New from photo",
  "library.enterManually": "Enter manually",
  "library.selected": "{n} selected",
  "library.downloadBundle": "Download bundle PGN",
  "library.clear": "Clear",
  "library.loadingGames": "Loading games…",
  "library.empty": "No saved games yet. Convert a photo or enter one manually to start.",
  "library.colWhite": "White",
  "library.colBlack": "Black",
  "library.colEvent": "Event",
  "library.colDate": "Date",
  "library.colResult": "Result",
  "library.colMoves": "Moves",
  "library.colActions": "Actions",
  "library.view": "View",
  "library.open": "Open",
  "library.pgn": "PGN",
  "library.previous": "Previous",
  "library.next": "Next",
  "library.pageOf": "Page {page} of {total}",
  "library.errLoad": "Could not load games.",
  "library.errDownload": "Download failed.",
  "library.errBundle": "Bundle export failed.",
  "library.errDraft": "Could not create a draft.",
  "library.selectAria": "Select {white} vs {black}",
  "library.searchPlaceholder": "Search…",
  "library.searchAria": "Search games",
  "library.playerPlaceholder": "Player",
  "library.playerAria": "Filter by player",
  "library.eventPlaceholder": "Event",
  "library.eventAria": "Filter by event",
  "library.fromPlaceholder": "From (YYYY.MM.DD)",
  "library.fromAria": "From date",
  "library.toPlaceholder": "To (YYYY.MM.DD)",
  "library.toAria": "To date",
  "library.apply": "Apply",

  // Header fields
  "header.white": "White",
  "header.black": "Black",
  "header.event": "Event",
  "header.site": "Site",
  "header.date": "Date (YYYY.MM.DD)",
  "header.round": "Round",
  "header.board": "Board",
  "header.result": "Result",
  "header.whitePlayer": "White player",
  "header.blackPlayer": "Black player",

  // Move list
  "moves.title": "Moves",
  "moves.start": "start",
  "moves.jumpStartAria": "Jump to starting position",
  "moves.none": "No moves yet.",
  "moves.addByDragging": "Add a move by dragging a piece on the board.",
  "moves.legal": "legal",
  "moves.illegal": "illegal",
  "moves.ambiguous": "ambiguous",
  "moves.readAs": "read as “{text}”",
  "moves.corrected": "(corrected)",
  "moves.didYouMean": "Did you mean",
  "moves.disambiguate": "Disambiguate",
  "moves.otherLegal": "Other legal moves",
  "moves.choose": "choose…",
  "moves.sanPlaceholder": "SAN, e.g. Nf3",
  "moves.editAria": "Edit move SAN",
  "moves.clickToView": "Click to view, again to edit",
  "moves.showPositionAria": "Show position after {label} {san}",
  "moves.markUnreadable": "Mark as unreadable placeholder",
  "moves.insertAfter": "Insert a move after this one",
  "moves.truncate": "Truncate game here (delete this and all later moves)",
  "moves.delete": "Delete this move",
  "moves.verify": "verify",
  "moves.verifyHint": "Recognized with low confidence — compare with the sheet and confirm.",

  // Engine controls
  "engine.unavailable": "Engine unavailable in this browser.",
  "engine.eval": "Engine eval",
  "engine.analyzing": "Analyzing… {pct}% (stop)",
  "engine.clear": "Clear analysis",
  "engine.analyze": "Analyze game",

  // Game viewer
  "viewer.vs": "vs",
  "viewer.round": "Round {n}",
  "viewer.board": "Board {n}",
  "viewer.flip": "Flip board",
  "viewer.stepCaption": "Use ◀ ▶ or the arrow keys to step through the game.",

  // Review workbench
  "review.manualNoPhoto": "Manual entry — no photo.",
  "review.edit": "Edit",
  "review.view": "View",
  "review.captionView": "Use ◀ ▶ or arrow keys to step through the game.",
  "review.captionEdit":
    "Drag a piece to add or replace the selected move. Click a move to view its position.",
  "review.serverRejected": "Server rejected move #{n} as illegal. Fix it and try again.",
  "review.working": "Working…",
  "review.resolveIllegal":
    "Resolve all illegal/ambiguous moves (or truncate) to continue.",
  "review.toVerify": "{n} to verify",
  "review.allVerified": "all checked",
  "review.nextUncertain": "Next to verify",
  "review.unverifiedNote":
    "Some moves were read with low confidence — review the highlighted ones against the sheet before saving.",

  // Photo viewer
  "photo.title": "Photo",
  "photo.zoomOut": "Zoom out",
  "photo.zoomIn": "Zoom in",
  "photo.reset": "Reset view",
  "photo.alt": "Score sheet",

  // Move nav
  "movenav.start": "Start position",
  "movenav.prev": "Previous move",
  "movenav.next": "Next move",
  "movenav.last": "Last move",

  // Review page
  "reviewPage.title": "Review",
  "reviewPage.saveGame": "Save game",
  "reviewPage.saved": "Saved.",
  "reviewPage.viewGame": "View game",
  "reviewPage.goToLibrary": "go to library",
  "reviewPage.saveFailed": "Save failed.",
  "reviewPage.couldNotLoad": "Could not load this game",
  "reviewPage.loadError": "Could not load game.",

  // View page
  "viewPage.title": "View game",
  "viewPage.edit": "Edit",

  // New / analyze-&-coach page
  "new.title": "Analyze & get coaching",
  "new.subtitle":
    "Paste a FEN or import a game (PGN or a Lichess study/game URL). The engine evaluates it in your browser, then a coach explains the why in plain words.",
  "new.resultSubtitle":
    "Run the engine, then click Explain on a move or Coach this game for a summary.",
  "new.resultSubtitlePosition":
    "Toggle the engine eval to see the assessment, then click Coach this position for an explanation.",
  "new.modeFen": "Paste FEN",
  "new.modeImport": "Import PGN / Lichess",
  "new.fenLabel": "FEN",
  "new.importLabel": "PGN or Lichess URL",
  "new.fenPlaceholder": "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
  "new.importPlaceholder":
    "Paste PGN here, or a Lichess study/game URL (https://lichess.org/…)",
  "new.submit": "Load",
  "new.working": "Loading…",
  "new.errEmpty": "No game found in that input.",
  "new.errInvalid": "That doesn't look like a valid position or game.",
  "new.errRateLimit": "Rate limit reached. Please wait a moment and try again.",
  "new.errGeneric": "Could not load that input.",
  "new.gameLabel": "Game {n}",
  "new.startOver": "Start over",
  "new.save": "Save to library",
  "new.downloadPgn": "Download PGN",
  "new.saved": "Saved to your library.",
  "new.viewGame": "View game",
  "new.promoPrefix": "Have a photo of a score sheet instead?",
  "new.promoConvert": "Convert a photo →",

  // Coach (engine → LLM explanation)
  "coach.title": "Coach",
  "coach.thinking": "Coach is thinking…",
  "coach.error": "Coaching failed. Please try again.",
  "coach.gameSummary": "Game summary",
  "coach.explain": "Explain",
  "coach.explained": "Explained ✓",
  "coach.explainHint": "Ask the coach why this move helps or hurts.",
  "coach.coachGame": "Coach this game",
  "coach.coachPosition": "Coach this position",
  "coach.play": "Play",
  "coach.stop": "Stop",
  "coach.replay": "Replay",

  // Text-to-speech (global toggle + source)
  "tts.on": "Speech on",
  "tts.off": "Speech off",
  "tts.sourceLabel": "Speech source",
  "tts.source.server": "Server",
  "tts.source.browser": "Browser",

  // Conversational coach (ask the coach by typing or speaking)
  "chat.title": "Ask the coach",
  "chat.open": "Ask the coach",
  "chat.close": "Close",
  "chat.placeholder": "Ask about this position…",
  "chat.send": "Send",
  "chat.thinking": "The coach is thinking…",
  "chat.error": "The coach could not answer. Please try again.",
  "chat.empty": "Ask anything about the position — best move, why a move is bad, mate, plans…",
  "chat.you": "You",
  "chat.coach": "Coach",
  "chat.mic.start": "Speak your question",
  "chat.mic.stop": "Stop recording",
  "chat.mic.recording": "Recording…",
  "chat.mic.listening": "Listening…",
  "chat.mic.denied": "Microphone access was denied. You can still type your question.",
  "chat.mic.unavailable": "Voice input is unavailable in this browser. Please type your question.",

  // Speech-to-text source
  "stt.sourceLabel": "Voice input",
  "stt.source.server": "Server",
  "stt.source.browser": "Browser",
};

const de: Catalog = {
  // Common
  "common.email": "E-Mail",
  "common.password": "Passwort",
  "common.name": "Name",
  "common.loading": "Wird geladen …",
  "common.loadingGame": "Partie wird geladen …",
  "common.backToLibrary": "Zurück zur Bibliothek",

  // Nav
  "nav.convert": "Umwandeln",
  "nav.scan": "Stellung scannen",
  "nav.new": "Analysieren & coachen",
  "nav.library": "Bibliothek",
  "nav.signOut": "Abmelden",
  "nav.login": "Anmelden",
  "nav.register": "Registrieren",

  // Landing page (Marketing)
  "landing.hero.eyebrow": "Partieformular & Brett → digitales Schach",
  "landing.hero.title": "Mach aus einem Foto deiner Partie ein PGN",
  "landing.hero.subtitle":
    "Fotografiere ein handgeschriebenes Partieformular oder eine Brettstellung. PGNize liest es aus, du prüfst jeden Zug und lädst sauberes, engine-geprüftes PGN herunter. Kein Abtippen.",
  "landing.hero.ctaConvert": "Partieformular umwandeln",
  "landing.hero.ctaScan": "Stellung scannen",
  "landing.hero.ctaCoach": "Analysieren & coachen lassen",
  "landing.hero.free": "Kostenlos ausprobieren – kein Konto nötig.",

  // Landing: Funktion 1 (Partieformular → PGN)
  "landing.f1.tag": "Handgeschriebenes Partieformular → PGN",
  "landing.f1.title": "Eine ganze Partie aus einem Foto",
  "landing.f1.body":
    "Lade ein Foto eines Partieformulars hoch. PGNize überträgt die Züge und spielt sie durch eine echte Schach-Engine, sodass dein Download garantiert legal ist.",
  "landing.f1.inputCaption": "Foto eines handgeschriebenen Partieformulars",
  "landing.f1.outputCaption": "Geprüftes PGN zum Herunterladen",
  "landing.f1.cta": "Mit eigenem Formular ausprobieren →",

  // Landing: Funktion 2 (Brettfoto → Stellung)
  "landing.f2.tag": "Brettfoto → bearbeitbare Stellung",
  "landing.f2.title": "Jede Stellung in Sekunden erfassen",
  "landing.f2.body":
    "Richte deine Kamera auf ein echtes Brett oder einen Bildschirm. PGNize erkennt die Stellung als FEN und gibt dir ein bearbeitbares Brett, um vor dem Export alles zu korrigieren.",
  "landing.f2.inputCaption": "Foto eines echten Bretts",
  "landing.f2.outputCaption": "Erkannte Stellung (FEN)",
  "landing.f2.cta": "Mit eigenem Brett ausprobieren →",

  // Landing: So funktioniert's
  "landing.how.title": "So funktioniert's",
  "landing.how.step1.title": "1 · Foto machen",
  "landing.how.step1.body":
    "Ein Partieformular für eine ganze Partie oder ein Brett für eine einzelne Stellung. Handyfotos genügen.",
  "landing.how.step2.title": "2 · Wir erkennen es",
  "landing.how.step2.body":
    "Züge und Figuren werden automatisch gelesen und gegen die Schachregeln geprüft.",
  "landing.how.step3.title": "3 · Du prüfst & lädst herunter",
  "landing.how.step3.body":
    "Prüfe markierte unsichere Züge, korrigiere bei Bedarf und exportiere sauberes PGN.",

  // Landing: Beispiel-/Credit-Beschriftungen
  "landing.exampleBadge": "Echtes Beispiel",
  "landing.input": "Eingabe",
  "landing.output": "Ausgabe",
  "landing.boardCredit":
    "Brettfoto: Chess Recognition Dataset (ChessReD), CC BY-NC-SA 4.0.",

  // Login
  "login.title": "Anmelden",
  "login.submit": "Anmelden",
  "login.submitting": "Anmeldung läuft …",
  "login.errInvalid": "E-Mail oder Passwort ist falsch.",
  "login.errGeneric": "Anmeldung fehlgeschlagen.",
  "login.noAccount": "Noch kein Konto?",

  // Register
  "register.title": "Konto erstellen",
  "register.submit": "Registrieren",
  "register.submitting": "Wird erstellt …",
  "register.errConflict": "Diese E-Mail ist bereits registriert.",
  "register.errGeneric": "Registrierung fehlgeschlagen.",
  "register.haveAccount": "Bereits ein Konto?",

  // Upload page
  "upload.title": "Neue Partie aus Foto",
  "upload.subtitle":
    "Lade ein Foto des Partieformulars hoch. Nach der Erkennung prüfst und bestätigst du die Züge, bevor sie in deiner Bibliothek gespeichert werden.",
  "upload.autoRedirect": "Du wirst automatisch zur Prüfungsansicht weitergeleitet.",
  "upload.consent":
    "Dieses Bild zur Verbesserung der Erkennung freigeben. Dein Foto und die korrigierte Übertragung können zum Training des Modells verwendet werden. Du kannst dies frei lassen; nicht freigegebene Uploads werden automatisch gelöscht.",
  "upload.submit": "Erkennen",
  "upload.submitting": "Wird hochgeladen …",
  "upload.errRateLimit": "Ratenlimit erreicht. Bitte warte kurz und versuche es erneut.",
  "upload.errGeneric": "Upload fehlgeschlagen.",
  "upload.kind.label": "Was ist auf dem Foto?",
  "upload.kind.scoresheet": "Partieformular (Züge)",
  "upload.kind.scan": "Brettstellung",
  "upload.recognized":
    "{n} Element(e) erkannt. Öffne jedes zum Prüfen und Speichern.",
  "upload.reviewLink": "Prüfen →",

  // Convert (anonymous) flow
  "convert.submit": "Umwandeln",
  "convert.modeCombine": "Eine Partie (weitere Seiten)",
  "convert.modeSeparate": "Separate Partien",
  "convert.resultLabel": "Partie {n}",
  "convert.title": "Partieformular umwandeln",
  "convert.subtitle":
    "Lade ein Foto eines handgeschriebenen Partieformulars hoch. Wir lesen es aus, und du prüfst die Züge, bevor du das PGN herunterlädst.",
  "convert.takesMinutes": "Das kann einige Minuten dauern.",
  "convert.loadingRecognized": "Erkannte Partie wird geladen …",
  "convert.downloadPgn": "PGN herunterladen",
  "convert.confidence": "Erkennungssicherheit: {pct} %",
  "convert.errTitle": "Etwas ist schiefgelaufen",
  "convert.tryAgain": "Erneut versuchen",
  "convert.errRateLimit":
    "Ratenlimit erreicht. Bitte warte einen Moment und versuche es erneut.",
  "convert.errLoadGame": "Die Partie konnte nicht geladen werden.",
  "convert.errExport": "Export fehlgeschlagen.",
  "convert.errUpload": "Upload fehlgeschlagen.",

  // Scan (anonymes Brettfoto -> Stellung) Ablauf
  "scan.submit": "Scannen",
  "scan.modeCombine": "Eine Stellung (weitere Perspektiven)",
  "scan.modeSeparate": "Separate Stellungen",
  "scan.resultLabel": "Stellung {n}",
  "scan.title": "Brettstellung scannen",
  "scan.subtitle":
    "Lade ein Foto eines Schachbretts hoch. Wir erkennen die Stellung; du korrigierst sie auf dem bearbeitbaren Brett, bevor du das PGN herunterlädst.",
  "scan.takesSeconds": "Das dauert in der Regel einige Sekunden.",
  "scan.loadingRecognized": "Erkannte Stellung wird geladen …",
  "scan.downloadPgn": "PGN herunterladen",
  "scan.confidence": "Erkennungssicherheit: {pct} %",
  "scan.errTitle": "Etwas ist schiefgelaufen",
  "scan.tryAgain": "Erneut versuchen",
  "scan.errRateLimit":
    "Ratenlimit erreicht. Bitte warte einen Moment und versuche es erneut.",
  "scan.errLoadGame": "Die Stellung konnte nicht geladen werden.",
  "scan.errExport": "Export fehlgeschlagen.",
  "scan.errUpload": "Upload fehlgeschlagen.",
  "scan.promoPrefix": "Hast du ein Foto eines Bretts statt eines Partieformulars?",
  "scan.promoLink": "Stellung scannen →",

  // Stellungseditor
  "editor.palette": "Figuren",
  "editor.tool.erase": "Löschen",
  "editor.clearBoard": "Brett leeren",
  "editor.startingPosition": "Grundstellung",
  "editor.flip": "Brett drehen",
  "editor.sideToMove": "Am Zug",
  "editor.white": "Weiß",
  "editor.black": "Schwarz",
  "editor.castling": "Rochade",
  "editor.castling.K": "Weiß O-O",
  "editor.castling.Q": "Weiß O-O-O",
  "editor.castling.k": "Schwarz O-O",
  "editor.castling.q": "Schwarz O-O-O",
  "editor.enPassant": "En passant",
  "editor.enPassant.none": "keine",
  "editor.illegalWarning":
    "Diese Stellung ist möglicherweise illegal. Das PGN wird trotzdem erstellt, prüfe es aber sorgfältig.",
  "editor.placeHint":
    "Wähle eine Figur und klicke dann auf ein Feld, um sie zu setzen. Ziehe eine Figur, um sie zu bewegen.",

  // Anonymous banner
  "anon.prefix": "Anonyme Umwandlungen werden",
  "anon.notSaved": "nicht gespeichert",
  "anon.middle": "in einer Bibliothek.",
  "anon.createAccount": "Erstelle ein Konto",
  "anon.or": "oder",
  "anon.login": "melde dich an",
  "anon.suffix": ", um einen durchsuchbaren Verlauf deiner Partien zu behalten.",

  // Recognition status / errors (shared)
  "recog.reading": "Handschrift wird gelesen …",
  "recog.queued": "In der Warteschlange für die Erkennung …",
  "recog.failed": "Erkennung fehlgeschlagen.",
  "recog.timeout": "Zeitüberschreitung beim Warten auf die Erkennung.",

  // Recognizer select
  "recognizer.label": "Erkennungs-Engine",

  // Upload dropzone
  "dropzone.changePhoto": "klicken, um ein anderes Foto zu wählen",
  "dropzone.dragHere": "Ziehe ein Foto des Partieformulars hierher oder",
  "dropzone.browse": "durchsuchen",
  "dropzone.aria": "Foto eines Partieformulars hochladen",
  "dropzone.selectedAlt": "Ausgewähltes Partieformular",
  "dropzone.takePhoto": "Tippen, um ein Foto aufzunehmen",
  "dropzone.capture": "Foto aufnehmen",
  "dropzone.cancel": "Abbrechen",
  "dropzone.retake": "tippen, um neu aufzunehmen",
  "dropzone.switchCamera": "Wechseln",
  "dropzone.cameraStarting": "Kamera wird gestartet…",
  "dropzone.cameraError":
    "Kein Zugriff auf die Kamera. Bitte erlaube den Kamerazugriff oder lade stattdessen eine Datei hoch.",
  "dropzone.orUpload": "oder stattdessen eine Datei hochladen",

  // Multi-image picker
  "multiupload.add": "Weiteres Bild hinzufügen",
  "multiupload.remove": "Entfernen",
  "multiupload.imageLabel": "Bild {n}",
  "multiupload.hint": "Weitere Seiten oder Perspektiven hinzufügen — optional.",
  "multiupload.modePrompt": "Diese Bilder sind:",
  "multiupload.startOver": "Von vorne beginnen",
  "multiupload.someRejected":
    "{n} Bild(er) wurden nicht angenommen (Ratenlimit). Der Rest wird angezeigt.",

  // Library
  "library.title": "Bibliothek",
  "library.newFromPhoto": "Neu aus Foto",
  "library.enterManually": "Manuell eingeben",
  "library.selected": "{n} ausgewählt",
  "library.downloadBundle": "Sammel-PGN herunterladen",
  "library.clear": "Leeren",
  "library.loadingGames": "Partien werden geladen …",
  "library.empty":
    "Noch keine gespeicherten Partien. Wandle ein Foto um oder gib eine manuell ein, um zu beginnen.",
  "library.colWhite": "Weiß",
  "library.colBlack": "Schwarz",
  "library.colEvent": "Turnier",
  "library.colDate": "Datum",
  "library.colResult": "Ergebnis",
  "library.colMoves": "Züge",
  "library.colActions": "Aktionen",
  "library.view": "Ansehen",
  "library.open": "Öffnen",
  "library.pgn": "PGN",
  "library.previous": "Zurück",
  "library.next": "Weiter",
  "library.pageOf": "Seite {page} von {total}",
  "library.errLoad": "Partien konnten nicht geladen werden.",
  "library.errDownload": "Download fehlgeschlagen.",
  "library.errBundle": "Sammel-Export fehlgeschlagen.",
  "library.errDraft": "Entwurf konnte nicht erstellt werden.",
  "library.selectAria": "{white} gegen {black} auswählen",
  "library.searchPlaceholder": "Suchen …",
  "library.searchAria": "Partien durchsuchen",
  "library.playerPlaceholder": "Spieler",
  "library.playerAria": "Nach Spieler filtern",
  "library.eventPlaceholder": "Turnier",
  "library.eventAria": "Nach Turnier filtern",
  "library.fromPlaceholder": "Von (JJJJ.MM.TT)",
  "library.fromAria": "Von-Datum",
  "library.toPlaceholder": "Bis (JJJJ.MM.TT)",
  "library.toAria": "Bis-Datum",
  "library.apply": "Anwenden",

  // Header fields
  "header.white": "Weiß",
  "header.black": "Schwarz",
  "header.event": "Turnier",
  "header.site": "Ort",
  "header.date": "Datum (JJJJ.MM.TT)",
  "header.round": "Runde",
  "header.board": "Brett",
  "header.result": "Ergebnis",
  "header.whitePlayer": "Spieler Weiß",
  "header.blackPlayer": "Spieler Schwarz",

  // Move list
  "moves.title": "Züge",
  "moves.start": "Anfang",
  "moves.jumpStartAria": "Zur Ausgangsstellung springen",
  "moves.none": "Noch keine Züge.",
  "moves.addByDragging": "Füge einen Zug hinzu, indem du eine Figur auf dem Brett ziehst.",
  "moves.legal": "legal",
  "moves.illegal": "illegal",
  "moves.ambiguous": "mehrdeutig",
  "moves.readAs": "gelesen als „{text}“",
  "moves.corrected": "(korrigiert)",
  "moves.didYouMean": "Meintest du",
  "moves.disambiguate": "Eindeutig machen",
  "moves.otherLegal": "Andere legale Züge",
  "moves.choose": "wählen …",
  "moves.sanPlaceholder": "SAN, z. B. Sf3",
  "moves.editAria": "Zug-SAN bearbeiten",
  "moves.clickToView": "Klicken zum Ansehen, erneut zum Bearbeiten",
  "moves.showPositionAria": "Stellung nach {label} {san} anzeigen",
  "moves.markUnreadable": "Als unleserlichen Platzhalter markieren",
  "moves.insertAfter": "Einen Zug danach einfügen",
  "moves.truncate": "Partie hier abschneiden (diesen und alle späteren Züge löschen)",
  "moves.delete": "Diesen Zug löschen",
  "moves.verify": "prüfen",
  "moves.verifyHint": "Mit geringer Sicherheit erkannt – mit dem Formular vergleichen und bestätigen.",

  // Engine controls
  "engine.unavailable": "Engine in diesem Browser nicht verfügbar.",
  "engine.eval": "Engine-Bewertung",
  "engine.analyzing": "Analyse läuft … {pct} % (stopp)",
  "engine.clear": "Analyse löschen",
  "engine.analyze": "Partie analysieren",

  // Game viewer
  "viewer.vs": "gegen",
  "viewer.round": "Runde {n}",
  "viewer.board": "Brett {n}",
  "viewer.flip": "Brett drehen",
  "viewer.stepCaption": "Nutze ◀ ▶ oder die Pfeiltasten, um durch die Partie zu blättern.",

  // Review workbench
  "review.manualNoPhoto": "Manuelle Eingabe – kein Foto.",
  "review.edit": "Bearbeiten",
  "review.view": "Ansehen",
  "review.captionView": "Nutze ◀ ▶ oder die Pfeiltasten, um durch die Partie zu blättern.",
  "review.captionEdit":
    "Ziehe eine Figur, um den ausgewählten Zug hinzuzufügen oder zu ersetzen. Klicke auf einen Zug, um seine Stellung zu sehen.",
  "review.serverRejected":
    "Der Server hat Zug Nr. {n} als illegal abgelehnt. Korrigiere ihn und versuche es erneut.",
  "review.working": "Wird verarbeitet …",
  "review.resolveIllegal":
    "Behebe alle illegalen/mehrdeutigen Züge (oder schneide ab), um fortzufahren.",
  "review.toVerify": "{n} zu prüfen",
  "review.allVerified": "alle geprüft",
  "review.nextUncertain": "Nächster zu prüfen",
  "review.unverifiedNote":
    "Einige Züge wurden mit geringer Sicherheit erkannt – prüfe die markierten Züge vor dem Speichern anhand des Formulars.",

  // Photo viewer
  "photo.title": "Foto",
  "photo.zoomOut": "Verkleinern",
  "photo.zoomIn": "Vergrößern",
  "photo.reset": "Ansicht zurücksetzen",
  "photo.alt": "Partieformular",

  // Move nav
  "movenav.start": "Ausgangsstellung",
  "movenav.prev": "Vorheriger Zug",
  "movenav.next": "Nächster Zug",
  "movenav.last": "Letzter Zug",

  // Review page
  "reviewPage.title": "Prüfen",
  "reviewPage.saveGame": "Partie speichern",
  "reviewPage.saved": "Gespeichert.",
  "reviewPage.viewGame": "Partie ansehen",
  "reviewPage.goToLibrary": "zur Bibliothek",
  "reviewPage.saveFailed": "Speichern fehlgeschlagen.",
  "reviewPage.couldNotLoad": "Diese Partie konnte nicht geladen werden",
  "reviewPage.loadError": "Partie konnte nicht geladen werden.",

  // View page
  "viewPage.title": "Partie ansehen",
  "viewPage.edit": "Bearbeiten",

  // New / Analysieren-&-coachen-Seite
  "new.title": "Analysieren & coachen lassen",
  "new.subtitle":
    "Füge eine FEN ein oder importiere eine Partie (PGN oder eine Lichess-Studie/-Partie-URL). Die Engine bewertet sie in deinem Browser, dann erklärt dir ein Trainer das Warum in klaren Worten.",
  "new.resultSubtitle":
    "Starte die Engine, dann klicke bei einem Zug auf „Erklären“ oder auf „Partie coachen“ für eine Zusammenfassung.",
  "new.resultSubtitlePosition":
    "Aktiviere die Engine-Bewertung, um die Einschätzung zu sehen, und klicke dann auf „Stellung coachen“ für eine Erklärung.",
  "new.modeFen": "FEN einfügen",
  "new.modeImport": "PGN / Lichess importieren",
  "new.fenLabel": "FEN",
  "new.importLabel": "PGN oder Lichess-URL",
  "new.fenPlaceholder": "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
  "new.importPlaceholder":
    "PGN hier einfügen oder eine Lichess-Studie/-Partie-URL (https://lichess.org/…)",
  "new.submit": "Laden",
  "new.working": "Wird geladen …",
  "new.errEmpty": "In dieser Eingabe wurde keine Partie gefunden.",
  "new.errInvalid": "Das sieht nicht nach einer gültigen Stellung oder Partie aus.",
  "new.errRateLimit":
    "Ratenlimit erreicht. Bitte warte einen Moment und versuche es erneut.",
  "new.errGeneric": "Diese Eingabe konnte nicht geladen werden.",
  "new.gameLabel": "Partie {n}",
  "new.startOver": "Von vorne beginnen",
  "new.save": "In Bibliothek speichern",
  "new.downloadPgn": "PGN herunterladen",
  "new.saved": "In deiner Bibliothek gespeichert.",
  "new.viewGame": "Partie ansehen",
  "new.promoPrefix": "Hast du stattdessen ein Foto eines Partieformulars?",
  "new.promoConvert": "Foto umwandeln →",

  // Coach (Engine → LLM-Erklärung)
  "coach.title": "Trainer",
  "coach.thinking": "Trainer denkt nach …",
  "coach.error": "Coaching fehlgeschlagen. Bitte versuche es erneut.",
  "coach.gameSummary": "Partie-Zusammenfassung",
  "coach.explain": "Erklären",
  "coach.explained": "Erklärt ✓",
  "coach.explainHint": "Frag den Trainer, warum dieser Zug hilft oder schadet.",
  "coach.coachGame": "Partie coachen",
  "coach.coachPosition": "Stellung coachen",
  "coach.play": "Abspielen",
  "coach.stop": "Stopp",
  "coach.replay": "Wiederholen",

  // Sprachausgabe (globaler Schalter + Quelle)
  "tts.on": "Sprachausgabe an",
  "tts.off": "Sprachausgabe aus",
  "tts.sourceLabel": "Sprachquelle",
  "tts.source.server": "Server",
  "tts.source.browser": "Browser",

  // Dialog-Trainer (den Trainer per Text oder Sprache fragen)
  "chat.title": "Frag den Trainer",
  "chat.open": "Frag den Trainer",
  "chat.close": "Schließen",
  "chat.placeholder": "Frag etwas zu dieser Stellung …",
  "chat.send": "Senden",
  "chat.thinking": "Der Trainer überlegt …",
  "chat.error": "Der Trainer konnte nicht antworten. Bitte versuche es erneut.",
  "chat.empty": "Frag alles zur Stellung — bester Zug, warum ein Zug schlecht ist, Matt, Pläne …",
  "chat.you": "Du",
  "chat.coach": "Trainer",
  "chat.mic.start": "Frage sprechen",
  "chat.mic.stop": "Aufnahme stoppen",
  "chat.mic.recording": "Aufnahme …",
  "chat.mic.listening": "Höre zu …",
  "chat.mic.denied": "Mikrofonzugriff wurde verweigert. Du kannst deine Frage auch eintippen.",
  "chat.mic.unavailable": "Spracheingabe ist in diesem Browser nicht verfügbar. Bitte tippe deine Frage.",

  // Spracheingabe-Quelle (Speech-to-Text)
  "stt.sourceLabel": "Spracheingabe",
  "stt.source.server": "Server",
  "stt.source.browser": "Browser",
};

export const messages: Record<Locale, Catalog> = { de, en };
