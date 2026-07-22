import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import { appendFile, mkdtemp, mkdir, rm, symlink, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';
import { captureResolvedSessionBoundary, captureSessionBoundary, collectSessionResult, collectSessionResultAfterBoundary, extractFinalizedResult, resolvePiSessionReference, waitForFinalizedSessionResult, waitForPiSessionReference } from '../session-collector.js';

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
    const path = join(dir, 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa.jsonl');
    const header = { type: 'session', id: 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa' };
    const oldUser = { type: 'message', id: 'u-old', message: { role: 'user', content: 'old' } };
    const oldAnswer = { type: 'message', id: 'a-old', message: { role: 'assistant', content: [{ type: 'text', text: 'old answer' }], stopReason: 'stop', usage } };
    await writeFile(path, [header, oldUser, oldAnswer].map(JSON.stringify).join('\n') + '\n');
    try {
      const boundary = await captureSessionBoundary({ kind: 'path', value: path });
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
      await assert.rejects(collectSessionResultAfterBoundary(path, { sessionId: header.id, anchorId: 'anchor', anchorLine: 2, lineCount: 3 }), /anchor is missing or ambiguous/);
      await writeFile(path, [header, { type: 'message', id: 'one', message: { role: 'user', content: 'one' } }, { type: 'message', id: 'two', message: { role: 'user', content: 'two' } }].map(JSON.stringify).join('\n') + '\n');
      await assert.rejects(collectSessionResultAfterBoundary(path, { sessionId: header.id, anchorId: header.id, anchorLine: 1, lineCount: 1 }), /second post-boundary user message/);
      await writeFile(path, JSON.stringify({ type: 'session', id: 'different' }) + '\n' + JSON.stringify({ type: 'message', id: 'anchor', message: { role: 'user' } }) + '\n');
      await assert.rejects(collectSessionResultAfterBoundary(path, { sessionId: header.id, anchorId: 'anchor', anchorLine: 2, lineCount: 2 }), /header changed/);
      await writeFile(path, [header, { type: 'message', id: 'anchor', message: { role: 'user' } }].map(JSON.stringify).join('\n') + '\n' + JSON.stringify({ type: 'message', id: 'partial', message: { role: 'assistant', stopReason: 'stop' } }));
      await assert.rejects(collectSessionResultAfterBoundary(path, { sessionId: header.id, anchorId: 'anchor', anchorLine: 2, lineCount: 2 }), /trailing post-boundary fragment/);
    } finally { await rm(dir, { recursive: true, force: true }); }
  });

  it('tolerates malformed complete history captured before the boundary', async () => {
    const dir = await mkdtemp(join(tmpdir(), 'pi-historical-corruption-')); const path = join(dir, 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa.jsonl');
    const header = { type: 'session', id: 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa' };
    await writeFile(path, `${JSON.stringify(header)}\n${JSON.stringify({ type: 'message', id: 'old', message: { role: 'assistant', content: [], stopReason: 'stop' } })}\n{bad historical json}\n`);
    try {
      const boundary = await captureSessionBoundary({ kind: 'path', value: path });
      assert.deepEqual(boundary, { sessionId: header.id, anchorId: 'old', anchorLine: 2, lineCount: 3 });
      await appendFile(path, `${JSON.stringify({ type: 'message', id: 'new-user', message: { role: 'user', content: 'new' } })}\n${JSON.stringify({ type: 'message', id: 'new-answer', message: { role: 'assistant', content: [{ type: 'text', text: 'safe' }], stopReason: 'stop', usage } })}\n`);
      assert.equal((await collectSessionResultAfterBoundary(path, boundary)).text, 'safe');
    } finally { await rm(dir, { recursive: true, force: true }); }
  });

  it('rejects a trailing fragment when capturing a prompt boundary', async () => {
    const dir = await mkdtemp(join(tmpdir(), 'pi-capture-fragment-')); const path = join(dir, 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa.jsonl');
    await writeFile(path, `${JSON.stringify({ type: 'session', id: 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa' })}\n{"type":"message"`);
    try { await assert.rejects(captureSessionBoundary({ kind: 'path', value: path }), /trailing fragment/); }
    finally { await rm(dir, { recursive: true, force: true }); }
  });

  it('fails closed across malformed or non-object post-boundary records and later terminal output', async () => {
    const dir = await mkdtemp(join(tmpdir(), 'pi-post-corruption-')); const path = join(dir, 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa.jsonl');
    const header = { type: 'session', id: 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa' };
    const boundary = { sessionId: header.id, anchorId: header.id, anchorLine: 1, lineCount: 1 };
    try {
      for (const corruptLine of ['{bad json}', '17', '[]']) {
        await writeFile(path, `${JSON.stringify(header)}\n${JSON.stringify({ type: 'message', id: 'user', message: { role: 'user', content: 'new' } })}\n${corruptLine}\n${JSON.stringify({ type: 'message', id: 'answer', message: { role: 'assistant', content: [{ type: 'text', text: 'must not escape' }], stopReason: 'stop', usage } })}\n`);
        await assert.rejects(collectSessionResultAfterBoundary(path, boundary), /post-boundary.*invalid/);
      }
    } finally { await rm(dir, { recursive: true, force: true }); }
  });

  it('treats trailing post-boundary data as retryable but throws if it remains at timeout', async () => {
    const dir = await mkdtemp(join(tmpdir(), 'pi-post-fragment-')); const path = join(dir, 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa.jsonl');
    const header = { type: 'session', id: 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa' };
    const boundary = { sessionId: header.id, anchorId: header.id, anchorLine: 1, lineCount: 1 };
    await writeFile(path, `${JSON.stringify(header)}\n${JSON.stringify({ type: 'message', id: 'user', message: { role: 'user', content: 'new' } })}\n{"type":"message"`);
    try {
      await assert.rejects(collectSessionResultAfterBoundary(path, boundary), /trailing post-boundary fragment/);
      await assert.rejects(waitForFinalizedSessionResult(path, 10, undefined, boundary), /trailing post-boundary fragment/);
      await appendFile(path, `\n${JSON.stringify({ type: 'message', id: 'answer', message: { role: 'assistant', content: [{ type: 'text', text: 'later' }], stopReason: 'stop', usage } })}\n`);
      await assert.rejects(collectSessionResultAfterBoundary(path, boundary), /post-boundary.*invalid/);
    } finally { await rm(dir, { recursive: true, force: true }); }
  });

  it('allows valid non-message entries with IDs but rejects missing IDs', async () => {
    const dir = await mkdtemp(join(tmpdir(), 'pi-post-metadata-')); const path = join(dir, 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa.jsonl');
    const header = { type: 'session', id: 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa' };
    const boundary = { sessionId: header.id, anchorId: header.id, anchorLine: 1, lineCount: 1 };
    try {
      await writeFile(path, [header, { type: 'message', id: 'user', message: { role: 'user', content: 'new' } }, { type: 'model_change', id: 'model-1', model: 'test' }, { type: 'message', id: 'answer', message: { role: 'assistant', content: [{ type: 'text', text: 'okay' }], stopReason: 'stop', usage } }].map(JSON.stringify).join('\n') + '\n');
      assert.equal((await collectSessionResultAfterBoundary(path, boundary)).text, 'okay');
      await writeFile(path, [header, { type: 'message', id: 'user', message: { role: 'user', content: 'new' } }, { type: 'model_change', model: 'test' }].map(JSON.stringify).join('\n') + '\n');
      await assert.rejects(collectSessionResultAfterBoundary(path, boundary), /valid entry ID/);
    } finally { await rm(dir, { recursive: true, force: true }); }
  });

  it('uses a header-only boundary for a fresh absent path and never returns a stale result', async () => {
    const dir = await mkdtemp(join(tmpdir(), 'pi-fresh-boundary-')); const path = join(dir, 'dddddddd-dddd-dddd-dddd-dddddddddddd.jsonl');
    const session = { kind: 'path', value: path };
    try {
      const boundary = await captureSessionBoundary(session);
      assert.deepEqual(boundary, { sessionId: 'dddddddd-dddd-dddd-dddd-dddddddddddd', anchorId: null, anchorLine: null, lineCount: 0 });
      await writeFile(path, [
        { type: 'session', id: boundary.sessionId },
        { type: 'message', id: 'u-first', message: { role: 'user', content: 'first' } },
        { type: 'message', id: 'a-first', message: { role: 'assistant', content: [{ type: 'text', text: 'fresh result' }], stopReason: 'stop', usage } },
      ].map(JSON.stringify).join('\n') + '\n');
      assert.equal((await collectSessionResultAfterBoundary(path, boundary)).text, 'fresh result');
    } finally { await rm(dir, { recursive: true, force: true }); }
  });

  it('rejects a captured anchor moved to a different physical line', async () => {
    const dir = await mkdtemp(join(tmpdir(), 'pi-anchor-line-')); const path = join(dir, 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa.jsonl');
    const header = { type: 'session', id: 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa' };
    try {
      await writeFile(path, [header, { type: 'message', id: 'anchor', message: { role: 'user', content: 'old' } }, { type: 'model_change', id: 'model', model: 'one' }].map(JSON.stringify).join('\n') + '\n');
      const boundary = await captureSessionBoundary({ kind: 'path', value: path });
      assert.deepEqual(boundary, { sessionId: header.id, anchorId: 'model', anchorLine: 3, lineCount: 3 });
      await writeFile(path, [header, { type: 'model_change', id: 'model', model: 'one' }, { type: 'message', id: 'anchor', message: { role: 'user', content: 'old' } }, { type: 'message', id: 'new', message: { role: 'user', content: 'new' } }].map(JSON.stringify).join('\n') + '\n');
      await assert.rejects(collectSessionResultAfterBoundary(path, boundary), /anchor.*moved/);
    } finally { await rm(dir, { recursive: true, force: true }); }
  });

  it('fails closed for malformed recognized post-boundary messages before a later terminal answer', async () => {
    const dir = await mkdtemp(join(tmpdir(), 'pi-message-shape-')); const path = join(dir, 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa.jsonl');
    const header = { type: 'session', id: 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa' };
    const boundary = { sessionId: header.id, anchorId: header.id, anchorLine: 1, lineCount: 1 };
    const answer = { type: 'message', id: 'answer', message: { role: 'assistant', content: [{ type: 'text', text: 'must not escape' }], stopReason: 'stop', usage } };
    const malformed = [
      { type: 'message', id: 'bad', message: null },
      { type: 'message', id: 'bad', message: {} },
      { type: 'message', id: 'bad', message: { role: 'unknown' } },
      { type: 'message', id: 'bad', message: { role: 'user', content: 7 } },
      { type: 'message', id: 'bad', message: { role: 'assistant', content: 'bad', stopReason: 'stop' } },
      { type: 'message', id: 'bad', message: { role: 'assistant', content: [], stopReason: 'wat' } },
      { type: 'message', id: 'bad', message: { role: 'toolResult', toolCallId: '', content: [], isError: false } },
      { type: 'message', id: 'bad', message: { role: 'toolResult', toolCallId: 'call', content: [], isError: 'false' } },
    ];
    try {
      for (const bad of malformed) {
        await writeFile(path, [header, { type: 'message', id: 'user', message: { role: 'user', content: 'new' } }, bad, answer].map(JSON.stringify).join('\n') + '\n');
        await assert.rejects(collectSessionResultAfterBoundary(path, boundary), /post-boundary message/);
      }
    } finally { await rm(dir, { recursive: true, force: true }); }
  });

  it('resolves an ID-backed session before each capture so its second boundary is exact', async () => {
    const root = await mkdtemp(join(tmpdir(), 'pi-id-boundary-')); const id = 'eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee'; const path = join(root, 'session.jsonl');
    const session = { kind: 'id', value: id };
    try {
      assert.deepEqual(await captureResolvedSessionBoundary(session, root), { sessionId: id, anchorId: null, anchorLine: null, lineCount: 0 });
      await writeFile(path, [{ type: 'session', id }, { type: 'message', id: 'u-one', message: { role: 'user', content: 'one' } }, { type: 'message', id: 'a-one', message: { role: 'assistant', content: [{ type: 'text', text: 'one' }], stopReason: 'stop', usage } }].map(JSON.stringify).join('\n') + '\n');
      const boundary = await captureResolvedSessionBoundary(session, root);
      assert.deepEqual(boundary, { sessionId: id, anchorId: 'a-one', anchorLine: 3, lineCount: 3 });
      await appendFile(path, [{ type: 'message', id: 'u-two', message: { role: 'user', content: 'two' } }, { type: 'message', id: 'a-two', message: { role: 'assistant', content: [{ type: 'text', text: 'two' }], stopReason: 'stop', usage } }].map(JSON.stringify).join('\n') + '\n');
      assert.equal((await collectSessionResultAfterBoundary(path, boundary)).text, 'two');
    } finally { await rm(root, { recursive: true, force: true }); }
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

  it('bounds ID discovery, reads only capped first lines, and rejects ambiguity', async () => {
    const root = await mkdtemp(join(tmpdir(), 'pi-discovery-bounds-'));
    const id = '22222222-2222-2222-2222-222222222222';
    try {
      const deep = join(root, 'one', 'two'); await mkdir(deep, { recursive: true });
      await writeFile(join(deep, 'target.jsonl'), `${JSON.stringify({ type: 'session', id })}\n`);
      await assert.rejects(resolvePiSessionReference({ kind: 'id', value: id }, root, { maxDepth: 1 }), /depth bound/);
      await assert.rejects(resolvePiSessionReference({ kind: 'id', value: id }, root, { maxDirectories: 1 }), /directory bound/);
      await assert.rejects(resolvePiSessionReference({ kind: 'id', value: id }, root, { maxEntries: 1 }), /entry bound/);
      const extraCandidate = join(root, 'extra.jsonl'); await writeFile(extraCandidate, `${JSON.stringify({ type: 'not-session', id: 'other' })}\n`);
      await assert.rejects(resolvePiSessionReference({ kind: 'id', value: id }, root, { maxCandidates: 1 }), /candidate bound/);
      await rm(extraCandidate);

      const overlong = join(root, 'overlong.jsonl'); await writeFile(overlong, `${' '.repeat(40)}${JSON.stringify({ type: 'session', id: 'other' })}\n`);
      await assert.rejects(resolvePiSessionReference({ kind: 'id', value: id }, root, { maxFirstLineBytes: 16 }), /first line.*bound/);
      await rm(overlong);

      const unrelated = join(root, 'large.jsonl'); await writeFile(unrelated, `${JSON.stringify({ type: 'not-session', id: 'other' })}\n${'x'.repeat(1024 * 1024)}`);
      assert.equal(await resolvePiSessionReference({ kind: 'id', value: id }, root), join(deep, 'target.jsonl'));

      await writeFile(join(root, 'duplicate.jsonl'), `${JSON.stringify({ type: 'session', id })}\n`);
      await assert.rejects(resolvePiSessionReference({ kind: 'id', value: id }, root), /ambiguous/);
    } finally { await rm(root, { recursive: true, force: true }); }
  });

  it('does not follow symlinks during ID discovery', async () => {
    const root = await mkdtemp(join(tmpdir(), 'pi-discovery-symlink-'));
    const outside = await mkdtemp(join(tmpdir(), 'pi-discovery-outside-'));
    const id = '66666666-6666-6666-6666-666666666666';
    await writeFile(join(outside, 'session.jsonl'), `${JSON.stringify({ type: 'session', id })}\n`);
    await symlink(outside, join(root, 'linked'));
    try { await assert.rejects(resolvePiSessionReference({ kind: 'id', value: id }, root), /not found/); }
    finally { await rm(root, { recursive: true, force: true }); await rm(outside, { recursive: true, force: true }); }
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
