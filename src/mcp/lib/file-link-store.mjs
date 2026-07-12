// In-memory store for short-lived file-link tokens.
//
// Remote MCP clients (e.g. a cloud AI over OAuth) cannot read a local file path,
// so instead we hand them an opaque URL `${publicBaseUrl}/files/${token}`. The
// token binds to exactly one file (rep_name + file_name) and carries the gkill
// session that was authenticated when it was minted, so the public delivery
// route can fetch the bytes without any credential appearing in the URL.
//
// The token itself is the security boundary: it is unguessable
// (crypto.randomBytes), scoped to a single file, and expires. Knowing a URL lets
// the holder fetch that one file until it expires — nothing else.

import { generateToken } from "./oauth-store.mjs";

// Default token lifetime. Remote clients fetch the URL shortly after it is
// minted, so a short window is safe; overridable for "look back at the
// conversation later" use cases.
export const FILE_LINK_TTL_MS = Math.max(
  1000,
  Number(process.env.GKILL_MCP_FILE_LINK_TTL_MS) || 60 * 60 * 1000,
);

/**
 * In-memory file-link store with TTL-based expiration.
 * Each entry is stored as { value, expiresAt }.
 */
export class FileLinkStore {
  /**
   * @param {number} [ttlMs=FILE_LINK_TTL_MS]
   */
  constructor(ttlMs = FILE_LINK_TTL_MS) {
    /** @type {Map<string, {value: object, expiresAt: number}>} */
    this.links = new Map();
    this.ttlMs = ttlMs;
    this._cleanupInterval = null;
  }

  /** Start periodic cleanup of expired entries (default: every 5 minutes). */
  startCleanup(intervalMs = 5 * 60 * 1000) {
    this.stopCleanup();
    this._cleanupInterval = setInterval(() => this.sweep(), intervalMs);
    if (this._cleanupInterval.unref) this._cleanupInterval.unref();
  }

  /** Stop periodic cleanup. */
  stopCleanup() {
    if (this._cleanupInterval) {
      clearInterval(this._cleanupInterval);
      this._cleanupInterval = null;
    }
  }

  /** Remove all expired entries. */
  sweep() {
    const now = Date.now();
    for (const [token, entry] of this.links) {
      if (now > entry.expiresAt) this.links.delete(token);
    }
  }

  /**
   * Mint a token for one file.
   * @param {object} data - { gkillSessionId, repName, fileName, isImage }
   * @param {number} [ttlMs=this.ttlMs]
   * @returns {string} the opaque token
   */
  mint(data, ttlMs = this.ttlMs) {
    const token = generateToken();
    this.links.set(token, { value: data, expiresAt: Date.now() + ttlMs });
    return token;
  }

  /**
   * Resolve a token to its file info. Returns null if unknown or expired
   * (expired entries are dropped on access).
   * @param {string} token
   * @returns {object|null} { gkillSessionId, repName, fileName, isImage }
   */
  resolve(token) {
    const entry = this.links.get(token);
    if (!entry) return null;
    if (Date.now() > entry.expiresAt) {
      this.links.delete(token);
      return null;
    }
    return entry.value;
  }
}
