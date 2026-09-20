package mcp

// plugin_tools.go — read / write / readwrite の3サーバが共有する
// プラグイン関連ツールの定義とハンドラ、および gkill_get_kyous のレスポンスへ
// プラグイン本文を埋め込む InlinePluginContents の検査。

import (
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/mcp/jsonobj"
)

const contentHTML = "<html><head><style>body{color:red}</style><script>var a=1;</script></head>" +
	"<body><div class=\"conv-title\">セッション名</div><div class=\"msg\">" +
	"<div class=\"sender\">あなた</div>プラグインの内容取得を実装したい</div></body></html>"

// pluginKyou はプラグイン Kyou を1件持つ get_kyous レスポンス相当の要素を作る。
// 本文取得の鍵（rep_name / id）は Kyou 側にあり、ペイロードには写さない（ADR-0629）。
func pluginKyou(repName, kyouID string, extra ...any) *jsonobj.Object {
	payload := obj("kind", "plugin", "plugin_name", "gkill_plugin_claudecode").Merge(obj(extra...))
	return obj(
		"id", kyouID,
		"rep_name", repName,
		"data_type", "claude_code_message",
		"related_time", "2026-08-05T10:00:00+09:00",
		"payload", payload,
	)
}

func payloadOf(t *testing.T, kyou any) *jsonobj.Object {
	t.Helper()
	return objAt(t, kyou, "payload")
}

// pluginCallMock は vi.fn() 相当。呼び出しを記録し、impl で応答を決める。
type pluginCallMock struct {
	mu    sync.Mutex
	calls []apiCall
	impl  func(pathname string, body *jsonobj.Object) (*jsonobj.Object, error)
}

func (m *pluginCallMock) fn() PluginCallFunc {
	return func(pathname string, body *jsonobj.Object) (*jsonobj.Object, error) {
		m.mu.Lock()
		m.calls = append(m.calls, apiCall{Pathname: pathname, Body: body})
		impl := m.impl
		m.mu.Unlock()
		if impl == nil {
			return obj(), nil
		}
		return impl(pathname, body)
	}
}

func (m *pluginCallMock) callCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.calls)
}

func (m *pluginCallMock) calledWith(pathname string, body *jsonobj.Object) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, call := range m.calls {
		if call.Pathname == pathname && jsonobj.Equal(call.Body, body) {
			return true
		}
	}
	return false
}

func resolvedCall(body string) *pluginCallMock {
	return &pluginCallMock{impl: func(_ string, _ *jsonobj.Object) (*jsonobj.Object, error) {
		return jsonobj.MustUnmarshal(body).(*jsonobj.Object), nil
	}}
}

func rejectedCall(message string) *pluginCallMock {
	return &pluginCallMock{impl: func(_ string, _ *jsonobj.Object) (*jsonobj.Object, error) {
		return nil, errors.New(message)
	}}
}

func htmlResponse(html string) *jsonobj.Object { return obj("html", html, "errors", arr()) }

func inlineStatsObj(s InlineStats) *jsonobj.Object {
	return obj("requested", s.Requested, "inlined", s.Inlined, "truncated", s.Truncated, "skipped", s.Skipped, "errors", s.Errors, "total_text_length", s.TotalTextLength)
}

func pluginPayloadRefsObj(refs []PluginPayloadRef) []any {
	out := []any{}
	for _, ref := range refs {
		out = append(out, obj("rep_name", ref.RepName, "kyou_id", ref.KyouID, "payload", ref.Payload))
	}
	return out
}

// waitUntil は条件が真になるまで（最長 2 秒）待つ。
func waitUntil(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	if !cond() {
		t.Fatalf("condition not met within 2s")
	}
}

// ---------------------------------------------------------------------------
// Tool definitions
// ---------------------------------------------------------------------------
func TestPluginTools(t *testing.T) {
	t.Run("exposes exactly the names listed in PLUGIN_TOOL_NAMES", func(t *testing.T) {
		expectEqual(t, toolNames(PluginTools), PluginToolNames)
	})

	t.Run("every tool has a description and an object inputSchema", func(t *testing.T) {
		for _, tool := range PluginTools {
			description, ok := tool.String("description")
			expectTrue(t, ok, "description is not a string")
			expectTrue(t, len(description) > 20, "description too short")
			schema := objAt(t, tool, "inputSchema")
			expectEqual(t, schema.Value("type"), "object")
			expectEqual(t, schema.Value("additionalProperties"), false)
		}
	})

	t.Run("gkill_get_plugin_list takes only locale_name", func(t *testing.T) {
		tool := findTool(PluginTools, "gkill_get_plugin_list")
		expectEqual(t, objAt(t, tool, "inputSchema", "properties").Keys(), []string{"locale_name"})
		expectTrue(t, !objAt(t, tool, "inputSchema").Has("required"), "required set")
	})

	t.Run("the single-kyou content tool is no longer exposed", func(t *testing.T) {
		expectEqual(t, PluginToolNames, []string{"gkill_get_plugin_list"})
		expectEqual(t, len(PluginTools), 1)
	})

	// The description used to say "filter with query.reps or query.rep_types",
	// which contradicts gkill's own warning ("plugin records are matched via
	// query.reps or data_types, not rep_types") and sends the caller down a path
	// that silently returns nothing.
	t.Run("does not send callers to query.rep_types for plugin records", func(t *testing.T) {
		tool := findTool(PluginTools, "gkill_get_plugin_list")
		mustNotMatch(t, strAt(t, tool, "description"), `query\.reps or query\.rep_types`)
		mustContain(t, strAt(t, tool, "description"), "query.rep_types does NOT work for plugins")
	})

	// 役割の区別（Kyou を出すのか、GPS ログだけなのか）は emits_kyou / provides で表す。
	// capabilities のような3つ目の語彙を作らない（manifest 側の語彙が正本）。
	t.Run("explains emits_kyou / provides and where a non-kyou plugin's data lives", func(t *testing.T) {
		tool := findTool(PluginTools, "gkill_get_plugin_list")
		description := strAt(t, tool, "description")
		mustContain(t, description, "emits_kyou")
		mustContain(t, description, "provides")
		mustContain(t, description, "gkill_get_gps_log")
		mustNotContain(t, description, "capabilities")
	})
}

func TestIsPluginToolName(t *testing.T) {
	t.Run("recognizes plugin tools", func(t *testing.T) {
		expectTrue(t, IsPluginToolName("gkill_get_plugin_list"), "not recognized")
	})

	t.Run("rejects everything else", func(t *testing.T) {
		for _, name := range []string{"gkill_get_plugin_content", "gkill_get_kyous", "gkill_add_kmemo", ""} {
			expectTrue(t, !IsPluginToolName(name), "%q recognized", name)
		}
	})
}

// ---------------------------------------------------------------------------
// handlePluginToolCall
// ---------------------------------------------------------------------------
func TestHandlePluginToolCallGetPluginList(t *testing.T) {
	t.Run("calls the plugin list endpoint and returns plugins", func(t *testing.T) {
		plugins := arr(obj(
			"name", "gkill_plugin_claudecode",
			"version", "1.0.0",
			"description", "Claude Code chat log",
			"data_type", "claude_code_message",
			"rep_name", "Claude Code",
			"is_alive", true,
		))
		call := &pluginCallMock{impl: func(_ string, _ *jsonobj.Object) (*jsonobj.Object, error) {
			return obj("plugins", plugins, "errors", arr()), nil
		}}
		payload, err := HandlePluginToolCall(call.fn(), "gkill_get_plugin_list", obj())
		expectNoError(t, err)
		expectTrue(t, call.calledWith(GetPluginListEndpoint, obj()), "not called with the list endpoint")
		expectEqual(t, payload, obj("plugins", plugins))
	})

	t.Run("passes locale_name through", func(t *testing.T) {
		call := resolvedCall(`{"plugins":[],"errors":[]}`)
		_, err := HandlePluginToolCall(call.fn(), "gkill_get_plugin_list", obj("locale_name", "en"))
		expectNoError(t, err)
		expectTrue(t, call.calledWith(GetPluginListEndpoint, obj("locale_name", "en")), "locale_name not forwarded")
	})

	t.Run("returns an empty array when the server omits plugins", func(t *testing.T) {
		call := resolvedCall(`{"errors":[]}`)
		payload, err := HandlePluginToolCall(call.fn(), "gkill_get_plugin_list", obj())
		expectNoError(t, err)
		expectEqual(t, payload, obj("plugins", arr()))
	})
}

// ---------------------------------------------------------------------------
// 診断文（last_error / typed_index.last_build_error）を AI へ返さない
// 経緯: documents/adr/0707-redact-environment-specific-strings.md
// ---------------------------------------------------------------------------
func TestHandlePluginToolCallWithholdsPluginDiagnostics(t *testing.T) {
	t.Run("drops last_error and reports only that there is one", func(t *testing.T) {
		call := resolvedCall(`{"plugins":[{"name":"p","is_alive":true,"last_error":"gkill: failed to start plugin: ..."}],"errors":[]}`)
		payload, err := HandlePluginToolCall(call.fn(), "gkill_get_plugin_list", obj())
		expectNoError(t, err)
		first := objAt(t, arrAt(t, payload, "plugins")[0])
		expectTrue(t, !first.Defined("last_error"), "last_error kept")
		expectEqual(t, first.Value("has_last_error"), true)
		expectEqual(t, first.Value("name"), "p")
		warnings := arrAt(t, payload, "warnings")
		expectTrue(t, len(warnings) == 1, "%d warnings", len(warnings))
		mustContain(t, jsString(warnings[0]), "withheld")
	})

	t.Run("drops typed_index.last_build_error and keeps the rest of typed_index", func(t *testing.T) {
		call := resolvedCall(`{"plugins":[{"name":"p","typed_index":{"ok":false,"state":"failed","record_count":0,"last_build_error":"index build failed"}}],"errors":[]}`)
		payload, err := HandlePluginToolCall(call.fn(), "gkill_get_plugin_list", obj())
		expectNoError(t, err)
		typedIndex := objAt(t, arrAt(t, payload, "plugins")[0], "typed_index")
		expectTrue(t, !typedIndex.Defined("last_build_error"), "last_build_error kept")
		expectEqual(t, typedIndex.Value("has_last_build_error"), true)
		expectEqual(t, typedIndex.Value("state"), "failed")
		expectEqual(t, typedIndex.Value("record_count"), 0)
		expectTrue(t, len(arrAt(t, payload, "warnings")) == 1, "warnings missing")
	})

	// 落としていないのに警告を出すと常時ノイズになり、本当に効く警告まで読まれなくなる。
	t.Run("adds no warning and no has_* flags when nothing was withheld", func(t *testing.T) {
		call := resolvedCall(`{"plugins":[{"name":"p","is_alive":true,"typed_index":{"ok":true,"state":"ok","record_count":3}}],"errors":[]}`)
		payload, err := HandlePluginToolCall(call.fn(), "gkill_get_plugin_list", obj())
		expectNoError(t, err)
		expectTrue(t, !payload.Defined("warnings"), "warnings set")
		first := objAt(t, arrAt(t, payload, "plugins")[0])
		expectTrue(t, !first.Defined("has_last_error"), "has_last_error set")
		expectTrue(t, !objAt(t, first, "typed_index").Defined("has_last_build_error"), "has_last_build_error set")
	})

	// 空文字の last_error は「何も書かれていない」。has_last_error を立てると
	// 存在しない診断を追いかけさせることになる。
	t.Run("treats an empty last_error as nothing to report", func(t *testing.T) {
		call := resolvedCall(`{"plugins":[{"name":"p","last_error":""}],"errors":[]}`)
		payload, err := HandlePluginToolCall(call.fn(), "gkill_get_plugin_list", obj())
		expectNoError(t, err)
		expectTrue(t, !objAt(t, arrAt(t, payload, "plugins")[0]).Defined("has_last_error"), "has_last_error set")
		expectTrue(t, !payload.Defined("warnings"), "warnings set")
	})
}

func TestSummarizePluginToolPayloadMarksWithheldDiagnostics(t *testing.T) {
	// 本文の warnings を読まない経路でも気づけるようにする。
	t.Run("notes the warning in the one-line summary", func(t *testing.T) {
		summary, _ := SummarizePluginToolPayload("gkill_get_plugin_list", obj(
			"plugins", arr(obj("name", "p", "has_last_error", true)),
			"warnings", strs("plugin diagnostics are withheld from this response"),
		))
		mustContain(t, summary, "Fetched 1 plugins.")
		mustContain(t, summary, "withheld")
	})

	t.Run("stays plain when there is nothing to warn about", func(t *testing.T) {
		summary, _ := SummarizePluginToolPayload("gkill_get_plugin_list", obj("plugins", arr()))
		expectEqual(t, summary, "Fetched 0 plugins.")
	})
}

func TestGetPluginListDescription(t *testing.T) {
	// 「stderr の末尾を読め」と案内したままだと、AI は返ってこない値を待つ。
	t.Run("does not send callers to the raw stderr text", func(t *testing.T) {
		description := strAt(t, findTool(PluginTools, "gkill_get_plugin_list"), "description")
		mustNotContain(t, description, "tail of the plugin process stderr")
		mustContain(t, description, "has_last_error")
		mustContain(t, description, "has_last_build_error")
		mustContain(t, description, "never copy it into documents or commit messages")
	})
}

func TestHandlePluginToolCallUnknownTool(t *testing.T) {
	t.Run("throws for a non-plugin tool name", func(t *testing.T) {
		call := &pluginCallMock{}
		_, err := HandlePluginToolCall(call.fn(), "gkill_get_kyous", obj())
		expectErrorMatches(t, err, `Unknown tool`)
		expectTrue(t, call.callCount() == 0, "call was made")
	})

	t.Run("throws for the removed content tool name", func(t *testing.T) {
		call := &pluginCallMock{}
		_, err := HandlePluginToolCall(call.fn(), "gkill_get_plugin_content", obj())
		expectErrorMatches(t, err, `Unknown tool`)
		expectTrue(t, call.callCount() == 0, "call was made")
	})
}

// ---------------------------------------------------------------------------
// collectPluginPayloads
// ---------------------------------------------------------------------------
func TestCollectPluginPayloads(t *testing.T) {
	t.Run("collects plugin payloads in kyou order", func(t *testing.T) {
		kyous := arr(pluginKyou("A", "1"), pluginKyou("B", "2"))
		ids := []string{}
		for _, ref := range CollectPluginPayloads(kyous) {
			ids = append(ids, ref.KyouID)
		}
		expectEqual(t, ids, []string{"1", "2"})
	})

	t.Run("ignores non-plugin payloads", func(t *testing.T) {
		kyous := arr(
			obj("data_type", "kmemo", "payload", obj("kind", "kmemo", "content", "x")),
			obj("data_type", "idf", "payload", obj("kind", "idf", "rep_name", "Files", "file_name", "a.png")),
			pluginKyou("A", "1"),
		)
		expectEqual(t, len(CollectPluginPayloads(kyous)), 1)
	})

	t.Run("ignores kyous without a usable plugin payload", func(t *testing.T) {
		kyous := arr(
			nil,
			"not an object",
			obj(),
			obj("payload", nil),
			obj("rep_name", "", "id", "1", "payload", obj("kind", "plugin")),
			obj("rep_name", "A", "payload", obj("kind", "plugin")),
		)
		expectEqual(t, pluginPayloadRefsObj(CollectPluginPayloads(kyous)), arr())
	})

	t.Run("takes rep_name and kyou_id from the kyou, not from the payload", func(t *testing.T) {
		payload := obj("kind", "plugin", "rep_name", "stale", "kyou_id", "stale")
		kyous := arr(obj("rep_name", "A", "id", "1", "payload", payload))
		expectEqual(t, pluginPayloadRefsObj(CollectPluginPayloads(kyous)), arr(obj("rep_name", "A", "kyou_id", "1", "payload", payload)))
	})

	t.Run("returns an empty array for a non-array input", func(t *testing.T) {
		expectEqual(t, pluginPayloadRefsObj(CollectPluginPayloads(jsonobj.Undefined)), arr())
		expectEqual(t, pluginPayloadRefsObj(CollectPluginPayloads(nil)), arr())
		expectEqual(t, pluginPayloadRefsObj(CollectPluginPayloads(obj())), arr())
	})
}

// ---------------------------------------------------------------------------
// runGroupedWithConcurrency
// ---------------------------------------------------------------------------
func TestRunGroupedWithConcurrency(t *testing.T) {
	t.Run("runs items of the same key strictly one at a time", func(t *testing.T) {
		var inFlight, peak int64
		entries := []GroupedEntry{{"same", 1}, {"same", 2}, {"same", 3}}
		RunGroupedWithConcurrency(entries, 4, func(_ any) (bool, error) {
			current := atomic.AddInt64(&inFlight, 1)
			for {
				old := atomic.LoadInt64(&peak)
				if current <= old || atomic.CompareAndSwapInt64(&peak, old, current) {
					break
				}
			}
			time.Sleep(time.Millisecond)
			atomic.AddInt64(&inFlight, -1)
			return true, nil
		})
		expectEqual(t, atomic.LoadInt64(&peak), 1)
	})

	t.Run("preserves item order within a key", func(t *testing.T) {
		seen := []string{}
		entries := []GroupedEntry{{"k", "a"}, {"k", "b"}, {"k", "c"}}
		RunGroupedWithConcurrency(entries, 2, func(item any) (bool, error) {
			seen = append(seen, item.(string))
			return true, nil
		})
		expectEqual(t, seen, []string{"a", "b", "c"})
	})

	t.Run("runs different keys concurrently", func(t *testing.T) {
		var inFlight, peak int64
		gates := []chan struct{}{make(chan struct{}), make(chan struct{})}
		entries := []GroupedEntry{{"x", 0}, {"y", 1}}
		done := make(chan struct{})
		go func() {
			RunGroupedWithConcurrency(entries, 2, func(item any) (bool, error) {
				current := atomic.AddInt64(&inFlight, 1)
				for {
					old := atomic.LoadInt64(&peak)
					if current <= old || atomic.CompareAndSwapInt64(&peak, old, current) {
						break
					}
				}
				<-gates[item.(int)]
				atomic.AddInt64(&inFlight, -1)
				return true, nil
			})
			close(done)
		}()
		waitUntil(t, func() bool { return atomic.LoadInt64(&inFlight) == 2 })
		close(gates[0])
		close(gates[1])
		<-done
		expectEqual(t, atomic.LoadInt64(&peak), 2)
	})

	t.Run("never exceeds the concurrency limit", func(t *testing.T) {
		var inFlight, peak int64
		entries := []GroupedEntry{}
		for _, key := range []string{"a", "b", "c", "d", "e"} {
			entries = append(entries, GroupedEntry{key, key})
		}
		RunGroupedWithConcurrency(entries, 2, func(_ any) (bool, error) {
			current := atomic.AddInt64(&inFlight, 1)
			for {
				old := atomic.LoadInt64(&peak)
				if current <= old || atomic.CompareAndSwapInt64(&peak, old, current) {
					break
				}
			}
			time.Sleep(time.Millisecond)
			atomic.AddInt64(&inFlight, -1)
			return true, nil
		})
		expectTrue(t, atomic.LoadInt64(&peak) <= 2, "peak %d", peak)
	})

	t.Run("stops only the affected key when the worker returns false", func(t *testing.T) {
		var mu sync.Mutex
		seen := NewStringSet()
		entries := []GroupedEntry{{"x", "x1"}, {"x", "x2"}, {"y", "y1"}}
		RunGroupedWithConcurrency(entries, 2, func(item any) (bool, error) {
			mu.Lock()
			seen.Add(item.(string))
			mu.Unlock()
			return item != "x1", nil
		})
		expectTrue(t, seen.Has("x1") && seen.Has("y1"), "x1 / y1 not seen")
		expectTrue(t, !seen.Has("x2"), "x2 was run")
	})

	t.Run("swallows a worker exception and stops only that key", func(t *testing.T) {
		var mu sync.Mutex
		seen := NewStringSet()
		entries := []GroupedEntry{{"x", "x1"}, {"x", "x2"}, {"y", "y1"}}
		RunGroupedWithConcurrency(entries, 2, func(item any) (bool, error) {
			mu.Lock()
			seen.Add(item.(string))
			mu.Unlock()
			if item == "x1" {
				return false, errors.New("boom")
			}
			return true, nil
		})
		expectTrue(t, seen.Has("y1"), "y1 not seen")
		expectTrue(t, !seen.Has("x2"), "x2 was run")
	})

	t.Run("resolves immediately for an empty entry list", func(t *testing.T) {
		called := false
		RunGroupedWithConcurrency([]GroupedEntry{}, 4, func(_ any) (bool, error) {
			called = true
			return true, nil
		})
		expectTrue(t, !called, "worker was called")
	})
}

// ---------------------------------------------------------------------------
// inlinePluginContents
// ---------------------------------------------------------------------------
func TestInlinePluginContents(t *testing.T) {
	t.Run("issues no request and reports zeroes when there is no plugin kyou", func(t *testing.T) {
		call := &pluginCallMock{}
		stats := InlinePluginContents(call.fn(), arr(obj("data_type", "kmemo", "payload", obj("kind", "kmemo"))), InlineOptions{})
		expectTrue(t, call.callCount() == 0, "call was made")
		expectEqual(t, inlineStatsObj(stats), obj(
			"requested", 0,
			"inlined", 0,
			"truncated", 0,
			"skipped", 0,
			"errors", 0,
			"total_text_length", 0,
		))
	})

	t.Run("embeds converted text and marks the payload ok", func(t *testing.T) {
		call := &pluginCallMock{impl: func(_ string, _ *jsonobj.Object) (*jsonobj.Object, error) { return htmlResponse(contentHTML), nil }}
		kyous := arr(pluginKyou("Claude Code", "abc"))
		stats := InlinePluginContents(call.fn(), kyous, InlineOptions{})
		p := payloadOf(t, kyous[0])
		mustContain(t, strAt(t, p, "content_text"), "セッション名")
		mustNotContain(t, strAt(t, p, "content_text"), "color:red")
		expectEqual(t, p.Value("content_status"), "ok")
		expectEqual(t, stats.Inlined, 1)
		expectEqual(t, stats.TotalTextLength, jsLength(strAt(t, p, "content_text")))
	})

	t.Run("posts the content endpoint once per plugin kyou with only rep_name and kyou_id", func(t *testing.T) {
		call := &pluginCallMock{impl: func(_ string, _ *jsonobj.Object) (*jsonobj.Object, error) { return htmlResponse(contentHTML), nil }}
		kyous := arr(pluginKyou("A", "1"), pluginKyou("B", "2"))
		InlinePluginContents(call.fn(), kyous, InlineOptions{})
		expectEqual(t, call.callCount(), 2)
		expectTrue(t, call.calledWith(GetPluginContentHTMLEndpoint, obj("rep_name", "A", "kyou_id", "1")), "A/1 not requested")
		expectTrue(t, call.calledWith(GetPluginContentHTMLEndpoint, obj("rep_name", "B", "kyou_id", "2")), "B/2 not requested")
	})

	t.Run("never passes an abort signal to the call", func(t *testing.T) {
		call := &pluginCallMock{impl: func(_ string, _ *jsonobj.Object) (*jsonobj.Object, error) { return htmlResponse(contentHTML), nil }}
		InlinePluginContents(call.fn(), arr(pluginKyou("A", "1")), InlineOptions{})
		expectEqual(t, sortedCopy(call.calls[0].Body.Keys()), []string{"kyou_id", "rep_name"})
	})

	t.Run("forwards locale_name only when given", func(t *testing.T) {
		call := &pluginCallMock{impl: func(_ string, _ *jsonobj.Object) (*jsonobj.Object, error) { return htmlResponse(contentHTML), nil }}
		InlinePluginContents(call.fn(), arr(pluginKyou("A", "1")), InlineOptions{LocaleName: "en"})
		expectTrue(t, call.calledWith(GetPluginContentHTMLEndpoint, obj("rep_name", "A", "kyou_id", "1", "locale_name", "en")), "locale_name not forwarded")
	})

	t.Run("leaves non-plugin payloads untouched", func(t *testing.T) {
		call := &pluginCallMock{impl: func(_ string, _ *jsonobj.Object) (*jsonobj.Object, error) { return htmlResponse(contentHTML), nil }}
		kmemo := obj("data_type", "kmemo", "payload", obj("kind", "kmemo", "content", "hello"))
		kyous := arr(kmemo, pluginKyou("A", "1"))
		InlinePluginContents(call.fn(), kyous, InlineOptions{})
		expectEqual(t, kmemo.Value("payload"), obj("kind", "kmemo", "content", "hello"))
	})

	t.Run("never adds a file_name field to a plugin payload", func(t *testing.T) {
		// file_name を持たせると IsIdfPayload が IDF と誤認し、ApplyFileLinks が
		// 無意味なファイルトークンを発行してしまう。
		call := &pluginCallMock{impl: func(_ string, _ *jsonobj.Object) (*jsonobj.Object, error) { return htmlResponse(contentHTML), nil }}
		kyous := arr(pluginKyou("A", "1"))
		InlinePluginContents(call.fn(), kyous, InlineOptions{})
		expectTrue(t, !payloadOf(t, kyous[0]).Has("file_name"), "file_name added")
		expectTrue(t, !payloadOf(t, kyous[0]).Has("file_path"), "file_path added")
	})

	t.Run("marks the payload truncated when the body exceeds maxTextLength", func(t *testing.T) {
		call := &pluginCallMock{impl: func(_ string, _ *jsonobj.Object) (*jsonobj.Object, error) { return htmlResponse(contentHTML), nil }}
		kyous := arr(pluginKyou("A", "1"))
		stats := InlinePluginContents(call.fn(), kyous, InlineOptions{MaxTextLength: 5})
		p := payloadOf(t, kyous[0])
		expectEqual(t, p.Value("content_status"), "truncated")
		expectTrue(t, strings.HasPrefix(strAt(t, p, "content_text"), "セッション"), "text %q", strAt(t, p, "content_text"))
		expectEqual(t, stats.Truncated, 1)
	})

	t.Run("returns raw html when format is html", func(t *testing.T) {
		call := &pluginCallMock{impl: func(_ string, _ *jsonobj.Object) (*jsonobj.Object, error) { return htmlResponse(contentHTML), nil }}
		kyous := arr(pluginKyou("A", "1"))
		InlinePluginContents(call.fn(), kyous, InlineOptions{Format: "html"})
		p := payloadOf(t, kyous[0])
		expectEqual(t, p.Value("content_html"), contentHTML)
		expectTrue(t, !p.Defined("content_text"), "content_text set")
	})

	t.Run("returns both text and html when format is both", func(t *testing.T) {
		call := &pluginCallMock{impl: func(_ string, _ *jsonobj.Object) (*jsonobj.Object, error) { return htmlResponse(contentHTML), nil }}
		kyous := arr(pluginKyou("A", "1"))
		InlinePluginContents(call.fn(), kyous, InlineOptions{Format: "both"})
		p := payloadOf(t, kyous[0])
		expectEqual(t, p.Value("content_html"), contentHTML)
		_, isString := p.String("content_text")
		expectTrue(t, isString, "content_text is not a string")
	})

	t.Run("clips an oversized html before converting and marks it truncated", func(t *testing.T) {
		huge := "<div>" + strings.Repeat("あ", 1000) + "</div>"
		call := &pluginCallMock{impl: func(_ string, _ *jsonobj.Object) (*jsonobj.Object, error) { return htmlResponse(huge), nil }}
		kyous := arr(pluginKyou("A", "1"))
		InlinePluginContents(call.fn(), kyous, InlineOptions{MaxHTMLLength: 50, MaxTextLength: 100000})
		p := payloadOf(t, kyous[0])
		expectEqual(t, p.Value("content_status"), "truncated")
		expectTrue(t, jsLength(strAt(t, p, "content_text")) < 60, "text length %d", jsLength(strAt(t, p, "content_text")))
	})

	t.Run("treats a missing html field as empty content", func(t *testing.T) {
		call := resolvedCall(`{"errors":[]}`)
		kyous := arr(pluginKyou("A", "1"))
		stats := InlinePluginContents(call.fn(), kyous, InlineOptions{})
		p := payloadOf(t, kyous[0])
		expectEqual(t, p.Value("content_text"), "")
		expectEqual(t, p.Value("content_status"), "ok")
		expectEqual(t, stats.Inlined, 1)
	})

	t.Run("fetches a repeated rep_name/kyou_id pair only once and fills both payloads", func(t *testing.T) {
		call := &pluginCallMock{impl: func(_ string, _ *jsonobj.Object) (*jsonobj.Object, error) { return htmlResponse(contentHTML), nil }}
		kyous := arr(pluginKyou("A", "1"), pluginKyou("A", "1"))
		stats := InlinePluginContents(call.fn(), kyous, InlineOptions{})
		expectEqual(t, call.callCount(), 1)
		expectEqual(t, payloadOf(t, kyous[0]).Value("content_status"), "ok")
		expectEqual(t, payloadOf(t, kyous[1]).Value("content_status"), "ok")
		expectEqual(t, stats.Inlined, 2)
	})

	t.Run("isolates a failing rep and still resolves", func(t *testing.T) {
		call := &pluginCallMock{impl: func(_ string, body *jsonobj.Object) (*jsonobj.Object, error) {
			if body.Value("rep_name") == "A" {
				return nil, errors.New("plugin not found")
			}
			return obj("html", contentHTML), nil
		}}
		kyous := arr(pluginKyou("A", "1"), pluginKyou("B", "2"))
		stats := InlinePluginContents(call.fn(), kyous, InlineOptions{})
		expectEqual(t, payloadOf(t, kyous[0]).Value("content_status"), "error")
		expectEqual(t, payloadOf(t, kyous[0]).Value("content_error"), "plugin not found")
		expectEqual(t, payloadOf(t, kyous[1]).Value("content_status"), "ok")
		expectEqual(t, stats.Errors, 1)
		expectEqual(t, stats.Inlined, 1)
	})

	t.Run("stops requesting the rest of a rep after its first failure", func(t *testing.T) {
		call := rejectedCall("dead plugin")
		kyous := arr(pluginKyou("A", "1"), pluginKyou("A", "2"), pluginKyou("A", "3"))
		stats := InlinePluginContents(call.fn(), kyous, InlineOptions{})
		expectEqual(t, call.callCount(), 1)
		expectEqual(t, payloadOf(t, kyous[0]).Value("content_status"), "error")
		expectEqual(t, payloadOf(t, kyous[1]).Value("content_skipped_reason"), "rep_error")
		expectEqual(t, payloadOf(t, kyous[2]).Value("content_skipped_reason"), "rep_error")
		expectEqual(t, stats.Errors, 1)
		expectEqual(t, stats.Skipped, 2)
	})

	t.Run("truncates a long error message", func(t *testing.T) {
		call := rejectedCall(strings.Repeat("x", 500))
		kyous := arr(pluginKyou("A", "1"))
		InlinePluginContents(call.fn(), kyous, InlineOptions{})
		expectEqual(t, jsLength(strAt(t, payloadOf(t, kyous[0]), "content_error")), 200)
	})

	t.Run("skips entries beyond maxKyous without issuing a request", func(t *testing.T) {
		call := &pluginCallMock{impl: func(_ string, _ *jsonobj.Object) (*jsonobj.Object, error) { return htmlResponse(contentHTML), nil }}
		kyous := arr(pluginKyou("A", "1"), pluginKyou("A", "2"), pluginKyou("A", "3"))
		stats := InlinePluginContents(call.fn(), kyous, InlineOptions{MaxKyous: 2})
		expectEqual(t, call.callCount(), 2)
		expectEqual(t, payloadOf(t, kyous[2]).Value("content_status"), "skipped")
		expectEqual(t, payloadOf(t, kyous[2]).Value("content_skipped_reason"), "max_kyous")
		expectEqual(t, stats.Skipped, 1)
	})

	t.Run("applies the total budget in kyou order regardless of completion order", func(t *testing.T) {
		slow := make(chan struct{})
		var bCalled int64
		call := &pluginCallMock{impl: func(_ string, body *jsonobj.Object) (*jsonobj.Object, error) {
			if body.Value("kyou_id") == "1" {
				<-slow
				return obj("html", contentHTML), nil
			}
			atomic.StoreInt64(&bCalled, 1)
			return obj("html", contentHTML), nil
		}}
		kyous := arr(pluginKyou("A", "1"), pluginKyou("B", "2"))
		var stats InlineStats
		done := make(chan struct{})
		go func() {
			stats = InlinePluginContents(call.fn(), kyous, InlineOptions{TotalTextLength: 10})
			close(done)
		}()
		waitUntil(t, func() bool { return atomic.LoadInt64(&bCalled) == 1 })
		close(slow)
		<-done
		// 先頭は予算に関係なく必ず載る。2件目は予算超過で skipped。
		expectEqual(t, payloadOf(t, kyous[0]).Value("content_status"), "ok")
		expectEqual(t, payloadOf(t, kyous[1]).Value("content_status"), "skipped")
		expectEqual(t, payloadOf(t, kyous[1]).Value("content_skipped_reason"), "budget")
		expectEqual(t, stats.Inlined, 1)
	})

	t.Run("stops starting requests once the deadline passed", func(t *testing.T) {
		var clock int64
		base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		call := &pluginCallMock{impl: func(_ string, _ *jsonobj.Object) (*jsonobj.Object, error) {
			atomic.AddInt64(&clock, 1000)
			return obj("html", contentHTML), nil
		}}
		kyous := arr(pluginKyou("A", "1"), pluginKyou("A", "2"))
		stats := InlinePluginContents(call.fn(), kyous, InlineOptions{
			Deadline: 100 * time.Millisecond,
			Now:      func() time.Time { return base.Add(time.Duration(atomic.LoadInt64(&clock)) * time.Millisecond) },
		})
		expectEqual(t, call.callCount(), 1)
		expectEqual(t, payloadOf(t, kyous[1]).Value("content_status"), "skipped")
		expectEqual(t, payloadOf(t, kyous[1]).Value("content_skipped_reason"), "deadline")
		expectEqual(t, stats.Skipped, 1)
	})

	t.Run("serializes calls within a rep and parallelizes across reps", func(t *testing.T) {
		var mu sync.Mutex
		inFlight := map[string]int{}
		peak := map[string]int{}
		gates := map[string]chan struct{}{}
		call := &pluginCallMock{impl: func(_ string, body *jsonobj.Object) (*jsonobj.Object, error) {
			rep := jsString(body.Value("rep_name"))
			mu.Lock()
			inFlight[rep]++
			if inFlight[rep] > peak[rep] {
				peak[rep] = inFlight[rep]
			}
			gate := make(chan struct{})
			gates[rep+jsString(body.Value("kyou_id"))] = gate
			mu.Unlock()
			<-gate
			mu.Lock()
			inFlight[rep]--
			mu.Unlock()
			return obj("html", contentHTML), nil
		}}
		gateOf := func(key string) chan struct{} {
			mu.Lock()
			defer mu.Unlock()
			return gates[key]
		}
		kyous := arr(pluginKyou("A", "1"), pluginKyou("A", "2"), pluginKyou("B", "1"))
		done := make(chan struct{})
		go func() {
			InlinePluginContents(call.fn(), kyous, InlineOptions{Concurrency: 4})
			close(done)
		}()
		waitUntil(t, func() bool { return call.callCount() == 2 }) // A/1 と B/1 が同時に飛ぶ
		expectEqual(t, call.callCount(), 2)
		close(gateOf("A1"))
		close(gateOf("B1"))
		waitUntil(t, func() bool { return gateOf("A2") != nil })
		close(gateOf("A2"))
		<-done
		mu.Lock()
		expectEqual(t, peak["A"], 1)
		mu.Unlock()
		expectEqual(t, call.callCount(), 3)
	})

	t.Run("keeps the stats arithmetic consistent across a mixed run", func(t *testing.T) {
		call := &pluginCallMock{impl: func(_ string, body *jsonobj.Object) (*jsonobj.Object, error) {
			if body.Value("rep_name") == "B" {
				return nil, errors.New("nope")
			}
			return obj("html", contentHTML), nil
		}}
		kyous := arr(pluginKyou("A", "1"), pluginKyou("B", "2"), pluginKyou("C", "3"), pluginKyou("C", "4"))
		stats := InlinePluginContents(call.fn(), kyous, InlineOptions{MaxKyous: 3})
		expectEqual(t, stats.Requested, 4)
		expectEqual(t, stats.Inlined+stats.Skipped+stats.Errors, stats.Requested)
	})

	t.Run("resolves even when every rep fails", func(t *testing.T) {
		call := rejectedCall("all dead")
		kyous := arr(pluginKyou("A", "1"), pluginKyou("B", "2"))
		stats := InlinePluginContents(call.fn(), kyous, InlineOptions{})
		expectEqual(t, stats.Errors, 2)
		expectEqual(t, stats.Inlined, 0)
	})
}

// ---------------------------------------------------------------------------
// summarize
// ---------------------------------------------------------------------------
func TestSummarizePluginToolPayload(t *testing.T) {
	t.Run("summarizes the plugin list", func(t *testing.T) {
		summary, _ := SummarizePluginToolPayload("gkill_get_plugin_list", obj("plugins", arr(obj(), obj())))
		expectEqual(t, summary, "Fetched 2 plugins.")
		summary, _ = SummarizePluginToolPayload("gkill_get_plugin_list", obj())
		expectEqual(t, summary, "Fetched 0 plugins.")
	})

	t.Run("returns null for non-plugin tools so callers can fall back", func(t *testing.T) {
		_, ok := SummarizePluginToolPayload("gkill_get_kyous", obj())
		expectTrue(t, !ok, "summarized gkill_get_kyous")
		_, ok = SummarizePluginToolPayload("gkill_get_plugin_content", obj())
		expectTrue(t, !ok, "summarized gkill_get_plugin_content")
	})
}

func TestSummarizeInlinePluginContent(t *testing.T) {
	t.Run("returns an empty string when nothing was inlined", func(t *testing.T) {
		expectEqual(t, SummarizeInlinePluginContent(nil), "")
		expectEqual(t, SummarizeInlinePluginContent(&InlineStats{}), "")
	})

	t.Run("reports the embedded count", func(t *testing.T) {
		expectEqual(t, SummarizeInlinePluginContent(&InlineStats{Requested: 3, Inlined: 3}), " Embedded plugin content for 3 of 3 plugin kyous.")
	})

	t.Run("reports truncated, skipped and failed counts", func(t *testing.T) {
		summary := SummarizeInlinePluginContent(&InlineStats{Requested: 6, Inlined: 3, Truncated: 1, Skipped: 2, Errors: 1})
		expectEqual(t, summary, " Embedded plugin content for 3 of 6 plugin kyous (1 truncated, 2 not fetched, 1 failed).")
	})
}

func TestGetPluginListPluginsWithoutIngestCount(t *testing.T) {
	// provides を宣言していないプラグインには typed_index が付かない。すると
	// is_alive:true / process_running:true のまま1件も取り込めていない状態が、
	// 一覧の上では完全に正常に見える（実測 2026-08-25）。
	t.Run("emits_kyou なのに typed_index が無いものを名指しする", func(t *testing.T) {
		call := resolvedCall(`{"plugins":[
			{"name":"p1","data_type":"claude_conversation","emits_kyou":true},
			{"name":"p2","data_type":"kc","emits_kyou":true,"typed_index":{"state":"ok","record_count":22}}
		]}`)
		payload, err := HandlePluginToolCall(call.fn(), "gkill_get_plugin_list", obj())
		expectNoError(t, err)
		joined := []string{}
		if warnings, ok := payload.Array("warnings"); ok {
			for _, w := range warnings {
				joined = append(joined, jsString(w))
			}
		}
		text := strings.Join(joined, "\n")
		mustContain(t, text, "claude_conversation")
		mustNotContain(t, text, `"kc"`)
		mustContain(t, text, "count_only")
	})

	// 記録を出さないプラグイン（GPS ログ専用など）は対象外。
	t.Run("emits_kyou:false は名指ししない", func(t *testing.T) {
		call := resolvedCall(`{"plugins":[{"name":"p3","data_type":"google_location_visit","emits_kyou":false}]}`)
		payload, err := HandlePluginToolCall(call.fn(), "gkill_get_plugin_list", obj())
		expectNoError(t, err)
		expectTrue(t, !payload.Defined("warnings"), "warnings set")
	})

	t.Run("全部が索引を持つなら警告なし", func(t *testing.T) {
		call := resolvedCall(`{"plugins":[{"name":"p2","data_type":"kc","emits_kyou":true,"typed_index":{"state":"ok"}}]}`)
		payload, err := HandlePluginToolCall(call.fn(), "gkill_get_plugin_list", obj())
		expectNoError(t, err)
		expectTrue(t, !payload.Defined("warnings"), "warnings set")
	})
}
