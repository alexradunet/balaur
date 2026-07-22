/**
 * Structured client for the Herdr 0.7.5 / protocol-17 socket API.
 *
 * Connects over a Unix socket, sends JSON-RPC-like requests, and parses
 * responses. Each request gets a unique ID and a response is matched by ID.
 *
 * @module herdr-client
 */

import net from 'node:net';
import { randomUUID } from 'node:crypto';

/** Expected protocol version for Herdr 0.7.5 */
export const EXPECTED_PROTOCOL = 17;

/** Default request timeout in milliseconds */
export const DEFAULT_TIMEOUT_MS = 5000;

/** Maximum response size in bytes to prevent memory exhaustion */
export const MAX_RESPONSE_BYTES = 1024 * 1024; // 1 MB

/**
 * @typedef {Object} HerdrRequest
 * @property {string} id
 * @property {string} method
 * @property {Record<string, unknown>} params
 */

/**
 * @typedef {Object} HerdrSuccessResponse
 * @property {string} id
 * @property {Record<string, unknown>} result
 */

/**
 * @typedef {Object} HerdrErrorResponse
 * @property {string} id
 * @property {{ code: string, message: string }} error
 */

/**
 * @typedef {Object} HerdrClientOptions
 * @property {string} socketPath - Unix socket path.
 * @property {number} [timeoutMs] - Default request timeout.
 * @property {function} [createConnection] - Injectable connection factory for testing.
 */

/**
 * Low-level Herdr socket client.
 */
export class HerdrClient {
  /** @type {string} */
  #socketPath;
  /** @type {number} */
  #timeoutMs;
  /** @type {function|null} */
  #createConnection;

  /**
   * @param {HerdrClientOptions} options
   */
  constructor(options) {
    if (!options || !options.socketPath) {
      throw new Error('HerdrClient requires socketPath');
    }
    this.#socketPath = options.socketPath;
    this.#timeoutMs = options.timeoutMs ?? DEFAULT_TIMEOUT_MS;
    this.#createConnection = options.createConnection ?? null;
  }

  /**
   * Send a request and wait for a response.
   *
   * @param {string} method
   * @param {Record<string, unknown>} params
   * @param {number} [timeoutMs]
   * @returns {Promise<HerdrSuccessResponse>}
   * @throws {Error} On timeout, connection failure, or error response.
   */
  async request(method, params = {}, timeoutMs) {
    const id = `balaur-${randomUUID()}`;
    const request = { id, method, params };
    const timeout = timeoutMs ?? this.#timeoutMs;

    return this.#sendRaw(request, timeout);
  }

  /**
   * Send a ping to check server availability and protocol version.
   *
   * @returns {Promise<{ version: string, protocol: number }>}
   */
  async ping() {
    const response = await this.request('ping', {});
    const result = response.result;
    return {
      version: /** @type {string} */ (result.version),
      protocol: /** @type {number} */ (result.protocol),
    };
  }

  /**
   * Check whether we are running inside a Herdr pane.
   *
   * @param {NodeJS.ProcessEnv} env
   * @returns {boolean}
   */
  static isInHerdrPane(env = process.env) {
    return env.HERDR_ENV === '1' && !!env.HERDR_SOCKET_PATH && !!env.HERDR_PANE_ID;
  }

  /**
   * Get the Herdr environment context from process env.
   *
   * @param {NodeJS.ProcessEnv} env
   * @returns {{ socketPath: string, paneId: string } | null}
   */
  static getHerdrEnv(env = process.env) {
    if (!HerdrClient.isInHerdrPane(env)) return null;
    return {
      socketPath: /** @type {string} */ (env.HERDR_SOCKET_PATH),
      paneId: /** @type {string} */ (env.HERDR_PANE_ID),
    };
  }

  /**
   * @param {HerdrRequest} request
   * @param {number} timeout
   * @returns {Promise<HerdrSuccessResponse>}
   */
  async #sendRaw(request, timeout) {
    const payload = JSON.stringify(request) + '\n';

    return new Promise((resolve, reject) => {
      let done = false;
      let buffer = '';
      let receivedBytes = 0;
      /** @type {ReturnType<typeof setTimeout>|undefined} */
      let timer;

      const finish = (err, result) => {
        if (done) return;
        done = true;
        if (timer) clearTimeout(timer);
        try { socket.destroy(); } catch { /* ignore */ }
        if (err) reject(err);
        else resolve(result);
      };

      const connectFn = this.#createConnection || ((cb) => {
        const s = net.createConnection(this.#socketPath, cb);
        return s;
      });

      const socket = connectFn(() => {
        socket.write(payload);
      });

      socket.on('data', (chunk) => {
        receivedBytes += chunk.length;
        if (receivedBytes > MAX_RESPONSE_BYTES) {
          finish(new Error('response exceeded maximum size'));
          return;
        }
        buffer += chunk.toString('utf-8');
        const newlineIdx = buffer.indexOf('\n');
        if (newlineIdx !== -1) {
          const line = buffer.slice(0, newlineIdx).trim();
          if (!line) return;
          try {
            const response = JSON.parse(line);
            if (response.error) {
              finish(new Error(`Herdr error [${response.error.code}]: ${response.error.message}`));
            } else {
              resolve(response);
              // Mark done via the resolve path
              done = true;
              if (timer) clearTimeout(timer);
              try { socket.destroy(); } catch { /* ignore */ }
            }
          } catch (parseErr) {
            finish(new Error(`failed to parse Herdr response: ${parseErr.message}`));
          }
        }
      });

      socket.on('error', (err) => {
        finish(new Error(`Herdr connection failed: ${err.message}`));
      });

      socket.on('end', () => {
        if (!done) {
          finish(new Error('Herdr connection closed before response'));
        }
      });

      timer = setTimeout(() => {
        finish(new Error(`Herdr request timed out after ${timeout}ms`));
      }, timeout);
      if (timer.unref) timer.unref();
    });
  }
}

/**
 * Create a HerdrClient from the current process environment.
 * Returns null if not inside a Herdr pane.
 *
 * @param {NodeJS.ProcessEnv} [env]
 * @returns {HerdrClient|null}
 */
export function createClientFromEnv(env) {
  const herdrEnv = HerdrClient.getHerdrEnv(env ?? process.env);
  if (!herdrEnv) return null;
  return new HerdrClient({ socketPath: herdrEnv.socketPath });
}
