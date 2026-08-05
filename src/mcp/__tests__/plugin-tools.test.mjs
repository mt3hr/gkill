/**
 * Tests for lib/plugin-tools.mjs — read / write / readwrite の3サーバが共有する
 * プラグイン関連ツールの定義とハンドラ、および gkill_get_kyous のレスポンスへ
 * プラグイン本文を埋め込む inlinePluginContents。
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
  collectPluginPayloads,
  runGroupedWithConcurrency,
  inlinePluginContents,
  summarizeInlinePluginContent,
} from "../lib/plugin-tools.mjs";

const CONTENT_HTML =
  "<html><head><style>body{color:red}</style><script>var a=1;</script></head>" +
  "<body><div class=\"conv-title\">セッション名</div><div class=\"msg\">" +
  "<div class=\"sender\">あなた</div>プラグインの内容取得を実装したい</div></body></html>";

// プラグインKyouを1件持つ get_kyous レスポンス相当の kyous 配列を作る。
function pluginKyou(repName, kyouID, extra = {}) {
  return {
    data_type: "claude_code_message",
    related_time: "2026-08-05T10:00:00+09:00",
    payload: { kind: "plugin", data_type: "claude_code_message", rep_name: repName, kyou_id: kyouID, ...extra },
  };
}

function deferred() {
  let resolve;
  let reject;
  const promise = new Promise((res, rej) => {
    resolve = res;
    reject = rej;
  });
  return { promise, resolve, reject };
}

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

  test("the single-kyou content tool is no longer exposed", () => {
    expect(PLUGIN_TOOL_NAMES).toEqual(["gkill_get_plugin_list"]);
    expect(PLUGIN_TOOLS).toHaveLength(1);
  });
});

describe("isPluginToolName", () => {
  test("recognizes plugin tools", () => {
    expect(isPluginToolName("gkill_get_plugin_list")).toBe(true);
  });

  test("rejects everything else", () => {
    expect(isPluginToolName("gkill_get_plugin_content")).toBe(false);
    expect(isPluginToolName("gkill_get_kyous")).toBe(false);
    expect(isPluginToolName("gkill_add_kmemo")).toBe(false);
    expect(isPluginToolName("")).toBe(false);
    expect(isPluginToolName(undefined)).toBe(false);
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

describe("handlePluginToolCall unknown tool", () => {
  test("throws for a non-plugin tool name", async () => {
    const call = vi.fn();
    await expect(handlePluginToolCall(call, "gkill_get_kyous", {})).rejects.toThrow(/Unknown tool/);
    expect(call).not.toHaveBeenCalled();
  });

  test("throws for the removed content tool name", async () => {
    const call = vi.fn();
    await expect(handlePluginToolCall(call, "gkill_get_plugin_content", {})).rejects.toThrow(/Unknown tool/);
    expect(call).not.toHaveBeenCalled();
  });
});

// ---------------------------------------------------------------------------
// collectPluginPayloads
// ---------------------------------------------------------------------------
describe("collectPluginPayloads", () => {
  test("collects plugin payloads in kyou order", () => {
    const kyous = [pluginKyou("A", "1"), pluginKyou("B", "2")];
    expect(collectPluginPayloads(kyous).map((p) => p.kyou_id)).toEqual(["1", "2"]);
  });

  test("ignores non-plugin payloads", () => {
    const kyous = [
      { data_type: "kmemo", payload: { kind: "kmemo", content: "x" } },
      { data_type: "idf", payload: { kind: "idf", rep_name: "Files", file_name: "a.png" } },
      pluginKyou("A", "1"),
    ];
    expect(collectPluginPayloads(kyous)).toHaveLength(1);
  });

  test("ignores kyous without a usable plugin payload", () => {
    const kyous = [
      null,
      "not an object",
      {},
      { payload: null },
      { payload: { kind: "plugin", rep_name: "", kyou_id: "1" } },
      { payload: { kind: "plugin", rep_name: "A" } },
    ];
    expect(collectPluginPayloads(kyous)).toEqual([]);
  });

  test("returns an empty array for a non-array input", () => {
    expect(collectPluginPayloads(undefined)).toEqual([]);
    expect(collectPluginPayloads(null)).toEqual([]);
    expect(collectPluginPayloads({})).toEqual([]);
  });
});

// ---------------------------------------------------------------------------
// runGroupedWithConcurrency
// ---------------------------------------------------------------------------
describe("runGroupedWithConcurrency", () => {
  test("runs items of the same key strictly one at a time", async () => {
    let inFlight = 0;
    let peak = 0;
    const entries = [1, 2, 3].map((n) => ({ key: "same", item: n }));
    await runGroupedWithConcurrency(entries, 4, async () => {
      inFlight++;
      peak = Math.max(peak, inFlight);
      await Promise.resolve();
      inFlight--;
      return true;
    });
    expect(peak).toBe(1);
  });

  test("preserves item order within a key", async () => {
    const seen = [];
    const entries = ["a", "b", "c"].map((item) => ({ key: "k", item }));
    await runGroupedWithConcurrency(entries, 2, async (item) => {
      seen.push(item);
      return true;
    });
    expect(seen).toEqual(["a", "b", "c"]);
  });

  test("runs different keys concurrently", async () => {
    let inFlight = 0;
    let peak = 0;
    const gates = [deferred(), deferred()];
    const entries = [
      { key: "x", item: 0 },
      { key: "y", item: 1 },
    ];
    const run = runGroupedWithConcurrency(entries, 2, async (index) => {
      inFlight++;
      peak = Math.max(peak, inFlight);
      await gates[index].promise;
      inFlight--;
      return true;
    });
    await Promise.resolve();
    await Promise.resolve();
    gates[0].resolve();
    gates[1].resolve();
    await run;
    expect(peak).toBe(2);
  });

  test("never exceeds the concurrency limit", async () => {
    let inFlight = 0;
    let peak = 0;
    const entries = ["a", "b", "c", "d", "e"].map((key) => ({ key, item: key }));
    await runGroupedWithConcurrency(entries, 2, async () => {
      inFlight++;
      peak = Math.max(peak, inFlight);
      await new Promise((resolve) => setTimeout(resolve, 1));
      inFlight--;
      return true;
    });
    expect(peak).toBeLessThanOrEqual(2);
  });

  test("stops only the affected key when the worker returns false", async () => {
    const seen = [];
    const entries = [
      { key: "x", item: "x1" },
      { key: "x", item: "x2" },
      { key: "y", item: "y1" },
    ];
    await runGroupedWithConcurrency(entries, 2, async (item) => {
      seen.push(item);
      return item !== "x1";
    });
    expect(seen).toEqual(expect.arrayContaining(["x1", "y1"]));
    expect(seen).not.toContain("x2");
  });

  test("swallows a worker exception and stops only that key", async () => {
    const seen = [];
    const entries = [
      { key: "x", item: "x1" },
      { key: "x", item: "x2" },
      { key: "y", item: "y1" },
    ];
    await expect(
      runGroupedWithConcurrency(entries, 2, async (item) => {
        seen.push(item);
        if (item === "x1") {
          throw new Error("boom");
        }
        return true;
      }),
    ).resolves.toBeUndefined();
    expect(seen).toContain("y1");
    expect(seen).not.toContain("x2");
  });

  test("resolves immediately for an empty entry list", async () => {
    const worker = vi.fn();
    await expect(runGroupedWithConcurrency([], 4, worker)).resolves.toBeUndefined();
    expect(worker).not.toHaveBeenCalled();
  });
});

// ---------------------------------------------------------------------------
// inlinePluginContents
// ---------------------------------------------------------------------------
describe("inlinePluginContents", () => {
  test("issues no request and reports zeroes when there is no plugin kyou", async () => {
    const call = vi.fn();
    const stats = await inlinePluginContents(call, [{ data_type: "kmemo", payload: { kind: "kmemo" } }]);
    expect(call).not.toHaveBeenCalled();
    expect(stats).toEqual({
      requested: 0,
      inlined: 0,
      truncated: 0,
      skipped: 0,
      errors: 0,
      total_text_length: 0,
    });
  });

  test("embeds converted text and marks the payload ok", async () => {
    const call = vi.fn().mockResolvedValue({ html: CONTENT_HTML, errors: [] });
    const kyous = [pluginKyou("Claude Code", "abc")];
    const stats = await inlinePluginContents(call, kyous);
    expect(kyous[0].payload.content_text).toContain("セッション名");
    expect(kyous[0].payload.content_text).not.toContain("color:red");
    expect(kyous[0].payload.content_status).toBe("ok");
    expect(stats.inlined).toBe(1);
    expect(stats.total_text_length).toBe(kyous[0].payload.content_text.length);
  });

  test("posts the content endpoint once per plugin kyou with only rep_name and kyou_id", async () => {
    const call = vi.fn().mockResolvedValue({ html: CONTENT_HTML, errors: [] });
    const kyous = [pluginKyou("A", "1"), pluginKyou("B", "2")];
    await inlinePluginContents(call, kyous);
    expect(call).toHaveBeenCalledTimes(2);
    expect(call).toHaveBeenCalledWith(GET_PLUGIN_CONTENT_HTML_ENDPOINT, { rep_name: "A", kyou_id: "1" });
    expect(call).toHaveBeenCalledWith(GET_PLUGIN_CONTENT_HTML_ENDPOINT, { rep_name: "B", kyou_id: "2" });
  });

  test("never passes an abort signal to the call", async () => {
    const call = vi.fn().mockResolvedValue({ html: CONTENT_HTML, errors: [] });
    await inlinePluginContents(call, [pluginKyou("A", "1")]);
    expect(call.mock.calls[0]).toHaveLength(2);
    expect(Object.keys(call.mock.calls[0][1]).sort()).toEqual(["kyou_id", "rep_name"]);
  });

  test("forwards locale_name only when given", async () => {
    const call = vi.fn().mockResolvedValue({ html: CONTENT_HTML, errors: [] });
    await inlinePluginContents(call, [pluginKyou("A", "1")], { localeName: "en" });
    expect(call).toHaveBeenCalledWith(GET_PLUGIN_CONTENT_HTML_ENDPOINT, {
      rep_name: "A",
      kyou_id: "1",
      locale_name: "en",
    });
  });

  test("leaves non-plugin payloads untouched", async () => {
    const call = vi.fn().mockResolvedValue({ html: CONTENT_HTML, errors: [] });
    const kmemo = { data_type: "kmemo", payload: { kind: "kmemo", content: "hello" } };
    const kyous = [kmemo, pluginKyou("A", "1")];
    await inlinePluginContents(call, kyous);
    expect(kmemo.payload).toEqual({ kind: "kmemo", content: "hello" });
  });

  test("never adds a file_name field to a plugin payload", async () => {
    // file_name を持たせると isIdfPayload がIDFと誤認し、applyFileLinks が
    // 無意味なファイルトークンを発行してしまう。
    const call = vi.fn().mockResolvedValue({ html: CONTENT_HTML, errors: [] });
    const kyous = [pluginKyou("A", "1")];
    await inlinePluginContents(call, kyous);
    expect(kyous[0].payload).not.toHaveProperty("file_name");
    expect(kyous[0].payload).not.toHaveProperty("file_path");
  });

  test("marks the payload truncated when the body exceeds maxTextLength", async () => {
    const call = vi.fn().mockResolvedValue({ html: CONTENT_HTML, errors: [] });
    const kyous = [pluginKyou("A", "1")];
    const stats = await inlinePluginContents(call, kyous, { maxTextLength: 5 });
    expect(kyous[0].payload.content_status).toBe("truncated");
    expect(kyous[0].payload.content_text.startsWith("セッション")).toBe(true);
    expect(stats.truncated).toBe(1);
  });

  test("returns raw html when format is html", async () => {
    const call = vi.fn().mockResolvedValue({ html: CONTENT_HTML, errors: [] });
    const kyous = [pluginKyou("A", "1")];
    await inlinePluginContents(call, kyous, { format: "html" });
    expect(kyous[0].payload.content_html).toBe(CONTENT_HTML);
    expect(kyous[0].payload.content_text).toBeUndefined();
  });

  test("returns both text and html when format is both", async () => {
    const call = vi.fn().mockResolvedValue({ html: CONTENT_HTML, errors: [] });
    const kyous = [pluginKyou("A", "1")];
    await inlinePluginContents(call, kyous, { format: "both" });
    expect(kyous[0].payload.content_html).toBe(CONTENT_HTML);
    expect(typeof kyous[0].payload.content_text).toBe("string");
  });

  test("clips an oversized html before converting and marks it truncated", async () => {
    const huge = `<div>${"あ".repeat(1000)}</div>`;
    const call = vi.fn().mockResolvedValue({ html: huge, errors: [] });
    const kyous = [pluginKyou("A", "1")];
    await inlinePluginContents(call, kyous, { maxHtmlLength: 50, maxTextLength: 100000 });
    expect(kyous[0].payload.content_status).toBe("truncated");
    expect(kyous[0].payload.content_text.length).toBeLessThan(60);
  });

  test("treats a missing html field as empty content", async () => {
    const call = vi.fn().mockResolvedValue({ errors: [] });
    const kyous = [pluginKyou("A", "1")];
    const stats = await inlinePluginContents(call, kyous);
    expect(kyous[0].payload.content_text).toBe("");
    expect(kyous[0].payload.content_status).toBe("ok");
    expect(stats.inlined).toBe(1);
  });

  test("fetches a repeated rep_name/kyou_id pair only once and fills both payloads", async () => {
    const call = vi.fn().mockResolvedValue({ html: CONTENT_HTML, errors: [] });
    const kyous = [pluginKyou("A", "1"), pluginKyou("A", "1")];
    const stats = await inlinePluginContents(call, kyous);
    expect(call).toHaveBeenCalledTimes(1);
    expect(kyous[0].payload.content_status).toBe("ok");
    expect(kyous[1].payload.content_status).toBe("ok");
    expect(stats.inlined).toBe(2);
  });

  test("isolates a failing rep and still resolves", async () => {
    const call = vi.fn().mockImplementation((_pathname, body) =>
      body.rep_name === "A" ? Promise.reject(new Error("plugin not found")) : Promise.resolve({ html: CONTENT_HTML }),
    );
    const kyous = [pluginKyou("A", "1"), pluginKyou("B", "2")];
    const stats = await inlinePluginContents(call, kyous);
    expect(kyous[0].payload.content_status).toBe("error");
    expect(kyous[0].payload.content_error).toBe("plugin not found");
    expect(kyous[1].payload.content_status).toBe("ok");
    expect(stats.errors).toBe(1);
    expect(stats.inlined).toBe(1);
  });

  test("stops requesting the rest of a rep after its first failure", async () => {
    const call = vi.fn().mockRejectedValue(new Error("dead plugin"));
    const kyous = [pluginKyou("A", "1"), pluginKyou("A", "2"), pluginKyou("A", "3")];
    const stats = await inlinePluginContents(call, kyous);
    expect(call).toHaveBeenCalledTimes(1);
    expect(kyous[0].payload.content_status).toBe("error");
    expect(kyous[1].payload.content_skipped_reason).toBe("rep_error");
    expect(kyous[2].payload.content_skipped_reason).toBe("rep_error");
    expect(stats.errors).toBe(1);
    expect(stats.skipped).toBe(2);
  });

  test("truncates a long error message", async () => {
    const call = vi.fn().mockRejectedValue(new Error("x".repeat(500)));
    const kyous = [pluginKyou("A", "1")];
    await inlinePluginContents(call, kyous);
    expect(kyous[0].payload.content_error).toHaveLength(200);
  });

  test("skips entries beyond maxKyous without issuing a request", async () => {
    const call = vi.fn().mockResolvedValue({ html: CONTENT_HTML, errors: [] });
    const kyous = [pluginKyou("A", "1"), pluginKyou("A", "2"), pluginKyou("A", "3")];
    const stats = await inlinePluginContents(call, kyous, { maxKyous: 2 });
    expect(call).toHaveBeenCalledTimes(2);
    expect(kyous[2].payload.content_status).toBe("skipped");
    expect(kyous[2].payload.content_skipped_reason).toBe("max_kyous");
    expect(stats.skipped).toBe(1);
  });

  test("applies the total budget in kyou order regardless of completion order", async () => {
    const slow = deferred();
    const call = vi.fn().mockImplementation((_pathname, body) =>
      body.kyou_id === "1" ? slow.promise : Promise.resolve({ html: CONTENT_HTML }),
    );
    const kyous = [pluginKyou("A", "1"), pluginKyou("B", "2")];
    const run = inlinePluginContents(call, kyous, { totalTextLength: 10 });
    await Promise.resolve();
    slow.resolve({ html: CONTENT_HTML });
    const stats = await run;
    // 先頭は予算に関係なく必ず載る。2件目は予算超過で skipped。
    expect(kyous[0].payload.content_status).toBe("ok");
    expect(kyous[1].payload.content_status).toBe("skipped");
    expect(kyous[1].payload.content_skipped_reason).toBe("budget");
    expect(stats.inlined).toBe(1);
  });

  test("stops starting requests once the deadline passed", async () => {
    let clock = 0;
    const call = vi.fn().mockImplementation(() => {
      clock += 1000;
      return Promise.resolve({ html: CONTENT_HTML });
    });
    const kyous = [pluginKyou("A", "1"), pluginKyou("A", "2")];
    const stats = await inlinePluginContents(call, kyous, { deadlineMs: 100, now: () => clock });
    expect(call).toHaveBeenCalledTimes(1);
    expect(kyous[1].payload.content_status).toBe("skipped");
    expect(kyous[1].payload.content_skipped_reason).toBe("deadline");
    expect(stats.skipped).toBe(1);
  });

  test("serializes calls within a rep and parallelizes across reps", async () => {
    const inFlight = new Map();
    const peak = new Map();
    const gates = new Map();
    const call = vi.fn().mockImplementation((_pathname, body) => {
      const rep = body.rep_name;
      const current = (inFlight.get(rep) ?? 0) + 1;
      inFlight.set(rep, current);
      peak.set(rep, Math.max(peak.get(rep) ?? 0, current));
      const gate = deferred();
      gates.set(`${rep}${body.kyou_id}`, gate);
      return gate.promise.then(() => {
        inFlight.set(rep, inFlight.get(rep) - 1);
        return { html: CONTENT_HTML };
      });
    });
    const kyous = [pluginKyou("A", "1"), pluginKyou("A", "2"), pluginKyou("B", "1")];
    const run = inlinePluginContents(call, kyous, { concurrency: 4 });
    await Promise.resolve();
    await Promise.resolve();
    expect(call).toHaveBeenCalledTimes(2); // A/1 と B/1 が同時に飛ぶ
    gates.get("A1").resolve();
    gates.get("B1").resolve();
    await Promise.resolve();
    await Promise.resolve();
    await Promise.resolve();
    gates.get("A2").resolve();
    await run;
    expect(peak.get("A")).toBe(1);
    expect(call).toHaveBeenCalledTimes(3);
  });

  test("keeps the stats arithmetic consistent across a mixed run", async () => {
    const call = vi.fn().mockImplementation((_pathname, body) =>
      body.rep_name === "B" ? Promise.reject(new Error("nope")) : Promise.resolve({ html: CONTENT_HTML }),
    );
    const kyous = [pluginKyou("A", "1"), pluginKyou("B", "2"), pluginKyou("C", "3"), pluginKyou("C", "4")];
    const stats = await inlinePluginContents(call, kyous, { maxKyous: 3 });
    expect(stats.requested).toBe(4);
    expect(stats.inlined + stats.skipped + stats.errors).toBe(stats.requested);
  });

  test("resolves even when every rep fails", async () => {
    const call = vi.fn().mockRejectedValue(new Error("all dead"));
    const kyous = [pluginKyou("A", "1"), pluginKyou("B", "2")];
    const stats = await inlinePluginContents(call, kyous);
    expect(stats.errors).toBe(2);
    expect(stats.inlined).toBe(0);
  });
});

// ---------------------------------------------------------------------------
// summarize
// ---------------------------------------------------------------------------
describe("summarizePluginToolPayload", () => {
  test("summarizes the plugin list", () => {
    expect(summarizePluginToolPayload("gkill_get_plugin_list", { plugins: [{}, {}] })).toBe(
      "Fetched 2 plugins.",
    );
    expect(summarizePluginToolPayload("gkill_get_plugin_list", {})).toBe("Fetched 0 plugins.");
  });

  test("returns null for non-plugin tools so callers can fall back", () => {
    expect(summarizePluginToolPayload("gkill_get_kyous", {})).toBeNull();
    expect(summarizePluginToolPayload("gkill_get_plugin_content", {})).toBeNull();
  });
});

describe("summarizeInlinePluginContent", () => {
  test("returns an empty string when nothing was inlined", () => {
    expect(summarizeInlinePluginContent(undefined)).toBe("");
    expect(summarizeInlinePluginContent({ requested: 0, inlined: 0, truncated: 0, skipped: 0, errors: 0 })).toBe(
      "",
    );
  });

  test("reports the embedded count", () => {
    expect(
      summarizeInlinePluginContent({ requested: 3, inlined: 3, truncated: 0, skipped: 0, errors: 0 }),
    ).toBe(" Embedded plugin content for 3 of 3 plugin kyous.");
  });

  test("reports truncated, skipped and failed counts", () => {
    const summary = summarizeInlinePluginContent({
      requested: 6,
      inlined: 3,
      truncated: 1,
      skipped: 2,
      errors: 1,
    });
    expect(summary).toBe(" Embedded plugin content for 3 of 6 plugin kyous (1 truncated, 2 not fetched, 1 failed).");
  });
});
