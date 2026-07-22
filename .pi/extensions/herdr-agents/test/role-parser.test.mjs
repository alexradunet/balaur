import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { resolve, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import { parseRoleFile, roleNameFromFilename, roleToPiArgs } from '../role-parser.js';

const __dirname = dirname(fileURLToPath(import.meta.url));
const fixtures = resolve(__dirname, 'fixtures');

function readFixture(name) {
  return readFileSync(resolve(fixtures, name), 'utf-8');
}

describe('role-parser', () => {
  describe('parseRoleFile', () => {
    it('parses a valid role file', () => {
      const content = readFixture('valid-role.md');
      const role = parseRoleFile(content, '/test/valid-role.md');

      assert.equal(role.description, 'General-purpose worker agent');
      assert.equal(role.model, 'qwen-token-plan/qwen3.7-plus');
      assert.equal(role.thinking, 'high');
      assert.deepEqual(role.tools, ['read', 'bash', 'edit', 'write']);
      assert.deepEqual(role.skills, ['tdd', 'browser-check']);
      assert.equal(role.prompt_mode, 'replace');
      assert.ok(role.prompt.includes('helpful worker agent'));
    });

    it('parses wildcard tools', () => {
      const content = readFixture('wildcard-tools.md');
      const role = parseRoleFile(content, '/test/wildcard-tools.md');
      assert.deepEqual(role.tools, ['*']);
    });

    it('rejects empty content', () => {
      assert.throws(() => parseRoleFile('', '/test/empty.md'), /empty file/);
    });

    it('rejects missing frontmatter delimiter', () => {
      const content = readFixture('malformed-no-frontmatter.md');
      assert.throws(
        () => parseRoleFile(content, '/test/bad.md'),
        /missing frontmatter delimiter/
      );
    });

    it('rejects unterminated frontmatter', () => {
      const content = readFixture('malformed-unterminated.md');
      assert.throws(
        () => parseRoleFile(content, '/test/bad.md'),
        /unterminated frontmatter/
      );
    });

    it('rejects unsafe model with shell metacharacters', () => {
      const content = readFixture('malformed-unsafe.md');
      assert.throws(
        () => parseRoleFile(content, '/test/bad.md'),
        /unsafe characters.*model/
      );
    });

    it('rejects invalid thinking level', () => {
      const content = readFixture('malformed-thinking.md');
      assert.throws(
        () => parseRoleFile(content, '/test/bad.md'),
        /invalid thinking level/
      );
    });

    it('rejects missing description', () => {
      const content = readFixture('malformed-no-description.md');
      assert.throws(
        () => parseRoleFile(content, '/test/bad.md'),
        /missing or empty 'description'/
      );
    });

    it('handles BOM prefix', () => {
      const content = '\uFEFF---\ndescription: BOM test\n---\nBody';
      const role = parseRoleFile(content, '/test/bom.md');
      assert.equal(role.description, 'BOM test');
    });

    it('handles quoted values', () => {
      const content = '---\ndescription: "quoted desc"\nmodel: \'some/model\'\n---\nBody';
      const role = parseRoleFile(content, '/test/quoted.md');
      assert.equal(role.description, 'quoted desc');
      assert.equal(role.model, 'some/model');
    });
  });

  describe('roleNameFromFilename', () => {
    it('extracts name from valid filename', () => {
      assert.equal(roleNameFromFilename('implementer.md'), 'implementer');
      assert.equal(roleNameFromFilename('reviewer-sol.md'), 'reviewer-sol');
    });

    it('rejects non-.md files', () => {
      assert.throws(() => roleNameFromFilename('test.txt'), /not a \.md file/);
    });

    it('rejects invalid names', () => {
      assert.throws(() => roleNameFromFilename('.md'), /invalid role filename/);
      assert.throws(() => roleNameFromFilename('UPPER.md'), /invalid role filename/);
    });
  });

  describe('roleToPiArgs', () => {
    it('builds args with model and thinking', () => {
      const role = { description: 'test', model: 'gpt-5.6-sol', thinking: 'high', prompt: '' };
      const args = roleToPiArgs(role);
      assert.deepEqual(args, ['--model', 'gpt-5.6-sol', '--thinking', 'high']);
    });

    it('builds args with specific tools', () => {
      const role = { description: 'test', tools: ['read', 'bash'], prompt: '' };
      const args = roleToPiArgs(role);
      assert.deepEqual(args, ['--tools', 'read,bash']);
    });

    it('omits tools arg for wildcard', () => {
      const role = { description: 'test', tools: ['*'], prompt: '' };
      const args = roleToPiArgs(role);
      assert.deepEqual(args, []);
    });

    it('returns empty array for minimal role', () => {
      const role = { description: 'test', prompt: '' };
      const args = roleToPiArgs(role);
      assert.deepEqual(args, []);
    });
  });
});
