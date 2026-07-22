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
 * @property {{sessionId:string,anchorId:string}} [promptBoundary]
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
export function reconcileHandles(store, currentPanes) {
  let updated = store;
  for (const handle of Object.values(store.handles)) {
    const samePane = currentPanes.filter((agent) => agent.pane_id === handle.paneId);
    const exact = samePane.find((agent) => agent.name === handle.agentName && agent.terminal_id === handle.terminalId && (!handle.sessionKind || (agent.agent_session?.kind === handle.sessionKind && agent.agent_session?.value === handle.sessionValue)));
    if (exact) continue;
    const replacement = samePane.length > 0 || currentPanes.some((agent) => agent.name === handle.agentName);
    updated = updateHandle(updated, handle.handleId, { status: replacement ? 'replaced' : 'missing' });
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
  return JSON.stringify(store);
}

/**
 * Deserialize a store from persisted JSON.
 *
 * @param {string} json
 * @returns {HandleStoreState}
 */
export function deserializeStore(json) {
  try {
    const parsed = JSON.parse(json);
    if (!parsed || typeof parsed !== 'object') return createHandleStore();
    if (!parsed.handles || typeof parsed.handles !== 'object') return createHandleStore();
    // Validate each handle has required fields
    const handles = {};
    for (const [id, handle] of Object.entries(parsed.handles)) {
      if (handle && typeof handle === 'object' && handle.handleId && handle.paneId && handle.role && handle.terminalId) {
        handles[id] = handle;
      }
    }
    return { handles, version: parsed.version ?? STORE_VERSION };
  } catch {
    return createHandleStore();
  }
}
