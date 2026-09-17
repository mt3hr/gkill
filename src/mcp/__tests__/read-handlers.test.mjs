/**
 * lib/read-handlers.mjs の v2 機能のテスト。
 * - get_kyous v2: 新パラメータの転送・応答の素通し（total_count はカーソルページで作らない）
 * - application_config: fields 射影 + UI状態キーの strip
 * - GPS: Node側ページング（複合カーソル・同一時刻ラン跨ぎ・count_only・日別バケット）
 * - rep_infos: ディスパッチと applyFileLinks 不変（file-link トークンの誤発行防止）
 * - idf_file: /files/ クエリ組み立て（?is_video=true&thumb=WxH）・thumb エコー・サイズ上限超過の案内
 */

import { describe, test, expect, vi } from "vitest";

import {
  handleReadToolCall,
  isReadToolName,
  stripAppConfigUiState,
  paginateGpsLogs,
  paginateRepNames,
  encodeGpsCursor,
  decodeGpsCursor,
  summarizeReadToolPayload,
} from "../lib/read-handlers.mjs";
import { normalizeGpsArgs } from "../lib/normalization.mjs";
import { applyFileLinks } from "../lib/payload.mjs";
import { FileLinkStore } from "../lib/file-link-store.mjs";
import { GkillApiError } from "../lib/errors.mjs";
import { MAX_IDF_FILE_BYTES } from "../lib/constants.mjs";

function makeCtx(callApiImpl) {
  return {
    client: {
      callApi: vi.fn(callApiImpl ?? (async () => ({ errors: [], messages: [] }))),
      fetchFile: vi.fn(),
      login: vi.fn(async () => "mock-session"),
    },
    sid: "sid-1",
    isLocalTransport: true,
  };
}

// ---------------------------------------------------------------------------
// get_kyous v2
// ---------------------------------------------------------------------------
describe("handleReadToolCall — gkill_get_kyous v2", () => {
  test("forwards v2 params and does not forward deprecated include flags", async () => {
    const ctx = makeCtx(async () => ({
      kyous: [],
      total_count: 0,
      returned_count: 0,
      remaining_count: 0,
      has_more: false,
    }));
    // count_only と group_by は併用できない（2026-09-18 以降はエラー）ので、group_by は別の呼び出しで確かめる
    await handleReadToolCall(ctx, "gkill_get_kyous", {
      count_only: true,
      data_types: ["nlog"],
      create_apps: ["appA"],
      update_apps: ["appB"],
      num_min: 100,
      num_max: 500,
      idf_kinds: ["image"],
      include_file_size: true,
      include_id: true, // deprecated: 受理はするが転送しない
    });
    const [pathname, body] = ctx.client.callApi.mock.calls[0];
    expect(pathname).toBe("/api/get_kyous_mcp");
    expect(body.count_only).toBe(true);
    await handleReadToolCall(ctx, "gkill_get_kyous", { group_by: "month" });
    expect(ctx.client.callApi.mock.calls[1][1].group_by).toBe("month");
    expect(ctx.client.callApi.mock.calls[1][1].count_only).toBe(false);
    expect(body.data_types).toEqual(["nlog"]);
    expect(body.create_apps).toEqual(["appA"]);
    expect(body.update_apps).toEqual(["appB"]);
    expect(body.num_min).toBe(100);
    expect(body.num_max).toBe(500);
    expect(body.idf_kinds).toEqual(["image"]);
    expect(body.include_file_size).toBe(true);
    expect(body).not.toHaveProperty("include_id");
    expect(body).not.toHaveProperty("include_rep_name");
  });

  test("passes opaque composite cursor verbatim", async () => {
    const ctx = makeCtx(async () => ({ kyous: [], returned_count: 0, remaining_count: 0, has_more: false }));
    const cursor = "2026-08-01T20:00:00.123456789+09:00::plugin::id::with::colons";
    await handleReadToolCall(ctx, "gkill_get_kyous", { cursor });
    expect(ctx.client.callApi.mock.calls[0][1].cursor).toBe(cursor);
  });

  test("omits total_count on cursor pages instead of fabricating 0", async () => {
    // 旧実装は total_count ?? 0 で埋めており、カーソルページで嘘の0を作っていた
    const ctx = makeCtx(async () => ({
      kyous: [{ id: "a" }],
      returned_count: 1,
      remaining_count: 3,
      has_more: true,
      next_cursor: "t::a",
    }));
    // カーソルは形を検証するようになったので、実物と同じ `{RFC3339}::{id}` を使う
    const payload = await handleReadToolCall(ctx, "gkill_get_kyous", {
      cursor: "2026-08-24T03:00:00+09:00::z",
    });
    expect(payload).not.toHaveProperty("total_count");
    expect(payload.remaining_count).toBe(3);
    expect(payload.next_cursor).toBe("t::a");
  });

  test("passes through buckets and warnings", async () => {
    const ctx = makeCtx(async () => ({
      kyous: [],
      total_count: 7,
      returned_count: 0,
      remaining_count: 0,
      has_more: false,
      buckets: [{ key: "2026-01", count: 7 }],
      warnings: ["unknown rep_type \"Kmemo\""],
    }));
    const payload = await handleReadToolCall(ctx, "gkill_get_kyous", { group_by: "month" });
    expect(payload.buckets).toEqual([{ key: "2026-01", count: 7 }]);
    expect(payload.warnings).toEqual(['unknown rep_type "Kmemo"']);
    expect(payload.total_count).toBe(7);
  });
});

// ---------------------------------------------------------------------------
// application_config
// ---------------------------------------------------------------------------
describe("handleReadToolCall — gkill_get_application_config", () => {
  const config = {
    tag_struct: {
      name: "root",
      id: "uuid-root",
      key: "__root__",
      is_checked: true,
      indeterminate: false,
      seq_in_parent: 1,
      check_when_inited: true,
      is_force_hide: false,
      children: [
        {
          name: "tagA",
          tag_name: "tagA",
          id: "uuid-a",
          key: "tagA",
          is_checked: false,
          check_when_inited: true,
          is_force_hide: true,
        },
      ],
    },
    mi_board_struct: { name: "boards" },
    rep_struct: { name: "reps" },
    rep_type_struct: { name: "types" },
    device_struct: { name: "devices" },
    kftl_template_struct: { name: "templates" },
    mi_default_board: "Inbox",
    show_tags_in_list: true,
  };

  test("strips UI-state keys by default but keeps check_when_inited / is_force_hide", async () => {
    const ctx = makeCtx(async () => ({ application_config: config }));
    const payload = await handleReadToolCall(ctx, "gkill_get_application_config", {});
    expect(payload.tag_struct).not.toHaveProperty("is_checked");
    expect(payload.tag_struct).not.toHaveProperty("key");
    expect(payload.tag_struct).not.toHaveProperty("id");
    expect(payload.tag_struct).not.toHaveProperty("seq_in_parent");
    expect(payload.tag_struct.check_when_inited).toBe(true);
    expect(payload.tag_struct.children[0].is_force_hide).toBe(true);
    expect(payload.tag_struct.children[0].tag_name).toBe("tagA");
  });

  // どのアカウントに繋がっているかを答えられること。
  // read サーバと readwrite サーバが別アカウントを向いていても AI から区別できず、
  // 「同じAPIなのに件数が違う」「query.ids が壊れている」と誤診されていた
  // （2026-08-24 の実利用レビュー）。gkill は元から返しており、射影が捨てていただけ。
  test("exposes user_id / device so the caller can tell which account it is on", async () => {
    const ctx = makeCtx(async () => ({
      application_config: { ...config, user_id: "testuser", device: "testdevice" },
    }));
    const payload = await handleReadToolCall(ctx, "gkill_get_application_config", {
      fields: ["user_id", "device"],
    });
    expect(payload).toEqual({ user_id: "testuser", device: "testdevice" });
  });

  test("fields projection returns only the requested fields", async () => {
    const ctx = makeCtx(async () => ({ application_config: config }));
    const payload = await handleReadToolCall(ctx, "gkill_get_application_config", { fields: ["tag_struct", "mi_default_board"] });
    expect(Object.keys(payload).sort()).toEqual(["mi_default_board", "tag_struct"]);
    expect(payload.mi_default_board).toBe("Inbox");
  });

  test("include_ui_state: true keeps everything", async () => {
    const ctx = makeCtx(async () => ({ application_config: config }));
    const payload = await handleReadToolCall(ctx, "gkill_get_application_config", { include_ui_state: true });
    expect(payload.tag_struct.is_checked).toBe(true);
    expect(payload.tag_struct.key).toBe("__root__");
  });
});

// ---------------------------------------------------------------------------
// gkill_status（2026-09-14 レビュー P0: クライアントの古い一覧を AI 自身が見分ける経路）
// ---------------------------------------------------------------------------
describe("handleReadToolCall — gkill_status", () => {
  const serverInfo = {
    kind: "readwrite",
    name: "gkill-readwrite-mcp",
    version: "1.2.3",
    schemaRevision: "0123456789ab",
    toolCount: 32,
    transport: "http",
    startedAt: new Date(Date.now() - 90_000),
  };

  test("returns the server description plus the account and build gkill reports", async () => {
    const ctx = {
      ...makeCtx(async () => ({
        application_config: {
          user_id: "testuser",
          device: "testdevice",
          version: "9.9.9",
          commit_hash: "abcdef0",
          build_time: "2026-09-14T00:00:00+09:00",
          tag_struct: { name: "root" },
        },
      })),
      server: serverInfo,
    };
    const payload = await handleReadToolCall(ctx, "gkill_status", {});
    expect(payload.server_kind).toBe("readwrite");
    expect(payload.server_name).toBe("gkill-readwrite-mcp");
    expect(payload.server_version).toBe("1.2.3");
    expect(payload.schema_revision).toBe("0123456789ab");
    expect(payload.tool_count).toBe(32);
    expect(payload.transport).toBe("http");
    expect(payload.started_at).toMatch(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}[+-]\d{2}:\d{2}$/);
    expect(payload.uptime_seconds).toBeGreaterThanOrEqual(89);
    expect(payload.gkill_reachable).toBe(true);
    expect(payload.account).toEqual({ user_id: "testuser", device: "testdevice" });
    expect(payload.gkill).toEqual({ version: "9.9.9", commit_hash: "abcdef0", build_time: "2026-09-14T00:00:00+09:00" });
    // ApplicationConfig の残り（tag_struct 等）は載せない
    expect(payload).not.toHaveProperty("tag_struct");
    expect(ctx.client.callApi).toHaveBeenCalledWith("/api/get_application_config", {}, true, "sid-1");
  });

  // gkill へ届かなくても MCP 側の情報は返す。「MCP は生きているが gkill が落ちている」を
  // 区別できるのがこのツールの仕事で、失敗させるとその区別が消える。
  test("still answers when gkill is unreachable, without leaking the error text", async () => {
    const warn = vi.fn();
    const ctx = {
      ...makeCtx(async () => {
        throw new GkillApiError("connect ECONNREFUSED http://127.0.0.1:9999/api/get_application_config", { status: 503 });
      }),
      server: serverInfo,
      accessLog: { warn },
    };
    const payload = await handleReadToolCall(ctx, "gkill_status", {});
    expect(payload.gkill_reachable).toBe(false);
    expect(payload.gkill_error).toBe("HTTP 503");
    expect(payload.schema_revision).toBe("0123456789ab");
    expect(payload).not.toHaveProperty("account");
    // 接続先 URL を含みうる本文は応答に載せず、ログにだけ残す（ADR-0707）
    expect(JSON.stringify(payload)).not.toContain("127.0.0.1");
    expect(warn).toHaveBeenCalledWith("status_gkill_unreachable", expect.objectContaining({ error: expect.stringContaining("ECONNREFUSED") }));
  });

  test("reports 'unreachable' when the failure carries no HTTP status", async () => {
    const ctx = { ...makeCtx(async () => { throw new Error("socket hang up"); }), server: serverInfo };
    const payload = await handleReadToolCall(ctx, "gkill_status", {});
    expect(payload.gkill_reachable).toBe(false);
    expect(payload.gkill_error).toBe("unreachable");
  });

  // 接続先 URL を detail に持つ Network error も "unreachable" に畳む（URL は載せない）
  test("folds a network error (whose detail carries the URL) into 'unreachable'", async () => {
    const ctx = {
      ...makeCtx(async () => {
        throw new GkillApiError("Network error at /api/get_application_config.", { url: "https://127.0.0.1:9999/api/get_application_config", message: "ECONNREFUSED" });
      }),
      server: serverInfo,
    };
    const payload = await handleReadToolCall(ctx, "gkill_status", {});
    expect(payload.gkill_error).toBe("unreachable");
    expect(JSON.stringify(payload)).not.toContain("127.0.0.1");
  });

  // 資格情報の誤りは何度呼んでも直らず、ログインの回数制限を食う。名指しで止める。
  test("names a login failure with its error code and tells the caller not to retry", async () => {
    const ctx = {
      ...makeCtx(async () => {
        throw new GkillApiError("Login failed: ERR000005: ユーザIDまたはパスワードが違います", {
          errors: [{ error_code: "ERR000005", error_message: "ユーザIDまたはパスワードが違います" }],
          messages: null,
        });
      }),
      server: serverInfo,
    };
    const payload = await handleReadToolCall(ctx, "gkill_status", {});
    expect(payload.gkill_reachable).toBe(false);
    expect(payload.gkill_error).toMatch(/^login_failed \(ERR000005\) — do not retry/);
    expect(payload.gkill_error).toMatch(/rate limit/);
  });

  test("names an API error by its error code only", async () => {
    const ctx = {
      ...makeCtx(async () => {
        throw new GkillApiError("API error at /api/get_application_config: ERR000999: x", {
          errors: [{ error_code: "ERR000999", error_message: "x" }],
        });
      }),
      server: serverInfo,
    };
    const payload = await handleReadToolCall(ctx, "gkill_status", {});
    expect(payload.gkill_error).toBe("api_error (ERR000999)");
  });

  test("takes no arguments and names the stale-list possibility when one is sent", async () => {
    const ctx = { ...makeCtx(), server: serverInfo };
    await expect(handleReadToolCall(ctx, "gkill_status", { locale_name: "ja" })).rejects.toThrow(
      /arguments\.locale_name.*is not supported.*stale/s,
    );
  });

  test("summary names the account, the kind and the revision", () => {
    expect(
      summarizeReadToolPayload("gkill_status", {
        server_kind: "read",
        schema_revision: "0123456789ab",
        uptime_seconds: 42,
        gkill_reachable: true,
        account: { user_id: "testuser", device: "testdevice" },
      }),
    ).toBe("Connected to testuser@testdevice via read server (schema_revision 0123456789ab, up 42s).");
    expect(
      summarizeReadToolPayload("gkill_status", {
        server_kind: "read",
        schema_revision: "0123456789ab",
        uptime_seconds: 42,
        gkill_reachable: false,
        gkill_error: "HTTP 503",
      }),
    ).toBe("MCP read server is up (schema_revision 0123456789ab, up 42s) but gkill is NOT reachable (HTTP 503).");
  });

  test("isReadToolName covers gkill_status", () => {
    expect(isReadToolName("gkill_status")).toBe(true);
  });
});

describe("stripAppConfigUiState", () => {
  test("recurses arrays and objects, leaves scalars", () => {
    const stripped = stripAppConfigUiState({
      key: "drop-me",
      keep: [{ id: "drop", name: "keep" }],
      name: "n",
    });
    expect(stripped).toEqual({ keep: [{ name: "keep" }], name: "n" });
  });
});

// ---------------------------------------------------------------------------
// GPS ページング（純関数）
// ---------------------------------------------------------------------------
describe("paginateGpsLogs", () => {
  const point = (t, lat) => ({ related_time: t, latitude: lat, longitude: 139.5 });
  // サーバの並び: 時刻降順・座標タイブレーク（決定的）
  const logs = [
    point("2026-08-03T12:00:00+09:00", 35.31),
    point("2026-08-02T12:00:00+09:00", 35.31),
    point("2026-08-02T12:00:00+09:00", 35.32),
    point("2026-08-02T12:00:00+09:00", 35.33),
    point("2026-08-01T12:00:00+09:00", 35.31),
  ];

  test("count_only returns only total_count", () => {
    const result = paginateGpsLogs(logs, { count_only: true, limit: 500 });
    expect(result).toEqual({ gps_logs: [], total_count: 5, returned_count: 0, remaining_count: 0, has_more: false });
  });

  test("group_by day returns ascending daily buckets", () => {
    const result = paginateGpsLogs(logs, { group_by: "day", limit: 500 });
    expect(result.buckets).toEqual([
      { key: "2026-08-01", count: 1 },
      { key: "2026-08-02", count: 3 },
      { key: "2026-08-03", count: 1 },
    ]);
    expect(result.total_count).toBe(5);
  });

  test("walks all points exactly once with limit=2 across a same-time run", () => {
    // 同一時刻3点のランがページ境界(limit=2)をまたぐ形。カーソルが位置を保てないと
    // 重複または取りこぼしが出る
    const seen = [];
    let cursor = undefined;
    for (let i = 0; i < 10; i++) {
      const page = paginateGpsLogs(logs, { limit: 2, cursor });
      seen.push(...page.gps_logs);
      if (!page.has_more) {
        expect(page.remaining_count).toBe(0);
        break;
      }
      expect(page.next_cursor).toBeDefined();
      cursor = page.next_cursor;
    }
    expect(seen).toHaveLength(5);
    // 全点がちょうど1回ずつ
    const keys = seen.map((p) => `${p.related_time}/${p.latitude}`);
    expect(new Set(keys).size).toBe(5);
    // 1ページ目だけ total_count
    const first = paginateGpsLogs(logs, { limit: 2 });
    expect(first.total_count).toBe(5);
    expect(first.returned_count + first.remaining_count).toBe(5);
    const second = paginateGpsLogs(logs, { limit: 2, cursor: first.next_cursor });
    expect(second).not.toHaveProperty("total_count");
  });

  test("count_only and group_by reject cursor", () => {
    expect(() => paginateGpsLogs(logs, { count_only: true, cursor: "x", limit: 1 })).toThrow(GkillApiError);
    expect(() => paginateGpsLogs(logs, { group_by: "day", cursor: "x", limit: 1 })).toThrow(GkillApiError);
  });

  // GPS 側も count_only の早期 return が group_by を黙って捨てていた（get_kyous と同じ順序の同じ穴）。
  test("count_only and group_by reject each other instead of silently dropping the buckets", () => {
    expect(() => paginateGpsLogs(logs, { count_only: true, group_by: "day", limit: 1 })).toThrow(
      /count_only[\s\S]*cannot be combined with group_by/,
    );
  });

  test("cursor roundtrip and invalid cursor", () => {
    const cursor = encodeGpsCursor("2026-08-02T12:00:00+09:00", 2);
    expect(decodeGpsCursor(cursor)).toEqual({ t: "2026-08-02T12:00:00+09:00", n: 2 });
    expect(() => decodeGpsCursor("not-base64-json")).toThrow(GkillApiError);
  });

  // 発行(paginateGpsLogs) と 受理(normalizeGpsArgs) の境界をまたぐ回帰テスト。
  //
  // 両者は別々にテストされていたが往復が一度も通されておらず、
  // normalizeGpsArgs が get_kyous 用の `{RFC3339}::{ID}` 検証をコピペしたまま
  // 残っていたために「説明文どおり next_cursor を verbatim で渡すと 100% 失敗する」
  // 状態が出荷されていた（2026-08-24 の実利用報告）。片側だけのテストでは検出できない。
  test("next_cursor survives normalizeGpsArgs and pages the whole list exactly once", () => {
    const period = { start_date: "2026-08-01", end_date: "2026-08-03" };
    const seen = [];
    let cursor = undefined;
    for (let i = 0; i < 10; i++) {
      // 実際の呼び出しと同じ経路: クライアントが渡した引数を normalize してから paginate する
      const normalized = normalizeGpsArgs(cursor === undefined ? { ...period, limit: 2 } : { ...period, limit: 2, cursor });
      const page = paginateGpsLogs(logs, normalized);
      seen.push(...page.gps_logs);
      if (!page.has_more) {
        break;
      }
      cursor = page.next_cursor;
    }
    expect(seen).toHaveLength(5);
    expect(new Set(seen.map((p) => `${p.related_time}/${p.latitude}`)).size).toBe(5);
  });
});

// ---------------------------------------------------------------------------
// rep名の絞り込み（純関数）
// ---------------------------------------------------------------------------
describe("paginateRepNames", () => {
  const names = ["Fitbit", "GoogleLocation", "Kmemo", "kmemo_backup", "Tag"];

  test("returns everything within limit and reports no truncation", () => {
    expect(paginateRepNames(names, { limit: 200 })).toEqual({
      rep_names: names,
      total_count: 5,
      returned_count: 5,
      truncated: false,
    });
  });

  test("contains matches case-insensitively", () => {
    const result = paginateRepNames(names, { contains: "KMEMO", limit: 200 });
    expect(result.rep_names).toEqual(["Kmemo", "kmemo_backup"]);
    expect(result.total_count).toBe(2);
  });

  // total_count は「絞り込み後・limit適用前」。limit前の件数を返さないと
  // 何件一致したのかが読めず、truncated の意味も決まらない
  test("total_count counts matches before limit", () => {
    const result = paginateRepNames(names, { contains: "kmemo", limit: 1 });
    expect(result.rep_names).toEqual(["Kmemo"]);
    expect(result.returned_count).toBe(1);
    expect(result.total_count).toBe(2);
    expect(result.truncated).toBe(true);
  });
});

// ---------------------------------------------------------------------------
// rep_infos
// ---------------------------------------------------------------------------
describe("handleReadToolCall — gkill_get_rep_infos", () => {
  // 本番では rep_infos[] だけで数百件になるのに、
  // 「正準値と対応表だけ欲しい」呼び出しが一番多かった（2026-08-24 の実利用レビュー）。
  test("fields projection can skip the large rep_infos list", async () => {
    const ctx = makeCtx(async () => ({
      rep_infos: [{ rep_name: "Kmemo", rep_type: "kmemo" }],
      canonical_rep_types: ["kmemo", "kc"],
      plugins: [{ rep_name: "ChatGPT", data_type: "chatgpt_conversation", plugin_name: "p" }],
      attached_data_reps: [{ rep_name: "Tag", data_kind: "tag" }],
    }));
    const payload = await handleReadToolCall(ctx, "gkill_get_rep_infos", {
      fields: ["canonical_rep_types", "plugins", "attached_data_reps"],
    });
    expect(Object.keys(payload).sort()).toEqual(["attached_data_reps", "canonical_rep_types", "plugins"]);
    expect(payload).not.toHaveProperty("rep_infos");
    // 絞り込みは Node 側。gkill には fields の受け口が無いので送らない
    expect(ctx.client.callApi).toHaveBeenCalledWith("/api/get_rep_infos_mcp", {}, true, "sid-1");
  });


  test("dispatches to /api/get_rep_infos_mcp and passes arrays through", async () => {
    const ctx = makeCtx(async () => ({
      rep_infos: [{ rep_name: "Kmemo", rep_type: "kmemo" }],
      canonical_rep_types: ["kmemo", "directory"],
      plugins: [{ rep_name: "ClaudeCode", data_type: "claude_code_turn", plugin_name: "gkill_plugin_claudecode" }],
      attached_data_reps: [
        { rep_name: "Tag", data_kind: "tag" },
        { rep_name: "GPSLog", data_kind: "gpslog" },
      ],
    }));
    const payload = await handleReadToolCall(ctx, "gkill_get_rep_infos", {});
    expect(ctx.client.callApi).toHaveBeenCalledWith("/api/get_rep_infos_mcp", {}, true, "sid-1");
    expect(payload.rep_infos).toHaveLength(1);
    expect(payload.canonical_rep_types).toContain("directory");
    expect(payload.plugins[0].data_type).toBe("claude_code_turn");
    // タグ等の書き込み先（query.reps へは渡せない）も素通しで返ること
    expect(payload.attached_data_reps).toEqual([
      { rep_name: "Tag", data_kind: "tag" },
      { rep_name: "GPSLog", data_kind: "gpslog" },
    ]);
  });

  test("isReadToolName covers the new tool", () => {
    expect(isReadToolName("gkill_get_rep_infos")).toBe(true);
  });

  // 行の絞り込み（2026-09-18 の実利用報告）。応答の形は本番の縮図:
  // Kyou rep は rep_type ごとに1行、Archived Git の rep 名は plugins[] に、
  // 歴代端末の Tag_ / Text_ / GPSLogs_ は attached_data_reps[] に並ぶ。
  const ROW_FILTER_RESPONSE = {
    rep_infos: [
      { rep_name: "Kmemo_pc", rep_type: "kmemo", use_to_write: true },
      { rep_name: "Kmemo_phone_2024", rep_type: "kmemo", use_to_write: false },
      { rep_name: "Files_pc", rep_type: "directory", use_to_write: true },
    ],
    canonical_rep_types: ["kmemo", "directory", "mi"],
    plugins: [
      { rep_name: "archived_git_alpha", data_type: "git_commit_log", plugin_name: "archived" },
      { rep_name: "archived_git_beta", data_type: "git_commit_log", plugin_name: "archived" },
    ],
    attached_data_reps: [
      { rep_name: "Tag_pc", data_kind: "tag", use_to_write: true },
      { rep_name: "Tag_phone_2024", data_kind: "tag", use_to_write: false },
      { rep_name: "Text_pc", data_kind: "text", use_to_write: true },
      { rep_name: "GPSLogs_phone_2024", data_kind: "gpslog", use_to_write: false },
    ],
  };

  test("writable_only keeps the write targets only and empties plugins", async () => {
    const ctx = makeCtx(async () => structuredClone(ROW_FILTER_RESPONSE));
    const payload = await handleReadToolCall(ctx, "gkill_get_rep_infos", { writable_only: true });
    expect(payload.rep_infos.map((rep) => rep.rep_name)).toEqual(["Kmemo_pc", "Files_pc"]);
    expect(payload.attached_data_reps.map((rep) => rep.rep_name)).toEqual(["Tag_pc", "Text_pc"]);
    expect(payload.plugins).toEqual([]);
    // 正準値の語彙は行絞り込みの影響を受けない
    expect(payload.canonical_rep_types).toEqual(["kmemo", "directory", "mi"]);
  });

  test("『gkill_add_tag はどこへ書くか』は writable_only + data_kinds:[tag] で1行になる", async () => {
    const ctx = makeCtx(async () => structuredClone(ROW_FILTER_RESPONSE));
    const payload = await handleReadToolCall(ctx, "gkill_get_rep_infos", {
      writable_only: true,
      data_kinds: ["tag"],
      fields: ["attached_data_reps"],
    });
    expect(payload).toEqual({ attached_data_reps: [{ rep_name: "Tag_pc", data_kind: "tag", use_to_write: true }] });
  });

  test("rep_types narrows rep_infos and is checked against canonical_rep_types", async () => {
    const ctx = makeCtx(async () => structuredClone(ROW_FILTER_RESPONSE));
    const payload = await handleReadToolCall(ctx, "gkill_get_rep_infos", { rep_types: ["directory"] });
    expect(payload.rep_infos.map((rep) => rep.rep_name)).toEqual(["Files_pc"]);
    // 他の配列には効かない
    expect(payload.plugins).toHaveLength(2);
    expect(payload.attached_data_reps).toHaveLength(4);
    await expect(handleReadToolCall(ctx, "gkill_get_rep_infos", { rep_types: ["Kmemo"] })).rejects.toThrow(
      /rep_types[\s\S]*canonical rep types: kmemo, directory, mi/,
    );
  });

  test("rep_names is exact and case-sensitive across all three lists", async () => {
    const ctx = makeCtx(async () => structuredClone(ROW_FILTER_RESPONSE));
    const payload = await handleReadToolCall(ctx, "gkill_get_rep_infos", {
      rep_names: ["Kmemo_pc", "archived_git_beta", "Text_pc", "tag_pc"],
    });
    expect(payload.rep_infos.map((rep) => rep.rep_name)).toEqual(["Kmemo_pc"]);
    expect(payload.plugins.map((rep) => rep.rep_name)).toEqual(["archived_git_beta"]);
    expect(payload.attached_data_reps.map((rep) => rep.rep_name)).toEqual(["Text_pc"]);
  });

  test("contains is a case-insensitive substring across all three lists", async () => {
    const ctx = makeCtx(async () => structuredClone(ROW_FILTER_RESPONSE));
    const payload = await handleReadToolCall(ctx, "gkill_get_rep_infos", { contains: "PHONE_2024" });
    expect(payload.rep_infos.map((rep) => rep.rep_name)).toEqual(["Kmemo_phone_2024"]);
    expect(payload.plugins).toEqual([]);
    expect(payload.attached_data_reps.map((rep) => rep.rep_name)).toEqual(["Tag_phone_2024", "GPSLogs_phone_2024"]);
  });

  test("applyFileLinks leaves rep_infos payload unchanged (no file-link mint)", () => {
    // applyFileLinks は rep_name+file_name の同居で idf とみなしてトークンを鋳造する。
    // rep_infos / attached_data_reps の行は rep_name を持つが file_name を持たないので、
    // 不変であること（サーバ側が file_name 系キーを同居させない契約の防御線）。
    const payload = {
      rep_infos: [{ rep_name: "Kmemo", rep_type: "kmemo" }],
      canonical_rep_types: ["kmemo"],
      plugins: [{ rep_name: "ClaudeCode", data_type: "claude_code_turn", plugin_name: "p" }],
      attached_data_reps: [{ rep_name: "Tag", data_kind: "tag" }],
    };
    const before = JSON.stringify(payload);
    const store = new FileLinkStore();
    applyFileLinks(payload, { publicBaseUrl: "https://example.com", store }, "sid");
    expect(JSON.stringify(payload)).toBe(before);
  });
});

// ---------------------------------------------------------------------------
// gkill_get_kyou_history — 削除済みと過去版を読む唯一の経路
// ---------------------------------------------------------------------------
describe("handleReadToolCall — gkill_get_kyou_history", () => {
  test("uses the per-type endpoint, never the type-agnostic /api/get_kyou", () => {
    // 型非依存の Repositories.GetKyouHistoriesByRepName は UnWrap() で
    // キャッシュrepを丸ごとバイパスする（11rep→約940rep・実測20.7秒）
    const ctx = makeCtx(async () => ({ kmemo_histories: [{ id: "k1", is_deleted: false, update_time: "2026-01-02T00:00:00+09:00" }] }));
    return handleReadToolCall(ctx, "gkill_get_kyou_history", { id: "k1", data_type: "kmemo" }).then(() => {
      expect(ctx.client.callApi.mock.calls[0][0]).toBe("/api/get_kmemo");
      expect(ctx.client.callApi.mock.calls[0][0]).not.toBe("/api/get_kyou");
    });
  });

  test("returns every version newest first and flags a deleted latest version", async () => {
    const ctx = makeCtx(async () => ({
      kmemo_histories: [
        { id: "k1", content: "gone", is_deleted: true, update_time: "2026-01-03T00:00:00+09:00" },
        { id: "k1", content: "second", is_deleted: false, update_time: "2026-01-02T00:00:00+09:00" },
        { id: "k1", content: "first", is_deleted: false, update_time: "2026-01-01T00:00:00+09:00" },
      ],
    }));
    const result = await handleReadToolCall(ctx, "gkill_get_kyou_history", { id: "k1", data_type: "kmemo" });

    expect(result.latest_is_deleted).toBe(true);
    expect(result.version_count).toBe(3);
    expect(result.returned_count).toBe(3);
    expect(result.has_more).toBe(false);
    expect(result.versions[0].content).toBe("gone");
    expect(result.versions[2].content).toBe("first");
  });

  test("caps the versions by limit and reports has_more", async () => {
    // 履歴は編集のたびに1件伸びるので無制限には返さない
    const ctx = makeCtx(async () => ({
      kmemo_histories: Array.from({ length: 5 }, (_unused, i) => ({ id: "k1", is_deleted: false, update_time: `2026-01-0${i + 1}T00:00:00+09:00` })),
    }));
    const result = await handleReadToolCall(ctx, "gkill_get_kyou_history", { id: "k1", data_type: "kmemo", limit: 2 });

    expect(result.returned_count).toBe(2);
    expect(result.version_count).toBe(5);
    expect(result.has_more).toBe(true);
  });

  test("throws when the id has no history at all", async () => {
    const ctx = makeCtx(async () => ({ kmemo_histories: [] }));
    await expect(handleReadToolCall(ctx, "gkill_get_kyou_history", { id: "nope", data_type: "kmemo" }))
      .rejects.toThrow(/Entity not found/);
  });

  test("summary names the deleted state so it is visible before reading the JSON", () => {
    const summary = summarizeReadToolPayload("gkill_get_kyou_history", {
      version_count: 3, returned_count: 3, has_more: false, latest_is_deleted: true,
    });
    expect(summary).toContain("3 of 3");
    expect(summary).toContain("DELETED");
  });
});

// ---------------------------------------------------------------------------
// gkill_get_idf_file — /files/ クエリ組み立てと thumb エコー
// ---------------------------------------------------------------------------
describe("handleReadToolCall — gkill_get_idf_file のクエリ組み立て", () => {
  // ?is_video=true&thumb=WxH の順序とエンコードを決める唯一の箇所
  // (payload.mjs の file_url 注入 / http-transport.mjs の /files/ 配信 /
  //  クライアントの build_media_url と同じ形であること)。
  function makeFileCtx(file) {
    const ctx = makeCtx();
    ctx.client.fetchFile.mockResolvedValue(file);
    return ctx;
  }

  test("builds ?is_video=true&thumb=WxH in that order", async () => {
    const ctx = makeFileCtx({ buffer: Buffer.from("frame"), contentType: "image/jpeg" });
    await handleReadToolCall(ctx, "gkill_get_idf_file", {
      rep_name: "Video",
      file_name: "clip.mp4",
      is_video: true,
      thumb: "640x480",
    });
    expect(ctx.client.fetchFile).toHaveBeenCalledWith(
      "/files/Video/clip.mp4?is_video=true&thumb=640x480",
      "sid-1",
    );
  });

  test("thumb alone appends only ?thumb=WxH", async () => {
    const ctx = makeFileCtx({ buffer: Buffer.from("img"), contentType: "image/png" });
    await handleReadToolCall(ctx, "gkill_get_idf_file", {
      rep_name: "Photo",
      file_name: "p.png",
      thumb: "320x240",
    });
    expect(ctx.client.fetchFile).toHaveBeenCalledWith("/files/Photo/p.png?thumb=320x240", "sid-1");
  });

  test("no thumb and no is_video appends no query string", async () => {
    const ctx = makeFileCtx({ buffer: Buffer.from("img"), contentType: "image/png" });
    await handleReadToolCall(ctx, "gkill_get_idf_file", { rep_name: "Photo", file_name: "p.png" });
    expect(ctx.client.fetchFile).toHaveBeenCalledWith("/files/Photo/p.png", "sid-1");
  });

  test("encodes rep_name and each file_name segment, keeping / separators", async () => {
    const ctx = makeFileCtx({ buffer: Buffer.from("img"), contentType: "image/png" });
    await handleReadToolCall(ctx, "gkill_get_idf_file", {
      rep_name: "My Photos",
      file_name: "2026 08/pic 1.png",
    });
    expect(ctx.client.fetchFile).toHaveBeenCalledWith(
      "/files/My%20Photos/2026%2008/pic%201.png",
      "sid-1",
    );
  });

  test("echoes thumb in the payload only when the fetch was downscaled", async () => {
    // thumb エコーは「縮小して取った」ことの唯一の印。原寸と取り違えないための防御線
    const downscaled = makeFileCtx({ buffer: Buffer.from("small"), contentType: "image/jpeg" });
    const withThumb = await handleReadToolCall(downscaled, "gkill_get_idf_file", {
      rep_name: "Photo",
      file_name: "p.png",
      thumb: "640x480",
    });
    expect(withThumb.thumb).toBe("640x480");

    const original = makeFileCtx({ buffer: Buffer.from("orig"), contentType: "image/jpeg" });
    const withoutThumb = await handleReadToolCall(original, "gkill_get_idf_file", {
      rep_name: "Photo",
      file_name: "p.png",
    });
    expect(withoutThumb).not.toHaveProperty("thumb");
  });
});

// ---------------------------------------------------------------------------
// gkill_get_idf_file — サイズ上限超過の案内
// ---------------------------------------------------------------------------
describe("gkill_get_idf_file のサイズ上限超過メッセージ", () => {
  // payload.mjs の applyFileLinks は is_image のときだけ file_url_full を注入する。
  // 非画像へ file_url_full を案内すると、存在しないフィールドを探させてしまう
  const hugeBuffer = Buffer.alloc(MAX_IDF_FILE_BYTES + 1);

  test("advises file_url_full for an oversized image", async () => {
    const ctx = makeCtx();
    ctx.client.fetchFile.mockResolvedValue({ buffer: hugeBuffer, contentType: "image/png" });
    const error = await handleReadToolCall(ctx, "gkill_get_idf_file", {
      rep_name: "Photo",
      file_name: "big.png",
    }).catch((e) => e);
    expect(error).toBeInstanceOf(GkillApiError);
    expect(error.message).toContain("file_url_full");
  });

  test("advises file_url (not file_url_full) for an oversized non-image", async () => {
    const ctx = makeCtx();
    ctx.client.fetchFile.mockResolvedValue({ buffer: hugeBuffer, contentType: "video/mp4" });
    const error = await handleReadToolCall(ctx, "gkill_get_idf_file", {
      rep_name: "Video",
      file_name: "big.mp4",
    }).catch((e) => e);
    expect(error).toBeInstanceOf(GkillApiError);
    expect(error.message).toContain("file_url");
    expect(error.message).not.toContain("file_url_full");
  });
});

describe("gkill_get_idf_file の 404 (2026-08-24 再監査 P-13)", () => {
  test("turns the raw HTTP 404 into something that names the likely cause", async () => {
    // 「HTTP 404 fetching file /files/NoSuchRep/x.png」だけだと、rep 名が悪いのか
    // ファイル名が悪いのか、そもそも消えたのかが読めない
    const ctx = {
      client: {
        fetchFile: vi.fn(async () => {
          throw new GkillApiError("HTTP 404 fetching file /files/NoSuchRep/x.png.", { status: 404 });
        }),
      },
      sid: "sid-1",
    };
    await expect(
      handleReadToolCall(ctx, "gkill_get_idf_file", { rep_name: "NoSuchRep", file_name: "x.png" }),
    ).rejects.toThrow(/gkill_get_rep_infos/);
  });

  test("passes other failures through untouched", async () => {
    const ctx = {
      client: {
        fetchFile: vi.fn(async () => {
          throw new GkillApiError("HTTP 500 fetching file /files/r/x.png.", { status: 500 });
        }),
      },
      sid: "sid-1",
    };
    await expect(
      handleReadToolCall(ctx, "gkill_get_idf_file", { rep_name: "r", file_name: "x.png" }),
    ).rejects.toThrow(/HTTP 500/);
  });
});

describe("0件の要約 (2026-08-24 再監査 Q-05)", () => {
  // count_only の応答も通常検索の0件も kyous:[] なので payload からは区別できない。
  // 「Counted 0 entries.」だと、count_only を指定していない呼び出し側に
  // 「集計モードで返ってきた」と読めてしまう。
  test("an empty ordinary result does not claim to have counted", () => {
    expect(
      summarizeReadToolPayload("gkill_get_kyous", {
        kyous: [],
        total_count: 0,
        returned_count: 0,
        remaining_count: 0,
        has_more: false,
      }),
    ).toBe("No entries matched.");
  });

  test("count_only with matches still reports the count", () => {
    expect(
      summarizeReadToolPayload("gkill_get_kyous", {
        kyous: [],
        total_count: 12,
        returned_count: 0,
        remaining_count: 0,
        has_more: false,
      }),
    ).toBe("Counted 12 entries.");
  });

  test("gps logs follow the same rule", () => {
    expect(
      summarizeReadToolPayload("gkill_get_gps_log", { gps_logs: [], total_count: 0, has_more: false }),
    ).toBe("No GPS points matched.");
    expect(
      summarizeReadToolPayload("gkill_get_gps_log", { gps_logs: [], total_count: 5, has_more: false }),
    ).toBe("Counted 5 GPS points.");
  });
});

// ---------------------------------------------------------------------------
// トップレベル plugins[] の素通し (2026-08-30 レビュー P1)
//
// Go はページに現れたプラグインの説明を rep 名ごと1回だけ応答トップレベルの plugins[]
// で返し、各 Kyou の payload には rep_name / plugin_name しか載せない設計
// (説明の焼き込み排除)。ここがコピーを落とすと、ツール説明が約束しているのに
// 「各 Kyou にもトップレベルにも説明が無い」状態になる。
// ---------------------------------------------------------------------------
describe("handleReadToolCall — top-level plugins[] passthrough", () => {
  const PLUGINS = [
    { rep_name: "ExamplePluginRep", plugin_name: "example_plugin", description: "プラグインの説明文" },
  ];

  function pageResponse(extra = {}) {
    return {
      kyous: [{ id: "k1", rep_name: "ExamplePluginRep", data_type: "example_type" }],
      total_count: 1,
      returned_count: 1,
      remaining_count: 0,
      has_more: false,
      ...extra,
    };
  }

  test("copies plugins[] from the gkill response verbatim", async () => {
    const ctx = makeCtx(async () => pageResponse({ plugins: PLUGINS }));
    const payload = await handleReadToolCall(ctx, "gkill_get_kyous", {});
    expect(payload.plugins).toEqual(PLUGINS);
  });

  test("omits plugins when the page carries none (ordinary kyous only)", async () => {
    const ctx = makeCtx(async () => pageResponse());
    const payload = await handleReadToolCall(ctx, "gkill_get_kyous", {});
    expect(payload.plugins).toBeUndefined();
  });

  test("omits plugins when gkill returns an empty array", async () => {
    const ctx = makeCtx(async () => pageResponse({ plugins: [] }));
    const payload = await handleReadToolCall(ctx, "gkill_get_kyous", {});
    expect(payload.plugins).toBeUndefined();
  });

  test("count_only passes plugins through when the response carries them (copy, not a filter)", async () => {
    // Node 側の plugins は「来たら載せる」条件付きコピーで、count_only かどうかで
    // 落としたりしない (経路は通常検索と同じ1つの組み立て)。
    // 「count_only では plugins が来ない」のは Go 側の性質であって、Node で検証できる
    // 命題ではない — 以前のこのテストはモックに plugins を入れておらず、
    // 隣の「無ければ載せない」テストと同じ分岐しか通っていなかった。
    const ctx = makeCtx(async () => ({
      kyous: [],
      total_count: 3,
      returned_count: 0,
      remaining_count: 0,
      has_more: false,
      plugins: PLUGINS,
    }));
    const payload = await handleReadToolCall(ctx, "gkill_get_kyous", { count_only: true });
    expect(payload.plugins).toEqual(PLUGINS);
  });
});

// ---------------------------------------------------------------------------
// 古いツールスキーマを掴んだクライアントへの警告
//
// MCPのツール一覧はクライアントのセッション寿命で固定されるため、サーバを直しても
// 生きているセッションには届かない。その結果「もう直っている機能が永久に見えない」
// が起きる（2026-08-24 の実利用報告では、指摘9件のうち4件がこれだった）。
// 警告は**古さが証明できるときだけ**出す。推測で出すと常時ノイズになる。
// ---------------------------------------------------------------------------
describe("handleReadToolCall — stale tool schema warning", () => {
  test("warns when a non-string argument arrived as a JSON string", async () => {
    const ctx = makeCtx(async () => ({ kyous: [], returned_count: 0, remaining_count: 0, has_more: false }));
    // 旧スキーマのクライアントは data_types を知らないので正規JSON文字列として送ってくる
    const payload = await handleReadToolCall(ctx, "gkill_get_kyous", { data_types: '["nlog"]' });
    expect(payload.warnings).toHaveLength(1);
    expect(payload.warnings[0]).toContain("tool schema snapshot looks stale");
    expect(payload.warnings[0]).toContain("data_types");
  });

  test("warns when deprecated arguments are sent", async () => {
    const ctx = makeCtx(async () => ({ kyous: [], returned_count: 0, remaining_count: 0, has_more: false }));
    const payload = await handleReadToolCall(ctx, "gkill_get_kyous", {
      include_id: true,
      query: { use_tags: true },
    });
    expect(payload.warnings[0]).toContain("include_id");
    expect(payload.warnings[0]).toContain("query.use_tags");
  });

  // 誤警告を出さないことが本体と同じくらい重要。現行スキーマどおりの呼び出しで
  // 警告が付くと、警告そのものが読まれなくなる
  test("does not warn for a current-schema call", async () => {
    const ctx = makeCtx(async () => ({ kyous: [], returned_count: 0, remaining_count: 0, has_more: false }));
    const payload = await handleReadToolCall(ctx, "gkill_get_kyous", { data_types: ["nlog"], count_only: false });
    expect(payload.warnings).toBeUndefined();
  });

  test("keeps warnings from gkill and appends to them", async () => {
    const ctx = makeCtx(async () => ({
      kyous: [],
      returned_count: 0,
      remaining_count: 0,
      has_more: false,
      warnings: ['unknown rep "GoogleLocation" in query.reps: plugin emits no kyou'],
    }));
    const payload = await handleReadToolCall(ctx, "gkill_get_kyous", { count_only: "true" });
    expect(payload.warnings).toHaveLength(2);
    expect(payload.warnings[0]).toContain("GoogleLocation");
    expect(payload.warnings[1]).toContain("tool schema snapshot looks stale");
  });

  test("the one-line summary carries the stale marker too", () => {
    const summary = summarizeReadToolPayload("gkill_get_all_rep_names", {
      rep_names: ["Fitbit"],
      total_count: 1,
      returned_count: 1,
      truncated: false,
      warnings: ["this MCP client's tool schema snapshot looks stale (…)"],
    });
    expect(summary).toContain("reconnect the MCP client");
  });
});

// ---------------------------------------------------------------------------
// rep名一覧（ディスパッチ）
// ---------------------------------------------------------------------------
describe("handleReadToolCall — gkill_get_all_rep_names", () => {
  test("filters node-side and does not forward contains/limit to gkill", async () => {
    const ctx = makeCtx(async () => ({ rep_names: ["Fitbit", "GoogleLocation", "Kmemo"] }));
    const payload = await handleReadToolCall(ctx, "gkill_get_all_rep_names", { contains: "o", limit: 1 });
    expect(payload).toEqual({
      rep_names: ["GoogleLocation"],
      total_count: 2,
      returned_count: 1,
      truncated: true,
    });
    // "o" は GoogleLocation と Kmemo に一致する（Fitbit には無い）
    // gkill 側には絞り込みの口が無いので送らない（送ると未知キーで弾かれる）
    expect(ctx.client.callApi).toHaveBeenCalledWith("/api/get_all_rep_names", {}, true, "sid-1");
  });
});
