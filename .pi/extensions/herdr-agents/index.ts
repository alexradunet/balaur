/** Balaur-owned visible Herdr worker bridge. */
import { readFile, readdir } from "node:fs/promises";
import { existsSync } from "node:fs";
import { join, resolve } from "node:path";
import { Type } from "typebox";
import { StringEnum } from "@earendil-works/pi-ai";
import { truncateHead, formatSize, DEFAULT_MAX_BYTES, DEFAULT_MAX_LINES, type ExtensionAPI } from "@earendil-works/pi-coding-agent";
import { Text } from "@earendil-works/pi-tui";
import { HerdrClient, HerdrRemoteError, EXPECTED_PROTOCOL } from "./herdr-client.js";
import { parseRoleFile, roleNameFromFilename, type RoleConfig } from "./role-parser.js";
import { assertAgentIdentity, assertPinnedAgent, buildWorkerEnv, captureAgentIdentity, createPane, listAgents, makeAgentLabel, promptAgent, readAgent, removeRolePromptFile, reportPaneMetadata, startAgent, waitForAgent, waitForInteractiveReady, waitForSessionIdentity } from "./pane-manager.js";
import { captureResolvedSessionBoundary, waitForFinalizedSessionResult, waitForPiSessionReference } from "./session-collector.js";
import { classifyHandleInventory, createHandleStore, createHandle, deserializeStore, listHandles, reconcileHandles, serializeStore, type WorkerHandle } from "./handle-store.js";

const ActionSchema = StringEnum(["start", "list", "status", "wait", "read", "prompt", "collect", "close"] as const);
const BOUNDED_STRING = (description: string, maxLength = 16000) => Type.String({ description, minLength: 1, maxLength });
const HerdrAgentParams = Type.Object({ action: ActionSchema, role: Type.Optional(BOUNDED_STRING("Role name from .pi/agents/*.md", 64)), handle: Type.Optional(BOUNDED_STRING("Worker handle ID", 64)), prompt: Type.Optional(BOUNDED_STRING("Prompt text", 16000)), timeout_ms: Type.Optional(Type.Integer({ minimum: 3000, maximum: 300000 })), lines: Type.Optional(Type.Integer({ minimum: 1, maximum: 5000 })) }, { additionalProperties: false });
const CUSTOM_STORE_TYPE = "balaur-herdr-agent-store";
const handleMap = new Map<string, WorkerHandle>();
const MAX_ERROR_BYTES = 4000;

export default function (pi: ExtensionAPI) {
  if (process.env.BALAUR_WORKER === "1" || !HerdrClient.isInHerdrPane()) return;
  const herdrEnv = HerdrClient.getHerdrEnv();
  if (!herdrEnv) return;
  const makeClient = () => new HerdrClient({ socketPath: herdrEnv.socketPath, timeoutMs: 10000 });
  const handleLeases = new Map<string, { action: string; token: symbol }>();
  const createStore = () => { const store = createHandleStore(); for (const handle of handleMap.values()) store.handles[handle.handleId] = handle; return store; };
  const setStore = (store: ReturnType<typeof createHandleStore>) => { handleMap.clear(); for (const handle of listHandles(store)) handleMap.set(handle.handleId, handle); };
  const persistStore = () => pi.appendEntry(CUSTOM_STORE_TYPE, { version: 1, store: serializeStore(createStore()) });
  const details = (extra: Record<string, unknown> = {}) => { const store = createStore(); return { store: serializeStore(store), handles: listHandles(store), ...extra }; };
  const save = (handle?: WorkerHandle) => { if (handle) handle.updatedAt = new Date().toISOString(); persistStore(); };
  const bounded = (text: string) => { const value = truncateHead(text, { maxBytes: DEFAULT_MAX_BYTES, maxLines: DEFAULT_MAX_LINES }); return value.truncated ? `${value.content}\n\n[Output truncated: ${value.outputLines}/${value.totalLines} lines, ${formatSize(value.outputBytes)}/${formatSize(value.totalBytes)}. Full evidence is in details.]` : value.content; };
  const boundedError = (error: unknown) => { const text = error instanceof Error ? error.message : String(error); return Buffer.byteLength(text) > MAX_ERROR_BYTES ? `${text.slice(0, MAX_ERROR_BYTES)}… [error truncated]` : text; };

  pi.on("session_start", async (_event, ctx) => {
    handleMap.clear();
    const entries = ctx?.sessionManager?.getBranch?.() || [];
    let restored: ReturnType<typeof createHandleStore> | undefined;
    for (const entry of entries) {
      let raw: unknown;
      if (entry?.type === "custom" && entry.customType === CUSTOM_STORE_TYPE) {
        if (entry.data?.version !== 1) continue;
        raw = entry.data?.store;
      } else if (entry?.type === "message" && entry.message?.role === "toolResult" && entry.message.toolName === "herdr_agent") raw = entry.message.details?.store;
      if (typeof raw !== "string") continue;
      try { restored = deserializeStore(raw); } catch { /* retain the latest fully valid snapshot only */ }
    }
    if (!restored) return;
    setStore(restored);
    try { setStore(reconcileHandles(createStore(), await listAgents(makeClient()))); persistStore(); } catch { /* unavailable Herdr never rebinds persisted handles */ }
  });

  function requireHandle(id: string | undefined) { if (!id) throw new Error("handle is required for this action"); const handle = handleMap.get(id); if (!handle) throw new Error(`Unknown handle: ${id}`); return handle; }
  function validateReturnedIdentity(handle: WorkerHandle, agent: any) {
    try { return assertAgentIdentity(handle, agent); }
    catch (error) { if (handle.status === "replaced") save(handle); throw error; }
  }
  async function pinned(handle: WorkerHandle, signal?: AbortSignal) {
    const priorStatus = handle.status;
    try { return await assertPinnedAgent(makeClient(), handle, signal); }
    catch (error) { if (handle.status === "replaced" && priorStatus !== "replaced") save(handle); throw error; }
  }
  function operational(handle: WorkerHandle) { if (!handle.sessionKind || !handle.sessionValue || ["starting", "error", "missing", "replaced"].includes(handle.status)) throw new Error(`worker ${handle.handleId} is not ready for operational actions`); }

  async function discoverRoles(cwd: string) {
    const roles = new Map<string, { role: RoleConfig; filePath: string }>(); const errors = new Map<string, string>(); const agentsDir = resolve(cwd, ".pi", "agents");
    if (!existsSync(agentsDir)) return { roles, errors };
    for (const entry of await readdir(agentsDir)) if (entry.endsWith(".md")) { const filePath = join(agentsDir, entry); let name = entry.slice(0, -3); try { name = roleNameFromFilename(entry); roles.set(name, { role: parseRoleFile(await readFile(filePath, "utf8"), filePath), filePath }); } catch (error) { errors.set(name, boundedError(error)); } }
    return { roles, errors };
  }

  async function withHandleLease<T>(handleId: string | undefined, action: string, callback: () => Promise<T>): Promise<T> {
    if (!handleId) throw new Error("handle is required for this action");
    const active = handleLeases.get(handleId);
    if (active) throw new Error(`worker ${handleId} is busy with ${active.action}; cannot run ${action}`);
    const token = Symbol(action);
    handleLeases.set(handleId, { action, token });
    try { return await callback(); }
    finally { if (handleLeases.get(handleId)?.token === token) handleLeases.delete(handleId); }
  }

  pi.registerTool({
    name: "herdr_agent", label: "Herdr Agent", description: "Start and control visible interactive Pi workers in Herdr panes. read is diagnostic; collect is the authoritative latest bridge-prompt Pi JSONL result.", promptSnippet: "Start and control visible Herdr Pi workers", promptGuidelines: ["Use herdr_agent for visible Herdr workers.", "Use herdr_agent collect, not read, for the authoritative finalized result."], parameters: HerdrAgentParams,
    async execute(_id, params, signal, _update, ctx) { if (!HerdrClient.isInHerdrPane()) throw new Error("herdr_agent is only available inside a Herdr pane"); try { switch (params.action) { case "start": return await start(params, ctx, signal); case "list": return { content: [{ type: "text", text: bounded(renderList()) }], details: details() }; case "status": return await withHandleLease(params.handle, "status", () => status(params, signal)); case "wait": return await withHandleLease(params.handle, "wait", () => wait(params, signal)); case "read": return await withHandleLease(params.handle, "read", () => read(params, signal)); case "prompt": return await withHandleLease(params.handle, "prompt", () => prompt(params, signal)); case "collect": return await withHandleLease(params.handle, "collect", () => collect(params, signal)); case "close": return close(params); } } catch (error) { throw new Error(boundedError(error)); } },
    renderCall(args, theme) { const suffix = args.role || args.handle || ""; return new Text(theme.fg("toolTitle", theme.bold("herdr_agent ")) + theme.fg("accent", `${args.action} ${suffix}`.trim()), 0, 0); },
    renderResult(result, { expanded }, theme) { const text = result.content[0]?.type === "text" ? result.content[0].text : "(no output)"; const handles = (result.details as any)?.handles || []; const extra = expanded && handles.length ? `\n${handles.map((h: WorkerHandle) => `${h.handleId} ${h.status} ${h.role} @ ${h.paneId}`).join("\n")}` : ""; return new Text(text + extra, 0, 0); },
  });

  async function start(params: any, ctx: any, signal?: AbortSignal) {
    if (!params.role) throw new Error("role is required for start"); const catalog = await discoverRoles(ctx.cwd); const entry = catalog.roles.get(params.role);
    if (!entry) throw new Error(catalog.errors.get(params.role) || `Unknown role '${params.role}'. Available: ${[...catalog.roles.keys()].join(", ") || "none"}`);
    const client = makeClient(); const ping = await client.ping(signal); if (ping.protocol !== EXPECTED_PROTOCOL || !ping.capabilities.live_handoff) throw new Error(`Herdr protocol/capability mismatch (protocol ${ping.protocol})`);
    const name = makeAgentLabel(params.role); const pane = await createPane(client, { currentPaneId: herdrEnv.paneId, direction: "right", ratio: 0.5, cwd: ctx.cwd, env: buildWorkerEnv() }, signal);
    const handle = createHandle({ paneId: pane.pane_id, terminalId: pane.terminal_id, role: params.role, workspaceId: pane.workspace_id, worktreePath: ctx.cwd, agentName: name }); handleMap.set(handle.handleId, handle); save(handle);
    let started: any;
    try {
      started = await startAgent(client, { paneId: pane.pane_id, terminalId: pane.terminal_id, agentName: name, role: entry.role, cwd: ctx.cwd }, signal);
      await waitForInteractiveReady(client, name, 60000, signal);
      const identity = await waitForSessionIdentity(client, name, 10000, signal);
      if (identity.agentName !== handle.agentName || identity.paneId !== handle.paneId || identity.terminalId !== handle.terminalId || handle.sessionKind) throw new Error("worker launch identity did not match its provisional handle");
      Object.assign(handle, identity, { status: "idle", error: undefined }); save(handle);
    } catch (error) {
      handle.status = "error"; handle.error = boundedError(error); save(handle);
      if (started?.promptFile) await removeRolePromptFile(started.promptFile);
      return { content: [{ type: "text", text: bounded(`Worker pane ${handle.paneId} is retained as ${handle.handleId} but launch completion is uncertain: ${handle.error}\nCall status to recover; do not start a replacement.`) }], details: details({ handle: handle.handleId, recovery: "status" }) };
    } finally { if (started?.promptFile) await removeRolePromptFile(started.promptFile); }
    let metadataWarning: string | undefined;
    try { await reportPaneMetadata(client, pane.pane_id, { role: params.role, bridge: "herdr-agent", state: "ready" }, signal); } catch (error) { metadataWarning = boundedError(error); }
    return { content: [{ type: "text", text: bounded(`Started ${params.role}: handle ${handle.handleId}, pane ${pane.pane_id}, session ${handle.sessionKind}:${handle.sessionValue}.${metadataWarning ? ` Metadata warning: ${metadataWarning}` : ""}`) }], details: details({ handle: handle.handleId, session: { kind: handle.sessionKind, value: handle.sessionValue }, metadataWarning }) };
  }
  const renderList = () => handleMap.size ? [...handleMap.values()].map((h) => `${h.handleId}: ${h.role} @ ${h.paneId} [${h.status}]`).join("\n") : "No active workers.";
  const statusResult = (handle: WorkerHandle) => ({ content: [{ type: "text", text: bounded(`${handle.handleId}: ${handle.status}\nagent=${handle.agentName}\npane=${handle.paneId}\nsession=${handle.sessionKind}:${handle.sessionValue}`) }], details: details({ handle: handle.handleId }) });
  async function status(params: any, signal?: AbortSignal) {
    const handle = requireHandle(params.handle);
    if (handle.status === "replaced") return statusResult(handle);
    try {
      const agent = await pinned(handle, signal);
      if (!handle.sessionKind || !handle.sessionValue) Object.assign(handle, captureAgentIdentity(agent), { error: undefined });
      handle.status = agent.agent_status;
      save(handle);
    } catch (error) {
      if (!(error instanceof HerdrRemoteError) || error.code !== "agent_not_found") throw error;
      const inventory = await listAgents(makeClient(), signal);
      const classification = classifyHandleInventory(handle, inventory);
      if (classification !== "exact") handle.status = classification;
      save(handle);
    }
    return statusResult(handle);
  }
  async function wait(params: any, signal?: AbortSignal) { const handle = requireHandle(params.handle); operational(handle); await pinned(handle, signal); const result = await waitForAgent(makeClient(), { target: handle.agentName, until: ["idle", "done", "blocked"], timeoutMs: params.timeout_ms || 120000 }, signal); if (result.timedOut) return { content: [{ type: "text", text: bounded(`Worker ${handle.handleId} timed out; it was not killed.`) }], details: details({ handle: handle.handleId, timedOut: true }) }; validateReturnedIdentity(handle, result.agent); handle.status = result.status; save(handle); return { content: [{ type: "text", text: bounded(`Worker ${handle.handleId} reached ${result.status}.`) }], details: details({ handle: handle.handleId }) }; }
  async function read(params: any, signal?: AbortSignal) { const handle = requireHandle(params.handle); operational(handle); await pinned(handle, signal); const output = await readAgent(makeClient(), { target: handle.agentName, lines: params.lines || 200 }, signal); return { content: [{ type: "text", text: bounded(`[Diagnostic terminal output; use collect for the finalized result]\n${output.text}`) }], details: details({ handle: handle.handleId, rawOutput: output.text, truncated: output.truncated }) }; }
  async function prompt(params: any, signal?: AbortSignal) { if (!params.prompt) throw new Error("prompt is required for prompt"); const handle = requireHandle(params.handle); operational(handle); const agent = await pinned(handle, signal); handle.status = agent.agent_status; save(handle); if (handle.status !== "idle" && handle.status !== "blocked") throw new Error(`worker ${handle.handleId} must be idle or blocked before prompting`); const session = { kind: handle.sessionKind, value: handle.sessionValue }; handle.promptBoundary = await captureResolvedSessionBoundary(session); handle.promptPhase = "submitting"; save(handle); try { const prompted = await promptAgent(makeClient(), { target: handle.agentName, text: params.prompt, wait: true, timeoutMs: params.timeout_ms || 30000 }, signal); validateReturnedIdentity(handle, prompted.agent); handle.promptPhase = "accepted"; handle.status = prompted.status; save(handle); return { content: [{ type: "text", text: bounded(`Prompted ${handle.handleId}; Herdr observed ${prompted.status} after submission.`) }], details: details({ handle: handle.handleId }) }; } catch (error) { handle.promptPhase = "uncertain"; if (handle.status !== "replaced") handle.status = "unknown"; handle.error = boundedError(error); save(handle); throw error; } }
  async function collect(params: any, signal?: AbortSignal) { const handle = requireHandle(params.handle); operational(handle); await pinned(handle, signal); if (handle.promptPhase === "uncertain") return { content: [{ type: "text", text: "Latest bridge prompt submission is uncertain; inspect the visible worker before collecting." }], details: details({ handle: handle.handleId, stopReason: "uncertain" }) }; if (handle.promptPhase !== "accepted" || !handle.promptBoundary) return { content: [{ type: "text", text: "No bridge prompt recorded for this worker." }], details: details({ handle: handle.handleId, stopReason: "no bridge prompt recorded" }) }; const filePath = await waitForPiSessionReference({ kind: handle.sessionKind, value: handle.sessionValue }, 10000, signal); const result = await waitForFinalizedSessionResult(filePath, 10000, signal, handle.promptBoundary); if (result.stopReason === "incomplete") return { content: [{ type: "text", text: "Worker latest bridge prompt has no finalized result yet." }], details: details({ handle: handle.handleId, stopReason: result.stopReason }) }; return { content: [{ type: "text", text: bounded(result.text) }], details: details({ handle: handle.handleId, sessionPath: filePath, stopReason: result.stopReason, toolCalls: result.toolCalls, usage: result.usage, fullText: result.text }) }; }
  function close(params: any) { const handle = requireHandle(params.handle); throw new Error(`Automated close is disabled under Herdr protocol 17; retained handle ${handle.handleId} in pane ${handle.paneId} must be inspected and closed manually.`); }
}
