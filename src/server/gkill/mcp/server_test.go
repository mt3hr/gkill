package mcp

// 読み取りサーバ（NewReadServer）の検査。
//
// サーバはモックの GkillAPI で検査するので、実 HTTP 呼び出しは発生しない。

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/mt3hr/gkill/src/server/gkill/mcp/jsonobj"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func createMockClient() *mockClient {
	return &mockClient{
		callApi: func(_ string, _ *jsonobj.Object, _ bool, _ string) (*jsonobj.Object, error) {
			return obj("errors", arr(), "messages", arr()), nil
		},
		fetchFile: func(_ string, _ string) (*FileResponse, error) {
			return &FileResponse{Buffer: []byte("test"), ContentType: "application/octet-stream"}, nil
		},
		login:         func() (string, error) { return "mock-session-id", nil },
		defaultLocale: "ja",
	}
}

// mockResolvedValue は以後の全 CallApi にこの応答を返す（vitest の mockResolvedValue）。
func (m *mockClient) mockResolvedValue(response *jsonobj.Object) {
	m.callApi = func(_ string, _ *jsonobj.Object, _ bool, _ string) (*jsonobj.Object, error) { return response, nil }
}

// mockImplementation は pathname ごとに応答を決める。
func (m *mockClient) mockImplementation(impl func(pathname string, body *jsonobj.Object) (*jsonobj.Object, error)) {
	m.callApi = func(pathname string, body *jsonobj.Object, _ bool, _ string) (*jsonobj.Object, error) {
		return impl(pathname, body)
	}
}

func (m *mockClient) mockFetchFile(file *FileResponse) {
	m.fetchFile = func(_ string, _ string) (*FileResponse, error) { return file, nil }
}

// get_kyous が返すプラグイン Kyou 1件分。
func pluginKyouResult() *jsonobj.Object {
	return obj(
		"id", "kyou-1",
		"rep_name", "Claude Code",
		"data_type", "claude_code_message",
		"related_time", "2026-08-05T10:00:00+09:00",
		"payload", obj("kind", "plugin", "plugin_name", "gkill_plugin_claudecode"),
	)
}

func serverToolCall(t *testing.T, server *Server, name string, args *jsonobj.Object) (*jsonobj.Object, error) {
	t.Helper()
	return server.HandleToolCall(context.Background(), name, args, nil)
}

func serverMessage(t *testing.T, server *Server, message *jsonobj.Object) *jsonobj.Object {
	t.Helper()
	return server.HandleMessage(context.Background(), message, nil)
}

func firstText(t *testing.T, result *jsonobj.Object) string {
	t.Helper()
	return strAt(t, objAt(t, arrAt(t, result, "content")[0]), "text")
}

// ---------------------------------------------------------------------------
// Constructor
// ---------------------------------------------------------------------------
func TestMcpServerConstructor(t *testing.T) {
	t.Run("accepts a client", func(t *testing.T) {
		mock := createMockClient()
		server := NewReadServer(mock, nil)
		expectTrue(t, server.Client == GkillAPI(mock), "client is not the one passed in")
	})
}

// ---------------------------------------------------------------------------
// handleToolCall dispatch
// ---------------------------------------------------------------------------
func TestHandleToolCall(t *testing.T) {
	setup := func() (*mockClient, *Server) {
		client := createMockClient()
		return client, NewReadServer(client, nil)
	}

	t.Run("dispatches gkill_get_kyous to /api/get_kyous_mcp", func(t *testing.T) {
		client, server := setup()
		client.mockResolvedValue(obj("kyous", arr(obj("id", "1")), "total_count", 1, "returned_count", 1, "has_more", false, "errors", arr()))

		result, err := serverToolCall(t, server, "gkill_get_kyous", obj("query", obj()))
		expectNoError(t, err)

		expectEqual(t, len(client.calls), 1)
		expectEqual(t, client.calls[0].Pathname, "/api/get_kyous_mcp")
		expectEqual(t, len(arrAt(t, result, "kyous")), 1)
		expectEqual(t, result.Value("returned_count"), 1)
		// M-05: 付随データの取得が全て成功なら partial は付かない
		expectTrue(t, !result.Defined("partial"), "partial set")
	})

	t.Run("gkill_get_kyous surfaces partial/warnings when attached data fetch failed (M-05)", func(t *testing.T) {
		client, server := setup()
		client.mockResolvedValue(obj(
			"kyous", arr(obj("id", "1")),
			"total_count", 1,
			"returned_count", 1,
			"has_more", false,
			"partial", true,
			"warnings", strs("failed to fetch tags for 2 record(s); attached data is incomplete"),
			"errors", arr(),
		))

		result, err := serverToolCall(t, server, "gkill_get_kyous", obj("query", obj()))
		expectNoError(t, err)

		expectEqual(t, result.Value("partial"), true)
		expectEqual(t, result.Value("warnings"), strs("failed to fetch tags for 2 record(s); attached data is incomplete"))
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

	t.Run("dispatches gkill_get_all_rep_names to /api/get_all_rep_names", func(t *testing.T) {
		client, server := setup()
		client.mockResolvedValue(obj("rep_names", strs("repo1"), "errors", arr()))

		result, err := serverToolCall(t, server, "gkill_get_all_rep_names", obj())
		expectNoError(t, err)

		expectEqual(t, client.calls[0].Pathname, "/api/get_all_rep_names")
		expectEqual(t, result.Value("rep_names"), strs("repo1"))
	})

	t.Run("dispatches gkill_get_gps_log to /api/get_gps_log", func(t *testing.T) {
		client, server := setup()
		client.mockResolvedValue(obj("gps_logs", arr(obj("lat", 35.0, "lng", 139.0)), "errors", arr()))

		result, err := serverToolCall(t, server, "gkill_get_gps_log", obj("start_date", "2026-01-01", "end_date", "2026-01-31"))
		expectNoError(t, err)

		expectEqual(t, client.calls[0].Pathname, "/api/get_gps_log")
		expectEqual(t, len(arrAt(t, result, "gps_logs")), 1)
	})

	t.Run("dispatches gkill_get_application_config to /api/get_application_config", func(t *testing.T) {
		client, server := setup()
		client.mockResolvedValue(obj(
			"application_config", obj(
				"tag_struct", obj("tags", arr()),
				"mi_board_struct", obj(),
				"rep_struct", obj(),
				"rep_type_struct", obj(),
				"device_struct", obj(),
				"kftl_template_struct", obj(),
				"mi_default_board", "default",
				"show_tags_in_list", true,
			),
			"errors", arr(),
		))

		result, err := serverToolCall(t, server, "gkill_get_application_config", obj())
		expectNoError(t, err)

		expectEqual(t, client.calls[0].Pathname, "/api/get_application_config")
		expectTrue(t, result.Defined("tag_struct"), "tag_struct missing")
		expectEqual(t, result.Value("mi_default_board"), "default")
	})

	t.Run("dispatches gkill_get_plugin_list to /api/get_plugin_list", func(t *testing.T) {
		client, server := setup()
		client.mockResolvedValue(obj("plugins", arr(obj("name", "gkill_plugin_claudecode", "rep_name", "Claude Code", "is_alive", true)), "errors", arr()))

		result, err := serverToolCall(t, server, "gkill_get_plugin_list", obj())
		expectNoError(t, err)

		expectCalledWith(t, client, "/api/get_plugin_list", obj(), true, "")
		expectEqual(t, len(arrAt(t, result, "plugins")), 1)
	})

	t.Run("passes currentSessionId to plugin tools too", func(t *testing.T) {
		client, server := setup()
		client.mockResolvedValue(obj("plugins", arr(), "errors", arr()))
		server.CurrentSessionID = "oauth-session-xyz"

		_, err := serverToolCall(t, server, "gkill_get_plugin_list", obj())
		expectNoError(t, err)

		call := client.calls[0]
		expectEqual(t, call.Pathname, "/api/get_plugin_list")
		expectEqual(t, call.RequiresAuth, true)
		expectEqual(t, call.SID, "oauth-session-xyz")
	})

	t.Run("gkill_get_kyous leaves plugin bodies alone by default", func(t *testing.T) {
		client, server := setup()
		client.mockResolvedValue(obj("kyous", arr(pluginKyouResult()), "total_count", 1, "returned_count", 1, "errors", arr()))

		result, err := serverToolCall(t, server, "gkill_get_kyous", obj())
		expectNoError(t, err)

		expectEqual(t, len(client.calls), 1)
		expectTrue(t, !result.Defined("plugin_content"), "plugin_content set")
		expectTrue(t, !objAt(t, arrAt(t, result, "kyous")[0], "payload").Defined("content_status"), "content_status set")
	})

	t.Run("gkill_get_kyous inlines plugin bodies when include_plugin_content is true", func(t *testing.T) {
		client, server := setup()
		client.mockImplementation(func(pathname string, _ *jsonobj.Object) (*jsonobj.Object, error) {
			if pathname == "/api/get_kyous_mcp" {
				return obj("kyous", arr(pluginKyouResult()), "total_count", 1, "returned_count", 1, "errors", arr()), nil
			}
			return obj("html", "<html><head><style>p{color:red}</style></head><body><p>会話の本文</p></body></html>", "errors", arr()), nil
		})

		result, err := serverToolCall(t, server, "gkill_get_kyous", obj("include_plugin_content", true))
		expectNoError(t, err)

		expectCalledWith(t, client, "/api/get_plugin_content_html", obj("rep_name", "Claude Code", "kyou_id", "kyou-1"), true, "")
		payload := objAt(t, arrAt(t, result, "kyous")[0], "payload")
		expectEqual(t, payload.Value("content_text"), "会話の本文")
		expectEqual(t, payload.Value("content_status"), "ok")
		expectEqual(t, objAt(t, result, "plugin_content").Value("inlined"), 1)
	})

	t.Run("gkill_get_kyous does not forward the inline args to the gkill endpoint", func(t *testing.T) {
		client, server := setup()
		client.mockImplementation(func(pathname string, _ *jsonobj.Object) (*jsonobj.Object, error) {
			if pathname == "/api/get_kyous_mcp" {
				return obj("kyous", arr(pluginKyouResult()), "total_count", 1, "returned_count", 1, "errors", arr()), nil
			}
			return obj("html", "<p>x</p>", "errors", arr()), nil
		})

		_, err := serverToolCall(t, server, "gkill_get_kyous", obj(
			"include_plugin_content", true,
			"plugin_content_max_text_length", 100,
			"plugin_content_format", "text",
		))
		expectNoError(t, err)

		body := client.calls[0].Body
		expectTrue(t, !body.Has("include_plugin_content"), "include_plugin_content forwarded")
		expectTrue(t, !body.Has("plugin_content_max_text_length"), "plugin_content_max_text_length forwarded")
		expectTrue(t, !body.Has("plugin_content_format"), "plugin_content_format forwarded")
	})

	t.Run("gkill_get_kyous still returns results when a plugin body fetch fails", func(t *testing.T) {
		client, server := setup()
		client.mockImplementation(func(pathname string, _ *jsonobj.Object) (*jsonobj.Object, error) {
			if pathname == "/api/get_kyous_mcp" {
				return obj("kyous", arr(pluginKyouResult()), "total_count", 1, "returned_count", 1, "errors", arr()), nil
			}
			return nil, Errorf("plugin is down")
		})

		result, err := serverToolCall(t, server, "gkill_get_kyous", obj("include_plugin_content", true))
		expectNoError(t, err)

		expectEqual(t, len(arrAt(t, result, "kyous")), 1)
		expectEqual(t, objAt(t, arrAt(t, result, "kyous")[0], "payload").Value("content_status"), "error")
		expectEqual(t, objAt(t, result, "plugin_content").Value("errors"), 1)
	})

	t.Run("throws for unknown tool name", func(t *testing.T) {
		_, server := setup()
		_, err := serverToolCall(t, server, "nonexistent_tool", obj())
		expectErrorContains(t, err, "Unknown tool: nonexistent_tool")
	})

	t.Run("a removed tool name is dispatched to the replacement hint, not a dead end", func(t *testing.T) {
		// ツール一覧はクライアントのセッション寿命で固定されるので、削除済みの
		// gkill_get_idf_file_path は旧セッションから呼ばれ続ける。plugin でも read でも
		// ない名前がサーバディスパッチの行き止まりへ落ちたとき、removedToolHints
		// (constants.go) の案内文まで届くことを実経路で確かめる。
		// ツール名の echo 自体が gkill_get_idf_file を含むので、案内文にしか無い文言で見る
		client, server := setup()
		_, err := serverToolCall(t, server, "gkill_get_idf_file_path", obj())
		expectErrorMatches(t, err, `removed on 2026-08-24`)
		_, err = serverToolCall(t, server, "gkill_get_idf_file_path", obj())
		expectErrorMatches(t, err, `otherwise call gkill_get_idf_file`)
		// 案内で完結し、gkill への API 呼び出しは発生しない
		expectEqual(t, len(client.calls), 0)
		expectEqual(t, len(client.fetchCalls), 0)
	})

	t.Run("passes currentSessionId as sessionIdOverride to callApi", func(t *testing.T) {
		client, server := setup()
		client.mockResolvedValue(obj("tag_names", strs("t1"), "errors", arr()))
		server.CurrentSessionID = "oauth-session-xyz"

		_, err := serverToolCall(t, server, "gkill_get_all_tag_names", obj())
		expectNoError(t, err)

		// 4th argument should be the session override
		call := client.calls[0]
		expectEqual(t, call.Pathname, "/api/get_all_tag_names")
		expectEqual(t, call.RequiresAuth, true)
		expectEqual(t, call.SID, "oauth-session-xyz")
	})

	t.Run("passes null sessionIdOverride when currentSessionId is not set", func(t *testing.T) {
		client, server := setup()
		client.mockResolvedValue(obj("tag_names", strs("t1"), "errors", arr()))

		_, err := serverToolCall(t, server, "gkill_get_all_tag_names", obj())
		expectNoError(t, err)

		call := client.calls[0]
		expectEqual(t, call.Pathname, "/api/get_all_tag_names")
		expectEqual(t, call.RequiresAuth, true)
		expectEqual(t, call.SID, "")
	})
}

// ---------------------------------------------------------------------------
// handleMessage — JSON-RPC level
// ---------------------------------------------------------------------------
func TestHandleMessage(t *testing.T) {
	setup := func() (*mockClient, *Server) {
		client := createMockClient()
		return client, NewReadServer(client, nil)
	}

	t.Run("responds to initialize with server info", func(t *testing.T) {
		_, server := setup()
		response := serverMessage(t, server, obj("jsonrpc", "2.0", "method", "initialize", "id", 1))

		expectEqual(t, response.Value("jsonrpc"), "2.0")
		expectEqual(t, response.Value("id"), 1)
		expectEqual(t, objAt(t, response, "result", "serverInfo").Value("name"), "gkill-read-mcp")
		expectTrue(t, objAt(t, response, "result", "capabilities").Defined("tools"), "capabilities.tools missing")
	})

	// ツール一覧の世代（2026-09-14 レビュー P0）。initialize の version と
	// gkill_status の description 末尾と gkill_status の応答が、同じ値を指すこと。
	t.Run("initialize version, the stamped gkill_status description and the gkill_status response agree on schema_revision", func(t *testing.T) {
		client, server := setup()
		initialize := serverMessage(t, server, obj("jsonrpc", "2.0", "method", "initialize", "id", 1))
		version := strAt(t, objAt(t, initialize, "result", "serverInfo"), "version")
		idx := strings.LastIndex(version, "+schema.")
		expectTrue(t, idx >= 0, "version %q lacks +schema.", version)
		revision := version[idx+len("+schema."):]
		mustMatch(t, revision, `^[0-9a-f]{12}$`)

		list := serverMessage(t, server, obj("jsonrpc", "2.0", "method", "tools/list", "id", 2))
		tools := arrAt(t, list, "result", "tools")
		var status *jsonobj.Object
		for _, tool := range tools {
			if objAt(t, tool).Value("name") == "gkill_status" {
				status = objAt(t, tool)
			}
		}
		expectTrue(t, status != nil, "gkill_status missing")
		expectTrue(t, strings.HasSuffix(strAt(t, status, "description"), " [schema_revision: "+revision+"]"), "description not stamped")
		// 他のツールには焼き込まない
		for _, tool := range tools {
			if objAt(t, tool).Value("name") != "gkill_status" {
				mustNotContain(t, strAt(t, objAt(t, tool), "description"), "[schema_revision:")
			}
		}

		client.mockResolvedValue(obj("application_config", obj("user_id", "testuser", "device", "testdevice", "version", "9.9.9")))
		call := serverMessage(t, server, obj(
			"jsonrpc", "2.0",
			"method", "tools/call",
			"id", 3,
			"params", obj("name", "gkill_status", "arguments", obj()),
		))
		expectEqual(t, objAt(t, call, "result").Value("isError"), false)
		structured := objAt(t, call, "result", "structuredContent")
		expectEqual(t, structured.Value("schema_revision"), revision)
		expectEqual(t, structured.Value("server_kind"), "read")
		expectEqual(t, structured.Value("tool_count"), len(tools))
		expectEqual(t, structured.Value("transport"), "http") // IsLocalTransport は既定 false
		mustContain(t, firstText(t, objAt(t, call, "result")), "Connected to testuser@testdevice via read server")
	})

	t.Run("responds to ping", func(t *testing.T) {
		_, server := setup()
		response := serverMessage(t, server, obj("jsonrpc", "2.0", "method", "ping", "id", 42))
		expectEqual(t, response, obj("jsonrpc", "2.0", "id", 42, "result", obj()))
	})

	t.Run("responds to tools/list with tool definitions", func(t *testing.T) {
		_, server := setup()
		response := serverMessage(t, server, obj("jsonrpc", "2.0", "method", "tools/list", "id", 2))

		expectEqual(t, response.Value("jsonrpc"), "2.0")
		expectEqual(t, response.Value("id"), 2)
		tools := arrAt(t, response, "result", "tools")
		expectEqual(t, len(tools), 14)

		names := NewStringSet()
		for _, tool := range tools {
			names.Add(strAt(t, objAt(t, tool), "name"))
		}
		for _, name := range []string{"gkill_status", "gkill_get_kyous", "gkill_get_all_tag_names", "gkill_get_idf_file", "gkill_get_plugin_list"} {
			expectTrue(t, names.Has(name), "%s missing", name)
		}
		expectTrue(t, !names.Has("gkill_get_plugin_content"), "gkill_get_plugin_content present")
	})

	t.Run("responds to tools/call with tool result", func(t *testing.T) {
		client, server := setup()
		client.mockResolvedValue(obj("tag_names", strs("alpha", "beta"), "errors", arr()))

		response := serverMessage(t, server, obj(
			"jsonrpc", "2.0",
			"method", "tools/call",
			"params", obj("name", "gkill_get_all_tag_names", "arguments", obj()),
			"id", 3,
		))

		expectEqual(t, response.Value("jsonrpc"), "2.0")
		expectEqual(t, response.Value("id"), 3)
		expectEqual(t, objAt(t, response, "result").Value("isError"), false)
		mustContain(t, firstText(t, objAt(t, response, "result")), "2 tag names")
		expectEqual(t, objAt(t, response, "result", "structuredContent").Value("tag_names"), strs("alpha", "beta"))
	})

	t.Run("returns error result for unknown tool via tools/call", func(t *testing.T) {
		_, server := setup()
		response := serverMessage(t, server, obj(
			"jsonrpc", "2.0",
			"method", "tools/call",
			"params", obj("name", "bad_tool", "arguments", obj()),
			"id", 4,
		))

		expectEqual(t, response.Value("jsonrpc"), "2.0")
		expectEqual(t, response.Value("id"), 4)
		expectEqual(t, objAt(t, response, "result").Value("isError"), true)
		mustContain(t, firstText(t, objAt(t, response, "result")), "bad_tool failed")
	})

	t.Run("returns method-not-found for unknown methods", func(t *testing.T) {
		_, server := setup()
		response := serverMessage(t, server, obj("jsonrpc", "2.0", "method", "nonexistent/method", "id", 5))

		expectEqual(t, objAt(t, response, "error").Value("code"), -32601)
		mustContain(t, strAt(t, objAt(t, response, "error"), "message"), "nonexistent/method")
	})

	t.Run("returns null for notifications/initialized", func(t *testing.T) {
		_, server := setup()
		response := serverMessage(t, server, obj("jsonrpc", "2.0", "method", "notifications/initialized"))
		expectTrue(t, response == nil, "notification got a response")
	})

	t.Run("returns invalid-request for malformed messages", func(t *testing.T) {
		_, server := setup()
		response := serverMessage(t, server, obj("not_jsonrpc", true))

		expectEqual(t, objAt(t, response, "error").Value("code"), -32600)
		expectEqual(t, objAt(t, response, "error").Value("message"), "Invalid Request")
	})
}

// ---------------------------------------------------------------------------
// gkill_get_idf_file tool
// ---------------------------------------------------------------------------
func TestGkillGetIdfFile(t *testing.T) {
	setup := func() (*mockClient, *Server) {
		client := createMockClient()
		return client, NewReadServer(client, nil)
	}

	t.Run("dispatches to fetchFile with correct path", func(t *testing.T) {
		client, server := setup()
		fileContent := []byte{0x89, 0x50, 0x4e, 0x47}
		client.mockFetchFile(&FileResponse{Buffer: fileContent, ContentType: "image/png"})
		server.CurrentSessionID = "test-session"

		result, err := serverToolCall(t, server, "gkill_get_idf_file", obj("rep_name", "my_repo", "file_name", "photo.png"))
		expectNoError(t, err)

		expectFetchCalledWith(t, client, "/files/my_repo/photo.png", "test-session")
		expectEqual(t, result.Value("file_name"), "photo.png")
		expectEqual(t, result.Value("mime_type"), "image/png")
		expectEqual(t, result.Value("file_size_bytes"), 4)
		expectEqual(t, result.Value("is_image"), true)
		expectEqual(t, result.Value("file_content_base64"), base64.StdEncoding.EncodeToString(fileContent))
	})

	t.Run("returns is_image false for non-image files", func(t *testing.T) {
		client, server := setup()
		client.mockFetchFile(&FileResponse{Buffer: []byte("data"), ContentType: "application/pdf"})
		server.CurrentSessionID = "sess"

		result, err := serverToolCall(t, server, "gkill_get_idf_file", obj("rep_name", "repo", "file_name", "doc.pdf"))
		expectNoError(t, err)

		expectEqual(t, result.Value("is_image"), false)
		expectEqual(t, result.Value("mime_type"), "application/pdf")
	})

	t.Run("handles nested file paths", func(t *testing.T) {
		client, server := setup()
		client.mockFetchFile(&FileResponse{Buffer: []byte("x"), ContentType: "text/plain"})
		server.CurrentSessionID = "sess"

		_, err := serverToolCall(t, server, "gkill_get_idf_file", obj("rep_name", "repo", "file_name", "sub/dir/file.txt"))
		expectNoError(t, err)

		expectFetchCalledWith(t, client, "/files/repo/sub/dir/file.txt", "sess")
	})

	t.Run("uses login when currentSessionId is not set", func(t *testing.T) {
		client, server := setup()
		client.mockFetchFile(&FileResponse{Buffer: []byte("x"), ContentType: "text/plain"})
		loginCalled := false
		client.login = func() (string, error) {
			loginCalled = true
			return "mock-session-id", nil
		}
		server.CurrentSessionID = ""

		_, err := serverToolCall(t, server, "gkill_get_idf_file", obj("rep_name", "repo", "file_name", "file.txt"))
		expectNoError(t, err)

		expectTrue(t, loginCalled, "login was not called")
		expectFetchCalledWith(t, client, "/files/repo/file.txt", "mock-session-id")
	})

	t.Run("buildToolResult includes image content block for images", func(t *testing.T) {
		_, server := setup()
		payload := obj(
			"file_name", "img.jpg",
			"mime_type", "image/jpeg",
			"file_size_bytes", 100,
			"is_image", true,
			"file_content_base64", "base64data",
		)

		result := server.BuildToolResult("gkill_get_idf_file", payload, false, nil)

		expectEqual(t, result.Value("isError"), false)
		content := arrAt(t, result, "content")
		expectEqual(t, len(content), 2)
		expectEqual(t, objAt(t, content[0]).Value("type"), "text")
		expectEqual(t, objAt(t, content[1]).Value("type"), "image")
		expectEqual(t, objAt(t, content[1]).Value("data"), "base64data")
		expectEqual(t, objAt(t, content[1]).Value("mimeType"), "image/jpeg")
	})

	t.Run("buildToolResult does not include image block for non-images", func(t *testing.T) {
		_, server := setup()
		payload := obj(
			"file_name", "doc.pdf",
			"mime_type", "application/pdf",
			"file_size_bytes", 200,
			"is_image", false,
			"file_content_base64", "pdfdata",
		)

		result := server.BuildToolResult("gkill_get_idf_file", payload, false, nil)

		content := arrAt(t, result, "content")
		expectEqual(t, len(content), 1)
		expectEqual(t, objAt(t, content[0]).Value("type"), "text")
	})

	// 画像のバイト列は image ブロックで届く。structuredContent にも同じ base64 が入ると
	// 1レスポンスに同じデータが2回乗り、クライアントのツール結果上限を超えて切り捨てられ、
	// 画像そのものが届かなくなる。
	t.Run("buildToolResult omits base64 from text and structuredContent for images", func(t *testing.T) {
		_, server := setup()
		payload := obj(
			"file_name", "img.jpg",
			"mime_type", "image/jpeg",
			"file_size_bytes", 100,
			"is_image", true,
			"file_content_base64", "base64data",
		)

		result := server.BuildToolResult("gkill_get_idf_file", payload, false, nil)

		expectTrue(t, !objAt(t, result, "structuredContent").Defined("file_content_base64"), "base64 kept in structuredContent")
		expectEqual(t, objAt(t, result, "structuredContent").Value("file_name"), "img.jpg")
		mustNotContain(t, firstText(t, result), "base64data")
		// image ブロックからは従来どおりバイト列が届く
		expectEqual(t, objAt(t, arrAt(t, result, "content")[1]).Value("data"), "base64data")
	})

	// 非画像には image ブロックが付かないため、structuredContent が唯一のバイト列の渡し口。
	t.Run("buildToolResult keeps base64 in structuredContent for non-images", func(t *testing.T) {
		_, server := setup()
		payload := obj(
			"file_name", "doc.pdf",
			"mime_type", "application/pdf",
			"file_size_bytes", 200,
			"is_image", false,
			"file_content_base64", "pdfdata",
		)

		result := server.BuildToolResult("gkill_get_idf_file", payload, false, nil)

		expectEqual(t, objAt(t, result, "structuredContent").Value("file_content_base64"), "pdfdata")
		mustNotContain(t, firstText(t, result), "pdfdata")
	})

	t.Run("buildToolResult strips charset parameter from the image block mimeType", func(t *testing.T) {
		_, server := setup()
		payload := obj(
			"file_name", "img.jpg",
			"mime_type", "image/jpeg; charset=binary",
			"file_size_bytes", 100,
			"is_image", true,
			"file_content_base64", "base64data",
		)

		result := server.BuildToolResult("gkill_get_idf_file", payload, false, nil)

		expectEqual(t, objAt(t, arrAt(t, result, "content")[1]).Value("mimeType"), "image/jpeg")
	})

	// structuredContent だけを見ると base64 が無く「画像が返っていない」と誤読された
	// (2026-08-30 レビュー 5.5)。image ブロックへ移した印を構造化側にも残す。
	t.Run("buildToolResult marks image payloads with image_content_attached", func(t *testing.T) {
		_, server := setup()
		payload := obj(
			"file_name", "img.jpg",
			"mime_type", "image/jpeg",
			"file_size_bytes", 100,
			"is_image", true,
			"file_content_base64", "base64data",
		)

		result := server.BuildToolResult("gkill_get_idf_file", payload, false, nil)

		expectEqual(t, objAt(t, result, "structuredContent").Value("image_content_attached"), true)
	})

	// 非画像は structuredContent が唯一のバイト列の渡し口 (base64 が残る) なので印は不要。
	t.Run("buildToolResult does not mark non-image payloads", func(t *testing.T) {
		_, server := setup()
		payload := obj(
			"file_name", "doc.pdf",
			"mime_type", "application/pdf",
			"file_size_bytes", 200,
			"is_image", false,
			"file_content_base64", "pdfdata",
		)

		result := server.BuildToolResult("gkill_get_idf_file", payload, false, nil)

		expectTrue(t, !objAt(t, result, "structuredContent").Defined("image_content_attached"), "marked a non-image")
	})

	t.Run("rejects files larger than the size limit", func(t *testing.T) {
		client, server := setup()
		client.mockFetchFile(&FileResponse{Buffer: make([]byte, MaxIDFFileBytes+1), ContentType: "video/mp4"})
		server.CurrentSessionID = "sess"

		_, err := serverToolCall(t, server, "gkill_get_idf_file", obj("rep_name", "repo", "file_name", "big.mp4"))
		expectErrorMatches(t, err, `too large`)
	})

	t.Run("points at thumb as the way out when the file is too large", func(t *testing.T) {
		client, server := setup()
		client.mockFetchFile(&FileResponse{Buffer: make([]byte, MaxIDFFileBytes+1), ContentType: "image/png"})
		server.CurrentSessionID = "sess"

		_, err := serverToolCall(t, server, "gkill_get_idf_file", obj("rep_name", "repo", "file_name", "huge.png"))
		expectErrorMatches(t, err, `retry with thumb`)
	})

	t.Run("appends thumb to the file path in the same form as file_url", func(t *testing.T) {
		client, server := setup()
		client.mockFetchFile(&FileResponse{Buffer: []byte("j"), ContentType: "image/jpeg"})
		server.CurrentSessionID = "sess"

		result, err := serverToolCall(t, server, "gkill_get_idf_file", obj("rep_name", "repo", "file_name", "photo.png", "thumb", "1024x1024"))
		expectNoError(t, err)

		expectFetchCalledWith(t, client, "/files/repo/photo.png?thumb=1024x1024", "sess")
		// 縮小して取ったことが応答から分かる (原寸と取り違えない)
		expectEqual(t, result.Value("thumb"), "1024x1024")
		expectEqual(t, result.Value("is_image"), true)
	})

	t.Run("puts is_video before thumb, matching the client's build_media_url", func(t *testing.T) {
		client, server := setup()
		client.mockFetchFile(&FileResponse{Buffer: []byte("j"), ContentType: "image/jpeg"})
		server.CurrentSessionID = "sess"

		_, err := serverToolCall(t, server, "gkill_get_idf_file", obj("rep_name", "repo", "file_name", "clip.mp4", "thumb", "400x400", "is_video", true))
		expectNoError(t, err)

		expectFetchCalledWith(t, client, "/files/repo/clip.mp4?is_video=true&thumb=400x400", "sess")
	})

	t.Run("omits the query entirely when no thumb is asked for", func(t *testing.T) {
		client, server := setup()
		client.mockFetchFile(&FileResponse{Buffer: []byte("x"), ContentType: "image/png"})
		server.CurrentSessionID = "sess"

		result, err := serverToolCall(t, server, "gkill_get_idf_file", obj("rep_name", "repo", "file_name", "photo.png"))
		expectNoError(t, err)

		expectFetchCalledWith(t, client, "/files/repo/photo.png", "sess")
		expectTrue(t, !result.Defined("thumb"), "thumb echoed")
	})
}

// ---------------------------------------------------------------------------
// file_path exposure (local stdio clients only)
// ---------------------------------------------------------------------------
func TestFilePathExposure(t *testing.T) {
	setup := func() *Server { return NewReadServer(createMockClient(), nil) }

	t.Run("defaults to non-local so a transport that forgets to opt in never leaks paths", func(t *testing.T) {
		server := setup()
		expectEqual(t, server.IsLocalTransport, false)
	})

	t.Run("buildToolResult keeps file_path in kyou payloads for local clients", func(t *testing.T) {
		server := setup()
		server.IsLocalTransport = true
		payload := obj("kyous", arr(obj("data_type", "idf", "payload", obj("kind", "idf", "file_path", "$HOME/gkill/photo.png"))))

		result := server.BuildToolResult("gkill_get_kyous", payload, false, nil)

		expectEqual(t, objAt(t, arrAt(t, result, "structuredContent", "kyous")[0], "payload").Value("file_path"), "$HOME/gkill/photo.png")
	})

	t.Run("buildToolResult strips file_path from kyou payloads for remote clients", func(t *testing.T) {
		server := setup()
		server.IsLocalTransport = false
		payload := obj("kyous", arr(obj("data_type", "idf", "payload", obj("kind", "idf", "file_path", "$HOME/gkill/photo.png"))))

		result := server.BuildToolResult("gkill_get_kyous", payload, false, nil)

		expectTrue(t, !objAt(t, arrAt(t, result, "structuredContent", "kyous")[0], "payload").Defined("file_path"), "file_path kept")
		mustNotContain(t, firstText(t, result), "$HOME/gkill/photo.png")
	})
}

// ---------------------------------------------------------------------------
// warnings / partial の1行要約への昇格 (2026-08-30 レビュー P1)
// ---------------------------------------------------------------------------
func TestWarningsPartialElevationIntoTheOneLineSummary(t *testing.T) {
	server := NewReadServer(createMockClient(), nil)

	// content[0].text は `${summary}\n\n${JSON}` で、JSON 側には warnings が常に入る。
	// 要約行だけを検査するため1行目を切り出す。
	summaryLine := func(t *testing.T, result *jsonobj.Object) string {
		t.Helper()
		return strings.Split(firstText(t, result), "\n")[0]
	}
	base := func() *jsonobj.Object {
		return obj("kyous", arr(), "total_count", 0, "returned_count", 0, "remaining_count", 0, "has_more", false)
	}

	t.Run("a zero-hit result with a warning does not read as a clean zero", func(t *testing.T) {
		payload := base().Set("warnings", strs(`unknown tag "no-such-tag" in query.tags`))
		line := summaryLine(t, server.BuildToolResult("gkill_get_kyous", payload, false, nil))
		mustContain(t, line, "No entries matched.")
		mustContain(t, line, `WARNING: unknown tag "no-such-tag"`)
	})

	t.Run("partial shows up even without warnings", func(t *testing.T) {
		payload := obj("kyous", arr(obj("id", "k1")), "total_count", 1, "returned_count", 1, "remaining_count", 0, "has_more", false, "partial", true)
		line := summaryLine(t, server.BuildToolResult("gkill_get_kyous", payload, false, nil))
		mustContain(t, line, "PARTIAL:")
	})

	t.Run("a warning shows up even when partial is false (broken-rep count case)", func(t *testing.T) {
		// 壊れた rep の warning があっても、ページング上の打ち切りが無ければ partial は false の
		// ままになりうる (ADR-0216)。partial だけを見て warnings の表示を省略してはいけない。
		payload := obj("kyous", arr(), "total_count", 12345, "returned_count", 0, "remaining_count", 0, "has_more", false,
			"warnings", strs(`repository "BrokenRep" could not be read; results may be incomplete`))
		line := summaryLine(t, server.BuildToolResult("gkill_get_kyous", payload, false, nil))
		mustContain(t, line, "Counted 12345 entries.")
		mustContain(t, line, "WARNING:")
		mustNotContain(t, line, "PARTIAL:")
	})

	t.Run("multiple warnings show the first plus a count", func(t *testing.T) {
		payload := base().Set("warnings", strs("first warning", "second warning", "third warning"))
		line := summaryLine(t, server.BuildToolResult("gkill_get_kyous", payload, false, nil))
		mustContain(t, line, "WARNING: first warning")
		mustContain(t, line, "(+2 more)")
		mustNotContain(t, line, "second warning")
	})

	t.Run("an over-long warning is truncated in the summary (full text stays in warnings[])", func(t *testing.T) {
		longWarning := strings.Repeat("w", 250)
		payload := base().Set("warnings", strs(longWarning))
		result := server.BuildToolResult("gkill_get_kyous", payload, false, nil)
		line := summaryLine(t, result)
		mustContain(t, line, "WARNING:")
		mustContain(t, line, "…")
		expectTrue(t, jsLength(line) < 300, "line length %d", jsLength(line))
		expectEqual(t, arrAt(t, result, "structuredContent", "warnings")[0], longWarning)
	})

	t.Run("a stale-schema warning keeps its dedicated note and is not double-reported", func(t *testing.T) {
		// 古スキーマ警告は AppendStaleSchemaNoteToSummary が専用文言で扱う。
		// 汎用の WARNING: にも載せると同じ指摘が1行に2回並ぶ。
		payload := base().Set("warnings", strs("this client's tool schema snapshot looks stale: data_types arrived as a JSON string"))
		line := summaryLine(t, server.BuildToolResult("gkill_get_kyous", payload, false, nil))
		mustContain(t, line, "tool schema looks stale")
		mustNotContain(t, line, "WARNING:")
	})

	t.Run("a clean result gets no WARNING / PARTIAL suffix", func(t *testing.T) {
		payload := obj("kyous", arr(obj("id", "k1")), "total_count", 1, "returned_count", 1, "remaining_count", 0, "has_more", false)
		line := summaryLine(t, server.BuildToolResult("gkill_get_kyous", payload, false, nil))
		mustNotContain(t, line, "WARNING:")
		mustNotContain(t, line, "PARTIAL:")
	})

	t.Run("partial and a warning appear together, PARTIAL first", func(t *testing.T) {
		// 実運用で最も起きやすい複合 (壊れた rep で欠けつつ、別 rep の付随データも落ちた等)。
		// 並び順まで固定するのは、順序が入れ替わると「PARTIAL の説明が WARNING の続き」に
		// 読めてしまうため。
		payload := obj("kyous", arr(obj("id", "k1")), "total_count", 1, "returned_count", 1, "remaining_count", 0, "has_more", false,
			"partial", true,
			"warnings", strs(`repository "BrokenRep" could not be read; results may be incomplete`))
		line := summaryLine(t, server.BuildToolResult("gkill_get_kyous", payload, false, nil))
		partialIndex := strings.Index(line, "PARTIAL:")
		warningIndex := strings.Index(line, "WARNING:")
		expectTrue(t, partialIndex > -1, "PARTIAL missing")
		expectTrue(t, warningIndex > partialIndex, "WARNING before PARTIAL")
	})

	t.Run("a stale warning mixed with a real one is not counted into (+N more)", func(t *testing.T) {
		// stale はフィルタ後に数えるので、実 warning 1件 + stale 1件で "(+1 more)" が
		// 付いてはいけない (残った実 warning は1件だけ)。数え方を取り違えると件数が嘘になる。
		payload := base().Set("warnings", strs(
			"this client's tool schema snapshot looks stale: count_only arrived as a JSON string",
			`unknown tag "no-such-tag" in query.tags`,
		))
		line := summaryLine(t, server.BuildToolResult("gkill_get_kyous", payload, false, nil))
		mustContain(t, line, `WARNING: unknown tag "no-such-tag"`)
		mustNotContain(t, line, "more)")
		mustContain(t, line, "tool schema looks stale")
	})

	t.Run("the summary warning is cut at exactly 200 characters, not before", func(t *testing.T) {
		// 境界: 200文字ちょうどは切らない / 201文字で切って "…" を付ける。
		exactly200 := strings.Repeat("w", 200)
		over200 := strings.Repeat("w", 201)

		lineAtLimit := summaryLine(t, server.BuildToolResult("gkill_get_kyous", base().Set("warnings", strs(exactly200)), false, nil))
		mustContain(t, lineAtLimit, exactly200)
		mustNotContain(t, lineAtLimit, "…")

		lineOverLimit := summaryLine(t, server.BuildToolResult("gkill_get_kyous", base().Set("warnings", strs(over200)), false, nil))
		mustContain(t, lineOverLimit, exactly200+"…")
		mustNotContain(t, lineOverLimit, over200)
	})
}
