package mcp

// 読み書きサーバ（NewReadWriteServer）の検査。
//
// サーバはモックの GkillAPI で検査するので、実 HTTP 呼び出しは発生しない。

import (
	"context"
	"testing"

	"github.com/mt3hr/gkill/src/server/gkill/mcp/jsonobj"
)

func createReadWriteMockClient() *mockClient {
	client := createMockClient()
	client.userID = "testuser"
	return client
}

// ---------------------------------------------------------------------------
// Constructor
// ---------------------------------------------------------------------------
func TestReadWriteMcpServerConstructor(t *testing.T) {
	t.Run("accepts a client", func(t *testing.T) {
		mock := createReadWriteMockClient()
		server := NewReadWriteServer(mock, nil)
		expectTrue(t, server.Client == GkillAPI(mock), "client is not the one passed in")
	})
}

// ---------------------------------------------------------------------------
// handleToolCall dispatch — Read tools
// ---------------------------------------------------------------------------
func TestReadWriteHandleToolCallReadTools(t *testing.T) {
	setup := func() (*mockClient, *Server) {
		client := createReadWriteMockClient()
		return client, NewReadWriteServer(client, nil)
	}

	t.Run("dispatches gkill_get_kyous to /api/get_kyous_mcp", func(t *testing.T) {
		client, server := setup()
		client.mockResolvedValue(obj("kyous", arr(obj("id", "1")), "total_count", 1, "returned_count", 1, "has_more", false, "errors", arr()))

		result, err := serverToolCall(t, server, "gkill_get_kyous", obj("query", obj()))
		expectNoError(t, err)
		expectEqual(t, client.calls[0].Pathname, "/api/get_kyous_mcp")
		expectEqual(t, len(arrAt(t, result, "kyous")), 1)
	})

	t.Run("dispatches gkill_get_mi_board_list", func(t *testing.T) {
		client, server := setup()
		client.mockResolvedValue(obj("boards", strs("b1", "b2"), "errors", arr()))
		result, err := serverToolCall(t, server, "gkill_get_mi_board_list", obj())
		expectNoError(t, err)
		expectEqual(t, client.calls[0].Pathname, "/api/get_mi_board_list")
		expectEqual(t, result.Value("boards"), strs("b1", "b2"))
	})

	t.Run("dispatches gkill_get_all_tag_names", func(t *testing.T) {
		client, server := setup()
		client.mockResolvedValue(obj("tag_names", strs("t1"), "errors", arr()))
		result, err := serverToolCall(t, server, "gkill_get_all_tag_names", obj())
		expectNoError(t, err)
		expectEqual(t, client.calls[0].Pathname, "/api/get_all_tag_names")
		expectEqual(t, result.Value("tag_names"), strs("t1"))
	})

	t.Run("dispatches gkill_get_all_rep_names", func(t *testing.T) {
		client, server := setup()
		client.mockResolvedValue(obj("rep_names", strs("r1"), "errors", arr()))
		result, err := serverToolCall(t, server, "gkill_get_all_rep_names", obj())
		expectNoError(t, err)
		expectEqual(t, client.calls[0].Pathname, "/api/get_all_rep_names")
		expectEqual(t, result.Value("rep_names"), strs("r1"))
	})

	t.Run("dispatches gkill_get_gps_log", func(t *testing.T) {
		client, server := setup()
		client.mockResolvedValue(obj("gps_logs", arr(obj("lat", 35.6)), "errors", arr()))
		result, err := serverToolCall(t, server, "gkill_get_gps_log", obj("start_date", "2026-03-01", "end_date", "2026-03-07"))
		expectNoError(t, err)
		expectEqual(t, client.calls[0].Pathname, "/api/get_gps_log")
		expectEqual(t, len(arrAt(t, result, "gps_logs")), 1)
	})

	t.Run("dispatches gkill_get_application_config", func(t *testing.T) {
		client, server := setup()
		client.mockResolvedValue(obj("application_config", obj("tag_struct", obj()), "errors", arr()))
		result, err := serverToolCall(t, server, "gkill_get_application_config", obj())
		expectNoError(t, err)
		expectEqual(t, client.calls[0].Pathname, "/api/get_application_config")
		expectTrue(t, result.Has("tag_struct"), "tag_struct missing")
	})

	t.Run("dispatches gkill_get_idf_file", func(t *testing.T) {
		client, server := setup()
		client.mockFetchFile(&FileResponse{Buffer: []byte("png-data"), ContentType: "image/png"})
		result, err := serverToolCall(t, server, "gkill_get_idf_file", obj("rep_name", "repo", "file_name", "photo.png"))
		expectNoError(t, err)
		expectEqual(t, result.Value("file_name"), "photo.png")
		expectEqual(t, result.Value("mime_type"), "image/png")
		expectEqual(t, result.Value("is_image"), true)
	})
}

// ---------------------------------------------------------------------------
// handleToolCall dispatch — Write tools
// ---------------------------------------------------------------------------
func TestReadWriteHandleToolCallWriteTools(t *testing.T) {
	setup := func() (*mockClient, *Server) {
		client := createReadWriteMockClient()
		return client, NewReadWriteServer(client, nil)
	}

	t.Run("dispatches gkill_add_kmemo", func(t *testing.T) {
		client, server := setup()
		client.mockResolvedValue(obj("added_kmemo", obj("id", "k1", "content", "hello"), "added_kyou", obj("id", "k1"), "errors", arr()))
		result, err := serverToolCall(t, server, "gkill_add_kmemo", obj("content", "hello"))
		expectNoError(t, err)
		expectEqual(t, client.calls[0].Pathname, "/api/add_kmemo")
		expectEqual(t, objAt(t, result, "added_kmemo").Value("id"), "k1")
	})

	t.Run("dispatches gkill_add_urlog", func(t *testing.T) {
		client, server := setup()
		client.mockResolvedValue(obj("added_urlog", obj("id", "u1"), "added_kyou", obj("id", "u1"), "errors", arr()))
		result, err := serverToolCall(t, server, "gkill_add_urlog", obj("url", "https://example.com"))
		expectNoError(t, err)
		expectEqual(t, client.calls[0].Pathname, "/api/add_urlog")
		expectEqual(t, objAt(t, result, "added_urlog").Value("id"), "u1")
	})

	t.Run("dispatches gkill_add_nlog", func(t *testing.T) {
		client, server := setup()
		client.mockResolvedValue(obj("added_nlog", obj("id", "n1", "amount", 1500), "added_kyou", obj("id", "n1"), "errors", arr()))
		result, err := serverToolCall(t, server, "gkill_add_nlog", obj("title", "lunch", "amount", 1500))
		expectNoError(t, err)
		expectEqual(t, client.calls[0].Pathname, "/api/add_nlog")
		expectEqual(t, objAt(t, result, "added_nlog").Value("amount"), 1500)
	})

	t.Run("dispatches gkill_add_lantana", func(t *testing.T) {
		client, server := setup()
		client.mockResolvedValue(obj("added_lantana", obj("id", "l1", "mood", 7), "added_kyou", obj("id", "l1"), "errors", arr()))
		result, err := serverToolCall(t, server, "gkill_add_lantana", obj("mood", 7))
		expectNoError(t, err)
		expectEqual(t, client.calls[0].Pathname, "/api/add_lantana")
		expectEqual(t, objAt(t, result, "added_lantana").Value("mood"), 7)
	})

	t.Run("dispatches gkill_add_timeis", func(t *testing.T) {
		client, server := setup()
		client.mockResolvedValue(obj("added_timeis", obj("id", "t1", "title", "coding"), "added_kyou", obj("id", "t1"), "errors", arr()))
		result, err := serverToolCall(t, server, "gkill_add_timeis", obj("title", "coding"))
		expectNoError(t, err)
		expectEqual(t, client.calls[0].Pathname, "/api/add_timeis")
		expectEqual(t, objAt(t, result, "added_timeis").Value("title"), "coding")
	})

	t.Run("dispatches gkill_add_mi", func(t *testing.T) {
		client, server := setup()
		client.mockResolvedValue(obj("added_mi", obj("id", "m1", "title", "fix bug", "board_name", "dev"), "added_kyou", obj("id", "m1"), "errors", arr()))
		result, err := serverToolCall(t, server, "gkill_add_mi", obj("title", "fix bug", "board_name", "dev"))
		expectNoError(t, err)
		expectEqual(t, client.calls[0].Pathname, "/api/add_mi")
		expectEqual(t, objAt(t, result, "added_mi").Value("title"), "fix bug")
	})

	t.Run("dispatches gkill_add_kc", func(t *testing.T) {
		client, server := setup()
		client.mockResolvedValue(obj("added_kc", obj("id", "c1", "num_value", 42), "added_kyou", obj("id", "c1"), "errors", arr()))
		result, err := serverToolCall(t, server, "gkill_add_kc", obj("title", "steps", "num_value", 42))
		expectNoError(t, err)
		expectEqual(t, client.calls[0].Pathname, "/api/add_kc")
		expectEqual(t, objAt(t, result, "added_kc").Value("num_value"), 42)
	})

	t.Run("dispatches gkill_add_tag", func(t *testing.T) {
		client, server := setup()
		client.mockResolvedValue(obj("added_tag", obj("id", "tg1", "tag", "important"), "added_kyou", obj("id", "tg1"), "errors", arr()))
		result, err := serverToolCall(t, server, "gkill_add_tag", obj("tag", "important", "target_id", "k1"))
		expectNoError(t, err)
		expectEqual(t, client.calls[0].Pathname, "/api/add_tag")
		expectEqual(t, objAt(t, result, "added_tag").Value("tag"), "important")
	})

	t.Run("dispatches gkill_add_text", func(t *testing.T) {
		client, server := setup()
		client.mockResolvedValue(obj("added_text", obj("id", "tx1", "text", "note"), "added_kyou", obj("id", "tx1"), "errors", arr()))
		result, err := serverToolCall(t, server, "gkill_add_text", obj("text", "note", "target_id", "k1"))
		expectNoError(t, err)
		expectEqual(t, client.calls[0].Pathname, "/api/add_text")
		expectEqual(t, objAt(t, result, "added_text").Value("text"), "note")
	})

	t.Run("dispatches gkill_submit_kftl", func(t *testing.T) {
		client, server := setup()
		client.mockResolvedValue(obj("messages", arr(obj("message", "ok")), "errors", arr()))
		result, err := serverToolCall(t, server, "gkill_submit_kftl", obj("kftl_text", "/mi Buy milk"))
		expectNoError(t, err)
		expectEqual(t, client.calls[0].Pathname, "/api/submit_kftl_text")
		expectEqual(t, len(arrAt(t, result, "messages")), 1)
	})

	t.Run("dispatches gkill_delete_kyou", func(t *testing.T) {
		client, server := setup()
		// First call: GET to fetch current entity
		client.mockResolvedValueOnce(obj("kmemo_histories", arr(obj("id", "k1", "content", "hello", "is_deleted", false, "rep_name", "rep1")), "errors", arr()))
		// Second call: UPDATE with is_deleted=true
		client.mockResolvedValueOnce(obj("updated_kmemo", obj("id", "k1", "is_deleted", true), "updated_kyou", obj("id", "k1"), "errors", arr()))
		result, err := serverToolCall(t, server, "gkill_delete_kyou", obj("id", "k1", "data_type", "kmemo"))
		expectNoError(t, err)
		expectEqual(t, client.calls[0].Pathname, "/api/get_kmemo")
		expectEqual(t, client.calls[1].Pathname, "/api/update_kmemo")
		expectEqual(t, objAt(t, result, "updated_kmemo").Value("is_deleted"), true)
		expectEqual(t, objAt(t, result, "updated_kmemo").Value("content"), "hello")
	})
}

// ---------------------------------------------------------------------------
// handleToolCall — entity defaults for write
// ---------------------------------------------------------------------------
func TestReadWriteHandleToolCallEntityDefaults(t *testing.T) {
	t.Run("sets common fields on write entities", func(t *testing.T) {
		client := createReadWriteMockClient()
		client.mockResolvedValue(obj("added_kmemo", obj("id", "k1"), "added_kyou", obj("id", "k1"), "errors", arr()))
		server := NewReadWriteServer(client, nil)

		_, err := serverToolCall(t, server, "gkill_add_kmemo", obj("content", "test"))
		expectNoError(t, err)
		kmemo := objAt(t, client.calls[0].Body, "kmemo")
		expectTrue(t, jsTruthy(kmemo.Value("id")), "id missing")
		expectEqual(t, kmemo.Value("rep_name"), "")
		expectEqual(t, kmemo.Value("data_type"), "kmemo")
		expectEqual(t, kmemo.Value("create_app"), "gkill_mcp_readwrite")
		expectEqual(t, kmemo.Value("create_device"), "mcp")
		expectEqual(t, kmemo.Value("create_user"), "testuser")
		expectEqual(t, kmemo.Value("is_deleted"), false)
	})
}

// ---------------------------------------------------------------------------
// handleToolCall — plugin tools
// ---------------------------------------------------------------------------
func TestReadWriteHandleToolCallPluginTools(t *testing.T) {
	t.Run("dispatches gkill_get_plugin_list to /api/get_plugin_list", func(t *testing.T) {
		client := createReadWriteMockClient()
		client.mockResolvedValue(obj("plugins", arr(obj("name", "gkill_plugin_claudecode", "rep_name", "Claude Code", "is_alive", true)), "errors", arr()))
		server := NewReadWriteServer(client, nil)

		result, err := serverToolCall(t, server, "gkill_get_plugin_list", obj())
		expectNoError(t, err)

		expectCalledWith(t, client, "/api/get_plugin_list", obj(), true, "")
		expectEqual(t, len(arrAt(t, result, "plugins")), 1)
	})

	t.Run("gkill_get_kyous leaves plugin bodies alone by default", func(t *testing.T) {
		client := createReadWriteMockClient()
		client.mockResolvedValue(obj("kyous", arr(pluginKyouResult()), "total_count", 1, "returned_count", 1, "errors", arr()))
		server := NewReadWriteServer(client, nil)

		result, err := serverToolCall(t, server, "gkill_get_kyous", obj())
		expectNoError(t, err)

		expectEqual(t, len(client.calls), 1)
		expectTrue(t, !result.Defined("plugin_content"), "plugin_content set")
		expectTrue(t, !objAt(t, arrAt(t, result, "kyous")[0], "payload").Defined("content_status"), "content_status set")
	})

	t.Run("gkill_get_kyous inlines plugin bodies when include_plugin_content is true", func(t *testing.T) {
		client := createReadWriteMockClient()
		client.mockImplementation(func(pathname string, _ *jsonobj.Object) (*jsonobj.Object, error) {
			if pathname == "/api/get_kyous_mcp" {
				return obj("kyous", arr(pluginKyouResult()), "total_count", 1, "returned_count", 1, "errors", arr()), nil
			}
			return obj("html", "<html><head><style>p{color:red}</style></head><body><p>会話の本文</p></body></html>", "errors", arr()), nil
		})
		server := NewReadWriteServer(client, nil)

		result, err := serverToolCall(t, server, "gkill_get_kyous", obj("include_plugin_content", true))
		expectNoError(t, err)

		expectCalledWith(t, client, "/api/get_plugin_content_html", obj("rep_name", "Claude Code", "kyou_id", "kyou-1"), true, "")
		payload := objAt(t, arrAt(t, result, "kyous")[0], "payload")
		expectEqual(t, payload.Value("content_text"), "会話の本文")
		expectEqual(t, payload.Value("content_status"), "ok")
		expectEqual(t, objAt(t, result, "plugin_content").Value("inlined"), 1)
	})

	t.Run("gkill_get_kyous returns raw html when plugin_content_format is html", func(t *testing.T) {
		html := "<html><body><p>本文</p></body></html>"
		client := createReadWriteMockClient()
		client.mockImplementation(func(pathname string, _ *jsonobj.Object) (*jsonobj.Object, error) {
			if pathname == "/api/get_kyous_mcp" {
				return obj("kyous", arr(pluginKyouResult()), "total_count", 1, "returned_count", 1, "errors", arr()), nil
			}
			return obj("html", html, "errors", arr()), nil
		})
		server := NewReadWriteServer(client, nil)

		result, err := serverToolCall(t, server, "gkill_get_kyous", obj("include_plugin_content", true, "plugin_content_format", "html"))
		expectNoError(t, err)

		payload := objAt(t, arrAt(t, result, "kyous")[0], "payload")
		expectEqual(t, payload.Value("content_html"), html)
		expectTrue(t, !payload.Defined("content_text"), "content_text set")
	})

	t.Run("gkill_get_kyous still returns results when a plugin body fetch fails", func(t *testing.T) {
		client := createReadWriteMockClient()
		client.mockImplementation(func(pathname string, _ *jsonobj.Object) (*jsonobj.Object, error) {
			if pathname == "/api/get_kyous_mcp" {
				return obj("kyous", arr(pluginKyouResult()), "total_count", 1, "returned_count", 1, "errors", arr()), nil
			}
			return nil, Errorf("plugin is down")
		})
		server := NewReadWriteServer(client, nil)

		result, err := serverToolCall(t, server, "gkill_get_kyous", obj("include_plugin_content", true))
		expectNoError(t, err)

		expectEqual(t, len(arrAt(t, result, "kyous")), 1)
		expectEqual(t, objAt(t, arrAt(t, result, "kyous")[0], "payload").Value("content_status"), "error")
		expectEqual(t, objAt(t, result, "plugin_content").Value("errors"), 1)
	})
}

// ---------------------------------------------------------------------------
// handleToolCall — unknown tool
// ---------------------------------------------------------------------------
func TestReadWriteHandleToolCallErrorCases(t *testing.T) {
	t.Run("throws for unknown tool", func(t *testing.T) {
		server := NewReadWriteServer(createReadWriteMockClient(), nil)
		_, err := serverToolCall(t, server, "unknown_tool", obj())
		expectErrorContains(t, err, "Unknown tool")
	})

	t.Run("the removed plugin content tool falls through to the replacement hint", func(t *testing.T) {
		// gkill_get_plugin_content は削除済みで IsPluginToolName がもう受け付けない。
		// 旧セッションのクライアントが呼ぶと plugin → read → write と流れ、
		// write_handlers の default で removedToolHints (constants.go) の案内文になる。
		// HandlePluginToolCall 直叩き (plugin_tools_test.go) とは別の、実ディスパッチ経路。
		client := createReadWriteMockClient()
		server := NewReadWriteServer(client, nil)

		_, err := serverToolCall(t, server, "gkill_get_plugin_content", obj())
		expectErrorMatches(t, err, `include_plugin_content:true`)
		_, err = serverToolCall(t, server, "gkill_get_plugin_content", obj())
		expectErrorMatches(t, err, `gkill_get_kyous`)
		// 案内で完結し、gkill への API 呼び出しは発生しない
		expectEqual(t, len(client.calls), 0)
	})
}

// ---------------------------------------------------------------------------
// JSON-RPC protocol
// ---------------------------------------------------------------------------
func TestReadWriteJSONRPCProtocol(t *testing.T) {
	setup := func() *Server { return NewReadWriteServer(createReadWriteMockClient(), nil) }

	t.Run("initialize returns server info", func(t *testing.T) {
		response := serverMessage(t, setup(), obj("jsonrpc", "2.0", "id", 1, "method", "initialize", "params", obj()))
		expectEqual(t, objAt(t, response, "result", "serverInfo").Value("name"), "gkill-readwrite-mcp")
		expectEqual(t, objAt(t, response, "result").Value("protocolVersion"), "2024-11-05")
	})

	t.Run("ping returns empty result", func(t *testing.T) {
		response := serverMessage(t, setup(), obj("jsonrpc", "2.0", "id", 2, "method", "ping"))
		expectEqual(t, response.Value("result"), obj())
	})

	t.Run("tools/list returns 33 tools", func(t *testing.T) {
		response := serverMessage(t, setup(), obj("jsonrpc", "2.0", "id", 3, "method", "tools/list"))
		expectEqual(t, len(arrAt(t, response, "result", "tools")), 33)
	})

	t.Run("tools/list includes all expected tool names", func(t *testing.T) {
		response := serverMessage(t, setup(), obj("jsonrpc", "2.0", "id", 4, "method", "tools/list"))
		names := NewStringSet()
		for _, tool := range arrAt(t, response, "result", "tools") {
			names.Add(strAt(t, objAt(t, tool), "name"))
		}
		for _, name := range []string{
			// Read tools
			"gkill_status", "gkill_get_kyous", "gkill_get_mi_board_list", "gkill_get_all_tag_names", "gkill_get_all_rep_names",
			"gkill_get_gps_log", "gkill_get_application_config", "gkill_get_idf_file",
			// Write tools
			"gkill_add_kmemo", "gkill_add_urlog", "gkill_add_nlog", "gkill_add_lantana", "gkill_add_timeis", "gkill_add_mi",
			"gkill_add_kc", "gkill_add_tag", "gkill_add_text", "gkill_submit_kftl", "gkill_delete_kyou",
			// Update tools
			"gkill_update_kmemo", "gkill_update_urlog", "gkill_update_nlog", "gkill_update_lantana", "gkill_update_timeis",
			"gkill_update_mi", "gkill_update_kc", "gkill_update_tag", "gkill_update_text",
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

	t.Run("notifications/initialized returns null", func(t *testing.T) {
		response := serverMessage(t, setup(), obj("jsonrpc", "2.0", "method", "notifications/initialized"))
		expectTrue(t, response == nil, "notification got a response")
	})
}

// ---------------------------------------------------------------------------
// buildToolResult — IDF image block
// ---------------------------------------------------------------------------
func TestReadWriteBuildToolResult(t *testing.T) {
	t.Run("includes image block for IDF images", func(t *testing.T) {
		server := NewReadWriteServer(createReadWriteMockClient(), nil)
		payload := obj(
			"file_name", "photo.png",
			"mime_type", "image/png",
			"file_size_bytes", 1024,
			"is_image", true,
			"file_content_base64", "iVBORw0KGgo=",
		)
		result := server.BuildToolResult("gkill_get_idf_file", payload, false, nil)
		expectEqual(t, result.Value("isError"), false)
		var imageBlock *jsonobj.Object
		for _, c := range arrAt(t, result, "content") {
			if objAt(t, c).Value("type") == "image" {
				imageBlock = objAt(t, c)
			}
		}
		expectTrue(t, imageBlock != nil, "image block missing")
		expectEqual(t, imageBlock.Value("data"), "iVBORw0KGgo=")
		expectEqual(t, imageBlock.Value("mimeType"), "image/png")
	})

	t.Run("excludes file_content_base64 from text for IDF", func(t *testing.T) {
		server := NewReadWriteServer(createReadWriteMockClient(), nil)
		payload := obj(
			"file_name", "doc.pdf",
			"mime_type", "application/pdf",
			"file_size_bytes", 2048,
			"is_image", false,
			"file_content_base64", "AAAA",
		)
		result := server.BuildToolResult("gkill_get_idf_file", payload, false, nil)
		mustNotContain(t, firstText(t, result), "AAAA")
	})

	t.Run("includes structuredContent for write tools", func(t *testing.T) {
		server := NewReadWriteServer(createReadWriteMockClient(), nil)
		result := server.BuildToolResult("gkill_add_kmemo", obj("added_kmemo", obj("id", "k1")), false, nil)
		expectTrue(t, result.Defined("structuredContent"), "structuredContent missing")
		mustContain(t, firstText(t, result), "Created kmemo: k1")
	})
}

// ---------------------------------------------------------------------------
// handlePayload — batch
// ---------------------------------------------------------------------------
func TestReadWriteHandlePayload(t *testing.T) {
	t.Run("handles batch of messages", func(t *testing.T) {
		server := NewReadWriteServer(createReadWriteMockClient(), nil)
		result := server.HandlePayload(context.Background(), arr(
			obj("jsonrpc", "2.0", "id", 1, "method", "ping"),
			obj("jsonrpc", "2.0", "id", 2, "method", "ping"),
		), nil)
		responses, ok := jsonobj.AsArray(result)
		expectTrue(t, ok, "batch did not return an array")
		expectEqual(t, len(responses), 2)
	})

	t.Run("handles single message", func(t *testing.T) {
		server := NewReadWriteServer(createReadWriteMockClient(), nil)
		result := server.HandlePayload(context.Background(), obj("jsonrpc", "2.0", "id", 1, "method", "ping"), nil)
		expectEqual(t, objAt(t, result).Value("result"), obj())
	})
}
