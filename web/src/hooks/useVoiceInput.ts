"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { voiceLang } from "@/lib/tts";
import type { SttSource } from "@/i18n/SpeechSettingsProvider";

// Minimal ambient types for the Web Speech API (not in the standard DOM lib).
interface SpeechRecognitionLike {
  lang: string;
  interimResults: boolean;
  continuous: boolean;
  onresult: ((e: SpeechRecognitionEventLike) => void) | null;
  onerror: ((e: { error: string }) => void) | null;
  onend: (() => void) | null;
  start: () => void;
  stop: () => void;
  abort: () => void;
}
interface SpeechRecognitionEventLike {
  resultIndex: number;
  results: ArrayLike<{ 0: { transcript: string }; isFinal: boolean }>;
}

function getRecognitionCtor(): (new () => SpeechRecognitionLike) | null {
  if (typeof window === "undefined") return null;
  const w = window as unknown as {
    SpeechRecognition?: new () => SpeechRecognitionLike;
    webkitSpeechRecognition?: new () => SpeechRecognitionLike;
  };
  return w.SpeechRecognition ?? w.webkitSpeechRecognition ?? null;
}

function canRecord(): boolean {
  return (
    typeof window !== "undefined" &&
    typeof navigator !== "undefined" &&
    !!navigator.mediaDevices?.getUserMedia &&
    typeof MediaRecorder !== "undefined"
  );
}

// Prefer Opus in WebM; fall back through what the browser supports.
function pickMime(): string {
  if (typeof MediaRecorder === "undefined" || !MediaRecorder.isTypeSupported)
    return "";
  for (const m of [
    "audio/webm;codecs=opus",
    "audio/webm",
    "audio/mp4",
    "audio/ogg;codecs=opus",
  ]) {
    if (MediaRecorder.isTypeSupported(m)) return m;
  }
  return "";
}

// Safety cap so a forgotten recording cannot grow unbounded.
const MAX_RECORD_MS = 30_000;

export type VoiceAvailability = "server" | "browser" | "none";

export interface VoiceInputApi {
  // Effective availability after downgrading to what the browser supports.
  available: VoiceAvailability;
  recording: boolean; // server-STT capture in progress
  listening: boolean; // browser-STT capture in progress
  active: boolean; // recording || listening
  error: string | null;
  start: () => void;
  stop: () => void;
}

export interface VoiceInputOptions {
  sttSource: SttSource;
  lang: string;
  onAudio: (blob: Blob) => void; // server path: a finished recording
  onPartialTranscript?: (text: string) => void; // browser path: interim text
  onFinalTranscript?: (text: string) => void; // browser path: committed text
  onError?: (key: string) => void; // i18n key (chat.mic.denied / chat.mic.unavailable)
}

// useVoiceInput abstracts the two voice paths. Server STT records audio with
// MediaRecorder and hands back a Blob to upload; browser STT streams transcripts
// from the Web Speech API. It downgrades gracefully (server → browser → none) to
// whatever the browser actually supports.
export function useVoiceInput(opts: VoiceInputOptions): VoiceInputApi {
  const { sttSource, lang, onAudio, onPartialTranscript, onFinalTranscript, onError } = opts;

  const [recording, setRecording] = useState(false);
  const [listening, setListening] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const recorderRef = useRef<MediaRecorder | null>(null);
  const streamRef = useRef<MediaStream | null>(null);
  const chunksRef = useRef<Blob[]>([]);
  const recognitionRef = useRef<SpeechRecognitionLike | null>(null);
  const autoStopRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Resolve the effective source, downgrading to what the browser supports.
  const recordOK = canRecord();
  const browserOK = getRecognitionCtor() !== null;
  let available: VoiceAvailability = "none";
  if (sttSource === "server") available = recordOK ? "server" : browserOK ? "browser" : "none";
  else available = browserOK ? "browser" : recordOK ? "server" : "none";

  const fail = useCallback(
    (key: string) => {
      setError(key);
      onError?.(key);
    },
    [onError],
  );

  const cleanupRecorder = useCallback(() => {
    if (autoStopRef.current) {
      clearTimeout(autoStopRef.current);
      autoStopRef.current = null;
    }
    streamRef.current?.getTracks().forEach((t) => t.stop());
    streamRef.current = null;
    recorderRef.current = null;
  }, []);

  const startServer = useCallback(async () => {
    setError(null);
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
      streamRef.current = stream;
      const mime = pickMime();
      const rec = new MediaRecorder(stream, mime ? { mimeType: mime } : undefined);
      chunksRef.current = [];
      rec.ondataavailable = (e) => {
        if (e.data.size > 0) chunksRef.current.push(e.data);
      };
      rec.onstop = () => {
        const blob = new Blob(chunksRef.current, { type: mime || "audio/webm" });
        cleanupRecorder();
        setRecording(false);
        if (blob.size > 0) onAudio(blob);
      };
      recorderRef.current = rec;
      rec.start();
      setRecording(true);
      autoStopRef.current = setTimeout(() => rec.state !== "inactive" && rec.stop(), MAX_RECORD_MS);
    } catch {
      cleanupRecorder();
      setRecording(false);
      fail("chat.mic.denied");
    }
  }, [cleanupRecorder, onAudio, fail]);

  const startBrowser = useCallback(() => {
    setError(null);
    const Ctor = getRecognitionCtor();
    if (!Ctor) {
      fail("chat.mic.unavailable");
      return;
    }
    const rec = new Ctor();
    rec.lang = voiceLang(lang);
    rec.interimResults = true;
    rec.continuous = false;
    rec.onresult = (e) => {
      let interim = "";
      let final = "";
      for (let i = e.resultIndex; i < e.results.length; i++) {
        const r = e.results[i];
        if (r.isFinal) final += r[0].transcript;
        else interim += r[0].transcript;
      }
      if (interim) onPartialTranscript?.(interim);
      if (final) onFinalTranscript?.(final);
    };
    rec.onerror = (ev) => {
      setListening(false);
      if (ev.error === "not-allowed" || ev.error === "service-not-allowed") {
        fail("chat.mic.denied");
      }
    };
    rec.onend = () => setListening(false);
    recognitionRef.current = rec;
    try {
      rec.start();
      setListening(true);
    } catch {
      setListening(false);
      fail("chat.mic.unavailable");
    }
  }, [lang, onPartialTranscript, onFinalTranscript, fail]);

  const start = useCallback(() => {
    if (recording || listening) return;
    if (available === "server") void startServer();
    else if (available === "browser") startBrowser();
    else fail("chat.mic.unavailable");
  }, [available, recording, listening, startServer, startBrowser, fail]);

  const stop = useCallback(() => {
    if (recorderRef.current && recorderRef.current.state !== "inactive") {
      recorderRef.current.stop(); // fires onstop -> onAudio
    }
    if (recognitionRef.current) {
      recognitionRef.current.stop();
      recognitionRef.current = null;
    }
  }, []);

  // Release the mic / recognizer on unmount.
  useEffect(() => {
    return () => {
      if (recorderRef.current && recorderRef.current.state !== "inactive") {
        recorderRef.current.stop();
      }
      recognitionRef.current?.abort();
      cleanupRecorder();
    };
  }, [cleanupRecorder]);

  return {
    available,
    recording,
    listening,
    active: recording || listening,
    error,
    start,
    stop,
  };
}
