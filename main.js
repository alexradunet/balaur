import { registerOffline } from "./offline/register.js";

window.orbitComponentsReady = import("./elements/register.js").catch(error => {
  console.warn("Balaur components could not be registered; continuing with native fallback markup.", error);
  return null;
});
await window.orbitComponentsReady;
await import("./app.js");
window.orbitOfflineReady = registerOffline();
