import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import { addHandle, createHandle, createHandleStore, deserializeStore, getHandle, reconcileHandles, serializeStore } from '../handle-store.js';

function stored() { let store = createHandleStore(); const handle = { ...createHandle({ paneId: 'w1:p2', terminalId: 'term-2', role: 'executor', agentName: 'worker', sessionKind: 'id', sessionValue: 'id-1' }), status: 'ready' }; return { handle, store: addHandle(store, handle) }; }
describe('handle persistence and exact reconciliation', () => {
  it('round trips the started agent name and exact session identity', () => { const { store, handle } = stored(); const restored = deserializeStore(serializeStore(store)); assert.equal(getHandle(restored, handle.handleId).agentName, 'worker'); assert.equal(getHandle(restored, handle.handleId).sessionValue, 'id-1'); });
  it('marks absent workers missing', () => { const { store, handle } = stored(); assert.equal(getHandle(reconcileHandles(store, []), handle.handleId).status, 'missing'); });
  it('marks same-pane replacement replaced rather than rebinding', () => { const { store, handle } = stored(); const agents = [{ pane_id: 'w1:p2', terminal_id: 'term-2', name: 'worker', agent_session: { kind: 'id', value: 'new-id' } }]; assert.equal(getHandle(reconcileHandles(store, agents), handle.handleId).status, 'replaced'); });
  it('preserves ready status only for exact pane/name/session match even with an unnamed lead row', () => { const { store, handle } = stored(); const agents = [{ pane_id: 'w1:p1' }, { pane_id: 'w1:p2', terminal_id: 'term-2', name: 'worker', agent_session: { kind: 'id', value: 'id-1' } }]; assert.equal(getHandle(reconcileHandles(store, agents), handle.handleId).status, 'ready'); });
  it('rejects syntactically malformed and semantically corrupt snapshots atomically', () => {
    assert.throws(() => deserializeStore('{broken'), /valid JSON/);
    for (const value of [null, [], {}, { version: 1 }, { version: 1, handles: [] }, { version: 2, handles: {} }]) {
      assert.throws(() => deserializeStore(JSON.stringify(value)), /snapshot/);
    }
    const { store, handle } = stored();
    const other = { ...handle, handleId: 'bh-abcdef12', paneId: 'w1:p3' };
    const mixed = { version: 1, handles: { [handle.handleId]: handle, [other.handleId]: { ...other, terminalId: '' } } };
    assert.throws(() => deserializeStore(JSON.stringify(mixed)), /terminalId/);
  });

  it('rejects invalid status, session pairing, prompt boundary, unsupported version, and key mismatch', () => {
    const { store, handle } = stored();
    const corrupt = (changes, key = handle.handleId) => JSON.stringify({ version: 1, handles: { [key]: { ...handle, ...changes } } });
    assert.throws(() => deserializeStore(JSON.stringify({ version: 1, handles: { 'not-a-bridge-handle': { ...handle, handleId: 'not-a-bridge-handle' } } })), /handleId/);
    assert.throws(() => deserializeStore(corrupt({ status: 'paused' })), /status/);
    assert.throws(() => deserializeStore(corrupt({ sessionValue: undefined })), /sessionKind.*sessionValue|paired/);
    assert.throws(() => deserializeStore(corrupt({ promptPhase: 'accepted', promptBoundary: { sessionId: 'bad', anchorId: 'a', lineCount: 2 } })), /promptBoundary/);
    assert.throws(() => deserializeStore(JSON.stringify({ ...store, version: 9 })), /version/);
    assert.throws(() => deserializeStore(corrupt({}, 'bh-wrongkey')), /map key/);
  });

  it('accepts a deliberate empty snapshot and validates before serialization', () => {
    assert.deepEqual(deserializeStore('{"version":1,"handles":{}}'), createHandleStore());
    const { store, handle } = stored();
    store.handles[handle.handleId].createdAt = 'yesterday';
    assert.throws(() => serializeStore(store), /createdAt/);
  });

  it('retains a provisional handle only for its exact pane, generated name, and terminal', () => {
    const provisional = createHandle({ paneId: 'w1:p2', terminalId: 'term-2', role: 'executor', agentName: 'generated' });
    const store = addHandle(createHandleStore(), provisional);
    assert.equal(getHandle(reconcileHandles(store, [{ pane_id: 'w1:p2', terminal_id: 'term-2', name: 'generated' }]), provisional.handleId).status, 'starting');
    assert.equal(getHandle(reconcileHandles(store, [{ pane_id: 'w1:p2', terminal_id: 'other', name: 'generated' }]), provisional.handleId).status, 'replaced');
  });
});
