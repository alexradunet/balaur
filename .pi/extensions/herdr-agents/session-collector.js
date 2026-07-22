/**
 * Structured result collection from finalized Pi session JSONL.
 *
 * Parses the JSONL session file produced by Pi and extracts the authoritative
 * finalized assistant result with tool/usage evidence. Ignores partial
 * assistant output and intermediate tool results.
 *
 * @module session-collector
 */

import { createReadStream } from 'node:fs';
import { createInterface } from 'node:readline';

/** Maximum JSONL file size to parse (10 MB) */
export const MAX_SESSION_BYTES = 10 * 1024 * 1024;

/** Maximum number of lines to parse */
export const MAX_SESSION_LINES = 100_000;

/**
 * @typedef {Object} SessionResult
 * @property {string} text           - The finalized assistant text output.
 * @property {string} stopReason     - Why the session ended (end, error, aborted, etc.)
 * @property {Object} [usage]        - Token usage statistics.
 * @property {string} [model]        - Model used.
 * @property {ToolCallEvidence[]} toolCalls - Tool calls made during the session.
 * @property {number} turns          - Number of assistant turns.
 */

/**
 * @typedef {Object} ToolCallEvidence
 * @property {string} name       - Tool name.
 * @property {Object} arguments  - Tool arguments.
 * @property {Object} [result]   - Tool result summary.
 */

/**
 * Parse a Pi session JSONL file and extract the finalized result.
 *
 * @param {string} filePath - Path to the JSONL session file.
 * @returns {Promise<SessionResult>}
 */
export async function collectSessionResult(filePath) {
  const entries = await readJsonlEntries(filePath);
  return extractFinalizedResult(entries);
}

/**
 * Parse JSONL entries from a file, respecting size and line limits.
 *
 * @param {string} filePath
 * @returns {Promise<Object[]>}
 */
export async function readJsonlEntries(filePath) {
  const entries = [];
  let totalBytes = 0;
  let lineCount = 0;

  return new Promise((resolve, reject) => {
    const stream = createReadStream(filePath, { encoding: 'utf-8' });
    let buffer = '';

    stream.on('data', (chunk) => {
      totalBytes += Buffer.byteLength(chunk);
      if (totalBytes > MAX_SESSION_BYTES) {
        stream.destroy();
        reject(new Error(`session file exceeds maximum size (${MAX_SESSION_BYTES} bytes)`));
        return;
      }

      buffer += chunk;
      const lines = buffer.split('\n');
      buffer = lines.pop() || '';

      for (const line of lines) {
        if (!line.trim()) continue;
        lineCount++;
        if (lineCount > MAX_SESSION_LINES) {
          stream.destroy();
          reject(new Error(`session file exceeds maximum lines (${MAX_SESSION_LINES})`));
          return;
        }
        try {
          entries.push(JSON.parse(line));
        } catch {
          // Skip malformed lines
        }
      }
    });

    stream.on('end', () => {
      if (buffer.trim()) {
        lineCount++;
        try {
          entries.push(JSON.parse(buffer.trim()));
        } catch {
          // Skip trailing malformed line
        }
      }
      resolve(entries);
    });

    stream.on('error', (err) => {
      reject(new Error(`failed to read session file: ${err.message}`));
    });
  });
}

/**
 * Extract the finalized assistant result from parsed JSONL entries.
 *
 * Pi JSONL session files contain entries with types like:
 * - { type: "message", message: { role: "assistant", content: [...], ... } }
 * - { type: "message", message: { role: "toolResult", ... } }
 * - { type: "message_end", message: { ... } }
 *
 * We want the LAST finalized assistant message with stopReason !== "error"
 * and collect tool call evidence from the session.
 *
 * @param {Object[]} entries
 * @returns {SessionResult}
 */
export function extractFinalizedResult(entries) {
  if (!Array.isArray(entries) || entries.length === 0) {
    return { text: '', stopReason: 'unknown', toolCalls: [], turns: 0 };
  }

  let lastAssistantText = '';
  let stopReason = 'unknown';
  let model;
  let usage = null;
  let turns = 0;
  /** @type {ToolCallEvidence[]} */
  const toolCalls = [];
  let hasFinalizedAssistant = false;

  for (const entry of entries) {
    if (!entry || typeof entry !== 'object') continue;

    // Handle message_end events (finalized assistant messages)
    if (entry.type === 'message_end' && entry.message) {
      const msg = entry.message;
      if (msg.role === 'assistant') {
        turns++;
        // Extract text from content
        const textParts = [];
        const calls = [];
        if (Array.isArray(msg.content)) {
          for (const part of msg.content) {
            if (part.type === 'text' && part.text) {
              textParts.push(part.text);
            } else if (part.type === 'toolCall') {
              calls.push({
                name: part.name || 'unknown',
                arguments: part.arguments || {},
              });
            }
          }
        }
        if (textParts.length > 0) {
          lastAssistantText = textParts.join('\n');
        }
        toolCalls.push(...calls);
        if (msg.stopReason) stopReason = msg.stopReason;
        if (msg.model) model = msg.model;
        if (msg.usage) {
          usage = { ...msg.usage };
        }
        hasFinalizedAssistant = true;
      }
      continue;
    }

    // Handle regular message entries
    if (entry.type === 'message' && entry.message) {
      const msg = entry.message;
      if (msg.role === 'assistant') {
        // Only count as a turn if it has a stopReason (finalized)
        if (msg.stopReason) {
          turns++;
          const textParts = [];
          if (Array.isArray(msg.content)) {
            for (const part of msg.content) {
              if (part.type === 'text' && part.text) {
                textParts.push(part.text);
              }
            }
          }
          if (textParts.length > 0) {
            lastAssistantText = textParts.join('\n');
          }
          stopReason = msg.stopReason;
          if (msg.model) model = msg.model;
          if (msg.usage) usage = { ...msg.usage };
          hasFinalizedAssistant = true;
        }
      } else if (msg.role === 'toolResult') {
        // Attach result to the most recent matching tool call
        const toolName = msg.toolName;
        if (toolName) {
          const matchingCall = [...toolCalls].reverse().find((c) => c.name === toolName && !c.result);
          if (matchingCall) {
            matchingCall.result = summarizeToolResult(msg);
          }
        }
      }
    }
  }

  if (!hasFinalizedAssistant) {
    return { text: '', stopReason: 'incomplete', toolCalls, turns: 0 };
  }

  return {
    text: lastAssistantText,
    stopReason,
    model,
    usage,
    toolCalls,
    turns,
  };
}

/**
 * Summarize a tool result message for evidence.
 * @param {Object} msg
 * @returns {Object}
 */
function summarizeToolResult(msg) {
  const summary = { isError: !!msg.isError };
  if (Array.isArray(msg.content)) {
    const textParts = msg.content
      .filter((p) => p.type === 'text' && p.text)
      .map((p) => p.text);
    if (textParts.length > 0) {
      const combined = textParts.join('\n');
      // Truncate for evidence summary
      summary.text = combined.length > 500 ? combined.slice(0, 500) + '...' : combined;
    }
  }
  return summary;
}
