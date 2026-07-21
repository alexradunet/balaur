// Stable content hashing (Phase 1, plan §4.2, §10.1).
//
// The API is asynchronous because the browser implementation uses
// crypto.subtle.digest (secure-context only; GitHub Pages is HTTPS). Node 20+
// also exposes globalThis.crypto, so the same code path runs in tests.
// A content hash is a correctness check, never derived from mtime (plan §4.2).

function hex(bytes) {
  let out = "";
  for (const b of bytes) out += b.toString(16).padStart(2, "0");
  return out;
}

export async function contentHash(text) {
  const subtle = globalThis.crypto?.subtle;
  if (!subtle) {
    throw new Error("contentHash requires WebCrypto (crypto.subtle); unavailable in this context");
  }
  const bytes = new TextEncoder().encode(String(text));
  const digest = await subtle.digest("SHA-256", bytes);
  return hex(new Uint8Array(digest));
}

// Short, filename-safe identity suffix derived from a full id (plan §7.2).
// Deterministic and stable; used only to keep filenames readable, never as identity.
export function shortId(id) {
  return String(id).replace(/[^a-z0-9]/gi, "").slice(-6).toLowerCase() || "000000";
}
