import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import { resolve, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import { collectSessionResult, readJsonlEntries, extractFinalizedResult } from '../session-collector.js';

const __dirname = dirname(fileURLToPath(import.meta.url));
const fixtures = resolve(__dirname, 'fixtures');

describe('session-collector', () => {
  describe('readJsonlEntries', () => {
    it('reads valid JSONL file', async () => {
      const entries = await readJsonlEntries(resolve(fixtures, 'session.jsonl'));
      assert.ok(entries.length >= 3);
    });

    it('skips malformed lines', async () => {
      const entries = await readJsonlEntries(resolve(fixtures, 'error-session.jsonl'));
      // Should have 3 valid entries, skipping the invalid JSON line
      assert.equal(entries.length, 3);
    });

    it('rejects non-existent file', async () => {
      await assert.rejects(
        readJsonlEntries('/nonexistent/file.jsonl'),
        /failed to read session file/
      );
    });
  });

  describe('extractFinalizedResult', () => {
    it('returns empty result for empty entries', () => {
      const result = extractFinalizedResult([]);
      assert.equal(result.text, '');
      assert.equal(result.stopReason, 'unknown');
      assert.equal(result.turns, 0);
    });

    it('returns incomplete for partial session', () => {
      const entries = [
        { type: 'message', message: { role: 'assistant', content: [{ type: 'text', text: 'Working' }] } }
      ];
      const result = extractFinalizedResult(entries);
      assert.equal(result.text, '');
      assert.equal(result.stopReason, 'incomplete');
    });

    it('extracts finalized result from message_end', () => {
      const entries = [
        {
          type: 'message_end',
          message: {
            role: 'assistant',
            content: [{ type: 'text', text: 'Done!' }],
            stopReason: 'end',
            model: 'test-model',
            usage: { input: 100, output: 50 },
          },
        },
      ];
      const result = extractFinalizedResult(entries);
      assert.equal(result.text, 'Done!');
      assert.equal(result.stopReason, 'end');
      assert.equal(result.model, 'test-model');
      assert.equal(result.turns, 1);
      assert.deepEqual(result.usage, { input: 100, output: 50 });
    });

    it('collects tool call evidence', () => {
      const entries = [
        {
          type: 'message_end',
          message: {
            role: 'assistant',
            content: [
              { type: 'text', text: 'Result' },
              { type: 'toolCall', name: 'bash', arguments: { command: 'echo hi' } },
            ],
            stopReason: 'end',
          },
        },
      ];
      const result = extractFinalizedResult(entries);
      assert.equal(result.toolCalls.length, 1);
      assert.equal(result.toolCalls[0].name, 'bash');
      assert.deepEqual(result.toolCalls[0].arguments, { command: 'echo hi' });
    });

    it('returns the last finalized assistant text', () => {
      const entries = [
        {
          type: 'message_end',
          message: {
            role: 'assistant',
            content: [{ type: 'text', text: 'First' }],
            stopReason: 'end',
          },
        },
        {
          type: 'message_end',
          message: {
            role: 'assistant',
            content: [{ type: 'text', text: 'Second' }],
            stopReason: 'end',
          },
        },
      ];
      const result = extractFinalizedResult(entries);
      assert.equal(result.text, 'Second');
      assert.equal(result.turns, 2);
    });

    it('attaches tool results to matching calls', () => {
      const entries = [
        {
          type: 'message_end',
          message: {
            role: 'assistant',
            content: [
              { type: 'toolCall', name: 'bash', arguments: { command: 'ls' } },
            ],
            stopReason: 'end',
          },
        },
        {
          type: 'message',
          message: {
            role: 'toolResult',
            toolName: 'bash',
            content: [{ type: 'text', text: 'file1\nfile2' }],
          },
        },
      ];
      const result = extractFinalizedResult(entries);
      assert.equal(result.toolCalls.length, 1);
      assert.ok(result.toolCalls[0].result);
      assert.ok(result.toolCalls[0].result.text.includes('file1'));
    });

    it('handles non-array/non-object entries gracefully', () => {
      const result = extractFinalizedResult([null, undefined, 'string', 42]);
      // No finalized assistant message found among invalid entries
      assert.equal(result.stopReason, 'incomplete');
      assert.equal(result.text, '');
      assert.equal(result.turns, 0);
    });
  });

  describe('collectSessionResult (integration)', () => {
    it('collects from a valid session file', async () => {
      const result = await collectSessionResult(resolve(fixtures, 'session.jsonl'));
      assert.equal(result.stopReason, 'end');
      assert.ok(result.text.includes('Task completed'));
      assert.equal(result.model, 'test-model');
      assert.ok(result.turns >= 1);
    });

    it('collects from multi-turn session', async () => {
      const result = await collectSessionResult(resolve(fixtures, 'multi-turn-session.jsonl'));
      assert.equal(result.stopReason, 'end');
      assert.ok(result.text.includes('Final result'));
      assert.equal(result.turns, 2);
    });

    it('handles error session', async () => {
      const result = await collectSessionResult(resolve(fixtures, 'error-session.jsonl'));
      // The last finalized message has stopReason "error"
      assert.equal(result.stopReason, 'error');
    });

    it('returns incomplete for partial session', async () => {
      const result = await collectSessionResult(resolve(fixtures, 'partial-session.jsonl'));
      assert.equal(result.stopReason, 'incomplete');
      assert.equal(result.text, '');
    });
  });
});
