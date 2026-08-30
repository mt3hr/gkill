/**
 * MCP access logger.
 *
 * Writes JSON-per-line entries to a log file in the same format as the gkill
 * Go backend (slog JSON handler).  The logger is zero-dependency (only
 * node:fs and node:path) and uses synchronous writes so that entries are
 * flushed immediately.
 *
 * Environment variable: MCP_LOG (default "info")
 *   none | error | warn | info | debug | trace
 *
 * ERROR entries are additionally mirrored to a separate file (gkill_mcp_error.log)
 * so that an operator grepping the error logs finds MCP failures too.  Without it
 * the only MCP log file is named "*_access.log", and nobody looking for a failure
 * opens an access log.  This mirrors the Go side, which splits per level.
 *
 * Do NOT write logs with process.stderr.write: that path ignores MCP_LOG, so
 * `MCP_LOG=none` would not silence it even though the manuals say it does.
 */

import fs from "node:fs";
import path from "node:path";

// ---------------------------------------------------------------------------
// Log levels — numeric values mirror slog conventions (lower = more verbose)
// ---------------------------------------------------------------------------

/** @type {Record<string, number>} */
const LEVELS = {
  none: 999,
  error: 50,
  warn: 40,
  info: 30,
  debug: 20,
  trace: 10,
};

const LEVEL_NAMES = {
  50: "ERROR",
  40: "WARN",
  30: "INFO",
  20: "DEBUG",
  10: "TRACE",
};

// ---------------------------------------------------------------------------
// McpAccessLog
// ---------------------------------------------------------------------------

export class McpAccessLog {
  /**
   * @param {string} logFilePath  Absolute path to the log file.
   * @param {string} [levelName]  Minimum level name (default "info").
   * @param {string} [source]     Source identifier for log entries.
   * @param {string} [errorFilePath] Absolute path for the ERROR-only mirror file.
   */
  constructor(logFilePath, levelName = "info", source = "gkill-read-server.mjs", errorFilePath = null) {
    this.path = logFilePath;
    this.minLevel = LEVELS[levelName] ?? LEVELS.info;
    this.source = source;
    this.errorPath = errorFilePath;
    /** @type {number|null} */
    this.fd = null;
    /** @type {number|null} */
    this.errorFd = null;
  }

  // -- public convenience methods ------------------------------------------

  /** @param {string} msg @param {Record<string,unknown>} [fields] */
  error(msg, fields) { this._write(50, msg, fields); }
  /** @param {string} msg @param {Record<string,unknown>} [fields] */
  warn(msg, fields) { this._write(40, msg, fields); }
  /** @param {string} msg @param {Record<string,unknown>} [fields] */
  info(msg, fields) { this._write(30, msg, fields); }
  /** @param {string} msg @param {Record<string,unknown>} [fields] */
  debug(msg, fields) { this._write(20, msg, fields); }
  /** @param {string} msg @param {Record<string,unknown>} [fields] */
  trace(msg, fields) { this._write(10, msg, fields); }

  // -- internal ------------------------------------------------------------

  /**
   * @param {number} levelNum
   * @param {string} msg
   * @param {Record<string,unknown>} [fields]
   */
  _write(levelNum, msg, fields = {}) {
    if (levelNum < this.minLevel) return;

    // Lazy open — create directory + open file on first real write.
    if (this.fd === null) {
      try {
        fs.mkdirSync(path.dirname(this.path), { recursive: true });
        this.fd = fs.openSync(this.path, "a");
      } catch (err) {
        process.stderr.write(
          `[McpAccessLog] failed to open ${this.path}: ${err.message}\n`,
        );
        return;
      }
    }

    const entry = {
      time: new Date().toISOString(),
      level: LEVEL_NAMES[levelNum] || "INFO",
      source: this.source,
      msg,
      app: "gkill_mcp",
      ...fields,
    };

    try {
      fs.writeSync(this.fd, JSON.stringify(entry) + "\n");
    } catch (err) {
      // If the write fails (e.g. disk full), warn on stderr and reset fd so
      // the next call will attempt to re-open.
      process.stderr.write(
        `[McpAccessLog] write error: ${err.message}\n`,
      );
      this.fd = null;
    }
  }

  /** Close the underlying file descriptor (if open). */
  close() {
    if (this.fd !== null) {
      try { fs.closeSync(this.fd); } catch { /* ignore */ }
      this.fd = null;
    }
  }
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/**
 * Parse and validate a log-level string.
 * Returns the canonical level name, falling back to `"info"`.
 *
 * @param {string} [str]
 * @returns {string}
 */
export function parseMcpLogLevel(str) {
  const s = (str || "").toLowerCase();
  return s in LEVELS ? s : "info";
}
