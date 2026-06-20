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
  "nav.library": "Library",
  "nav.signOut": "Sign out",
  "nav.login": "Log in",
  "nav.register": "Register",

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

  // Convert (anonymous) flow
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
  "nav.library": "Bibliothek",
  "nav.signOut": "Abmelden",
  "nav.login": "Anmelden",
  "nav.register": "Registrieren",

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

  // Convert (anonymous) flow
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
};

export const messages: Record<Locale, Catalog> = { de, en };
