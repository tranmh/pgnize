import { APIRequestContext, BrowserContext, Page, request as pwRequest } from "@playwright/test";

const API_BASE = process.env.PGNIZE_API_BASE || "http://localhost:8080";

// 1x1 PNG; the fake recognizer ignores the bytes.
export const PIXEL_PNG = Buffer.from(
  "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+M8AAAMBAQDJ/pLvAAAAAElFTkSuQmCC",
  "base64",
);

let uiCounter = 0;

// registerAndAuth creates a fresh account directly against the API (with a unique client
// IP so per-IP rate limits never trip, even on repeated runs) and grafts the issued session
// cookie onto the browser context. The returned APIRequestContext is authenticated as the
// same user, for seeding data the UI will then display. Caller disposes it.
export async function registerAndAuth(
  context: BrowserContext,
  workerIndex: number,
): Promise<{ email: string; api: APIRequestContext }> {
  uiCounter += 1;
  const ip = `10.40.${workerIndex % 250}.${(uiCounter % 250) + 1}`;
  const email = `ui-${Math.random().toString(36).slice(2)}${Date.now().toString(36)}@example.com`;
  const api = await pwRequest.newContext({
    baseURL: API_BASE,
    extraHTTPHeaders: { "X-Forwarded-For": ip },
  });
  const res = await api.post("/api/auth/register", {
    data: { name: "UI User", email, password: "password1234" },
  });
  if (res.status() !== 201) {
    throw new Error(`register failed ${res.status()}: ${await res.text()}`);
  }
  const state = await api.storageState();
  const cookie = state.cookies.find((c) => c.name === "pgnize_session");
  if (!cookie) throw new Error("no pgnize_session cookie issued");
  await context.addCookies([
    {
      name: cookie.name,
      value: cookie.value,
      domain: "localhost",
      path: "/",
      httpOnly: true,
      sameSite: "Lax",
    },
  ]);
  return { email, api };
}

// useEnglish forces the English UI locale so selectors can match stable English copy
// instead of the German default. Must be called before the first navigation.
export async function useEnglish(page: Page) {
  await page.addInitScript(() => {
    try {
      localStorage.setItem("pgnize.locale", "en");
    } catch {
      /* ignore */
    }
  });
}

// trackPageErrors records uncaught exceptions; a crash like the null-moves bug surfaces here.
export function trackPageErrors(page: Page): string[] {
  const errors: string[] = [];
  page.on("pageerror", (e) => errors.push(String(e)));
  return errors;
}

// seedSavedGame creates + saves a manual game via the authenticated API and returns its id.
export async function seedSavedGame(
  api: APIRequestContext,
  header: Record<string, string>,
  moves: { ply: number; san: string }[] = [],
): Promise<string> {
  const created = await api.post("/api/games", { data: { source: "manual" } });
  const id = (await created.json()).game.id as string;
  const saved = await api.patch(`/api/games/${id}`, { data: { header, moves } });
  if (saved.status() !== 200) {
    throw new Error(`seed save failed ${saved.status()}: ${await saved.text()}`);
  }
  return id;
}
