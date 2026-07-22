import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import {
  createHandleStore,
  createHandle,
  addHandle,
  updateHandle,
  removeHandle,
  getHandle,
  findHandleByPaneId,
  listHandles,
  reconcileHandles,
  serializeStore,
  deserializeStore,
} from '../handle-store.js';

describe('handle-store', () => {
  describe('createHandleStore', () => {
    it('creates an empty store', () => {
      const store = createHandleStore();
      assert.deepEqual(store.handles, {});
      assert.ok(store.version > 0);
    });
  });

  describe('createHandle', () => {
    it('creates a handle with required fields', () => {
      const h = createHandle({ paneId: 'w1-2', role: 'implementer' });
      assert.ok(h.handleId.startsWith('bh-'));
      assert.equal(h.paneId, 'w1-2');
      assert.equal(h.role, 'implementer');
      assert.equal(h.status, 'starting');
      assert.ok(h.createdAt);
      assert.ok(h.updatedAt);
    });

    it('includes optional fields', () => {
      const h = createHandle({
        paneId: 'w1-2',
        role: 'reviewer',
        workspaceId: 'w1',
        worktreePath: '/tmp/work',
      });
      assert.equal(h.workspaceId, 'w1');
      assert.equal(h.worktreePath, '/tmp/work');
    });

    it('throws on missing paneId', () => {
      assert.throws(() => createHandle({ role: 'test' }), /paneId is required/);
    });

    it('throws on missing role', () => {
      assert.throws(() => createHandle({ paneId: 'w1-2' }), /role is required/);
    });
  });

  describe('addHandle / getHandle', () => {
    it('adds and retrieves a handle', () => {
      let store = createHandleStore();
      const h = createHandle({ paneId: 'w1-2', role: 'test' });
      store = addHandle(store, h);
      const retrieved = getHandle(store, h.handleId);
      assert.ok(retrieved);
      assert.equal(retrieved.role, 'test');
    });

    it('returns undefined for unknown handle', () => {
      const store = createHandleStore();
      assert.equal(getHandle(store, 'unknown'), undefined);
    });

    it('returns a copy', () => {
      let store = createHandleStore();
      const h = createHandle({ paneId: 'w1-2', role: 'test' });
      store = addHandle(store, h);
      const retrieved = getHandle(store, h.handleId);
      retrieved.status = 'done';
      const original = getHandle(store, h.handleId);
      assert.equal(original.status, 'starting');
    });
  });

  describe('updateHandle', () => {
    it('updates status and fields', () => {
      let store = createHandleStore();
      const h = createHandle({ paneId: 'w1-2', role: 'test' });
      store = addHandle(store, h);
      store = updateHandle(store, h.handleId, { status: 'ready', agentName: 'pi-test' });
      const retrieved = getHandle(store, h.handleId);
      assert.equal(retrieved.status, 'ready');
      assert.equal(retrieved.agentName, 'pi-test');
      assert.ok(retrieved.updatedAt >= h.updatedAt);
    });

    it('does not override handleId or paneId', () => {
      let store = createHandleStore();
      const h = createHandle({ paneId: 'w1-2', role: 'test' });
      store = addHandle(store, h);
      store = updateHandle(store, h.handleId, {
        handleId: 'different',
        paneId: 'different',
      });
      const retrieved = getHandle(store, h.handleId);
      assert.equal(retrieved.handleId, h.handleId);
      assert.equal(retrieved.paneId, 'w1-2');
    });

    it('is a no-op for unknown handle', () => {
      const store = createHandleStore();
      const updated = updateHandle(store, 'unknown', { status: 'done' });
      assert.equal(updated, store);
    });
  });

  describe('removeHandle', () => {
    it('removes a handle', () => {
      let store = createHandleStore();
      const h = createHandle({ paneId: 'w1-2', role: 'test' });
      store = addHandle(store, h);
      store = removeHandle(store, h.handleId);
      assert.equal(getHandle(store, h.handleId), undefined);
    });
  });

  describe('findHandleByPaneId', () => {
    it('finds handle by pane ID', () => {
      let store = createHandleStore();
      const h1 = createHandle({ paneId: 'w1-1', role: 'a' });
      const h2 = createHandle({ paneId: 'w1-2', role: 'b' });
      store = addHandle(store, h1);
      store = addHandle(store, h2);
      const found = findHandleByPaneId(store, 'w1-2');
      assert.equal(found?.handleId, h2.handleId);
    });

    it('returns undefined for unknown pane', () => {
      const store = createHandleStore();
      assert.equal(findHandleByPaneId(store, 'unknown'), undefined);
    });
  });

  describe('listHandles', () => {
    it('lists all handles', () => {
      let store = createHandleStore();
      store = addHandle(store, createHandle({ paneId: 'w1-1', role: 'a' }));
      store = addHandle(store, createHandle({ paneId: 'w1-2', role: 'b' }));
      const list = listHandles(store);
      assert.equal(list.length, 2);
    });

    it('returns copies', () => {
      let store = createHandleStore();
      const h = createHandle({ paneId: 'w1-1', role: 'a' });
      store = addHandle(store, h);
      const list = listHandles(store);
      list[0].status = 'done';
      assert.equal(listHandles(store)[0].status, 'starting');
    });
  });

  describe('reconcileHandles', () => {
    it('marks missing panes as missing', () => {
      let store = createHandleStore();
      const h = createHandle({ paneId: 'w1-1', role: 'a' });
      store = addHandle(store, h);
      // Pane list doesn't include w1-1
      store = reconcileHandles(store, [{ pane_id: 'w1-2' }]);
      assert.equal(getHandle(store, h.handleId).status, 'missing');
    });

    it('does not change status for existing panes', () => {
      let store = createHandleStore();
      const h = createHandle({ paneId: 'w1-1', role: 'a' });
      store = addHandle(store, h);
      store = reconcileHandles(store, [{ pane_id: 'w1-1' }]);
      assert.equal(getHandle(store, h.handleId).status, 'starting');
    });

    it('does not re-bind handles to different panes', () => {
      let store = createHandleStore();
      const h = createHandle({ paneId: 'w1-1', role: 'a' });
      store = addHandle(store, h);
      store = updateHandle(store, h.handleId, { status: 'ready' });
      // Pane w1-1 exists but handle status remains ready, not re-bound
      store = reconcileHandles(store, [{ pane_id: 'w1-1' }]);
      assert.equal(getHandle(store, h.handleId).status, 'ready');
    });

    it('does not mark already-missing handles again', () => {
      let store = createHandleStore();
      const h = createHandle({ paneId: 'w1-1', role: 'a' });
      store = addHandle(store, h);
      store = updateHandle(store, h.handleId, { status: 'missing' });
      store = reconcileHandles(store, []);
      // Should stay missing, not error
      assert.equal(getHandle(store, h.handleId).status, 'missing');
    });
  });

  describe('serializeStore / deserializeStore', () => {
    it('round-trips a store', () => {
      let store = createHandleStore();
      store = addHandle(store, createHandle({ paneId: 'w1-1', role: 'a' }));
      const json = serializeStore(store);
      const restored = deserializeStore(json);
      assert.equal(Object.keys(restored.handles).length, 1);
    });

    it('handles malformed JSON gracefully', () => {
      const restored = deserializeStore('not json');
      assert.deepEqual(restored.handles, {});
    });

    it('handles invalid handle structures', () => {
      const restored = deserializeStore(JSON.stringify({
        handles: {
          bad: { foo: 'bar' }, // missing required fields
          good: { handleId: 'good', paneId: 'w1', role: 'test', status: 'starting' },
        },
        version: 1,
      }));
      assert.equal(Object.keys(restored.handles).length, 1);
      assert.ok(restored.handles.good);
    });
  });
});
