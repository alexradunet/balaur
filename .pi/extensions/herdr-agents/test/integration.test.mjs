import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import net from 'node:net';
import os from 'node:os';
import fs from 'node:fs';
import { resolve } from 'node:path';
import { HerdrClient, HerdrRemoteError, EXPECTED_PROTOCOL } from '../herdr-client.js';
import { assertPinnedAgent, buildWorkerEnv, captureAgentIdentity, createPane, HERDR_AGENT_STATUSES, listAgents, makeAgentLabel, promptAgent, reportPaneMetadata, resolveRoleSkillArgs, startAgent, waitForAgent } from '../pane-manager.js';
import { parseRoleFile } from '../role-parser.js';

const session = { source: 'herdr:pi', agent: 'pi', kind: 'id', value: '33333333-3333-3333-3333-333333333333' };
const agent = (name = 'worker', pane = 'w1:p2') => ({ name, pane_id: pane, workspace_id: 'w1', tab_id: 'w1:t1', terminal_id: 'term-2', agent_status: 'idle', launch_pending: false, interactive_ready: true, agent_session: session });
class FakeHerdr {
  constructor(handler = {}) { this.path = `${os.tmpdir()}/herdr-${process.pid}-${Date.now()}-${Math.random()}.sock`; this.requests = []; this.handler = handler; }
  async start() { this.server = net.createServer((socket) => { let buffer = ''; socket.on('data', (chunk) => { buffer += chunk; const lines = buffer.split('\n'); buffer = lines.pop() || ''; for (const line of lines) if (line) { const request = JSON.parse(line); this.requests.push(request); const result = this.handler[request.method]?.(request) ?? defaults(request); if (result?.error) socket.write(JSON.stringify({ id: result.id ?? request.id, error: result.error }) + '\n'); else if (result !== null) socket.write(JSON.stringify({ id: result?.id ?? request.id, result }) + '\n'); } }); }); await new Promise((resolvePromise) => this.server.listen(this.path, resolvePromise)); }
  async stop() { await new Promise((resolvePromise) => this.server.close(resolvePromise)); }
}
function defaults(request) {
  switch (request.method) {
    case 'ping': return { type: 'pong', version: '0.7.5', protocol: EXPECTED_PROTOCOL, capabilities: { live_handoff: true, detached_server_daemon: true } };
    case 'pane.split': return { type: 'pane_info', pane: { pane_id: 'w1:p2', workspace_id: 'w1', tab_id: 'w1:t1', terminal_id: 'term-2' } };
    case 'agent.start': return { type: 'agent_started', agent: agent(request.params.name), argv: request.params.args };
    case 'agent.get': return { type: 'agent_info', agent: agent(request.params.target) };
    case 'agent.list': return { type: 'agent_list', agents: [agent()] };
    case 'pane.close': return { type: 'ok' };
    case 'pane.report_metadata': return { type: 'ok' };
    default: return { type: 'ok' };
  }
}
async function withServer(handler, run) { const server = new FakeHerdr(handler); await server.start(); try { await run(new HerdrClient({ socketPath: server.path, timeoutMs: 500 }), server); } finally { await server.stop(); } }

describe('protocol-17 socket and lifecycle validation', () => {
  it('rejects a mismatched response id', async () => {
    await withServer({ ping: (req) => ({ id: 'wrong', type: 'pong', version: '0.7.5', protocol: 17, capabilities: { live_handoff: true, detached_server_daemon: true } }) }, async (client) => {
      await assert.rejects(client.ping(), /response ID mismatch/);
    });
  });
  it('classifies only well-formed remote errors as typed bounded remote failures', async () => {
    const oversized = new HerdrRemoteError('x'.repeat(500), '💥'.repeat(5000));
    assert.ok(Buffer.byteLength(oversized.code) <= 128);
    assert.ok(Buffer.byteLength(oversized.message) <= 4000);
    await withServer({ ping: () => ({ error: { code: 'agent_not_found', message: 'missing worker' } }) }, async (client) => {
      await assert.rejects(client.ping(), (error) => error instanceof HerdrRemoteError && error.code === 'agent_not_found' && error.message === 'missing worker');
    });
    await withServer({ ping: () => ({ error: { code: 17, message: 'bad shape' } }) }, async (client) => {
      await assert.rejects(client.ping(), (error) => !(error instanceof HerdrRemoteError) && /error code/.test(error.message));
    });
  });

  it('accepts exactly protocol-17 statuses and rejects invalid agent status shapes', async () => {
    assert.deepEqual([...HERDR_AGENT_STATUSES], ['idle', 'working', 'blocked', 'done', 'unknown']);
    await withServer({ 'agent.get': () => ({ type: 'agent_info', agent: { ...agent(), agent_status: 'paused' } }) }, async (client) => {
      await assert.rejects(assertPinnedAgent(client, { agentName: 'worker', paneId: 'w1:p2', terminalId: 'term-2', sessionKind: 'id', sessionValue: session.value, status: 'idle' }), /agent_status.*invalid/);
    });
  });

  it('requires agent.wait to return one of the requested statuses', async () => {
    await withServer({ 'agent.wait': (request) => ({ type: 'agent_info', agent: { ...agent(request.params.target), agent_status: 'working' } }) }, async (client) => {
      await assert.rejects(waitForAgent(client, { target: 'worker', until: ['idle', 'done'], timeoutMs: 3000 }), /not in requested until set/);
    });
  });

  it('distinguishes remote wait timeout from transport timeout text', async () => {
    const remote = { request: async () => { throw new HerdrRemoteError('timeout', 'settled wait timeout'); } };
    assert.deepEqual(await waitForAgent(remote, { target: 'worker', until: ['idle'], timeoutMs: 3000 }), { status: 'timeout', timedOut: true });
    const transport = { request: async () => { throw new Error('Herdr request timed out after 10ms'); } };
    await assert.rejects(waitForAgent(transport, { target: 'worker', until: ['idle'], timeoutMs: 3000 }), /request timed out/);
  });

  it('fails closed for missing ping capability fields', async () => {
    await withServer({ ping: () => ({ type: 'pong', version: '0.7.5', protocol: 17, capabilities: {} }) }, async (client) => {
      await assert.rejects(client.ping(), /capabilities are incomplete/);
    });
  });
  it('launches omitted-tool roles with no tools and marks split panes as workers', async () => {
    const role = parseRoleFile('---\ndescription: no tools\n---\nPrompt body', '/no-tools.md');
    await withServer({}, async (client, server) => {
      const pane = await createPane(client, { currentPaneId: 'w1:p1', cwd: process.cwd(), env: buildWorkerEnv() });
      const started = await startAgent(client, { paneId: pane.pane_id, agentName: 'worker', terminalId: pane.terminal_id, role, cwd: process.cwd() });
      try {
        const split = server.requests.find((request) => request.method === 'pane.split');
        const args = server.requests.find((request) => request.method === 'agent.start').params.args;
        assert.equal(split.params.env.BALAUR_WORKER, '1');
        assert.ok(args.includes('--no-tools'));
        assert.ok(!args.includes('--tools'));
      } finally { await import('../pane-manager.js').then(({ removeRolePromptFile }) => removeRolePromptFile(started.promptFile)); }
    });
  });

  it('prepares a role launch as argv plus explicit skill paths', async () => {
    const cwd = fs.mkdtempSync(`${os.tmpdir()}/roles-`); fs.mkdirSync(resolve(cwd, '.agents/skills/custom'), { recursive: true }); fs.writeFileSync(resolve(cwd, '.agents/skills/custom/SKILL.md'), '# custom');
    const role = parseRoleFile('---\ndescription: test\nskills: custom\ntools: read, ext:pi-web-access/web_search\n---\nPrompt body', '/test.md');
    await withServer({}, async (client, server) => {
      const started = await startAgent(client, { paneId: 'w1:p2', agentName: 'worker', terminalId: 'term-2', role, cwd });
      try {
        const args = server.requests.find((request) => request.method === 'agent.start').params.args;
        assert.ok(args.some((arg) => String(arg).includes('ext:pi-web-access/web_search')));
        assert.ok(args.includes(resolve(cwd, '.agents/skills/custom/SKILL.md')));
        const promptFlag = args.findIndex((arg) => arg === '--system-prompt');
        assert.ok(promptFlag >= 0 && args[promptFlag + 1].endsWith('/system-prompt.md'), 'Pi #287 path-backed --system-prompt is used');
        assert.ok(!args.some((arg) => String(arg).includes('\n')));
      } finally { await import('../pane-manager.js').then(({ removeRolePromptFile }) => removeRolePromptFile(started.promptFile)); }
    });
  });
  it('resolves declared skills from both project skill roots', () => {
    const cwd = fs.mkdtempSync(`${os.tmpdir()}/skills-`);
    fs.mkdirSync(resolve(cwd, '.pi/skills/project'), { recursive: true }); fs.writeFileSync(resolve(cwd, '.pi/skills/project/SKILL.md'), '# project');
    fs.mkdirSync(resolve(cwd, '.agents/skills/agents'), { recursive: true }); fs.writeFileSync(resolve(cwd, '.agents/skills/agents/SKILL.md'), '# agents');
    const role = parseRoleFile('---\ndescription: skills\nskills: project, agents\n---\nPrompt', '/skills.md');
    assert.deepEqual(resolveRoleSkillArgs(role, cwd), ['--skill', resolve(cwd, '.pi/skills/project/SKILL.md'), '--skill', resolve(cwd, '.agents/skills/agents/SKILL.md')]);
  });
  it('accepts a real unnamed lead row in agent.list without rejecting workers', async () => {
    await withServer({ 'agent.list': () => ({ type: 'agent_list', agents: [{ pane_id: 'w1:p1', terminal_id: 'term-1', agent_status: 'idle' }, agent()] }) }, async (client) => {
      const rows = await listAgents(client);
      assert.equal(rows.length, 2);
      assert.equal(rows[0].name, undefined);
    });
  });
  it('creates valid unique <=32-character labels for long role names', () => {
    const role = 'a'.repeat(64);
    const one = makeAgentLabel(role, 12345, 'first');
    const two = makeAgentLabel(role, 12345, 'other');
    assert.match(one, /^[a-z0-9][a-z0-9_-]*$/);
    assert.ok(one.length <= 32);
    assert.notEqual(one, two);
  });
  it('uses a prompt activity wait and accepts blocked as a settled standalone wait outcome', async () => {
    await withServer({
      'agent.prompt': (request) => ({ type: 'agent_prompted', agent: { ...agent(request.params.target), agent_status: 'idle' } }),
      'agent.wait': (request) => ({ type: 'agent_info', agent: { ...agent(request.params.target), agent_status: 'blocked' } }),
    }, async (client, server) => {
      await promptAgent(client, { target: 'worker', text: 'next', wait: true, timeoutMs: 3000 });
      await waitForAgent(client, { target: 'worker', until: ['idle', 'done', 'blocked'], timeoutMs: 3000 });
      assert.deepEqual(server.requests.find((request) => request.method === 'agent.prompt').params.wait.until, ['working', 'idle', 'blocked', 'done', 'unknown']);
      assert.deepEqual(server.requests.find((request) => request.method === 'agent.wait').params.until, ['idle', 'done', 'blocked']);
    });
  });
  it('uses protocol-valid metadata source and token fields', async () => {
    await withServer({}, async (client, server) => {
      await reportPaneMetadata(client, 'w1:p2', { role: 'executor', bridge: 'herdr-agent', state: 'ready' });
      const params = server.requests.find((request) => request.method === 'pane.report_metadata').params;
      assert.equal(params.source, 'balaur-herdr-agent');
      assert.deepEqual(Object.keys(params.tokens), ['role', 'bridge', 'state']);
      assert.ok(Object.keys(params.tokens).every((key) => /^[A-Za-z0-9_-]{1,32}$/.test(key)));
    });
  });
  it('marks same-pane session replacement and blocks close', async () => {
    const handle = { agentName: 'worker', paneId: 'w1:p2', terminalId: 'term-2', sessionKind: 'id', sessionValue: session.value, status: 'ready' };
    await withServer({ 'agent.get': () => ({ type: 'agent_info', agent: { ...agent('worker', 'w1:p2'), agent_session: { ...session, value: '44444444-4444-4444-4444-444444444444' } } }) }, async (client, server) => {
      await assert.rejects(assertPinnedAgent(client, handle), /session identity was replaced/);
      assert.equal(handle.status, 'replaced');
      // A close caller must pin first; no close request has been issued.
      assert.equal(server.requests.some((request) => request.method === 'pane.close'), false);
    });
  });
  it('accepts exact agent/pane/terminal/session identity', async () => {
    const handle = { agentName: 'worker', paneId: 'w1:p2', terminalId: 'term-2', sessionKind: 'id', sessionValue: session.value, status: 'ready' };
    await withServer({}, async (client) => { assert.deepEqual(captureAgentIdentity(await assertPinnedAgent(client, handle)), { agentName: 'worker', paneId: 'w1:p2', terminalId: 'term-2', sessionKind: 'id', sessionValue: session.value }); });
  });
});
