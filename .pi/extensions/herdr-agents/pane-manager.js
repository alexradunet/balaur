/** Pane creation and protocol-17 Pi agent lifecycle management. */
import { existsSync } from 'node:fs';
import { randomUUID } from 'node:crypto';
import { mkdtemp, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { resolve, join } from 'node:path';
import { expectObject, HerdrRemoteError, requireString } from './herdr-client.js';
import { roleToPiArgs } from './role-parser.js';

export const METADATA_SOURCE = 'balaur-herdr-agent';
export const HERDR_AGENT_STATUSES = new Set(['idle', 'working', 'blocked', 'done', 'unknown']);
const SOURCE_RE = /^[A-Za-z0-9_-]{1,32}$/;

function sleep(ms, signal) {
  return new Promise((resolvePromise, reject) => {
    if (signal?.aborted) return reject(new Error('operation aborted'));
    let timer;
    const onAbort = () => finish(new Error('operation aborted'));
    const finish = (error) => {
      clearTimeout(timer);
      signal?.removeEventListener('abort', onAbort);
      if (error) reject(error); else resolvePromise();
    };
    timer = setTimeout(() => finish(), ms);
    signal?.addEventListener('abort', onAbort, { once: true });
  });
}
function resultOf(response, type, field) {
  const result = expectObject(response.result, `${type} result`);
  if (result.type !== type) throw new Error(`Herdr response missing type=${type}`);
  return expectObject(result[field], `${type}.${field}`);
}
function requirePane(value, label = 'pane') {
  const pane = expectObject(value, label);
  for (const key of ['pane_id', 'workspace_id', 'tab_id', 'terminal_id']) requireString(pane[key], `${label}.${key}`);
  return pane;
}
function requireAgent(value, label = 'agent', requireName = true) {
  const agent = expectObject(value, label);
  if (requireName) requireString(agent.name, `${label}.name`);
  requireString(agent.pane_id, `${label}.pane_id`);
  requireString(agent.terminal_id, `${label}.terminal_id`);
  if (!HERDR_AGENT_STATUSES.has(agent.agent_status)) throw new Error(`${label}.agent_status is missing or invalid`);
  // Protocol-17 omits false booleans through serde skip_serializing_if.
  agent.interactive_ready = agent.interactive_ready === true;
  agent.launch_pending = agent.launch_pending === true;
  return agent;
}
function requireSessionIdentity(agent) {
  const session = expectObject(agent.agent_session, 'agent.agent_session');
  const kind = requireString(session.kind, 'agent.agent_session.kind');
  const value = requireString(session.value, 'agent.agent_session.value');
  if (kind !== 'id' && kind !== 'path') throw new Error(`unsupported agent session kind: ${kind}`);
  return { kind, value };
}

export async function createPane(client, opts, signal) {
  const response = await client.request('pane.split', { target_pane_id: opts.currentPaneId, direction: opts.direction || 'right', ratio: opts.ratio || 0.5, cwd: opts.cwd, focus: false, env: opts.env || {} }, undefined, signal);
  return requirePane(resultOf(response, 'pane_info', 'pane'));
}

export async function startAgent(client, opts, signal) {
  const promptFile = await writeRolePromptFile(opts.role);
  try {
    const args = replacePromptArg(roleToPiArgs(opts.role), promptFile.path);
    args.push(...resolveRoleSkillArgs(opts.role, opts.cwd));
    const response = await client.request('agent.start', { name: opts.agentName, kind: 'pi', pane_id: opts.paneId, args, timeout_ms: opts.timeoutMs || 30000 }, undefined, signal);
    const result = resultOf(response, 'agent_started', 'agent');
    const agent = requireAgent(result);
    if (agent.pane_id !== opts.paneId || agent.terminal_id !== opts.terminalId) throw new Error('agent.start identity does not match the split pane');
    return { agent_name: agent.name, pane_id: agent.pane_id, terminal_id: agent.terminal_id, promptFile };
  } catch (error) {
    await removeRolePromptFile(promptFile);
    throw error;
  }
}

async function writeRolePromptFile(role) {
  const dir = await mkdtemp(join(tmpdir(), 'balaur-herdr-role-'));
  const path = join(dir, 'system-prompt.md');
  await writeFile(path, role.prompt, { encoding: 'utf8', mode: 0o600 });
  return { dir, path };
}
function replacePromptArg(args, path) {
  const index = args.findIndex((arg) => arg === '--system-prompt' || arg === '--append-system-prompt');
  if (index === -1) throw new Error('role prompt launch argument is missing');
  const copy = [...args];
  copy[index + 1] = path;
  return copy;
}
export async function removeRolePromptFile(promptFile) {
  if (promptFile?.dir) await rm(promptFile.dir, { recursive: true, force: true });
}

export function resolveRoleSkillArgs(role, cwd) {
  const args = [];
  for (const skill of role.skills || []) {
    const candidates = [
      resolve(cwd || process.cwd(), '.pi', 'skills', skill, 'SKILL.md'),
      resolve(cwd || process.cwd(), '.agents', 'skills', skill, 'SKILL.md'),
    ];
    const path = candidates.find((candidate) => existsSync(candidate));
    if (!path) throw new Error(`role skill '${skill}' is missing (checked ${candidates.join(' and ')})`);
    args.push('--skill', path);
  }
  return args;
}

export async function getAgent(client, target, signal) {
  const response = await client.request('agent.get', { target }, undefined, signal);
  return requireAgent(resultOf(response, 'agent_info', 'agent'));
}

export async function waitForInteractiveReady(client, agentName, timeoutMs = 60000, signal) {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    const agent = await getAgent(client, agentName, signal);
    if (agent.interactive_ready && !agent.launch_pending) return agent;
    await sleep(250, signal);
  }
  throw new Error(`agent ${agentName} did not become interactive-ready within ${timeoutMs}ms`);
}

export async function waitForAgent(client, opts, signal) {
  if (!Array.isArray(opts.until) || !opts.until.length || opts.until.some((status) => !HERDR_AGENT_STATUSES.has(status))) throw new Error('agent.wait until contains an invalid Herdr status');
  try {
    const response = await client.request('agent.wait', { target: opts.target, until: opts.until, timeout_ms: opts.timeoutMs || 60000 }, (opts.timeoutMs || 60000) + 5000, signal);
    const agent = requireAgent(resultOf(response, 'agent_info', 'agent'));
    if (!opts.until.includes(agent.agent_status)) throw new Error(`agent.wait returned ${agent.agent_status}, which is not in requested until set`);
    return { status: agent.agent_status, agent, timedOut: false };
  } catch (error) {
    if (error instanceof HerdrRemoteError && error.code === 'timeout') return { status: 'timeout', timedOut: true };
    throw error;
  }
}

export async function promptAgent(client, opts, signal) {
  const params = { target: opts.target, text: opts.text };
  if (opts.wait) params.wait = { until: opts.until || ['working', 'idle', 'blocked', 'done', 'unknown'], timeout_ms: opts.timeoutMs || 120000 };
  const response = await client.request('agent.prompt', params, opts.wait ? (opts.timeoutMs || 120000) + 5000 : undefined, signal);
  const agent = requireAgent(resultOf(response, 'agent_prompted', 'agent'));
  return { status: agent.agent_status, agent };
}

export async function readAgent(client, opts, signal) {
  const response = await client.request('agent.read', { target: opts.target, source: opts.source || 'recent', lines: opts.lines || 200, format: 'text', strip_ansi: true }, undefined, signal);
  const read = expectObject(resultOf(response, 'pane_read', 'read'), 'pane_read.read');
  return { text: typeof read.text === 'string' ? read.text : (() => { throw new Error('pane_read.read.text is missing'); })(), truncated: !!read.truncated };
}

export async function listAgents(client, signal) {
  const response = await client.request('agent.list', {}, undefined, signal);
  const result = expectObject(response.result, 'agent_list result');
  if (result.type !== 'agent_list' || !Array.isArray(result.agents)) throw new Error('Herdr agent.list response shape is invalid');
  // Herdr's lead pane can be an unnamed agent-list row. It is valid inventory,
  // but cannot match a persisted named worker identity.
  return result.agents.map((agent) => requireAgent(agent, 'agent', false));
}

export function assertAgentIdentity(handle, agent) {
  if (!handle.agentName) throw new Error('worker has no agent name');
  if (agent.name !== handle.agentName || agent.pane_id !== handle.paneId || agent.terminal_id !== handle.terminalId) {
    handle.status = 'replaced';
    throw new Error(`worker occupant was replaced for pane ${handle.paneId}`);
  }
  if (handle.sessionKind && (agent.agent_session?.kind !== handle.sessionKind || agent.agent_session?.value !== handle.sessionValue)) {
    handle.status = 'replaced';
    throw new Error(`worker session identity was replaced for pane ${handle.paneId}`);
  }
  return agent;
}

export async function assertPinnedAgent(client, handle, signal) {
  if (!handle.agentName) throw new Error('worker has no agent name');
  return assertAgentIdentity(handle, await getAgent(client, handle.agentName, signal));
}

export function makeAgentLabel(roleName, now = Date.now(), nonce = randomUUID().slice(0, 8)) {
  const suffix = `-${Number(now).toString(36)}-${nonce}`;
  // Roles are filename-validated, and the suffix is alphanumeric; trim only
  // the role prefix so the final Herdr name is valid and no longer than 32.
  return `${String(roleName).slice(0, Math.max(1, 32 - suffix.length))}${suffix}`.slice(0, 32);
}

export function captureAgentIdentity(agent) {
  const identity = requireSessionIdentity(agent);
  return { agentName: agent.name, paneId: agent.pane_id, terminalId: agent.terminal_id, sessionKind: identity.kind, sessionValue: identity.value };
}

export async function waitForSessionIdentity(client, agentName, timeoutMs = 10000, signal) {
  const deadline = Date.now() + timeoutMs;
  let lastError;
  while (Date.now() < deadline) {
    try { return captureAgentIdentity(await getAgent(client, agentName, signal)); } catch (error) { lastError = error; }
    await sleep(250, signal);
  }
  throw lastError || new Error(`agent ${agentName} did not report a session identity`);
}

export async function reportPaneMetadata(client, paneId, tokens, signal) {
  if (!SOURCE_RE.test(METADATA_SOURCE)) throw new Error('invalid bridge metadata source');
  for (const key of Object.keys(tokens)) if (!SOURCE_RE.test(key)) throw new Error(`invalid metadata token key: ${key}`);
  const response = await client.request('pane.report_metadata', { pane_id: paneId, source: METADATA_SOURCE, agent: 'pi', tokens }, undefined, signal);
  const result = expectObject(response.result, 'pane.report_metadata result');
  if (result.type !== 'ok') throw new Error('Herdr pane.report_metadata response shape is invalid');
}

export function buildWorkerEnv(baseEnv) {
  return { ...(baseEnv || {}), BALAUR_WORKER: '1' };
}
