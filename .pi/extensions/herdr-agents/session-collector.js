/** Authoritative Pi v3 JSONL session result collection. */
import { createReadStream } from 'node:fs';
import { open, opendir } from 'node:fs/promises';
import { homedir } from 'node:os';
import { join } from 'node:path';

export const MAX_SESSION_BYTES = 10 * 1024 * 1024;
export const MAX_SESSION_LINES = 100_000;
export const DEFAULT_DISCOVERY_LIMITS = Object.freeze({
  maxDepth: 12,
  maxDirectories: 2_000,
  maxEntries: 50_000,
  maxCandidates: 10_000,
  maxFirstLineBytes: 64 * 1024,
});
const SESSION_ID_RE = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

class RetryableSessionError extends Error {
  constructor(message, completeLineCount) {
    super(message);
    this.completeLineCount = completeLineCount;
  }
}

export async function resolvePiSessionReference(session, sessionRoot = join(homedir(), '.pi', 'agent', 'sessions'), discoveryOptions = {}) {
  if (!session || (session.kind !== 'path' && session.kind !== 'id') || typeof session.value !== 'string') throw new Error('invalid Pi session reference');
  if (session.kind === 'path') {
    if (!session.value.startsWith('/') || !session.value.endsWith('.jsonl') || /[\x00\n\r]/.test(session.value)) throw new Error('unsafe Pi session path reference');
    return session.value;
  }
  if (!SESSION_ID_RE.test(session.value)) throw new Error('invalid Pi session ID reference');
  const matches = await findSessionFilesById(sessionRoot, session.value, discoveryOptions);
  if (matches.length !== 1) throw new Error(matches.length ? `ambiguous Pi session ID: ${session.value}` : `Pi session ID not found: ${session.value}`);
  return matches[0];
}

function discoveryLimits(options) {
  const limits = { ...DEFAULT_DISCOVERY_LIMITS, ...(options || {}) };
  for (const [key, value] of Object.entries(limits)) if (!Number.isSafeInteger(value) || value < 1) throw new Error(`invalid session discovery limit ${key}`);
  return limits;
}

async function findSessionFilesById(root, sessionId, options) {
  const limits = discoveryLimits(options);
  const matches = [];
  const pending = [{ directory: root, depth: 0 }];
  let directories = 0;
  let entries = 0;
  let candidates = 0;
  while (pending.length) {
    const current = pending.pop();
    if (++directories > limits.maxDirectories) throw new Error(`Pi session discovery directory bound exceeded (${limits.maxDirectories})`);
    const directory = await opendir(current.directory);
    for await (const entry of directory) {
      if (++entries > limits.maxEntries) throw new Error(`Pi session discovery entry bound exceeded (${limits.maxEntries})`);
      if (entry.isSymbolicLink()) continue;
      const path = join(current.directory, entry.name);
      if (entry.isDirectory()) {
        if (current.depth >= limits.maxDepth) throw new Error(`Pi session discovery depth bound exceeded (${limits.maxDepth})`);
        pending.push({ directory: path, depth: current.depth + 1 });
      } else if (entry.isFile() && entry.name.endsWith('.jsonl')) {
        if (++candidates > limits.maxCandidates) throw new Error(`Pi session discovery candidate bound exceeded (${limits.maxCandidates})`);
        const header = await readCandidateHeader(path, limits.maxFirstLineBytes);
        if (header?.type === 'session' && header.id === sessionId) matches.push(path);
      }
    }
  }
  return matches;
}

async function readCandidateHeader(path, maxBytes) {
  const file = await open(path, 'r');
  try {
    const buffer = Buffer.alloc(maxBytes + 1);
    const { bytesRead } = await file.read(buffer, 0, buffer.length, 0);
    const newline = buffer.subarray(0, bytesRead).indexOf(0x0a);
    if (newline === -1 && bytesRead > maxBytes) throw new Error(`Pi session candidate first line exceeds discovery bound (${maxBytes} bytes): ${path}`);
    const end = newline === -1 ? bytesRead : newline;
    if (end > maxBytes) throw new Error(`Pi session candidate first line exceeds discovery bound (${maxBytes} bytes): ${path}`);
    const firstLine = buffer.subarray(0, end).toString('utf8').replace(/\r$/, '');
    try { return JSON.parse(firstLine); } catch { return null; }
  } finally { await file.close(); }
}

export async function waitForPiSessionReference(session, timeoutMs = 10000, signal, sessionRoot, discoveryOptions) {
  const deadline = Date.now() + timeoutMs;
  let lastError;
  while (Date.now() < deadline) {
    if (signal?.aborted) throw new Error('collection aborted');
    try { return await resolvePiSessionReference(session, sessionRoot, discoveryOptions); } catch (error) { lastError = error; }
    await sleep(250, signal);
  }
  throw lastError || new Error('Pi session reference was not resolved');
}

/** Resolve an ID-backed session before capture, accepting only an absent fresh file. */
export async function captureResolvedSessionBoundary(session, sessionRoot, discoveryOptions) {
  let filePath;
  try { filePath = await resolvePiSessionReference(session, sessionRoot, discoveryOptions); }
  catch (error) {
    if (session?.kind !== 'id' || !String(error?.message).startsWith('Pi session ID not found:')) throw error;
  }
  return captureSessionBoundary(session, filePath);
}

/** Capture a durable complete-physical-line boundary before bridge prompt submission. */
export async function captureSessionBoundary(session, filePath) {
  const sessionId = sessionUuid(session);
  const path = filePath || (session.kind === 'path' ? session.value : undefined);
  if (!path) return { sessionId, anchorId: null, anchorLine: null, lineCount: 0 };
  let data;
  try { data = await readJsonlRecords(path); } catch (error) {
    // A fresh idle Pi worker may not create its JSONL file until this first
    // prompt. The pinned session UUID is a safe header-only boundary.
    if (error?.code === 'ENOENT' || /ENOENT/.test(String(error?.message))) return { sessionId, anchorId: null, anchorLine: null, lineCount: 0 };
    throw error;
  }
  if (data.trailingFragment.length) throw new Error('Pi session has a trailing fragment before prompt boundary capture');
  const validRecords = data.records.filter((record) => isPlainObject(record.value));
  const headers = validRecords.filter((record) => record.value.type === 'session' && typeof record.value.id === 'string');
  if (headers.length !== 1 || headers[0].value.id !== sessionId) throw new Error('Pi session header changed or is invalid');
  const anchor = [...validRecords].reverse().find((record) => typeof record.value.id === 'string' && record.value.id);
  if (!anchor) throw new Error('Pi session has no complete entry ID for prompt boundary');
  return { sessionId, anchorId: anchor.value.id, anchorLine: anchor.lineNumber, lineCount: data.completeLineCount };
}

function sessionUuid(session) {
  if (!session || (session.kind !== 'id' && session.kind !== 'path') || typeof session.value !== 'string') throw new Error('invalid Pi session reference');
  if (session.kind === 'id' && SESSION_ID_RE.test(session.value)) return session.value;
  const suffix = session.kind === 'path' && session.value.match(new RegExp(`(${SESSION_ID_RE.source.slice(1, -1)})\\.jsonl$`, 'i'));
  if (suffix) return suffix[1];
  throw new Error('Pi session reference has no strict UUID');
}

export async function collectSessionResult(filePath) {
  return extractFinalizedResult(await readJsonlEntries(filePath));
}

export async function collectSessionResultAfterBoundary(filePath, boundary) {
  return (await collectSessionResultAfterBoundaryData(filePath, boundary)).result;
}

async function collectSessionResultAfterBoundaryData(filePath, boundary) {
  validateBoundary(boundary);
  const data = await readJsonlRecords(filePath);
  const validEntries = data.records.filter((record) => isPlainObject(record.value)).map((record) => record.value);
  const headers = validEntries.filter((entry) => entry.type === 'session' && typeof entry.id === 'string');
  if (headers.length !== 1 || headers[0].id !== boundary.sessionId) throw new Error('Pi session header changed or is invalid');
  if (data.completeLineCount < boundary.lineCount) throw new Error('Pi prompt boundary line count was truncated');

  if (boundary.anchorId === null) {
    if (boundary.lineCount !== 0 || boundary.anchorLine !== null) throw new Error('Pi prompt boundary anchor is missing');
  } else {
    const anchors = data.records.filter((record) => record.value?.id === boundary.anchorId);
    if (anchors.length !== 1 || anchors[0].lineNumber !== boundary.anchorLine || boundary.anchorLine > boundary.lineCount) throw new Error('Pi prompt boundary anchor is missing or ambiguous, or moved from its original boundary');
  }

  const postBoundaryRecords = data.records.slice(boundary.lineCount);
  for (const record of postBoundaryRecords) {
    if (!isPlainObject(record.value)) throw new Error(`post-boundary JSONL line ${record.lineNumber} is invalid`);
    if (typeof record.value.id !== 'string' || !record.value.id) throw new Error(`post-boundary JSONL line ${record.lineNumber} has no valid entry ID`);
  }
  if (data.trailingFragment.length) throw new RetryableSessionError('Pi session has a trailing post-boundary fragment', data.completeLineCount);
  const entries = postBoundaryRecords.map((record) => record.value).filter((entry) => entry.type !== 'session');
  for (const entry of entries) validatePostBoundaryMessage(entry);
  return { result: extractPostBoundaryResult(entries), completeLineCount: data.completeLineCount };
}

function validateBoundary(boundary) {
  if (!isPlainObject(boundary) || Object.keys(boundary).some((key) => !['sessionId', 'anchorId', 'anchorLine', 'lineCount'].includes(key)) || !SESSION_ID_RE.test(boundary.sessionId) || !Number.isSafeInteger(boundary.lineCount) || boundary.lineCount < 0) throw new Error('invalid prompt boundary');
  if (boundary.lineCount === 0) {
    if (boundary.anchorId !== null || boundary.anchorLine !== null) throw new Error('invalid prompt boundary');
  } else if (typeof boundary.anchorId !== 'string' || !boundary.anchorId || !Number.isSafeInteger(boundary.anchorLine) || boundary.anchorLine < 1 || boundary.anchorLine > boundary.lineCount) throw new Error('invalid prompt boundary');
}

export async function waitForFinalizedSessionResult(filePath, timeoutMs = 10000, signal, boundary) {
  const deadline = Date.now() + timeoutMs;
  let lastError;
  let trailingLineCount;
  while (Date.now() < deadline) {
    if (signal?.aborted) throw new Error('collection aborted');
    try {
      const collected = boundary ? await collectSessionResultAfterBoundaryData(filePath, boundary) : { result: await collectSessionResult(filePath) };
      if (trailingLineCount !== undefined && collected.completeLineCount > trailingLineCount + 1) throw new Error('Pi session trailing post-boundary fragment was followed by later data');
      if (collected.result.stopReason !== 'incomplete') return collected.result;
      lastError = undefined;
    } catch (error) {
      if (!(error instanceof RetryableSessionError) && error?.code !== 'ENOENT') throw error;
      if (error instanceof RetryableSessionError) {
        if (trailingLineCount !== undefined && error.completeLineCount > trailingLineCount) throw new Error('Pi session trailing post-boundary fragment was followed by later data');
        trailingLineCount ??= error.completeLineCount;
      }
      lastError = error;
    }
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

function isPlainObject(value) {
  return !!value && typeof value === 'object' && !Array.isArray(value);
}

/** Reads complete physical JSONL records while retaining parse and boundary metadata. */
async function readJsonlRecords(filePath) {
  const records = [];
  let bytes = 0;
  let completeLineCount = 0;
  return new Promise((resolvePromise, reject) => {
    const stream = createReadStream(filePath, { encoding: 'utf8' });
    let buffer = '';
    let settled = false;
    const fail = (error) => {
      if (settled) return;
      settled = true;
      stream.destroy();
      reject(error);
    };
    const parse = (line) => {
      completeLineCount++;
      if (completeLineCount > MAX_SESSION_LINES) return fail(new Error(`session file exceeds maximum lines (${MAX_SESSION_LINES})`));
      try { records.push({ lineNumber: completeLineCount, value: JSON.parse(line.replace(/\r$/, '')) }); }
      catch (error) { records.push({ lineNumber: completeLineCount, error }); }
    };
    stream.on('data', (chunk) => {
      if (settled) return;
      bytes += Buffer.byteLength(chunk);
      if (bytes > MAX_SESSION_BYTES) return fail(new Error(`session file exceeds maximum size (${MAX_SESSION_BYTES} bytes)`));
      buffer += chunk;
      const parts = buffer.split('\n');
      buffer = parts.pop() ?? '';
      for (const line of parts) parse(line);
    });
    stream.on('end', () => {
      if (settled) return;
      settled = true;
      resolvePromise({ records, completeLineCount, trailingFragment: buffer });
    });
    stream.on('error', (error) => {
      if (settled) return;
      const wrapped = new Error(`failed to read session file: ${error.message}`);
      wrapped.code = error.code;
      settled = true;
      reject(wrapped);
    });
  });
}

/** Reads only valid complete newline-terminated JSONL values. */
export async function readJsonlEntries(filePath) {
  const data = await readJsonlRecords(filePath);
  return data.records.filter((record) => record.value !== undefined).map((record) => record.value);
}

const STOP_REASONS = new Set(['stop', 'length', 'toolUse', 'error', 'aborted']);
const EXTENDED_ROLES = new Set(['bashExecution', 'custom', 'branchSummary', 'compactionSummary']);

function validContentArray(content, allowedParts) {
  return Array.isArray(content) && content.every((part) => isPlainObject(part) && allowedParts(part));
}
function validUserPart(part) {
  return (part.type === 'text' && typeof part.text === 'string') || (part.type === 'image' && typeof part.data === 'string' && typeof part.mimeType === 'string');
}
function validAssistantPart(part) {
  return (part.type === 'text' && typeof part.text === 'string')
    || (part.type === 'thinking' && typeof part.thinking === 'string')
    || (part.type === 'toolCall' && typeof part.id === 'string' && !!part.id && typeof part.name === 'string' && !!part.name && isPlainObject(part.arguments));
}
function postBoundaryMessageError(reason) { throw new Error(`post-boundary message is invalid: ${reason}`); }
function validatePostBoundaryMessage(entry) {
  if (entry.type !== 'message') return;
  const message = entry.message;
  if (!isPlainObject(message)) postBoundaryMessageError('message must be an object');
  if (typeof message.role !== 'string') postBoundaryMessageError('role is missing');
  if (message.role === 'user') {
    if (typeof message.content !== 'string' && !validContentArray(message.content, validUserPart)) postBoundaryMessageError('user content');
  } else if (message.role === 'assistant') {
    if (!validContentArray(message.content, validAssistantPart) || !STOP_REASONS.has(message.stopReason)) postBoundaryMessageError('assistant content or stop reason');
  } else if (message.role === 'toolResult') {
    if (typeof message.toolCallId !== 'string' || !message.toolCallId || typeof message.toolName !== 'string' || !message.toolName || !validContentArray(message.content, validUserPart) || typeof message.isError !== 'boolean') postBoundaryMessageError('tool result');
  } else if (message.role === 'bashExecution') {
    if (typeof message.command !== 'string' || typeof message.output !== 'string' || (message.exitCode !== undefined && !Number.isSafeInteger(message.exitCode)) || typeof message.cancelled !== 'boolean' || typeof message.truncated !== 'boolean') postBoundaryMessageError('bash execution');
  } else if (message.role === 'custom') {
    if (typeof message.customType !== 'string' || !message.customType || (typeof message.content !== 'string' && !validContentArray(message.content, validUserPart)) || typeof message.display !== 'boolean') postBoundaryMessageError('custom message');
  } else if (message.role === 'branchSummary') {
    if (typeof message.summary !== 'string' || typeof message.fromId !== 'string' || !message.fromId) postBoundaryMessageError('branch summary');
  } else if (message.role === 'compactionSummary') {
    if (typeof message.summary !== 'string' || !Number.isSafeInteger(message.tokensBefore)) postBoundaryMessageError('compaction summary');
  } else if (!EXTENDED_ROLES.has(message.role)) postBoundaryMessageError('unknown role');
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
  if (userIndexes.length > 1) throw new Error('Pi session contains a second post-boundary user message');
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
