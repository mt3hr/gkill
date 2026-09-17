/**
 * gkill_get_mcp_help（lib/help-topics.mjs）の検査。
 *
 * ツール一覧の説明文は要約にとどめ、本文はここへ移した（ADR-0622）。守るのは
 *   - 全 topic に本文があり、index が全 topic を列挙すること
 *   - 説明文から移した知識が本文に実在すること（移した瞬間に消えたら要約化は劣化）
 *   - 本文が名指しするツール名が実在すること（本文は tool-handlers.test.mjs の走査対象外）
 *   - 3サーバ全部に載り、gkill へ往復せずに応答すること
 */

import { describe, test, expect } from "vitest";

import {
  HELP_TOPICS,
  HELP_TOPIC_NAMES,
  HELP_INDEX_TOPIC,
  buildHelpPayload,
  listHelpTopics,
} from "../lib/help-topics.mjs";
import { READ_TOOLS } from "../lib/read-tools.mjs";
import { WRITE_TOOLS } from "../lib/write-tools.mjs";
import { PLUGIN_TOOLS } from "../lib/plugin-tools.mjs";
import { WRITE_SERVER_READ_TOOL_NAMES } from "../gkill-write-server.mjs";
import { handleReadToolCall, summarizeReadToolPayload } from "../lib/read-handlers.mjs";
import { normalizeMcpHelpArgs } from "../lib/normalization.mjs";
import { GkillApiError } from "../lib/errors.mjs";

const ALL_TOOL_NAMES = new Set([...READ_TOOLS, ...WRITE_TOOLS, ...PLUGIN_TOOLS].map((tool) => tool.name));

describe("help topics", () => {
  test("every topic has a title and a substantial body", () => {
    for (const [topic, entry] of Object.entries(HELP_TOPICS)) {
      expect(typeof entry.title, topic).toBe("string");
      expect(entry.title.length, topic).toBeGreaterThan(10);
      expect(typeof entry.text, topic).toBe("string");
      expect(entry.text.length, topic).toBeGreaterThan(300);
    }
  });

  test("HELP_TOPIC_NAMES is index plus every topic, and matches the tool's enum", () => {
    expect(HELP_TOPIC_NAMES[0]).toBe(HELP_INDEX_TOPIC);
    expect([...HELP_TOPIC_NAMES].slice(1)).toEqual(Object.keys(HELP_TOPICS));
    const tool = READ_TOOLS.find((candidate) => candidate.name === "gkill_get_mcp_help");
    expect(tool).toBeDefined();
    expect(tool.inputSchema.properties.topic.enum).toEqual([...HELP_TOPIC_NAMES]);
  });

  test("the index lists every topic", () => {
    const index = buildHelpPayload(undefined);
    expect(index.topic).toBe(HELP_INDEX_TOPIC);
    expect(index.topics).toEqual(listHelpTopics());
    for (const topic of Object.keys(HELP_TOPICS)) {
      expect(index.text).toContain(`- ${topic}: `);
    }
    expect(buildHelpPayload("index")).toEqual(index);
    expect(buildHelpPayload("")).toEqual(index);
  });

  test("a topic payload carries its title and text", () => {
    const payload = buildHelpPayload("kftl");
    expect(payload.topic).toBe("kftl");
    expect(payload.title).toBe(HELP_TOPICS.kftl.title);
    expect(payload.text).toBe(HELP_TOPICS.kftl.text);
    expect(payload).not.toHaveProperty("topics");
  });

  // 説明文から移した知識。要約化で消えていないことを、移した先で固定する。
  test("knowledge moved out of the tool descriptions lives in a topic", () => {
    const kftl = HELP_TOPICS.kftl.text;
    for (const phrase of ["~~", "??", "/endt?", "/end?", "/expense", "monthly 31 skips February", "idempotency_key", "打刻終了タイトルを指定してください"]) {
      expect(kftl, phrase).toContain(phrase);
    }
    const mi = HELP_TOPICS.mi.text;
    for (const phrase of ["mi_create", "mi_check", "for_mi", "include_create_mi", "mi_sort_type", "collapse"]) {
      expect(mi, phrase).toContain(phrase);
    }
    const search = HELP_TOPICS.search.text;
    for (const phrase of ["even when partial is false", "do not put a repository named by that warning back into query.reps", "is_include_timeis", "payload.kind"]) {
      expect(search, phrase).toContain(phrase);
    }
    const pagination = HELP_TOPICS.pagination.text;
    for (const phrase of ["remaining_count", "total_count", "count_only", "group_by", "cannot be combined"]) {
      expect(pagination, phrase).toContain(phrase);
    }
    const dataTypes = HELP_TOPICS.data_types.text;
    for (const phrase of ["timeis_start", "expand", "num_min", "create_apps", "gkill_kftl"]) {
      expect(dataTypes, phrase).toContain(phrase);
    }
    const idf = HELP_TOPICS.idf.text;
    for (const phrase of ["file_path", "file_url", "gkill_get_idf_file", "thumb", "indexed_at"]) {
      expect(idf, phrase).toContain(phrase);
    }
    const deleted = HELP_TOPICS.deleted.text;
    for (const phrase of ["include_deleted_data", "gkill_get_kyou_history", "gkill_restore_kyou", "one-second"]) {
      expect(deleted, phrase).toContain(phrase);
    }
    const rep = HELP_TOPICS.rep.text;
    for (const phrase of ["canonical_rep_types", "attached_data_reps", "use_to_write", "writable_only", "directory"]) {
      expect(rep, phrase).toContain(phrase);
    }
    const plugin = HELP_TOPICS.plugin.text;
    for (const phrase of ["include_plugin_content", "content_status", "plugin_content_max_text_length", "gkill_get_plugin_list"]) {
      expect(plugin, phrase).toContain(phrase);
    }
  });

  // 本文は説明文の走査（tool-handlers.test.mjs）の対象外なので、綴り違いのツール名はここで落とす。
  test("tool names mentioned in the bodies exist on some server", () => {
    for (const [topic, entry] of Object.entries(HELP_TOPICS)) {
      for (const match of entry.text.matchAll(/gkill_[a-z_]+/g)) {
        const name = match[0];
        // create_app の値（gkill_kftl / gkill_mcp_readwrite / gkill_mcp_write / gkill_wear）は素通し
        if (["gkill_kftl", "gkill_mcp_readwrite", "gkill_mcp_write", "gkill_wear"].includes(name)) continue;
        expect(ALL_TOOL_NAMES.has(name), `${topic}: ${name}`).toBe(true);
      }
    }
  });
});

describe("gkill_get_mcp_help tool", () => {
  test("is carried by all three servers", () => {
    expect(READ_TOOLS.some((tool) => tool.name === "gkill_get_mcp_help")).toBe(true);
    expect(WRITE_SERVER_READ_TOOL_NAMES.has("gkill_get_mcp_help")).toBe(true);
  });

  test("normalizeMcpHelpArgs accepts a known topic, index by omission, and rejects the rest", () => {
    expect(normalizeMcpHelpArgs({})).toEqual({});
    expect(normalizeMcpHelpArgs(undefined)).toEqual({});
    expect(normalizeMcpHelpArgs({ topic: "mi" })).toEqual({ topic: "mi" });
    expect(normalizeMcpHelpArgs({ topic: "index" })).toEqual({ topic: "index" });
    expect(() => normalizeMcpHelpArgs({ topic: "tasks" })).toThrow(/must be one of/);
    expect(() => normalizeMcpHelpArgs({ topic: 3 })).toThrow(GkillApiError);
    expect(() => normalizeMcpHelpArgs({ subject: "mi" })).toThrow(/is not supported/);
  });

  test("dispatch returns the topic without calling gkill", async () => {
    let calls = 0;
    const ctx = {
      client: {
        callApi: async () => {
          calls++;
          return {};
        },
      },
      sid: "sid-1",
      isLocalTransport: true,
    };
    const payload = await handleReadToolCall(ctx, "gkill_get_mcp_help", { topic: "pagination" });
    expect(calls).toBe(0);
    expect(payload.topic).toBe("pagination");
    expect(payload.text).toBe(HELP_TOPICS.pagination.text);
    const index = await handleReadToolCall(ctx, "gkill_get_mcp_help", {});
    expect(index.topic).toBe("index");
    expect(index.topics.length).toBe(Object.keys(HELP_TOPICS).length);
  });

  test("summary names the topic", () => {
    const summary = summarizeReadToolPayload("gkill_get_mcp_help", buildHelpPayload("kftl"));
    expect(summary).toContain('Help topic "kftl"');
    expect(summary).toContain(HELP_TOPICS.kftl.title);
  });
});
