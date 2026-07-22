/**
 * Balaur Herdr agent bridge — project-local Pi extension.
 *
 * Starts and controls interactive Pi workers in visible, persistent Herdr
 * panes. This stage proves the generic bridge while intentionally retaining
 * `npm:@tintinweb/pi-subagents` so existing workflows remain usable until
 * the final cutover.
 *
 * @module herdr-agents
 */

import { readFile, readdir } from "node:fs/promises";
import { existsSync } from "node:fs";
import { resolve, join, dirname } from "node:path";
import { fileURLToPath } from "node:url";
import { Type } from "typebox";
import { StringEnum } from "@earendil-works/pi-ai";
import {
  truncateHead,
  formatSize,
  DEFAULT_MAX_BYTES,
  DEFAULT_MAX_LINES,
  type ExtensionAPI,
} from "@earendil-works/pi-coding-agent";
import { Text } from "@earendil-works/pi-tui";

import {
  HerdrClient,
  EXPECTED_PROTOCOL,
} from "./herdr-client.js";
import {
  parseRoleFile,
  roleNameFromFilename,
  roleToPiArgs,
  type RoleConfig,
} from "./role-parser.js";
import {
  createPane,
  startAgent,
  waitForAgent,
  promptAgent,
  readAgent,
  listAgents,
  getAgent,
  closePane,
  reportPaneMetadata,
  buildWorkerEnv,
} from "./pane-manager.js";
import { collectSessionResult } from "./session-collector.js";
import {
  createHandleStore,
  createHandle,
  addHandle,
  updateHandle,
  removeHandle,
  getHandle,
  listHandles,
  reconcileHandles,
  serializeStore,
  deserializeStore,
  type WorkerHandle,
} from "./handle-store.js";

const __dirname = dirname(fileURLToPath(import.meta.url));

const ActionSchema = StringEnum([
  "start",
  "list",
  "status",
  "wait",
  "read",
  "prompt",
  "collect",
  "close",
] as const);

const HerdrAgentParams = Type.Object({
  action: ActionSchema,
  role: Type.Optional(Type.String({ description: "Role name from .pi/agents/*.md (for start)" })),
  handle: Type.Optional(Type.String({ description: "Worker handle ID (for status/wait/read/prompt/collect/close)" })),
  prompt: Type.Optional(Type.String({ description: "Prompt text (for prompt action)" })),
  timeout_ms: Type.Optional(
    Type.Integer({ minimum: 3000, maximum: 300000, description: "Timeout in milliseconds (for wait/prompt)" }),
  ),
  lines: Type.Optional(
    Type.Integer({ minimum: 1, maximum: 5000, description: "Number of terminal lines (for read)" }),
  ),
});

/** @type {Map<string, WorkerHandle>} */
const handleMap = new Map();

export default function (pi: ExtensionAPI) {
  // Do not activate inside worker sessions (BALAUR_WORKER=1).
  // Workers must not be able to spawn orchestration tools.
  if (process.env.BALAUR_WORKER === "1") {
    return;
  }

  // Do not activate outside a Herdr pane — fail closed.
  if (!HerdrClient.isInHerdrPane()) {
    return;
  }

  const herdrEnv = HerdrClient.getHerdrEnv();
  if (!herdrEnv) return;
  const { socketPath, paneId: leadPaneId } = herdrEnv;

  const makeClient = () => new HerdrClient({ socketPath, timeoutMs: 10000 });

  // Reconstruct state from session on reload/resume.
  pi.on("session_start", async (_event, ctx) => {
    handleMap.clear();
    // Reconstruct handles from tool result details in the session branch.
    if (ctx?.sessionManager?.getBranch) {
      try {
        for (const entry of ctx.sessionManager.getBranch()) {
          if (entry?.type === "message" && entry.message?.role === "toolResult") {
            const toolName = entry.message.toolName;
            if (toolName === "herdr_agent" && entry.message.details?.store) {
              const store = deserializeStore(entry.message.details.store);
              for (const h of listHandles(store)) {
                handleMap.set(h.handleId, h);
              }
            }
          }
        }
        // Reconcile against current panes after reconstruction.
        await reconcileStore();
      } catch {
        // Best-effort reconstruction; continue with empty store.
      }
    }
  });

  async function reconcileStore() {
    if (handleMap.size === 0) return;
    try {
      const client = makeClient();
      const agents = await listAgents(client);
      const paneIds = new Set(agents.map((a: any) => a.pane_id));
      for (const handle of handleMap.values()) {
        if (!paneIds.has(handle.paneId)) {
          handle.status = "missing";
        }
      }
    } catch {
      // If we can't reach Herdr, leave handles as-is.
    }
  }

  async function discoverRoles(): Promise<Map<string, { role: RoleConfig; filePath: string }>> {
    const roles = new Map();
    const agentsDir = resolve(process.cwd(), ".pi/agents");
    if (!existsSync(agentsDir)) return roles;
    const entries = await readdir(agentsDir);
    for (const entry of entries) {
      if (!entry.endsWith(".md")) continue;
      const filePath = join(agentsDir, entry);
      try {
        const content = await readFile(filePath, "utf-8");
        const role = parseRoleFile(content, filePath);
        const name = roleNameFromFilename(entry);
        roles.set(name, { role, filePath });
      } catch {
        // Skip unparseable role files.
      }
    }
    return roles;
  }

  function truncateOutput(text: string): string {
    const truncation = truncateHead(text, {
      maxLines: DEFAULT_MAX_LINES,
      maxBytes: DEFAULT_MAX_BYTES,
    });
    let result = truncation.content;
    if (truncation.truncated) {
      result += `\n\n[Output truncated: ${truncation.outputLines} of ${truncation.totalLines} lines (${formatSize(truncation.outputBytes)} of ${formatSize(truncation.totalBytes)}). Full evidence preserved in tool details.]`;
    }
    return result;
  }

  function storeToDetails() {
    const store = createHandleStore();
    for (const h of handleMap.values()) {
      store.handles[h.handleId] = h;
    }
    return { store: serializeStore(store), handles: listHandles(store) };
  }

  pi.registerTool({
    name: "herdr_agent",
    label: "Herdr Agent",
    description: [
      "Start and control interactive Pi workers in visible, persistent Herdr panes.",
      "Actions: start (create worker), list (show workers), status (inspect), wait (block until idle/done),",
      "read (diagnostic terminal output), prompt (send message), collect (authoritative finalized result),",
      "close (human-confirmed pane cleanup).",
      "Returns from start as soon as the worker is ready; does not wait for task completion.",
    ].join(" "),
    promptSnippet: "Start and control visible Herdr Pi workers",
    promptGuidelines: [
      "Use herdr_agent to start visible worker Pi sessions in Herdr panes for delegated implementation or review.",
      "Always use collect (not read) for the authoritative finalized worker result; read is diagnostic terminal output only.",
      "Wait for a worker to reach idle/done before collecting its result.",
    ],
    parameters: HerdrAgentParams,

    async execute(_toolCallId, params, signal, _onUpdate, ctx) {
      const action = params.action;

      // Fail closed outside Herdr.
      if (!HerdrClient.isInHerdrPane()) {
        throw new Error("herdr_agent is only available inside a Herdr pane");
      }

      try {
        switch (action) {
          case "start":
            return await handleStart(params, ctx, signal);
          case "list":
            return await handleList();
          case "status":
            return await handleStatus(params);
          case "wait":
            return await handleWait(params, signal);
          case "read":
            return await handleRead(params);
          case "prompt":
            return await handlePrompt(params, signal);
          case "collect":
            return await handleCollect(params);
          case "close":
            return await handleClose(params, ctx);
          default:
            throw new Error(`Unknown action: ${action}`);
        }
      } catch (err) {
        // Return error as a tool result (not thrown) so the lead sees it.
        const message = err instanceof Error ? err.message : String(err);
        return {
          content: [{ type: "text", text: `Error: ${message}` }],
          details: { ...storeToDetails(), error: message },
          isError: true,
        };
      }
    },

    renderCall(args, theme, _context) {
      const action = args.action || "...";
      let text =
        theme.fg("toolTitle", theme.bold("herdr_agent ")) +
        theme.fg("accent", action);

      if (args.role) {
        text += " " + theme.fg("accent", args.role);
      }
      if (args.handle) {
        text += " " + theme.fg("dim", args.handle);
      }
      if (args.prompt) {
        const preview = args.prompt.length > 50 ? `${args.prompt.slice(0, 50)}...` : args.prompt;
        text += `\n  ${theme.fg("dim", preview)}`;
      }
      return new Text(text, 0, 0);
    },

    renderResult(result, { expanded }, theme, _context) {
      const text = result.content[0];
      let content = text?.type === "text" ? text.text : "(no output)";

      const details = result.details as any;
      if (details?.error) {
        return new Text(theme.fg("error", `Error: ${details.error}`), 0, 0);
      }

      if (expanded && details?.handles?.length > 0) {
        content += "\n\nWorkers:";
        for (const h of details.handles) {
          const status = h.status === "ready" || h.status === "idle" || h.status === "done"
            ? theme.fg("success", h.status)
            : h.status === "error" || h.status === "missing" || h.status === "replaced"
              ? theme.fg("error", h.status)
              : theme.fg("warning", h.status);
          content += `\n  ${theme.fg("accent", h.handleId)} ${status} ${theme.fg("dim", `(${h.role} @ ${h.paneId})`)}`;
        }
      }
      return new Text(content, 0, 0);
    },
  });

  async function handleStart(params: any, ctx: any, signal: AbortSignal | undefined) {
    if (!params.role) {
      throw new Error("role is required for start action");
    }

    const roles = await discoverRoles();
    const roleEntry = roles.get(params.role);
    if (!roleEntry) {
      const available = [...roles.keys()].join(", ") || "none";
      throw new Error(`Unknown role '${params.role}'. Available: ${available}`);
    }

    const client = makeClient();

    // Verify protocol version.
    const ping = await client.ping();
    if (ping.protocol !== EXPECTED_PROTOCOL) {
      throw new Error(
        `Herdr protocol mismatch: expected ${EXPECTED_PROTOCOL}, got ${ping.protocol}`
      );
    }

    // Create a new visible pane by splitting the lead pane.
    const workerEnv = buildWorkerEnv();
    const pane = await createPane(client, {
      currentPaneId: leadPaneId,
      direction: "right",
      ratio: 0.5,
      cwd: ctx?.cwd || process.cwd(),
      env: workerEnv,
    });

    // Start an interactive Pi agent in the new pane.
    const started = await startAgent(client, {
      paneId: pane.pane_id,
      agentName: params.role,
      role: roleEntry.role,
      cwd: ctx?.cwd || process.cwd(),
    });

    // Create a stable handle.
    let handle = createHandle({
      paneId: pane.pane_id,
      role: params.role,
      workspaceId: pane.workspace_id,
      worktreePath: ctx?.cwd || process.cwd(),
    });
    handle = {
      ...handle,
      agentName: started.agent_name,
      status: "ready",
    };
    handleMap.set(handle.handleId, handle);

    // Publish pane metadata for role and bridge state.
    await reportPaneMetadata(client, pane.pane_id, {
      "balaur.role": params.role,
      "balaur.bridge": "herdr-agent",
    }).catch(() => { /* best-effort */ });

    const summary = `Started worker '${params.role}' in pane ${pane.pane_id} (handle: ${handle.handleId}). The worker is interactive and ready for prompts.`;
    return {
      content: [{ type: "text", text: truncateOutput(summary) }],
      details: { ...storeToDetails(), handle: handle.handleId, paneId: pane.pane_id },
    };
  }

  async function handleList() {
    const list = [...handleMap.values()];
    if (list.length === 0) {
      return {
        content: [{ type: "text", text: "No active workers." }],
        details: storeToDetails(),
      };
    }
    const lines = list.map((h) =>
      `${h.handleId}: ${h.role} @ ${h.paneId} [${h.status}]${h.agentName ? ` agent=${h.agentName}` : ""}`
    );
    return {
      content: [{ type: "text", text: truncateOutput(lines.join("\n")) }],
      details: storeToDetails(),
    };
  }

  async function handleStatus(params: any) {
    if (!params.handle) throw new Error("handle is required for status action");
    const handle = handleMap.get(params.handle);
    if (!handle) throw new Error(`Unknown handle: ${params.handle}`);

    const client = makeClient();
    let agentStatus = "unknown";
    let occupant = handle.agentName;
    try {
      if (handle.agentName) {
        const agent = await getAgent(client, handle.agentName);
        agentStatus = agent?.agent_status || "unknown";
        // Detect replaced occupant.
        if (agent?.name && agent.name !== handle.agentName) {
          handle.status = "replaced";
          occupant = agent.name;
        }
      }
    } catch {
      // Can't reach agent — mark as missing if not already.
      if (handle.status !== "missing") {
        handle.status = "missing";
      }
    }

    const lines = [
      `Handle: ${handle.handleId}`,
      `Role: ${handle.role}`,
      `Pane: ${handle.paneId}`,
      `Agent: ${occupant || "(none)"}`,
      `Status: ${handle.status}`,
      `Agent status: ${agentStatus}`,
      handle.worktreePath ? `CWD: ${handle.worktreePath}` : null,
      handle.sessionPath ? `Session: ${handle.sessionPath}` : null,
    ].filter(Boolean);
    return {
      content: [{ type: "text", text: truncateOutput(lines.join("\n")) }],
      details: { ...storeToDetails(), handle: handle.handleId },
    };
  }

  async function handleWait(params: any, signal: AbortSignal | undefined) {
    if (!params.handle) throw new Error("handle is required for wait action");
    const handle = handleMap.get(params.handle);
    if (!handle) throw new Error(`Unknown handle: ${params.handle}`);
    if (!handle.agentName) throw new Error("worker has no agent name");

    const client = makeClient();
    const result = await waitForAgent(client, {
      target: handle.agentName,
      until: ["idle", "done"],
      timeoutMs: params.timeout_ms || 120000,
    });

    if (result.timedOut) {
      // Timeouts never kill a worker — just report blocked/timeout.
      handle.status = "working";
      return {
        content: [{
          type: "text",
          text: `Worker '${handle.role}' is still running (timeout after ${params.timeout_ms || 120000}ms). Use read for diagnostic output or wait again.`,
        }],
        details: { ...storeToDetails(), handle: handle.handleId, timedOut: true },
      };
    }

    handle.status = result.status === "done" ? "done" : "idle";
    return {
      content: [{
        type: "text",
        text: `Worker '${handle.role}' reached status: ${result.status}`,
      }],
      details: { ...storeToDetails(), handle: handle.handleId, status: result.status },
    };
  }

  async function handleRead(params: any) {
    if (!params.handle) throw new Error("handle is required for read action");
    const handle = handleMap.get(params.handle);
    if (!handle) throw new Error(`Unknown handle: ${params.handle}`);
    if (!handle.agentName) throw new Error("worker has no agent name");

    const client = makeClient();
    const result = await readAgent(client, {
      target: handle.agentName,
      source: "recent",
      lines: params.lines || 200,
    });

    const header = `[diagnostic terminal output for '${handle.role}' — this is NOT the finalized result; use collect for that]`;
    const text = `${header}\n${result.text}`;
    return {
      content: [{ type: "text", text: truncateOutput(text) }],
      details: {
        ...storeToDetails(),
        handle: handle.handleId,
        truncated: result.truncated,
        rawOutput: result.text, // Preserve full evidence in details
      },
    };
  }

  async function handlePrompt(params: any, signal: AbortSignal | undefined) {
    if (!params.handle) throw new Error("handle is required for prompt action");
    if (!params.prompt) throw new Error("prompt text is required for prompt action");
    const handle = handleMap.get(params.handle);
    if (!handle) throw new Error(`Unknown handle: ${params.handle}`);
    if (!handle.agentName) throw new Error("worker has no agent name");

    const client = makeClient();
    // Prompt without waiting — return immediately.
    const result = await promptAgent(client, {
      target: handle.agentName,
      text: params.prompt,
      wait: false,
    });

    handle.status = "working";
    return {
      content: [{
        type: "text",
        text: `Prompted worker '${handle.role}'. Use wait then collect for the result.`,
      }],
      details: { ...storeToDetails(), handle: handle.handleId, status: result.status },
    };
  }

  async function handleCollect(params: any) {
    if (!params.handle) throw new Error("handle is required for collect action");
    const handle = handleMap.get(params.handle);
    if (!handle) throw new Error(`Unknown handle: ${params.handle}`);

    // The authoritative result comes from the finalized Pi session JSONL.
    // We need the session file path. Herdr's agent.get may report it via
    // agent_session_path, but the bridge also tries to discover it.
    let sessionPath = handle.sessionPath;

    if (!sessionPath) {
      // Try to get the session path from the agent info.
      try {
        const client = makeClient();
        if (handle.agentName) {
          const agent = await getAgent(client, handle.agentName);
          // Herdr's AgentInfo has agent_session which has value (path or id).
          if (agent?.agent_session?.value) {
            sessionPath = agent.agent_session.value;
            handle.sessionPath = sessionPath;
          }
        }
      } catch {
        // Fall through to error.
      }
    }

    if (!sessionPath) {
      throw new Error(
        `No session path available for worker '${handle.role}'. ` +
        "Ensure the worker has produced output (wait for idle/done first)."
      );
    }

    const result = await collectSessionResult(sessionPath);

    if (result.stopReason === "incomplete") {
      return {
        content: [{
          type: "text",
          text: `Worker '${handle.role}' session is incomplete (no finalized assistant result yet). Wait for idle/done first.`,
        }],
        details: { ...storeToDetails(), handle: handle.handleId, stopReason: result.stopReason },
      };
    }

    handle.status = result.stopReason === "end" ? "done" : handle.status;

    // Build a concise summary for the model, with full evidence in details.
    let summary = result.text;
    if (result.toolCalls.length > 0) {
      summary += `\n\n[Tool calls: ${result.toolCalls.length}, turns: ${result.turns}]`;
    }

    return {
      content: [{ type: "text", text: truncateOutput(summary) }],
      details: {
        ...storeToDetails(),
        handle: handle.handleId,
        stopReason: result.stopReason,
        model: result.model,
        usage: result.usage,
        turns: result.turns,
        toolCalls: result.toolCalls,
        fullText: result.text, // Preserve full text in details
      },
    };
  }

  async function handleClose(params: any, ctx: any) {
    if (!params.handle) throw new Error("handle is required for close action");
    const handle = handleMap.get(params.handle);
    if (!handle) throw new Error(`Unknown handle: ${params.handle}`);

    // Close requires interactive human confirmation.
    if (!ctx?.hasUI) {
      throw new Error("close requires interactive confirmation — no UI available");
    }

    const confirmed = await ctx.ui.confirm(
      "Close worker pane?",
      `This will close Herdr pane ${handle.paneId} (worker '${handle.role}'). The pane content is not saved.`,
    );
    if (!confirmed) {
      return {
        content: [{ type: "text", text: "Close cancelled by user." }],
        details: { ...storeToDetails(), handle: handle.handleId, cancelled: true },
      };
    }

    const client = makeClient();
    await closePane(client, handle.paneId);
    handleMap.delete(handle.handleId);

    return {
      content: [{
        type: "text",
        text: `Closed worker '${handle.role}' (pane ${handle.paneId}, handle ${handle.handleId}).`,
      }],
      details: { ...storeToDetails(), closed: handle.handleId },
    };
  }
}
