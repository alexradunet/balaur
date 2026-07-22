#!/usr/bin/env node
/**
 * Opt-in live smoke test for the Herdr agent bridge.
 *
 * Starts a harmless visible Pi worker in a Herdr pane, prompts it, waits,
 * collects its finalized result, and does NOT auto-close its pane.
 *
 * This script must be run from inside a Herdr pane (the lead session).
 * It creates a visible sibling pane that stays open for inspection.
 *
 * Usage:
 *   node .pi/extensions/herdr-agents/scripts/live-smoke.mjs
 *
 * Prerequisites:
 *   - Running inside Herdr (HERDR_ENV=1, HERDR_SOCKET_PATH, HERDR_PANE_ID set)
 *   - Herdr 0.7.5 (protocol 17)
 *   - Pi installed and on PATH
 *   - A .pi/agents/*.md role file (defaults to looking for "executor" or first role)
 *
 * The script exits 0 on success and 1 on failure. It never closes the
 * created pane — inspect it manually in Herdr afterward.
 */

import { resolve, join, dirname } from "node:path";
import { fileURLToPath } from "node:url";
import { readFile, readdir } from "node:fs/promises";
import { existsSync } from "node:fs";

import { HerdrClient, EXPECTED_PROTOCOL } from "../herdr-client.js";
import { parseRoleFile, roleNameFromFilename, roleToPiArgs } from "../role-parser.js";
import { createPane, startAgent, waitForAgent, promptAgent, getAgent, reportPaneMetadata, buildWorkerEnv } from "../pane-manager.js";
import { collectSessionResult } from "../session-collector.js";

const __dirname = dirname(fileURLToPath(import.meta.url));

async function discoverRoles(cwd) {
  const roles = new Map();
  const agentsDir = resolve(cwd, ".pi/agents");
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

async function main() {
  // Fail closed outside Herdr.
  if (!HerdrClient.isInHerdrPane()) {
    console.error("ERROR: Not inside a Herdr pane (HERDR_ENV/HERDR_SOCKET_PATH/HERDR_PANE_ID not set).");
    process.exit(1);
  }

  const herdrEnv = HerdrClient.getHerdrEnv();
  if (!herdrEnv) {
    console.error("ERROR: Could not read Herdr environment.");
    process.exit(1);
  }

  const { socketPath, paneId: leadPaneId } = herdrEnv;
  const cwd = process.cwd();

  console.log("=== Balaur Herdr Agent Bridge — Live Smoke Test ===");
  console.log(`Socket: ${socketPath}`);
  console.log(`Lead pane: ${leadPaneId}`);
  console.log();

  // 1. Create client and ping.
  const client = new HerdrClient({ socketPath, timeoutMs: 10000 });
  console.log("Step 1: Pinging Herdr server...");
  let ping;
  try {
    ping = await client.ping();
  } catch (err) {
    console.error(`  FAIL: Could not ping Herdr: ${err.message}`);
    process.exit(1);
  }
  console.log(`  OK: Herdr ${ping.version}, protocol ${ping.protocol}`);

  if (ping.protocol !== EXPECTED_PROTOCOL) {
    console.error(`  FAIL: Protocol mismatch: expected ${EXPECTED_PROTOCOL}, got ${ping.protocol}`);
    process.exit(1);
  }
  console.log(`  OK: Protocol matches expected version ${EXPECTED_PROTOCOL}`);

  // 2. Discover roles.
  console.log("\nStep 2: Discovering roles...");
  const roles = await discoverRoles(cwd);
  if (roles.size === 0) {
    console.error("  FAIL: No valid roles found in .pi/agents/*.md");
    process.exit(1);
  }
  console.log(`  OK: Found ${roles.size} role(s): ${[...roles.keys()].join(", ")}`);

  // Pick a role — prefer "executor", otherwise the first.
  let roleName = "executor";
  if (!roles.has(roleName)) {
    roleName = [...roles.keys()][0];
  }
  const roleEntry = roles.get(roleName);
  console.log(`  Using role: ${roleName}`);

  // 3. Create a visible pane.
  console.log("\nStep 3: Creating visible worker pane...");
  const workerEnv = buildWorkerEnv();
  let pane;
  try {
    pane = await createPane(client, {
      currentPaneId: leadPaneId,
      direction: "right",
      ratio: 0.5,
      cwd,
      env: workerEnv,
    });
  } catch (err) {
    console.error(`  FAIL: Could not create pane: ${err.message}`);
    process.exit(1);
  }
  console.log(`  OK: Created pane ${pane.pane_id} (workspace ${pane.workspace_id})`);

  // 4. Start an interactive Pi agent.
  console.log("\nStep 4: Starting interactive Pi agent...");
  let started;
  try {
    started = await startAgent(client, {
      paneId: pane.pane_id,
      agentName: roleName,
      role: roleEntry.role,
      cwd,
    });
  } catch (err) {
    console.error(`  FAIL: Could not start agent: ${err.message}`);
    process.exit(1);
  }
  console.log(`  OK: Started agent '${started.agent_name}' in pane ${started.pane_id}`);

  // Publish metadata.
  await reportPaneMetadata(client, pane.pane_id, {
    "balaur.role": roleName,
    "balaur.bridge": "herdr-agent",
    "balaur.smoke": "1",
  }).catch(() => {});

  // 5. Prompt the worker with a harmless task.
  console.log("\nStep 5: Prompting worker...");
  const promptText = "Reply with exactly: SMOKE_TEST_OK";
  try {
    await promptAgent(client, {
      target: started.agent_name,
      text: promptText,
      wait: false,
    });
  } catch (err) {
    console.error(`  FAIL: Could not prompt agent: ${err.message}`);
    process.exit(1);
  }
  console.log(`  OK: Sent prompt: "${promptText}"`);

  // 6. Wait for the worker to reach idle/done.
  console.log("\nStep 6: Waiting for worker to finish...");
  const waitResult = await waitForAgent(client, {
    target: started.agent_name,
    until: ["idle", "done"],
    timeoutMs: 120000,
  });

  if (waitResult.timedOut) {
    console.log(`  WARN: Worker timed out (still running). Pane ${pane.pane_id} remains open for inspection.`);
    console.log("\n=== Smoke test completed with timeout (pane left open) ===");
    console.log(`Inspect pane ${pane.pane_id} in Herdr.`);
    process.exit(0);
  }

  console.log(`  OK: Worker reached status: ${waitResult.status}`);

  // 7. Try to collect the finalized result from the session JSONL.
  console.log("\nStep 7: Collecting finalized result...");
  let sessionPath = null;
  try {
    const agent = await getAgent(client, started.agent_name);
    if (agent?.agent_session?.value) {
      sessionPath = agent.agent_session.value;
    }
  } catch (err) {
    console.log(`  WARN: Could not get agent session path: ${err.message}`);
  }

  if (sessionPath) {
    console.log(`  Session path: ${sessionPath}`);
    try {
      const result = await collectSessionResult(sessionPath);
      console.log(`  stopReason: ${result.stopReason}`);
      console.log(`  turns: ${result.turns}`);
      if (result.text) {
        const preview = result.text.length > 200 ? result.text.slice(0, 200) + "..." : result.text;
        console.log(`  text: ${preview}`);
      }
      console.log("  OK: Collected finalized result");
    } catch (err) {
      console.log(`  WARN: Could not collect session result: ${err.message}`);
    }
  } else {
    console.log("  WARN: No session path available yet (worker may still be initializing).");
  }

  // 8. Do NOT close the pane — leave it for inspection.
  console.log("\n=== Live smoke test completed successfully ===");
  console.log(`Worker pane ${pane.pane_id} remains OPEN for inspection.`);
  console.log("Close it manually in Herdr when done.");
  process.exit(0);
}

main().catch((err) => {
  console.error(`FATAL: ${err.message}`);
  process.exit(1);
});
