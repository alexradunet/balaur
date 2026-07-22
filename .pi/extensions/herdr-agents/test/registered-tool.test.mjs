import { after, before, describe, it } from 'node:test';
import assert from 'node:assert/strict';
import net from 'node:net';
import { existsSync } from 'node:fs';
import { appendFile, mkdtemp, mkdir, readFile, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { dirname, resolve } from 'node:path';
import { pathToFileURL } from 'node:url';
import { createHandle, addHandle, createHandleStore, deserializeStore, serializeStore } from '../handle-store.js';

const extensionDir = resolve('.pi/extensions/herdr-agents');
const sessionPath = resolve('.pi/extensions/herdr-agents/test/fixtures/session.jsonl');
const originalEnv = { HERDR_ENV: process.env.HERDR_ENV, HERDR_SOCKET_PATH: process.env.HERDR_SOCKET_PATH, HERDR_PANE_ID: process.env.HERDR_PANE_ID, BALAUR_WORKER: process.env.BALAUR_WORKER };
let stubDir;

function deferred() {
  let resolvePromise;
  let rejectPromise;
  const promise = new Promise((resolve, reject) => { resolvePromise = resolve; rejectPromise = reject; });
  return { promise, resolve: resolvePromise, reject: rejectPromise };
}

class FakeHerdr {
  constructor(options = {}) { this.options = options; this.requests = []; this.name = ''; this.getCount = 0; this.status = options.agentStatus ?? 'idle'; this.removed = false; this.path = `${tmpdir()}/herdr-registered-${process.pid}-${Date.now()}-${Math.random()}.sock`; }
  agent(name = this.name, session = this.options.session ?? { kind: 'id', value: '11111111-1111-1111-1111-111111111111' }) { return { name, pane_id: 'w1:p2', workspace_id: 'w1', tab_id: 'w1:t1', terminal_id: 'term-2', agent_status: this.status, interactive_ready: true, launch_pending: false, agent_session: { source: 'herdr:pi', agent: 'pi', ...session } }; }
  response(request) {
    const custom = this.options.handlers?.[request.method]; if (custom) return custom(request, this);
    switch (request.method) {
      case 'ping': return { type: 'pong', version: '0.7.5', protocol: this.options.protocol ?? 17, capabilities: { live_handoff: true, detached_server_daemon: true } };
      case 'pane.split': return { type: 'pane_info', pane: { pane_id: 'w1:p2', workspace_id: 'w1', tab_id: 'w1:t1', terminal_id: 'term-2' } };
      case 'agent.start': this.name = request.params.name; this.removed = false; return { type: 'agent_started', agent: this.agent() };
      case 'agent.get': this.getCount++; return this.removed ? { error: { code: 'agent_not_found', message: 'agent was removed' } } : { type: 'agent_info', agent: this.agent(request.params.target) };
      case 'agent.list': return { type: 'agent_list', agents: [{ pane_id: 'w1:p1', terminal_id: 'term-1', agent_status: 'idle' }, ...(this.removed ? [] : [this.agent()])] };
      case 'agent.wait': return request.params.until.includes(this.status) ? { type: 'agent_info', agent: this.agent(request.params.target) } : { error: { code: 'timeout', message: `timed out waiting for ${request.params.until.join(',')}` } };
      case 'agent.prompt':
        if (this.status === 'working') return { error: { code: 'agent_busy', message: 'agent is already working' } };
        this.status = 'working';
        return { type: 'agent_prompted', agent: this.agent(request.params.target) };
      case 'agent.read': return { type: 'pane_read', read: { text: 'diagnostic', truncated: false } };
      case 'pane.report_metadata': return { type: 'ok' };
      case 'pane.close': this.removed = true; return { type: 'ok' };
      default: throw new Error(`unhandled ${request.method}`);
    }
  }
  async start() { this.server = net.createServer((socket) => { let buffer = ''; socket.on('data', (chunk) => { buffer += chunk; const lines = buffer.split('\n'); buffer = lines.pop() || ''; for (const line of lines) { if (!line) continue; const request = JSON.parse(line); this.requests.push(request); void Promise.resolve().then(() => this.response(request)).then((result) => { if (result === null) socket.end(); else if (result?.error) socket.write(JSON.stringify({ id: request.id, error: result.error }) + '\n'); else socket.write(JSON.stringify({ id: request.id, result }) + '\n'); }, (error) => { socket.write(JSON.stringify({ id: request.id, error: { code: 'fake_error', message: error.message } }) + '\n'); }); } }); }); await new Promise((ok) => this.server.listen(this.path, ok)); }
  async stop() { await new Promise((ok) => this.server.close(ok)); }
}

async function writeStub(packageName, source) { const dir = resolve(stubDir, ...packageName.split('/')); await mkdir(dir, { recursive: true }); await writeFile(resolve(dir, 'package.json'), JSON.stringify({ type: 'module', exports: './index.js' })); await writeFile(resolve(dir, 'index.js'), source); }
before(async () => {
  stubDir = resolve(extensionDir, 'node_modules');
  await writeStub('typebox', `export const Type={Object:(x)=>x,String:(x)=>x,Optional:(x)=>x,Integer:(x)=>x};`);
  await writeStub('@earendil-works/pi-ai', `export const StringEnum=(values)=>({enum:values});`);
  await writeStub('@earendil-works/pi-coding-agent', `export const DEFAULT_MAX_BYTES=51200,DEFAULT_MAX_LINES=2000; export const formatSize=(n)=>String(n); export const truncateHead=(text)=>({content:text,truncated:false,outputLines:text.split('\\n').length,totalLines:text.split('\\n').length,outputBytes:Buffer.byteLength(text),totalBytes:Buffer.byteLength(text)});`);
  await writeStub('@earendil-works/pi-tui', `export class Text { constructor(text){ this.text=text; } }`);
});
after(async () => { await rm(stubDir, { recursive: true, force: true }); for (const [key, value] of Object.entries(originalEnv)) value === undefined ? delete process.env[key] : process.env[key] = value; });

async function loadRegisteredTool(server, branch = []) {
  process.env.HERDR_ENV = '1'; process.env.HERDR_SOCKET_PATH = server.path; process.env.HERDR_PANE_ID = 'w1:p1'; delete process.env.BALAUR_WORKER;
  const registered = []; const events = new Map();
  const pi = { appendEntry: (customType, data) => branch.push({ type: 'custom', customType, data }), registerTool: (tool) => registered.push(tool), on: (name, handler) => events.set(name, handler) };
  const mod = await import(`${pathToFileURL(resolve(extensionDir, 'index.ts')).href}?registered=${Date.now()}-${Math.random()}`);
  mod.default(pi);
  const ctx = { cwd: resolve('.'), hasUI: true, ui: { confirm: async () => true }, sessionManager: { getBranch: () => branch } };
  await events.get('session_start')?.({}, ctx);
  assert.equal(registered.length, 1, 'extension registers exactly herdr_agent in a lead Herdr pane');
  return { tool: registered[0], ctx, branch };
}

async function startWorker(tool, ctx) { const result = await tool.execute('start', { action: 'start', role: 'implementer-openai' }, undefined, undefined, ctx); return result.details.handle; }
function latestPersistedHandle(branch, handleId) {
  const entry = [...branch].reverse().find((candidate) => candidate.type === 'custom' && candidate.customType === 'balaur-herdr-agent-store');
  return deserializeStore(entry.data.store).handles[handleId];
}

describe('registered herdr_agent extension acceptance matrix', { concurrency: false }, () => {
  it('invokes registered start and every handle action against fake Herdr', async () => {
    const dir = await mkdtemp(resolve(tmpdir(), 'registered-session-')); const currentSession = resolve(dir, '11111111-1111-1111-1111-111111111111.jsonl'); await writeFile(currentSession, await readFile(sessionPath));
    const server = new FakeHerdr({ session: { kind: 'path', value: currentSession } }); await server.start();
    try {
      const { tool, ctx } = await loadRegisteredTool(server); const handle = await startWorker(tool, ctx);
      const started = (await tool.execute('status', { action: 'status', handle }, undefined, undefined, ctx)).details.handles[0];
      assert.equal(started.agentName, server.name, 'start persists the generated Herdr agent name');
      await tool.execute('wait', { action: 'wait', handle, timeout_ms: 3000 }, undefined, undefined, ctx);
      await tool.execute('read', { action: 'read', handle, lines: 10 }, undefined, undefined, ctx);
      await tool.execute('prompt', { action: 'prompt', handle, prompt: 'continue' }, undefined, undefined, ctx);
      await appendFile(currentSession, `${JSON.stringify({ type: 'message', id: 'u-bridge', message: { role: 'user', content: 'continue' } })}\n${JSON.stringify({ type: 'message', id: 'a-bridge', message: { role: 'assistant', content: [{ type: 'text', text: 'Task completed successfully.' }], stopReason: 'stop', usage: {} } })}\n`);
      const collected = await tool.execute('collect', { action: 'collect', handle }, undefined, undefined, ctx);
      assert.match(collected.content[0].text, /Task completed successfully/);
      await assert.rejects(tool.execute('close', { action: 'close', handle }, undefined, undefined, ctx), /automated close is disabled.*protocol 17/i);
      for (const method of ['pane.split', 'agent.start', 'agent.wait', 'agent.read', 'agent.prompt']) assert.ok(server.requests.some((request) => request.method === method), `${method} was invoked`);
      assert.equal(server.requests.some((request) => request.method === 'pane.close'), false);
    } finally { await server.stop(); await rm(dir, { recursive: true, force: true }); }
  });

  it('rejects a concurrent same-handle prompt before a second Herdr request', async () => {
    const gate = deferred();
    const firstPrompted = deferred();
    let promptCount = 0;
    const server = new FakeHerdr({ handlers: { 'agent.prompt': async (request, fake) => {
      promptCount++;
      firstPrompted.resolve();
      await gate.promise;
      return { type: 'agent_prompted', agent: fake.agent(request.params.target) };
    } } });
    await server.start();
    try {
      const { tool, ctx } = await loadRegisteredTool(server);
      const handle = await startWorker(tool, ctx);
      const first = tool.execute('prompt-1', { action: 'prompt', handle, prompt: 'first' }, undefined, undefined, ctx);
      await firstPrompted.promise;
      await assert.rejects(tool.execute('prompt-2', { action: 'prompt', handle, prompt: 'second' }, undefined, undefined, ctx), new RegExp(`${handle}.*prompt`));
      assert.equal(promptCount, 1);
      gate.resolve();
      await first;
    } finally { gate.resolve(); await server.stop(); }
  });

  it('always disables close without touching the handle, UI, or Herdr', async () => {
    for (const hasUI of [true, false]) {
      const server = new FakeHerdr(); await server.start();
      try {
        const { tool, ctx, branch } = await loadRegisteredTool(server); const handle = await startWorker(tool, ctx);
        const before = latestPersistedHandle(branch, handle);
        const requestsBefore = server.requests.length;
        let confirmations = 0; ctx.hasUI = hasUI; ctx.ui.confirm = async () => { confirmations++; return true; };
        await assert.rejects(tool.execute('close', { action: 'close', handle }, undefined, undefined, ctx), new RegExp(`automated close is disabled.*${handle}.*w1:p2`, 'i'));
        assert.equal(confirmations, 0);
        assert.equal(server.requests.length, requestsBefore);
        assert.deepEqual(latestPersistedHandle(branch, handle), before);
      } finally { await server.stop(); }
    }
  });

  it('allows manually closed panes to reconcile as missing after disabled close guidance', async () => {
    const server = new FakeHerdr(); await server.start();
    try {
      const { tool, ctx } = await loadRegisteredTool(server); const handle = await startWorker(tool, ctx);
      await assert.rejects(tool.execute('close', { action: 'close', handle }, undefined, undefined, ctx), /automated close is disabled/i);
      server.removed = true;
      const status = await tool.execute('status', { action: 'status', handle }, undefined, undefined, ctx);
      assert.equal(status.details.handles.find((candidate) => candidate.handleId === handle).status, 'missing');
    } finally { await server.stop(); }
  });

  it('rejects stale operational status from fresh pinned prompt admission before boundary persistence or submission', async () => {
    const cases = [
      ['idle', 'working'],
      ['blocked', 'done'],
      ['blocked', 'unknown'],
    ];
    for (const [persisted, live] of cases) {
      const server = new FakeHerdr(); await server.start();
      try {
        const { tool, ctx, branch } = await loadRegisteredTool(server); const handle = await startWorker(tool, ctx);
        server.status = persisted;
        await tool.execute('status', { action: 'status', handle }, undefined, undefined, ctx);
        const before = latestPersistedHandle(branch, handle);
        server.status = live;
        await assert.rejects(tool.execute('prompt', { action: 'prompt', handle, prompt: 'must not submit' }, undefined, undefined, ctx), /must be idle or blocked/);
        const after = latestPersistedHandle(branch, handle);
        assert.equal(after.status, live);
        assert.equal(after.promptBoundary, before.promptBoundary);
        assert.equal(server.requests.some((request) => request.method === 'agent.prompt'), false);
      } finally { await server.stop(); }
    }
  });

  it('admits a fresh idle worker despite persisted working status and persists fresh-pin replacement', async () => {
    const server = new FakeHerdr(); await server.start();
    try {
      const loaded = await loadRegisteredTool(server); const handle = await startWorker(loaded.tool, loaded.ctx);
      server.status = 'working';
      await loaded.tool.execute('status', { action: 'status', handle }, undefined, undefined, loaded.ctx);
      server.status = 'idle';
      await loaded.tool.execute('prompt', { action: 'prompt', handle, prompt: 'fresh is authoritative' }, undefined, undefined, loaded.ctx);
      assert.equal(server.requests.filter((request) => request.method === 'agent.prompt').length, 1);

      const second = await startWorker(loaded.tool, loaded.ctx);
      (server.options.handlers ||= {})['agent.get'] = (request, fake) => ({ type: 'agent_info', agent: fake.agent(request.params.target, { kind: 'id', value: '99999999-9999-9999-9999-999999999999' }) });
      await assert.rejects(loaded.tool.execute('prompt', { action: 'prompt', handle: second, prompt: 'mismatch' }, undefined, undefined, loaded.ctx), /session identity was replaced/);
      assert.equal(latestPersistedHandle(loaded.branch, second).status, 'replaced');
    } finally { await server.stop(); }
  });

  it('rejects status, wait, read, and collect while prompt owns the handle lease', async () => {
    const gate = deferred();
    const entered = deferred();
    const server = new FakeHerdr({ handlers: { 'agent.prompt': async (request, fake) => { entered.resolve(); await gate.promise; return { type: 'agent_prompted', agent: fake.agent(request.params.target) }; } } });
    await server.start();
    try {
      const { tool, ctx } = await loadRegisteredTool(server);
      const handle = await startWorker(tool, ctx);
      const prompting = tool.execute('prompt', { action: 'prompt', handle, prompt: 'hold' }, undefined, undefined, ctx);
      await entered.promise;
      const requestsBefore = server.requests.length;
      for (const action of ['status', 'wait', 'read', 'collect']) {
        await assert.rejects(tool.execute(action, { action, handle, timeout_ms: 3000 }, undefined, undefined, ctx), /busy with prompt/);
      }
      assert.equal(server.requests.length, requestsBefore, 'rejected actions make no Herdr request');
      gate.resolve();
      await prompting;
    } finally { gate.resolve(); await server.stop(); }
  });

  it('allows unrelated handles to prompt concurrently', async () => {
    const gate = deferred();
    const bothEntered = deferred();
    let entered = 0;
    const server = new FakeHerdr({ handlers: { 'agent.prompt': async (request, fake) => { if (++entered === 2) bothEntered.resolve(); await gate.promise; return { type: 'agent_prompted', agent: fake.agent(request.params.target) }; } } });
    await server.start();
    try {
      const { tool, ctx } = await loadRegisteredTool(server);
      const firstHandle = await startWorker(tool, ctx);
      const secondHandle = await startWorker(tool, ctx);
      const first = tool.execute('first', { action: 'prompt', handle: firstHandle, prompt: 'first' }, undefined, undefined, ctx);
      const second = tool.execute('second', { action: 'prompt', handle: secondHandle, prompt: 'second' }, undefined, undefined, ctx);
      await bothEntered.promise;
      assert.equal(entered, 2);
      gate.resolve();
      await Promise.all([first, second]);
    } finally { gate.resolve(); await server.stop(); }
  });

  it('releases a handle lease after errors, aborts, remote timeout, and disabled close guidance', async () => {
    let readFails = true;
    const server = new FakeHerdr({ handlers: { 'agent.read': () => {
      if (readFails) { readFails = false; return { error: { code: 'read_failed', message: 'read failed' } }; }
      return { type: 'pane_read', read: { text: 'diagnostic', truncated: false } };
    } } });
    await server.start();
    try {
      const { tool, ctx } = await loadRegisteredTool(server);
      const handle = await startWorker(tool, ctx);
      await assert.rejects(tool.execute('read-error', { action: 'read', handle }, undefined, undefined, ctx), /read failed/);
      await tool.execute('read-after-error', { action: 'read', handle }, undefined, undefined, ctx);

      const controller = new AbortController();
      controller.abort();
      await assert.rejects(tool.execute('status-abort', { action: 'status', handle }, controller.signal, undefined, ctx), /aborted/);
      await tool.execute('status-after-abort', { action: 'status', handle }, undefined, undefined, ctx);

      server.status = 'working';
      const timedOut = await tool.execute('wait-timeout', { action: 'wait', handle, timeout_ms: 3000 }, undefined, undefined, ctx);
      assert.match(timedOut.content[0].text, /timed out/);
      server.status = 'idle';
      await tool.execute('status-after-timeout', { action: 'status', handle }, undefined, undefined, ctx);

      await assert.rejects(tool.execute('close-disabled', { action: 'close', handle }, undefined, undefined, ctx), /automated close is disabled/i);
      await tool.execute('prompt-after-disabled-close', { action: 'prompt', handle, prompt: 'still open' }, undefined, undefined, ctx);
    } finally { await server.stop(); }
  });

  it('fails closed when no Herdr environment is present', async () => {
    delete process.env.HERDR_ENV; delete process.env.HERDR_SOCKET_PATH; delete process.env.HERDR_PANE_ID;
    const tools = []; const mod = await import(`${pathToFileURL(resolve(extensionDir, 'index.ts')).href}?unavailable=${Date.now()}`); mod.default({ registerTool: (tool) => tools.push(tool), on: () => {} });
    assert.equal(tools.length, 0);
  });

  it('keeps a ready handle when auxiliary metadata reporting fails and persists custom snapshots', async () => {
    const server = new FakeHerdr({ handlers: { 'pane.report_metadata': () => ({ error: { code: 'metadata_failed', message: 'metadata unavailable' } }) } }); await server.start();
    try {
      const loaded = await loadRegisteredTool(server); const handle = await startWorker(loaded.tool, loaded.ctx);
      const result = await loaded.tool.execute('list', { action: 'list' }, undefined, undefined, loaded.ctx);
      assert.match(result.content[0].text, new RegExp(handle));
      assert.ok(loaded.branch.some((entry) => entry.type === 'custom' && entry.customType === 'balaur-herdr-agent-store'));
      assert.equal(server.requests.some((request) => request.method === 'pane.close'), false);
    } finally { await server.stop(); }
  });

  it('retains a provisional error handle when launch identity polling fails', async () => {
    const server = new FakeHerdr({ handlers: { 'agent.get': () => ({ error: { code: 'not_ready', message: 'not ready' } }) } }); await server.start();
    try {
      const { tool, ctx } = await loadRegisteredTool(server); const controller = new AbortController(); setTimeout(() => controller.abort(), 25); const result = await tool.execute('start', { action: 'start', role: 'implementer-openai' }, controller.signal, undefined, ctx);
      assert.match(result.content[0].text, /launch completion is uncertain/);
      assert.equal(result.details.handles[0].status, 'error');
      assert.equal(server.requests.some((request) => request.method === 'pane.close'), false);
    } finally { await server.stop(); }
  });

  it('hydrates a recoverable provisional handle exactly once when status finds its delayed session', async () => {
    const server = new FakeHerdr({ handlers: { 'agent.get': (request, fake) => { fake.getCount++; const agent = fake.agent(request.params.target); if (fake.getCount <= 2) delete agent.agent_session; return { type: 'agent_info', agent }; } } }); await server.start();
    try {
      const { tool, ctx } = await loadRegisteredTool(server); const controller = new AbortController(); setTimeout(() => controller.abort(), 25);
      const started = await tool.execute('start', { action: 'start', role: 'implementer-openai' }, controller.signal, undefined, ctx);
      const handle = started.details.handle;
      const recovered = await tool.execute('status', { action: 'status', handle }, undefined, undefined, ctx);
      assert.equal(recovered.details.handles[0].status, 'idle');
      assert.equal(recovered.details.handles[0].sessionKind, 'id');
      assert.equal(recovered.details.handles[0].error, undefined);
    } finally { await server.stop(); }
  });

  it('never rebinds a recoverable provisional handle to a mismatched terminal', async () => {
    const server = new FakeHerdr({ handlers: { 'agent.get': (request, fake) => { fake.getCount++; const agent = fake.agent(request.params.target); if (fake.getCount <= 2) delete agent.agent_session; else agent.terminal_id = 'other-terminal'; return { type: 'agent_info', agent }; } } }); await server.start();
    try {
      const { tool, ctx } = await loadRegisteredTool(server); const controller = new AbortController(); setTimeout(() => controller.abort(), 25);
      const started = await tool.execute('start', { action: 'start', role: 'implementer-openai' }, controller.signal, undefined, ctx);
      await assert.rejects(tool.execute('status', { action: 'status', handle: started.details.handle }, undefined, undefined, ctx), /occupant was replaced/);
      const recovered = await tool.execute('list', { action: 'list' }, undefined, undefined, ctx);
      assert.equal(recovered.details.handles[0].status, 'replaced');
      assert.equal(recovered.details.handles[0].sessionKind, undefined);
    } finally { await server.stop(); }
  });

  it('rejects unsupported executor isolation before splitting a pane', async () => {
    const server = new FakeHerdr(); await server.start();
    try {
      const { tool, ctx } = await loadRegisteredTool(server);
      await assert.rejects(tool.execute('start', { action: 'start', role: 'executor' }, undefined, undefined, ctx), /executor\.md: unsupported role key 'isolation'/);
      assert.equal(server.requests.some((request) => request.method === 'pane.split'), false);
    } finally { await server.stop(); }
  });

  it('rejects a numeric protocol mismatch through registered start', async () => {
    const server = new FakeHerdr({ protocol: 16 }); await server.start();
    try { const { tool, ctx } = await loadRegisteredTool(server); await assert.rejects(startWorker(tool, ctx), /protocol\/capability mismatch \(protocol 16\)/); } finally { await server.stop(); }
  });

  it('persists wait-returned identity replacement before propagating it', async () => {
    const server = new FakeHerdr({ handlers: { 'agent.wait': (request, fake) => ({ type: 'agent_info', agent: fake.agent(request.params.target, { kind: 'id', value: '99999999-9999-9999-9999-999999999999' }) }) } });
    await server.start();
    try {
      const loaded = await loadRegisteredTool(server);
      const handle = await startWorker(loaded.tool, loaded.ctx);
      await assert.rejects(loaded.tool.execute('wait', { action: 'wait', handle, timeout_ms: 3000 }, undefined, undefined, loaded.ctx), /session identity was replaced/);
      assert.equal(latestPersistedHandle(loaded.branch, handle).status, 'replaced');
      delete server.options.handlers['agent.wait'];
      const status = await loaded.tool.execute('status-after-replacement', { action: 'status', handle }, undefined, undefined, loaded.ctx);
      assert.equal(status.details.handles.find((candidate) => candidate.handleId === handle).status, 'replaced');
    } finally { await server.stop(); }
  });

  it('persists prompt-returned replacement as replaced with an uncertain retained boundary', async () => {
    const server = new FakeHerdr({ handlers: { 'agent.prompt': (request, fake) => ({ type: 'agent_prompted', agent: fake.agent(request.params.target, { kind: 'id', value: '99999999-9999-9999-9999-999999999999' }) }) } });
    await server.start();
    try {
      const loaded = await loadRegisteredTool(server);
      const handle = await startWorker(loaded.tool, loaded.ctx);
      await assert.rejects(loaded.tool.execute('prompt', { action: 'prompt', handle, prompt: 'mismatch' }, undefined, undefined, loaded.ctx), /session identity was replaced/);
      const persisted = latestPersistedHandle(loaded.branch, handle);
      assert.equal(persisted.status, 'replaced');
      assert.equal(persisted.promptPhase, 'uncertain');
      assert.equal(persisted.promptBoundary.sessionId, '11111111-1111-1111-1111-111111111111');
    } finally { await server.stop(); }
  });

  it('classifies agent_not_found only after successful inventory reconciliation', async () => {
    for (const [inventory, expected] of [
      [[{ name: 'other', pane_id: 'w1:p2', terminal_id: 'term-other', agent_status: 'idle' }], 'replaced'],
      [[{ name: 'other', pane_id: 'other', terminal_id: 'term-2', agent_status: 'idle' }], 'replaced'],
      [[{ name: 'other', pane_id: 'other', terminal_id: 'other', agent_status: 'idle', agent_session: { kind: 'id', value: '11111111-1111-1111-1111-111111111111' } }], 'replaced'],
      [null, 'idle'],
      [[], 'missing'],
    ]) {
      const server = new FakeHerdr({ handlers: {
        'agent.get': () => ({ error: { code: 'agent_not_found', message: 'gone' } }),
        'agent.list': (_request, fake) => ({ type: 'agent_list', agents: inventory ?? [fake.agent()] }),
      } });
      await server.start();
      try {
        const loaded = await loadRegisteredTool(server);
        // Start needs normal get responses before status exercises not-found.
        delete server.options.handlers['agent.get'];
        const handle = await startWorker(loaded.tool, loaded.ctx);
        server.options.handlers['agent.get'] = () => ({ error: { code: 'agent_not_found', message: 'gone' } });
        const result = await loaded.tool.execute('status', { action: 'status', handle }, undefined, undefined, loaded.ctx);
        assert.equal(result.details.handles.find((candidate) => candidate.handleId === handle).status, expected);
        assert.equal(latestPersistedHandle(loaded.branch, handle).status, expected);
      } finally { await server.stop(); }
    }
  });

  it('preserves prior status and throws on transient, malformed, invalid, oversized, aborted, and inventory failures', async () => {
    const cases = [
      ['transport', () => null],
      ['malformed', () => ({ type: 'agent_info', agent: null })],
      ['invalid status', (request, fake) => ({ type: 'agent_info', agent: { ...fake.agent(request.params.target), agent_status: 'paused' } })],
      ['oversized status', (request, fake) => ({ type: 'agent_info', agent: { ...fake.agent(request.params.target), agent_status: 'x'.repeat(10000) } })],
    ];
    for (const [label, handler] of cases) {
      const server = new FakeHerdr(); await server.start();
      try {
        const loaded = await loadRegisteredTool(server); const handle = await startWorker(loaded.tool, loaded.ctx);
        server.options.handlers = { 'agent.get': handler };
        await assert.rejects(loaded.tool.execute(label, { action: 'status', handle }, undefined, undefined, loaded.ctx));
        assert.equal(latestPersistedHandle(loaded.branch, handle).status, 'idle', label);
      } finally { await server.stop(); }
    }

    const abortServer = new FakeHerdr(); await abortServer.start();
    try {
      const loaded = await loadRegisteredTool(abortServer); const handle = await startWorker(loaded.tool, loaded.ctx);
      const controller = new AbortController(); controller.abort();
      await assert.rejects(loaded.tool.execute('abort', { action: 'status', handle }, controller.signal, undefined, loaded.ctx), /aborted/);
      assert.equal(latestPersistedHandle(loaded.branch, handle).status, 'idle');
    } finally { await abortServer.stop(); }

    const inventoryServer = new FakeHerdr(); await inventoryServer.start();
    try {
      const loaded = await loadRegisteredTool(inventoryServer); const handle = await startWorker(loaded.tool, loaded.ctx);
      inventoryServer.options.handlers = {
        'agent.get': () => ({ error: { code: 'agent_not_found', message: 'gone' } }),
        'agent.list': () => null,
      };
      await assert.rejects(loaded.tool.execute('inventory', { action: 'status', handle }, undefined, undefined, loaded.ctx));
      assert.equal(latestPersistedHandle(loaded.branch, handle).status, 'idle');
    } finally { await inventoryServer.stop(); }
  });

  it('reports an event-driven timeout without killing the worker', async () => {
    const server = new FakeHerdr({ handlers: { 'agent.wait': () => ({ error: { code: 'timeout', message: 'timed out waiting for idle' } }) } }); await server.start();
    try { const { tool, ctx } = await loadRegisteredTool(server); const handle = await startWorker(tool, ctx); const result = await tool.execute('wait', { action: 'wait', handle, timeout_ms: 3000 }, undefined, undefined, ctx); assert.match(result.content[0].text, /timed out; it was not killed/); assert.equal(server.requests.some((request) => request.method.includes('stop') || request.method.includes('kill')), false); } finally { await server.stop(); }
  });

  it('preserves blocked status and sends the protocol activity gate without stopping a worker', async () => {
    const server = new FakeHerdr({ agentStatus: 'blocked' }); await server.start();
    try {
      const { tool, ctx } = await loadRegisteredTool(server); const handle = await startWorker(tool, ctx);
      const status = await tool.execute('status', { action: 'status', handle }, undefined, undefined, ctx);
      assert.match(status.content[0].text, /blocked/);
      const waited = await tool.execute('wait', { action: 'wait', handle, timeout_ms: 3000 }, undefined, undefined, ctx);
      assert.match(waited.content[0].text, /reached blocked/);
      await tool.execute('prompt', { action: 'prompt', handle, prompt: 'continue' }, undefined, undefined, ctx);
      assert.deepEqual(server.requests.find((request) => request.method === 'agent.prompt').params.wait.until, ['working', 'idle', 'blocked', 'done', 'unknown']);
      assert.equal(server.requests.some((request) => /stop|kill|close/.test(request.method)), false);
    } finally { await server.stop(); }
  });

  it('ignores a malformed real session JSONL record and collects its later terminal Pi-v3 message', async () => {
    const dir = await mkdtemp(resolve(tmpdir(), 'bad-pi-')); const bad = resolve(dir, 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb.jsonl');
    await writeFile(bad, JSON.stringify({ type: 'session', id: 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb' }) + '\n' + JSON.stringify({ type: 'message', id: 'old', message: { role: 'assistant', content: [{ type: 'text', text: 'old' }], stopReason: 'stop', usage: {} } }) + '\n{bad json}\n');
    const server = new FakeHerdr({ session: { kind: 'path', value: bad } }); await server.start();
    try { const { tool, ctx } = await loadRegisteredTool(server); const handle = await startWorker(tool, ctx); await tool.execute('prompt', { action: 'prompt', handle, prompt: 'recover' }, undefined, undefined, ctx); await appendFile(bad, JSON.stringify({ type: 'message', id: 'u-new', message: { role: 'user', content: 'recover' } }) + '\n' + JSON.stringify({ type: 'message', id: 'a-new', message: { role: 'assistant', content: [{ type: 'text', text: 'recovered' }], stopReason: 'stop', usage: {} } }) + '\n'); const result = await tool.execute('collect', { action: 'collect', handle }, undefined, undefined, ctx); assert.equal(result.content[0].text, 'recovered'); } finally { await server.stop(); await rm(dir, { recursive: true, force: true }); }
  });


  it('persists terminal-only and paired-session-only conflicts while restoring a handle', async () => {
    for (const inventory of [
      [{ name: 'other', pane_id: 'other', terminal_id: 'term-2', agent_status: 'idle' }],
      [{ name: 'other', pane_id: 'other', terminal_id: 'other', agent_status: 'idle', agent_session: { kind: 'id', value: '11111111-1111-1111-1111-111111111111' } }],
    ]) {
      const handle = { ...createHandle({ paneId: 'w1:p2', terminalId: 'term-2', role: 'implementer-openai', agentName: 'worker', sessionKind: 'id', sessionValue: '11111111-1111-1111-1111-111111111111' }), status: 'idle' };
      const store = addHandle(createHandleStore(), handle);
      const server = new FakeHerdr({ handlers: { 'agent.list': () => ({ type: 'agent_list', agents: inventory }) } }); await server.start();
      try {
        const loaded = await loadRegisteredTool(server, [{ type: 'custom', customType: 'balaur-herdr-agent-store', data: { version: 1, store: serializeStore(store) } }]);
        assert.equal(latestPersistedHandle(loaded.branch, handle.handleId).status, 'replaced');
      } finally { await server.stop(); }
    }
  });

  it('falls back to the latest fully valid snapshot and validates custom-entry versions', async () => {
    const old = { ...createHandle({ paneId: 'w1:p2', terminalId: 'term-2', role: 'implementer-openai', agentName: 'worker', sessionKind: 'path', sessionValue: sessionPath }), status: 'ready' };
    const oldStore = addHandle(createHandleStore(), old);
    const corruptStore = JSON.stringify({ version: 1, handles: { [old.handleId]: { ...old, status: 'paused' } } });
    const server = new FakeHerdr(); await server.start();
    try {
      let loaded = await loadRegisteredTool(server, [
        { type: 'custom', customType: 'balaur-herdr-agent-store', data: { version: 1, store: serializeStore(oldStore) } },
        { type: 'custom', customType: 'balaur-herdr-agent-store', data: { version: 1, store: corruptStore } },
      ]);
      assert.equal((await loaded.tool.execute('list', { action: 'list' }, undefined, undefined, loaded.ctx)).details.handles.length, 1);

      loaded = await loadRegisteredTool(server, [
        { type: 'custom', customType: 'balaur-herdr-agent-store', data: { version: 1, store: serializeStore(oldStore) } },
        { type: 'custom', customType: 'balaur-herdr-agent-store', data: { version: 2, store: serializeStore(createHandleStore()) } },
      ]);
      assert.equal((await loaded.tool.execute('list', { action: 'list' }, undefined, undefined, loaded.ctx)).details.handles.length, 1);
    } finally { await server.stop(); }
  });

  it('restores only the latest full snapshot and reconciles a latest replacement', async () => {
    const old = { ...createHandle({ paneId: 'w1:p2', terminalId: 'term-2', role: 'executor', agentName: 'worker', sessionKind: 'path', sessionValue: sessionPath }), status: 'ready' };
    const oldStore = addHandle(createHandleStore(), old); const emptyStore = createHandleStore();
    const server = new FakeHerdr(); await server.start();
    try {
      let loaded = await loadRegisteredTool(server, [
        { type: 'message', message: { role: 'toolResult', toolName: 'herdr_agent', details: { store: serializeStore(oldStore) } } },
        { type: 'message', message: { role: 'toolResult', toolName: 'herdr_agent', details: { store: serializeStore(emptyStore) } } },
      ]);
      assert.match((await loaded.tool.execute('list', { action: 'list' }, undefined, undefined, loaded.ctx)).content[0].text, /No active workers/);
      loaded = await loadRegisteredTool(server, [{ type: 'message', message: { role: 'toolResult', toolName: 'herdr_agent', details: { store: serializeStore(oldStore) } } }]);
      const list = await loaded.tool.execute('list', { action: 'list' }, undefined, undefined, loaded.ctx);
      assert.equal(list.details.handles[0].status, 'replaced');
    } finally { await server.stop(); }
  });
});
