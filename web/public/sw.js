/* PGNize service worker — offline app shell + smart runtime caching.
 *
 * Correctness first: this SW never caches API or auth traffic. Anything under
 * /api/ or /healthz (proxied to the Go backend) always goes straight to the
 * network so recognition, review, and session state are never served stale.
 */
const VERSION = "v1";
const STATIC_CACHE = `pgnize-static-${VERSION}`;
const RUNTIME_CACHE = `pgnize-runtime-${VERSION}`;
const OFFLINE_URL = "/offline.html";

const PRECACHE = [
  OFFLINE_URL,
  "/logo.svg",
  "/icons/icon-192.png",
  "/icons/icon-512.png",
  "/manifest.webmanifest",
];

self.addEventListener("install", (event) => {
  event.waitUntil(
    caches
      .open(STATIC_CACHE)
      .then((cache) => cache.addAll(PRECACHE))
      .then(() => self.skipWaiting()),
  );
});

self.addEventListener("activate", (event) => {
  event.waitUntil(
    caches
      .keys()
      .then((keys) =>
        Promise.all(
          keys
            .filter((k) => k !== STATIC_CACHE && k !== RUNTIME_CACHE)
            .map((k) => caches.delete(k)),
        ),
      )
      .then(() => self.clients.claim()),
  );
});

// Allow the page to trigger an immediate activation after an update.
self.addEventListener("message", (event) => {
  if (event.data === "SKIP_WAITING") self.skipWaiting();
});

function isStaticAsset(url) {
  return (
    url.pathname.startsWith("/_next/static/") ||
    url.pathname.startsWith("/icons/") ||
    url.pathname === "/logo.svg" ||
    /\.(?:css|js|woff2?|png|jpg|jpeg|gif|svg|ico|webp)$/.test(url.pathname)
  );
}

self.addEventListener("fetch", (event) => {
  const { request } = event;

  // Only GET is cacheable; everything else (POST/PUT/DELETE) passes through.
  if (request.method !== "GET") return;

  const url = new URL(request.url);

  // Same-origin only; leave cross-origin (e.g. CDNs, analytics) alone.
  if (url.origin !== self.location.origin) return;

  // Never intercept backend traffic — must always be live.
  if (url.pathname.startsWith("/api/") || url.pathname === "/healthz") return;

  // App navigations: network-first so users get fresh HTML, with an offline
  // fallback when the network is unavailable.
  if (request.mode === "navigate") {
    event.respondWith(
      (async () => {
        try {
          const fresh = await fetch(request);
          const cache = await caches.open(RUNTIME_CACHE);
          cache.put(request, fresh.clone());
          return fresh;
        } catch {
          const cached = await caches.match(request);
          return cached || (await caches.match(OFFLINE_URL));
        }
      })(),
    );
    return;
  }

  // Static assets: stale-while-revalidate for instant loads + background refresh.
  if (isStaticAsset(url)) {
    event.respondWith(
      (async () => {
        const cache = await caches.open(STATIC_CACHE);
        const cached = await cache.match(request);
        const network = fetch(request)
          .then((res) => {
            if (res && res.status === 200) cache.put(request, res.clone());
            return res;
          })
          .catch(() => cached);
        return cached || network;
      })(),
    );
  }
});
