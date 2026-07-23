import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import { readdir, readFile } from 'node:fs/promises';
import { resolve } from 'node:path';
import { filteredRoleTools, ORCHESTRATION_TOOLS, parseRoleFile, roleNameFromFilename, roleToPiArgs } from '../role-parser.js';

const agentsDir = resolve('.pi/agents');
describe('role compatibility', () => {
  it('parses all supported project roles without orchestration leakage', async () => {
    const files = (await readdir(agentsDir)).filter((file) => file.endsWith('.md'));
    assert.ok(files.length >= 1);
    for (const file of files) {
      const path = resolve(agentsDir, file);
      const content = await readFile(path, 'utf8');
      assert.equal(roleNameFromFilename(file), file.slice(0, -3));
      const role = parseRoleFile(content, path);
      const args = roleToPiArgs(role);
      assert.ok(args.includes('--system-prompt') || args.includes('--append-system-prompt'));
      for (const forbidden of ORCHESTRATION_TOOLS) assert.ok(!filteredRoleTools(role).includes(forbidden), `${file} leaked ${forbidden}`);
    }
  });

  it('rejects every unsupported role key with its path and key name', () => {
    assert.throws(() => parseRoleFile('---\ndescription: unknown\nretry_count: 3\n---\nPrompt', '/roles/unknown.md'), /\/roles\/unknown\.md: unsupported role key 'retry_count'/);
    assert.throws(() => parseRoleFile('---\ndescription: background\nrun_in_background: true\n---\nPrompt', '/roles/bg.md'), /\/roles\/bg\.md: unsupported role key 'run_in_background'/);
  });

  it('accepts installed extension tool identifiers while excluding only herdr_agent', () => {
    const role = parseRoleFile('---\ndescription: extension tools\ntools: read, ext:pi-web-access/web_search, herdr_agent\n---\nPrompt', '/roles/ext.md');
    assert.deepEqual(filteredRoleTools(role), ['read', 'ext:pi-web-access/web_search']);
  });

  it('preserves wildcard semantics with Pi exclude-tools orchestration denylist', () => {
    const role = parseRoleFile('---\ndescription: wildcard\ntools: "*"\n---\nPrompt', '/roles/wild.md');
    assert.deepEqual(filteredRoleTools(role), ['*']);
    const args = roleToPiArgs(role);
    assert.ok(!args.includes('--tools'));
    assert.deepEqual(args.slice(args.indexOf('--exclude-tools') + 1, args.indexOf('--exclude-tools') + 2), [ORCHESTRATION_TOOLS.join(',')]);
  });

  it('uses an allowlist for explicit roles while removing herdr_agent', () => {
    const role = parseRoleFile('---\ndescription: explicit\ntools: read, herdr_agent, bash\n---\nPrompt', '/roles/explicit.md');
    assert.deepEqual(roleToPiArgs(role).slice(0, 2), ['--tools', 'read,bash']);
  });

  it('denies all tools when tools are omitted or orchestration filtering empties the allowlist', () => {
    for (const content of [
      '---\ndescription: omitted\n---\nPrompt',
      '---\ndescription: one orchestrator\ntools: herdr_agent\n---\nPrompt',
    ]) {
      const args = roleToPiArgs(parseRoleFile(content, '/roles/no-tools.md'));
      assert.ok(args.includes('--no-tools'));
      assert.ok(!args.includes('--tools'));
    }
  });

  it('retains a path-specific malformed-role error', () => {
    assert.throws(() => parseRoleFile('---\ndescription: bad\ntools: ext:bad tool\n---\nPrompt', '/roles/bad.md'), /\/roles\/bad\.md: unsafe characters/);
  });

  it('requires delimiter-only opening and closing frontmatter lines', () => {
    for (const content of [
      '---trailing\ndescription: bad\n---\nPrompt',
      '----\ndescription: bad\n---\nPrompt',
      '---\ndescription: bad\n---trailing\nPrompt',
      '---\ndescription: bad\n----\nPrompt',
    ]) assert.throws(() => parseRoleFile(content, '/roles/delimiter.md'), /frontmatter delimiter|unterminated frontmatter/);
    assert.equal(parseRoleFile('\uFEFF---\r\ndescription: valid\r\n---\r\nPrompt', '/roles/crlf.md').description, 'valid');
  });
});
