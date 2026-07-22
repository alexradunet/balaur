/**
 * Worker handle persistence and reconciliation.
 *
 * Stores launched worker handles so they can be reconstructed after
 * session reload/resume. Marks missing or replaced panes without
 * silently rebinding them.
 *
 * @module handle-store
 */

import { randomUUID } from 'node:crypto';

/**
 * @typedef {Object} WorkerHandle
 * @property {string} handleId       - Stable bridge-local handle identifier.
 * @property {string} paneId         - Herdr pane ID.
 * @property {string} role           - Role name from .pi/agents/*.md.
 * @property {string} [workspaceId]  - Herdr workspace ID.
 * @property {string} [worktreePath] - Working directory / worktree path.
 * @property {string} [agentName]    - Herdr agent name (generated before launch).
 * @property {string} terminalId     - Exact terminal created by pane.split.
 * @property {'id'|'path'} [sessionKind] - Herdr agent session reference kind.
 * @property {string} [sessionValue] - Exact Herdr agent session reference value.
 * @property {'starting'|'ready'|'working'|'idle'|'blocked'|'done'|'unknown'|'error'|'replaced'|'missing'} status
 * @property {'submitting'|'accepted'|'uncertain'} [promptPhase]
 * @property {{sessionId:string,anchorId:string|null,anchorLine:number|null,lineCount:number}} [promptBoundary]
 * @property {string} createdAt      - ISO 8601 creation timestamp.
 * @property {string} [updatedAt]    - ISO 8601 last update timestamp.
 * @property {string} [error]        - Error message if status is 'error'.
 */

/**
 * @typedef {Object} HandleStoreState
 * @property {Record<string, WorkerHandle>} handles
 * @property {number} version
 */

const STORE_VERSION = 1;
const HANDLE_STATUSES = new Set(['starting', 'ready', 'working', 'idle', 'blocked', 'done', 'unknown', 'error', 'replaced', 'missing']);
const PROMPT_PHASES = new Set(['submitting', 'accepted', 'uncertain']);
const HANDLE_KEYS = new Set(['handleId', 'paneId', 'role', 'workspaceId', 'worktreePath', 'agentName', 'terminalId', 'sessionKind', 'sessionValue', 'status', 'promptPhase', 'promptBoundary', 'createdAt', 'updatedAt', 'error']);
const SESSION_ID_RE = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

function isPlainObject(value) {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return false;
  const prototype = Object.getPrototypeOf(value);
  return prototype === Object.prototype || prototype === null;
}
function requireNonEmptyString(value, label) {
  if (typeof value !== 'string' || !value) throw new Error(`handle snapshot ${label} must be a non-empty string`);
}
function requireOptionalString(value, label) {
  if (value !== undefined) requireNonEmptyString(value, label);
}
function requireTimestamp(value, label) {
  requireNonEmptyString(value, label);
  const match = /^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2}):(\d{2})(?:\.\d+)?Z$/.exec(value);
  if (!match) throw new Error(`handle snapshot ${label} must be an ISO 8601 instant`);
  const [, year, month, day, hour, minute, second] = match.map(Number);
  const leapYear = year % 4 === 0 && (year % 100 !== 0 || year % 400 === 0);
  const daysInMonth = [0, 31, leapYear ? 29 : 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31][month] || 0;
  if (day < 1 || day > daysInMonth || hour > 23 || minute > 59 || second > 59) throw new Error(`handle snapshot ${label} must be an ISO 8601 instant`);
}
function sessionIdForHandle(handle) {
  if (handle.sessionKind === 'id') return handle.sessionValue;
  const suffix = handle.sessionKind === 'path' && handle.sessionValue?.match(new RegExp(`(${SESSION_ID_RE.source.slice(1, -1)})\\.jsonl$`, 'i'));
  return suffix?.[1];
}
function validateBoundary(boundary, label, handle) {
  if (!isPlainObject(boundary) || Object.keys(boundary).some((key) => !['sessionId', 'anchorId', 'anchorLine', 'lineCount'].includes(key))) throw new Error(`handle snapshot ${label} is invalid`);
  if (!SESSION_ID_RE.test(boundary.sessionId)) throw new Error(`handle snapshot ${label}.sessionId is invalid`);
  if (!Number.isSafeInteger(boundary.lineCount) || boundary.lineCount < 0) throw new Error(`handle snapshot ${label}.lineCount is invalid`);
  if (boundary.lineCount === 0) {
    if (boundary.anchorId !== null || boundary.anchorLine !== null) throw new Error(`handle snapshot ${label} has an invalid header-only boundary`);
  } else {
    requireNonEmptyString(boundary.anchorId, `${label}.anchorId`);
    if (!Number.isSafeInteger(boundary.anchorLine) || boundary.anchorLine < 1 || boundary.anchorLine > boundary.lineCount) throw new Error(`handle snapshot ${label}.anchorLine is invalid`);
  }
  if (!handle.sessionKind || boundary.sessionId !== sessionIdForHandle(handle)) throw new Error(`handle snapshot ${label}.sessionId does not agree with handle session identity`);
}
function validateHandle(handle, mapKey) {
  if (!isPlainObject(handle)) throw new Error(`handle snapshot ${mapKey} must be a plain object`);
  for (const key of Object.keys(handle)) if (!HANDLE_KEYS.has(key)) throw new Error(`handle snapshot ${mapKey} has unsupported field '${key}'`);
  for (const key of ['handleId', 'paneId', 'role', 'terminalId']) requireNonEmptyString(handle[key], `${mapKey}.${key}`);
  if (!/^bh-[0-9a-f]{8}$/.test(handle.handleId)) throw new Error(`handle snapshot ${mapKey}.handleId is invalid`);
  if (handle.handleId !== mapKey) throw new Error(`handle snapshot map key ${mapKey} does not match handleId ${handle.handleId}`);
  for (const key of ['workspaceId', 'worktreePath', 'agentName', 'error']) requireOptionalString(handle[key], `${mapKey}.${key}`);
  if (!HANDLE_STATUSES.has(handle.status)) throw new Error(`handle snapshot ${mapKey}.status is invalid`);
  requireTimestamp(handle.createdAt, `${mapKey}.createdAt`);
  if (handle.updatedAt !== undefined) requireTimestamp(handle.updatedAt, `${mapKey}.updatedAt`);
  const hasSessionKind = handle.sessionKind !== undefined;
  const hasSessionValue = handle.sessionValue !== undefined;
  if (hasSessionKind !== hasSessionValue) throw new Error(`handle snapshot ${mapKey} sessionKind and sessionValue must be paired`);
  if (hasSessionKind) {
    if (handle.sessionKind !== 'id' && handle.sessionKind !== 'path') throw new Error(`handle snapshot ${mapKey}.sessionKind is invalid`);
    requireNonEmptyString(handle.sessionValue, `${mapKey}.sessionValue`);
  }
  const hasPhase = handle.promptPhase !== undefined;
  const hasBoundary = handle.promptBoundary !== undefined;
  if (hasPhase !== hasBoundary) throw new Error(`handle snapshot ${mapKey} promptPhase and promptBoundary must be paired`);
  if (hasPhase) {
    if (!PROMPT_PHASES.has(handle.promptPhase)) throw new Error(`handle snapshot ${mapKey}.promptPhase is invalid`);
    validateBoundary(handle.promptBoundary, `${mapKey}.promptBoundary`, handle);
  }
}
function validateStore(store) {
  if (!isPlainObject(store) || Object.keys(store).some((key) => key !== 'version' && key !== 'handles')) throw new Error('handle store snapshot must be a plain versioned object');
  if (store.version !== STORE_VERSION) throw new Error(`handle store snapshot version must be ${STORE_VERSION}`);
  if (!isPlainObject(store.handles)) throw new Error('handle store snapshot handles must be a plain object');
  for (const [key, handle] of Object.entries(store.handles)) validateHandle(handle, key);
  return store;
}

/**
 * Create a new empty handle store.
 * @returns {HandleStoreState}
 */
export function createHandleStore() {
  return { handles: {}, version: STORE_VERSION };
}

/**
 * Create a new worker handle.
 *
 * @param {Object} opts
 * @param {string} opts.paneId
 * @param {string} opts.role
 * @param {string} opts.terminalId
 * @param {string} [opts.workspaceId]
 * @param {string} [opts.worktreePath]
 * @param {string} [opts.agentName]
 * @param {'id'|'path'} [opts.sessionKind]
 * @param {string} [opts.sessionValue]
 * @returns {WorkerHandle}
 */
export function createHandle(opts) {
  if (!opts.paneId) throw new Error('paneId is required');
  if (!opts.role) throw new Error('role is required');
  if (!opts.terminalId) throw new Error('terminalId is required');

  const now = new Date().toISOString();
  return {
    handleId: `bh-${randomUUID().slice(0, 8)}`,
    paneId: opts.paneId,
    role: opts.role,
    workspaceId: opts.workspaceId,
    worktreePath: opts.worktreePath,
    agentName: opts.agentName,
    terminalId: opts.terminalId,
    sessionKind: opts.sessionKind,
    sessionValue: opts.sessionValue,
    status: 'starting',
    createdAt: now,
    updatedAt: now,
  };
}

/**
 * Add a handle to the store.
 *
 * @param {HandleStoreState} store
 * @param {WorkerHandle} handle
 * @returns {HandleStoreState}
 */
export function addHandle(store, handle) {
  return {
    ...store,
    handles: { ...store.handles, [handle.handleId]: { ...handle } },
  };
}

/**
 * Update a handle's status and optional fields.
 *
 * @param {HandleStoreState} store
 * @param {string} handleId
 * @param {Partial<WorkerHandle>} updates
 * @returns {HandleStoreState}
 */
export function updateHandle(store, handleId, updates) {
  const existing = store.handles[handleId];
  if (!existing) return store;

  return {
    ...store,
    handles: {
      ...store.handles,
      [handleId]: {
        ...existing,
        ...updates,
        handleId: existing.handleId, // prevent handleId override
        paneId: existing.paneId,     // prevent paneId override
        updatedAt: new Date().toISOString(),
      },
    },
  };
}

/**
 * Remove a handle from the store.
 *
 * @param {HandleStoreState} store
 * @param {string} handleId
 * @returns {HandleStoreState}
 */
export function removeHandle(store, handleId) {
  const { [handleId]: _, ...rest } = store.handles;
  return { ...store, handles: rest };
}

/**
 * Get a handle by ID.
 *
 * @param {HandleStoreState} store
 * @param {string} handleId
 * @returns {WorkerHandle|undefined}
 */
export function getHandle(store, handleId) {
  const h = store.handles[handleId];
  return h ? { ...h } : undefined;
}

/**
 * Find a handle by pane ID.
 *
 * @param {HandleStoreState} store
 * @param {string} paneId
 * @returns {WorkerHandle|undefined}
 */
export function findHandleByPaneId(store, paneId) {
  for (const h of Object.values(store.handles)) {
    if (h.paneId === paneId) return { ...h };
  }
  return undefined;
}

/**
 * List all handles.
 *
 * @param {HandleStoreState} store
 * @returns {WorkerHandle[]}
 */
export function listHandles(store) {
  return Object.values(store.handles).map((h) => ({ ...h }));
}

/**
 * Reconcile handles against the current pane list from Herdr.
 * Marks handles whose panes are missing or have been replaced by a
 * different process. Does NOT silently rebind handles to new panes.
 *
 * @param {HandleStoreState} store
 * @param {Array<{ pane_id: string, name?: string, agent_session?: { kind: string, value: string } }>} currentPanes
 * @returns {HandleStoreState}
 */
export function classifyHandleInventory(handle, currentPanes) {
  const exact = currentPanes.some((agent) => agent.pane_id === handle.paneId && agent.name === handle.agentName && agent.terminal_id === handle.terminalId && (!handle.sessionKind || (agent.agent_session?.kind === handle.sessionKind && agent.agent_session?.value === handle.sessionValue)));
  if (exact) return 'exact';
  const conflict = currentPanes.some((agent) => agent.pane_id === handle.paneId
    || (handle.agentName && agent.name === handle.agentName)
    || agent.terminal_id === handle.terminalId
    || (handle.sessionKind && agent.agent_session?.kind === handle.sessionKind && agent.agent_session?.value === handle.sessionValue));
  return conflict ? 'replaced' : 'missing';
}

export function reconcileHandles(store, currentPanes) {
  let updated = store;
  for (const handle of Object.values(store.handles)) {
    const status = classifyHandleInventory(handle, currentPanes);
    if (status !== 'exact') updated = updateHandle(updated, handle.handleId, { status });
  }
  return updated;
}

/**
 * Serialize the store for persistence in tool result details.
 *
 * @param {HandleStoreState} store
 * @returns {string}
 */
export function serializeStore(store) {
  validateStore(store);
  return JSON.stringify(store);
}

/**
 * Deserialize a store from persisted JSON.
 *
 * @param {string} json
 * @returns {HandleStoreState}
 */
export function deserializeStore(json) {
  let parsed;
  try { parsed = JSON.parse(json); }
  catch (error) { throw new Error(`handle store snapshot is not valid JSON: ${error.message}`); }
  validateStore(parsed);
  return parsed;
}
