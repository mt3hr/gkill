/**
 * lib/write-handlers.mjs のディスパッチのテスト。
 *
 * write / readwrite の2サーバが共有する1本の実装なので、ここが唯一の正本。
 * サーバ経由の統合は write-server.test.mjs / readwrite-server.test.mjs が見る。
 * 読み取り側の同じ形は read-handlers.test.mjs。
 *
 * 見張っているもの:
 * - add はサーバへ1回、update / delete は「現在値を取ってから送る」2回
 * - create_app / update_app が ctx.appName（サーバ種別）で埋まる
 * - update は patch セマンティクス（送っていない項目が現在値のまま残る）
 * - delete は is_deleted=true を立てた現在値を update エンドポイントへ送る
 */

import { describe, test, expect, vi } from "vitest";

import { handleWriteToolCall, isWriteToolName } from "../lib/write-handlers.mjs";

function makeCtx(callApiImpl) {
  return {
    client: {
      callApi: vi.fn(callApiImpl ?? (async () => ({ errors: [], messages: [] }))),
    },
    sid: "sid-1",
    userId: "testuser",
    appName: "gkill_mcp_readwrite",
  };
}

describe("handleWriteToolCall — add tools", () => {
  test("gkill_add_kmemo posts to /api/add_kmemo with server-side metadata", async () => {
    const ctx = makeCtx(async () => ({ added_kmemo: { id: "k1" }, added_kyou: { id: "k1" } }));
    const result = await handleWriteToolCall(ctx, "gkill_add_kmemo", { content: "hello" });

    const [pathname, body] = ctx.client.callApi.mock.calls[0];
    expect(pathname).toBe("/api/add_kmemo");
    expect(body.kmemo.content).toBe("hello");
    expect(body.want_response_kyou).toBe(true);
    expect(result.added_kmemo.id).toBe("k1");
  });

  test("create_app comes from ctx.appName so each server labels its own writes", async () => {
    // write サーバと readwrite サーバで create_app が違う。ここを取り違えると
    // 「どのサーバが書いたか」が記録から分からなくなる
    const ctx = makeCtx(async () => ({ added_kmemo: { id: "k1" } }));
    ctx.appName = "gkill_mcp_write";
    await handleWriteToolCall(ctx, "gkill_add_kmemo", { content: "hello" });

    const [, body] = ctx.client.callApi.mock.calls[0];
    expect(body.kmemo.create_app).toBe("gkill_mcp_write");
    expect(body.kmemo.update_app).toBe("gkill_mcp_write");
    expect(body.kmemo.create_device).toBe("mcp");
    expect(body.kmemo.create_user).toBe("testuser");
  });

  test("every add tool posts to its own /api/add_* endpoint", async () => {
    const cases = [
      ["gkill_add_kmemo", { content: "x" }, "/api/add_kmemo"],
      ["gkill_add_urlog", { url: "https://example.com" }, "/api/add_urlog"],
      ["gkill_add_nlog", { title: "x", amount: -1 }, "/api/add_nlog"],
      ["gkill_add_lantana", { mood: 5 }, "/api/add_lantana"],
      ["gkill_add_timeis", { title: "x" }, "/api/add_timeis"],
      ["gkill_add_mi", { title: "x", board_name: "b" }, "/api/add_mi"],
      ["gkill_add_kc", { title: "x", num_value: 1 }, "/api/add_kc"],
      ["gkill_add_tag", { tag: "t", target_id: "id1" }, "/api/add_tag"],
      ["gkill_add_text", { text: "t", target_id: "id1" }, "/api/add_text"],
    ];
    for (const [name, args, endpoint] of cases) {
      const ctx = makeCtx();
      await handleWriteToolCall(ctx, name, args);
      expect(ctx.client.callApi.mock.calls[0][0]).toBe(endpoint);
    }
  });

  test("gkill_submit_kftl posts the raw text", async () => {
    const ctx = makeCtx(async () => ({ messages: [{ message: "ok" }] }));
    await handleWriteToolCall(ctx, "gkill_submit_kftl", { kftl_text: "memo" });

    const [pathname, body] = ctx.client.callApi.mock.calls[0];
    expect(pathname).toBe("/api/submit_kftl_text");
    expect(body.kftl_text).toBe("memo");
  });
});

describe("handleWriteToolCall — update tools", () => {
  test("gkill_update_kmemo reads the current version before writing", async () => {
    const ctx = makeCtx();
    ctx.client.callApi
      .mockResolvedValueOnce({ kmemo_histories: [{ id: "k1", content: "old", related_time: "2026-01-01T00:00:00+09:00" }] })
      .mockResolvedValueOnce({ updated_kyou: { id: "k1" } });

    const result = await handleWriteToolCall(ctx, "gkill_update_kmemo", { id: "k1", content: "new" });

    expect(ctx.client.callApi.mock.calls[0][0]).toBe("/api/get_kmemo");
    expect(ctx.client.callApi.mock.calls[1][0]).toBe("/api/update_kmemo");
    expect(result.updated_kmemo.content).toBe("new");
  });

  test("fields that were not sent keep their current value (patch semantics)", async () => {
    // スキーマの required が過剰でも実装は patch。ここが崩れると
    // 「一部だけ直したつもりが他の項目を空で上書き」になる
    const ctx = makeCtx();
    ctx.client.callApi
      .mockResolvedValueOnce({ kmemo_histories: [{ id: "k1", content: "keep me", related_time: "2026-01-01T00:00:00+09:00" }] })
      .mockResolvedValueOnce({ updated_kyou: { id: "k1" } });

    const result = await handleWriteToolCall(ctx, "gkill_update_kmemo", {
      id: "k1",
      related_time: "2026-02-02T10:00:00+09:00",
    });

    expect(result.updated_kmemo.content).toBe("keep me");
    expect(result.updated_kmemo.related_time).toBe("2026-02-02T10:00:00+09:00");
  });

  test("throws when the entity has no history", async () => {
    const ctx = makeCtx(async () => ({ kmemo_histories: [] }));
    await expect(handleWriteToolCall(ctx, "gkill_update_kmemo", { id: "missing", content: "x" }))
      .rejects.toThrow(/not found/i);
  });
});

describe("handleWriteToolCall — delete", () => {
  test("gkill_delete_kyou sets is_deleted on the current version and sends it to update", async () => {
    const ctx = makeCtx();
    ctx.client.callApi
      .mockResolvedValueOnce({ kmemo_histories: [{ id: "k1", content: "bye", is_deleted: false }] })
      .mockResolvedValueOnce({ updated_kmemo: { id: "k1" }, updated_kyou: { id: "k1" } });

    const result = await handleWriteToolCall(ctx, "gkill_delete_kyou", { id: "k1", data_type: "kmemo" });

    expect(ctx.client.callApi.mock.calls[0][0]).toBe("/api/get_kmemo");
    expect(ctx.client.callApi.mock.calls[1][0]).toBe("/api/update_kmemo");
    expect(ctx.client.callApi.mock.calls[1][1].kmemo.is_deleted).toBe(true);
    expect(result.updated_kmemo.is_deleted).toBe(true);
  });

  test("throws Entity not found when the id has no history", async () => {
    const ctx = makeCtx(async () => ({ kmemo_histories: [] }));
    await expect(handleWriteToolCall(ctx, "gkill_delete_kyou", { id: "nope", data_type: "kmemo" }))
      .rejects.toThrow(/Entity not found/);
  });
});

describe("handleWriteToolCall — unknown tools", () => {
  test("throws for a tool it does not own", async () => {
    const ctx = makeCtx();
    await expect(handleWriteToolCall(ctx, "unknown_tool", {})).rejects.toThrow(/Unknown tool/);
  });

  test("isWriteToolName gates the dispatch", () => {
    expect(isWriteToolName("gkill_add_kmemo")).toBe(true);
    expect(isWriteToolName("unknown_tool")).toBe(false);
  });
});
