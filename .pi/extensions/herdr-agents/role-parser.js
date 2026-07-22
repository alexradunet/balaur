/** Strict `.pi/agents/*.md` role parsing and safe Pi launch preparation. */

const ALLOWED_THINKING = new Set(['off', 'minimal', 'low', 'medium', 'high', 'xhigh', 'max']);
const ALLOWED_PROMPT_MODE = new Set(['replace', 'append']);
const SUPPORTED_ROLE_KEYS = new Set(['description', 'model', 'thinking', 'tools', 'skills', 'prompt_mode']);
const TOOL_WILDCARD = '*';
export const ORCHESTRATION_TOOLS = Object.freeze([
  'herdr_agent',
  'balaur_workflow',
  'Agent',
  'get_subagent_result',
  'steer_subagent',
  'ext:pi-subagents/Agent',
]);
const EXCLUDED_TOOLS = new Set(ORCHESTRATION_TOOLS);
// A wildcard must retain Pi's normal wildcard semantics. Pi's --exclude-tools
// supports built-in, extension, and custom IDs, so deny orchestration there
// rather than silently shrinking a role's requested tool surface.
export const SAFE_WILDCARD_TOOLS = Object.freeze([]);
const UNSAFE_CHARS = /[\x00`$\\;|&<>(){}!\n\r]/;

export function parseRoleFile(content, filePath) {
  if (typeof content !== 'string' || content.length === 0) throw new Error(`${filePath}: empty file`);
  const { frontmatter, body } = splitFrontmatter(content, filePath);
  return buildRoleConfig(frontmatter, body, filePath);
}

export function roleNameFromFilename(filename) {
  if (!filename.endsWith('.md')) throw new Error(`not a .md file: ${filename}`);
  const name = filename.slice(0, -3);
  if (!name || !/^[a-z0-9][a-z0-9_-]*$/.test(name)) throw new Error(`invalid role filename: ${filename}`);
  return name;
}

function splitFrontmatter(content, filePath) {
  const text = content.replace(/^\uFEFF/, '');
  const lines = text.split('\n');
  const delimiter = (line) => line.replace(/\r$/, '') === '---';
  if (!delimiter(lines[0] || '')) throw new Error(`${filePath}: missing frontmatter delimiter`);
  const closingLine = lines.findIndex((line, index) => index > 0 && delimiter(line));
  if (closingLine === -1) throw new Error(`${filePath}: unterminated frontmatter`);
  const frontmatter = {};
  for (const rawLine of lines.slice(1, closingLine)) {
    const line = rawLine.trim();
    if (!line || line.startsWith('#')) continue;
    const colonIdx = line.indexOf(':');
    if (colonIdx === -1) throw new Error(`${filePath}: malformed frontmatter line: ${line}`);
    const key = line.slice(0, colonIdx).trim();
    let value = line.slice(colonIdx + 1).trim();
    if ((value.startsWith('"') && value.endsWith('"')) || (value.startsWith("'") && value.endsWith("'"))) value = value.slice(1, -1);
    frontmatter[key] = value;
  }
  return { frontmatter, body: lines.slice(closingLine + 1).join('\n').replace(/^\r?\n/, '') };
}

function buildRoleConfig(fm, body, filePath) {
  for (const key of Object.keys(fm)) if (!SUPPORTED_ROLE_KEYS.has(key)) throw new Error(`${filePath}: unsupported role key '${key}'`);
  if (!fm.description) throw new Error(`${filePath}: missing or empty 'description'`);
  if (!body.trim()) throw new Error(`${filePath}: missing role prompt body`);
  const config = { filePath, description: fm.description, prompt: body };
  if (fm.model) { assertSafeIdentifier(fm.model, 'model', filePath); config.model = fm.model; }
  if (fm.thinking) {
    if (!ALLOWED_THINKING.has(fm.thinking)) throw new Error(`${filePath}: invalid thinking level '${fm.thinking}'; allowed: ${[...ALLOWED_THINKING].join(', ')}`);
    config.thinking = fm.thinking;
  }
  if (fm.tools) {
    if (fm.tools.trim() === TOOL_WILDCARD) config.tools = [TOOL_WILDCARD];
    else {
      config.tools = fm.tools.split(',').map((tool) => tool.trim()).filter(Boolean);
      for (const tool of config.tools) assertSafeToolName(tool, filePath);
    }
  }
  if (fm.skills) {
    config.skills = fm.skills.split(',').map((skill) => skill.trim()).filter(Boolean);
    for (const skill of config.skills) assertSafeSkillName(skill, filePath);
  }
  if (fm.prompt_mode) {
    if (!ALLOWED_PROMPT_MODE.has(fm.prompt_mode)) throw new Error(`${filePath}: invalid prompt_mode '${fm.prompt_mode}'; allowed: ${[...ALLOWED_PROMPT_MODE].join(', ')}`);
    config.prompt_mode = fm.prompt_mode;
  }
  return config;
}

function assertSafeIdentifier(value, field, filePath) {
  if (UNSAFE_CHARS.test(value) || !/^[A-Za-z0-9_.:/@+-]+$/.test(value)) throw new Error(`${filePath}: unsafe characters in '${field}': ${value}`);
}
function assertSafeToolName(value, filePath) {
  if (value !== TOOL_WILDCARD && (UNSAFE_CHARS.test(value) || !/^[A-Za-z0-9_.:/@+-]+$/.test(value))) throw new Error(`${filePath}: unsafe characters in 'tools': ${value}`);
}
function assertSafeSkillName(value, filePath) {
  if (UNSAFE_CHARS.test(value) || !/^[A-Za-z0-9_-]+$/.test(value)) throw new Error(`${filePath}: unsafe characters in 'skills': ${value}`);
}

export function filteredRoleTools(role) {
  if (role.tools?.[0] === TOOL_WILDCARD) return [TOOL_WILDCARD];
  return (role.tools || []).filter((tool) => !EXCLUDED_TOOLS.has(tool));
}

export function roleToPiArgs(role) {
  const args = [];
  if (role.model) args.push('--model', role.model);
  if (role.thinking) args.push('--thinking', role.thinking);
  const tools = filteredRoleTools(role);
  if (tools[0] === TOOL_WILDCARD) args.push('--exclude-tools', ORCHESTRATION_TOOLS.join(','));
  else if (tools.length) args.push('--tools', tools.join(','));
  else args.push('--no-tools');
  // Pi changelog #287 documents that --system-prompt accepts a file path;
  // pane-manager replaces this value with a mode-0600 prompt file for
  // protocol-17-safe argv transport while retaining replace/append semantics.
  if (role.prompt) args.push(role.prompt_mode === 'append' ? '--append-system-prompt' : '--system-prompt', role.prompt);
  return args;
}
