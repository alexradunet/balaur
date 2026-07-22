/**
 * Strict `.pi/agents/*.md` frontmatter and prompt parsing.
 *
 * Supports the currently used role fields: description, model, thinking,
 * tools, skills, and prompt_mode. Rejects malformed or unsafe values with
 * path-specific errors.
 *
 * @module role-parser
 */

/**
 * @typedef {Object} RoleConfig
 * @property {string} name          - Derived from filename (without .md).
 * @property {string} filePath      - Absolute path to the role file.
 * @property {string} description   - Required human-readable description.
 * @property {string} [model]       - Optional model identifier.
 * @property {string} [thinking]    - Optional thinking level.
 * @property {string[]} [tools]     - Optional tool allowlist.
 * @property {string[]} [skills]    - Optional skill allowlist.
 * @property {string} [prompt_mode] - Optional prompt mode.
 * @property {string} prompt        - The Markdown body (system prompt).
 */

const ALLOWED_THINKING = new Set(['off', 'low', 'medium', 'high', 'max']);
const ALLOWED_PROMPT_MODE = new Set(['replace', 'append']);
const TOOL_WILDCARD = '*';

/**
 * Characters forbidden in model identifiers, tool names, and skill names.
 * We reject shell metacharacters and null bytes to prevent injection.
 */
const UNSAFE_CHARS = /[\x00`$\\;|&<>(){}!\n\r]/;

/**
 * Parse a `.pi/agents/*.md` file content into a RoleConfig.
 *
 * @param {string} content  - Raw file content.
 * @param {string} filePath - Absolute path (used in error messages).
 * @returns {RoleConfig}
 * @throws {Error} On malformed or unsafe values.
 */
export function parseRoleFile(content, filePath) {
  if (typeof content !== 'string' || content.length === 0) {
    throw new Error(`${filePath}: empty file`);
  }

  const { frontmatter, body } = splitFrontmatter(content, filePath);
  return buildRoleConfig(frontmatter, body, filePath);
}

/**
 * Derive the role name from a filename.
 * @param {string} filename - e.g. "implementer.md"
 * @returns {string} e.g. "implementer"
 */
export function roleNameFromFilename(filename) {
  if (!filename.endsWith('.md')) {
    throw new Error(`not a .md file: ${filename}`);
  }
  const name = filename.slice(0, -3);
  if (!name || !/^[a-z0-9][a-z0-9_-]*$/.test(name)) {
    throw new Error(`invalid role filename: ${filename}`);
  }
  return name;
}

/**
 * Split YAML frontmatter from Markdown body.
 * @param {string} content
 * @param {string} filePath
 * @returns {{ frontmatter: Record<string, string>, body: string }}
 */
function splitFrontmatter(content, filePath) {
  // Normalize BOM
  const text = content.replace(/^\uFEFF/, '');

  if (!text.startsWith('---')) {
    throw new Error(`${filePath}: missing frontmatter delimiter`);
  }

  const endIdx = text.indexOf('\n---', 3);
  if (endIdx === -1) {
    throw new Error(`${filePath}: unterminated frontmatter`);
  }

  const fmBlock = text.slice(3, endIdx).trim();
  const bodyStart = endIdx + 4; // skip \n---
  const body = text.slice(bodyStart).replace(/^\r?\n/, '');

  const frontmatter = {};
  for (const rawLine of fmBlock.split('\n')) {
    const line = rawLine.trim();
    if (!line || line.startsWith('#')) continue;
    const colonIdx = line.indexOf(':');
    if (colonIdx === -1) {
      throw new Error(`${filePath}: malformed frontmatter line: ${line}`);
    }
    const key = line.slice(0, colonIdx).trim();
    let value = line.slice(colonIdx + 1).trim();
    // Strip surrounding quotes
    if (
      (value.startsWith('"') && value.endsWith('"')) ||
      (value.startsWith("'") && value.endsWith("'"))
    ) {
      value = value.slice(1, -1);
    }
    frontmatter[key] = value;
  }

  return { frontmatter, body };
}

/**
 * Build a validated RoleConfig from parsed frontmatter and body.
 */
function buildRoleConfig(fm, body, filePath) {
  // description is required
  if (!fm.description || typeof fm.description !== 'string') {
    throw new Error(`${filePath}: missing or empty 'description'`);
  }

  const config = {
    description: fm.description,
    prompt: body,
  };

  // model (optional)
  if (fm.model !== undefined && fm.model !== '') {
    assertSafeIdentifier(fm.model, 'model', filePath);
    config.model = fm.model;
  }

  // thinking (optional)
  if (fm.thinking !== undefined && fm.thinking !== '') {
    if (!ALLOWED_THINKING.has(fm.thinking)) {
      throw new Error(
        `${filePath}: invalid thinking level '${fm.thinking}'; allowed: ${[...ALLOWED_THINKING].join(', ')}`
      );
    }
    config.thinking = fm.thinking;
  }

  // tools (optional, comma-separated or wildcard)
  if (fm.tools !== undefined && fm.tools !== '') {
    if (fm.tools.trim() === TOOL_WILDCARD) {
      config.tools = [TOOL_WILDCARD];
    } else {
      config.tools = fm.tools
        .split(',')
        .map((t) => t.trim())
        .filter(Boolean);
      for (const tool of config.tools) {
        assertSafeIdentifier(tool, 'tools', filePath);
      }
    }
  }

  // skills (optional, comma-separated)
  if (fm.skills !== undefined && fm.skills !== '') {
    config.skills = fm.skills
      .split(',')
      .map((s) => s.trim())
      .filter(Boolean);
    for (const skill of config.skills) {
      assertSafeIdentifier(skill, 'skills', filePath);
    }
  }

  // prompt_mode (optional)
  if (fm.prompt_mode !== undefined && fm.prompt_mode !== '') {
    if (!ALLOWED_PROMPT_MODE.has(fm.prompt_mode)) {
      throw new Error(
        `${filePath}: invalid prompt_mode '${fm.prompt_mode}'; allowed: ${[...ALLOWED_PROMPT_MODE].join(', ')}`
      );
    }
    config.prompt_mode = fm.prompt_mode;
  }

  return config;
}

/**
 * Reject values containing shell metacharacters or null bytes.
 */
function assertSafeIdentifier(value, field, filePath) {
  if (UNSAFE_CHARS.test(value)) {
    throw new Error(`${filePath}: unsafe characters in '${field}': ${value}`);
  }
}

/**
 * Build pi CLI args from a RoleConfig.
 * Returns an argv array (no shell concatenation).
 *
 * @param {RoleConfig} role
 * @returns {string[]}
 */
export function roleToPiArgs(role) {
  const args = [];
  if (role.model) {
    args.push('--model', role.model);
  }
  if (role.thinking) {
    args.push('--thinking', role.thinking);
  }
  if (role.tools && role.tools.length > 0 && role.tools[0] !== TOOL_WILDCARD) {
    args.push('--tools', role.tools.join(','));
  }
  return args;
}
