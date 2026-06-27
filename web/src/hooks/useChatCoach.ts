"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import {
  chatTurn as apiChatTurn,
  chatTurnAudio as apiChatTurnAudio,
  getChatHistory as apiGetChatHistory,
  type ChatEngineFact,
  type Side,
} from "@/lib/api-client";

export interface ChatMessage {
  id: string;
  role: "user" | "coach";
  text: string;
  facts?: ChatEngineFact[];
}

export interface ChatCoachState {
  messages: ChatMessage[];
  conversationId: string | null;
  inFlight: boolean;
  error: string | null;
  sendText: (question: string) => Promise<void>;
  sendAudio: (blob: Blob) => Promise<void>;
  reset: () => void;
}

export interface ChatCoachOptions {
  fen: string;
  side: Side;
  gameId?: string;
  ply?: number | null;
  lang?: string;
}

let idSeq = 0;
function nextId(): string {
  idSeq += 1;
  return `m${idSeq}`;
}

// useChatCoach owns the conversation with the server-side coach. Each turn snapshots
// the live position context so a question always refers to the board the user sees.
// Logged-in callers (gameId present) get server-persisted history; anonymous callers
// keep history in memory and re-send it for continuity.
export function useChatCoach(opts: ChatCoachOptions): ChatCoachState {
  const { fen, side, gameId, ply, lang } = opts;
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [conversationId, setConversationId] = useState<string | null>(null);
  const [inFlight, setInFlight] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const abortRef = useRef<AbortController | null>(null);
  const messagesRef = useRef<ChatMessage[]>([]);
  useEffect(() => {
    messagesRef.current = messages;
  }, [messages]);

  // Re-hydrate a persisted conversation for this game (logged-in owner only).
  useEffect(() => {
    if (!gameId) return;
    let cancelled = false;
    apiGetChatHistory({ gameId })
      .then((res) => {
        if (cancelled || res.messages.length === 0) return;
        setConversationId(res.conversationId || null);
        setMessages(res.messages.map((m) => ({ id: nextId(), role: m.role, text: m.text })));
      })
      .catch(() => {
        /* no prior conversation, or not logged in — start fresh */
      });
    return () => {
      cancelled = true;
    };
  }, [gameId]);

  const reset = useCallback(() => {
    abortRef.current?.abort();
    setMessages([]);
    setConversationId(null);
    setError(null);
    setInFlight(false);
  }, []);

  // Shared turn runner: optimistically append the user bubble, call `run`, append
  // the coach reply (or surface an error and keep the user message for retry).
  const runTurn = useCallback(
    async (
      userText: string,
      run: (ctx: { signal: AbortSignal; history: ChatMessage[] }) => Promise<{
        conversationId: string;
        userText: string;
        reply: string;
        engineFacts: ChatEngineFact[];
      }>,
    ) => {
      abortRef.current?.abort();
      const ctrl = new AbortController();
      abortRef.current = ctrl;

      const priorHistory = messagesRef.current;
      const optimistic: ChatMessage = { id: nextId(), role: "user", text: userText };
      setMessages((prev) => [...prev, optimistic]);
      setInFlight(true);
      setError(null);

      try {
        const res = await run({ signal: ctrl.signal, history: priorHistory });
        if (ctrl.signal.aborted) return;
        // For audio turns the server transcript replaces the optimistic placeholder.
        setMessages((prev) =>
          prev
            .map((m) => (m.id === optimistic.id && res.userText ? { ...m, text: res.userText } : m))
            .concat({ id: nextId(), role: "coach", text: res.reply, facts: res.engineFacts }),
        );
        if (res.conversationId) setConversationId(res.conversationId);
      } catch (e) {
        if (ctrl.signal.aborted) return;
        setError(e instanceof Error ? e.message : "chat failed");
      } finally {
        if (!ctrl.signal.aborted) setInFlight(false);
      }
    },
    [],
  );

  const historyForServer = useCallback(
    (history: ChatMessage[]) =>
      conversationId
        ? undefined // server already has it
        : history.map((m) => ({ role: m.role, text: m.text })),
    [conversationId],
  );

  const sendText = useCallback(
    async (question: string) => {
      const q = question.trim();
      if (!q || inFlight) return;
      await runTurn(q, async ({ history }) =>
        apiChatTurn({
          conversationId,
          fen,
          side,
          gameId,
          ply,
          lang,
          question: q,
          history: historyForServer(history),
        }),
      );
    },
    [conversationId, fen, side, gameId, ply, lang, inFlight, runTurn, historyForServer],
  );

  const sendAudio = useCallback(
    async (blob: Blob) => {
      if (inFlight) return;
      // The user bubble shows a placeholder until the server transcript returns.
      await runTurn("🎤 …", async () =>
        apiChatTurnAudio({
          audio: blob,
          context: { fen, side, gameId, ply },
          conversationId,
          lang,
        }),
      );
    },
    [conversationId, fen, side, gameId, ply, lang, inFlight, runTurn],
  );

  return { messages, conversationId, inFlight, error, sendText, sendAudio, reset };
}
