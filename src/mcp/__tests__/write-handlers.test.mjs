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

import { handleWriteToolCall, isWriteToolName, summarizeWriteToolPayload } from "../lib/write-handlers.mjs";

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

  test("gkill_add_mi falls back to the account default board when board_name is omitted", async () => {
    const ctx = makeCtx();
    ctx.client.callApi
      .mockResolvedValueOnce({ application_config: { mi_default_board: "Inbox" } })
      .mockResolvedValueOnce({ added_mi: { id: "m1" } });

    await handleWriteToolCall(ctx, "gkill_add_mi", { title: "buy milk" });

    expect(ctx.client.callApi.mock.calls[0][0]).toBe("/api/get_application_config");
    expect(ctx.client.callApi.mock.calls[1][0]).toBe("/api/add_mi");
    expect(ctx.client.callApi.mock.calls[1][1].mi.board_name).toBe("Inbox");
  });

  test("gkill_add_mi does not look up the config when board_name is given", async () => {
    const ctx = makeCtx(async () => ({ added_mi: { id: "m1" } }));
    await handleWriteToolCall(ctx, "gkill_add_mi", { title: "buy milk", board_name: "errands" });

    expect(ctx.client.callApi).toHaveBeenCalledTimes(1);
    expect(ctx.client.callApi.mock.calls[0][0]).toBe("/api/add_mi");
    expect(ctx.client.callApi.mock.calls[0][1].mi.board_name).toBe("errands");
  });

  test("gkill_add_mi still creates the task when the config lookup fails", async () => {
    // 既定板が引けないことを理由にタスク作成そのものを失敗させない
    const ctx = makeCtx();
    ctx.client.callApi
      .mockRejectedValueOnce(new Error("config unavailable"))
      .mockResolvedValueOnce({ added_mi: { id: "m1" } });

    await handleWriteToolCall(ctx, "gkill_add_mi", { title: "buy milk" });

    expect(ctx.client.callApi.mock.calls[1][1].mi.board_name).toBe("Inbox");
  });

  test("gkill_add_urlog drops the image base64 from the response", async () => {
    // 1件2KB前後の浪費。検索結果の urlog payload には元から載っていない
    const ctx = makeCtx(async () => ({
      added_urlog: { id: "u1", url: "https://example.com", title: "T", favicon_image: "AAAA", thumbnail_image: "BBBB" },
      added_kyou: { id: "u1" },
    }));
    const result = await handleWriteToolCall(ctx, "gkill_add_urlog", { url: "https://example.com" });

    expect(result.added_urlog.id).toBe("u1");
    expect(result.added_urlog.title).toBe("T");
    expect(result.added_urlog).not.toHaveProperty("favicon_image");
    expect(result.added_urlog).not.toHaveProperty("thumbnail_image");
  });

  test("gkill_add_tag sends related_time and does not claim a parent kyou", async () => {
    // related_time を送らないと Go のゼロ値 0001-01-01 が保存される。
    // AddTagResponse に added_kyou は無いので、返しても常に null だった
    const ctx = makeCtx(async () => ({ added_tag: { id: "t1" } }));
    const result = await handleWriteToolCall(ctx, "gkill_add_tag", { tag: "x", target_id: "id1" });

    const [, body] = ctx.client.callApi.mock.calls[0];
    expect(body.tag.related_time).toEqual(expect.any(String));
    expect(body.tag.related_time.startsWith("0001-")).toBe(false);
    expect(result).not.toHaveProperty("added_kyou");
  });

  test("gkill_add_text sends related_time and does not claim a parent kyou", async () => {
    const ctx = makeCtx(async () => ({ added_text: { id: "x1" } }));
    const result = await handleWriteToolCall(ctx, "gkill_add_text", { text: "note", target_id: "id1" });

    const [, body] = ctx.client.callApi.mock.calls[0];
    expect(body.text.related_time.startsWith("0001-")).toBe(false);
    expect(result).not.toHaveProperty("added_kyou");
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

// ---------------------------------------------------------------------------
// gkill_restore_kyou — 削除の取り消し
// ---------------------------------------------------------------------------
describe("handleWriteToolCall — restore", () => {
  test("clears is_deleted on the current version and sends it to update", async () => {
    const ctx = makeCtx();
    ctx.client.callApi
      .mockResolvedValueOnce({ kmemo_histories: [{ id: "k1", content: "back", is_deleted: true, update_time: "2026-01-01T00:00:00+09:00" }] })
      .mockResolvedValueOnce({ updated_kyou: { id: "k1" } });

    const result = await handleWriteToolCall(ctx, "gkill_restore_kyou", { id: "k1", data_type: "kmemo" });

    expect(ctx.client.callApi.mock.calls[0][0]).toBe("/api/get_kmemo");
    expect(ctx.client.callApi.mock.calls[1][0]).toBe("/api/update_kmemo");
    expect(ctx.client.callApi.mock.calls[1][1].kmemo.is_deleted).toBe(false);
    expect(result.restored_kmemo.is_deleted).toBe(false);
  });

  test("refuses to append a pointless version when the entry is already active", async () => {
    const ctx = makeCtx(async () => ({ kmemo_histories: [{ id: "k1", is_deleted: false, update_time: "2026-01-01T00:00:00+09:00" }] }));
    await expect(handleWriteToolCall(ctx, "gkill_restore_kyou", { id: "k1", data_type: "kmemo" }))
      .rejects.toThrow(/already active/);
    // 現在値を取っただけで更新は送っていない
    expect(ctx.client.callApi).toHaveBeenCalledTimes(1);
  });

  test("throws Entity not found when the id has no history", async () => {
    const ctx = makeCtx(async () => ({ kmemo_histories: [] }));
    await expect(handleWriteToolCall(ctx, "gkill_restore_kyou", { id: "nope", data_type: "kmemo" }))
      .rejects.toThrow(/Entity not found/);
  });
});

// ---------------------------------------------------------------------------
// update_time は必ず前進する（1秒解像度の罠）
// ---------------------------------------------------------------------------
describe("update_time always moves forward", () => {
  test("restore in the same second as the delete still gets a later update_time", async () => {
    // UPDATE_TIME は秒までしか保存されず、最新版の判定は厳密な After なので、
    // 同じ秒のまま送ると復活が黙って無視される
    const sameSecond = new Date().toISOString();
    const ctx = makeCtx();
    ctx.client.callApi
      .mockResolvedValueOnce({ kmemo_histories: [{ id: "k1", is_deleted: true, update_time: sameSecond }] })
      .mockResolvedValueOnce({ updated_kyou: { id: "k1" } });

    await handleWriteToolCall(ctx, "gkill_restore_kyou", { id: "k1", data_type: "kmemo" });

    const sent = ctx.client.callApi.mock.calls[1][1].kmemo.update_time;
    expect(Math.floor(Date.parse(sent) / 1000)).toBeGreaterThan(Math.floor(Date.parse(sameSecond) / 1000));
  });

  test("delete has the same guarantee", async () => {
    const sameSecond = new Date().toISOString();
    const ctx = makeCtx();
    ctx.client.callApi
      .mockResolvedValueOnce({ kmemo_histories: [{ id: "k1", is_deleted: false, update_time: sameSecond }] })
      .mockResolvedValueOnce({ updated_kyou: { id: "k1" } });

    await handleWriteToolCall(ctx, "gkill_delete_kyou", { id: "k1", data_type: "kmemo" });

    const sent = ctx.client.callApi.mock.calls[1][1].kmemo.update_time;
    expect(Math.floor(Date.parse(sent) / 1000)).toBeGreaterThan(Math.floor(Date.parse(sameSecond) / 1000));
  });
});

describe("削除の冪等性とエラー品質 (2026-08-24 再監査 P-30 / P-34 / Q-03)", () => {
  test("refuses to delete an already-deleted entity instead of stacking another version", async () => {
    // 2回目も成功を返していたので「消えたのか、元から無かったのか、既に消えていたのか」が
    // 区別できず、しかも履歴に無意味な版が積まれていた
    const ctx = makeCtx(async () => ({ kmemo_histories: [{ id: "k1", is_deleted: true }] }));
    await expect(
      handleWriteToolCall(ctx, "gkill_delete_kyou", { id: "k1", data_type: "kmemo" }),
    ).rejects.toThrow(/already deleted/);
    // 更新は投げていない (GET だけで止まる)
    expect(ctx.client.callApi.mock.calls).toHaveLength(1);
  });

  test("still deletes an entity that is not deleted yet", async () => {
    const ctx = makeCtx(async (pathname) =>
      pathname === "/api/get_kmemo"
        ? { kmemo_histories: [{ id: "k1", is_deleted: false, update_time: "2026-08-24T03:00:00+09:00" }] }
        : { updated_kyou: { id: "k1" } },
    );
    const result = await handleWriteToolCall(ctx, "gkill_delete_kyou", { id: "k1", data_type: "kmemo" });
    expect(result.updated_kmemo.is_deleted).toBe(true);
  });

  test("entity-not-found names the data_type it looked under, because a wrong type looks identical", async () => {
    const ctx = makeCtx(async () => ({ urlog_histories: [] }));
    await expect(
      handleWriteToolCall(ctx, "gkill_delete_kyou", { id: "k1", data_type: "urlog" }),
    ).rejects.toThrow(/data_type "urlog"/);
  });

  test("restore reports not-found the same way", async () => {
    const ctx = makeCtx(async () => ({ kmemo_histories: [] }));
    await expect(
      handleWriteToolCall(ctx, "gkill_restore_kyou", { id: "k1", data_type: "kmemo" }),
    ).rejects.toThrow(/data_type "kmemo"/);
  });

  test("a removed tool name comes back with what to use instead", async () => {
    // ツール一覧はクライアントのセッション寿命で固定されるので、消しても呼ばれ続ける
    const ctx = makeCtx();
    await expect(handleWriteToolCall(ctx, "gkill_get_idf_file_path", {})).rejects.toThrow(
      /gkill_get_idf_file/,
    );
  });

  test("an ordinary unknown tool keeps the plain message", async () => {
    const ctx = makeCtx();
    await expect(handleWriteToolCall(ctx, "nonexistent_tool", {})).rejects.toThrow(
      /^Unknown tool: nonexistent_tool$/,
    );
  });
});

describe("応答はサーバが保存した版を返す (2026-08-24 再監査 P-09)", () => {
  // 以前はローカルで組んだ current をそのまま返していたので、update_time が
  // JS の UTC・ミリ秒つきのままになり、同じ応答の updated_kyou (JST・秒) と
  // 日付表記まで食い違っていた。保存は1秒解像度なのでミリ秒は存在しない精度でもある。
  test("update takes update_time from the server, not from the locally built object", async () => {
    const ctx = makeCtx(async (pathname) =>
      pathname === "/api/get_kmemo"
        ? { kmemo_histories: [{ id: "k1", content: "before", update_time: "2026-08-24T03:00:00+09:00" }] }
        : {
            updated_kmemo: { id: "k1", content: "after", update_time: "2026-08-24T03:47:31+09:00" },
            updated_kyou: { id: "k1", update_time: "2026-08-24T03:47:31+09:00" },
          },
    );
    const result = await handleWriteToolCall(ctx, "gkill_update_kmemo", { id: "k1", content: "after" });

    expect(result.updated_kmemo.update_time).toBe("2026-08-24T03:47:31+09:00");
    // 同じ応答の中で2つの時刻表現が割れない
    expect(result.updated_kmemo.update_time).toBe(result.updated_kyou.update_time);
    // 送った側は依然として「必ず後」になる時刻を送っている
    expect(ctx.client.callApi.mock.calls[1][1].kmemo.update_time).not.toBe("2026-08-24T03:00:00+09:00");
  });

  test("delete and restore follow the same rule", async () => {
    const deleteCtx = makeCtx(async (pathname) =>
      pathname === "/api/get_kmemo"
        ? { kmemo_histories: [{ id: "k1", content: "x", is_deleted: false, update_time: "2026-08-24T03:00:00+09:00" }] }
        : { updated_kmemo: { id: "k1", is_deleted: true, update_time: "2026-08-24T03:48:35+09:00" } },
    );
    const deleted = await handleWriteToolCall(deleteCtx, "gkill_delete_kyou", { id: "k1", data_type: "kmemo" });
    expect(deleted.updated_kmemo.update_time).toBe("2026-08-24T03:48:35+09:00");

    const restoreCtx = makeCtx(async (pathname) =>
      pathname === "/api/get_kmemo"
        ? { kmemo_histories: [{ id: "k1", content: "x", is_deleted: true, update_time: "2026-08-24T03:48:35+09:00" }] }
        : { updated_kmemo: { id: "k1", is_deleted: false, update_time: "2026-08-24T03:49:47+09:00" } },
    );
    const restored = await handleWriteToolCall(restoreCtx, "gkill_restore_kyou", { id: "k1", data_type: "kmemo" });
    expect(restored.restored_kmemo.update_time).toBe("2026-08-24T03:49:47+09:00");
  });

  test("a partial server response does not drop fields we already had", async () => {
    // 実サーバは完全なエンティティを返すが、部分応答でも手元の値を落とさない
    const ctx = makeCtx(async (pathname) =>
      pathname === "/api/get_kmemo"
        ? { kmemo_histories: [{ id: "k1", content: "keepme", is_deleted: false }] }
        : { updated_kmemo: { id: "k1", is_deleted: true } },
    );
    const result = await handleWriteToolCall(ctx, "gkill_delete_kyou", { id: "k1", data_type: "kmemo" });
    expect(result.updated_kmemo.content).toBe("keepme");
    expect(result.updated_kmemo.is_deleted).toBe(true);
  });
});

// ---------------------------------------------------------------------------
// gkill_update_timeis — end_time の3値 (2026-08-24 再監査 P-41)
// ---------------------------------------------------------------------------
describe("gkill_update_timeis end_time three-state patch", () => {
  // 「未指定=据え置き / null=進行中へ戻す / 値=その時刻」の3値は、正規化層
  // (write-normalization.test.mjs) だけでなくハンドラの `!== undefined` ガードが
  // 支えている。ガードが `!= null` 等に変わっても正規化層のテストは緑のままなので、
  // リクエスト本文まで届く/届かないことはここで固定する。
  test("end_time:null reaches the request body as null (finished back to ongoing)", async () => {
    // null を保存する以外に、一度終わらせた TimeIs を進行中へ戻す手段は無い
    // (Go 側 reps.TimeIs.EndTime は *time.Time で nil を保存できる)
    const ctx = makeCtx();
    ctx.client.callApi
      .mockResolvedValueOnce({
        timeis_histories: [{
          id: "t1",
          title: "work",
          start_time: "2026-08-24T09:00:00+09:00",
          end_time: "2026-08-24T13:00:00+09:00",
        }],
      })
      .mockResolvedValueOnce({ updated_kyou: { id: "t1" } });

    const result = await handleWriteToolCall(ctx, "gkill_update_timeis", { id: "t1", end_time: null });

    expect(ctx.client.callApi.mock.calls[0][0]).toBe("/api/get_timeis");
    expect(ctx.client.callApi.mock.calls[1][0]).toBe("/api/update_timeis");
    expect(ctx.client.callApi.mock.calls[1][1].timeis.end_time).toBeNull();
    expect(result.updated_timeis.end_time).toBeNull();
  });

  test("omitting end_time keeps the stored value untouched", async () => {
    // 未指定は「触らない」。ここが崩れると、タイトルを直しただけの update が
    // 終了時刻を消して記録を進行中へ戻してしまう
    const ctx = makeCtx();
    ctx.client.callApi
      .mockResolvedValueOnce({
        timeis_histories: [{
          id: "t1",
          title: "work",
          start_time: "2026-08-24T09:00:00+09:00",
          end_time: "2026-08-24T13:00:00+09:00",
        }],
      })
      .mockResolvedValueOnce({ updated_kyou: { id: "t1" } });

    const result = await handleWriteToolCall(ctx, "gkill_update_timeis", { id: "t1", title: "renamed" });

    const sent = ctx.client.callApi.mock.calls[1][1].timeis;
    expect(sent.title).toBe("renamed");
    expect(sent.end_time).toBe("2026-08-24T13:00:00+09:00");
    expect(result.updated_timeis.end_time).toBe("2026-08-24T13:00:00+09:00");
  });
});

// ---------------------------------------------------------------------------
// 一括削除／一括復活
//
// KFTL は1回で5件作れるのに、消すのは1件ずつ2往復だった（検証の後片付けに11往復）。
// gkill_submit_kftl の created[] が {id, data_type} の配列なので、
// **応答をそのまま入力へ渡せる**形にしてある（2026-08-24 の実利用レビュー）。
// ---------------------------------------------------------------------------

describe("handleWriteToolCall — batch delete / restore", () => {
  test("processes every target and reports per-entry results", async () => {
    const ctx = makeCtx();
    ctx.client.callApi
      .mockResolvedValueOnce({ kmemo_histories: [{ id: "k1", is_deleted: false }] })
      .mockResolvedValueOnce({ updated_kmemo: { id: "k1" } })
      .mockResolvedValueOnce({ lantana_histories: [{ id: "l1", is_deleted: false }] })
      .mockResolvedValueOnce({ updated_lantana: { id: "l1" } });

    const result = await handleWriteToolCall(ctx, "gkill_delete_kyou", {
      targets: [
        { id: "k1", data_type: "kmemo" },
        { id: "l1", data_type: "lantana" },
      ],
    });

    expect(result.succeeded_count).toBe(2);
    expect(result.failed_count).toBe(0);
    expect(result.results).toEqual([
      { id: "k1", data_type: "kmemo", ok: true },
      { id: "l1", data_type: "lantana", ok: true },
    ]);
  });

  // DBトランザクションではないので、途中で失敗しても止めない。
  // 「どこまで消したか」を返さないと利用者は後始末ができない。
  test("keeps going after a failure and says how far it got", async () => {
    const ctx = makeCtx();
    ctx.client.callApi
      .mockResolvedValueOnce({ kmemo_histories: [] }) // 1件目: 見つからない
      .mockResolvedValueOnce({ lantana_histories: [{ id: "l1", is_deleted: false }] })
      .mockResolvedValueOnce({ updated_lantana: { id: "l1" } });

    const result = await handleWriteToolCall(ctx, "gkill_delete_kyou", {
      targets: [
        { id: "missing", data_type: "kmemo" },
        { id: "l1", data_type: "lantana" },
      ],
    });

    expect(result.succeeded_count).toBe(1);
    expect(result.failed_count).toBe(1);
    expect(result.results[0].ok).toBe(false);
    expect(result.results[0].error).toMatch(/not found/i);
    expect(result.results[1].ok).toBe(true);
  });

  // 単件の応答の形は変えない（既存の呼び出し側を壊さない）。
  test("the single-entry form still returns the entity, not a results list", async () => {
    const ctx = makeCtx();
    ctx.client.callApi
      .mockResolvedValueOnce({ kmemo_histories: [{ id: "k1", is_deleted: false }] })
      .mockResolvedValueOnce({ updated_kmemo: { id: "k1" } });

    const result = await handleWriteToolCall(ctx, "gkill_delete_kyou", { id: "k1", data_type: "kmemo" });
    expect(result.updated_kmemo.is_deleted).toBe(true);
    expect(result).not.toHaveProperty("results");
  });

  test("restore accepts the batch form too", async () => {
    const ctx = makeCtx();
    ctx.client.callApi
      .mockResolvedValueOnce({ kmemo_histories: [{ id: "k1", is_deleted: true }] })
      .mockResolvedValueOnce({ updated_kmemo: { id: "k1" } });

    const result = await handleWriteToolCall(ctx, "gkill_restore_kyou", {
      targets: [{ id: "k1", data_type: "kmemo" }],
    });
    expect(result.succeeded_count).toBe(1);
  });
});

// ---------------------------------------------------------------------------
// 一括削除のサマリ
//
// failed_count:1 でも「Deleted (soft): completed」と出ていた。JSON を読めば分かるが、
// サマリ行だけを読むと全成功に見える。一括は DB トランザクションではないので、
// **どこまで済んだか**こそがサマリの本題（2026-08-25 の実利用レビュー）。
// ---------------------------------------------------------------------------

describe("summarizeWriteToolPayload — batch soft delete", () => {
  test("says how many failed instead of reporting completion", () => {
    const summary = summarizeWriteToolPayload("gkill_delete_kyou", {
      results: [{ ok: true }, { ok: false }],
      succeeded_count: 1,
      failed_count: 1,
    });
    expect(summary).toContain("1/2");
    expect(summary).toContain("FAILED");
    expect(summary).not.toContain("completed");
  });

  test("reports a clean batch without crying failure", () => {
    const summary = summarizeWriteToolPayload("gkill_delete_kyou", {
      results: [{ ok: true }, { ok: true }],
      succeeded_count: 2,
      failed_count: 0,
    });
    expect(summary).toBe("Deleted (soft): 2/2 entries.");
  });

  test("restore uses the same shape", () => {
    const summary = summarizeWriteToolPayload("gkill_restore_kyou", {
      results: [{ ok: false }],
      succeeded_count: 0,
      failed_count: 1,
    });
    expect(summary).toContain("Restored");
    expect(summary).toContain("FAILED");
  });

  // 単件の応答は従来の文言のまま（既存の読み手を壊さない）
  test("the single-entry form keeps the old wording", () => {
    const summary = summarizeWriteToolPayload("gkill_delete_kyou", { updated_kmemo: { id: "k1" } });
    expect(summary).toBe("Deleted (soft): updated_kmemo");
  });
});

// ---------------------------------------------------------------------------
// 一本化した update と「見つからない」の文言
//
// update 9本は取得→patch→更新の手順が同じで、24行のブロックが9本並んでいた。
// 取得先・更新先・応答キーの対応は ENTITY_TARGETS に既にあり、softDeleteOne と
// gkill_get_kyou_history はそちらを使っていた。表と直書きの2形態が併存していた状態を
// 表側へ寄せた（2026-08-25）。直書きだったころは「見つからない」も3種類に割れていた。
// ---------------------------------------------------------------------------

describe("update tools are table-driven", () => {
  test.each([
    ["gkill_update_kmemo", "kmemo", "/api/get_kmemo", "/api/update_kmemo", "kmemo_histories", "updated_kmemo"],
    ["gkill_update_urlog", "urlog", "/api/get_urlog", "/api/update_urlog", "urlog_histories", "updated_urlog"],
    ["gkill_update_nlog", "nlog", "/api/get_nlog", "/api/update_nlog", "nlog_histories", "updated_nlog"],
    ["gkill_update_lantana", "lantana", "/api/get_lantana", "/api/update_lantana", "lantana_histories", "updated_lantana"],
    ["gkill_update_timeis", "timeis", "/api/get_timeis", "/api/update_timeis", "timeis_histories", "updated_timeis"],
    ["gkill_update_mi", "mi", "/api/get_mi", "/api/update_mi", "mi_histories", "updated_mi"],
    ["gkill_update_kc", "kc", "/api/get_kc", "/api/update_kc", "kc_histories", "updated_kc"],
    ["gkill_update_tag", "tag", "/api/get_tag_histories_by_tag_id", "/api/update_tag", "tag_histories", "updated_tag"],
    ["gkill_update_text", "text", "/api/get_text_histories_by_text_id", "/api/update_text", "text_histories", "updated_text"],
  ])("%s uses the ENTITY_TARGETS endpoints", async (tool, _type, getEndpoint, updateEndpoint, historiesKey, responseKey) => {
    const ctx = makeCtx();
    ctx.client.callApi
      .mockResolvedValueOnce({ [historiesKey]: [{ id: "x1", is_deleted: false }] })
      .mockResolvedValueOnce({ [responseKey]: { id: "x1" }, updated_kyou: { id: "x1" } });

    // 各型の必須引数は id だけ（他は patch なので省略できる）
    const result = await handleWriteToolCall(ctx, tool, { id: "x1" });

    expect(ctx.client.callApi.mock.calls[0][0]).toBe(getEndpoint);
    expect(ctx.client.callApi.mock.calls[1][0]).toBe(updateEndpoint);
    expect(result[responseKey]).toBeDefined();
  });

  // 「見つからない」は read / write / update で同じ1文になる。
  // 型を取り違えたのか ID が無いのかはサーバの応答から区別できないので、
  // **区別できないことを言う**のが唯一正しい案内。
  test("not-found says the lookup is per-type instead of blaming the id", async () => {
    const ctx = makeCtx(async () => ({ kmemo_histories: [] }));
    await expect(handleWriteToolCall(ctx, "gkill_update_kmemo", { id: "missing" }))
      .rejects.toThrow(/looked it up as data_type/);
  });

  test("not-found no longer uses a per-type wording", async () => {
    const ctx = makeCtx(async () => ({ urlog_histories: [] }));
    await expect(handleWriteToolCall(ctx, "gkill_update_urlog", { id: "missing" }))
      .rejects.not.toThrow(/^Urlog not found/);
  });
});

// ---------------------------------------------------------------------------
// 古いツールスキーマの警告は書き込み側にも掛かる
//
// 読み取り側だけが appendStaleSchemaWarning で包まれており、書き込み側は素通しだった。
// delete / restore の targets は後から足した非string型の引数なので、
// 古いクライアントからは正規JSON文字列で届く。片側だけに掛けると
// 「同じ古さなのに読み取りでしか知らされない」ことになる。
// ---------------------------------------------------------------------------

describe("handleWriteToolCall — stale tool schema warning", () => {
  test("warns when targets arrived as a JSON string", async () => {
    const ctx = makeCtx();
    ctx.client.callApi
      .mockResolvedValueOnce({ kmemo_histories: [{ id: "k1", is_deleted: false }] })
      .mockResolvedValueOnce({ updated_kmemo: { id: "k1" } });

    const payload = await handleWriteToolCall(ctx, "gkill_delete_kyou", {
      targets: '[{"id":"k1","data_type":"kmemo"}]',
    });

    expect(payload.warnings).toHaveLength(1);
    expect(payload.warnings[0]).toContain("tool schema snapshot looks stale");
  });

  // 誤警告を出さないことが本体と同じくらい重要。
  test("does not warn for a current-schema call", async () => {
    const ctx = makeCtx();
    ctx.client.callApi
      .mockResolvedValueOnce({ kmemo_histories: [{ id: "k1", is_deleted: false }] })
      .mockResolvedValueOnce({ updated_kmemo: { id: "k1" } });

    const payload = await handleWriteToolCall(ctx, "gkill_delete_kyou", {
      targets: [{ id: "k1", data_type: "kmemo" }],
    });
    expect(payload.warnings).toBeUndefined();
  });
});

describe("summarizeWriteToolPayload — 表から作る", () => {
  // 18本の case を並べると、欄を1つ足すとき18箇所を触ることになり、
  // 1つ落としてもテストは緑のまま（各ツールのテストは自分の case しか見ない）。
  test("9型すべての add / update が要約を返す", () => {
    for (const dataType of ["kmemo", "urlog", "nlog", "lantana", "timeis", "mi", "kc", "tag", "text"]) {
      const added = summarizeWriteToolPayload(`gkill_add_${dataType}`, { [`added_${dataType}`]: { id: "x1" } });
      expect(added).toContain(dataType);
      expect(added).toContain("x1");

      const updated = summarizeWriteToolPayload(`gkill_update_${dataType}`, { [`updated_${dataType}`]: { id: "x2" } });
      expect(updated).toBe(`Updated ${dataType}: x2`);
    }
  });

  // tag / text は「作る」のではなく既存の記録へ「付ける」。
  test("動詞は tag / text だけ Added", () => {
    expect(summarizeWriteToolPayload("gkill_add_tag", { added_tag: { id: "t1" } })).toBe("Added tag: t1");
    expect(summarizeWriteToolPayload("gkill_add_kmemo", { added_kmemo: { id: "k1" } })).toBe("Created kmemo: k1");
  });

  test("対象外のツールは null", () => {
    expect(summarizeWriteToolPayload("gkill_get_kyous", {})).toBeNull();
  });

  // 古スキーマの印は読み取りの要約にしか無かった。delete/restore の targets が
  // まさに古スキーマで壊れる側なので、片側だけだと読み取りでしか知らされない。
  test("古スキーマの印が書き込みの要約にも付く", () => {
    const summary = summarizeWriteToolPayload("gkill_add_kmemo", {
      added_kmemo: { id: "k1" },
      warnings: ["this client's tool schema snapshot looks stale"],
    });
    expect(summary).toContain("reconnect the MCP client");
  });

  test("警告が無ければ印は付かない", () => {
    expect(summarizeWriteToolPayload("gkill_add_kmemo", { added_kmemo: { id: "k1" } })).toBe("Created kmemo: k1");
  });
});
