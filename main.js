import { registerOffline } from "./offline/register.js";

window.balaurComponentsReady = import("./elements/register.js").catch(error => {
  console.warn("Balaur components could not be registered; continuing with native fallback markup.", error);
  return null;
});
await window.balaurComponentsReady;
await import("./app.js");
window.balaurOfflineReady = registerOffline();
