/**
 * Tests for lib/plugin-tools.mjs — read / write / readwrite の3サーバが共有する
 * プラグイン関連ツールの定義とハンドラ。
 */

import { describe, test, expect, vi } from "vitest";
import {
  PLUGIN_TOOLS,
  PLUGIN_TOOL_NAMES,
  GET_PLUGIN_LIST_ENDPOINT,
  GET_PLUGIN_CONTENT_HTML_ENDPOINT,
  handlePluginToolCall,
  isPluginToolName,
  summarizePluginToolPayload,
} from "../lib/plugin-tools.mjs";
import {
  DEFAULT_PLUGIN_CONTENT_MAX_TEXT_LENGTH,
  MAX_PLUGIN_CONTENT_MAX_TEXT_LENGTH,
} from "../lib/constants.mjs";
import { normalizePluginContentArgs } from "../lib/normalization.mjs";

const CONTENT_HTML =
  "<html><head><style>body{color:red}</style><script>var a=1;</script></head>" +
  "<body><div class=\"conv-title\">セッション名</div><div class=\"msg\">" +
  "<div class=\"sender\">あなた</div>プラグインの内容取得を実装したい</div></body></html>";

// ---------------------------------------------------------------------------
// Tool definitions
// ---------------------------------------------------------------------------
describe("PLUGIN_TOOLS", () => {
  test("exposes exactly the names listed in PLUGIN_TOOL_NAMES", () => {
    expect(PLUGIN_TOOLS.map((t) => t.name)).toEqual(PLUGIN_TOOL_NAMES);
  });

  test("every tool has a description and an object inputSchema", () => {
    for (const tool of PLUGIN_TOOLS) {
      expect(typeof tool.description).toBe("string");
      expect(tool.description.length).toBeGreaterThan(20);
      expect(tool.inputSchema.type).toBe("object");
      expect(tool.inputSchema.additionalProperties).toBe(false);
    }
  });

  test("gkill_get_plugin_list takes only locale_name", () => {
    const tool = PLUGIN_TOOLS.find((t) => t.name === "gkill_get_plugin_list");
    expect(Object.keys(tool.inputSchema.properties)).toEqual(["locale_name"]);
    expect(tool.inputSchema.required).toBeUndefined();
  });

  test("gkill_get_plugin_content requires rep_name and kyou_id", () => {
    const tool = PLUGIN_TOOLS.find((t) => t.name === "gkill_get_plugin_content");
    expect(tool.inputSchema.required).toEqual(["rep_name", "kyou_id"]);
    expect(tool.inputSchema.properties.format.enum).toEqual(["text", "html", "both"]);
    expect(tool.inputSchema.properties.format.default).toBe("text");
  });
});

describe("isPluginToolName", () => {
  test("recognizes plugin tools", () => {
    expect(isPluginToolName("gkill_get_plugin_list")).toBe(true);
    expect(isPluginToolName("gkill_get_plugin_content")).toBe(true);
  });

  test("rejects everything else", () => {
    expect(isPluginToolName("gkill_get_kyous")).toBe(false);
    expect(isPluginToolName("gkill_add_kmemo")).toBe(false);
    expect(isPluginToolName("")).toBe(false);
    expect(isPluginToolName(undefined)).toBe(false);
  });
});

// ---------------------------------------------------------------------------
// normalizePluginContentArgs
// ---------------------------------------------------------------------------
describe("normalizePluginContentArgs", () => {
  test("defaults format to text and applies the default max_text_length", () => {
    const normalized = normalizePluginContentArgs({ rep_name: "Claude Code", kyou_id: "abc" });
    expect(normalized).toEqual({
      rep_name: "Claude Code",
      kyou_id: "abc",
      format: "text",
      max_text_length: DEFAULT_PLUGIN_CONTENT_MAX_TEXT_LENGTH,
    });
  });

  test("trims rep_name and kyou_id", () => {
    const normalized = normalizePluginContentArgs({ rep_name: "  Claude Code  ", kyou_id: " abc " });
    expect(normalized.rep_name).toBe("Claude Code");
    expect(normalized.kyou_id).toBe("abc");
  });

  test("lower-cases the format", () => {
    expect(normalizePluginContentArgs({ rep_name: "r", kyou_id: "k", format: "HTML" }).format).toBe("html");
  });

  test("rejects an unknown format", () => {
    expect(() => normalizePluginContentArgs({ rep_name: "r", kyou_id: "k", format: "markdown" })).toThrow(
      /format/,
    );
  });

  test("rejects a missing rep_name or kyou_id", () => {
    expect(() => normalizePluginContentArgs({ kyou_id: "k" })).toThrow(/rep_name/);
    expect(() => normalizePluginContentArgs({ rep_name: "r" })).toThrow(/kyou_id/);
    expect(() => normalizePluginContentArgs({ rep_name: "", kyou_id: "k" })).toThrow(/rep_name/);
  });

  test("rejects unknown keys", () => {
    expect(() => normalizePluginContentArgs({ rep_name: "r", kyou_id: "k", plugin: "x" })).toThrow(
      /plugin/,
    );
  });

  test("rejects an out-of-range max_text_length", () => {
    expect(() => normalizePluginContentArgs({ rep_name: "r", kyou_id: "k", max_text_length: 0 })).toThrow(
      /max_text_length/,
    );
    expect(() =>
      normalizePluginContentArgs({
        rep_name: "r",
        kyou_id: "k",
        max_text_length: MAX_PLUGIN_CONTENT_MAX_TEXT_LENGTH + 1,
      }),
    ).toThrow(/max_text_length/);
  });

  test("keeps locale_name when given", () => {
    expect(normalizePluginContentArgs({ rep_name: "r", kyou_id: "k", locale_name: "en" }).locale_name).toBe(
      "en",
    );
  });
});

// ---------------------------------------------------------------------------
// handlePluginToolCall
// ---------------------------------------------------------------------------
describe("handlePluginToolCall gkill_get_plugin_list", () => {
  test("calls the plugin list endpoint and returns plugins", async () => {
    const plugins = [
      {
        name: "gkill_plugin_claudecode",
        version: "1.0.0",
        description: "Claude Code chat log",
        data_type: "claude_code_message",
        rep_name: "Claude Code",
        is_alive: true,
      },
    ];
    const call = vi.fn().mockResolvedValue({ plugins, errors: [] });
    const payload = await handlePluginToolCall(call, "gkill_get_plugin_list", {});
    expect(call).toHaveBeenCalledWith(GET_PLUGIN_LIST_ENDPOINT, {});
    expect(payload).toEqual({ plugins });
  });

  test("passes locale_name through", async () => {
    const call = vi.fn().mockResolvedValue({ plugins: [], errors: [] });
    await handlePluginToolCall(call, "gkill_get_plugin_list", { locale_name: "en" });
    expect(call).toHaveBeenCalledWith(GET_PLUGIN_LIST_ENDPOINT, { locale_name: "en" });
  });

  test("returns an empty array when the server omits plugins", async () => {
    const call = vi.fn().mockResolvedValue({ errors: [] });
    const payload = await handlePluginToolCall(call, "gkill_get_plugin_list", {});
    expect(payload).toEqual({ plugins: [] });
  });
});

describe("handlePluginToolCall gkill_get_plugin_content", () => {
  test("calls the content endpoint with rep_name and kyou_id", async () => {
    const call = vi.fn().mockResolvedValue({ html: CONTENT_HTML, errors: [] });
    await handlePluginToolCall(call, "gkill_get_plugin_content", {
      rep_name: "Claude Code",
      kyou_id: "abc",
    });
    expect(call).toHaveBeenCalledWith(GET_PLUGIN_CONTENT_HTML_ENDPOINT, {
      rep_name: "Claude Code",
      kyou_id: "abc",
    });
  });

  test("returns converted text and no html by default", async () => {
    const call = vi.fn().mockResolvedValue({ html: CONTENT_HTML, errors: [] });
    const payload = await handlePluginToolCall(call, "gkill_get_plugin_content", {
      rep_name: "Claude Code",
      kyou_id: "abc",
    });
    expect(payload.format).toBe("text");
    expect(payload.text).toContain("セッション名");
    expect(payload.text).toContain("プラグインの内容取得を実装したい");
    expect(payload.text).not.toContain("color:red");
    expect(payload.text_truncated).toBe(false);
    expect(payload.html).toBeUndefined();
    expect(payload.html_size_bytes).toBe(Buffer.byteLength(CONTENT_HTML, "utf8"));
  });

  test("returns raw html and no text when format=html", async () => {
    const call = vi.fn().mockResolvedValue({ html: CONTENT_HTML, errors: [] });
    const payload = await handlePluginToolCall(call, "gkill_get_plugin_content", {
      rep_name: "r",
      kyou_id: "k",
      format: "html",
    });
    expect(payload.html).toBe(CONTENT_HTML);
    expect(payload.text).toBeUndefined();
    expect(payload.text_truncated).toBeUndefined();
  });

  test("returns both when format=both", async () => {
    const call = vi.fn().mockResolvedValue({ html: CONTENT_HTML, errors: [] });
    const payload = await handlePluginToolCall(call, "gkill_get_plugin_content", {
      rep_name: "r",
      kyou_id: "k",
      format: "both",
    });
    expect(payload.html).toBe(CONTENT_HTML);
    expect(typeof payload.text).toBe("string");
  });

  test("truncates the text at max_text_length", async () => {
    const call = vi.fn().mockResolvedValue({ html: CONTENT_HTML, errors: [] });
    const payload = await handlePluginToolCall(call, "gkill_get_plugin_content", {
      rep_name: "r",
      kyou_id: "k",
      max_text_length: 5,
    });
    expect(payload.text_truncated).toBe(true);
    expect(payload.text.startsWith("セッション")).toBe(true);
  });

  test("forwards locale_name only when given", async () => {
    const call = vi.fn().mockResolvedValue({ html: "", errors: [] });
    await handlePluginToolCall(call, "gkill_get_plugin_content", {
      rep_name: "r",
      kyou_id: "k",
      locale_name: "en",
    });
    expect(call).toHaveBeenCalledWith(GET_PLUGIN_CONTENT_HTML_ENDPOINT, {
      rep_name: "r",
      kyou_id: "k",
      locale_name: "en",
    });
  });

  test("tolerates a missing html field", async () => {
    const call = vi.fn().mockResolvedValue({ errors: [] });
    const payload = await handlePluginToolCall(call, "gkill_get_plugin_content", {
      rep_name: "r",
      kyou_id: "k",
    });
    expect(payload.text).toBe("");
    expect(payload.html_size_bytes).toBe(0);
  });

  test("propagates API errors", async () => {
    const call = vi.fn().mockRejectedValue(new Error("plugin not found"));
    await expect(
      handlePluginToolCall(call, "gkill_get_plugin_content", { rep_name: "r", kyou_id: "k" }),
    ).rejects.toThrow("plugin not found");
  });
});

describe("handlePluginToolCall unknown tool", () => {
  test("throws for a non-plugin tool name", async () => {
    const call = vi.fn();
    await expect(handlePluginToolCall(call, "gkill_get_kyous", {})).rejects.toThrow(/Unknown tool/);
    expect(call).not.toHaveBeenCalled();
  });
});

// ---------------------------------------------------------------------------
// summarizePluginToolPayload
// ---------------------------------------------------------------------------
describe("summarizePluginToolPayload", () => {
  test("summarizes the plugin list", () => {
    expect(summarizePluginToolPayload("gkill_get_plugin_list", { plugins: [{}, {}] })).toBe(
      "Fetched 2 plugins.",
    );
    expect(summarizePluginToolPayload("gkill_get_plugin_list", {})).toBe("Fetched 0 plugins.");
  });

  test("summarizes text content", () => {
    const summary = summarizePluginToolPayload("gkill_get_plugin_content", {
      rep_name: "Claude Code",
      kyou_id: "abc",
      text: "hello",
      text_truncated: false,
    });
    expect(summary).toBe("Fetched plugin content for Claude Code/abc, 5 chars of text.");
  });

  test("marks truncated text", () => {
    const summary = summarizePluginToolPayload("gkill_get_plugin_content", {
      rep_name: "r",
      kyou_id: "k",
      text: "hello",
      text_truncated: true,
    });
    expect(summary).toContain("(truncated)");
  });

  test("summarizes html content", () => {
    const summary = summarizePluginToolPayload("gkill_get_plugin_content", {
      rep_name: "r",
      kyou_id: "k",
      html: "<p>x</p>",
      html_size_bytes: 8,
    });
    expect(summary).toBe("Fetched plugin content for r/k, 8 bytes of HTML.");
  });

  test("returns null for non-plugin tools so callers can fall back", () => {
    expect(summarizePluginToolPayload("gkill_get_kyous", {})).toBeNull();
  });
});
