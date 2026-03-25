// In-memory store for OAuth authorization codes, access tokens, refresh tokens,
// and dynamic client registrations. Entries expire based on TTL.
// Optionally persists refresh tokens and client registrations to a JSON file.

import crypto from "node:crypto";
import { readFileSync, writeFileSync, mkdirSync } from "node:fs";
import { dirname } from "node:path";

/** Default TTLs in milliseconds. */
export const TTL = Object.freeze({
  AUTHORIZATION_CODE: 5 * 60 * 1000,    // 5 minutes
  ACCESS_TOKEN: 60 * 60 * 1000,          // 1 hour
  REFRESH_TOKEN: 30 * 24 * 60 * 60 * 1000, // 30 days
});

/** Generate a cryptographically random opaque token (64 hex chars). */
export function generateToken() {
  return crypto.randomBytes(32).toString("hex");
}

/**
 * In-memory OAuth store with TTL-based expiration.
 * Each entry is stored as { value, expiresAt }.
 */
export class OAuthStore {
  /**
   * @param {string|null} [persistPath=null] - Path to JSON file for persisting refresh tokens and clients.
   *   When null, operates in memory only (backwards compatible with existing tests).
   */
  constructor(persistPath = null) {
    /** @type {Map<string, {value: object, expiresAt: number}>} */
    this.codes = new Map();
    /** @type {Map<string, {value: object, expiresAt: number}>} */
    this.accessTokens = new Map();
    /** @type {Map<string, {value: object, expiresAt: number}>} */
    this.refreshTokens = new Map();
    /** @type {Map<string, object>} client_id -> metadata (no TTL) */
    this.clients = new Map();

    this._cleanupInterval = null;
    this._persistPath = persistPath || null;
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

  // --- Authorization Codes ---

  /**
   * Store an authorization code.
   * @param {string} code
   * @param {object} data - { clientId, redirectUri, codeChallenge, codeChallengeMethod, scope, gkillSessionId }
   * @param {number} [ttl=TTL.AUTHORIZATION_CODE]
   */
  putCode(code, data, ttl = TTL.AUTHORIZATION_CODE) {
    this.codes.set(code, { value: data, expiresAt: Date.now() + ttl });
  }

  /**
   * Retrieve and delete an authorization code (one-time use).
   * Returns null if not found or expired.
   * @param {string} code
   * @returns {object|null}
   */
  getAndDeleteCode(code) {
    const entry = this.codes.get(code);
    if (!entry) return null;
    this.codes.delete(code);
    if (Date.now() > entry.expiresAt) return null;
    return entry.value;
  }

  // --- Access Tokens ---

  /**
   * Store an access token.
   * @param {string} token
   * @param {object} data - { clientId, scope, gkillSessionId, userId }
   * @param {number} [ttl=TTL.ACCESS_TOKEN]
   */
  putAccessToken(token, data, ttl = TTL.ACCESS_TOKEN) {
    this.accessTokens.set(token, { value: data, expiresAt: Date.now() + ttl });
  }

  /**
   * Retrieve an access token's data. Returns null if not found or expired.
   * @param {string} token
   * @returns {object|null}
   */
  getAccessToken(token) {
    const entry = this.accessTokens.get(token);
    if (!entry) return null;
    if (Date.now() > entry.expiresAt) {
      this.accessTokens.delete(token);
      return null;
    }
    return entry.value;
  }

  /** Delete an access token. */
  deleteAccessToken(token) {
    this.accessTokens.delete(token);
  }

  // --- Refresh Tokens ---

  /**
   * Store a refresh token.
   * @param {string} token
   * @param {object} data - { clientId, scope, gkillSessionId, userId }
   * @param {number} [ttl=TTL.REFRESH_TOKEN]
   */
  putRefreshToken(token, data, ttl = TTL.REFRESH_TOKEN) {
    this.refreshTokens.set(token, { value: data, expiresAt: Date.now() + ttl });
    this._save();
  }

  /**
   * Retrieve a refresh token's data. Returns null if not found or expired.
   * @param {string} token
   * @returns {object|null}
   */
  getRefreshToken(token) {
    const entry = this.refreshTokens.get(token);
    if (!entry) return null;
    if (Date.now() > entry.expiresAt) {
      this.refreshTokens.delete(token);
      return null;
    }
    return entry.value;
  }

  /** Delete a refresh token (e.g., on rotation). */
  deleteRefreshToken(token) {
    this.refreshTokens.delete(token);
    this._save();
  }

  // --- Client Registrations ---

  /**
   * Store a dynamically registered client.
   * @param {string} clientId
   * @param {object} metadata - { client_name, redirect_uris, ... }
   */
  putClient(clientId, metadata) {
    this.clients.set(clientId, metadata);
    this._save();
  }

  /**
   * Retrieve a registered client's metadata.
   * @param {string} clientId
   * @returns {object|null}
   */
  getClient(clientId) {
    return this.clients.get(clientId) || null;
  }

  // --- Cleanup ---

  /** Remove all expired entries from codes, accessTokens, and refreshTokens. */
  sweep() {
    const now = Date.now();
    for (const [key, entry] of this.codes) {
      if (now > entry.expiresAt) this.codes.delete(key);
    }
    for (const [key, entry] of this.accessTokens) {
      if (now > entry.expiresAt) this.accessTokens.delete(key);
    }
    for (const [key, entry] of this.refreshTokens) {
      if (now > entry.expiresAt) this.refreshTokens.delete(key);
    }
  }

  /** Return counts for debugging. */
  stats() {
    return {
      codes: this.codes.size,
      accessTokens: this.accessTokens.size,
      refreshTokens: this.refreshTokens.size,
      clients: this.clients.size,
    };
  }

  // --- Persistence ---

  /**
   * Load refresh tokens and client registrations from the persist file.
   * Silently ignores missing or invalid files. Skips expired refresh tokens.
   */
  load() {
    if (!this._persistPath) return;
    let data;
    try {
      data = JSON.parse(readFileSync(this._persistPath, "utf8"));
    } catch {
      return; // file missing or invalid JSON — start fresh
    }
    const now = Date.now();
    if (data.refreshTokens && typeof data.refreshTokens === "object") {
      for (const [token, entry] of Object.entries(data.refreshTokens)) {
        if (entry && entry.expiresAt > now) {
          this.refreshTokens.set(token, { value: entry.value, expiresAt: entry.expiresAt });
        }
      }
    }
    if (data.clients && typeof data.clients === "object") {
      for (const [clientId, metadata] of Object.entries(data.clients)) {
        this.clients.set(clientId, metadata);
      }
    }
  }

  /** Persist refresh tokens and client registrations to file. No-op if no persistPath. */
  _save() {
    if (!this._persistPath) return;
    const data = {
      refreshTokens: Object.fromEntries(this.refreshTokens),
      clients: Object.fromEntries(this.clients),
    };
    try {
      mkdirSync(dirname(this._persistPath), { recursive: true });
      writeFileSync(this._persistPath, JSON.stringify(data, null, 2), "utf8");
    } catch (err) {
      process.stderr.write(`OAuth state save error: ${err.message}\n`);
    }
  }
}
