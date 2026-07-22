/** Authoritative Pi v3 JSONL session result collection. */
import { createReadStream } from 'node:fs';
import { readdir, readFile } from 'node:fs/promises';
import { homedir } from 'node:os';
import { join } from 'node:path';

export const MAX_SESSION_BYTES = 10 * 1024 * 1024;
export const MAX_SESSION_LINES = 100_000;
const SESSION_ID_RE = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

export async function resolvePiSessionReference(session, sessionRoot = join(homedir(), '.pi', 'agent', 'sessions')) {
  if (!session || (session.kind !== 'path' && session.kind !== 'id') || typeof session.value !== 'string') throw new Error('invalid Pi session reference');
  if (session.kind === 'path') {
    if (!session.value.startsWith('/') || !session.value.endsWith('.jsonl') || /[\x00\n\r]/.test(session.value)) throw new Error('unsafe Pi session path reference');
    return session.value;
  }
  if (!SESSION_ID_RE.test(session.value)) throw new Error('invalid Pi session ID reference');
  const matches = await findSessionFilesById(sessionRoot, session.value);
  if (matches.length !== 1) throw new Error(matches.length ? `ambiguous Pi session ID: ${session.value}` : `Pi session ID not found: ${session.value}`);
  return matches[0];
}

async function findSessionFilesById(root, sessionId) {
  const files = [];
  async function walk(directory) {
    const entries = await readdir(directory, { withFileTypes: true });
    for (const entry of entries) {
      const path = join(directory, entry.name);
      if (entry.isDirectory()) await walk(path);
      else if (entry.isFile() && entry.name.endsWith('.jsonl')) {
        const firstLine = (await readFile(path, 'utf8')).split('\n', 1)[0];
        try { if (JSON.parse(firstLine)?.type === 'session' && JSON.parse(firstLine)?.id === sessionId) files.push(path); } catch { /* ignore malformed candidate */ }
      }
    }
  }
  await walk(root);
  return files;
}

export async function waitForPiSessionReference(session, timeoutMs = 10000, signal, sessionRoot) {
  const deadline = Date.now() + timeoutMs;
  let lastError;
  while (Date.now() < deadline) {
    if (signal?.aborted) throw new Error('collection aborted');
    try { return await resolvePiSessionReference(session, sessionRoot); } catch (error) { lastError = error; }
    await sleep(250, signal);
  }
  throw lastError || new Error('Pi session reference was not resolved');
}

/** Capture a durable, complete-record boundary before bridge prompt submission. */
export async function captureSessionBoundary(filePath) {
  const entries = await readJsonlEntries(filePath);
  const header = entries.find((entry) => entry?.type === 'session' && typeof entry.id === 'string' && entry.id);
  if (!header) throw new Error('Pi session header is missing or invalid');
  const last = entries.at(-1);
  if (!last || typeof last.id !== 'string' || !last.id) throw new Error('Pi session has no complete entry ID for prompt boundary');
  return { sessionId: header.id, anchorId: last.id };
}

export async function collectSessionResult(filePath) {
  return extractFinalizedResult(await readJsonlEntries(filePath));
}

export async function collectSessionResultAfterBoundary(filePath, boundary) {
  if (!boundary?.sessionId || !boundary?.anchorId) throw new Error('invalid prompt boundary');
  const entries = await readJsonlEntries(filePath);
  const headers = entries.filter((entry) => entry?.type === 'session' && typeof entry.id === 'string');
  if (headers.length !== 1 || headers[0].id !== boundary.sessionId) throw new Error('Pi session header changed or is invalid');
  const anchors = entries.reduce((found, entry, index) => entry?.id === boundary.anchorId ? [...found, index] : found, []);
  if (anchors.length !== 1) throw new Error('Pi prompt boundary anchor is missing or ambiguous');
  return extractPostBoundaryResult(entries.slice(anchors[0] + 1));
}

export async function waitForFinalizedSessionResult(filePath, timeoutMs = 10000, signal, boundary) {
  const deadline = Date.now() + timeoutMs;
  let lastError;
  while (Date.now() < deadline) {
    if (signal?.aborted) throw new Error('collection aborted');
    try {
      const result = boundary ? await collectSessionResultAfterBoundary(filePath, boundary) : await collectSessionResult(filePath);
      if (result.stopReason !== 'incomplete') return result;
    } catch (error) { lastError = error; }
    await sleep(250, signal);
  }
  if (lastError) throw lastError;
  return boundary ? collectSessionResultAfterBoundary(filePath, boundary) : collectSessionResult(filePath);
}

function sleep(ms, signal) {
  return new Promise((resolvePromise, reject) => {
    if (signal?.aborted) return reject(new Error('collection aborted'));
    let timer;
    const onAbort = () => finish(new Error('collection aborted'));
    const finish = (error) => {
      clearTimeout(timer);
      signal?.removeEventListener('abort', onAbort);
      if (error) reject(error); else resolvePromise();
    };
    timer = setTimeout(() => finish(), ms);
    signal?.addEventListener('abort', onAbort, { once: true });
  });
}

/** Reads only complete newline-terminated JSONL records. */
export async function readJsonlEntries(filePath) {
  const entries = [];
  let bytes = 0;
  let lines = 0;
  return new Promise((resolvePromise, reject) => {
    const stream = createReadStream(filePath, { encoding: 'utf8' });
    let buffer = '';
    const parse = (line) => {
      if (!line.trim()) return;
      lines++;
      if (lines > MAX_SESSION_LINES) { stream.destroy(); reject(new Error(`session file exceeds maximum lines (${MAX_SESSION_LINES})`)); return; }
      try { entries.push(JSON.parse(line)); } catch { /* malformed records do not become results */ }
    };
    stream.on('data', (chunk) => {
      bytes += Buffer.byteLength(chunk);
      if (bytes > MAX_SESSION_BYTES) { stream.destroy(); reject(new Error(`session file exceeds maximum size (${MAX_SESSION_BYTES} bytes)`)); return; }
      buffer += chunk;
      const parts = buffer.split('\n');
      buffer = parts.pop() || '';
      for (const line of parts) parse(line);
    });
    // A non-newline-terminated trailing record may be in flight: never make it
    // a prompt anchor or finalized result.
    stream.on('end', () => resolvePromise(entries));
    stream.on('error', (error) => reject(new Error(`failed to read session file: ${error.message}`)));
  });
}

export function extractFinalizedResult(entries) {
  return extractTurnResult(entries, false);
}

function extractPostBoundaryResult(entries) {
  const userIndexes = [];
  for (let index = 0; index < entries.length; index++) {
    if (entries[index]?.type === 'message' && entries[index].message?.role === 'user') userIndexes.push(index);
  }
  if (!userIndexes.length) return incomplete([]);
  if (userIndexes.length > 1) return incomplete([]);
  return extractTurnResult(entries.slice(userIndexes[0] + 1), true);
}

function extractTurnResult(entries, requireTerminal) {
  if (!Array.isArray(entries)) return incomplete([]);
  const calls = new Map();
  let terminalAssistant;
  let turns = 0;
  for (const entry of entries) {
    if (entry?.type !== 'message' || !entry.message || typeof entry.message !== 'object') continue;
    const message = entry.message;
    if (message.role === 'assistant' && typeof message.stopReason === 'string') {
      turns++;
      for (const part of Array.isArray(message.content) ? message.content : []) {
        if (part?.type === 'toolCall' && typeof part.id === 'string') calls.set(part.id, { id: part.id, name: String(part.name || 'unknown'), arguments: part.arguments || {} });
      }
      if (message.stopReason !== 'toolUse') terminalAssistant = message;
    }
    if (message.role === 'toolResult' && typeof message.toolCallId === 'string') {
      const call = calls.get(message.toolCallId);
      if (call) call.result = summarizeToolResult(message);
    }
  }
  const toolCalls = [...calls.values()];
  if (!terminalAssistant) return incomplete(toolCalls, turns);
  return {
    text: (Array.isArray(terminalAssistant.content) ? terminalAssistant.content : []).filter((part) => part?.type === 'text' && typeof part.text === 'string').map((part) => part.text).join('\n'),
    stopReason: terminalAssistant.stopReason,
    model: terminalAssistant.model,
    usage: terminalAssistant.usage,
    toolCalls,
    turns,
  };
}

function incomplete(toolCalls, turns = 0) { return { text: '', stopReason: 'incomplete', toolCalls, turns }; }
function summarizeToolResult(message) {
  const text = (Array.isArray(message.content) ? message.content : []).filter((part) => part?.type === 'text' && typeof part.text === 'string').map((part) => part.text).join('\n');
  return { isError: !!message.isError, text: text.length > 500 ? `${text.slice(0, 500)}...` : text };
}
