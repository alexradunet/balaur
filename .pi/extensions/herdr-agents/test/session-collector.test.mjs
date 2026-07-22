import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import { mkdtemp, mkdir, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';
import { captureSessionBoundary, collectSessionResult, collectSessionResultAfterBoundary, extractFinalizedResult, resolvePiSessionReference, waitForFinalizedSessionResult, waitForPiSessionReference } from '../session-collector.js';

const fixture = resolve('.pi/extensions/herdr-agents/test/fixtures/session.jsonl');
const usage = { input: 1, output: 1, cacheRead: 0, cacheWrite: 0, totalTokens: 2, cost: { input: 0, output: 0, cacheRead: 0, cacheWrite: 0, total: 0 } };

describe('Pi v3 session collection', () => {
  it('parses persisted type=message entries and associates results by toolCallId', async () => {
    const result = await collectSessionResult(fixture);
    assert.equal(result.stopReason, 'stop');
    assert.equal(result.text, 'Task completed successfully.');
    assert.equal(result.toolCalls.length, 1);
    assert.equal(result.toolCalls[0].id, 'call_1');
    assert.equal(result.toolCalls[0].result.text, 'ok');
    assert.equal(result.turns, 2);
  });

  it('binds collection to a post-boundary user turn instead of returning an old terminal result', async () => {
    const dir = await mkdtemp(join(tmpdir(), 'pi-boundary-'));
    const path = join(dir, 'session.jsonl');
    const header = { type: 'session', id: 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa' };
    const oldUser = { type: 'message', id: 'u-old', message: { role: 'user', content: 'old' } };
    const oldAnswer = { type: 'message', id: 'a-old', message: { role: 'assistant', content: [{ type: 'text', text: 'old answer' }], stopReason: 'stop', usage } };
    await writeFile(path, [header, oldUser, oldAnswer].map(JSON.stringify).join('\n') + '\n');
    try {
      const boundary = await captureSessionBoundary(path);
      await writeFile(path, JSON.stringify({ type: 'message', id: 'u-new', message: { role: 'user', content: 'new' } }) + '\n', { flag: 'a' });
      const partial = await collectSessionResultAfterBoundary(path, boundary);
      assert.equal(partial.stopReason, 'incomplete');
      await writeFile(path, JSON.stringify({ type: 'message', id: 'a-new', message: { role: 'assistant', content: [{ type: 'text', text: 'new answer' }], stopReason: 'stop', usage } }) + '\n', { flag: 'a' });
      assert.equal((await collectSessionResultAfterBoundary(path, boundary)).text, 'new answer');
    } finally { await rm(dir, { recursive: true, force: true }); }
  });

  it('fails closed for a changed header, ambiguous anchor, second user, and incomplete trailing record', async () => {
    const dir = await mkdtemp(join(tmpdir(), 'pi-boundary-invalid-')); const path = join(dir, 'session.jsonl');
    const header = { type: 'session', id: 'cccccccc-cccc-cccc-cccc-cccccccccccc' };
    await writeFile(path, [header, { type: 'message', id: 'anchor', message: { role: 'user', content: 'old' } }, { type: 'message', id: 'anchor', message: { role: 'assistant', content: [], stopReason: 'stop' } }].map(JSON.stringify).join('\n') + '\n');
    try {
      await assert.rejects(collectSessionResultAfterBoundary(path, { sessionId: header.id, anchorId: 'anchor' }), /anchor is missing or ambiguous/);
      await writeFile(path, [header, { type: 'message', id: 'one', message: { role: 'user', content: 'one' } }, { type: 'message', id: 'two', message: { role: 'user', content: 'two' } }].map(JSON.stringify).join('\n') + '\n');
      assert.equal((await collectSessionResultAfterBoundary(path, { sessionId: header.id, anchorId: header.id })).stopReason, 'incomplete');
      await writeFile(path, JSON.stringify({ type: 'session', id: 'different' }) + '\n' + JSON.stringify({ type: 'message', id: 'anchor', message: { role: 'user' } }) + '\n');
      await assert.rejects(collectSessionResultAfterBoundary(path, { sessionId: header.id, anchorId: 'anchor' }), /header changed/);
      await writeFile(path, [header, { type: 'message', id: 'anchor', message: { role: 'user' } }].map(JSON.stringify).join('\n') + '\n' + JSON.stringify({ type: 'message', id: 'partial', message: { role: 'assistant', stopReason: 'stop' } }));
      assert.equal((await collectSessionResultAfterBoundary(path, { sessionId: header.id, anchorId: 'anchor' })).stopReason, 'incomplete');
    } finally { await rm(dir, { recursive: true, force: true }); }
  });

  it('does not associate same-name tools without matching toolCallId', () => {
    const entries = [
      { type: 'message', message: { role: 'assistant', content: [{ type: 'toolCall', id: 'call-a', name: 'bash', arguments: {} }], stopReason: 'toolUse', usage } },
      { type: 'message', message: { role: 'toolResult', toolCallId: 'call-b', toolName: 'bash', content: [{ type: 'text', text: 'wrong' }], isError: false } },
      { type: 'message', message: { role: 'assistant', content: [{ type: 'text', text: 'final' }], stopReason: 'stop', usage } },
    ];
    const result = extractFinalizedResult(entries);
    assert.equal(result.toolCalls[0].result, undefined);
  });

  it('returns incomplete for only partial assistant output', () => {
    const result = extractFinalizedResult([{ type: 'message', message: { role: 'assistant', content: [{ type: 'text', text: 'partial' }] } }]);
    assert.equal(result.stopReason, 'incomplete');
  });

  it('treats a real Pi-v3 toolUse assistant turn as incomplete while retaining tool evidence', () => {
    const entries = [
      { type: 'message', message: { role: 'assistant', content: [{ type: 'toolCall', id: 'call_1', name: 'bash', arguments: { command: 'true' } }], stopReason: 'toolUse', usage } },
      { type: 'message', message: { role: 'toolResult', toolCallId: 'call_1', toolName: 'bash', content: [{ type: 'text', text: 'ok' }], isError: false } },
    ];
    const result = extractFinalizedResult(entries);
    assert.equal(result.stopReason, 'incomplete');
    assert.equal(result.turns, 1);
    assert.deepEqual(result.toolCalls, [{ id: 'call_1', name: 'bash', arguments: { command: 'true' }, result: { isError: false, text: 'ok' } }]);
  });

  it('resolves a session id only beneath the configured Pi session root', async () => {
    const root = await mkdtemp(join(tmpdir(), 'pi-sessions-'));
    const id = '22222222-2222-2222-2222-222222222222';
    const nested = join(root, 'project');
    await mkdir(nested);
    const path = join(nested, 'session.jsonl');
    await writeFile(path, JSON.stringify({ type: 'session', version: 3, id, cwd: '/test' }) + '\n');
    try {
      assert.equal(await resolvePiSessionReference({ kind: 'id', value: id }, root), path);
      await assert.rejects(resolvePiSessionReference({ kind: 'id', value: '../bad' }, root), /invalid Pi session ID/);
    } finally { await rm(root, { recursive: true, force: true }); }
  });

  it('retries a session id until its Pi file appears', async () => {
    const root = await mkdtemp(join(tmpdir(), 'pi-id-late-'));
    const id = '55555555-5555-5555-5555-555555555555';
    const dir = join(root, 'project'); await mkdir(dir);
    const path = join(dir, 'session.jsonl');
    setTimeout(() => void writeFile(path, JSON.stringify({ type: 'session', version: 3, id, cwd: '/test' }) + '\n'), 100);
    try { assert.equal(await waitForPiSessionReference({ kind: 'id', value: id }, 2000, undefined, root), path); }
    finally { await rm(root, { recursive: true, force: true }); }
  });

  it('retries a late-created final Pi session file', async () => {
    const root = await mkdtemp(join(tmpdir(), 'pi-late-'));
    const path = join(root, 'session.jsonl');
    setTimeout(() => void writeFile(path, JSON.stringify({ type: 'message', message: { role: 'assistant', content: [{ type: 'text', text: 'ready' }], stopReason: 'stop', usage } }) + '\n'), 100);
    try { assert.equal((await waitForFinalizedSessionResult(path, 2000)).text, 'ready'); }
    finally { await rm(root, { recursive: true, force: true }); }
  });

  it('honors AbortSignal while retrying collection', async () => {
    const controller = new AbortController();
    controller.abort();
    await assert.rejects(waitForFinalizedSessionResult('/missing/session.jsonl', 1000, controller.signal), /collection aborted/);
  });

  it('removes retry AbortSignal listeners after a normal sleep completion', async () => {
    const root = await mkdtemp(join(tmpdir(), 'pi-cleanup-')); const path = join(root, 'session.jsonl');
    let added = 0; let removed = 0; const listeners = new Set();
    const signal = { aborted: false, addEventListener: (_name, listener) => { added++; listeners.add(listener); }, removeEventListener: (_name, listener) => { removed++; listeners.delete(listener); } };
    setTimeout(() => void writeFile(path, JSON.stringify({ type: 'message', message: { role: 'assistant', content: [{ type: 'text', text: 'done' }], stopReason: 'stop', usage } }) + '\n'), 25);
    try { assert.equal((await waitForFinalizedSessionResult(path, 2000, signal)).text, 'done'); assert.equal(added, removed); assert.equal(listeners.size, 0); }
    finally { await rm(root, { recursive: true, force: true }); }
  });
});
