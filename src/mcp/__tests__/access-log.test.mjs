/**
 * Tests for McpAccessLog from lib/access-log.mjs.
 */

import { describe, test, expect, beforeEach, afterEach } from "vitest";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import crypto from "node:crypto";
import { McpAccessLog, parseMcpLogLevel } from "../lib/access-log.mjs";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function tmpLogPath() {
  return path.join(os.tmpdir(), `gkill-mcp-test-${crypto.randomUUID()}.log`);
}

function readLines(filePath) {
  return fs.readFileSync(filePath, "utf8").trim().split("\n").map((l) => JSON.parse(l));
}

// ---------------------------------------------------------------------------
// parseMcpLogLevel
// ---------------------------------------------------------------------------

describe("parseMcpLogLevel", () => {
  test("returns known levels as-is", () => {
    expect(parseMcpLogLevel("none")).toBe("none");
    expect(parseMcpLogLevel("error")).toBe("error");
    expect(parseMcpLogLevel("warn")).toBe("warn");
    expect(parseMcpLogLevel("info")).toBe("info");
    expect(parseMcpLogLevel("debug")).toBe("debug");
    expect(parseMcpLogLevel("trace")).toBe("trace");
  });

  test("is case-insensitive", () => {
    expect(parseMcpLogLevel("INFO")).toBe("info");
    expect(parseMcpLogLevel("WARN")).toBe("warn");
  });

  test("falls back to info for unknown values", () => {
    expect(parseMcpLogLevel("unknown")).toBe("info");
    expect(parseMcpLogLevel("")).toBe("info");
    expect(parseMcpLogLevel(undefined)).toBe("info");
    expect(parseMcpLogLevel(null)).toBe("info");
  });
});

// ---------------------------------------------------------------------------
// McpAccessLog — JSON format
// ---------------------------------------------------------------------------

describe("McpAccessLog JSON format", () => {
  let logPath;
  let logger;

  beforeEach(() => {
    logPath = tmpLogPath();
    logger = new McpAccessLog(logPath, "trace"); // lowest level
  });

  afterEach(() => {
    logger.close();
    try { fs.unlinkSync(logPath); } catch { /* ignore */ }
  });

  test("writes valid JSON with required fields", () => {
    logger.info("test_msg", { event: "test", key: "value" });
    const lines = readLines(logPath);
    expect(lines).toHaveLength(1);
    const entry = lines[0];
    expect(entry.time).toBeDefined();
    expect(entry.level).toBe("INFO");
    expect(entry.source).toBe("gkill-read-server.mjs");
    expect(entry.msg).toBe("test_msg");
    expect(entry.app).toBe("gkill_mcp");
    expect(entry.event).toBe("test");
    expect(entry.key).toBe("value");
  });

  test("time field is ISO 8601", () => {
    logger.info("ts_test");
    const entry = readLines(logPath)[0];
    expect(() => new Date(entry.time)).not.toThrow();
    expect(new Date(entry.time).toISOString()).toBe(entry.time);
  });

  test("writes correct level names", () => {
    logger.error("e");
    logger.warn("w");
    logger.info("i");
    logger.debug("d");
    logger.trace("t");
    const lines = readLines(logPath);
    expect(lines.map((l) => l.level)).toEqual(["ERROR", "WARN", "INFO", "DEBUG", "TRACE"]);
  });
});

// ---------------------------------------------------------------------------
// McpAccessLog — level filtering
// ---------------------------------------------------------------------------

describe("McpAccessLog level filtering", () => {
  let logPath;

  afterEach(() => {
    try { fs.unlinkSync(logPath); } catch { /* ignore */ }
  });

  test("info level filters out debug and trace", () => {
    logPath = tmpLogPath();
    const logger = new McpAccessLog(logPath, "info");
    logger.trace("t");
    logger.debug("d");
    logger.info("i");
    logger.warn("w");
    logger.error("e");
    logger.close();
    const lines = readLines(logPath);
    expect(lines.map((l) => l.msg)).toEqual(["i", "w", "e"]);
  });

  test("warn level filters out info, debug, trace", () => {
    logPath = tmpLogPath();
    const logger = new McpAccessLog(logPath, "warn");
    logger.trace("t");
    logger.debug("d");
    logger.info("i");
    logger.warn("w");
    logger.error("e");
    logger.close();
    const lines = readLines(logPath);
    expect(lines.map((l) => l.msg)).toEqual(["w", "e"]);
  });

  test("error level only includes error", () => {
    logPath = tmpLogPath();
    const logger = new McpAccessLog(logPath, "error");
    logger.trace("t");
    logger.debug("d");
    logger.info("i");
    logger.warn("w");
    logger.error("e");
    logger.close();
    const lines = readLines(logPath);
    expect(lines.map((l) => l.msg)).toEqual(["e"]);
  });
});

// ---------------------------------------------------------------------------
// McpAccessLog — none level (no file created)
// ---------------------------------------------------------------------------

describe("McpAccessLog none level", () => {
  test("does not create log file when level is none", () => {
    const logPath = tmpLogPath();
    const logger = new McpAccessLog(logPath, "none");
    logger.info("should_not_appear");
    logger.error("should_not_appear_either");
    logger.close();
    expect(fs.existsSync(logPath)).toBe(false);
  });
});

// ---------------------------------------------------------------------------
// McpAccessLog — lazy open
// ---------------------------------------------------------------------------

describe("McpAccessLog lazy open", () => {
  test("creates directory on first write", () => {
    const dir = path.join(os.tmpdir(), `gkill-mcp-test-dir-${crypto.randomUUID()}`);
    const logPath = path.join(dir, "sub", "access.log");
    const logger = new McpAccessLog(logPath, "info");

    // Directory does not exist yet
    expect(fs.existsSync(dir)).toBe(false);

    logger.info("first_write");
    logger.close();

    // Now it should exist
    expect(fs.existsSync(logPath)).toBe(true);
    const lines = readLines(logPath);
    expect(lines).toHaveLength(1);
    expect(lines[0].msg).toBe("first_write");

    // Cleanup
    fs.rmSync(dir, { recursive: true, force: true });
  });
});

// ---------------------------------------------------------------------------
// McpAccessLog — close
// ---------------------------------------------------------------------------

describe("McpAccessLog close", () => {
  let logPath;

  afterEach(() => {
    try { fs.unlinkSync(logPath); } catch { /* ignore */ }
  });

  test("close releases fd and allows re-open", () => {
    logPath = tmpLogPath();
    const logger = new McpAccessLog(logPath, "info");
    logger.info("before_close");
    logger.close();

    expect(logger.fd).toBeNull();

    // Writing after close should lazy-reopen
    logger.info("after_close");
    logger.close();

    const lines = readLines(logPath);
    expect(lines).toHaveLength(2);
    expect(lines[0].msg).toBe("before_close");
    expect(lines[1].msg).toBe("after_close");
  });

  test("close is safe to call multiple times", () => {
    logPath = tmpLogPath();
    const logger = new McpAccessLog(logPath, "none");
    logger.close();
    logger.close();
    logger.close();
    // No error thrown
  });
});

// ---------------------------------------------------------------------------
// McpAccessLog — context fields
// ---------------------------------------------------------------------------

describe("McpAccessLog context fields", () => {
  let logPath;

  afterEach(() => {
    try { fs.unlinkSync(logPath); } catch { /* ignore */ }
  });

  test("includes all context fields in output", () => {
    logPath = tmpLogPath();
    const logger = new McpAccessLog(logPath, "info");
    logger.info("auth_success", {
      event: "auth_success",
      user_id: "admin",
      remote_addr: "192.168.1.10",
    });
    logger.close();

    const entry = readLines(logPath)[0];
    expect(entry.event).toBe("auth_success");
    expect(entry.user_id).toBe("admin");
    expect(entry.remote_addr).toBe("192.168.1.10");
  });
});
