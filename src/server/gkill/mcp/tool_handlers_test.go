package mcp

// MCP 読み取りツール定義と共有ヘルパの検査。
//
// v2 から summarize / ディスパッチは read_handlers.go に一本化され公開されたので、
// ここは再実装のミラーではなく**実物を直接使って**検証する
// （ミラーは実装が変わっても緑のまま古び、二重管理の温床だった）。

import (
	"regexp"
	"testing"

	"github.com/mt3hr/gkill/src/server/gkill/mcp/jsonobj"
)

// ---------------------------------------------------------------------------
// Tool definition presence
// ---------------------------------------------------------------------------
func TestReadToolDefinitions(t *testing.T) {
	t.Run("read server exposes 12 tools (11 read + 1 plugin)", func(t *testing.T) {
		expectEqual(t, len(ReadTools), 11)
		expectEqual(t, len(PluginTools), 1)
	})

	t.Run("read tool names are the v2 set", func(t *testing.T) {
		expectEqual(t, toolNames(ReadTools), []string{
			// gkill_status は先頭（接続先と一覧の世代を最初に見せる）
			"gkill_status",
			// 説明文の本文（ADR-0622）。status の次に置き、検索より先に目に入るようにする
			"gkill_get_mcp_help",
			"gkill_get_kyous",
			"gkill_get_mi_board_list",
			"gkill_get_all_tag_names",
			"gkill_get_all_rep_names",
			"gkill_get_gps_log",
			"gkill_get_application_config",
			"gkill_get_rep_infos",
			"gkill_get_idf_file",
			"gkill_get_kyou_history",
		})
	})

	t.Run("isReadToolName matches the definitions", func(t *testing.T) {
		for _, tool := range ReadTools {
			expectTrue(t, IsReadToolName(strAt(t, tool, "name")), "%s is not a read tool", strAt(t, tool, "name"))
		}
		expectTrue(t, !IsReadToolName("gkill_get_plugin_list"), "plugin tool counted as read")
		expectTrue(t, !IsReadToolName("gkill_add_kmemo"), "write tool counted as read")
	})

	t.Run("every tool has an object inputSchema with additionalProperties: false", func(t *testing.T) {
		for _, tool := range ReadTools {
			schema := objAt(t, tool, "inputSchema")
			expectEqual(t, schema.Value("type"), "object")
			expectEqual(t, schema.Value("additionalProperties"), false)
		}
	})

	t.Run("gkill_get_kyous tells AI to inspect repository warnings independently of partial", func(t *testing.T) {
		description := strAt(t, findTool(ReadTools, "gkill_get_kyous"), "description")
		mustContain(t, description, "even when partial is false")
		mustContain(t, description, "failed to load")
		mustContain(t, description, "do not put a repository named by that warning back into query.reps")
	})

	// 説明文は要約で、本文は gkill_get_mcp_help（ADR-0622）。要約が本文の在処を案内していなければ、
	// 移した知識は誰にも読まれない。
	t.Run("summarized descriptions point at gkill_get_mcp_help", func(t *testing.T) {
		for _, name := range []string{"gkill_get_kyous", "gkill_get_rep_infos", "gkill_get_idf_file"} {
			mustContain(t, strAt(t, findTool(ReadTools, name), "description"), "gkill_get_mcp_help")
		}
		kftl := strAt(t, findTool(WriteTools, "gkill_submit_kftl"), "description")
		mustContain(t, kftl, "gkill_get_mcp_help topic:kftl")
	})
}

// ---------------------------------------------------------------------------
// summarizeReadToolPayload (v2)
// ---------------------------------------------------------------------------
func TestSummarizeReadToolPayload(t *testing.T) {
	summarize := func(t *testing.T, name string, payload *jsonobj.Object) string {
		t.Helper()
		summary, ok := SummarizeReadToolPayload(name, payload)
		expectTrue(t, ok, "%s: no summary", name)
		return summary
	}

	t.Run("gkill_get_kyous — first page carries total and remaining", func(t *testing.T) {
		result := summarize(t, "gkill_get_kyous", obj(
			"kyous", arr(obj()),
			"returned_count", 20,
			"total_count", 50,
			"remaining_count", 30,
			"has_more", true,
			"next_cursor", "2026-01-01T00:00:00+09:00::abc",
		))
		mustContain(t, result, "Returned 20 of 50")
		mustContain(t, result, "30 remaining")
		mustContain(t, result, "2026-01-01T00:00:00+09:00::abc")
	})

	t.Run("gkill_get_kyous — cursor page has no total_count and must not claim completion", func(t *testing.T) {
		// 旧実装は total_count ?? returned_count で「all results returned」と
		// 嘘の完了報告をしていた（v2 では total_count は cursor 無し応答のみ）
		result := summarize(t, "gkill_get_kyous", obj(
			"kyous", arr(obj()),
			"returned_count", 20,
			"remaining_count", 5,
			"has_more", true,
			"next_cursor", "cursor",
		))
		mustContain(t, result, "Returned 20 kyou entries (5 remaining)")
		mustNotContain(t, result, "all results returned")
	})

	t.Run("gkill_get_kyous — count_only", func(t *testing.T) {
		result := summarize(t, "gkill_get_kyous", obj(
			"kyous", arr(),
			"returned_count", 0,
			"remaining_count", 0,
			"total_count", 124056,
			"has_more", false,
		))
		expectEqual(t, result, "Counted 124056 entries.")
	})

	t.Run("gkill_get_kyous — group_by buckets", func(t *testing.T) {
		result := summarize(t, "gkill_get_kyous", obj(
			"kyous", arr(),
			"buckets", arr(obj("key", "2026-01", "count", 3), obj("key", "2026-02", "count", 4)),
			"total_count", 7,
			"returned_count", 0,
			"remaining_count", 0,
			"has_more", false,
		))
		expectEqual(t, result, "Aggregated 2 buckets (7 entries).")
	})

	t.Run("gkill_get_mi_board_list — counts boards", func(t *testing.T) {
		expectEqual(t, summarize(t, "gkill_get_mi_board_list", obj("boards", strs("a", "b"))), "Fetched 2 Mi boards.")
	})

	t.Run("gkill_get_all_tag_names — counts tags", func(t *testing.T) {
		expectEqual(t, summarize(t, "gkill_get_all_tag_names", obj("tag_names", strs("t1", "t2", "t3"))), "Fetched 3 tag names.")
	})

	t.Run("gkill_get_all_rep_names — counts repos", func(t *testing.T) {
		expectEqual(t, summarize(t, "gkill_get_all_rep_names", obj("rep_names", strs("r"))), "Fetched 1 repository names.")
	})

	t.Run("gkill_get_gps_log — paged points", func(t *testing.T) {
		result := summarize(t, "gkill_get_gps_log", obj(
			"gps_logs", arr(obj(), obj(), obj()),
			"returned_count", 3,
			"remaining_count", 7,
			"has_more", true,
			"next_cursor", "abc",
		))
		mustContain(t, result, "Returned 3 GPS points (7 remaining)")
	})

	t.Run("gkill_get_gps_log — daily buckets", func(t *testing.T) {
		result := summarize(t, "gkill_get_gps_log", obj(
			"gps_logs", arr(),
			"buckets", arr(obj("key", "2026-08-01", "count", 100)),
			"total_count", 100,
			"has_more", false,
		))
		expectEqual(t, result, "Aggregated 1 daily buckets (100 GPS points).")
	})

	t.Run("gkill_get_rep_infos — counts reps and types", func(t *testing.T) {
		result := summarize(t, "gkill_get_rep_infos", obj(
			"rep_infos", arr(obj("rep_name", "Kmemo", "rep_type", "kmemo")),
			"canonical_rep_types", strs("kmemo", "kc"),
			"plugins", arr(),
		))
		expectEqual(t, result, "Fetched 1 repositories, 2 canonical rep types.")
	})

	// fields で rep_infos を外した呼び出しに「Fetched 0 repositories」と言うと、
	// 自分で外しただけなのに「リポジトリが0件」と読める（2026-08-25 の実利用レビュー）。
	t.Run("gkill_get_rep_infos — says a field was omitted rather than reporting zero", func(t *testing.T) {
		result := summarize(t, "gkill_get_rep_infos", obj(
			"canonical_rep_types", strs("kmemo", "kc"),
			"plugins", arr(),
			"attached_data_reps", arr(),
		))
		mustContain(t, result, "omitted by fields")
		mustNotContain(t, result, "0 repositories")
	})

	t.Run("unknown tool — returns null (server falls back)", func(t *testing.T) {
		_, ok := SummarizeReadToolPayload("unknown_tool", obj())
		expectTrue(t, !ok, "unknown tool was summarized")
		_, ok = SummarizeReadToolPayload("gkill_add_kmemo", obj())
		expectTrue(t, !ok, "write tool was summarized by the read summarizer")
	})
}

// ---------------------------------------------------------------------------
// summarizeToolError (payload.go の実物)
// ---------------------------------------------------------------------------
func TestSummarizeToolErrorRead(t *testing.T) {
	t.Run("formats error with tool name", func(t *testing.T) {
		mustContain(t, SummarizeToolError("gkill_get_kyous", "Connection refused", nil), "gkill_get_kyous")
		mustContain(t, SummarizeToolError("gkill_get_kyous", "Connection refused", nil), "Connection refused")
	})

	t.Run("handles empty tool name", func(t *testing.T) {
		mustContain(t, SummarizeToolError("", "Timeout", nil), "Timeout")
	})
}

// ---------------------------------------------------------------------------
// 説明文が名指しするツールは、そのサーバに実在すること
// ---------------------------------------------------------------------------
//
// 読み取り専用サーバの説明が gkill_restore_kyou を案内し、書き込み専用サーバの
// 「見つからない」が gkill_get_application_config / gkill_get_kyous を案内していた。
// どちらも「案内されたツールがそのサーバに無い」で、AI は存在しないツールを探す。
// 同じ穴なので、説明文とエラーメッセージをまとめて機械検査する。

// collectMentions は説明文の中の gkill_* を全部拾う。inputSchema の中の description も見る
// （restore の案内は find_query_schema.go 側、つまりスキーマの奥にあった）。
func collectMentions(node any, found *StringSet) *StringSet {
	if found == nil {
		found = NewStringSet()
	}
	switch v := node.(type) {
	case string:
		for _, match := range regexp.MustCompile(`gkill_[a-z_]+`).FindAllString(v, -1) {
			found.Add(match)
		}
	case *jsonobj.Object:
		for _, key := range v.Keys() {
			collectMentions(v.Value(key), found)
		}
	default:
		if items, ok := jsonobj.AsArray(node); ok {
			for _, child := range items {
				collectMentions(child, found)
			}
		}
	}
	return found
}

func TestToolNamesInDescriptionsExistOnTheSameServer(t *testing.T) {
	writeServerReadTools := filterTools(ReadTools, WriteServerReadToolNames)
	// 書き込み専用サーバは read を数本に絞る設計なので、3サーバで共有している
	// 説明文が検索の口(gkill_get_kyous)に触れるのは避けられない。ここで検査すると
	// 説明を全部書き換えることになり、read / readwrite 側の案内が劣化する。
	// 代わりに、実害の出たランタイム文言（「見つからない」）を下の別テストで固定する。
	servers := []struct {
		label string
		tools []*jsonobj.Object
	}{
		{"read", concatTools(ReadTools, PluginTools)},
		{"readwrite", concatTools(ReadTools, WriteTools, PluginTools)},
	}
	allNames := allToolNames()

	for _, server := range servers {
		t.Run(server.label+" server", func(t *testing.T) {
			available := NewStringSet(toolNames(server.tools)...)
			missing := []string{}
			for _, tool := range server.tools {
				for _, mentioned := range collectMentions(tool, nil).Values() {
					// gkill_kftl / gkill_mcp_readwrite のような create_app 値は素通しする。
					// 判定したいのは「どこかのサーバに実在するツール名なのに、ここには無い」だけ。
					if !allNames.Has(mentioned) {
						continue
					}
					if CrossServerToolMentions.Has(mentioned) {
						continue
					}
					if !available.Has(mentioned) {
						missing = append(missing, strAt(t, tool, "name")+" -> "+mentioned)
					}
				}
			}
			expectEqual(t, missing, []string{})
		})
	}

	// 「見つからない」はランタイムの文言なので、スキーマ検査では拾えない。
	// 書き込み専用サーバはここで案内されるツールを持っている必要がある。
	t.Run("entityNotFoundMessage names only tools the write-only server has", func(t *testing.T) {
		available := NewStringSet()
		for _, name := range toolNames(concatTools(WriteTools, writeServerReadTools, PluginTools)) {
			available.Add(name)
		}
		for _, mentioned := range collectMentions(EntityNotFoundMessage("x1", "kmemo"), nil).Values() {
			if !allNames.Has(mentioned) {
				continue
			}
			expectTrue(t, available.Has(mentioned), "write-only server lacks %s", mentioned)
		}
	})
}

// ---------------------------------------------------------------------------
// read と readwrite は同じ read ツールを配ること
// ---------------------------------------------------------------------------
//
// 「readwrite にだけ user_id が無い」「group_by が違う」という報告が繰り返し来る。
// 実体は毎回「古いプロセスが昨日の定義を配っていた」だったが、コード側で
// 保証しているのは「同じ配列を連結している」という書き方だけで、テストは
// 本数と名前しか見ていなかった。スキーマそのものの同一性を固定する。
func TestReadAndReadwriteServeIdenticalReadTools(t *testing.T) {
	t.Run("same names and same inputSchema", func(t *testing.T) {
		readTools := concatTools(ReadTools, PluginTools)
		readwriteTools := concatTools(ReadTools, WriteTools, PluginTools)

		for _, tool := range readTools {
			name := strAt(t, tool, "name")
			counterpart := findTool(readwriteTools, name)
			expectTrue(t, counterpart != nil, "%s missing from readwrite", name)
			expectEqual(t, counterpart.Value("description"), tool.Value("description"))
			expectEqual(t, counterpart.Value("inputSchema"), tool.Value("inputSchema"))
		}
	})
}
