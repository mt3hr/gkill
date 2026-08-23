/**
 * lib/read-handlers.mjs の v2 機能のテスト。
 * - get_kyous v2: 新パラメータの転送・応答の素通し（total_count はカーソルページで作らない）
 * - application_config: fields 射影 + UI状態キーの strip
 * - GPS: Node側ページング（複合カーソル・同一時刻ラン跨ぎ・count_only・日別バケット）
 * - rep_infos: ディスパッチと applyFileLinks 不変（file-link トークンの誤発行防止）
 */

import { describe, test, expect, vi } from "vitest";

import {
  handleReadToolCall,
  isReadToolName,
  stripAppConfigUiState,
  paginateGpsLogs,
  encodeGpsCursor,
  decodeGpsCursor,
  summarizeReadToolPayload,
} from "../lib/read-handlers.mjs";
import { applyFileLinks } from "../lib/payload.mjs";
import { FileLinkStore } from "../lib/file-link-store.mjs";
import { GkillApiError } from "../lib/errors.mjs";

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
    await handleReadToolCall(ctx, "gkill_get_kyous", {
      count_only: true,
      group_by: "month",
      data_types: ["nlog"],
      num_min: 100,
      num_max: 500,
      idf_kinds: ["image"],
      include_file_size: true,
      include_id: true, // deprecated: 受理はするが転送しない
    });
    const [pathname, body] = ctx.client.callApi.mock.calls[0];
    expect(pathname).toBe("/api/get_kyous_mcp");
    expect(body.count_only).toBe(true);
    expect(body.group_by).toBe("month");
    expect(body.data_types).toEqual(["nlog"]);
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

  test("cursor roundtrip and invalid cursor", () => {
    const cursor = encodeGpsCursor("2026-08-02T12:00:00+09:00", 2);
    expect(decodeGpsCursor(cursor)).toEqual({ t: "2026-08-02T12:00:00+09:00", n: 2 });
    expect(() => decodeGpsCursor("not-base64-json")).toThrow(GkillApiError);
  });
});

// ---------------------------------------------------------------------------
// rep_infos
// ---------------------------------------------------------------------------
describe("handleReadToolCall — gkill_get_rep_infos", () => {
  test("dispatches to /api/get_rep_infos_mcp and passes arrays through", async () => {
    const ctx = makeCtx(async () => ({
      rep_infos: [{ rep_name: "Kmemo", rep_type: "kmemo" }],
      canonical_rep_types: ["kmemo", "directory"],
      plugins: [{ rep_name: "ClaudeCode", data_type: "claude_code_turn", plugin_name: "gkill_plugin_claudecode" }],
    }));
    const payload = await handleReadToolCall(ctx, "gkill_get_rep_infos", {});
    expect(ctx.client.callApi).toHaveBeenCalledWith("/api/get_rep_infos_mcp", {}, true, "sid-1");
    expect(payload.rep_infos).toHaveLength(1);
    expect(payload.canonical_rep_types).toContain("directory");
    expect(payload.plugins[0].data_type).toBe("claude_code_turn");
  });

  test("isReadToolName covers the new tool", () => {
    expect(isReadToolName("gkill_get_rep_infos")).toBe(true);
  });

  test("applyFileLinks leaves rep_infos payload unchanged (no file-link mint)", () => {
    // applyFileLinks は rep_name+file_name の同居で idf とみなしてトークンを鋳造する。
    // rep_infos の応答は rep_name を持つが file_name を持たないので、不変であること
    // （サーバ側が file_name 系キーを同居させない契約の防御線）。
    const payload = {
      rep_infos: [{ rep_name: "Kmemo", rep_type: "kmemo" }],
      canonical_rep_types: ["kmemo"],
      plugins: [{ rep_name: "ClaudeCode", data_type: "claude_code_turn", plugin_name: "p" }],
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
