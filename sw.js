const CACHE_NAME = "orbit-shell-v7";
const APP_SHELL = [
  "./",
  "./index.html",
  "./manifest.webmanifest",
  "./main.js",
  "./app.js",
  "./offline/register.js",
  "./storage/indexeddb-vault.js",
  "./storage/life-indexer.js",
  "./storage/life-query.js",
  "./storage/memory-index.js",
  "./storage/task-repository.js",
  "./storage/workspace-backup.js",
  "./storage/workspace-vault.js",
  "./storage/canvas-validate.js",
  "./storage/vault-store.js",
  "./storage/vault-path.js",
  "./storage/vault-errors.js",
  "./storage/content-hash.js",
  "./storage/frontmatter.js",
  "./storage/entity-codec.js",
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
  "./widgets/focus-orbit.html",
  "./icons/balaur.svg",
  "./icons/balaur-192.png",
  "./icons/balaur-512.png"
];

const SHELL_PATHS = new Set(APP_SHELL.map((path) => new URL(path, self.registration.scope).pathname));

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
  const url = new URL(request.url);
  const shellAsset = SHELL_PATHS.has(url.pathname);
  try {
    const response = await preloadResponse || await fetch(request);
    // Never write arbitrary same-origin responses, especially authenticated or
    // provider responses, into the application-shell cache.
    if (shellAsset && response.ok && response.type === "basic") await cache.put(request, response.clone());
    return response;
  } catch (error) {
    if (shellAsset) {
      const cached = await cache.match(request, { ignoreSearch: request.mode === "navigate" });
      if (cached) return cached;
    }
    if (request.mode === "navigate") return cache.match("./index.html");
    throw error;
  }
}

self.addEventListener("fetch", event => {
  const request = event.request;
  if (request.method !== "GET" || request.headers.has("range")) return;
  const url = new URL(request.url);
  if (url.origin !== self.location.origin || !url.pathname.startsWith(new URL(self.registration.scope).pathname)) return;
  if (request.headers.has("authorization")) return;
  const isShellNavigation = request.mode === "navigate";
  if (!isShellNavigation && !SHELL_PATHS.has(url.pathname)) return;
  event.respondWith(networkWithCache(request, isShellNavigation ? event.preloadResponse : null));
});
