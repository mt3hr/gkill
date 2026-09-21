package mcp

// write_handlers.go のディスパッチのテスト。
//
// write / readwrite の2サーバが共有する1本の実装なので、ここが唯一の正本。
// サーバ経由の統合は write_server_test.go / readwrite_server_test.go が見る。
// 読み取り側の同じ形は read_handlers_test.go。
//
// 見張っているもの:
//   - add はサーバへ1回、update / delete は「現在値を取ってから送る」2回
//   - create_app / update_app が ctx.AppName（サーバ種別）で埋まる
//   - update は patch セマンティクス（送っていない項目が現在値のまま残る）
//   - delete は is_deleted=true を立てた現在値を update エンドポイントへ送る

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/mcp/jsonobj"
)

func makeWriteCtx(impl callApiImpl) *CallContext {
	client := &mockClient{}
	if impl == nil {
		impl = func(_ string, _ *jsonobj.Object) (*jsonobj.Object, error) {
			return obj("errors", arr(), "messages", arr()), nil
		}
	}
	client.callApi = func(pathname string, body *jsonobj.Object, _ bool, _ string) (*jsonobj.Object, error) {
		return impl(pathname, body)
	}
	return &CallContext{Client: client, SID: "sid-1", UserID: "testuser", AppName: "gkill_mcp_readwrite"}
}

func writeSummary(t *testing.T, name string, payload *jsonobj.Object) string {
	t.Helper()
	summary, ok := SummarizeWriteToolPayload(name, payload)
	expectTrue(t, ok, "%s: no summary", name)
	return summary
}

// byPath は pathname で応答を分ける callApi 実装。
func byPath(getPath string, histories *jsonobj.Object, otherwise *jsonobj.Object) callApiImpl {
	return func(pathname string, _ *jsonobj.Object) (*jsonobj.Object, error) {
		if pathname == getPath {
			return histories, nil
		}
		return otherwise, nil
	}
}

func epochSeconds(t *testing.T, value string) int64 {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339Nano, value)
	expectNoError(t, err)
	return parsed.Unix()
}

func TestHandleWriteToolCallAddTools(t *testing.T) {
	t.Run("gkill_add_kmemo posts to /api/add_kmemo with server-side metadata", func(t *testing.T) {
		ctx := makeWriteCtx(resolving(func() *jsonobj.Object { return obj("added_kmemo", obj("id", "k1"), "added_kyou", obj("id", "k1")) }))
		result, err := HandleWriteToolCall(ctx, "gkill_add_kmemo", obj("content", "hello"))
		expectNoError(t, err)

		call := clientOf(ctx).calls[0]
		expectEqual(t, call.Pathname, "/api/add_kmemo")
		expectEqual(t, objAt(t, call.Body, "kmemo").Value("content"), "hello")
		expectEqual(t, call.Body.Value("want_response_kyou"), true)
		expectEqual(t, objAt(t, result, "added_kmemo").Value("id"), "k1")
	})

	t.Run("create_app comes from ctx.appName so each server labels its own writes", func(t *testing.T) {
		// write サーバと readwrite サーバで create_app が違う。ここを取り違えると
		// 「どのサーバが書いたか」が記録から分からなくなる
		ctx := makeWriteCtx(resolving(func() *jsonobj.Object { return obj("added_kmemo", obj("id", "k1")) }))
		ctx.AppName = "gkill_mcp_write"
		_, err := HandleWriteToolCall(ctx, "gkill_add_kmemo", obj("content", "hello"))
		expectNoError(t, err)

		kmemo := objAt(t, clientOf(ctx).calls[0].Body, "kmemo")
		expectEqual(t, kmemo.Value("create_app"), "gkill_mcp_write")
		expectEqual(t, kmemo.Value("update_app"), "gkill_mcp_write")
		expectEqual(t, kmemo.Value("create_device"), "mcp")
		expectEqual(t, kmemo.Value("create_user"), "testuser")
	})

	t.Run("every add tool posts to its own /api/add_* endpoint", func(t *testing.T) {
		cases := []struct {
			name     string
			args     *jsonobj.Object
			endpoint string
		}{
			{"gkill_add_kmemo", obj("content", "x"), "/api/add_kmemo"},
			{"gkill_add_urlog", obj("url", "https://example.com"), "/api/add_urlog"},
			{"gkill_add_nlog", obj("title", "x", "amount", -1), "/api/add_nlog"},
			{"gkill_add_lantana", obj("mood", 5), "/api/add_lantana"},
			{"gkill_add_timeis", obj("title", "x"), "/api/add_timeis"},
			{"gkill_add_mi", obj("title", "x", "board_name", "b"), "/api/add_mi"},
			{"gkill_add_kc", obj("title", "x", "num_value", 1), "/api/add_kc"},
			{"gkill_add_tag", obj("tag", "t", "target_id", "id1"), "/api/add_tag"},
			{"gkill_add_text", obj("text", "t", "target_id", "id1"), "/api/add_text"},
		}
		for _, tc := range cases {
			ctx := makeWriteCtx(nil)
			_, err := HandleWriteToolCall(ctx, tc.name, tc.args)
			expectNoError(t, err)
			expectEqual(t, clientOf(ctx).calls[0].Pathname, tc.endpoint)
		}
	})

	t.Run("gkill_add_mi falls back to the account default board when board_name is omitted", func(t *testing.T) {
		ctx := makeWriteCtx(nil)
		clientOf(ctx).
			mockResolvedValueOnce(obj("application_config", obj("mi_default_board", "Inbox"))).
			mockResolvedValueOnce(obj("added_mi", obj("id", "m1")))

		_, err := HandleWriteToolCall(ctx, "gkill_add_mi", obj("title", "buy milk"))
		expectNoError(t, err)

		expectEqual(t, clientOf(ctx).calls[0].Pathname, "/api/get_application_config")
		expectEqual(t, clientOf(ctx).calls[1].Pathname, "/api/add_mi")
		expectEqual(t, objAt(t, clientOf(ctx).calls[1].Body, "mi").Value("board_name"), "Inbox")
	})

	t.Run("gkill_add_mi does not look up the config when board_name is given", func(t *testing.T) {
		ctx := makeWriteCtx(resolving(func() *jsonobj.Object { return obj("added_mi", obj("id", "m1")) }))
		_, err := HandleWriteToolCall(ctx, "gkill_add_mi", obj("title", "buy milk", "board_name", "errands"))
		expectNoError(t, err)

		expectEqual(t, len(clientOf(ctx).calls), 1)
		expectEqual(t, clientOf(ctx).calls[0].Pathname, "/api/add_mi")
		expectEqual(t, objAt(t, clientOf(ctx).calls[0].Body, "mi").Value("board_name"), "errands")
	})

	t.Run("gkill_add_mi still creates the task when the config lookup fails", func(t *testing.T) {
		// 既定板が引けないことを理由にタスク作成そのものを失敗させない
		ctx := makeWriteCtx(nil)
		clientOf(ctx).
			mockRejectedValueOnce(Errorf("config unavailable")).
			mockResolvedValueOnce(obj("added_mi", obj("id", "m1")))

		_, err := HandleWriteToolCall(ctx, "gkill_add_mi", obj("title", "buy milk"))
		expectNoError(t, err)

		expectEqual(t, objAt(t, clientOf(ctx).calls[1].Body, "mi").Value("board_name"), "Inbox")
	})

	t.Run("gkill_add_urlog drops the image base64 from the response", func(t *testing.T) {
		// 1件2KB前後の浪費。検索結果の urlog payload には元から載っていない
		ctx := makeWriteCtx(resolving(func() *jsonobj.Object {
			return obj(
				"added_urlog", obj("id", "u1", "url", "https://example.com", "title", "T", "favicon_image", "AAAA", "thumbnail_image", "BBBB"),
				"added_kyou", obj("id", "u1"),
			)
		}))
		result, err := HandleWriteToolCall(ctx, "gkill_add_urlog", obj("url", "https://example.com"))
		expectNoError(t, err)

		urlog := objAt(t, result, "added_urlog")
		expectEqual(t, urlog.Value("id"), "u1")
		expectEqual(t, urlog.Value("title"), "T")
		expectTrue(t, !urlog.Has("favicon_image"), "favicon_image kept")
		expectTrue(t, !urlog.Has("thumbnail_image"), "thumbnail_image kept")
	})

	t.Run("gkill_add_tag sends related_time and does not claim a parent kyou", func(t *testing.T) {
		// related_time を送らないと Go のゼロ値 0001-01-01 が保存される。
		// AddTagResponse に added_kyou は無いので、返しても常に null だった
		ctx := makeWriteCtx(resolving(func() *jsonobj.Object { return obj("added_tag", obj("id", "t1")) }))
		result, err := HandleWriteToolCall(ctx, "gkill_add_tag", obj("tag", "x", "target_id", "id1"))
		expectNoError(t, err)

		tag := objAt(t, clientOf(ctx).calls[0].Body, "tag")
		relatedTime, isString := tag.String("related_time")
		expectTrue(t, isString, "related_time is not a string")
		expectTrue(t, !strings.HasPrefix(relatedTime, "0001-"), "related_time is the zero value")
		expectTrue(t, !result.Has("added_kyou"), "added_kyou claimed")
	})

	t.Run("gkill_add_text sends related_time and does not claim a parent kyou", func(t *testing.T) {
		ctx := makeWriteCtx(resolving(func() *jsonobj.Object { return obj("added_text", obj("id", "x1")) }))
		result, err := HandleWriteToolCall(ctx, "gkill_add_text", obj("text", "note", "target_id", "id1"))
		expectNoError(t, err)

		text := objAt(t, clientOf(ctx).calls[0].Body, "text")
		expectTrue(t, !strings.HasPrefix(strAt(t, text, "related_time"), "0001-"), "related_time is the zero value")
		expectTrue(t, !result.Has("added_kyou"), "added_kyou claimed")
	})

	t.Run("gkill_submit_kftl posts the raw text", func(t *testing.T) {
		ctx := makeWriteCtx(resolving(func() *jsonobj.Object { return obj("messages", arr(obj("message", "ok"))) }))
		_, err := HandleWriteToolCall(ctx, "gkill_submit_kftl", obj("kftl_text", "memo"))
		expectNoError(t, err)

		call := clientOf(ctx).calls[0]
		expectEqual(t, call.Pathname, "/api/submit_kftl_text")
		expectEqual(t, call.Body.Value("kftl_text"), "memo")
	})

	// 冪等キーで畳んだ再送は gkill が元の created[] を replayed:true で返す（ADR-0510）。
	// 落とすと「今回書いた」のか「元の控え」なのかが呼び出し側から区別できない。
	t.Run("gkill_submit_kftl passes replayed and the original created[] through", func(t *testing.T) {
		created := func() []any {
			return arr(obj("id", "k1", "data_type", "kmemo", "updated", false, "related_time", "2026-09-19T10:00:00+09:00"))
		}
		ctx := makeWriteCtx(resolving(func() *jsonobj.Object {
			return obj("messages", arr(obj("message", "ok")), "created", created(), "replayed", true)
		}))
		result, err := HandleWriteToolCall(ctx, "gkill_submit_kftl", obj("kftl_text", "memo", "idempotency_key", "k"))
		expectNoError(t, err)

		expectEqual(t, result.Value("replayed"), true)
		expectEqual(t, result.Value("created"), created())
		mustMatch(t, writeSummary(t, "gkill_submit_kftl", result), `replay.*nothing written`)
	})

	t.Run("gkill_submit_kftl reports replayed:false on a fresh submission", func(t *testing.T) {
		ctx := makeWriteCtx(resolving(func() *jsonobj.Object {
			return obj("messages", arr(obj("message", "ok")), "created", arr(obj("id", "k1", "data_type", "kmemo", "updated", false)))
		}))
		result, err := HandleWriteToolCall(ctx, "gkill_submit_kftl", obj("kftl_text", "memo"))
		expectNoError(t, err)

		expectEqual(t, result.Value("replayed"), false)
		mustContain(t, writeSummary(t, "gkill_submit_kftl", result), "wrote 1 record(s)")
	})
}

func TestHandleWriteToolCallUpdateTools(t *testing.T) {
	t.Run("gkill_update_kmemo reads the current version before writing", func(t *testing.T) {
		ctx := makeWriteCtx(nil)
		clientOf(ctx).
			mockResolvedValueOnce(obj("kmemo_histories", arr(obj("id", "k1", "content", "old", "related_time", "2026-01-01T00:00:00+09:00")))).
			mockResolvedValueOnce(obj("updated_kyou", obj("id", "k1")))

		result, err := HandleWriteToolCall(ctx, "gkill_update_kmemo", obj("id", "k1", "content", "new"))
		expectNoError(t, err)

		expectEqual(t, clientOf(ctx).calls[0].Pathname, "/api/get_kmemo")
		expectEqual(t, clientOf(ctx).calls[1].Pathname, "/api/update_kmemo")
		expectEqual(t, objAt(t, result, "updated_kmemo").Value("content"), "new")
	})

	t.Run("fields that were not sent keep their current value (patch semantics)", func(t *testing.T) {
		// スキーマの required が過剰でも実装は patch。ここが崩れると
		// 「一部だけ直したつもりが他の項目を空で上書き」になる
		ctx := makeWriteCtx(nil)
		clientOf(ctx).
			mockResolvedValueOnce(obj("kmemo_histories", arr(obj("id", "k1", "content", "keep me", "related_time", "2026-01-01T00:00:00+09:00")))).
			mockResolvedValueOnce(obj("updated_kyou", obj("id", "k1")))

		result, err := HandleWriteToolCall(ctx, "gkill_update_kmemo", obj("id", "k1", "related_time", "2026-02-02T10:00:00+09:00"))
		expectNoError(t, err)

		expectEqual(t, objAt(t, result, "updated_kmemo").Value("content"), "keep me")
		expectEqual(t, objAt(t, result, "updated_kmemo").Value("related_time"), "2026-02-02T10:00:00+09:00")
	})

	t.Run("throws when the entity has no history", func(t *testing.T) {
		ctx := makeWriteCtx(resolving(func() *jsonobj.Object { return obj("kmemo_histories", arr()) }))
		_, err := HandleWriteToolCall(ctx, "gkill_update_kmemo", obj("id", "missing", "content", "x"))
		expectErrorMatches(t, err, `(?i)not found`)
	})
}

func TestHandleWriteToolCallDelete(t *testing.T) {
	t.Run("gkill_delete_kyou sets is_deleted on the current version and sends it to update", func(t *testing.T) {
		ctx := makeWriteCtx(nil)
		clientOf(ctx).
			mockResolvedValueOnce(obj("kmemo_histories", arr(obj("id", "k1", "content", "bye", "is_deleted", false)))).
			mockResolvedValueOnce(obj("updated_kmemo", obj("id", "k1"), "updated_kyou", obj("id", "k1")))

		result, err := HandleWriteToolCall(ctx, "gkill_delete_kyou", obj("id", "k1", "data_type", "kmemo"))
		expectNoError(t, err)

		expectEqual(t, clientOf(ctx).calls[0].Pathname, "/api/get_kmemo")
		expectEqual(t, clientOf(ctx).calls[1].Pathname, "/api/update_kmemo")
		expectEqual(t, objAt(t, clientOf(ctx).calls[1].Body, "kmemo").Value("is_deleted"), true)
		expectEqual(t, objAt(t, result, "updated_kmemo").Value("is_deleted"), true)
	})

	t.Run("throws Entity not found when the id has no history", func(t *testing.T) {
		ctx := makeWriteCtx(resolving(func() *jsonobj.Object { return obj("kmemo_histories", arr()) }))
		_, err := HandleWriteToolCall(ctx, "gkill_delete_kyou", obj("id", "nope", "data_type", "kmemo"))
		expectErrorMatches(t, err, `Entity not found`)
	})
}

func TestHandleWriteToolCallUnknownTools(t *testing.T) {
	t.Run("throws for a tool it does not own", func(t *testing.T) {
		ctx := makeWriteCtx(nil)
		_, err := HandleWriteToolCall(ctx, "unknown_tool", obj())
		expectErrorMatches(t, err, `Unknown tool`)
	})

	t.Run("isWriteToolName gates the dispatch", func(t *testing.T) {
		expectTrue(t, IsWriteToolName("gkill_add_kmemo"), "gkill_add_kmemo is not a write tool")
		expectTrue(t, !IsWriteToolName("unknown_tool"), "unknown_tool is a write tool")
	})
}

// ---------------------------------------------------------------------------
// gkill_restore_kyou — 削除の取り消し
// ---------------------------------------------------------------------------
func TestHandleWriteToolCallRestore(t *testing.T) {
	t.Run("clears is_deleted on the current version and sends it to update", func(t *testing.T) {
		ctx := makeWriteCtx(nil)
		clientOf(ctx).
			mockResolvedValueOnce(obj("kmemo_histories", arr(obj("id", "k1", "content", "back", "is_deleted", true, "update_time", "2026-01-01T00:00:00+09:00")))).
			mockResolvedValueOnce(obj("updated_kyou", obj("id", "k1")))

		result, err := HandleWriteToolCall(ctx, "gkill_restore_kyou", obj("id", "k1", "data_type", "kmemo"))
		expectNoError(t, err)

		expectEqual(t, clientOf(ctx).calls[0].Pathname, "/api/get_kmemo")
		expectEqual(t, clientOf(ctx).calls[1].Pathname, "/api/update_kmemo")
		expectEqual(t, objAt(t, clientOf(ctx).calls[1].Body, "kmemo").Value("is_deleted"), false)
		expectEqual(t, objAt(t, result, "restored_kmemo").Value("is_deleted"), false)
	})

	t.Run("refuses to append a pointless version when the entry is already active", func(t *testing.T) {
		ctx := makeWriteCtx(resolving(func() *jsonobj.Object {
			return obj("kmemo_histories", arr(obj("id", "k1", "is_deleted", false, "update_time", "2026-01-01T00:00:00+09:00")))
		}))
		_, err := HandleWriteToolCall(ctx, "gkill_restore_kyou", obj("id", "k1", "data_type", "kmemo"))
		expectErrorMatches(t, err, `already active`)
		// 現在値を取っただけで更新は送っていない
		expectEqual(t, len(clientOf(ctx).calls), 1)
	})

	t.Run("throws Entity not found when the id has no history", func(t *testing.T) {
		ctx := makeWriteCtx(resolving(func() *jsonobj.Object { return obj("kmemo_histories", arr()) }))
		_, err := HandleWriteToolCall(ctx, "gkill_restore_kyou", obj("id", "nope", "data_type", "kmemo"))
		expectErrorMatches(t, err, `Entity not found`)
	})
}

// ---------------------------------------------------------------------------
// update_time は必ず前進する（1秒解像度の罠）
// ---------------------------------------------------------------------------
func TestUpdateTimeAlwaysMovesForward(t *testing.T) {
	t.Run("restore in the same second as the delete still gets a later update_time", func(t *testing.T) {
		// UPDATE_TIME は秒までしか保存されず、最新版の判定は厳密な After なので、
		// 同じ秒のまま送ると復活が黙って無視される
		sameSecond := jsISOString(time.Now())
		ctx := makeWriteCtx(nil)
		clientOf(ctx).
			mockResolvedValueOnce(obj("kmemo_histories", arr(obj("id", "k1", "is_deleted", true, "update_time", sameSecond)))).
			mockResolvedValueOnce(obj("updated_kyou", obj("id", "k1")))

		_, err := HandleWriteToolCall(ctx, "gkill_restore_kyou", obj("id", "k1", "data_type", "kmemo"))
		expectNoError(t, err)

		sent := strAt(t, objAt(t, clientOf(ctx).calls[1].Body, "kmemo"), "update_time")
		expectTrue(t, epochSeconds(t, sent) > epochSeconds(t, sameSecond), "update_time did not move forward: %s vs %s", sent, sameSecond)
	})

	// 更新も同じ保証を持つ。runUpdate だけ素の new Date() を使っていたため、
	// 同じ秒の中で2回更新すると2回目が最新版と見なされずに消えていた。
	t.Run("update in the same second as the current version still moves update_time forward", func(t *testing.T) {
		sameSecond := jsISOString(time.Now())
		ctx := makeWriteCtx(nil)
		clientOf(ctx).
			mockResolvedValueOnce(obj("kmemo_histories", arr(obj("id", "k1", "is_deleted", false, "update_time", sameSecond)))).
			mockResolvedValueOnce(obj("updated_kmemo", obj("id", "k1"), "updated_kyou", obj("id", "k1")))

		_, err := HandleWriteToolCall(ctx, "gkill_update_kmemo", obj("id", "k1", "content", "after"))
		expectNoError(t, err)

		sent := strAt(t, objAt(t, clientOf(ctx).calls[1].Body, "kmemo"), "update_time")
		expectTrue(t, epochSeconds(t, sent) > epochSeconds(t, sameSecond), "update_time did not move forward")
	})

	t.Run("delete has the same guarantee", func(t *testing.T) {
		sameSecond := jsISOString(time.Now())
		ctx := makeWriteCtx(nil)
		clientOf(ctx).
			mockResolvedValueOnce(obj("kmemo_histories", arr(obj("id", "k1", "is_deleted", false, "update_time", sameSecond)))).
			mockResolvedValueOnce(obj("updated_kyou", obj("id", "k1")))

		_, err := HandleWriteToolCall(ctx, "gkill_delete_kyou", obj("id", "k1", "data_type", "kmemo"))
		expectNoError(t, err)

		sent := strAt(t, objAt(t, clientOf(ctx).calls[1].Body, "kmemo"), "update_time")
		expectTrue(t, epochSeconds(t, sent) > epochSeconds(t, sameSecond), "update_time did not move forward")
	})
}

func TestDeleteIdempotencyAndErrorQuality(t *testing.T) {
	t.Run("refuses to delete an already-deleted entity instead of stacking another version", func(t *testing.T) {
		// 2回目も成功を返していたので「消えたのか、元から無かったのか、既に消えていたのか」が
		// 区別できず、しかも履歴に無意味な版が積まれていた
		ctx := makeWriteCtx(resolving(func() *jsonobj.Object { return obj("kmemo_histories", arr(obj("id", "k1", "is_deleted", true))) }))
		_, err := HandleWriteToolCall(ctx, "gkill_delete_kyou", obj("id", "k1", "data_type", "kmemo"))
		expectErrorMatches(t, err, `already deleted`)
		// 更新は投げていない (GET だけで止まる)
		expectEqual(t, len(clientOf(ctx).calls), 1)
	})

	t.Run("still deletes an entity that is not deleted yet", func(t *testing.T) {
		ctx := makeWriteCtx(byPath("/api/get_kmemo",
			obj("kmemo_histories", arr(obj("id", "k1", "is_deleted", false, "update_time", "2026-08-24T03:00:00+09:00"))),
			obj("updated_kyou", obj("id", "k1"))))
		result, err := HandleWriteToolCall(ctx, "gkill_delete_kyou", obj("id", "k1", "data_type", "kmemo"))
		expectNoError(t, err)
		expectEqual(t, objAt(t, result, "updated_kmemo").Value("is_deleted"), true)
	})

	t.Run("entity-not-found names the data_type it looked under, because a wrong type looks identical", func(t *testing.T) {
		ctx := makeWriteCtx(resolving(func() *jsonobj.Object { return obj("urlog_histories", arr()) }))
		_, err := HandleWriteToolCall(ctx, "gkill_delete_kyou", obj("id", "k1", "data_type", "urlog"))
		expectErrorMatches(t, err, `data_type "urlog"`)
	})

	t.Run("restore reports not-found the same way", func(t *testing.T) {
		ctx := makeWriteCtx(resolving(func() *jsonobj.Object { return obj("kmemo_histories", arr()) }))
		_, err := HandleWriteToolCall(ctx, "gkill_restore_kyou", obj("id", "k1", "data_type", "kmemo"))
		expectErrorMatches(t, err, `data_type "kmemo"`)
	})

	t.Run("a removed tool name comes back with what to use instead", func(t *testing.T) {
		// ツール一覧はクライアントのセッション寿命で固定されるので、消しても呼ばれ続ける
		ctx := makeWriteCtx(nil)
		_, err := HandleWriteToolCall(ctx, "gkill_get_idf_file_path", obj())
		expectErrorMatches(t, err, `gkill_get_idf_file`)
	})

	t.Run("an ordinary unknown tool keeps the plain message", func(t *testing.T) {
		ctx := makeWriteCtx(nil)
		_, err := HandleWriteToolCall(ctx, "nonexistent_tool", obj())
		expectErrorMatches(t, err, `^Unknown tool: nonexistent_tool$`)
	})
}

func TestResponseCarriesTheServerSavedVersion(t *testing.T) {
	// 以前はローカルで組んだ current をそのまま返していたので、update_time が
	// JS の UTC・ミリ秒つきのままになり、同じ応答の updated_kyou (JST・秒) と
	// 日付表記まで食い違っていた。保存は1秒解像度なのでミリ秒は存在しない精度でもある。
	t.Run("update takes update_time from the server, not from the locally built object", func(t *testing.T) {
		ctx := makeWriteCtx(byPath("/api/get_kmemo",
			obj("kmemo_histories", arr(obj("id", "k1", "content", "before", "update_time", "2026-08-24T03:00:00+09:00"))),
			obj(
				"updated_kmemo", obj("id", "k1", "content", "after", "update_time", "2026-08-24T03:47:31+09:00"),
				"updated_kyou", obj("id", "k1", "update_time", "2026-08-24T03:47:31+09:00"),
			)))
		result, err := HandleWriteToolCall(ctx, "gkill_update_kmemo", obj("id", "k1", "content", "after"))
		expectNoError(t, err)

		expectEqual(t, objAt(t, result, "updated_kmemo").Value("update_time"), "2026-08-24T03:47:31+09:00")
		// 同じ応答の中で2つの時刻表現が割れない
		expectEqual(t, objAt(t, result, "updated_kmemo").Value("update_time"), objAt(t, result, "updated_kyou").Value("update_time"))
		// 送った側は依然として「必ず後」になる時刻を送っている
		expectTrue(t, objAt(t, clientOf(ctx).calls[1].Body, "kmemo").Value("update_time") != "2026-08-24T03:00:00+09:00", "update_time unchanged")
	})

	t.Run("delete and restore follow the same rule", func(t *testing.T) {
		deleteCtx := makeWriteCtx(byPath("/api/get_kmemo",
			obj("kmemo_histories", arr(obj("id", "k1", "content", "x", "is_deleted", false, "update_time", "2026-08-24T03:00:00+09:00"))),
			obj("updated_kmemo", obj("id", "k1", "is_deleted", true, "update_time", "2026-08-24T03:48:35+09:00"))))
		deleted, err := HandleWriteToolCall(deleteCtx, "gkill_delete_kyou", obj("id", "k1", "data_type", "kmemo"))
		expectNoError(t, err)
		expectEqual(t, objAt(t, deleted, "updated_kmemo").Value("update_time"), "2026-08-24T03:48:35+09:00")

		restoreCtx := makeWriteCtx(byPath("/api/get_kmemo",
			obj("kmemo_histories", arr(obj("id", "k1", "content", "x", "is_deleted", true, "update_time", "2026-08-24T03:48:35+09:00"))),
			obj("updated_kmemo", obj("id", "k1", "is_deleted", false, "update_time", "2026-08-24T03:49:47+09:00"))))
		restored, err := HandleWriteToolCall(restoreCtx, "gkill_restore_kyou", obj("id", "k1", "data_type", "kmemo"))
		expectNoError(t, err)
		expectEqual(t, objAt(t, restored, "restored_kmemo").Value("update_time"), "2026-08-24T03:49:47+09:00")
	})

	t.Run("a partial server response does not drop fields we already had", func(t *testing.T) {
		// 実サーバは完全なエンティティを返すが、部分応答でも手元の値を落とさない
		ctx := makeWriteCtx(byPath("/api/get_kmemo",
			obj("kmemo_histories", arr(obj("id", "k1", "content", "keepme", "is_deleted", false))),
			obj("updated_kmemo", obj("id", "k1", "is_deleted", true))))
		result, err := HandleWriteToolCall(ctx, "gkill_delete_kyou", obj("id", "k1", "data_type", "kmemo"))
		expectNoError(t, err)
		expectEqual(t, objAt(t, result, "updated_kmemo").Value("content"), "keepme")
		expectEqual(t, objAt(t, result, "updated_kmemo").Value("is_deleted"), true)
	})
}

// ---------------------------------------------------------------------------
// gkill_update_timeis — end_time の3値 (2巡目の指摘 P-41)
// ---------------------------------------------------------------------------
func TestUpdateTimeIsEndTimeThreeStatePatch(t *testing.T) {
	timeisHistory := func() *jsonobj.Object {
		return obj("timeis_histories", arr(obj(
			"id", "t1",
			"title", "work",
			"start_time", "2026-08-24T09:00:00+09:00",
			"end_time", "2026-08-24T13:00:00+09:00",
		)))
	}

	t.Run("end_time:null reaches the request body as null (finished back to ongoing)", func(t *testing.T) {
		// null を保存する以外に、一度終わらせた TimeIs を進行中へ戻す手段は無い
		// (Go 側 reps.TimeIs.EndTime は *time.Time で nil を保存できる)
		ctx := makeWriteCtx(nil)
		clientOf(ctx).mockResolvedValueOnce(timeisHistory()).mockResolvedValueOnce(obj("updated_kyou", obj("id", "t1")))

		result, err := HandleWriteToolCall(ctx, "gkill_update_timeis", obj("id", "t1", "end_time", nil))
		expectNoError(t, err)

		expectEqual(t, clientOf(ctx).calls[0].Pathname, "/api/get_timeis")
		expectEqual(t, clientOf(ctx).calls[1].Pathname, "/api/update_timeis")
		sent := objAt(t, clientOf(ctx).calls[1].Body, "timeis")
		expectTrue(t, sent.Has("end_time") && sent.Value("end_time") == nil, "end_time is not null in the body")
		updated := objAt(t, result, "updated_timeis")
		expectTrue(t, updated.Has("end_time") && updated.Value("end_time") == nil, "end_time is not null in the response")
	})

	t.Run("omitting end_time keeps the stored value untouched", func(t *testing.T) {
		// 未指定は「触らない」。ここが崩れると、タイトルを直しただけの update が
		// 終了時刻を消して記録を進行中へ戻してしまう
		ctx := makeWriteCtx(nil)
		clientOf(ctx).mockResolvedValueOnce(timeisHistory()).mockResolvedValueOnce(obj("updated_kyou", obj("id", "t1")))

		result, err := HandleWriteToolCall(ctx, "gkill_update_timeis", obj("id", "t1", "title", "renamed"))
		expectNoError(t, err)

		sent := objAt(t, clientOf(ctx).calls[1].Body, "timeis")
		expectEqual(t, sent.Value("title"), "renamed")
		expectEqual(t, sent.Value("end_time"), "2026-08-24T13:00:00+09:00")
		expectEqual(t, objAt(t, result, "updated_timeis").Value("end_time"), "2026-08-24T13:00:00+09:00")
	})
}

// ---------------------------------------------------------------------------
// 一括削除／一括復活
// ---------------------------------------------------------------------------
func TestHandleWriteToolCallBatchDeleteRestore(t *testing.T) {
	t.Run("processes every target and reports per-entry results", func(t *testing.T) {
		ctx := makeWriteCtx(nil)
		clientOf(ctx).
			mockResolvedValueOnce(obj("kmemo_histories", arr(obj("id", "k1", "is_deleted", false)))).
			mockResolvedValueOnce(obj("updated_kmemo", obj("id", "k1"))).
			mockResolvedValueOnce(obj("lantana_histories", arr(obj("id", "l1", "is_deleted", false)))).
			mockResolvedValueOnce(obj("updated_lantana", obj("id", "l1")))

		result, err := HandleWriteToolCall(ctx, "gkill_delete_kyou", obj(
			"targets", arr(obj("id", "k1", "data_type", "kmemo"), obj("id", "l1", "data_type", "lantana")),
		))
		expectNoError(t, err)

		expectEqual(t, result.Value("succeeded_count"), 2)
		expectEqual(t, result.Value("failed_count"), 0)
		expectEqual(t, result.Value("results"), arr(
			obj("id", "k1", "data_type", "kmemo", "ok", true),
			obj("id", "l1", "data_type", "lantana", "ok", true),
		))
	})

	// DB トランザクションではないので、途中で失敗しても止めない。
	// 「どこまで消したか」を返さないと利用者は後始末ができない。
	t.Run("keeps going after a failure and says how far it got", func(t *testing.T) {
		ctx := makeWriteCtx(nil)
		clientOf(ctx).
			mockResolvedValueOnce(obj("kmemo_histories", arr())). // 1件目: 見つからない
			mockResolvedValueOnce(obj("lantana_histories", arr(obj("id", "l1", "is_deleted", false)))).
			mockResolvedValueOnce(obj("updated_lantana", obj("id", "l1")))

		result, err := HandleWriteToolCall(ctx, "gkill_delete_kyou", obj(
			"targets", arr(obj("id", "missing", "data_type", "kmemo"), obj("id", "l1", "data_type", "lantana")),
		))
		expectNoError(t, err)

		expectEqual(t, result.Value("succeeded_count"), 1)
		expectEqual(t, result.Value("failed_count"), 1)
		results := arrAt(t, result, "results")
		expectEqual(t, objAt(t, results[0]).Value("ok"), false)
		mustMatch(t, strAt(t, objAt(t, results[0]), "error"), `(?i)not found`)
		expectEqual(t, objAt(t, results[1]).Value("ok"), true)
	})

	// 単件の応答の形は変えない（既存の呼び出し側を壊さない）。
	t.Run("the single-entry form still returns the entity, not a results list", func(t *testing.T) {
		ctx := makeWriteCtx(nil)
		clientOf(ctx).
			mockResolvedValueOnce(obj("kmemo_histories", arr(obj("id", "k1", "is_deleted", false)))).
			mockResolvedValueOnce(obj("updated_kmemo", obj("id", "k1")))

		result, err := HandleWriteToolCall(ctx, "gkill_delete_kyou", obj("id", "k1", "data_type", "kmemo"))
		expectNoError(t, err)
		expectEqual(t, objAt(t, result, "updated_kmemo").Value("is_deleted"), true)
		expectTrue(t, !result.Has("results"), "results present on the single form")
	})

	t.Run("restore accepts the batch form too", func(t *testing.T) {
		ctx := makeWriteCtx(nil)
		clientOf(ctx).
			mockResolvedValueOnce(obj("kmemo_histories", arr(obj("id", "k1", "is_deleted", true)))).
			mockResolvedValueOnce(obj("updated_kmemo", obj("id", "k1")))

		result, err := HandleWriteToolCall(ctx, "gkill_restore_kyou", obj("targets", arr(obj("id", "k1", "data_type", "kmemo"))))
		expectNoError(t, err)
		expectEqual(t, result.Value("succeeded_count"), 1)
	})
}

// ---------------------------------------------------------------------------
// 一括削除のサマリ
// ---------------------------------------------------------------------------
func TestSummarizeWriteToolPayloadBatchSoftDelete(t *testing.T) {
	t.Run("says how many failed instead of reporting completion", func(t *testing.T) {
		summary := writeSummary(t, "gkill_delete_kyou", obj(
			"results", arr(obj("ok", true), obj("ok", false)),
			"succeeded_count", 1,
			"failed_count", 1,
		))
		mustContain(t, summary, "1/2")
		mustContain(t, summary, "FAILED")
		mustNotContain(t, summary, "completed")
	})

	t.Run("reports a clean batch without crying failure", func(t *testing.T) {
		summary := writeSummary(t, "gkill_delete_kyou", obj(
			"results", arr(obj("ok", true), obj("ok", true)),
			"succeeded_count", 2,
			"failed_count", 0,
		))
		expectEqual(t, summary, "Deleted (soft): 2/2 entries.")
	})

	t.Run("restore uses the same shape", func(t *testing.T) {
		summary := writeSummary(t, "gkill_restore_kyou", obj(
			"results", arr(obj("ok", false)),
			"succeeded_count", 0,
			"failed_count", 1,
		))
		mustContain(t, summary, "Restored")
		mustContain(t, summary, "FAILED")
	})

	// 単件の応答は従来の文言のまま（既存の読み手を壊さない）
	t.Run("the single-entry form names the type and id", func(t *testing.T) {
		expectEqual(t, writeSummary(t, "gkill_delete_kyou", obj("updated_kmemo", obj("id", "k1"))), "Deleted (soft): kmemo k1")
	})
}

// ---------------------------------------------------------------------------
// 一本化した update と「見つからない」の文言
// ---------------------------------------------------------------------------
func TestUpdateToolsAreTableDriven(t *testing.T) {
	cases := []struct {
		tool, dataType, getEndpoint, updateEndpoint, historiesKey, responseKey string
		patch                                                                  *jsonobj.Object
	}{
		{"gkill_update_kmemo", "kmemo", "/api/get_kmemo", "/api/update_kmemo", "kmemo_histories", "updated_kmemo", obj("content", "x")},
		{"gkill_update_urlog", "urlog", "/api/get_urlog", "/api/update_urlog", "urlog_histories", "updated_urlog", obj("title", "x")},
		{"gkill_update_nlog", "nlog", "/api/get_nlog", "/api/update_nlog", "nlog_histories", "updated_nlog", obj("title", "x")},
		{"gkill_update_lantana", "lantana", "/api/get_lantana", "/api/update_lantana", "lantana_histories", "updated_lantana", obj("mood", 5)},
		{"gkill_update_timeis", "timeis", "/api/get_timeis", "/api/update_timeis", "timeis_histories", "updated_timeis", obj("title", "x")},
		{"gkill_update_mi", "mi", "/api/get_mi", "/api/update_mi", "mi_histories", "updated_mi", obj("title", "x")},
		{"gkill_update_kc", "kc", "/api/get_kc", "/api/update_kc", "kc_histories", "updated_kc", obj("title", "x")},
		{"gkill_update_tag", "tag", "/api/get_tag_histories_by_tag_id", "/api/update_tag", "tag_histories", "updated_tag", obj("tag", "x")},
		{"gkill_update_text", "text", "/api/get_text_histories_by_text_id", "/api/update_text", "text_histories", "updated_text", obj("text", "x")},
	}
	for _, tc := range cases {
		t.Run(tc.tool+" uses the ENTITY_TARGETS endpoints", func(t *testing.T) {
			ctx := makeWriteCtx(nil)
			clientOf(ctx).
				mockResolvedValueOnce(obj(tc.historiesKey, arr(obj("id", "x1", "is_deleted", false)))).
				mockResolvedValueOnce(obj(tc.responseKey, obj("id", "x1"), "updated_kyou", obj("id", "x1")))

			// 更新する欄が1つも無いと no-op ガードに弾かれるので、型ごとに1つだけ渡す
			result, err := HandleWriteToolCall(ctx, tc.tool, obj("id", "x1").Merge(tc.patch))
			expectNoError(t, err)

			expectEqual(t, clientOf(ctx).calls[0].Pathname, tc.getEndpoint)
			expectEqual(t, clientOf(ctx).calls[1].Pathname, tc.updateEndpoint)
			expectTrue(t, result.Defined(tc.responseKey), "%s missing", tc.responseKey)
		})
	}

	// 「見つからない」は read / write / update で同じ1文になる。
	// 型を取り違えたのか ID が無いのかはサーバの応答から区別できないので、
	// **区別できないことを言う**のが唯一正しい案内。
	t.Run("not-found says the lookup is per-type instead of blaming the id", func(t *testing.T) {
		ctx := makeWriteCtx(resolving(func() *jsonobj.Object { return obj("kmemo_histories", arr()) }))
		_, err := HandleWriteToolCall(ctx, "gkill_update_kmemo", obj("id", "missing"))
		expectErrorMatches(t, err, `looked it up as data_type`)
	})

	t.Run("not-found no longer uses a per-type wording", func(t *testing.T) {
		ctx := makeWriteCtx(resolving(func() *jsonobj.Object { return obj("urlog_histories", arr()) }))
		_, err := HandleWriteToolCall(ctx, "gkill_update_urlog", obj("id", "missing"))
		expectTrue(t, err != nil, "accepted")
		mustNotMatch(t, err.Error(), `^Urlog not found`)
	})

	// id だけの更新は「内容の同じ版」を積むだけになる。追記型なので黙って通すと
	// 履歴が1つ増え、あとから読む側には何が変わったのか区別が付かない。
	// delete/restore が already deleted / already active を弾くのと同じ理由。
	t.Run("update with no patchable field is rejected instead of appending a no-op version", func(t *testing.T) {
		ctx := makeWriteCtx(resolving(func() *jsonobj.Object {
			return obj("kmemo_histories", arr(obj("id", "k1", "content", "same", "is_deleted", false)))
		}))

		_, err := HandleWriteToolCall(ctx, "gkill_update_kmemo", obj("id", "k1"))
		expectErrorMatches(t, err, `No fields to update`)

		// 取得はしても、更新 API は呼ばない
		for _, path := range clientOf(ctx).pathsCalled() {
			expectTrue(t, path != "/api/update_kmemo", "update was called")
		}
	})

	// 「見つからない」のほうが先に出る。欄が無いことより、対象が無いことを先に言う
	t.Run("not-found wins over the empty-patch guard", func(t *testing.T) {
		ctx := makeWriteCtx(resolving(func() *jsonobj.Object { return obj("kmemo_histories", arr()) }))
		_, err := HandleWriteToolCall(ctx, "gkill_update_kmemo", obj("id", "missing"))
		expectErrorMatches(t, err, `looked it up as data_type`)
	})
}

// ---------------------------------------------------------------------------
// 古いツールスキーマの警告は書き込み側にも掛かる
// ---------------------------------------------------------------------------
func TestHandleWriteToolCallStaleToolSchemaWarning(t *testing.T) {
	t.Run("warns when targets arrived as a JSON string", func(t *testing.T) {
		ctx := makeWriteCtx(nil)
		clientOf(ctx).
			mockResolvedValueOnce(obj("kmemo_histories", arr(obj("id", "k1", "is_deleted", false)))).
			mockResolvedValueOnce(obj("updated_kmemo", obj("id", "k1")))

		payload, err := HandleWriteToolCall(ctx, "gkill_delete_kyou", obj("targets", `[{"id":"k1","data_type":"kmemo"}]`))
		expectNoError(t, err)

		warnings := arrAt(t, payload, "warnings")
		expectEqual(t, len(warnings), 1)
		mustContain(t, jsString(warnings[0]), "tool schema snapshot looks stale")
	})

	// 誤警告を出さないことが本体と同じくらい重要。
	t.Run("does not warn for a current-schema call", func(t *testing.T) {
		ctx := makeWriteCtx(nil)
		clientOf(ctx).
			mockResolvedValueOnce(obj("kmemo_histories", arr(obj("id", "k1", "is_deleted", false)))).
			mockResolvedValueOnce(obj("updated_kmemo", obj("id", "k1")))

		payload, err := HandleWriteToolCall(ctx, "gkill_delete_kyou", obj("targets", arr(obj("id", "k1", "data_type", "kmemo"))))
		expectNoError(t, err)
		expectTrue(t, !payload.Defined("warnings"), "warnings set")
	})
}

func TestSummarizeWriteToolPayloadTableDriven(t *testing.T) {
	// 18本の case を並べると、欄を1つ足すとき18箇所を触ることになり、
	// 1つ落としてもテストは緑のまま（各ツールのテストは自分の case しか見ない）。
	t.Run("9型すべての add / update が要約を返す", func(t *testing.T) {
		for _, dataType := range []string{"kmemo", "urlog", "nlog", "lantana", "timeis", "mi", "kc", "tag", "text"} {
			added := writeSummary(t, "gkill_add_"+dataType, obj("added_"+dataType, obj("id", "x1")))
			mustContain(t, added, dataType)
			mustContain(t, added, "x1")

			updated := writeSummary(t, "gkill_update_"+dataType, obj("updated_"+dataType, obj("id", "x2")))
			expectEqual(t, updated, "Updated "+dataType+": x2")
		}
	})

	// tag / text は「作る」のではなく既存の記録へ「付ける」。
	t.Run("動詞は tag / text だけ Added", func(t *testing.T) {
		expectEqual(t, writeSummary(t, "gkill_add_tag", obj("added_tag", obj("id", "t1"))), "Added tag: t1")
		expectEqual(t, writeSummary(t, "gkill_add_kmemo", obj("added_kmemo", obj("id", "k1"))), "Created kmemo: k1")
	})

	t.Run("対象外のツールは null", func(t *testing.T) {
		_, ok := SummarizeWriteToolPayload("gkill_get_kyous", obj())
		expectTrue(t, !ok, "summarized gkill_get_kyous")
	})

	// 古スキーマの印は読み取りの要約にしか無かった。delete/restore の targets が
	// まさに古スキーマで壊れる側なので、片側だけだと読み取りでしか知らされない。
	t.Run("古スキーマの印が書き込みの要約にも付く", func(t *testing.T) {
		summary := writeSummary(t, "gkill_add_kmemo", obj(
			"added_kmemo", obj("id", "k1"),
			"warnings", strs("this client's tool schema snapshot looks stale"),
		))
		mustContain(t, summary, "reconnect the MCP client")
	})

	t.Run("警告が無ければ印は付かない", func(t *testing.T) {
		expectEqual(t, writeSummary(t, "gkill_add_kmemo", obj("added_kmemo", obj("id", "k1"))), "Created kmemo: k1")
	})
}

// ---------------------------------------------------------------------------
// gkill_add_urlog の外向き取得抑止フラグ (2026-08-30 MCP レビュー、フラグ追加)
// ---------------------------------------------------------------------------
func TestHandleWriteToolCallUrlogFetchSuppressionFlags(t *testing.T) {
	addedUrlog := resolving(func() *jsonobj.Object { return obj("added_urlog", obj("id", "u1")) })
	hasStaleWarning := func(t *testing.T, payload *jsonobj.Object) bool {
		t.Helper()
		warnings, ok := payload.Array("warnings")
		if !ok {
			return false
		}
		for _, w := range warnings {
			if strings.Contains(jsString(w), "tool schema snapshot looks stale") {
				return true
			}
		}
		return false
	}

	t.Run("既定では skip_fetch_* = false (従来どおり取得する)", func(t *testing.T) {
		ctx := makeWriteCtx(addedUrlog)
		_, err := HandleWriteToolCall(ctx, "gkill_add_urlog", obj("url", "https://example.com"))
		expectNoError(t, err)

		body := clientOf(ctx).calls[0].Body
		expectEqual(t, body.Value("skip_fetch_metadata"), false)
		expectEqual(t, body.Value("skip_fetch_favicon"), false)
		// フラグはリクエストの兄弟フィールドであってエンティティの列ではない。
		urlog := objAt(t, body, "urlog")
		expectTrue(t, !urlog.Defined("fetch_metadata"), "fetch_metadata leaked into the entity")
		expectTrue(t, !urlog.Defined("fetch_favicon"), "fetch_favicon leaked into the entity")
		expectTrue(t, !urlog.Defined("skip_fetch_metadata"), "skip_fetch_metadata leaked into the entity")
	})

	t.Run("fetch_metadata:false / fetch_favicon:false は反転して skip_fetch_* へ写る", func(t *testing.T) {
		ctx := makeWriteCtx(addedUrlog)
		_, err := HandleWriteToolCall(ctx, "gkill_add_urlog", obj(
			"url", "https://example.com",
			"title", "example title",
			"fetch_metadata", false,
			"fetch_favicon", false,
		))
		expectNoError(t, err)

		body := clientOf(ctx).calls[0].Body
		expectEqual(t, body.Value("skip_fetch_metadata"), true)
		expectEqual(t, body.Value("skip_fetch_favicon"), true)
	})

	t.Run("フラグは独立 (favicon だけ抑止できる)", func(t *testing.T) {
		ctx := makeWriteCtx(addedUrlog)
		_, err := HandleWriteToolCall(ctx, "gkill_add_urlog", obj("url", "https://example.com", "fetch_favicon", false))
		expectNoError(t, err)

		body := clientOf(ctx).calls[0].Body
		expectEqual(t, body.Value("skip_fetch_metadata"), false)
		expectEqual(t, body.Value("skip_fetch_favicon"), true)
	})

	t.Run(`古いスキーマからの文字列 "false" も型が復元され、古さの警告が付く`, func(t *testing.T) {
		// 後付け引数は古いツールスキーマを掴んだクライアントから正規JSON文字列で届く
		ctx := makeWriteCtx(addedUrlog)
		payload, err := HandleWriteToolCall(ctx, "gkill_add_urlog", obj("url", "https://example.com", "fetch_metadata", "false"))
		expectNoError(t, err)

		expectEqual(t, clientOf(ctx).calls[0].Body.Value("skip_fetch_metadata"), true)
		expectTrue(t, hasStaleWarning(t, payload), "no stale warning")
	})

	t.Run("update_urlog は再取得キーを送らない (抑止フラグが update に無い理由)", func(t *testing.T) {
		// Go 側が外向き取得をやり直すのは re_get_urlog_content:true を明示されたときだけ
		// (handle_update_urlog.go)。MCP の runUpdate はこのキーを送らないので、
		// fetch_metadata:false で登録したブックマークを update しても外向き通信は起きない。
		// ここが送るようになると、利用者が抑止したはずの取得が update 経路で復活する。
		ctx := makeWriteCtx(nil)
		clientOf(ctx).
			mockResolvedValueOnce(obj("urlog_histories", arr(obj("id", "u1", "url", "https://example.com/", "title", "", "update_time", "2026-08-30T10:00:00+09:00")))).
			mockResolvedValueOnce(obj("updated_urlog", obj("id", "u1"), "updated_kyou", obj("id", "u1")))

		_, err := HandleWriteToolCall(ctx, "gkill_update_urlog", obj("id", "u1", "title", "renamed"))
		expectNoError(t, err)

		call := clientOf(ctx).calls[1]
		expectEqual(t, call.Pathname, "/api/update_urlog")
		expectTrue(t, !call.Body.Has("re_get_urlog_content"), "re_get_urlog_content sent")
	})
}

// ---------------------------------------------------------------------------
// allow_create_board — 板名の typo ガード (2026-08-30 MCP レビュー、フラグ追加)
// ---------------------------------------------------------------------------
func TestHandleWriteToolCallAllowCreateBoard(t *testing.T) {
	hasStaleWarning := func(payload *jsonobj.Object) bool {
		warnings, ok := payload.Array("warnings")
		if !ok {
			return false
		}
		for _, w := range warnings {
			if strings.Contains(jsString(w), "tool schema snapshot looks stale") {
				return true
			}
		}
		return false
	}

	t.Run("add: false で未知の板名は登録前に弾かれる (/api/add_mi へ行かない)", func(t *testing.T) {
		ctx := makeWriteCtx(resolving(func() *jsonobj.Object { return obj("boards", strs("Inbox", "errands")) }))
		_, err := HandleWriteToolCall(ctx, "gkill_add_mi", obj("title", "x", "board_name", "erands", "allow_create_board", false))
		expectErrorMatches(t, err, `unknown board`)

		expectEqual(t, len(clientOf(ctx).calls), 1)
		expectEqual(t, clientOf(ctx).calls[0].Pathname, "/api/get_mi_board_list")
	})

	t.Run("板名エラーの文言が名指しするツールは書き込み専用サーバにも実在する", func(t *testing.T) {
		// 「説明文が名指しするツールはそのサーバに載っていること」の機械検査は
		// tool_handlers_test.go にあるが、対象はスキーマと EntityNotFoundMessage だけで、
		// assertBoardExists のランタイム文言は誰も見ていなかった。実際に投げさせて検査する。
		ctx := makeWriteCtx(resolving(func() *jsonobj.Object { return obj("boards", strs("Inbox")) }))
		_, err := HandleWriteToolCall(ctx, "gkill_add_mi", obj("title", "x", "board_name", "nope", "allow_create_board", false))
		expectTrue(t, err != nil, "accepted")
		message := err.Error()
		mustContain(t, message, "unknown board")
		availableOnWriteServer := NewStringSet(toolNames(concatTools(WriteTools, filterTools(ReadTools, WriteServerReadToolNames)))...)
		mentions := regexp.MustCompile(`gkill_[a-z_]+`).FindAllString(message, -1)
		expectTrue(t, len(mentions) > 0, "no tool mentioned")
		for _, mentioned := range mentions {
			expectTrue(t, availableOnWriteServer.Has(mentioned), "%s is not on the write-only server", mentioned)
		}
	})

	t.Run("add: false でも実在の板名なら登録される", func(t *testing.T) {
		ctx := makeWriteCtx(nil)
		clientOf(ctx).
			mockResolvedValueOnce(obj("boards", strs("Inbox", "errands"))).
			mockResolvedValueOnce(obj("added_mi", obj("id", "m1")))

		_, err := HandleWriteToolCall(ctx, "gkill_add_mi", obj("title", "x", "board_name", "errands", "allow_create_board", false))
		expectNoError(t, err)

		expectEqual(t, clientOf(ctx).calls[0].Pathname, "/api/get_mi_board_list")
		expectEqual(t, clientOf(ctx).calls[1].Pathname, "/api/add_mi")
		expectEqual(t, objAt(t, clientOf(ctx).calls[1].Body, "mi").Value("board_name"), "errands")
	})

	t.Run("add: 既定(true)では板一覧を照合しない (従来どおり未知の板名は新しい板になる)", func(t *testing.T) {
		ctx := makeWriteCtx(resolving(func() *jsonobj.Object { return obj("added_mi", obj("id", "m1")) }))
		_, err := HandleWriteToolCall(ctx, "gkill_add_mi", obj("title", "x", "board_name", "brand-new-board"))
		expectNoError(t, err)

		expectEqual(t, len(clientOf(ctx).calls), 1)
		expectEqual(t, clientOf(ctx).calls[0].Pathname, "/api/add_mi")
		// 修飾子フラグは mi 実体へ書かれない (urlog の fetch_* と同じ線引き)。
		expectTrue(t, !objAt(t, clientOf(ctx).calls[0].Body, "mi").Defined("allow_create_board"), "allow_create_board leaked into the entity")
	})

	t.Run("add: false + board_name 未指定なら、既定板への補完値は照合しない", func(t *testing.T) {
		// 板が1つも無い新規アカウントでは get_mi_board_list が [] を返すので、
		// 補完値まで照合すると既定板すら弾いてタスクが1件も作れなくなる。
		// この設計判断はコメントにしか無かったのでここで固定する。
		ctx := makeWriteCtx(nil)
		clientOf(ctx).
			mockResolvedValueOnce(obj("application_config", obj("mi_default_board", "Inbox"))).
			mockResolvedValueOnce(obj("added_mi", obj("id", "m1")))

		_, err := HandleWriteToolCall(ctx, "gkill_add_mi", obj("title", "x", "allow_create_board", false))
		expectNoError(t, err)

		expectEqual(t, clientOf(ctx).pathsCalled(), []string{"/api/get_application_config", "/api/add_mi"})
		expectEqual(t, objAt(t, clientOf(ctx).calls[1].Body, "mi").Value("board_name"), "Inbox")
	})

	t.Run("update: false で未知の板名は現在値の取得より前に弾かれる", func(t *testing.T) {
		ctx := makeWriteCtx(resolving(func() *jsonobj.Object { return obj("boards", strs("Inbox")) }))
		_, err := HandleWriteToolCall(ctx, "gkill_update_mi", obj("id", "m1", "board_name", "no-such-board", "allow_create_board", false))
		expectErrorMatches(t, err, `unknown board`)

		expectEqual(t, len(clientOf(ctx).calls), 1)
		expectEqual(t, clientOf(ctx).calls[0].Pathname, "/api/get_mi_board_list")
	})

	t.Run("update: false でも実在の板名なら移動できる", func(t *testing.T) {
		ctx := makeWriteCtx(nil)
		clientOf(ctx).
			mockResolvedValueOnce(obj("boards", strs("Inbox", "errands"))).
			mockResolvedValueOnce(obj("mi_histories", arr(obj("id", "m1", "title", "x", "board_name", "Inbox", "is_checked", false, "update_time", "2026-08-30T10:00:00+09:00")))).
			mockResolvedValueOnce(obj("updated_mi", obj("id", "m1", "board_name", "errands"), "updated_kyou", obj("id", "m1")))

		result, err := HandleWriteToolCall(ctx, "gkill_update_mi", obj("id", "m1", "board_name", "errands", "allow_create_board", false))
		expectNoError(t, err)

		expectEqual(t, clientOf(ctx).calls[1].Pathname, "/api/get_mi")
		expectEqual(t, clientOf(ctx).calls[2].Pathname, "/api/update_mi")
		expectEqual(t, objAt(t, result, "updated_mi").Value("board_name"), "errands")
	})

	t.Run("update: allow_create_board だけでは「変わる欄なし」として拒否される (修飾子であって変更ではない)", func(t *testing.T) {
		ctx := makeWriteCtx(resolving(func() *jsonobj.Object {
			return obj("mi_histories", arr(obj("id", "m1", "title", "x", "board_name", "Inbox", "update_time", "2026-08-30T10:00:00+09:00")))
		}))
		_, err := HandleWriteToolCall(ctx, "gkill_update_mi", obj("id", "m1", "allow_create_board", false))
		expectErrorMatches(t, err, `No fields to update`)
	})

	t.Run("update: 既定では照合の往復が1つも増えない (1本目は現在値の取得)", func(t *testing.T) {
		ctx := makeWriteCtx(nil)
		clientOf(ctx).
			mockResolvedValueOnce(obj("mi_histories", arr(obj("id", "m1", "title", "x", "board_name", "Inbox", "update_time", "2026-08-30T10:00:00+09:00")))).
			mockResolvedValueOnce(obj("updated_mi", obj("id", "m1"), "updated_kyou", obj("id", "m1")))

		_, err := HandleWriteToolCall(ctx, "gkill_update_mi", obj("id", "m1", "board_name", "somewhere-new"))
		expectNoError(t, err)

		expectEqual(t, clientOf(ctx).pathsCalled(), []string{"/api/get_mi", "/api/update_mi"})
		// 修飾子フラグは mi 実体へ書かれない (patchFields に無い)。
		expectTrue(t, !objAt(t, clientOf(ctx).calls[1].Body, "mi").Defined("allow_create_board"), "allow_create_board leaked into the entity")
	})

	t.Run(`add: 古いスキーマからの文字列 "false" も照合を発火させ、古さの警告が付く`, func(t *testing.T) {
		// 文字列のまま比較 (=== false) されると照合が静かにスキップされる。
		ctx := makeWriteCtx(nil)
		clientOf(ctx).
			mockResolvedValueOnce(obj("boards", strs("Inbox", "errands"))).
			mockResolvedValueOnce(obj("added_mi", obj("id", "m1")))

		payload, err := HandleWriteToolCall(ctx, "gkill_add_mi", obj("title", "x", "board_name", "errands", "allow_create_board", "false"))
		expectNoError(t, err)

		expectEqual(t, clientOf(ctx).calls[0].Pathname, "/api/get_mi_board_list")
		expectTrue(t, hasStaleWarning(payload), "no stale warning")
	})

	t.Run(`update: 古いスキーマからの文字列 "false" も照合を発火させ、古さの警告が付く`, func(t *testing.T) {
		ctx := makeWriteCtx(nil)
		clientOf(ctx).
			mockResolvedValueOnce(obj("boards", strs("Inbox", "errands"))).
			mockResolvedValueOnce(obj("mi_histories", arr(obj("id", "m1", "title", "x", "board_name", "Inbox", "update_time", "2026-08-30T10:00:00+09:00")))).
			mockResolvedValueOnce(obj("updated_mi", obj("id", "m1"), "updated_kyou", obj("id", "m1")))

		payload, err := HandleWriteToolCall(ctx, "gkill_update_mi", obj("id", "m1", "board_name", "errands", "allow_create_board", "false"))
		expectNoError(t, err)

		expectEqual(t, clientOf(ctx).calls[0].Pathname, "/api/get_mi_board_list")
		expectTrue(t, hasStaleWarning(payload), "no stale warning")
	})
}

// ---------------------------------------------------------------------------
// 2026-09-19 の MCP 実利用報告への対応（ADR-0628 ほか）
// ---------------------------------------------------------------------------
func TestHandleWriteToolCall20260919Additions(t *testing.T) {
	t.Run("an update whose values all equal the current version is rejected instead of stacking a version", func(t *testing.T) {
		ctx := makeWriteCtx(nil)
		clientOf(ctx).mockResolvedValueOnce(obj("mi_histories", arr(obj("id", "m1", "title", "same", "is_checked", false, "limit_time", "2026-01-01T00:00:00+09:00", "data_type", "mi_check"))))
		// 同じ瞬間を別のオフセットで書いても「同じ」
		_, err := HandleWriteToolCall(ctx, "gkill_update_mi", obj("id", "m1", "title", "same", "limit_time", "2025-12-31T15:00:00Z"))
		expectErrorMatches(t, err, `No effective change`)
		expectEqual(t, len(clientOf(ctx).calls), 1)
	})

	t.Run("update_mi clears limit_time with null and reports the entity data_type", func(t *testing.T) {
		ctx := makeWriteCtx(nil)
		clientOf(ctx).
			mockResolvedValueOnce(obj("mi_histories", arr(obj("id", "m1", "title", "t", "limit_time", "2026-01-01T00:00:00+09:00", "data_type", "mi_check")))).
			mockResolvedValueOnce(obj("updated_mi", obj("id", "m1", "data_type", "mi_create"), "updated_kyou", obj("id", "m1")))
		result, err := HandleWriteToolCall(ctx, "gkill_update_mi", obj("id", "m1", "limit_time", nil))
		expectNoError(t, err)

		sent := objAt(t, clientOf(ctx).calls[1].Body, "mi")
		expectTrue(t, sent.Has("limit_time") && sent.Value("limit_time") == nil, "limit_time is not null")
		expectEqual(t, objAt(t, result, "updated_mi").Value("data_type"), "mi")
	})

	t.Run("clearing a limit_time that is already empty is a no-op and is rejected", func(t *testing.T) {
		ctx := makeWriteCtx(nil)
		clientOf(ctx).mockResolvedValueOnce(obj("mi_histories", arr(obj("id", "m1", "title", "t", "limit_time", nil, "data_type", "mi_check"))))
		_, err := HandleWriteToolCall(ctx, "gkill_update_mi", obj("id", "m1", "limit_time", nil))
		expectErrorMatches(t, err, `No effective change`)
	})

	t.Run("delete reports the entity data_type and the summary names the id", func(t *testing.T) {
		ctx := makeWriteCtx(nil)
		clientOf(ctx).
			mockResolvedValueOnce(obj("mi_histories", arr(obj("id", "m1", "is_deleted", false, "data_type", "mi_check")))).
			mockResolvedValueOnce(obj("updated_mi", obj("id", "m1", "data_type", "mi_create"), "updated_kyou", obj("id", "m1")))
		result, err := HandleWriteToolCall(ctx, "gkill_delete_kyou", obj("id", "m1", "data_type", "mi_start"))
		expectNoError(t, err)

		expectEqual(t, objAt(t, result, "updated_mi").Value("data_type"), "mi")
		expectEqual(t, writeSummary(t, "gkill_delete_kyou", result), "Deleted (soft): mi m1")
	})

	t.Run("add_tag names the target_id when the server says the target does not exist", func(t *testing.T) {
		ctx := makeWriteCtx(rejecting(NewGkillApiError("ERR000092: タグ追加に失敗しました [kind=not_found]", obj(
			"errors", arr(obj("error_code", "ERR000092", "error_kind", "not_found")),
		))))
		_, err := HandleWriteToolCall(ctx, "gkill_add_tag", obj("tag", "t", "target_id", "nope"))
		expectErrorMatches(t, err, `target_id "nope" matched no kyou`)
	})

	t.Run("add_text leaves other failures untouched", func(t *testing.T) {
		ctx := makeWriteCtx(rejecting(NewGkillApiError("ERR000057: dup [kind=conflict]", obj("errors", arr(obj("error_kind", "conflict"))))))
		_, err := HandleWriteToolCall(ctx, "gkill_add_text", obj("text", "t", "target_id", "k1"))
		expectErrorMatches(t, err, `ERR000057`)
	})
}
