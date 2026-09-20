package mcp

// read_handlers.go の v2 機能のテスト。
//   - get_kyous v2: 新パラメータの転送・応答の素通し（total_count はカーソルページで作らない）
//   - application_config: fields 射影 + UI 状態キーの strip
//   - GPS: MCP 側ページング（複合カーソル・同一時刻ラン跨ぎ・count_only・日別バケット）
//   - rep_infos: ディスパッチと ApplyFileLinks 不変（file-link トークンの誤発行防止）
//   - idf_file: /files/ クエリ組み立て（?is_video=true&thumb=WxH）・thumb エコー・サイズ上限超過の案内

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/mcp/jsonobj"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

type callApiImpl func(pathname string, body *jsonobj.Object) (*jsonobj.Object, error)

// makeCtx は Node の makeCtx（client.callApi のモック + sid + ローカルトランスポート）。
func makeCtx(impl callApiImpl) *CallContext {
	client := &mockClient{
		login: func() (string, error) { return "mock-session", nil },
	}
	if impl == nil {
		impl = func(_ string, _ *jsonobj.Object) (*jsonobj.Object, error) {
			return obj("errors", arr(), "messages", arr()), nil
		}
	}
	client.callApi = func(pathname string, body *jsonobj.Object, _ bool, _ string) (*jsonobj.Object, error) {
		return impl(pathname, body)
	}
	return &CallContext{Client: client, SID: "sid-1", IsLocalTransport: true}
}

func clientOf(ctx *CallContext) *mockClient { return ctx.Client.(*mockClient) }

func resolving(body func() *jsonobj.Object) callApiImpl {
	return func(_ string, _ *jsonobj.Object) (*jsonobj.Object, error) { return body(), nil }
}

func resolvingJSON(src string) callApiImpl {
	return func(_ string, _ *jsonobj.Object) (*jsonobj.Object, error) {
		return jsonobj.MustUnmarshal(src).(*jsonobj.Object), nil
	}
}

func rejecting(err error) callApiImpl {
	return func(_ string, _ *jsonobj.Object) (*jsonobj.Object, error) { return nil, err }
}

// expectCalledWith は toHaveBeenCalledWith(pathname, body, requiresAuth, sid)。
func expectCalledWith(t *testing.T, client *mockClient, pathname string, body *jsonobj.Object, requiresAuth bool, sid string) {
	t.Helper()
	for _, call := range client.calls {
		if call.Pathname == pathname && jsonobj.Equal(call.Body, body) && call.RequiresAuth == requiresAuth && call.SID == sid {
			return
		}
	}
	t.Fatalf("no call matched %s %s auth=%v sid=%s; calls: %d", pathname, jsonobj.MarshalString(body), requiresAuth, sid, len(client.calls))
}

// expectFetchCalledWith は fetchFile の toHaveBeenCalledWith(path, sid)。
func expectFetchCalledWith(t *testing.T, client *mockClient, path string, sid string) {
	t.Helper()
	for _, call := range client.fetchCalls {
		if call.Pathname == path && call.SID == sid {
			return
		}
	}
	got := []string{}
	for _, call := range client.fetchCalls {
		got = append(got, call.Pathname+" / "+call.SID)
	}
	t.Fatalf("fetchFile was not called with %s / %s; calls: %v", path, sid, got)
}

// logRecord は recordingLogger が捕まえた1件。
type logRecord struct {
	Level slog.Level
	Msg   string
	Attrs *jsonobj.Object
}

// recordingHandler は slog.Handler の最小実装（テスト用）。
type recordingHandler struct {
	mu      sync.Mutex
	records []logRecord
	attrs   []slog.Attr
}

func (h *recordingHandler) Enabled(_ context.Context, _ slog.Level) bool { return true }
func (h *recordingHandler) Handle(_ context.Context, r slog.Record) error {
	attrs := jsonobj.New()
	for _, a := range h.attrs {
		attrs.Set(a.Key, a.Value.Any())
	}
	r.Attrs(func(a slog.Attr) bool {
		attrs.Set(a.Key, a.Value.Any())
		return true
	})
	h.mu.Lock()
	h.records = append(h.records, logRecord{Level: r.Level, Msg: r.Message, Attrs: attrs})
	h.mu.Unlock()
	return nil
}
func (h *recordingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &recordingHandler{attrs: append(append([]slog.Attr{}, h.attrs...), attrs...)}
}
func (h *recordingHandler) WithGroup(_ string) slog.Handler { return h }

func (h *recordingHandler) find(msg string) (logRecord, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, r := range h.records {
		if r.Msg == msg {
			return r, true
		}
	}
	return logRecord{}, false
}

func (h *recordingHandler) count(msg string) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	n := 0
	for _, r := range h.records {
		if r.Msg == msg {
			n++
		}
	}
	return n
}

// newRecordingLogger は記録用の Logger（vi.fn() の warn / info / error に相当）。
func newRecordingLogger() (*Logger, *recordingHandler) {
	h := &recordingHandler{}
	return NewLogger(slog.New(h)), h
}

func statusServer() *Server {
	return &Server{
		ServerKind:       "readwrite",
		ServerName:       "gkill-readwrite-mcp",
		ServerVersion:    "1.2.3",
		SchemaRevision:   "0123456789ab",
		Tools:            make([]*jsonobj.Object, 32),
		IsLocalTransport: false,
		StartedAt:        time.Now().Add(-90 * time.Second),
	}
}

func readSummary(t *testing.T, name string, payload *jsonobj.Object) string {
	t.Helper()
	summary, ok := SummarizeReadToolPayload(name, payload)
	expectTrue(t, ok, "%s: no summary", name)
	return summary
}

func repNamesOf(t *testing.T, payload *jsonobj.Object, key string) []string {
	t.Helper()
	out := []string{}
	for _, item := range arrAt(t, payload, key) {
		out = append(out, strAt(t, objAt(t, item), "rep_name"))
	}
	return out
}

// ---------------------------------------------------------------------------
// get_kyous v2
// ---------------------------------------------------------------------------
func TestHandleReadToolCallGetKyousV2(t *testing.T) {
	t.Run("forwards v2 params and does not forward deprecated include flags", func(t *testing.T) {
		ctx := makeCtx(resolving(func() *jsonobj.Object {
			return obj("kyous", arr(), "total_count", 0, "returned_count", 0, "remaining_count", 0, "has_more", false)
		}))
		// count_only と group_by は併用できない（2026-09-18 以降はエラー）ので、group_by は別の呼び出しで確かめる
		_, err := HandleReadToolCall(ctx, "gkill_get_kyous", obj(
			"count_only", true,
			"data_types", strs("nlog"),
			"create_apps", strs("appA"),
			"update_apps", strs("appB"),
			"num_min", 100,
			"num_max", 500,
			"idf_kinds", strs("image"),
			"include_file_size", true,
			"include_id", true, // deprecated: 受理はするが転送しない
		))
		expectNoError(t, err)
		call := clientOf(ctx).calls[0]
		expectEqual(t, call.Pathname, "/api/get_kyous_mcp")
		body := call.Body
		expectEqual(t, body.Value("count_only"), true)
		_, err = HandleReadToolCall(ctx, "gkill_get_kyous", obj("group_by", "month"))
		expectNoError(t, err)
		expectEqual(t, clientOf(ctx).calls[1].Body.Value("group_by"), "month")
		expectEqual(t, clientOf(ctx).calls[1].Body.Value("count_only"), false)
		expectEqual(t, body.Value("data_types"), strs("nlog"))
		expectEqual(t, body.Value("create_apps"), strs("appA"))
		expectEqual(t, body.Value("update_apps"), strs("appB"))
		expectEqual(t, body.Value("num_min"), 100)
		expectEqual(t, body.Value("num_max"), 500)
		expectEqual(t, body.Value("idf_kinds"), strs("image"))
		expectEqual(t, body.Value("include_file_size"), true)
		expectTrue(t, !body.Has("include_id"), "include_id forwarded")
		expectTrue(t, !body.Has("include_rep_name"), "include_rep_name forwarded")
	})

	t.Run("passes opaque composite cursor verbatim", func(t *testing.T) {
		ctx := makeCtx(resolving(func() *jsonobj.Object {
			return obj("kyous", arr(), "returned_count", 0, "remaining_count", 0, "has_more", false)
		}))
		cursor := "2026-08-01T20:00:00.123456789+09:00::plugin::id::with::colons"
		_, err := HandleReadToolCall(ctx, "gkill_get_kyous", obj("cursor", cursor))
		expectNoError(t, err)
		expectEqual(t, clientOf(ctx).calls[0].Body.Value("cursor"), cursor)
	})

	t.Run("omits total_count on cursor pages instead of fabricating 0", func(t *testing.T) {
		// 旧実装は total_count ?? 0 で埋めており、カーソルページで嘘の0を作っていた
		ctx := makeCtx(resolving(func() *jsonobj.Object {
			return obj("kyous", arr(obj("id", "a")), "returned_count", 1, "remaining_count", 3, "has_more", true, "next_cursor", "t::a")
		}))
		// カーソルは形を検証するようになったので、実物と同じ `{RFC3339}::{id}` を使う
		payload, err := HandleReadToolCall(ctx, "gkill_get_kyous", obj("cursor", "2026-08-24T03:00:00+09:00::z"))
		expectNoError(t, err)
		expectTrue(t, !payload.Has("total_count"), "total_count fabricated")
		expectEqual(t, payload.Value("remaining_count"), 3)
		expectEqual(t, payload.Value("next_cursor"), "t::a")
	})

	t.Run("passes through buckets and warnings", func(t *testing.T) {
		ctx := makeCtx(resolving(func() *jsonobj.Object {
			return obj(
				"kyous", arr(),
				"total_count", 7,
				"returned_count", 0,
				"remaining_count", 0,
				"has_more", false,
				"buckets", arr(obj("key", "2026-01", "count", 7)),
				"warnings", strs(`unknown rep_type "Kmemo"`),
			)
		}))
		payload, err := HandleReadToolCall(ctx, "gkill_get_kyous", obj("group_by", "month"))
		expectNoError(t, err)
		expectEqual(t, payload.Value("buckets"), arr(obj("key", "2026-01", "count", 7)))
		expectEqual(t, payload.Value("warnings"), strs(`unknown rep_type "Kmemo"`))
		expectEqual(t, payload.Value("total_count"), 7)
	})
}

// ---------------------------------------------------------------------------
// application_config
// ---------------------------------------------------------------------------
func appConfigFixture() *jsonobj.Object {
	return obj(
		"tag_struct", obj(
			"name", "root",
			"id", "uuid-root",
			"key", "__root__",
			"is_checked", true,
			"indeterminate", false,
			"seq_in_parent", 1,
			"check_when_inited", true,
			"is_force_hide", false,
			"children", arr(obj(
				"name", "tagA",
				"tag_name", "tagA",
				"id", "uuid-a",
				"key", "tagA",
				"is_checked", false,
				"check_when_inited", true,
				"is_force_hide", true,
			)),
		),
		"mi_board_struct", obj("name", "boards"),
		"rep_struct", obj("name", "reps"),
		"rep_type_struct", obj("name", "types"),
		"device_struct", obj("name", "devices"),
		"kftl_template_struct", obj("name", "templates"),
		"mi_default_board", "Inbox",
		"show_tags_in_list", true,
	)
}

func TestHandleReadToolCallGetApplicationConfig(t *testing.T) {
	t.Run("strips UI-state keys by default but keeps check_when_inited / is_force_hide", func(t *testing.T) {
		ctx := makeCtx(resolving(func() *jsonobj.Object { return obj("application_config", appConfigFixture()) }))
		payload, err := HandleReadToolCall(ctx, "gkill_get_application_config", obj())
		expectNoError(t, err)
		tagStruct := objAt(t, payload, "tag_struct")
		expectTrue(t, !tagStruct.Has("is_checked"), "is_checked kept")
		expectTrue(t, !tagStruct.Has("key"), "key kept")
		expectTrue(t, !tagStruct.Has("id"), "id kept")
		expectTrue(t, !tagStruct.Has("seq_in_parent"), "seq_in_parent kept")
		expectEqual(t, tagStruct.Value("check_when_inited"), true)
		child := objAt(t, arrAt(t, tagStruct, "children")[0])
		expectEqual(t, child.Value("is_force_hide"), true)
		expectEqual(t, child.Value("tag_name"), "tagA")
	})

	// どのアカウントに繋がっているかを答えられること。
	// read サーバと readwrite サーバが別アカウントを向いていても AI から区別できず、
	// 「同じ API なのに件数が違う」「query.ids が壊れている」と誤診されていた
	// （2026-08-24 の実利用レビュー）。gkill は元から返しており、射影が捨てていただけ。
	t.Run("exposes user_id / device so the caller can tell which account it is on", func(t *testing.T) {
		ctx := makeCtx(resolving(func() *jsonobj.Object {
			return obj("application_config", appConfigFixture().Merge(obj("user_id", "testuser", "device", "testdevice")))
		}))
		payload, err := HandleReadToolCall(ctx, "gkill_get_application_config", obj("fields", strs("user_id", "device")))
		expectNoError(t, err)
		expectEqual(t, payload, obj("user_id", "testuser", "device", "testdevice"))
	})

	t.Run("fields projection returns only the requested fields", func(t *testing.T) {
		ctx := makeCtx(resolving(func() *jsonobj.Object { return obj("application_config", appConfigFixture()) }))
		payload, err := HandleReadToolCall(ctx, "gkill_get_application_config", obj("fields", strs("tag_struct", "mi_default_board")))
		expectNoError(t, err)
		expectEqual(t, sortedCopy(payload.Keys()), []string{"mi_default_board", "tag_struct"})
		expectEqual(t, payload.Value("mi_default_board"), "Inbox")
	})

	t.Run("include_ui_state: true keeps everything", func(t *testing.T) {
		ctx := makeCtx(resolving(func() *jsonobj.Object { return obj("application_config", appConfigFixture()) }))
		payload, err := HandleReadToolCall(ctx, "gkill_get_application_config", obj("include_ui_state", true))
		expectNoError(t, err)
		expectEqual(t, objAt(t, payload, "tag_struct").Value("is_checked"), true)
		expectEqual(t, objAt(t, payload, "tag_struct").Value("key"), "__root__")
	})
}

// ---------------------------------------------------------------------------
// gkill_status（2026-09-14 レビュー P0: クライアントの古い一覧を AI 自身が見分ける経路）
// ---------------------------------------------------------------------------
func TestHandleReadToolCallGkillStatus(t *testing.T) {
	t.Run("returns the server description plus the account and build gkill reports", func(t *testing.T) {
		ctx := makeCtx(resolving(func() *jsonobj.Object {
			return obj("application_config", obj(
				"user_id", "testuser",
				"device", "testdevice",
				"version", "9.9.9",
				"commit_hash", "abcdef0",
				"build_time", "2026-09-14T00:00:00+09:00",
				"tag_struct", obj("name", "root"),
			))
		}))
		ctx.Server = statusServer()
		payload, err := HandleReadToolCall(ctx, "gkill_status", obj())
		expectNoError(t, err)
		expectEqual(t, payload.Value("server_kind"), "readwrite")
		expectEqual(t, payload.Value("server_name"), "gkill-readwrite-mcp")
		expectEqual(t, payload.Value("server_version"), "1.2.3")
		expectEqual(t, payload.Value("schema_revision"), "0123456789ab")
		expectEqual(t, payload.Value("tool_count"), 32)
		expectEqual(t, payload.Value("transport"), "http")
		mustMatch(t, strAt(t, payload, "started_at"), `^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}[+-]\d{2}:\d{2}$`)
		expectTrue(t, floatOf(t, payload.Value("uptime_seconds")) >= 89, "uptime %v", payload.Value("uptime_seconds"))
		expectEqual(t, payload.Value("gkill_reachable"), true)
		expectEqual(t, payload.Value("account"), obj("user_id", "testuser", "device", "testdevice"))
		expectEqual(t, payload.Value("gkill"), obj("version", "9.9.9", "commit_hash", "abcdef0", "build_time", "2026-09-14T00:00:00+09:00"))
		// ApplicationConfig の残り（tag_struct 等）は載せない
		expectTrue(t, !payload.Has("tag_struct"), "tag_struct leaked")
		expectCalledWith(t, clientOf(ctx), "/api/get_application_config", obj(), true, "sid-1")
	})

	// gkill へ届かなくても MCP 側の情報は返す。「MCP は生きているが gkill が落ちている」を
	// 区別できるのがこのツールの仕事で、失敗させるとその区別が消える。
	t.Run("still answers when gkill is unreachable, without leaking the error text", func(t *testing.T) {
		logger, recorder := newRecordingLogger()
		ctx := makeCtx(rejecting(NewGkillApiError("connect ECONNREFUSED http://127.0.0.1:9999/api/get_application_config", obj("status", 503))))
		ctx.Server = statusServer()
		ctx.Log = logger
		payload, err := HandleReadToolCall(ctx, "gkill_status", obj())
		expectNoError(t, err)
		expectEqual(t, payload.Value("gkill_reachable"), false)
		expectEqual(t, payload.Value("gkill_error"), "HTTP 503")
		expectEqual(t, payload.Value("schema_revision"), "0123456789ab")
		expectTrue(t, !payload.Has("account"), "account set")
		// 接続先 URL を含みうる本文は応答に載せず、ログにだけ残す（ADR-0707）
		mustNotContain(t, jsonobj.MarshalString(payload), "127.0.0.1")
		record, ok := recorder.find("status_gkill_unreachable")
		expectTrue(t, ok, "status_gkill_unreachable was not logged")
		mustContain(t, jsString(record.Attrs.Value("error")), "ECONNREFUSED")
	})

	t.Run("reports 'unreachable' when the failure carries no HTTP status", func(t *testing.T) {
		ctx := makeCtx(rejecting(errors.New("socket hang up")))
		ctx.Server = statusServer()
		payload, err := HandleReadToolCall(ctx, "gkill_status", obj())
		expectNoError(t, err)
		expectEqual(t, payload.Value("gkill_reachable"), false)
		expectEqual(t, payload.Value("gkill_error"), "unreachable")
	})

	// 接続先 URL を detail に持つ Network error も "unreachable" に畳む（URL は載せない）
	t.Run("folds a network error (whose detail carries the URL) into 'unreachable'", func(t *testing.T) {
		ctx := makeCtx(rejecting(NewGkillApiError("Network error at /api/get_application_config.", obj("url", "https://127.0.0.1:9999/api/get_application_config", "message", "ECONNREFUSED"))))
		ctx.Server = statusServer()
		payload, err := HandleReadToolCall(ctx, "gkill_status", obj())
		expectNoError(t, err)
		expectEqual(t, payload.Value("gkill_error"), "unreachable")
		mustNotContain(t, jsonobj.MarshalString(payload), "127.0.0.1")
	})

	// 資格情報の誤りは何度呼んでも直らず、ログインの回数制限を食う。名指しで止める。
	t.Run("names a login failure with its error code and tells the caller not to retry", func(t *testing.T) {
		ctx := makeCtx(rejecting(NewGkillApiError("Login failed: ERR000005: ユーザIDまたはパスワードが違います", obj(
			"errors", arr(obj("error_code", "ERR000005", "error_message", "ユーザIDまたはパスワードが違います")),
			"messages", nil,
		))))
		ctx.Server = statusServer()
		payload, err := HandleReadToolCall(ctx, "gkill_status", obj())
		expectNoError(t, err)
		expectEqual(t, payload.Value("gkill_reachable"), false)
		mustMatch(t, strAt(t, payload, "gkill_error"), `^login_failed \(ERR000005\) — do not retry`)
		mustMatch(t, strAt(t, payload, "gkill_error"), `rate limit`)
	})

	t.Run("names an API error by its error code only", func(t *testing.T) {
		ctx := makeCtx(rejecting(NewGkillApiError("API error at /api/get_application_config: ERR000999: x", obj(
			"errors", arr(obj("error_code", "ERR000999", "error_message", "x")),
		))))
		ctx.Server = statusServer()
		payload, err := HandleReadToolCall(ctx, "gkill_status", obj())
		expectNoError(t, err)
		expectEqual(t, payload.Value("gkill_error"), "api_error (ERR000999)")
	})

	t.Run("takes no arguments and names the stale-list possibility when one is sent", func(t *testing.T) {
		ctx := makeCtx(nil)
		ctx.Server = statusServer()
		_, err := HandleReadToolCall(ctx, "gkill_status", obj("locale_name", "ja"))
		expectErrorMatches(t, err, `(?s)arguments\.locale_name.*is not supported.*stale`)
	})

	t.Run("summary names the account, the kind and the revision", func(t *testing.T) {
		expectEqual(t, readSummary(t, "gkill_status", obj(
			"server_kind", "read",
			"schema_revision", "0123456789ab",
			"uptime_seconds", 42,
			"gkill_reachable", true,
			"account", obj("user_id", "testuser", "device", "testdevice"),
		)), "Connected to testuser@testdevice via read server (schema_revision 0123456789ab, up 42s).")
		expectEqual(t, readSummary(t, "gkill_status", obj(
			"server_kind", "read",
			"schema_revision", "0123456789ab",
			"uptime_seconds", 42,
			"gkill_reachable", false,
			"gkill_error", "HTTP 503",
		)), "MCP read server is up (schema_revision 0123456789ab, up 42s) but gkill is NOT reachable (HTTP 503).")
	})

	t.Run("isReadToolName covers gkill_status", func(t *testing.T) {
		expectTrue(t, IsReadToolName("gkill_status"), "gkill_status is not a read tool")
	})
}

func TestStripAppConfigUiState(t *testing.T) {
	t.Run("recurses arrays and objects, leaves scalars", func(t *testing.T) {
		stripped := StripAppConfigUiState(obj(
			"key", "drop-me",
			"keep", arr(obj("id", "drop", "name", "keep")),
			"name", "n",
		))
		expectEqual(t, stripped, obj("keep", arr(obj("name", "keep")), "name", "n"))
	})
}

// ---------------------------------------------------------------------------
// GPS ページング（純関数）
// ---------------------------------------------------------------------------
func gpsPoint(t string, lat float64) *jsonobj.Object {
	return obj("related_time", t, "latitude", lat, "longitude", 139.5)
}

// サーバの並び: 時刻降順・座標タイブレーク（決定的）
func gpsLogsFixture() []any {
	return arr(
		gpsPoint("2026-08-03T12:00:00+09:00", 35.31),
		gpsPoint("2026-08-02T12:00:00+09:00", 35.31),
		gpsPoint("2026-08-02T12:00:00+09:00", 35.32),
		gpsPoint("2026-08-02T12:00:00+09:00", 35.33),
		gpsPoint("2026-08-01T12:00:00+09:00", 35.31),
	)
}

func gpsKeys(t *testing.T, points []any) *StringSet {
	t.Helper()
	keys := NewStringSet()
	for _, p := range points {
		point := objAt(t, p)
		keys.Add(strAt(t, point, "related_time") + "/" + jsString(point.Value("latitude")))
	}
	return keys
}

func TestPaginateGpsLogs(t *testing.T) {
	logs := gpsLogsFixture()

	t.Run("count_only returns only total_count", func(t *testing.T) {
		result, err := PaginateGpsLogs(logs, obj("count_only", true, "limit", 500))
		expectNoError(t, err)
		expectEqual(t, result, obj("gps_logs", arr(), "total_count", 5, "returned_count", 0, "remaining_count", 0, "has_more", false))
	})

	t.Run("group_by day returns ascending daily buckets", func(t *testing.T) {
		result, err := PaginateGpsLogs(logs, obj("group_by", "day", "limit", 500))
		expectNoError(t, err)
		expectEqual(t, result.Value("buckets"), arr(
			obj("key", "2026-08-01", "count", 1),
			obj("key", "2026-08-02", "count", 3),
			obj("key", "2026-08-03", "count", 1),
		))
		expectEqual(t, result.Value("total_count"), 5)
	})

	t.Run("walks all points exactly once with limit=2 across a same-time run", func(t *testing.T) {
		// 同一時刻3点のランがページ境界(limit=2)をまたぐ形。カーソルが位置を保てないと
		// 重複または取りこぼしが出る
		seen := []any{}
		var cursor any = jsonobj.Undefined
		for range 10 {
			options := obj("limit", 2)
			if !jsonobj.IsUndefined(cursor) {
				options.Set("cursor", cursor)
			}
			page, err := PaginateGpsLogs(logs, options)
			expectNoError(t, err)
			seen = append(seen, arrAt(t, page, "gps_logs")...)
			if page.Value("has_more") != true {
				expectEqual(t, page.Value("remaining_count"), 0)
				break
			}
			expectTrue(t, page.Defined("next_cursor"), "next_cursor missing")
			cursor = page.Value("next_cursor")
		}
		expectEqual(t, len(seen), 5)
		// 全点がちょうど1回ずつ
		expectEqual(t, gpsKeys(t, seen).Len(), 5)
		// 1ページ目だけ total_count
		first, err := PaginateGpsLogs(logs, obj("limit", 2))
		expectNoError(t, err)
		expectEqual(t, first.Value("total_count"), 5)
		expectEqual(t, floatOf(t, first.Value("returned_count"))+floatOf(t, first.Value("remaining_count")), 5)
		second, err := PaginateGpsLogs(logs, obj("limit", 2, "cursor", first.Value("next_cursor")))
		expectNoError(t, err)
		expectTrue(t, !second.Has("total_count"), "total_count on a cursor page")
	})

	t.Run("count_only and group_by reject cursor", func(t *testing.T) {
		_, err := PaginateGpsLogs(logs, obj("count_only", true, "cursor", "x", "limit", 1))
		expectGkillApiError(t, err)
		_, err = PaginateGpsLogs(logs, obj("group_by", "day", "cursor", "x", "limit", 1))
		expectGkillApiError(t, err)
	})

	// GPS 側も count_only の早期 return が group_by を黙って捨てていた（get_kyous と同じ順序の同じ穴）。
	t.Run("count_only and group_by reject each other instead of silently dropping the buckets", func(t *testing.T) {
		_, err := PaginateGpsLogs(logs, obj("count_only", true, "group_by", "day", "limit", 1))
		expectErrorMatches(t, err, `(?s)count_only.*cannot be combined with group_by`)
	})

	t.Run("cursor roundtrip and invalid cursor", func(t *testing.T) {
		cursor := EncodeGpsCursor("2026-08-02T12:00:00+09:00", 2)
		decoded, err := DecodeGpsCursor(cursor)
		expectNoError(t, err)
		expectEqual(t, obj("t", decoded.T, "n", decoded.N), obj("t", "2026-08-02T12:00:00+09:00", "n", 2))
		_, err = DecodeGpsCursor("not-base64-json")
		expectGkillApiError(t, err)
	})

	// 発行(PaginateGpsLogs) と 受理(NormalizeGpsArgs) の境界をまたぐ回帰テスト。
	//
	// 両者は別々にテストされていたが往復が一度も通されておらず、
	// NormalizeGpsArgs が get_kyous 用の `{RFC3339}::{ID}` 検証をコピペしたまま
	// 残っていたために「説明文どおり next_cursor を verbatim で渡すと 100% 失敗する」
	// 状態が出荷されていた（2026-08-24 の実利用報告）。片側だけのテストでは検出できない。
	t.Run("next_cursor survives normalizeGpsArgs and pages the whole list exactly once", func(t *testing.T) {
		seen := []any{}
		var cursor any = jsonobj.Undefined
		for range 10 {
			// 実際の呼び出しと同じ経路: クライアントが渡した引数を normalize してから paginate する
			args := obj("start_date", "2026-08-01", "end_date", "2026-08-03", "limit", 2)
			if !jsonobj.IsUndefined(cursor) {
				args.Set("cursor", cursor)
			}
			normalized, err := NormalizeGpsArgs(args)
			expectNoError(t, err)
			page, err := PaginateGpsLogs(logs, normalized)
			expectNoError(t, err)
			seen = append(seen, arrAt(t, page, "gps_logs")...)
			if page.Value("has_more") != true {
				break
			}
			cursor = page.Value("next_cursor")
		}
		expectEqual(t, len(seen), 5)
		expectEqual(t, gpsKeys(t, seen).Len(), 5)
	})
}

// ---------------------------------------------------------------------------
// rep 名の絞り込み（純関数）
// ---------------------------------------------------------------------------
func TestPaginateRepNames(t *testing.T) {
	names := strs("Fitbit", "GoogleLocation", "Kmemo", "kmemo_backup", "Tag")

	t.Run("returns everything within limit and reports no truncation", func(t *testing.T) {
		expectEqual(t, PaginateRepNames(names, obj("limit", 200)), obj(
			"rep_names", names,
			"total_count", 5,
			"returned_count", 5,
			"truncated", false,
		))
	})

	t.Run("contains matches case-insensitively", func(t *testing.T) {
		result := PaginateRepNames(names, obj("contains", "KMEMO", "limit", 200))
		expectEqual(t, result.Value("rep_names"), strs("Kmemo", "kmemo_backup"))
		expectEqual(t, result.Value("total_count"), 2)
	})

	// total_count は「絞り込み後・limit 適用前」。limit 前の件数を返さないと
	// 何件一致したのかが読めず、truncated の意味も決まらない
	t.Run("total_count counts matches before limit", func(t *testing.T) {
		result := PaginateRepNames(names, obj("contains", "kmemo", "limit", 1))
		expectEqual(t, result.Value("rep_names"), strs("Kmemo"))
		expectEqual(t, result.Value("returned_count"), 1)
		expectEqual(t, result.Value("total_count"), 2)
		expectEqual(t, result.Value("truncated"), true)
	})
}

// ---------------------------------------------------------------------------
// rep_infos
// ---------------------------------------------------------------------------

// rowFilterResponse は行の絞り込み（2026-09-18 の実利用報告）用の、本番の縮図:
// Kyou rep は rep_type ごとに1行、Archived Git の rep 名は plugins[] に、
// 歴代端末の Tag_ / Text_ / GPSLogs_ は attached_data_reps[] に並ぶ。
func rowFilterResponse() *jsonobj.Object {
	return jsonobj.MustUnmarshal(`{
		"rep_infos": [
			{"rep_name": "Kmemo_pc", "rep_type": "kmemo", "use_to_write": true},
			{"rep_name": "Kmemo_phone_2024", "rep_type": "kmemo", "use_to_write": false},
			{"rep_name": "Files_pc", "rep_type": "directory", "use_to_write": true}
		],
		"canonical_rep_types": ["kmemo", "directory", "mi"],
		"plugins": [
			{"rep_name": "archived_git_alpha", "data_type": "git_commit_log", "plugin_name": "archived"},
			{"rep_name": "archived_git_beta", "data_type": "git_commit_log", "plugin_name": "archived"}
		],
		"attached_data_reps": [
			{"rep_name": "Tag_pc", "data_kind": "tag", "use_to_write": true},
			{"rep_name": "Tag_phone_2024", "data_kind": "tag", "use_to_write": false},
			{"rep_name": "Text_pc", "data_kind": "text", "use_to_write": true},
			{"rep_name": "GPSLogs_phone_2024", "data_kind": "gpslog", "use_to_write": false}
		]
	}`).(*jsonobj.Object)
}

func TestHandleReadToolCallGetRepInfos(t *testing.T) {
	// 本番では rep_infos[] だけで数百件になるのに、
	// 「正準値と対応表だけ欲しい」呼び出しが一番多かった（2026-08-24 の実利用レビュー）。
	t.Run("fields projection can skip the large rep_infos list", func(t *testing.T) {
		ctx := makeCtx(resolvingJSON(`{
			"rep_infos": [{"rep_name": "Kmemo", "rep_type": "kmemo"}],
			"canonical_rep_types": ["kmemo", "kc"],
			"plugins": [{"rep_name": "ChatGPT", "data_type": "chatgpt_conversation", "plugin_name": "p"}],
			"attached_data_reps": [{"rep_name": "Tag", "data_kind": "tag"}]
		}`))
		payload, err := HandleReadToolCall(ctx, "gkill_get_rep_infos", obj("fields", strs("canonical_rep_types", "plugins", "attached_data_reps")))
		expectNoError(t, err)
		expectEqual(t, sortedCopy(payload.Keys()), []string{"attached_data_reps", "canonical_rep_types", "plugins"})
		expectTrue(t, !payload.Has("rep_infos"), "rep_infos returned")
		// 絞り込みは MCP 側。gkill には fields の受け口が無いので送らない
		expectCalledWith(t, clientOf(ctx), "/api/get_rep_infos_mcp", obj(), true, "sid-1")
	})

	t.Run("dispatches to /api/get_rep_infos_mcp and passes arrays through", func(t *testing.T) {
		ctx := makeCtx(resolvingJSON(`{
			"rep_infos": [{"rep_name": "Kmemo", "rep_type": "kmemo"}],
			"canonical_rep_types": ["kmemo", "directory"],
			"plugins": [{"rep_name": "ClaudeCode", "data_type": "claude_code_turn", "plugin_name": "gkill_plugin_claudecode"}],
			"attached_data_reps": [
				{"rep_name": "Tag", "data_kind": "tag"},
				{"rep_name": "GPSLog", "data_kind": "gpslog"}
			]
		}`))
		payload, err := HandleReadToolCall(ctx, "gkill_get_rep_infos", obj())
		expectNoError(t, err)
		expectCalledWith(t, clientOf(ctx), "/api/get_rep_infos_mcp", obj(), true, "sid-1")
		expectEqual(t, len(arrAt(t, payload, "rep_infos")), 1)
		expectTrue(t, containsValue(arrAt(t, payload, "canonical_rep_types"), "directory"), "directory missing")
		expectEqual(t, objAt(t, arrAt(t, payload, "plugins")[0]).Value("data_type"), "claude_code_turn")
		// タグ等の書き込み先（query.reps へは渡せない）も素通しで返ること
		expectEqual(t, payload.Value("attached_data_reps"), arr(
			obj("rep_name", "Tag", "data_kind", "tag"),
			obj("rep_name", "GPSLog", "data_kind", "gpslog"),
		))
	})

	t.Run("isReadToolName covers the new tool", func(t *testing.T) {
		expectTrue(t, IsReadToolName("gkill_get_rep_infos"), "gkill_get_rep_infos is not a read tool")
	})

	t.Run("writable_only keeps the write targets only and empties plugins", func(t *testing.T) {
		ctx := makeCtx(resolving(rowFilterResponse))
		payload, err := HandleReadToolCall(ctx, "gkill_get_rep_infos", obj("writable_only", true))
		expectNoError(t, err)
		expectEqual(t, repNamesOf(t, payload, "rep_infos"), []string{"Kmemo_pc", "Files_pc"})
		expectEqual(t, repNamesOf(t, payload, "attached_data_reps"), []string{"Tag_pc", "Text_pc"})
		expectEqual(t, payload.Value("plugins"), arr())
		// 正準値の語彙は行絞り込みの影響を受けない
		expectEqual(t, payload.Value("canonical_rep_types"), strs("kmemo", "directory", "mi"))
	})

	t.Run("『gkill_add_tag はどこへ書くか』は writable_only + data_kinds:[tag] で1行になる", func(t *testing.T) {
		ctx := makeCtx(resolving(rowFilterResponse))
		payload, err := HandleReadToolCall(ctx, "gkill_get_rep_infos", obj(
			"writable_only", true,
			"data_kinds", strs("tag"),
			"fields", strs("attached_data_reps"),
		))
		expectNoError(t, err)
		expectEqual(t, payload, obj("attached_data_reps", arr(obj("rep_name", "Tag_pc", "data_kind", "tag", "use_to_write", true))))
	})

	t.Run("rep_types narrows rep_infos and is checked against canonical_rep_types", func(t *testing.T) {
		ctx := makeCtx(resolving(rowFilterResponse))
		payload, err := HandleReadToolCall(ctx, "gkill_get_rep_infos", obj("rep_types", strs("directory")))
		expectNoError(t, err)
		expectEqual(t, repNamesOf(t, payload, "rep_infos"), []string{"Files_pc"})
		// 他の配列には効かない
		expectEqual(t, len(arrAt(t, payload, "plugins")), 2)
		expectEqual(t, len(arrAt(t, payload, "attached_data_reps")), 4)
		_, err = HandleReadToolCall(ctx, "gkill_get_rep_infos", obj("rep_types", strs("Kmemo")))
		expectErrorMatches(t, err, `(?s)rep_types.*canonical rep types: kmemo, directory, mi`)
	})

	t.Run("rep_names is exact and case-sensitive across all three lists", func(t *testing.T) {
		ctx := makeCtx(resolving(rowFilterResponse))
		payload, err := HandleReadToolCall(ctx, "gkill_get_rep_infos", obj(
			"rep_names", strs("Kmemo_pc", "archived_git_beta", "Text_pc", "tag_pc"),
		))
		expectNoError(t, err)
		expectEqual(t, repNamesOf(t, payload, "rep_infos"), []string{"Kmemo_pc"})
		expectEqual(t, repNamesOf(t, payload, "plugins"), []string{"archived_git_beta"})
		expectEqual(t, repNamesOf(t, payload, "attached_data_reps"), []string{"Text_pc"})
	})

	t.Run("contains is a case-insensitive substring across all three lists", func(t *testing.T) {
		ctx := makeCtx(resolving(rowFilterResponse))
		payload, err := HandleReadToolCall(ctx, "gkill_get_rep_infos", obj("contains", "PHONE_2024"))
		expectNoError(t, err)
		expectEqual(t, repNamesOf(t, payload, "rep_infos"), []string{"Kmemo_phone_2024"})
		expectEqual(t, payload.Value("plugins"), arr())
		expectEqual(t, repNamesOf(t, payload, "attached_data_reps"), []string{"Tag_phone_2024", "GPSLogs_phone_2024"})
	})

	t.Run("applyFileLinks leaves rep_infos payload unchanged (no file-link mint)", func(t *testing.T) {
		// ApplyFileLinks は rep_name+file_name の同居で idf とみなしてトークンを鋳造する。
		// rep_infos / attached_data_reps の行は rep_name を持つが file_name を持たないので、
		// 不変であること（サーバ側が file_name 系キーを同居させない契約の防御線）。
		payload := obj(
			"rep_infos", arr(obj("rep_name", "Kmemo", "rep_type", "kmemo")),
			"canonical_rep_types", strs("kmemo"),
			"plugins", arr(obj("rep_name", "ClaudeCode", "data_type", "claude_code_turn", "plugin_name", "p")),
			"attached_data_reps", arr(obj("rep_name", "Tag", "data_kind", "tag")),
		)
		before := jsonobj.MarshalString(payload)
		store := NewFileLinkStore(0)
		ApplyFileLinks(payload, &FileLinkContext{PublicBaseURL: "https://example.com", Store: store}, "sid")
		expectEqual(t, jsonobj.MarshalString(payload), before)
	})
}

// ---------------------------------------------------------------------------
// gkill_get_kyou_history — 削除済みと過去版を読む唯一の経路
// ---------------------------------------------------------------------------
func TestHandleReadToolCallGetKyouHistory(t *testing.T) {
	t.Run("uses the per-type endpoint, never the type-agnostic /api/get_kyou", func(t *testing.T) {
		// 型非依存の Repositories.GetKyouHistoriesByRepName は UnWrap() で
		// キャッシュ rep を丸ごとバイパスする（11rep→約940rep・実測20.7秒）
		ctx := makeCtx(resolvingJSON(`{"kmemo_histories": [{"id": "k1", "is_deleted": false, "update_time": "2026-01-02T00:00:00+09:00"}]}`))
		_, err := HandleReadToolCall(ctx, "gkill_get_kyou_history", obj("id", "k1", "data_type", "kmemo"))
		expectNoError(t, err)
		expectEqual(t, clientOf(ctx).calls[0].Pathname, "/api/get_kmemo")
		expectTrue(t, clientOf(ctx).calls[0].Pathname != "/api/get_kyou", "used /api/get_kyou")
	})

	t.Run("returns every version newest first and flags a deleted latest version", func(t *testing.T) {
		ctx := makeCtx(resolvingJSON(`{"kmemo_histories": [
			{"id": "k1", "content": "gone", "is_deleted": true, "update_time": "2026-01-03T00:00:00+09:00"},
			{"id": "k1", "content": "second", "is_deleted": false, "update_time": "2026-01-02T00:00:00+09:00"},
			{"id": "k1", "content": "first", "is_deleted": false, "update_time": "2026-01-01T00:00:00+09:00"}
		]}`))
		result, err := HandleReadToolCall(ctx, "gkill_get_kyou_history", obj("id", "k1", "data_type", "kmemo"))
		expectNoError(t, err)

		expectEqual(t, result.Value("latest_is_deleted"), true)
		expectEqual(t, result.Value("version_count"), 3)
		expectEqual(t, result.Value("returned_count"), 3)
		expectEqual(t, result.Value("has_more"), false)
		versions := arrAt(t, result, "versions")
		expectEqual(t, objAt(t, versions[0]).Value("content"), "gone")
		expectEqual(t, objAt(t, versions[2]).Value("content"), "first")
	})

	t.Run("caps the versions by limit and reports has_more", func(t *testing.T) {
		// 履歴は編集のたびに1件伸びるので無制限には返さない
		histories := arr()
		for i := range 5 {
			histories = append(histories, obj("id", "k1", "is_deleted", false, "update_time", "2026-01-0"+itoa(i+1)+"T00:00:00+09:00"))
		}
		ctx := makeCtx(resolving(func() *jsonobj.Object { return obj("kmemo_histories", histories) }))
		result, err := HandleReadToolCall(ctx, "gkill_get_kyou_history", obj("id", "k1", "data_type", "kmemo", "limit", 2))
		expectNoError(t, err)

		expectEqual(t, result.Value("returned_count"), 2)
		expectEqual(t, result.Value("version_count"), 5)
		expectEqual(t, result.Value("has_more"), true)
	})

	t.Run("throws when the id has no history at all", func(t *testing.T) {
		ctx := makeCtx(resolvingJSON(`{"kmemo_histories": []}`))
		_, err := HandleReadToolCall(ctx, "gkill_get_kyou_history", obj("id", "nope", "data_type", "kmemo"))
		expectErrorMatches(t, err, `Entity not found`)
	})

	t.Run("summary names the deleted state so it is visible before reading the JSON", func(t *testing.T) {
		summary := readSummary(t, "gkill_get_kyou_history", obj(
			"version_count", 3, "returned_count", 3, "has_more", false, "latest_is_deleted", true,
		))
		mustContain(t, summary, "3 of 3")
		mustContain(t, summary, "DELETED")
	})
}

// ---------------------------------------------------------------------------
// gkill_get_idf_file — /files/ クエリ組み立てと thumb エコー
// ---------------------------------------------------------------------------
func TestHandleReadToolCallGetIdfFileQuery(t *testing.T) {
	// ?is_video=true&thumb=WxH の順序とエンコードを決める唯一の箇所
	// (payload.go の file_url 注入 / http_transport.go の /files/ 配信 /
	//  クライアントの build_media_url と同じ形であること)。
	makeFileCtx := func(file *FileResponse) *CallContext {
		ctx := makeCtx(nil)
		clientOf(ctx).fetchFile = func(_ string, _ string) (*FileResponse, error) { return file, nil }
		return ctx
	}

	t.Run("builds ?is_video=true&thumb=WxH in that order", func(t *testing.T) {
		ctx := makeFileCtx(&FileResponse{Buffer: []byte("frame"), ContentType: "image/jpeg"})
		_, err := HandleReadToolCall(ctx, "gkill_get_idf_file", obj(
			"rep_name", "Video",
			"file_name", "clip.mp4",
			"is_video", true,
			"thumb", "640x480",
		))
		expectNoError(t, err)
		expectFetchCalledWith(t, clientOf(ctx), "/files/Video/clip.mp4?is_video=true&thumb=640x480", "sid-1")
	})

	t.Run("thumb alone appends only ?thumb=WxH", func(t *testing.T) {
		ctx := makeFileCtx(&FileResponse{Buffer: []byte("img"), ContentType: "image/png"})
		_, err := HandleReadToolCall(ctx, "gkill_get_idf_file", obj(
			"rep_name", "Photo",
			"file_name", "p.png",
			"thumb", "320x240",
		))
		expectNoError(t, err)
		expectFetchCalledWith(t, clientOf(ctx), "/files/Photo/p.png?thumb=320x240", "sid-1")
	})

	t.Run("no thumb and no is_video appends no query string", func(t *testing.T) {
		ctx := makeFileCtx(&FileResponse{Buffer: []byte("img"), ContentType: "image/png"})
		_, err := HandleReadToolCall(ctx, "gkill_get_idf_file", obj("rep_name", "Photo", "file_name", "p.png"))
		expectNoError(t, err)
		expectFetchCalledWith(t, clientOf(ctx), "/files/Photo/p.png", "sid-1")
	})

	t.Run("encodes rep_name and each file_name segment, keeping / separators", func(t *testing.T) {
		ctx := makeFileCtx(&FileResponse{Buffer: []byte("img"), ContentType: "image/png"})
		_, err := HandleReadToolCall(ctx, "gkill_get_idf_file", obj(
			"rep_name", "My Photos",
			"file_name", "2026 08/pic 1.png",
		))
		expectNoError(t, err)
		expectFetchCalledWith(t, clientOf(ctx), "/files/My%20Photos/2026%2008/pic%201.png", "sid-1")
	})

	t.Run("echoes thumb in the payload only when the fetch was downscaled", func(t *testing.T) {
		// thumb エコーは「縮小して取った」ことの唯一の印。原寸と取り違えないための防御線
		downscaled := makeFileCtx(&FileResponse{Buffer: []byte("small"), ContentType: "image/jpeg"})
		withThumb, err := HandleReadToolCall(downscaled, "gkill_get_idf_file", obj(
			"rep_name", "Photo",
			"file_name", "p.png",
			"thumb", "640x480",
		))
		expectNoError(t, err)
		expectEqual(t, withThumb.Value("thumb"), "640x480")

		original := makeFileCtx(&FileResponse{Buffer: []byte("orig"), ContentType: "image/jpeg"})
		withoutThumb, err := HandleReadToolCall(original, "gkill_get_idf_file", obj(
			"rep_name", "Photo",
			"file_name", "p.png",
		))
		expectNoError(t, err)
		expectTrue(t, !withoutThumb.Has("thumb"), "thumb echoed for the original")
	})
}

// ---------------------------------------------------------------------------
// gkill_get_idf_file — サイズ上限超過の案内
// ---------------------------------------------------------------------------
func TestGetIdfFileOversizeMessage(t *testing.T) {
	// payload.go の ApplyFileLinks は is_image のときだけ file_url_full を注入する。
	// 非画像へ file_url_full を案内すると、存在しないフィールドを探させてしまう
	hugeBuffer := make([]byte, MaxIDFFileBytes+1)

	t.Run("advises file_url_full for an oversized image", func(t *testing.T) {
		ctx := makeCtx(nil)
		clientOf(ctx).fetchFile = func(_ string, _ string) (*FileResponse, error) {
			return &FileResponse{Buffer: hugeBuffer, ContentType: "image/png"}, nil
		}
		_, err := HandleReadToolCall(ctx, "gkill_get_idf_file", obj("rep_name", "Photo", "file_name", "big.png"))
		expectGkillApiError(t, err)
		mustContain(t, err.Error(), "file_url_full")
	})

	t.Run("advises file_url (not file_url_full) for an oversized non-image", func(t *testing.T) {
		ctx := makeCtx(nil)
		clientOf(ctx).fetchFile = func(_ string, _ string) (*FileResponse, error) {
			return &FileResponse{Buffer: hugeBuffer, ContentType: "video/mp4"}, nil
		}
		_, err := HandleReadToolCall(ctx, "gkill_get_idf_file", obj("rep_name", "Video", "file_name", "big.mp4"))
		expectGkillApiError(t, err)
		mustContain(t, err.Error(), "file_url")
		mustNotContain(t, err.Error(), "file_url_full")
	})
}

func TestGetIdfFile404(t *testing.T) {
	t.Run("turns the raw HTTP 404 into something that names the likely cause", func(t *testing.T) {
		// 「HTTP 404 fetching file /files/NoSuchRep/x.png」だけだと、rep 名が悪いのか
		// ファイル名が悪いのか、そもそも消えたのかが読めない
		ctx := &CallContext{
			Client: &mockClient{fetchFile: func(_ string, _ string) (*FileResponse, error) {
				return nil, NewGkillApiError("HTTP 404 fetching file /files/NoSuchRep/x.png.", obj("status", 404))
			}},
			SID: "sid-1",
		}
		_, err := HandleReadToolCall(ctx, "gkill_get_idf_file", obj("rep_name", "NoSuchRep", "file_name", "x.png"))
		expectErrorMatches(t, err, `gkill_get_rep_infos`)
	})

	t.Run("passes other failures through untouched", func(t *testing.T) {
		ctx := &CallContext{
			Client: &mockClient{fetchFile: func(_ string, _ string) (*FileResponse, error) {
				return nil, NewGkillApiError("HTTP 500 fetching file /files/r/x.png.", obj("status", 500))
			}},
			SID: "sid-1",
		}
		_, err := HandleReadToolCall(ctx, "gkill_get_idf_file", obj("rep_name", "r", "file_name", "x.png"))
		expectErrorMatches(t, err, `HTTP 500`)
	})
}

func TestZeroResultSummary(t *testing.T) {
	// count_only の応答も通常検索の0件も kyous:[] なので payload からは区別できない。
	// 「Counted 0 entries.」だと、count_only を指定していない呼び出し側に
	// 「集計モードで返ってきた」と読めてしまう。
	t.Run("an empty ordinary result does not claim to have counted", func(t *testing.T) {
		expectEqual(t, readSummary(t, "gkill_get_kyous", obj(
			"kyous", arr(),
			"total_count", 0,
			"returned_count", 0,
			"remaining_count", 0,
			"has_more", false,
		)), "No entries matched.")
	})

	t.Run("count_only with matches still reports the count", func(t *testing.T) {
		expectEqual(t, readSummary(t, "gkill_get_kyous", obj(
			"kyous", arr(),
			"total_count", 12,
			"returned_count", 0,
			"remaining_count", 0,
			"has_more", false,
		)), "Counted 12 entries.")
	})

	t.Run("gps logs follow the same rule", func(t *testing.T) {
		expectEqual(t, readSummary(t, "gkill_get_gps_log", obj("gps_logs", arr(), "total_count", 0, "has_more", false)), "No GPS points matched.")
		expectEqual(t, readSummary(t, "gkill_get_gps_log", obj("gps_logs", arr(), "total_count", 5, "has_more", false)), "Counted 5 GPS points.")
	})
}

// ---------------------------------------------------------------------------
// トップレベル plugins[] の素通し (2026-08-30 レビュー P1)
//
// Go はページに現れたプラグインの説明を rep 名ごと1回だけ応答トップレベルの plugins[]
// で返し、各 Kyou の payload には rep_name / plugin_name しか載せない設計
// (説明の焼き込み排除)。ここがコピーを落とすと、ツール説明が約束しているのに
// 「各 Kyou にもトップレベルにも説明が無い」状態になる。
// ---------------------------------------------------------------------------
func TestHandleReadToolCallTopLevelPluginsPassthrough(t *testing.T) {
	plugins := func() []any {
		return arr(obj("rep_name", "ExamplePluginRep", "plugin_name", "example_plugin", "description", "プラグインの説明文"))
	}
	pageResponse := func(extra *jsonobj.Object) *jsonobj.Object {
		return obj(
			"kyous", arr(obj("id", "k1", "rep_name", "ExamplePluginRep", "data_type", "example_type")),
			"total_count", 1,
			"returned_count", 1,
			"remaining_count", 0,
			"has_more", false,
		).Merge(extra)
	}

	t.Run("copies plugins[] from the gkill response verbatim", func(t *testing.T) {
		ctx := makeCtx(resolving(func() *jsonobj.Object { return pageResponse(obj("plugins", plugins())) }))
		payload, err := HandleReadToolCall(ctx, "gkill_get_kyous", obj())
		expectNoError(t, err)
		expectEqual(t, payload.Value("plugins"), plugins())
	})

	t.Run("omits plugins when the page carries none (ordinary kyous only)", func(t *testing.T) {
		ctx := makeCtx(resolving(func() *jsonobj.Object { return pageResponse(obj()) }))
		payload, err := HandleReadToolCall(ctx, "gkill_get_kyous", obj())
		expectNoError(t, err)
		expectTrue(t, !payload.Defined("plugins"), "plugins set")
	})

	t.Run("omits plugins when gkill returns an empty array", func(t *testing.T) {
		ctx := makeCtx(resolving(func() *jsonobj.Object { return pageResponse(obj("plugins", arr())) }))
		payload, err := HandleReadToolCall(ctx, "gkill_get_kyous", obj())
		expectNoError(t, err)
		expectTrue(t, !payload.Defined("plugins"), "plugins set")
	})

	t.Run("count_only passes plugins through when the response carries them (copy, not a filter)", func(t *testing.T) {
		// plugins は「来たら載せる」条件付きコピーで、count_only かどうかで
		// 落としたりしない (経路は通常検索と同じ1つの組み立て)。
		ctx := makeCtx(resolving(func() *jsonobj.Object {
			return obj("kyous", arr(), "total_count", 3, "returned_count", 0, "remaining_count", 0, "has_more", false, "plugins", plugins())
		}))
		payload, err := HandleReadToolCall(ctx, "gkill_get_kyous", obj("count_only", true))
		expectNoError(t, err)
		expectEqual(t, payload.Value("plugins"), plugins())
	})
}

// ---------------------------------------------------------------------------
// 古いツールスキーマを掴んだクライアントへの警告
// ---------------------------------------------------------------------------
func TestHandleReadToolCallStaleToolSchemaWarning(t *testing.T) {
	page := func() *jsonobj.Object {
		return obj("kyous", arr(), "returned_count", 0, "remaining_count", 0, "has_more", false)
	}

	t.Run("warns when a non-string argument arrived as a JSON string", func(t *testing.T) {
		ctx := makeCtx(resolving(page))
		// 旧スキーマのクライアントは data_types を知らないので正規JSON文字列として送ってくる
		payload, err := HandleReadToolCall(ctx, "gkill_get_kyous", obj("data_types", `["nlog"]`))
		expectNoError(t, err)
		warnings := arrAt(t, payload, "warnings")
		expectEqual(t, len(warnings), 1)
		mustContain(t, jsString(warnings[0]), "tool schema snapshot looks stale")
		mustContain(t, jsString(warnings[0]), "data_types")
	})

	t.Run("warns when deprecated arguments are sent", func(t *testing.T) {
		ctx := makeCtx(resolving(page))
		payload, err := HandleReadToolCall(ctx, "gkill_get_kyous", obj("include_id", true, "query", obj("use_tags", true)))
		expectNoError(t, err)
		first := jsString(arrAt(t, payload, "warnings")[0])
		mustContain(t, first, "include_id")
		mustContain(t, first, "query.use_tags")
	})

	// 誤警告を出さないことが本体と同じくらい重要。現行スキーマどおりの呼び出しで
	// 警告が付くと、警告そのものが読まれなくなる
	t.Run("does not warn for a current-schema call", func(t *testing.T) {
		ctx := makeCtx(resolving(page))
		payload, err := HandleReadToolCall(ctx, "gkill_get_kyous", obj("data_types", strs("nlog"), "count_only", false))
		expectNoError(t, err)
		expectTrue(t, !payload.Defined("warnings"), "warnings set")
	})

	t.Run("keeps warnings from gkill and appends to them", func(t *testing.T) {
		ctx := makeCtx(resolving(func() *jsonobj.Object {
			return page().Set("warnings", strs(`unknown rep "GoogleLocation" in query.reps: plugin emits no kyou`))
		}))
		payload, err := HandleReadToolCall(ctx, "gkill_get_kyous", obj("count_only", "true"))
		expectNoError(t, err)
		warnings := arrAt(t, payload, "warnings")
		expectEqual(t, len(warnings), 2)
		mustContain(t, jsString(warnings[0]), "GoogleLocation")
		mustContain(t, jsString(warnings[1]), "tool schema snapshot looks stale")
	})

	t.Run("the one-line summary carries the stale marker too", func(t *testing.T) {
		summary := readSummary(t, "gkill_get_all_rep_names", obj(
			"rep_names", strs("Fitbit"),
			"total_count", 1,
			"returned_count", 1,
			"truncated", false,
			"warnings", strs("this MCP client's tool schema snapshot looks stale (…)"),
		))
		mustContain(t, summary, "reconnect the MCP client")
	})
}

// ---------------------------------------------------------------------------
// rep 名一覧（ディスパッチ）
// ---------------------------------------------------------------------------
func TestHandleReadToolCallGetAllRepNames(t *testing.T) {
	t.Run("filters node-side and does not forward contains/limit to gkill", func(t *testing.T) {
		ctx := makeCtx(resolving(func() *jsonobj.Object { return obj("rep_names", strs("Fitbit", "GoogleLocation", "Kmemo")) }))
		payload, err := HandleReadToolCall(ctx, "gkill_get_all_rep_names", obj("contains", "o", "limit", 1))
		expectNoError(t, err)
		expectEqual(t, payload, obj(
			"rep_names", strs("GoogleLocation"),
			"total_count", 2,
			"returned_count", 1,
			"truncated", true,
		))
		// "o" は GoogleLocation と Kmemo に一致する（Fitbit には無い）
		// gkill 側には絞り込みの口が無いので送らない（送ると未知キーで弾かれる）
		expectCalledWith(t, clientOf(ctx), "/api/get_all_rep_names", obj(), true, "sid-1")
	})
}

// ---------------------------------------------------------------------------
// 2026-09-19 の MCP 実利用報告への対応（ADR-0624 / 0626 / 0627 / 0629 / 0630）
// ---------------------------------------------------------------------------
func TestEnforceKyousSizeBudget(t *testing.T) {
	kyou := func(id int, text string) *jsonobj.Object {
		return obj(
			"id", id,
			"rep_name", "P",
			"data_type", "t",
			"related_time", "2026-09-1"+itoa(id)+"T00:00:00+09:00",
			"payload", obj("kind", "plugin", "content_status", "ok", "content_text", text),
		)
	}

	t.Run("holds back the entries that push the page over the budget and moves the cursor", func(t *testing.T) {
		payload := obj(
			"kyous", arr(kyou(1, strings.Repeat("a", 300)), kyou(2, strings.Repeat("b", 300)), kyou(3, strings.Repeat("c", 300))),
			"returned_count", 3,
			"remaining_count", 0,
			"has_more", false,
			"plugin_content", obj("requested", 3, "inlined", 3, "truncated", 0, "skipped", 0, "errors", 0, "total_text_length", 900),
		)
		oneEntry := jsonobj.ByteLength(arrAt(t, payload, "kyous")[0])
		EnforceKyousSizeBudget(payload, float64(oneEntry*2)/(1024*1024))

		ids := []any{}
		for _, entry := range arrAt(t, payload, "kyous") {
			ids = append(ids, objAt(t, entry).Value("id"))
		}
		expectEqual(t, ids, arr(1, 2))
		expectEqual(t, payload.Value("returned_count"), 2)
		expectEqual(t, payload.Value("remaining_count"), 1)
		expectEqual(t, payload.Value("has_more"), true)
		// Go の encodeMCPCursor と同じ {related_time}::{id}
		expectEqual(t, payload.Value("next_cursor"), "2026-09-12T00:00:00+09:00::2")
		mustMatch(t, jsString(arrAt(t, payload, "warnings")), `1 of 3 entries were held back`)
		expectEqual(t, objAt(t, payload, "plugin_content").Value("inlined"), 2)
		expectEqual(t, objAt(t, payload, "plugin_content").Value("total_text_length"), 600)
	})

	t.Run("returns a first entry that alone exceeds the budget, with a warning", func(t *testing.T) {
		payload := obj("kyous", arr(kyou(1, strings.Repeat("a", 3000)), kyou(2, "b")), "returned_count", 2, "remaining_count", 0, "has_more", false)
		EnforceKyousSizeBudget(payload, 100/(1024*1024))

		expectEqual(t, len(arrAt(t, payload, "kyous")), 1)
		expectEqual(t, payload.Value("has_more"), true)
		found := false
		for _, warning := range arrAt(t, payload, "warnings") {
			if strings.Contains(jsString(warning), "exceeds max_size_mb") {
				found = true
			}
		}
		expectTrue(t, found, "no exceeds max_size_mb warning")
	})

	t.Run("leaves a page within the budget untouched", func(t *testing.T) {
		payload := obj("kyous", arr(kyou(1, "a")), "returned_count", 1, "remaining_count", 4, "has_more", true, "next_cursor", "keep")
		EnforceKyousSizeBudget(payload, 1)
		expectEqual(t, payload, obj("kyous", arr(kyou(1, "a")), "returned_count", 1, "remaining_count", 4, "has_more", true, "next_cursor", "keep"))
	})
}

func TestHandleReadToolCallGetKyous20260919Additions(t *testing.T) {
	page := func(extra *jsonobj.Object) *jsonobj.Object {
		return obj("kyous", arr(), "returned_count", 0, "remaining_count", 0, "has_more", false).Merge(extra)
	}

	t.Run("forwards include_attached_ids and marks file-link minting only when asked", func(t *testing.T) {
		ctx := makeCtx(resolving(func() *jsonobj.Object { return page(obj()) }))
		plain, err := HandleReadToolCall(ctx, "gkill_get_kyous", obj())
		expectNoError(t, err)
		expectEqual(t, clientOf(ctx).calls[0].Body.Value("include_attached_ids"), false)
		_, marked := plain.Meta(MintFileLinksMark)
		expectTrue(t, !marked, "mint mark set without include_file_urls")

		asked, err := HandleReadToolCall(ctx, "gkill_get_kyous", obj("include_attached_ids", true, "include_file_urls", true))
		expectNoError(t, err)
		expectEqual(t, clientOf(ctx).calls[1].Body.Value("include_attached_ids"), true)
		mark, _ := asked.Meta(MintFileLinksMark)
		expectEqual(t, mark, true)
		// 印は meta なので JSON には出ない
		mustNotContain(t, jsonobj.MarshalString(asked), "mint_file_links")
	})

	t.Run("for_mi without a projection flag assumes include_create_mi and says so in warnings", func(t *testing.T) {
		ctx := makeCtx(resolving(func() *jsonobj.Object { return page(obj("warnings", strs("from gkill"))) }))
		payload, err := HandleReadToolCall(ctx, "gkill_get_kyous", obj("query", obj("for_mi", true)))
		expectNoError(t, err)
		expectEqual(t, objAt(t, clientOf(ctx).calls[0].Body, "query").Value("include_create_mi"), true)
		warnings := arrAt(t, payload, "warnings")
		mustMatch(t, jsString(warnings[0]), `include_create_mi:true was assumed`)
		expectEqual(t, warnings[1], "from gkill")
	})

	t.Run("for_mi with an explicit projection flag is left alone", func(t *testing.T) {
		ctx := makeCtx(resolving(func() *jsonobj.Object { return page(obj()) }))
		payload, err := HandleReadToolCall(ctx, "gkill_get_kyous", obj("query", obj("for_mi", true, "include_limit_mi", true)))
		expectNoError(t, err)
		expectTrue(t, !objAt(t, clientOf(ctx).calls[0].Body, "query").Defined("include_create_mi"), "include_create_mi assumed")
		expectTrue(t, !payload.Defined("warnings"), "warnings set")
	})
}

func TestHandleReadToolCallGetKyouHistoryOffsetAndDataType(t *testing.T) {
	histories := func() *jsonobj.Object {
		return jsonobj.MustUnmarshal(`{"mi_histories": [
			{"id": "m1", "data_type": "mi_check", "update_time": "2026-01-03T00:00:00+09:00"},
			{"id": "m1", "data_type": "mi_check", "update_time": "2026-01-02T00:00:00+09:00"},
			{"id": "m1", "data_type": "mi_create", "update_time": "2026-01-01T00:00:00+09:00"}
		]}`).(*jsonobj.Object)
	}

	t.Run("offset reads the older versions and next_offset points at the rest", func(t *testing.T) {
		ctx := makeCtx(resolving(histories))
		first, err := HandleReadToolCall(ctx, "gkill_get_kyou_history", obj("id", "m1", "data_type", "mi_start", "limit", 1))
		expectNoError(t, err)
		expectEqual(t, first.Value("offset"), 0)
		expectEqual(t, first.Value("has_more"), true)
		expectEqual(t, first.Value("next_offset"), 1)
		// 版の data_type は射影名ではなくこの口の語彙（エンティティ名）
		expectEqual(t, objAt(t, arrAt(t, first, "versions")[0]).Value("data_type"), "mi")

		last, err := HandleReadToolCall(ctx, "gkill_get_kyou_history", obj("id", "m1", "data_type", "mi", "limit", 1, "offset", 2))
		expectNoError(t, err)
		expectEqual(t, last.Value("returned_count"), 1)
		expectEqual(t, objAt(t, arrAt(t, last, "versions")[0]).Value("update_time"), "2026-01-01T00:00:00+09:00")
		expectEqual(t, last.Value("has_more"), false)
		expectTrue(t, !last.Defined("next_offset"), "next_offset set on the last page")

		beyond, err := HandleReadToolCall(ctx, "gkill_get_kyou_history", obj("id", "m1", "data_type", "mi", "offset", 5))
		expectNoError(t, err)
		expectEqual(t, beyond.Value("returned_count"), 0)
		expectEqual(t, beyond.Value("has_more"), false)
	})

	t.Run("summary tells where to continue", func(t *testing.T) {
		summary := readSummary(t, "gkill_get_kyou_history", obj(
			"version_count", 3, "returned_count", 1, "offset", 1, "has_more", true, "next_offset", 2, "latest_is_deleted", false,
		))
		mustContain(t, summary, "from offset 1")
		mustContain(t, summary, "offset:2")
	})
}

func TestHandleReadToolCallGetApplicationConfigCompactContainsMaxSize(t *testing.T) {
	// 実データと同じ形: 各ツリーはルート 1 オブジェクト（{name:"__root__", children:[...], is_dir:true}）で、
	// 葉の識別欄は rep_name / tag_name（ADR-0632）。
	config := func() *jsonobj.Object {
		return jsonobj.MustUnmarshal(`{
			"user_id": "u",
			"device": "d",
			"tag_struct": {"name": "__root__", "tag_name": "", "is_dir": true, "children": [
				{"name": "life", "tag_name": "life", "is_dir": true, "description": "folder note", "children": [
					{"name": "diary", "tag_name": "diary", "is_dir": false, "children": null, "check_when_inited": true, "is_force_hide": false, "description": "written at night"},
					{"name": "morning", "tag_name": "morning", "is_dir": false, "children": null, "check_when_inited": false, "is_force_hide": true, "description": ""}
				]}
			]},
			"rep_struct": {"name": "__root__", "rep_name": "", "is_dir": true, "children": [
				{
					"name": "dir",
					"rep_name": "dir",
					"is_dir": true,
					"children": [
						{"name": "Kmemo_A", "rep_name": "Kmemo_A", "is_dir": false, "children": null, "ignore_check_rep_rykv": false, "check_when_inited": true},
						{"name": "別名", "rep_name": "Kmemo_B", "is_dir": false, "children": null, "ignore_check_rep_rykv": true, "check_when_inited": false, "description": "phone memos"}
					]
				}
			]},
			"mi_board_struct": null
		}`).(*jsonobj.Object)
	}
	withConfig := func() *CallContext {
		return makeCtx(resolving(func() *jsonobj.Object { return obj("application_config", config()) }))
	}
	repChildren := func(t *testing.T, payload *jsonobj.Object) []any {
		t.Helper()
		return arrAt(t, objAt(t, payload, "rep_struct"), "children")
	}

	t.Run("compact drops default-valued node fields and keeps the visibility flags", func(t *testing.T) {
		payload, err := HandleReadToolCall(withConfig(), "gkill_get_application_config", obj("fields", strs("rep_struct")))
		expectNoError(t, err)
		dir := objAt(t, repChildren(t, payload)[0])
		children := arrAt(t, dir, "children")
		expectEqual(t, dir.Value("is_dir"), true)
		expectTrue(t, !dir.Has("name"), "folder name equal to rep_name should be dropped")
		expectEqual(t, children[0], obj("rep_name", "Kmemo_A", "check_when_inited", true))
		expectEqual(t, children[1], obj("name", "別名", "rep_name", "Kmemo_B", "ignore_check_rep_rykv", true, "check_when_inited", false, "description", "phone memos"))
	})

	t.Run("compact drops only an EMPTY description and keeps the tag identity field", func(t *testing.T) {
		payload, err := HandleReadToolCall(withConfig(), "gkill_get_application_config", obj("fields", strs("tag_struct")))
		expectNoError(t, err)
		life := objAt(t, arrAt(t, objAt(t, payload, "tag_struct"), "children")[0])
		expectEqual(t, life.Value("description"), "folder note")
		expectTrue(t, !life.Has("name"), "folder name equal to tag_name should be dropped")
		leaves := arrAt(t, life, "children")
		expectEqual(t, leaves[0], obj("tag_name", "diary", "check_when_inited", true, "is_force_hide", false, "description", "written at night"))
		expectEqual(t, leaves[1], obj("tag_name", "morning", "check_when_inited", false, "is_force_hide", true))
	})

	t.Run("compact:false returns the raw tree", func(t *testing.T) {
		payload, err := HandleReadToolCall(withConfig(), "gkill_get_application_config", obj("fields", strs("rep_struct"), "compact", false))
		expectNoError(t, err)
		a := objAt(t, arrAt(t, objAt(t, repChildren(t, payload)[0]), "children")[0])
		expectTrue(t, a.Has("children") && a.Value("children") == nil, "children is not null")
		expectEqual(t, a.Value("is_dir"), false)
		expectEqual(t, a.Value("name"), "Kmemo_A")
	})

	t.Run("contains keeps matching leaves under a root object and drops folders left empty", func(t *testing.T) {
		hit, err := HandleReadToolCall(withConfig(), "gkill_get_application_config", obj("fields", strs("rep_struct"), "contains", "kmemo_b"))
		expectNoError(t, err)
		expectEqual(t, len(repChildren(t, hit)), 1)
		expectEqual(t, repNamesOf(t, objAt(t, repChildren(t, hit)[0]), "children"), []string{"Kmemo_B"})

		miss, err := HandleReadToolCall(withConfig(), "gkill_get_application_config", obj("fields", strs("rep_struct"), "contains", "zzz"))
		expectNoError(t, err)
		// 残らなければルートは children:[] で返す（compact が空配列を落とすので children 自体が消える）
		root := objAt(t, miss, "rep_struct")
		expectTrue(t, !root.Has("children"), "children should be dropped when nothing matched")
		expectEqual(t, root.Value("is_dir"), true)
	})

	t.Run("contains matches the identity field of tag leaves, not only name", func(t *testing.T) {
		hit, err := HandleReadToolCall(withConfig(), "gkill_get_application_config", obj("fields", strs("tag_struct"), "contains", "DIARY", "compact", false))
		expectNoError(t, err)
		life := objAt(t, arrAt(t, objAt(t, hit, "tag_struct"), "children")[0])
		leaves := arrAt(t, life, "children")
		expectEqual(t, len(leaves), 1)
		expectEqual(t, objAt(t, leaves[0]).Value("tag_name"), "diary")
	})

	t.Run("fields descriptions lists only nodes that have a note, with struct / name / path", func(t *testing.T) {
		payload, err := HandleReadToolCall(withConfig(), "gkill_get_application_config", obj("fields", strs("descriptions")))
		expectNoError(t, err)
		expectTrue(t, !payload.Has("tag_struct") && !payload.Has("rep_struct"), "trees must not be returned")
		expectEqual(t, payload.Value("descriptions"), arr(
			obj("struct", "tag_struct", "name", "life", "path", "life", "is_dir", true, "description", "folder note"),
			obj("struct", "tag_struct", "name", "diary", "path", "life/diary", "description", "written at night"),
			obj("struct", "rep_struct", "name", "Kmemo_B", "path", "dir/Kmemo_B", "description", "phone memos"),
		))
	})

	t.Run("contains filters the descriptions list by name / path / description", func(t *testing.T) {
		payload, err := HandleReadToolCall(withConfig(), "gkill_get_application_config", obj("fields", strs("descriptions"), "contains", "phone"))
		expectNoError(t, err)
		expectEqual(t, payload.Value("descriptions"), arr(
			obj("struct", "rep_struct", "name", "Kmemo_B", "path", "dir/Kmemo_B", "description", "phone memos"),
		))
	})

	t.Run("descriptions is not part of the default full response", func(t *testing.T) {
		payload, err := HandleReadToolCall(withConfig(), "gkill_get_application_config", obj())
		expectNoError(t, err)
		expectTrue(t, !payload.Has("descriptions"), "descriptions must be opt-in")
	})

	t.Run("descriptions is [] when no tree is configured", func(t *testing.T) {
		ctx := makeCtx(resolving(func() *jsonobj.Object { return obj("application_config", obj("user_id", "u", "device", "d")) }))
		payload, err := HandleReadToolCall(ctx, "gkill_get_application_config", obj("fields", strs("descriptions")))
		expectNoError(t, err)
		expectEqual(t, payload.Value("descriptions"), arr())
	})

	t.Run("max_size_mb replaces the largest struct with omitted_bytes and warns", func(t *testing.T) {
		payload, err := HandleReadToolCall(withConfig(), "gkill_get_application_config", obj(
			"fields", strs("rep_struct", "tag_struct"),
			"max_size_mb", 100.0/(1024*1024),
		))
		expectNoError(t, err)
		expectTrue(t, floatOf(t, objAt(t, payload, "rep_struct").Value("omitted_bytes")) > 0, "omitted_bytes not positive")
		expectTrue(t, floatOf(t, objAt(t, payload, "tag_struct").Value("omitted_bytes")) > 0, "tag_struct omitted_bytes not positive")
		mustContain(t, jsString(arrAt(t, payload, "warnings")), "rep_struct")
	})
}
