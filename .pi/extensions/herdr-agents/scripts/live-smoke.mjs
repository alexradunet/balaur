#!/usr/bin/env node
/** Strict opt-in race smoke. Leaves its visible worker pane open. */
import { join } from "node:path";
import { readFile } from "node:fs/promises";
import { existsSync } from "node:fs";
import { HerdrClient, EXPECTED_PROTOCOL } from "../herdr-client.js";
import { parseRoleFile, roleNameFromFilename } from "../role-parser.js";
import { buildWorkerEnv, createPane, promptAgent, removeRolePromptFile, reportPaneMetadata, startAgent, waitForAgent, waitForInteractiveReady, waitForSessionIdentity } from "../pane-manager.js";
import { captureResolvedSessionBoundary, waitForFinalizedSessionResult, waitForPiSessionReference } from "../session-collector.js";

async function role(cwd) {
  const filePath = join(cwd, ".pi", "agents", "herdr-smoke.md");
  if (!existsSync(filePath)) throw new Error(`missing smoke role: ${filePath}`);
  return { name: roleNameFromFilename("herdr-smoke.md"), role: parseRoleFile(await readFile(filePath, "utf8"), filePath) };
}

async function promptAndCollect(client, agentName, session, nonce) {
  const boundary = await captureResolvedSessionBoundary(session);
  const acknowledgement = await promptAgent(client, { target: agentName, text: `Reply with exactly: ${nonce}`, wait: true, timeoutMs: 30000 });
  if (!['working', 'idle', 'blocked', 'done', 'unknown'].includes(acknowledgement.status)) throw new Error(`invalid prompt acknowledgement: ${acknowledgement.status}`);
  const waited = await waitForAgent(client, { target: agentName, until: ["idle", "done", "blocked"], timeoutMs: 120000 });
  if (waited.timedOut) throw new Error(`worker timed out after acknowledgement for ${nonce}`);
  if (waited.status === 'blocked') throw new Error(`worker blocked after acknowledgement for ${nonce}; pane remains open`);
  const filePath = await waitForPiSessionReference(session, 10000);
  const result = await waitForFinalizedSessionResult(filePath, 10000, undefined, boundary);
  if (result.stopReason !== "stop" || result.text !== nonce) throw new Error(`wrong result for ${nonce}: ${JSON.stringify(result.text)}`);
  return { acknowledgement: acknowledgement.status, boundary };
}

async function main() {
  if (!HerdrClient.isInHerdrPane()) throw new Error("not inside a Herdr pane");
  const env = HerdrClient.getHerdrEnv(); const cwd = process.cwd();
  const client = new HerdrClient({ socketPath: env.socketPath, timeoutMs: 10000 }); const ping = await client.ping();
  if (ping.protocol !== EXPECTED_PROTOCOL || !ping.capabilities.live_handoff) throw new Error(`protocol/capability mismatch: ${ping.protocol}`);
  const selected = await role(cwd); console.log(`Herdr ${ping.version} protocol ${ping.protocol}; role ${selected.name}`);
  const pane = await createPane(client, { currentPaneId: env.paneId, direction: "right", ratio: 0.5, cwd, env: buildWorkerEnv() }); console.log(`Created visible pane ${pane.pane_id}`);
  const started = await startAgent(client, { paneId: pane.pane_id, terminalId: pane.terminal_id, agentName: `smoke-${Date.now().toString(36)}`, role: selected.role, cwd });
  let identity;
  try { await waitForInteractiveReady(client, started.agent_name, 60000); identity = await waitForSessionIdentity(client, started.agent_name, 10000); } finally { await removeRolePromptFile(started.promptFile); }
  try { await reportPaneMetadata(client, pane.pane_id, { role: selected.name, bridge: "herdr-agent", state: "smoke" }); } catch (error) { console.warn(`Metadata warning (non-fatal): ${error.message}`); }
  const first = `SMOKE_ONE_${Date.now().toString(36)}`; const second = `SMOKE_TWO_${Date.now().toString(36)}`;
  const session = { kind: identity.sessionKind, value: identity.sessionValue };
  const firstResult = await promptAndCollect(client, started.agent_name, session, first);
  const secondResult = await promptAndCollect(client, started.agent_name, session, second);
  if (session.kind === 'id' && secondResult.boundary.lineCount === 0) throw new Error('second ID-backed prompt captured a header-only boundary');
  console.log(`SUCCESS pane=${pane.pane_id} agent=${started.agent_name} session=${identity.sessionKind}:${identity.sessionValue} acknowledgements=${firstResult.acknowledgement},${secondResult.acknowledgement} secondBoundaryLine=${secondResult.boundary.anchorLine} results=${first},${second}`);
  console.log(`Pane ${pane.pane_id} remains OPEN for inspection.`);
}
main().catch((error) => { console.error(`SMOKE FAILED: ${error.message}`); process.exitCode = 1; });
