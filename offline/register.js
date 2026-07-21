export async function registerOffline() {
  const supported = "serviceWorker" in navigator;
  const secure = location.protocol === "https:" || ["localhost", "127.0.0.1", "::1"].includes(location.hostname);
  if (!supported || !secure) return null;
  try {
    const registration = await navigator.serviceWorker.register(new URL("../sw.js", import.meta.url), {
      scope: new URL("../", import.meta.url).href,
      updateViaCache: "none"
    });
    window.dispatchEvent(new CustomEvent("orbit:offline-ready", { detail: { scope: registration.scope } }));
    return registration;
  } catch (error) {
    console.warn("Balaur could not enable offline mode", error);
    window.dispatchEvent(new CustomEvent("orbit:offline-error", { detail: error }));
    return null;
  }
}
