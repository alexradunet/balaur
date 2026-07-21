const CACHE_NAME = "orbit-shell-v2";
const APP_SHELL = [
  "./",
  "./index.html",
  "./manifest.webmanifest",
  "./main.js",
  "./app.js",
  "./offline/register.js",
  "./storage/life-store.js",
  "./styles/layers.css",
  "./styles/tokens.css",
  "./styles/foundation.css",
  "./styles/shell.css",
  "./styles/canvas.css",
  "./styles/components.css",
  "./styles/themes.css",
  "./styles/responsive.css",
  "./styles/motion.css",
  "./vendor/pixel-loom/fonts.css",
  "./vendor/pixel-loom/tokens/linen.css",
  "./vendor/pixel-loom/fonts/jetbrainsmono-400-latin-ext.woff2",
  "./vendor/pixel-loom/fonts/jetbrainsmono-400-latin.woff2",
  "./vendor/pixel-loom/fonts/jetbrainsmono-500-latin-ext.woff2",
  "./vendor/pixel-loom/fonts/jetbrainsmono-500-latin.woff2",
  "./vendor/pixel-loom/fonts/newsreader-500-latin-ext.woff2",
  "./vendor/pixel-loom/fonts/newsreader-500-latin.woff2",
  "./vendor/pixel-loom/fonts/newsreader-600-latin-ext.woff2",
  "./vendor/pixel-loom/fonts/newsreader-600-latin.woff2",
  "./vendor/pixel-loom/fonts/worksans-400-latin-ext.woff2",
  "./vendor/pixel-loom/fonts/worksans-400-latin.woff2",
  "./vendor/pixel-loom/fonts/worksans-500-latin-ext.woff2",
  "./vendor/pixel-loom/fonts/worksans-500-latin.woff2",
  "./vendor/sqlite/sqlite3.mjs",
  "./vendor/sqlite/sqlite3.wasm",
  "./widgets/focus-orbit.html",
  "./icons/orbit.svg",
  "./icons/orbit-192.png",
  "./icons/orbit-512.png"
];

self.addEventListener("install", event => {
  event.waitUntil(caches.open(CACHE_NAME).then(cache => cache.addAll(APP_SHELL)));
});

self.addEventListener("activate", event => {
  event.waitUntil(Promise.all([
    caches.keys().then(names => Promise.all(names.filter(name => name.startsWith("orbit-shell-") && name !== CACHE_NAME).map(name => caches.delete(name)))),
    self.registration.navigationPreload?.enable() || Promise.resolve(),
    self.clients.claim()
  ]));
});

async function networkWithCache(request, preloadResponse) {
  const cache = await caches.open(CACHE_NAME);
  try {
    const response = await preloadResponse || await fetch(request);
    if (response.ok && response.type === "basic") await cache.put(request, response.clone());
    return response;
  } catch (error) {
    const cached = await cache.match(request, { ignoreSearch: request.mode === "navigate" });
    if (cached) return cached;
    if (request.mode === "navigate") return cache.match("./index.html");
    throw error;
  }
}

self.addEventListener("fetch", event => {
  const request = event.request;
  if (request.method !== "GET" || request.headers.has("range")) return;
  const url = new URL(request.url);
  if (url.origin !== self.location.origin || !url.pathname.startsWith(new URL(self.registration.scope).pathname)) return;
  event.respondWith(networkWithCache(request, request.mode === "navigate" ? event.preloadResponse : null));
});
