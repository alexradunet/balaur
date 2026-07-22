/** Structured Herdr 0.7.5 / protocol-17 socket client. */
import net from 'node:net';
import { randomUUID } from 'node:crypto';

export const EXPECTED_PROTOCOL = 17;
export const DEFAULT_TIMEOUT_MS = 5000;
export const MAX_RESPONSE_BYTES = 1024 * 1024;
const MAX_REMOTE_CODE_BYTES = 128;
const MAX_REMOTE_MESSAGE_BYTES = 4000;

function boundUtf8(value, maxBytes) {
  const text = String(value);
  if (Buffer.byteLength(text) <= maxBytes) return text;
  let low = 0;
  let high = text.length;
  while (low < high) {
    const middle = Math.ceil((low + high) / 2);
    if (Buffer.byteLength(text.slice(0, middle)) <= maxBytes) low = middle;
    else high = middle - 1;
  }
  if (low > 0 && /[\uD800-\uDBFF]/.test(text[low - 1])) low--;
  return text.slice(0, low);
}

export class HerdrRemoteError extends Error {
  constructor(code, message) {
    const boundedCode = boundUtf8(code, MAX_REMOTE_CODE_BYTES);
    const boundedMessage = boundUtf8(message, MAX_REMOTE_MESSAGE_BYTES);
    super(boundedMessage);
    this.name = 'HerdrRemoteError';
    this.code = boundedCode;
  }
}

export class HerdrClient {
  #socketPath;
  #timeoutMs;
  #createConnection;

  constructor(options) {
    if (!options?.socketPath) throw new Error('HerdrClient requires socketPath');
    this.#socketPath = options.socketPath;
    this.#timeoutMs = options.timeoutMs ?? DEFAULT_TIMEOUT_MS;
    this.#createConnection = options.createConnection ?? null;
  }

  async request(method, params = {}, timeoutMs, signal) {
    if (typeof method !== 'string' || !method) throw new Error('Herdr method is required');
    return this.#sendRaw({ id: `balaur-${randomUUID()}`, method, params }, timeoutMs ?? this.#timeoutMs, signal);
  }

  async ping(signal) {
    const response = await this.request('ping', {}, undefined, signal);
    const result = expectObject(response.result, 'ping result');
    if (result.type !== 'pong') throw new Error('Herdr ping response missing type=pong');
    const version = requireString(result.version, 'Herdr ping version');
    const protocol = requireNumber(result.protocol, 'Herdr ping protocol');
    const capabilities = expectObject(result.capabilities, 'Herdr ping capabilities');
    if (typeof capabilities.live_handoff !== 'boolean' || typeof capabilities.detached_server_daemon !== 'boolean') throw new Error('Herdr ping capabilities are incomplete');
    return { version, protocol, capabilities };
  }

  static isInHerdrPane(env = process.env) {
    return env.HERDR_ENV === '1' && !!env.HERDR_SOCKET_PATH && !!env.HERDR_PANE_ID;
  }
  static getHerdrEnv(env = process.env) {
    if (!HerdrClient.isInHerdrPane(env)) return null;
    return { socketPath: env.HERDR_SOCKET_PATH, paneId: env.HERDR_PANE_ID };
  }

  async #sendRaw(request, timeout, signal) {
    const payload = `${JSON.stringify(request)}\n`;
    return new Promise((resolve, reject) => {
      let done = false;
      let buffer = '';
      let receivedBytes = 0;
      let timer;
      let socket;
      const finish = (error, result) => {
        if (done) return;
        done = true;
        if (timer) clearTimeout(timer);
        if (signal) signal.removeEventListener('abort', onAbort);
        try { socket?.destroy(); } catch { /* best effort */ }
        if (error) reject(error); else resolve(result);
      };
      const onAbort = () => finish(new Error('Herdr request aborted'));
      if (signal?.aborted) { onAbort(); return; }
      const connectFn = this.#createConnection || ((cb) => net.createConnection(this.#socketPath, cb));
      try { socket = connectFn(() => socket.write(payload)); } catch (error) { finish(new Error(`Herdr connection failed: ${error.message}`)); return; }
      socket.on('data', (chunk) => {
        receivedBytes += chunk.length;
        if (receivedBytes > MAX_RESPONSE_BYTES) return finish(new Error('response exceeded maximum size'));
        buffer += chunk.toString('utf8');
        const newline = buffer.indexOf('\n');
        if (newline === -1) return;
        const line = buffer.slice(0, newline).trim();
        if (!line) return;
        let response;
        try { response = JSON.parse(line); } catch (error) { return finish(new Error(`failed to parse Herdr response: ${error.message}`)); }
        if (!response || typeof response !== 'object' || response.id !== request.id) return finish(new Error(`Herdr response ID mismatch: expected ${request.id}, got ${response?.id ?? '(missing)'}`));
        if (response.error) {
          try {
            const error = expectObject(response.error, 'Herdr error');
            const code = requireString(error.code, 'Herdr error code');
            const message = requireString(error.message, 'Herdr error message');
            return finish(new HerdrRemoteError(code, message));
          } catch (error) { return finish(error); }
        }
        if (!Object.prototype.hasOwnProperty.call(response, 'result')) return finish(new Error('Herdr response missing result'));
        finish(null, response);
      });
      socket.on('error', (error) => finish(new Error(`Herdr connection failed: ${error.message}`)));
      socket.on('end', () => { if (!done) finish(new Error('Herdr connection closed before response')); });
      timer = setTimeout(() => finish(new Error(`Herdr request timed out after ${timeout}ms`)), timeout);
      timer.unref?.();
      signal?.addEventListener('abort', onAbort, { once: true });
    });
  }
}

export function expectObject(value, label) {
  if (!value || typeof value !== 'object' || Array.isArray(value)) throw new Error(`${label} must be an object`);
  return value;
}
export function requireString(value, label) {
  if (typeof value !== 'string' || !value) throw new Error(`${label} is missing or invalid`);
  return value;
}
export function requireNumber(value, label) {
  if (typeof value !== 'number' || !Number.isFinite(value)) throw new Error(`${label} is missing or invalid`);
  return value;
}
export function createClientFromEnv(env) {
  const herdrEnv = HerdrClient.getHerdrEnv(env ?? process.env);
  return herdrEnv ? new HerdrClient({ socketPath: herdrEnv.socketPath }) : null;
}
