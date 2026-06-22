import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

// Mock the api-client speak() so no real network is used. Declared before
// importing the controller so the mock is in place when tts.ts binds it.
const apiSpeak = vi.fn();
vi.mock("./api-client", () => ({
  speak: (...args: unknown[]) => apiSpeak(...args),
}));

import {
  getSpeechController,
  voiceLang,
  __resetSpeechControllerForTests,
} from "./tts";
import { shouldSpeak } from "@/hooks/useSpeech";

// --- DOM stubs (vitest runs in the "node" environment, so there is no window)

class FakeAudio {
  src = "";
  currentTime = 0;
  paused = true;
  onended: (() => void) | null = null;
  onpause: (() => void) | null = null;
  onerror: (() => void) | null = null;
  static lastInstance: FakeAudio | null = null;
  play = vi.fn(async () => {
    this.paused = false;
  });
  pause = vi.fn(() => {
    this.paused = true;
    // Mirror the real <audio>: pausing fires the pause event.
    this.onpause?.();
  });
  // Test helper: simulate the audio playing to its natural end.
  end(): void {
    this.onended?.();
  }
  constructor() {
    FakeAudio.lastInstance = this;
  }
}

const synthSpeak = vi.fn();
const synthCancel = vi.fn();

// Capture the most recent utterance so tests can fire its lifecycle events.
interface UtterLike {
  onstart: (() => void) | null;
  onend: (() => void) | null;
  onerror: (() => void) | null;
}
let lastUtter: UtterLike | null = null;
function captureUtter(u: UtterLike): void {
  lastUtter = u;
}

function installDom(): void {
  FakeAudio.lastInstance = null;
  synthSpeak.mockClear();
  synthCancel.mockClear();
  vi.stubGlobal("window", {
    speechSynthesis: {
      speak: synthSpeak,
      cancel: synthCancel,
      getVoices: () => [{ lang: "de-DE", name: "Anna" }],
    },
  });
  vi.stubGlobal("Audio", FakeAudio as unknown as typeof Audio);
  vi.stubGlobal(
    "speechSynthesis",
    (globalThis as unknown as { window: { speechSynthesis: unknown } }).window
      .speechSynthesis,
  );
  lastUtter = null;
  vi.stubGlobal(
    "SpeechSynthesisUtterance",
    class {
      text: string;
      lang = "";
      voice: unknown = null;
      onstart: (() => void) | null = null;
      onend: (() => void) | null = null;
      onerror: (() => void) | null = null;
      constructor(text: string) {
        this.text = text;
        captureUtter(this);
      }
    } as unknown as typeof SpeechSynthesisUtterance,
  );
}

beforeEach(() => {
  installDom();
  apiSpeak.mockReset();
  // Fresh controller per test → fresh cached <audio> element.
  __resetSpeechControllerForTests();
});

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("voiceLang", () => {
  it("maps app locales to BCP-47 voice languages", () => {
    expect(voiceLang("de")).toBe("de-DE");
    expect(voiceLang("en")).toBe("en-US");
    expect(voiceLang("de-DE")).toBe("de-DE");
  });
});

describe("server backend (success)", () => {
  it("calls the speak API and plays the returned audioUrl", async () => {
    apiSpeak.mockResolvedValue({
      audioUrl: "/api/coach/audio/abc123",
      cached: false,
      provider: "fake",
      voice: "Kore",
    });

    await getSpeechController().speak("Guten Zug!", "de", "server");

    expect(apiSpeak).toHaveBeenCalledWith({ text: "Guten Zug!", lang: "de" });
    expect(FakeAudio.lastInstance?.src).toBe("/api/coach/audio/abc123");
    expect(FakeAudio.lastInstance?.play).toHaveBeenCalledTimes(1);
    // Browser fallback must NOT have been used.
    expect(synthSpeak).not.toHaveBeenCalled();
  });
});

describe("server backend (failure → browser fallback)", () => {
  it("falls back to speechSynthesis when the API rejects (e.g. 503)", async () => {
    apiSpeak.mockRejectedValue(new Error("tts_unavailable"));

    await getSpeechController().speak("Schöner Zug!", "de", "server");

    expect(apiSpeak).toHaveBeenCalledTimes(1);
    // cancel() before each utterance, then speak().
    expect(synthCancel).toHaveBeenCalled();
    expect(synthSpeak).toHaveBeenCalledTimes(1);
  });

  it("falls back when audio playback rejects", async () => {
    apiSpeak.mockResolvedValue({
      audioUrl: "/api/coach/audio/x",
      cached: true,
      provider: "fake",
      voice: "Kore",
    });
    // Prime the controller so it caches a FakeAudio, then force that cached
    // element's play() to reject on the next call.
    const ctl = getSpeechController();
    await ctl.speak("warmup", "de", "server"); // create + cache the FakeAudio
    const cached = FakeAudio.lastInstance!;
    cached.play.mockRejectedValueOnce(new Error("no audio"));

    await ctl.speak("Zug zwei", "de", "server");

    expect(synthSpeak).toHaveBeenCalledTimes(1);
  });
});

describe("browser backend", () => {
  it("speaks directly via speechSynthesis without calling the API", async () => {
    await getSpeechController().speak("Direkt", "de", "browser");
    expect(apiSpeak).not.toHaveBeenCalled();
    expect(synthSpeak).toHaveBeenCalledTimes(1);
  });
});

describe("playback lifecycle hooks", () => {
  it("server: onStart fires when playback starts, onEnd on natural end", async () => {
    apiSpeak.mockResolvedValue({
      audioUrl: "/api/coach/audio/abc",
      cached: false,
      provider: "fake",
      voice: "Kore",
    });
    const onStart = vi.fn();
    const onEnd = vi.fn();

    await getSpeechController().speak("Hallo", "de", "server", {
      onStart,
      onEnd,
    });

    // play() resolved → onStart fired, onEnd not yet.
    expect(onStart).toHaveBeenCalledTimes(1);
    expect(onEnd).not.toHaveBeenCalled();

    // Audio plays to its natural end → onEnd fires once.
    FakeAudio.lastInstance!.end();
    expect(onEnd).toHaveBeenCalledTimes(1);
  });

  it("server: stop() detaches handlers so no stale onEnd fires", async () => {
    apiSpeak.mockResolvedValue({
      audioUrl: "/api/coach/audio/abc",
      cached: false,
      provider: "fake",
      voice: "Kore",
    });
    const onEnd = vi.fn();
    const ctl = getSpeechController();
    await ctl.speak("Hallo", "de", "server", { onEnd });

    // stop() detaches handlers then pauses. useSpeech.stop() clears `speaking`
    // directly; the controller must not also fire a stale onEnd for the
    // superseded utterance (idempotent / no double-trigger).
    const audio = FakeAudio.lastInstance!;
    ctl.stop();
    audio.end(); // stale → handlers detached, must NOT fire onEnd
    expect(onEnd).not.toHaveBeenCalled();
  });

  it("server: a superseded utterance's late events do not fire onEnd", async () => {
    apiSpeak.mockResolvedValue({
      audioUrl: "/api/coach/audio/abc",
      cached: false,
      provider: "fake",
      voice: "Kore",
    });
    const ctl = getSpeechController();
    const firstEnd = vi.fn();
    await ctl.speak("Erste", "de", "server", { onEnd: firstEnd });
    const firstAudio = FakeAudio.lastInstance!;

    // A new utterance supersedes the first (same shared <audio>, new session).
    const secondEnd = vi.fn();
    await ctl.speak("Zweite", "de", "server", { onEnd: secondEnd });

    // A late ended event for the first session must be ignored.
    firstAudio.onended?.();
    expect(firstEnd).not.toHaveBeenCalled();
  });

  it("browser: onstart → onStart, onerror → onEnd", async () => {
    apiSpeak.mockRejectedValue(new Error("tts_unavailable"));
    const onStart = vi.fn();
    const onEnd = vi.fn();

    await getSpeechController().speak("Direkt", "de", "server", {
      onStart,
      onEnd,
    });

    // Browser fallback was used; drive its utterance lifecycle.
    expect(synthSpeak).toHaveBeenCalledTimes(1);
    lastUtter!.onstart?.();
    expect(onStart).toHaveBeenCalledTimes(1);
    lastUtter!.onerror?.();
    expect(onEnd).toHaveBeenCalledTimes(1);
  });

  it("browser: onend fires onEnd", async () => {
    const onEnd = vi.fn();
    await getSpeechController().speak("Direkt", "de", "browser", { onEnd });
    lastUtter!.onend?.();
    expect(onEnd).toHaveBeenCalledTimes(1);
  });
});

describe("shouldSpeak (dedupe)", () => {
  it("speaks a new key but not a repeat of the same key", () => {
    expect(shouldSpeak(null, 3)).toBe(true);
    expect(shouldSpeak(3, 3)).toBe(false);
    expect(shouldSpeak(3, -1)).toBe(true);
    expect(shouldSpeak(-2, -2)).toBe(false);
  });
});
