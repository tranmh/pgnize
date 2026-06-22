// Speech controller for the coach. Speaks a piece of text via one of two
// backends behind a single `speak(text, lang)` call:
//
//   - "server": POST /coach/speak → set a single shared <audio> src and play().
//     On any failure (network, rejection, 503 tts_unavailable, playback error)
//     it automatically falls back to the browser backend.
//   - "browser": window.speechSynthesis with a voice matching `lang`.
//
// Only one utterance is ever active: a new speak(), replay(), or stop() cancels
// any in-flight audio/utterance first. All window/Audio/speechSynthesis access
// is guarded for SSR — nothing touches the DOM until called on the client.

import { speak as apiSpeak } from "./api-client";

export type SpeechSource = "server" | "browser";

// Lifecycle hooks reporting REAL playback start/end, so consumers can drive a
// `speaking` flag from when audio actually plays/stops — not from when the
// start-promise resolves (which happens the instant playback begins).
export interface SpeechHooks {
  onStart?: () => void;
  onEnd?: () => void;
}

function isClient(): boolean {
  return typeof window !== "undefined";
}

// Map an app locale ("de" | "en" | "de-DE" | …) to a BCP-47 voice language.
export function voiceLang(lang: string): string {
  const base = (lang || "de").toLowerCase().split("-")[0];
  if (base === "de") return "de-DE";
  if (base === "en") return "en-US";
  return lang;
}

// Pick the best available speechSynthesis voice for a language, preferring an
// exact BCP-47 match, then a base-language match.
function pickVoice(target: string): SpeechSynthesisVoice | null {
  if (!isClient() || !window.speechSynthesis) return null;
  const voices = window.speechSynthesis.getVoices();
  if (!voices || voices.length === 0) return null;
  const lower = target.toLowerCase();
  const base = lower.split("-")[0];
  return (
    voices.find((v) => v.lang.toLowerCase() === lower) ??
    voices.find((v) => v.lang.toLowerCase().split("-")[0] === base) ??
    null
  );
}

class SpeechController {
  // A single shared audio element, reused across utterances (created lazily on
  // the client so SSR never constructs an Audio object).
  private audio: HTMLAudioElement | null = null;

  // Monotonic session id: each new speak()/replay()/stop() bumps it so a stale
  // audio handler (e.g. a pause fired by the next utterance superseding this
  // one) never reports onEnd for an utterance that is no longer current.
  private session = 0;

  private ensureAudio(): HTMLAudioElement | null {
    if (!isClient()) return null;
    if (!this.audio) this.audio = new Audio();
    return this.audio;
  }

  // Stop any active audio playback and any active browser utterance. Bumping
  // the session detaches the current audio handlers; the explicit pause() also
  // fires `audio.onpause` (when still current) so onEnd runs exactly once.
  stop(): void {
    if (!isClient()) return;
    this.session++;
    if (this.audio) {
      // Detach handlers from the superseded utterance before pausing so a
      // pause/ended event can't re-trigger onEnd for it.
      this.audio.onended = null;
      this.audio.onpause = null;
      this.audio.onerror = null;
      try {
        this.audio.pause();
        this.audio.currentTime = 0;
      } catch {
        /* ignore */
      }
    }
    if (window.speechSynthesis) {
      try {
        window.speechSynthesis.cancel();
      } catch {
        /* ignore */
      }
    }
  }

  // Speak via the requested backend. Resolves once playback has started (server)
  // or the utterance was queued (browser). On the server backend, any failure
  // transparently falls back to the browser backend.
  //
  // `hooks.onStart` / `hooks.onEnd` report the REAL playback lifecycle:
  // onStart when audio actually begins, onEnd on natural end, stop(), or error.
  // On server→browser fallback, the browser utterance drives the hooks.
  async speak(
    text: string,
    lang: string,
    source: SpeechSource,
    hooks?: SpeechHooks,
  ): Promise<void> {
    if (!isClient() || !text) return;
    this.stop();
    const session = this.session;
    if (source === "server") {
      try {
        const res = await apiSpeak({ text, lang });
        await this.playUrl(res.audioUrl, session, hooks);
        return;
      } catch {
        // Server TTS unavailable / failed → fall back to the browser voice.
      }
    }
    this.speakBrowser(text, lang, hooks);
  }

  // Play a server audio URL, wiring lifecycle handlers that only report for the
  // current session (so a superseded utterance's late events are ignored).
  private async playUrl(
    url: string,
    session: number,
    hooks?: SpeechHooks,
  ): Promise<void> {
    const audio = this.ensureAudio();
    if (!audio) return;
    let ended = false;
    const fireEnd = () => {
      if (ended) return;
      ended = true;
      hooks?.onEnd?.();
    };
    // Only honor handlers while this play session is the current one.
    audio.onended = () => {
      if (this.session === session) fireEnd();
    };
    audio.onpause = () => {
      if (this.session === session) fireEnd();
    };
    audio.onerror = () => {
      if (this.session === session) fireEnd();
    };
    audio.src = url;
    audio.currentTime = 0;
    await audio.play();
    // play() resolved → playback has actually started.
    hooks?.onStart?.();
  }

  private speakBrowser(text: string, lang: string, hooks?: SpeechHooks): void {
    if (!isClient() || !window.speechSynthesis) return;
    const synth = window.speechSynthesis;
    synth.cancel();
    const utter = new SpeechSynthesisUtterance(text);
    const target = voiceLang(lang);
    utter.lang = target;
    const voice = pickVoice(target);
    if (voice) utter.voice = voice;
    let ended = false;
    const fireEnd = () => {
      if (ended) return;
      ended = true;
      hooks?.onEnd?.();
    };
    utter.onstart = () => hooks?.onStart?.();
    utter.onend = () => fireEnd();
    utter.onerror = () => fireEnd();
    synth.speak(utter);
  }
}

// Module-level singleton: one active utterance across the whole app.
let controller: SpeechController | null = null;

export function getSpeechController(): SpeechController {
  if (!controller) controller = new SpeechController();
  return controller;
}

// Test seam: drop the cached singleton so each test starts with a fresh
// controller (and therefore a freshly constructed <audio> element).
export function __resetSpeechControllerForTests(): void {
  controller = null;
}

export type { SpeechController };
