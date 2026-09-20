package mcp

// 書き込みサーバ（NewWriteServer）の検査。
//
// サーバはモックの GkillAPI で検査するので、実 HTTP 呼び出しは発生しない。

import (
	"context"
	"regexp"
	"strings"
	"testing"

	"github.com/mt3hr/gkill/src/server/gkill/mcp/jsonobj"
)

func createWriteMockClient() *mockClient {
	return &mockClient{
		callApi: func(_ string, _ *jsonobj.Object, _ bool, _ string) (*jsonobj.Object, error) {
			return obj("errors", arr(), "messages", arr()), nil
		},
		login:         func() (string, error) { return "mock-session-id", nil },
		defaultLocale: "ja",
		userID:        "testuser",
	}
}

// ---------------------------------------------------------------------------
// Constructor
// ---------------------------------------------------------------------------
func TestMcpWriteServerConstructor(t *testing.T) {
	t.Run("accepts a client", func(t *testing.T) {
		mock := createWriteMockClient()
		server := NewWriteServer(mock, nil)
		expectTrue(t, server.Client == GkillAPI(mock), "client is not the one passed in")
	})
}

// ---------------------------------------------------------------------------
// handleToolCall dispatch — Write tools
// ---------------------------------------------------------------------------
func TestWriteHandleToolCallWriteTools(t *testing.T) {
	setup := func() (*mockClient, *Server) {
		client := createWriteMockClient()
		return client, NewWriteServer(client, nil)
	}

	t.Run("dispatches gkill_add_kmemo to /api/add_kmemo", func(t *testing.T) {
		client, server := setup()
		client.mockResolvedValue(obj("added_kmemo", obj("id", "k1", "content", "hello"), "added_kyou", obj("id", "k1", "data_type", "kmemo"), "errors", arr()))

		result, err := serverToolCall(t, server, "gkill_add_kmemo", obj("content", "hello"))
		expectNoError(t, err)

		expectEqual(t, len(client.calls), 1)
		call := client.calls[0]
		expectEqual(t, call.Pathname, "/api/add_kmemo")
		expectEqual(t, objAt(t, call.Body, "kmemo").Value("content"), "hello")
		expectEqual(t, call.Body.Value("want_response_kyou"), true)
		expectEqual(t, objAt(t, result, "added_kmemo").Value("id"), "k1")
		expectEqual(t, objAt(t, result, "added_kyou").Value("data_type"), "kmemo")
	})

	t.Run("dispatches gkill_add_urlog to /api/add_urlog", func(t *testing.T) {
		client, server := setup()
		client.mockResolvedValue(obj("added_urlog", obj("id", "u1", "url", "https://example.com"), "added_kyou", obj("id", "u1"), "errors", arr()))

		result, err := serverToolCall(t, server, "gkill_add_urlog", obj("url", "https://example.com"))
		expectNoError(t, err)

		call := client.calls[0]
		expectEqual(t, call.Pathname, "/api/add_urlog")
		expectEqual(t, objAt(t, call.Body, "urlog").Value("url"), "https://example.com")
		expectEqual(t, objAt(t, result, "added_urlog").Value("id"), "u1")
	})

	t.Run("dispatches gkill_add_nlog to /api/add_nlog", func(t *testing.T) {
		client, server := setup()
		client.mockResolvedValue(obj("added_nlog", obj("id", "n1", "amount", 1500), "added_kyou", obj("id", "n1"), "errors", arr()))

		result, err := serverToolCall(t, server, "gkill_add_nlog", obj("title", "lunch", "amount", 1500))
		expectNoError(t, err)

		call := client.calls[0]
		expectEqual(t, call.Pathname, "/api/add_nlog")
		expectEqual(t, objAt(t, call.Body, "nlog").Value("amount"), 1500)
		expectEqual(t, objAt(t, call.Body, "nlog").Value("title"), "lunch")
		expectEqual(t, objAt(t, result, "added_nlog").Value("amount"), 1500)
	})

	t.Run("dispatches gkill_add_lantana to /api/add_lantana", func(t *testing.T) {
		client, server := setup()
		client.mockResolvedValue(obj("added_lantana", obj("id", "l1", "mood", 7), "added_kyou", obj("id", "l1"), "errors", arr()))

		result, err := serverToolCall(t, server, "gkill_add_lantana", obj("mood", 7))
		expectNoError(t, err)

		call := client.calls[0]
		expectEqual(t, call.Pathname, "/api/add_lantana")
		expectEqual(t, objAt(t, call.Body, "lantana").Value("mood"), 7)
		expectEqual(t, objAt(t, result, "added_lantana").Value("mood"), 7)
	})

	t.Run("dispatches gkill_add_timeis to /api/add_timeis", func(t *testing.T) {
		client, server := setup()
		client.mockResolvedValue(obj("added_timeis", obj("id", "t1", "title", "coding"), "added_kyou", obj("id", "t1"), "errors", arr()))

		result, err := serverToolCall(t, server, "gkill_add_timeis", obj("title", "coding"))
		expectNoError(t, err)

		call := client.calls[0]
		expectEqual(t, call.Pathname, "/api/add_timeis")
		expectEqual(t, objAt(t, call.Body, "timeis").Value("title"), "coding")
		expectEqual(t, objAt(t, result, "added_timeis").Value("title"), "coding")
	})

	t.Run("dispatches gkill_add_mi to /api/add_mi", func(t *testing.T) {
		client, server := setup()
		client.mockResolvedValue(obj("added_mi", obj("id", "m1", "title", "fix bug", "board_name", "dev"), "added_kyou", obj("id", "m1"), "errors", arr()))

		result, err := serverToolCall(t, server, "gkill_add_mi", obj("title", "fix bug", "board_name", "dev"))
		expectNoError(t, err)

		call := client.calls[0]
		expectEqual(t, call.Pathname, "/api/add_mi")
		mi := objAt(t, call.Body, "mi")
		expectEqual(t, mi.Value("title"), "fix bug")
		expectEqual(t, mi.Value("board_name"), "dev")
		expectEqual(t, mi.Value("is_checked"), false)
		expectEqual(t, objAt(t, result, "added_mi").Value("title"), "fix bug")
	})

	t.Run("dispatches gkill_add_kc to /api/add_kc", func(t *testing.T) {
		client, server := setup()
		client.mockResolvedValue(obj("added_kc", obj("id", "c1", "num_value", 42), "added_kyou", obj("id", "c1"), "errors", arr()))

		result, err := serverToolCall(t, server, "gkill_add_kc", obj("title", "steps", "num_value", 42))
		expectNoError(t, err)

		call := client.calls[0]
		expectEqual(t, call.Pathname, "/api/add_kc")
		expectEqual(t, objAt(t, call.Body, "kc").Value("num_value"), 42)
		expectEqual(t, objAt(t, result, "added_kc").Value("num_value"), 42)
	})

	t.Run("dispatches gkill_add_tag to /api/add_tag", func(t *testing.T) {
		client, server := setup()
		client.mockResolvedValue(obj("added_tag", obj("id", "tg1", "tag", "important", "target_id", "k1"), "added_kyou", obj("id", "tg1"), "errors", arr()))

		result, err := serverToolCall(t, server, "gkill_add_tag", obj("tag", "important", "target_id", "k1"))
		expectNoError(t, err)

		call := client.calls[0]
		expectEqual(t, call.Pathname, "/api/add_tag")
		expectEqual(t, objAt(t, call.Body, "tag").Value("tag"), "important")
		expectEqual(t, objAt(t, call.Body, "tag").Value("target_id"), "k1")
		expectEqual(t, objAt(t, result, "added_tag").Value("tag"), "important")
	})

	t.Run("dispatches gkill_add_text to /api/add_text", func(t *testing.T) {
		client, server := setup()
		client.mockResolvedValue(obj("added_text", obj("id", "tx1", "text", "note", "target_id", "k1"), "added_kyou", obj("id", "tx1"), "errors", arr()))

		result, err := serverToolCall(t, server, "gkill_add_text", obj("text", "note", "target_id", "k1"))
		expectNoError(t, err)

		call := client.calls[0]
		expectEqual(t, call.Pathname, "/api/add_text")
		expectEqual(t, objAt(t, call.Body, "text").Value("text"), "note")
		expectEqual(t, objAt(t, call.Body, "text").Value("target_id"), "k1")
		expectEqual(t, objAt(t, result, "added_text").Value("text"), "note")
	})

	t.Run("dispatches gkill_submit_kftl to /api/submit_kftl_text", func(t *testing.T) {
		client, server := setup()
		client.mockResolvedValue(obj("messages", arr(obj("message", "created 2 records")), "errors", arr()))

		result, err := serverToolCall(t, server, "gkill_submit_kftl", obj("kftl_text", "/mi Buy milk"))
		expectNoError(t, err)

		call := client.calls[0]
		expectEqual(t, call.Pathname, "/api/submit_kftl_text")
		expectEqual(t, call.Body.Value("kftl_text"), "/mi Buy milk")
		expectEqual(t, len(arrAt(t, result, "messages")), 1)
	})

	t.Run("dispatches gkill_delete_kyou to correct update endpoint", func(t *testing.T) {
		client, server := setup()
		// First call: GET to fetch current entity
		client.mockResolvedValueOnce(obj("kmemo_histories", arr(obj("id", "k1", "content", "hello", "is_deleted", false, "rep_name", "rep1")), "errors", arr()))
		// Second call: UPDATE with is_deleted=true
		client.mockResolvedValueOnce(obj("updated_kmemo", obj("id", "k1", "is_deleted", true), "updated_kyou", obj("id", "k1"), "errors", arr()))

		result, err := serverToolCall(t, server, "gkill_delete_kyou", obj("id", "k1", "data_type", "kmemo"))
		expectNoError(t, err)

		expectEqual(t, client.calls[0].Pathname, "/api/get_kmemo")
		expectEqual(t, client.calls[1].Pathname, "/api/update_kmemo")
		kmemo := objAt(t, client.calls[1].Body, "kmemo")
		expectEqual(t, kmemo.Value("id"), "k1")
		expectEqual(t, kmemo.Value("is_deleted"), true)
		expectEqual(t, kmemo.Value("content"), "hello")
		expectEqual(t, objAt(t, result, "updated_kmemo").Value("is_deleted"), true)
	})

	t.Run("gkill_delete_kyou works for mi data_type", func(t *testing.T) {
		client, server := setup()
		client.mockResolvedValueOnce(obj("mi_histories", arr(obj("id", "m1", "title", "task1", "is_deleted", false, "rep_name", "rep1")), "errors", arr()))
		client.mockResolvedValueOnce(obj("updated_mi", obj("id", "m1", "is_deleted", true), "updated_kyou", obj("id", "m1"), "errors", arr()))

		result, err := serverToolCall(t, server, "gkill_delete_kyou", obj("id", "m1", "data_type", "mi"))
		expectNoError(t, err)

		expectEqual(t, client.calls[0].Pathname, "/api/get_mi")
		expectEqual(t, client.calls[1].Pathname, "/api/update_mi")
		expectEqual(t, objAt(t, result, "updated_mi").Value("is_deleted"), true)
	})
}

// ---------------------------------------------------------------------------
// handleToolCall dispatch — Read convenience tools
// ---------------------------------------------------------------------------
func TestWriteHandleToolCallReadConvenienceTools(t *testing.T) {
	setup := func() (*mockClient, *Server) {
		client := createWriteMockClient()
		return client, NewWriteServer(client, nil)
	}

	t.Run("dispatches gkill_get_all_rep_names to /api/get_all_rep_names", func(t *testing.T) {
		client, server := setup()
		client.mockResolvedValue(obj("rep_names", strs("rep1", "rep2"), "errors", arr()))
		result, err := serverToolCall(t, server, "gkill_get_all_rep_names", obj())
		expectNoError(t, err)
		expectEqual(t, client.calls[0].Pathname, "/api/get_all_rep_names")
		expectEqual(t, result.Value("rep_names"), strs("rep1", "rep2"))
	})

	t.Run("dispatches gkill_get_mi_board_list to /api/get_mi_board_list", func(t *testing.T) {
		client, server := setup()
		client.mockResolvedValue(obj("boards", strs("board1", "board2"), "errors", arr()))
		result, err := serverToolCall(t, server, "gkill_get_mi_board_list", obj())
		expectNoError(t, err)
		expectEqual(t, client.calls[0].Pathname, "/api/get_mi_board_list")
		expectEqual(t, result.Value("boards"), strs("board1", "board2"))
	})

	t.Run("dispatches gkill_get_all_tag_names to /api/get_all_tag_names", func(t *testing.T) {
		client, server := setup()
		client.mockResolvedValue(obj("tag_names", strs("tag1", "tag2"), "errors", arr()))
		result, err := serverToolCall(t, server, "gkill_get_all_tag_names", obj())
		expectNoError(t, err)
		expectEqual(t, client.calls[0].Pathname, "/api/get_all_tag_names")
		expectEqual(t, result.Value("tag_names"), strs("tag1", "tag2"))
	})
}

// ---------------------------------------------------------------------------
// handleToolCall — plugin tools
// ---------------------------------------------------------------------------
func TestWriteHandleToolCallPluginTools(t *testing.T) {
	t.Run("dispatches gkill_get_plugin_list to /api/get_plugin_list", func(t *testing.T) {
		client := createWriteMockClient()
		client.mockResolvedValue(obj("plugins", arr(obj("name", "gkill_plugin_claudecode", "rep_name", "Claude Code", "is_alive", true)), "errors", arr()))
		server := NewWriteServer(client, nil)

		result, err := serverToolCall(t, server, "gkill_get_plugin_list", obj())
		expectNoError(t, err)

		expectCalledWith(t, client, "/api/get_plugin_list", obj(), true, "")
		expectEqual(t, len(arrAt(t, result, "plugins")), 1)
	})

	// Write サーバには gkill_get_kyous が無いため、プラグイン本文を読む導線も無い。
	// 本文が要るときは ReadWrite サーバを使う。
	t.Run("does not expose the removed single-kyou content tool", func(t *testing.T) {
		client := createWriteMockClient()
		server := NewWriteServer(client, nil)

		_, err := serverToolCall(t, server, "gkill_get_plugin_content", obj("rep_name", "r", "kyou_id", "k"))
		expectErrorMatches(t, err, `Unknown tool`)
		expectEqual(t, len(client.calls), 0)
	})
}

// ---------------------------------------------------------------------------
// handleToolCall — unknown tool
// ---------------------------------------------------------------------------
func TestWriteHandleToolCallErrorCases(t *testing.T) {
	t.Run("throws for unknown tool", func(t *testing.T) {
		server := NewWriteServer(createWriteMockClient(), nil)
		_, err := serverToolCall(t, server, "unknown_tool", obj())
		expectErrorContains(t, err, "Unknown tool")
	})
}

// ---------------------------------------------------------------------------
// handleToolCall — entity defaults
// ---------------------------------------------------------------------------
func TestWriteHandleToolCallEntityDefaults(t *testing.T) {
	t.Run("sets common fields on kmemo entity", func(t *testing.T) {
		client := createWriteMockClient()
		client.mockResolvedValue(obj("added_kmemo", obj("id", "k1"), "added_kyou", obj("id", "k1"), "errors", arr()))
		server := NewWriteServer(client, nil)

		_, err := serverToolCall(t, server, "gkill_add_kmemo", obj("content", "test"))
		expectNoError(t, err)

		kmemo := objAt(t, client.calls[0].Body, "kmemo")
		expectTrue(t, jsTruthy(kmemo.Value("id")), "id missing")
		expectEqual(t, kmemo.Value("rep_name"), "")
		expectEqual(t, kmemo.Value("data_type"), "kmemo")
		expectEqual(t, kmemo.Value("create_app"), "gkill_mcp_write")
		expectEqual(t, kmemo.Value("create_device"), "mcp")
		expectEqual(t, kmemo.Value("create_user"), "testuser")
		expectEqual(t, kmemo.Value("is_deleted"), false)
		expectTrue(t, jsTruthy(kmemo.Value("create_time")), "create_time missing")
		expectTrue(t, jsTruthy(kmemo.Value("update_time")), "update_time missing")
	})
}

// ---------------------------------------------------------------------------
// JSON-RPC protocol
// ---------------------------------------------------------------------------
func TestWriteJSONRPCProtocol(t *testing.T) {
	setup := func() *Server { return NewWriteServer(createWriteMockClient(), nil) }

	t.Run("initialize returns server info", func(t *testing.T) {
		response := serverMessage(t, setup(), obj("jsonrpc", "2.0", "id", 1, "method", "initialize", "params", obj()))
		expectEqual(t, objAt(t, response, "result", "serverInfo").Value("name"), "gkill-write-mcp")
		expectEqual(t, objAt(t, response, "result").Value("protocolVersion"), "2024-11-05")
	})

	t.Run("ping returns empty result", func(t *testing.T) {
		response := serverMessage(t, setup(), obj("jsonrpc", "2.0", "id", 2, "method", "ping"))
		expectEqual(t, response.Value("result"), obj())
	})

	t.Run("tools/list returns 29 tools", func(t *testing.T) {
		response := serverMessage(t, setup(), obj("jsonrpc", "2.0", "id", 3, "method", "tools/list"))
		expectEqual(t, len(arrAt(t, response, "result", "tools")), 29)
	})

	t.Run("tools/list includes all expected tool names", func(t *testing.T) {
		response := serverMessage(t, setup(), obj("jsonrpc", "2.0", "id", 4, "method", "tools/list"))
		names := NewStringSet()
		for _, tool := range arrAt(t, response, "result", "tools") {
			names.Add(strAt(t, objAt(t, tool), "name"))
		}
		for _, name := range []string{
			"gkill_add_kmemo", "gkill_add_urlog", "gkill_add_nlog", "gkill_add_lantana", "gkill_add_timeis", "gkill_add_mi",
			"gkill_add_kc", "gkill_add_tag", "gkill_add_text", "gkill_submit_kftl", "gkill_delete_kyou",
			// Update tools
			"gkill_update_kmemo", "gkill_update_urlog", "gkill_update_nlog", "gkill_update_lantana", "gkill_update_timeis",
			"gkill_update_mi", "gkill_update_kc", "gkill_update_tag", "gkill_update_text",
			// Read convenience tools
			"gkill_status", "gkill_get_all_rep_names", "gkill_get_mi_board_list", "gkill_get_all_tag_names",
			// Plugin tools
			"gkill_get_plugin_list",
		} {
			expectTrue(t, names.Has(name), "%s missing", name)
		}
		expectTrue(t, !names.Has("gkill_get_plugin_content"), "gkill_get_plugin_content present")
	})

	t.Run("unknown method returns error", func(t *testing.T) {
		response := serverMessage(t, setup(), obj("jsonrpc", "2.0", "id", 5, "method", "unknown/method"))
		expectEqual(t, objAt(t, response, "error").Value("code"), -32601)
	})

	t.Run("invalid request returns error", func(t *testing.T) {
		response := serverMessage(t, setup(), obj("id", 6))
		expectEqual(t, objAt(t, response, "error").Value("code"), -32600)
	})

	t.Run("notifications/initialized returns null", func(t *testing.T) {
		response := serverMessage(t, setup(), obj("jsonrpc", "2.0", "method", "notifications/initialized"))
		expectTrue(t, response == nil, "notification got a response")
	})
}

// ---------------------------------------------------------------------------
// buildToolResult
// ---------------------------------------------------------------------------
func TestWriteBuildToolResult(t *testing.T) {
	t.Run("includes structuredContent for success", func(t *testing.T) {
		server := NewWriteServer(createWriteMockClient(), nil)
		result := server.BuildToolResult("gkill_add_kmemo", obj("added_kmemo", obj("id", "k1")), false, nil)
		expectEqual(t, result.Value("isError"), false)
		expectTrue(t, result.Defined("structuredContent"), "structuredContent missing")
		mustContain(t, firstText(t, result), "Created kmemo: k1")
	})

	t.Run("marks errors correctly", func(t *testing.T) {
		server := NewWriteServer(createWriteMockClient(), nil)
		result := server.BuildToolResult("gkill_add_kmemo", obj("error", "test error"), true, nil)
		expectEqual(t, result.Value("isError"), true)
		mustContain(t, firstText(t, result), "gkill_add_kmemo failed")
	})
}

// ---------------------------------------------------------------------------
// handlePayload — batch
// ---------------------------------------------------------------------------
func TestWriteHandlePayload(t *testing.T) {
	t.Run("handles batch of messages", func(t *testing.T) {
		server := NewWriteServer(createWriteMockClient(), nil)
		result := server.HandlePayload(context.Background(), arr(
			obj("jsonrpc", "2.0", "id", 1, "method", "ping"),
			obj("jsonrpc", "2.0", "id", 2, "method", "ping"),
		), nil)
		responses, ok := jsonobj.AsArray(result)
		expectTrue(t, ok, "batch did not return an array")
		expectEqual(t, len(responses), 2)
	})

	t.Run("handles empty batch", func(t *testing.T) {
		server := NewWriteServer(createWriteMockClient(), nil)
		result := server.HandlePayload(context.Background(), arr(), nil)
		expectEqual(t, objAt(t, result, "error").Value("code"), -32600)
	})

	t.Run("handles single message", func(t *testing.T) {
		server := NewWriteServer(createWriteMockClient(), nil)
		result := server.HandlePayload(context.Background(), obj("jsonrpc", "2.0", "id", 1, "method", "ping"), nil)
		expectEqual(t, objAt(t, result).Value("result"), obj())
	})
}

// ---------------------------------------------------------------------------
// warnings の1行要約昇格は書き込み側にも掛かる (server_base の1箇所配線)
// ---------------------------------------------------------------------------
func TestSummaryElevationAppliesToWriteToolsToo(t *testing.T) {
	// 昇格の実装は server_base の summarizeToolPayload 1箇所だが、
	// 検証は read サーバ側 (server_test.go) にしか無かった。書き込みの要約は
	// SummarizeWriteToolPayload が先に stale 専用文言を付け、その後
	// AppendWarningsToSummary が同じ warning をフィルタする二段構えなので、
	// 崩れると「同じ指摘が1行に2回」が書き込み側だけで再発する。
	summaryLine := func(t *testing.T, result *jsonobj.Object) string {
		t.Helper()
		return strings.Split(firstText(t, result), "\n")[0]
	}

	t.Run("a real warning is elevated into the write summary line", func(t *testing.T) {
		server := NewWriteServer(createWriteMockClient(), nil)
		result := server.BuildToolResult(
			"gkill_add_urlog",
			obj("added_urlog", obj("id", "u1"), "warnings", strs("favicon fetch was skipped by request")),
			false, nil,
		)
		line := summaryLine(t, result)
		mustContain(t, line, "urlog: u1")
		mustContain(t, line, "WARNING: favicon fetch was skipped by request")
	})

	t.Run("a stale-schema warning keeps the dedicated note and never doubles up", func(t *testing.T) {
		server := NewWriteServer(createWriteMockClient(), nil)
		result := server.BuildToolResult(
			"gkill_add_urlog",
			obj(
				"added_urlog", obj("id", "u1"),
				"warnings", strs("this client's tool schema snapshot looks stale: fetch_metadata arrived as a JSON string"),
			),
			false, nil,
		)
		line := summaryLine(t, result)
		// 専用文言は1回だけ。汎用の WARNING: には載らない (フィルタ済み)。
		expectEqual(t, len(regexp.MustCompile(`stale`).FindAllString(line, -1)), 1)
		mustContain(t, line, "tool schema looks stale")
		mustNotContain(t, line, "WARNING:")
	})
}
