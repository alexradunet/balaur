import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import { addHandle, createHandle, createHandleStore, deserializeStore, getHandle, reconcileHandles, serializeStore } from '../handle-store.js';

const sessionId = '11111111-1111-1111-1111-111111111111';
function stored() { let store = createHandleStore(); const handle = { ...createHandle({ paneId: 'w1:p2', terminalId: 'term-2', role: 'executor', agentName: 'worker', sessionKind: 'id', sessionValue: sessionId }), status: 'ready' }; return { handle, store: addHandle(store, handle) }; }
describe('handle persistence and exact reconciliation', () => {
  it('round trips the started agent name and exact session identity', () => { const { store, handle } = stored(); const restored = deserializeStore(serializeStore(store)); assert.equal(getHandle(restored, handle.handleId).agentName, 'worker'); assert.equal(getHandle(restored, handle.handleId).sessionValue, sessionId); });
  it('marks absent workers missing', () => { const { store, handle } = stored(); assert.equal(getHandle(reconcileHandles(store, []), handle.handleId).status, 'missing'); });
  it('marks same-pane replacement replaced rather than rebinding', () => { const { store, handle } = stored(); const agents = [{ pane_id: 'w1:p2', terminal_id: 'term-2', name: 'worker', agent_session: { kind: 'id', value: 'new-id' } }]; assert.equal(getHandle(reconcileHandles(store, agents), handle.handleId).status, 'replaced'); });
  it('marks terminal-only and exact session-pair-only inventory conflicts replaced', () => {
    const { store, handle } = stored();
    assert.equal(getHandle(reconcileHandles(store, [{ pane_id: 'other', terminal_id: 'term-2', name: 'other' }]), handle.handleId).status, 'replaced');
    assert.equal(getHandle(reconcileHandles(store, [{ pane_id: 'other', terminal_id: 'other', name: 'other', agent_session: { kind: 'id', value: sessionId } }]), handle.handleId).status, 'replaced');
  });
  it('does not treat the same session value under a different kind as a paired-session conflict', () => {
    const { store, handle } = stored();
    const inventory = [{ pane_id: 'other', terminal_id: 'other', name: 'other', agent_session: { kind: 'path', value: sessionId } }];
    assert.equal(getHandle(reconcileHandles(store, inventory), handle.handleId).status, 'missing');
  });
  it('preserves ready status only for exact pane/name/session match even with an unnamed lead row', () => { const { store, handle } = stored(); const agents = [{ pane_id: 'w1:p1' }, { pane_id: 'w1:p2', terminal_id: 'term-2', name: 'worker', agent_session: { kind: 'id', value: sessionId } }]; assert.equal(getHandle(reconcileHandles(store, agents), handle.handleId).status, 'ready'); });
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

  it('rejects calendar-invalid createdAt and updatedAt during serialization and deserialization', () => {
    const impossible = ['2026-02-29T12:00:00.000Z', '2026-04-31T12:00:00Z', '2026-13-01T12:00:00Z', '2026-01-01T24:00:00Z'];
    for (const field of ['createdAt', 'updatedAt']) for (const timestamp of impossible) {
      const { store, handle } = stored();
      store.handles[handle.handleId][field] = timestamp;
      assert.throws(() => serializeStore(store), new RegExp(field));
      assert.throws(() => deserializeStore(JSON.stringify(store)), new RegExp(field));
    }
    const { store, handle } = stored();
    store.handles[handle.handleId].createdAt = '2024-02-29T23:59:59.123Z';
    store.handles[handle.handleId].updatedAt = new Date().toISOString();
    assert.doesNotThrow(() => deserializeStore(serializeStore(store)));
    store.handles[handle.handleId].createdAt = '0000-02-29T00:00:00.000Z';
    assert.doesNotThrow(() => deserializeStore(serializeStore(store)), 'accepts a four-digit leap year emitted by Date#toISOString');
  });

  it('enforces exact prompt-boundary shape, relationships, and paired session agreement', () => {
    const { handle } = stored();
    const snapshot = (changes) => JSON.stringify({ version: 1, handles: { [handle.handleId]: { ...handle, promptPhase: 'accepted', ...changes } } });
    const valid = { sessionId, anchorId: 'anchor', anchorLine: 2, lineCount: 3 };
    assert.doesNotThrow(() => deserializeStore(snapshot({ promptBoundary: valid })));
    for (const promptBoundary of [
      { sessionId, anchorId: null, anchorLine: null, lineCount: 1 },
      { sessionId, anchorId: 'anchor', anchorLine: null, lineCount: 1 },
      { sessionId, anchorId: 'anchor', anchorLine: 0, lineCount: 1 },
      { sessionId, anchorId: 'anchor', anchorLine: 2, lineCount: 1 },
      { sessionId, anchorId: null, anchorLine: null, lineCount: 0, extra: true },
    ]) assert.throws(() => deserializeStore(snapshot({ promptBoundary })), /promptBoundary/);
    assert.doesNotThrow(() => deserializeStore(snapshot({ promptBoundary: { sessionId, anchorId: null, anchorLine: null, lineCount: 0 } })));
    assert.throws(() => deserializeStore(snapshot({ sessionKind: undefined, sessionValue: undefined, promptBoundary: valid })), /session|prompt/);
    assert.throws(() => deserializeStore(snapshot({ sessionValue: '22222222-2222-2222-2222-222222222222', promptBoundary: valid })), /sessionId|agree/);
    const path = `/tmp/${sessionId}.jsonl`;
    assert.doesNotThrow(() => deserializeStore(snapshot({ sessionKind: 'path', sessionValue: path, promptBoundary: valid })));
    assert.throws(() => deserializeStore(snapshot({ sessionKind: 'path', sessionValue: '/tmp/not-the-session.jsonl', promptBoundary: valid })), /sessionId|agree/);
  });

  it('retains a provisional handle only for its exact pane, generated name, and terminal', () => {
    const provisional = createHandle({ paneId: 'w1:p2', terminalId: 'term-2', role: 'executor', agentName: 'generated' });
    const store = addHandle(createHandleStore(), provisional);
    assert.equal(getHandle(reconcileHandles(store, [{ pane_id: 'w1:p2', terminal_id: 'term-2', name: 'generated' }]), provisional.handleId).status, 'starting');
    assert.equal(getHandle(reconcileHandles(store, [{ pane_id: 'w1:p2', terminal_id: 'other', name: 'generated' }]), provisional.handleId).status, 'replaced');
  });
});
