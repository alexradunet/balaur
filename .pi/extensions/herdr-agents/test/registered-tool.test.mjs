import { after, before, describe, it } from 'node:test';
import assert from 'node:assert/strict';
import net from 'node:net';
import { existsSync } from 'node:fs';
import { appendFile, mkdtemp, mkdir, readFile, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { dirname, resolve } from 'node:path';
import { pathToFileURL } from 'node:url';
import { createHandle, addHandle, createHandleStore, serializeStore } from '../handle-store.js';

const extensionDir = resolve('.pi/extensions/herdr-agents');
const sessionPath = resolve('.pi/extensions/herdr-agents/test/fixtures/session.jsonl');
const originalEnv = { HERDR_ENV: process.env.HERDR_ENV, HERDR_SOCKET_PATH: process.env.HERDR_SOCKET_PATH, HERDR_PANE_ID: process.env.HERDR_PANE_ID, BALAUR_WORKER: process.env.BALAUR_WORKER };
let stubDir;

class FakeHerdr {
  constructor(options = {}) { this.options = options; this.requests = []; this.name = ''; this.getCount = 0; this.path = `${tmpdir()}/herdr-registered-${process.pid}-${Date.now()}-${Math.random()}.sock`; }
  agent(name = this.name, session = this.options.session ?? { kind: 'path', value: sessionPath }) { return { name, pane_id: 'w1:p2', workspace_id: 'w1', tab_id: 'w1:t1', terminal_id: 'term-2', agent_status: this.options.agentStatus ?? 'idle', interactive_ready: true, launch_pending: false, agent_session: { source: 'herdr:pi', agent: 'pi', ...session } }; }
  response(request) {
    const custom = this.options.handlers?.[request.method]; if (custom) return custom(request, this);
    switch (request.method) {
      case 'ping': return { type: 'pong', version: '0.7.5', protocol: this.options.protocol ?? 17, capabilities: { live_handoff: true, detached_server_daemon: true } };
      case 'pane.split': return { type: 'pane_info', pane: { pane_id: 'w1:p2', workspace_id: 'w1', tab_id: 'w1:t1', terminal_id: 'term-2' } };
      case 'agent.start': this.name = request.params.name; return { type: 'agent_started', agent: this.agent() };
      case 'agent.get': this.getCount++; return { type: 'agent_info', agent: this.agent(request.params.target) };
      case 'agent.list': return { type: 'agent_list', agents: [{ pane_id: 'w1:p1', terminal_id: 'term-1', agent_status: 'idle' }, this.agent()] };
      case 'agent.wait': return { type: 'agent_info', agent: this.agent(request.params.target) };
      case 'agent.prompt': return { type: 'agent_prompted', agent: this.agent(request.params.target) };
      case 'agent.read': return { type: 'pane_read', read: { text: 'diagnostic', truncated: false } };
      case 'pane.report_metadata': return { type: 'ok' };
      case 'pane.close': return { type: 'ok' };
      default: throw new Error(`unhandled ${request.method}`);
    }
  }
  async start() { this.server = net.createServer((socket) => { let buffer = ''; socket.on('data', (chunk) => { buffer += chunk; const lines = buffer.split('\n'); buffer = lines.pop() || ''; for (const line of lines) { if (!line) continue; const request = JSON.parse(line); this.requests.push(request); try { const result = this.response(request); if (result?.error) socket.write(JSON.stringify({ id: request.id, error: result.error }) + '\n'); else socket.write(JSON.stringify({ id: request.id, result }) + '\n'); } catch (error) { socket.write(JSON.stringify({ id: request.id, error: { code: 'fake_error', message: error.message } }) + '\n'); } } }); }); await new Promise((ok) => this.server.listen(this.path, ok)); }
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

async function startWorker(tool, ctx) { const result = await tool.execute('start', { action: 'start', role: 'executor' }, undefined, undefined, ctx); return result.details.handle; }

describe('registered herdr_agent extension acceptance matrix', { concurrency: false }, () => {
  it('invokes registered start and every handle action against fake Herdr', async () => {
    const dir = await mkdtemp(resolve(tmpdir(), 'registered-session-')); const currentSession = resolve(dir, 'session.jsonl'); await writeFile(currentSession, await readFile(sessionPath));
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
      await tool.execute('close', { action: 'close', handle }, undefined, undefined, ctx);
      for (const method of ['pane.split', 'agent.start', 'agent.wait', 'agent.read', 'agent.prompt', 'pane.close']) assert.ok(server.requests.some((request) => request.method === method), `${method} was invoked`);
    } finally { await server.stop(); await rm(dir, { recursive: true, force: true }); }
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
      const { tool, ctx } = await loadRegisteredTool(server); const controller = new AbortController(); setTimeout(() => controller.abort(), 25); const result = await tool.execute('start', { action: 'start', role: 'executor' }, controller.signal, undefined, ctx);
      assert.match(result.content[0].text, /launch completion is uncertain/);
      assert.equal(result.details.handles[0].status, 'error');
      assert.equal(server.requests.some((request) => request.method === 'pane.close'), false);
    } finally { await server.stop(); }
  });

  it('rejects a numeric protocol mismatch through registered start', async () => {
    const server = new FakeHerdr({ protocol: 16 }); await server.start();
    try { const { tool, ctx } = await loadRegisteredTool(server); await assert.rejects(startWorker(tool, ctx), /protocol\/capability mismatch \(protocol 16\)/); } finally { await server.stop(); }
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
    const dir = await mkdtemp(resolve(tmpdir(), 'bad-pi-')); const bad = resolve(dir, 'bad.jsonl');
    await writeFile(bad, JSON.stringify({ type: 'session', id: 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb' }) + '\n' + JSON.stringify({ type: 'message', id: 'old', message: { role: 'assistant', content: [{ type: 'text', text: 'old' }], stopReason: 'stop', usage: {} } }) + '\n{bad json}\n');
    const server = new FakeHerdr({ session: { kind: 'path', value: bad } }); await server.start();
    try { const { tool, ctx } = await loadRegisteredTool(server); const handle = await startWorker(tool, ctx); await tool.execute('prompt', { action: 'prompt', handle, prompt: 'recover' }, undefined, undefined, ctx); await appendFile(bad, JSON.stringify({ type: 'message', id: 'u-new', message: { role: 'user', content: 'recover' } }) + '\n' + JSON.stringify({ type: 'message', id: 'a-new', message: { role: 'assistant', content: [{ type: 'text', text: 'recovered' }], stopReason: 'stop', usage: {} } }) + '\n'); const result = await tool.execute('collect', { action: 'collect', handle }, undefined, undefined, ctx); assert.equal(result.content[0].text, 'recovered'); } finally { await server.stop(); await rm(dir, { recursive: true, force: true }); }
  });

  it('blocks close on confirmation denial and on a replacement during confirmation', async () => {
    const server = new FakeHerdr({ handlers: { 'agent.get': (request, fake) => { fake.getCount++; const replacement = fake.getCount > 3; return { type: 'agent_info', agent: fake.agent(request.params.target, replacement ? { kind: 'path', value: '/tmp/replaced.jsonl' } : undefined) }; } } }); await server.start();
    try {
      const { tool, ctx } = await loadRegisteredTool(server); const handle = await startWorker(tool, ctx);
      ctx.ui.confirm = async () => false; await tool.execute('close', { action: 'close', handle }, undefined, undefined, ctx); assert.equal(server.requests.some((request) => request.method === 'pane.close'), false);
      ctx.ui.confirm = async () => true; await assert.rejects(tool.execute('close', { action: 'close', handle }, undefined, undefined, ctx), /session identity was replaced/); assert.equal(server.requests.some((request) => request.method === 'pane.close'), false);
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
