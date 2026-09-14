/**
 * 「tools/list どおりに呼ぶと失敗しない」の契約（2026-09-14 レビュー P0）。
 *
 * ChatGPT から「公開スキーマに載っている検索条件名で呼ぶと未知の引数として拒否される」が
 * 実測された。あの件はクライアントが古い一覧を握っていただけで、ソースは両側とも直っていたが、
 * 同じ症状は「スキーマと正規化器の片方だけを改名する」だけで再現できる。
 * ここではその事故を構造的に防ぐ:
 *
 * 1. スキーマのキー集合と正規化器の受理集合が一致する（廃止済みだけが受理側に多い）
 * 2. 各ツールを「スキーマの全プロパティを指定して」呼んでも未知キーで拒否されない
 * 3. read / write / readwrite の tools/list に載る同名ツールは同じ JSON
 *    （gkill_status だけ description 末尾の schema_revision が違う。それが唯一の例外）
 * 4. initialize の version・gkill_status の description・gkill_status の応答が同じ世代を指す
 */

import { describe, test, expect, vi } from "vitest";

import { McpServer as ReadServer } from "../gkill-read-server.mjs";
import { McpWriteServer } from "../gkill-write-server.mjs";
import { McpServer as ReadWriteServer } from "../gkill-readwrite-server.mjs";
import { READ_TOOLS } from "../lib/read-tools.mjs";
import { WRITE_TOOLS } from "../lib/write-tools.mjs";
import { PLUGIN_TOOLS, handlePluginToolCall } from "../lib/plugin-tools.mjs";
import { FIND_QUERY_SCHEMA } from "../lib/find-query-schema.mjs";
import { KYOUS_TOP_LEVEL_FIELDS, KYOUS_QUERY_ALL_FIELDS, ENTITY_TARGETS } from "../lib/constants.mjs";
import { DEPRECATED_TOP_LEVEL_ARGS, DEPRECATED_QUERY_FIELDS } from "../lib/normalization.mjs";
import { handleReadToolCall } from "../lib/read-handlers.mjs";
import { handleWriteToolCall } from "../lib/write-handlers.mjs";
import { GkillApiError } from "../lib/errors.mjs";
import { encodeGpsCursor } from "../lib/gps-cursor.mjs";
import { computeSchemaRevision, stripSchemaRevisionMark, STATUS_TOOL_NAME } from "../lib/status-tool.mjs";

// ---------------------------------------------------------------------------
// 1. キー集合の一致
// ---------------------------------------------------------------------------
describe("advertised keys and accepted keys agree", () => {
  test("FIND_QUERY_SCHEMA.properties == KYOUS_QUERY_ALL_FIELDS minus deprecated", () => {
    const advertised = new Set(Object.keys(FIND_QUERY_SCHEMA.properties));
    const accepted = new Set([...KYOUS_QUERY_ALL_FIELDS].filter((key) => !DEPRECATED_QUERY_FIELDS.has(key)));
    expect([...advertised].sort()).toEqual([...accepted].sort());
    // 廃止済みは受理側にだけある（公開しない・受理はする）
    for (const key of DEPRECATED_QUERY_FIELDS) {
      expect(KYOUS_QUERY_ALL_FIELDS.has(key)).toBe(true);
      expect(advertised.has(key)).toBe(false);
    }
  });

  test("gkill_get_kyous top-level properties == KYOUS_TOP_LEVEL_FIELDS minus deprecated", () => {
    const tool = READ_TOOLS.find((candidate) => candidate.name === "gkill_get_kyous");
    const advertised = new Set(Object.keys(tool.inputSchema.properties));
    const accepted = new Set([...KYOUS_TOP_LEVEL_FIELDS].filter((key) => !DEPRECATED_TOP_LEVEL_ARGS.has(key)));
    expect([...advertised].sort()).toEqual([...accepted].sort());
    for (const key of DEPRECATED_TOP_LEVEL_ARGS) {
      expect(KYOUS_TOP_LEVEL_FIELDS.has(key)).toBe(true);
      expect(advertised.has(key)).toBe(false);
    }
  });
});

// ---------------------------------------------------------------------------
// 2. 全プロパティ指定のスモーク
// ---------------------------------------------------------------------------

const SAMPLE_DATETIME = "2026-01-02T03:04:05+09:00";
const SAMPLE_ID = "00000000-0000-4000-8000-000000000001";

// 引数名から「型は string だが中身に形式がある」値を決める。
// スキーマの type / enum / minimum だけでは日時・URL・カーソルの形が分からない。
function sampleString(key, schema) {
  if (Array.isArray(schema.enum) && schema.enum.length > 0) {
    return schema.enum[0];
  }
  if (key === "cursor") {
    return schema.description.includes("gkill_get_kyous cursor")
      ? encodeGpsCursor(SAMPLE_DATETIME, 0) // GPS 側（説明文が get_kyous のカーソルと別物だと言う）
      : `${SAMPLE_DATETIME}::${SAMPLE_ID}`; // Kyou 側（複合カーソル）
  }
  if (key === "thumb") return "512x512";
  if (key === "url") return "https://example.com/";
  if (key === "locale_name") return "ja";
  if (key === "playing_time") return "now";
  if (/(_date|_time)$/.test(key) || key === "update_time") return SAMPLE_DATETIME;
  if (key === "id" || key === "target_id") return SAMPLE_ID;
  if (key === "data_type") return "kmemo";
  if (key === "board_name") return "Inbox";
  if (key === "kftl_text") return "sample";
  return "sample";
}

function sampleValue(key, schema) {
  const type = Array.isArray(schema.type) ? schema.type.find((candidate) => candidate !== "null") : schema.type;
  switch (type) {
    case "string":
      return sampleString(key, schema);
    case "integer":
    case "number":
      return typeof schema.minimum === "number" ? Math.max(schema.minimum, 1) : 1;
    case "boolean":
      return true;
    case "array":
      return [sampleValue(key, schema.items ?? { type: "string" })];
    case "object":
      return sampleObject(schema);
    default:
      throw new Error(`no sample for ${key}: ${JSON.stringify(schema)}`);
  }
}

function sampleObject(schema) {
  const out = {};
  for (const [key, child] of Object.entries(schema.properties ?? {})) {
    out[key] = sampleValue(key, child);
  }
  // 時間帯の窓・数値レンジは「始 <= 終」でないと意味検証で落ちるので、同じ値にする
  if ("period_of_time_start_time_second" in out) out.period_of_time_start_time_second = 0;
  if ("period_of_time_end_time_second" in out) out.period_of_time_end_time_second = 0;
  if ("period_of_time_week_of_days" in out) out.period_of_time_week_of_days = [0];
  if ("num_min" in out && "num_max" in out) out.num_max = out.num_min;
  return out;
}

// 型別エンドポイントの応答（履歴1件）と、その他の応答を1つの mock にまとめる。
// どのツールが何を読むかを個別に知らなくてよいよう、全部入りで返す。
// deleted: 履歴の最新版を削除済みにする（gkill_restore_kyou は未削除だと already active で断る）。
function smokeApiResponse({ deleted = false } = {}) {
  const response = { errors: null, messages: null, kyous: [], boards: ["Inbox"], tag_names: [], rep_names: [], gps_logs: [],
    rep_infos: [], canonical_rep_types: [], plugins: [], attached_data_reps: [],
    application_config: { user_id: "testuser", device: "testdevice" }, created: [] };
  for (const [dataType, target] of Object.entries(ENTITY_TARGETS)) {
    response[target.historiesKey] = [
      { id: SAMPLE_ID, data_type: dataType, is_deleted: deleted, update_time: "2020-01-01T00:00:00+09:00", tag: "t", text: "x", content: "c", title: "t", url: "https://example.com/", amount: 1, mood: 1, num_value: 1, is_checked: false, board_name: "Inbox" },
    ];
  }
  return response;
}

function smokeCtx(options = {}) {
  const callApi = vi.fn(async () => smokeApiResponse(options));
  return {
    client: {
      callApi,
      fetchFile: vi.fn(async () => ({ buffer: Buffer.from("x"), contentType: "image/jpeg" })),
      login: vi.fn(async () => "sid"),
    },
    sid: "sid",
    userId: "testuser",
    appName: "gkill_mcp_readwrite",
    isLocalTransport: true,
    server: { kind: "readwrite", name: "x", version: "0", schemaRevision: "0", toolCount: 0, transport: "stdio", startedAt: new Date() },
  };
}

// 未知キーの拒否だけを事故とみなす。意味検証（cursor と count_only の併用など）で落ちたら、
// その引数を除いて呼び直す —— 「スキーマにある名前を正規化器が知らない」だけを検出したい。
function isUnknownKeyError(error) {
  return error instanceof GkillApiError && /is not supported/.test(error.message);
}

function dropField(args, field) {
  const path = field.replace(/^arguments\./, "").split(".");
  const next = structuredClone(args);
  let cursor = next;
  for (const segment of path.slice(0, -1)) {
    cursor = cursor[segment];
  }
  delete cursor[path[path.length - 1]];
  return next;
}

async function callWithEverySchemaField(tool, invoke) {
  let args = sampleObject(tool.inputSchema);
  const dropped = [];
  for (let attempt = 0; attempt < 12; attempt++) {
    try {
      await invoke(args);
      return { ok: true, dropped };
    } catch (error) {
      if (isUnknownKeyError(error)) {
        throw new Error(`${tool.name}: a field advertised in inputSchema is rejected as unknown — ${error.message}`);
      }
      const field = error instanceof GkillApiError ? error.detail?.field : undefined;
      if (typeof field !== "string") {
        throw new Error(`${tool.name}: unexpected failure with all schema fields set — ${error.message}`);
      }
      dropped.push(field);
      args = dropField(args, field);
    }
  }
  throw new Error(`${tool.name}: gave up after dropping ${dropped.join(", ")}`);
}

describe("every advertised property is accepted by the tool's normalizer", () => {
  for (const tool of READ_TOOLS) {
    test(`${tool.name} (read)`, async () => {
      const ctx = smokeCtx();
      const result = await callWithEverySchemaField(tool, (args) => handleReadToolCall(ctx, tool.name, args));
      expect(result.ok).toBe(true);
    });
  }
  for (const tool of WRITE_TOOLS) {
    test(`${tool.name} (write)`, async () => {
      const ctx = smokeCtx({ deleted: tool.name === "gkill_restore_kyou" });
      const result = await callWithEverySchemaField(tool, (args) => handleWriteToolCall(ctx, tool.name, args));
      expect(result.ok).toBe(true);
    });
  }
  for (const tool of PLUGIN_TOOLS) {
    test(`${tool.name} (plugin)`, async () => {
      const ctx = smokeCtx();
      const result = await callWithEverySchemaField(tool, (args) =>
        handlePluginToolCall((pathname, body) => ctx.client.callApi(pathname, body), tool.name, args),
      );
      expect(result.ok).toBe(true);
    });
  }
});

// ---------------------------------------------------------------------------
// 3. 3サーバの tools/list に載る同名ツールは同じ JSON
// ---------------------------------------------------------------------------

function mockClient() {
  return {
    callApi: vi.fn(async () => ({ application_config: { user_id: "testuser", device: "testdevice" } })),
    fetchFile: vi.fn(),
    login: vi.fn(async () => "sid"),
    defaultLocale: "ja",
  };
}

async function toolsListOf(server) {
  const response = await server.handleMessage({ jsonrpc: "2.0", id: 1, method: "tools/list" });
  return response.result.tools;
}

describe("the three servers advertise identical definitions for a shared tool name", () => {
  test("same name => same JSON (gkill_status differs only by its schema_revision mark)", async () => {
    const servers = [new ReadServer(mockClient()), new McpWriteServer(mockClient()), new ReadWriteServer(mockClient())];
    const lists = await Promise.all(servers.map(toolsListOf));
    const byName = new Map();
    for (const tools of lists) {
      for (const tool of tools) {
        const canonical = JSON.stringify({ ...tool, description: stripSchemaRevisionMark(tool.description) });
        const seen = byName.get(tool.name);
        if (seen === undefined) {
          byName.set(tool.name, canonical);
        } else {
          expect(canonical, `${tool.name} differs between servers`).toBe(seen);
        }
      }
    }
    // gkill_status は3サーバ全部に載る
    for (const tools of lists) {
      expect(tools.some((tool) => tool.name === STATUS_TOOL_NAME)).toBe(true);
    }
  });
});

// ---------------------------------------------------------------------------
// 4. 世代の一致
// ---------------------------------------------------------------------------
describe("schema_revision is consistent per server", () => {
  test.each([
    ["read", () => new ReadServer(mockClient())],
    ["write", () => new McpWriteServer(mockClient())],
    ["readwrite", () => new ReadWriteServer(mockClient())],
  ])("%s: initialize version, stamped description, response and recomputation agree", async (kind, make) => {
    const server = make();
    const initialize = await server.handleMessage({ jsonrpc: "2.0", id: 1, method: "initialize" });
    const fromVersion = /\+schema\.([0-9a-f]{12})$/.exec(initialize.result.serverInfo.version)?.[1];
    const tools = await toolsListOf(server);
    const status = tools.find((tool) => tool.name === STATUS_TOOL_NAME);
    const fromDescription = / \[schema_revision: ([0-9a-f]{12})\]$/.exec(status.description)?.[1];
    const call = await server.handleMessage({
      jsonrpc: "2.0",
      id: 2,
      method: "tools/call",
      params: { name: STATUS_TOOL_NAME, arguments: {} },
    });
    expect(call.result.isError).toBe(false);
    const fromResponse = call.result.structuredContent.schema_revision;
    expect(fromVersion).toBeDefined();
    expect(fromDescription).toBe(fromVersion);
    expect(fromResponse).toBe(fromVersion);
    // 配られた一覧から計算し直しても同じ（焼き込みは計算対象に入らない）
    expect(computeSchemaRevision(tools)).toBe(fromVersion);
    expect(call.result.structuredContent.server_kind).toBe(kind);
    expect(call.result.structuredContent.tool_count).toBe(tools.length);
  });
});
