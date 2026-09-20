package mcp

// MCP 書き込みツール定義と共有ヘルパの検査。
//
// C1 から summarize / ディスパッチ / 削除の対応表は write_handlers.go に一本化され
// 公開されたので、ここは再実装のミラーではなく**実物を直接使って**検証する
// （ミラーは実装が変わっても緑のまま古び、二重管理の温床だった。
//  実際このファイルと readwrite-tool-handlers.test.mjs は
//  「統合サーバは29ツール」「read は8件」という実測と違う値を
//  自分のハードコード配列に対して検証し続けていた）。
//
// read_tools.go 側の同じ形は tool_handlers_test.go にある。

import (
	"sort"
	"strings"
	"testing"

	"github.com/mt3hr/gkill/src/server/gkill/mcp/jsonobj"
)

// ---------------------------------------------------------------------------
// Tool definitions
// ---------------------------------------------------------------------------
func TestWriteToolDefinitions(t *testing.T) {
	t.Run("write server exposes 21 write tools", func(t *testing.T) {
		expectEqual(t, len(WriteTools), 21)
	})

	t.Run("write tool names are the current set", func(t *testing.T) {
		expectEqual(t, toolNames(WriteTools), []string{
			"gkill_add_kmemo",
			"gkill_add_urlog",
			"gkill_add_nlog",
			"gkill_add_lantana",
			"gkill_add_timeis",
			"gkill_add_mi",
			"gkill_add_kc",
			"gkill_add_tag",
			"gkill_add_text",
			"gkill_submit_kftl",
			"gkill_delete_kyou",
			"gkill_update_kmemo",
			"gkill_update_urlog",
			"gkill_update_nlog",
			"gkill_update_lantana",
			"gkill_update_timeis",
			"gkill_update_mi",
			"gkill_update_kc",
			"gkill_update_tag",
			"gkill_update_text",
			"gkill_restore_kyou",
		})
	})

	t.Run("isWriteToolName matches the definitions", func(t *testing.T) {
		for _, tool := range WriteTools {
			expectTrue(t, IsWriteToolName(strAt(t, tool, "name")), "%s is not a write tool", strAt(t, tool, "name"))
		}
		expectTrue(t, !IsWriteToolName("gkill_get_kyous"), "read tool counted as write")
		expectTrue(t, !IsWriteToolName("gkill_get_plugin_list"), "plugin tool counted as write")
	})

	t.Run("every tool has an object inputSchema with additionalProperties: false", func(t *testing.T) {
		// 未知キーを黙って捨てないための不変条件。read 側は tool_handlers_test.go が同じ検査をしている
		for _, tool := range WriteTools {
			schema := objAt(t, tool, "inputSchema")
			expectEqual(t, schema.Value("type"), "object")
			expectEqual(t, schema.Value("additionalProperties"), false)
		}
	})

	t.Run("update tools require only id (patch semantics)", func(t *testing.T) {
		// 正規化層は id 以外すべて optional の patch なのに、スキーマの required だけが
		// title などを強制していた。スキーマに忠実な AI は
		// (a) title を取りに1往復増やすか (b) 推測して送って既存値を静かに上書きする。
		// update_mi の説明文の例 {id, is_checked:true} も自分の required に違反していた
		for _, tool := range WriteTools {
			if !strings.HasPrefix(strAt(t, tool, "name"), "gkill_update_") {
				continue
			}
			expectEqual(t, objAt(t, tool, "inputSchema").Value("required"), strs("id"))
		}
	})

	t.Run("no description advertises the deprecated include_id argument", func(t *testing.T) {
		// ADR-0604 で ID は常時付与になった。案内が残っていると AI が必須引数だと学習する
		for _, tool := range WriteTools {
			mustNotContain(t, strAt(t, tool, "description"), "include_id")
			mustNotContain(t, jsonobj.MarshalString(tool.Value("inputSchema")), "include_id")
		}
	})

	t.Run("urlog の説明は外向き取得の条件を add / update の両側で言い切っている", func(t *testing.T) {
		// add 側: title の「省略するとサーバが取得して埋める」を無条件形で書くと
		// fetch_metadata:false と矛盾する (tool 説明と field 説明の矛盾を直した直後に、
		// フラグ追加で field 側だけが再び無条件形へ取り残された実績がある)。
		addTool := findTool(WriteTools, "gkill_add_urlog")
		mustContain(t, strAt(t, objAt(t, addTool, "inputSchema", "properties", "title"), "description"), "fetch_metadata")
		// update 側: 逆に「外向き通信を起こさない」を明言する (add では選べたフラグが
		// update に無い理由。実挙動は write_handlers_test.go の re_get_urlog_content
		// 非送信テストが固定している)。
		updateTool := findTool(WriteTools, "gkill_update_urlog")
		mustContain(t, strAt(t, updateTool, "description"), "never causes outbound traffic")
	})

	t.Run("every tool has a non-empty description", func(t *testing.T) {
		for _, tool := range WriteTools {
			description, ok := tool.String("description")
			expectTrue(t, ok, "%s: description is not a string", strAt(t, tool, "name"))
			expectTrue(t, len(description) > 0, "%s: empty description", strAt(t, tool, "name"))
		}
	})
}

// ---------------------------------------------------------------------------
// Delete data_type: 語彙が constants.go の1箇所から派生していること
// ---------------------------------------------------------------------------
func TestDeleteKyouDataTypeVocabulary(t *testing.T) {
	t.Run("schema enum accepts projections while DELETE_DATA_TYPES stays the folded vocabulary", func(t *testing.T) {
		// 語彙が食い違うと「スキーマは受理するのにディスパッチで落ちる」
		// （あるいはその逆）になる。畳んだ後の正本は constants.go の EntityTargets。
		// ただし入口は ToEntityDataType を通すので、スキーマ側は射影名も許さないと
		// 「応答の data_type をそのまま次のツールへ渡せる」という説明と食い違う
		// （enum を畳んだ後の語彙だけにすると、送信前に検証するクライアントが
		// mi_start をサーバへ届く前に弾く。実利用レビュー）。
		deleteTool := findTool(WriteTools, "gkill_delete_kyou")
		canonical := []string{}
		for _, target := range EntityTargets {
			canonical = append(canonical, target.DataType)
		}
		sort.Strings(canonical)
		accepted := append([]string{}, EntityAndProjectionDataTypeValues...)
		sort.Strings(accepted)

		enumValues := []string{}
		for _, v := range arrAt(t, deleteTool, "inputSchema", "properties", "data_type", "enum") {
			enumValues = append(enumValues, v.(string))
		}
		sort.Strings(enumValues)
		expectEqual(t, enumValues, accepted)
		// 畳んだ後の語彙は従来どおり EntityTargets と一致する
		expectEqual(t, DeleteDataTypes.Sorted(), canonical)
		// スキーマが受理する射影名は、すべて畳んだ先が語彙にあること
		for _, pair := range ProjectionToEntityDataType {
			expectTrue(t, DeleteDataTypes.Has(ToEntityDataType(pair.Projection)), "%s folds outside the vocabulary", pair.Projection)
		}
	})

	t.Run("every delete target has both a get and an update endpoint", func(t *testing.T) {
		for _, target := range EntityTargets {
			mustMatch(t, target.GetEndpoint, `^/api/`)
			mustMatch(t, target.HistoriesKey, `_histories$`)
			mustMatch(t, target.UpdateEndpoint, `^/api/update_`)
			mustMatch(t, target.ResponseKey, `^updated_`)
			expectTrue(t, target.RequestKey != "", "%s: empty requestKey", target.DataType)
		}
	})
}

// ---------------------------------------------------------------------------
// summarizeWriteToolPayload
// ---------------------------------------------------------------------------
func TestSummarizeWriteToolPayload(t *testing.T) {
	summarize := func(t *testing.T, name string, payload *jsonobj.Object) string {
		t.Helper()
		summary, ok := SummarizeWriteToolPayload(name, payload)
		expectTrue(t, ok, "%s: no summary", name)
		return summary
	}

	t.Run("add tools report the created id", func(t *testing.T) {
		expectEqual(t, summarize(t, "gkill_add_kmemo", obj("added_kmemo", obj("id", "k1"))), "Created kmemo: k1")
		expectEqual(t, summarize(t, "gkill_add_mi", obj("added_mi", obj("id", "m1"))), "Created mi: m1")
		expectEqual(t, summarize(t, "gkill_add_tag", obj("added_tag", obj("id", "t1"))), "Added tag: t1")
		expectEqual(t, summarize(t, "gkill_add_text", obj("added_text", obj("id", "x1"))), "Added text: x1")
	})

	t.Run("add tools fall back to unknown when the entity is missing", func(t *testing.T) {
		expectEqual(t, summarize(t, "gkill_add_kmemo", obj()), "Created kmemo: unknown")
	})

	t.Run("update tools report the updated id", func(t *testing.T) {
		expectEqual(t, summarize(t, "gkill_update_kmemo", obj("updated_kmemo", obj("id", "k1"))), "Updated kmemo: k1")
		expectEqual(t, summarize(t, "gkill_update_mi", obj("updated_mi", obj("id", "m1"))), "Updated mi: m1")
	})

	t.Run("gkill_submit_kftl reports what was written, not how many messages came back", func(t *testing.T) {
		// 「N messages」はサーバの定型文の本数でしかなく、何が作られたかを伝えていなかった
		expectEqual(t, summarize(t, "gkill_submit_kftl", obj(
			"messages", arr(obj()),
			"created", arr(
				obj("id", "a", "data_type", "kmemo"),
				obj("id", "b", "data_type", "lantana"),
			),
		)), "KFTL submitted: wrote 2 record(s) — kmemo, lantana.")
	})

	t.Run("gkill_submit_kftl groups repeats and marks updates", func(t *testing.T) {
		// 打刻の終了は新規作成ではなく既存レコードの更新
		expectEqual(t, summarize(t, "gkill_submit_kftl", obj(
			"created", arr(
				obj("id", "a", "data_type", "kmemo"),
				obj("id", "b", "data_type", "kmemo"),
				obj("id", "c", "data_type", "timeis", "updated", true),
			),
		)), "KFTL submitted: wrote 3 record(s) — kmemo x2, timeis (updated).")
	})

	t.Run("gkill_submit_kftl says so when nothing was written", func(t *testing.T) {
		// 空行だけのテキストは何も書かない
		expectEqual(t, summarize(t, "gkill_submit_kftl", obj("messages", arr(obj()))), "KFTL submitted: nothing was written (blank lines write nothing).")
	})

	t.Run("gkill_submit_kftl says a replay returned the original created[]", func(t *testing.T) {
		// 冪等キーで畳んだ再送は元の created[] を replayed:true で返し、今回は何も書かない（ADR-0510）
		expectEqual(t, summarize(t, "gkill_submit_kftl", obj("messages", arr(obj()), "replayed", true, "created", arr(obj("id", "a", "data_type", "kmemo")))),
			"KFTL replay folded: 1 record(s) of the original submission returned again (replayed:true, nothing written this time).")
	})

	t.Run("gkill_delete_kyou names the type and id instead of the response keys", func(t *testing.T) {
		expectEqual(t, summarize(t, "gkill_delete_kyou", obj("updated_kmemo", obj("id", "k1"), "updated_kyou", obj("id", "k1"))), "Deleted (soft): kmemo k1")
		expectEqual(t, summarize(t, "gkill_delete_kyou", obj("updated_kmemo", obj(), "updated_kyou", obj())), "Deleted (soft): kmemo (id unknown)")
		expectEqual(t, summarize(t, "gkill_delete_kyou", obj()), "Deleted (soft): completed")
	})

	t.Run("returns null for tools it does not own (server falls through to read/plugin)", func(t *testing.T) {
		// 無しを返さないと、基底の3段フォールバックが read の要約を上書きしてしまう
		for _, name := range []string{"gkill_get_kyous", "gkill_get_plugin_list", "unknown_tool"} {
			_, ok := SummarizeWriteToolPayload(name, obj())
			expectTrue(t, !ok, "%s was summarized by the write summarizer", name)
		}
	})

	t.Run("covers every tool in WRITE_TOOLS", func(t *testing.T) {
		// ツールを足して要約の case を忘れると "Tool call completed." に落ちて静かに劣化する
		for _, tool := range WriteTools {
			_, ok := SummarizeWriteToolPayload(strAt(t, tool, "name"), obj())
			expectTrue(t, ok, "%s has no summary", strAt(t, tool, "name"))
		}
	})
}

// ---------------------------------------------------------------------------
// 後付け boolean 引数の2表整合 (メタテスト)
// ---------------------------------------------------------------------------
func TestLateBooleanArgsAreInBothTables(t *testing.T) {
	// ツールスキーマはクライアントのセッション寿命で固定されるので、後から足した
	// boolean 引数は既存セッションから正規 JSON 文字列 ("false") で届く。
	// 復元 (write_normalization の revivesStaleBoolean) と検出 (normalization の
	// StaleSchemaArgKindsByTool) のどちらか片方を忘れると、その引数は
	// 「新しいセッションでだけ動く」状態で出荷される (read 側の is_video で実際に起きた)。
	// 両ファイルの注意書きコメントだけが頼りだったので、スキーマの boolean プロパティ
	// 全件を回して機械強制する。

	// スキーマ導入時から boolean だった引数。最初のセッションから型付きで届くので
	// 救済表には載せない (ここへ足す行為自体が「後付けではない」という意思表示になる)。
	dayOneBooleans := NewStringSet("gkill_add_mi.is_checked", "gkill_update_mi.is_checked")

	// 後付け boolean を持つツールの実物の正規化器と最小引数。
	// 新しいツール名でこのテストが落ちたら、ここへ1行足す。
	normalizerByTool := map[string]func(extra *jsonobj.Object) (*jsonobj.Object, error){
		"gkill_add_urlog": func(extra *jsonobj.Object) (*jsonobj.Object, error) {
			return NormalizeUrlogArgs(obj("url", "https://example.com/").Merge(extra))
		},
		"gkill_add_mi": func(extra *jsonobj.Object) (*jsonobj.Object, error) {
			return NormalizeMiArgs(obj("title", "t").Merge(extra))
		},
		"gkill_update_mi": func(extra *jsonobj.Object) (*jsonobj.Object, error) {
			return NormalizeUpdateMiArgs(obj("id", "m1", "board_name", "b").Merge(extra))
		},
	}

	type booleanProp struct{ toolName, prop string }
	booleanProps := []booleanProp{}
	for _, tool := range WriteTools {
		name := strAt(t, tool, "name")
		properties, ok := objAt(t, tool, "inputSchema").Object("properties")
		if !ok {
			continue
		}
		for _, prop := range properties.Keys() {
			schema := objAt(t, properties, prop)
			if schema.Value("type") == "boolean" && !dayOneBooleans.Has(name+"."+prop) {
				booleanProps = append(booleanProps, booleanProp{name, prop})
			}
		}
	}

	t.Run("スキーマに後付け boolean が実在する (この検査自体の空振り防止)", func(t *testing.T) {
		expectTrue(t, len(booleanProps) >= 4, "only %d late booleans", len(booleanProps))
	})

	for _, bp := range booleanProps {
		t.Run(bp.toolName+"."+bp.prop+" は検出表に載り、正規JSON文字列から復元される", func(t *testing.T) {
			// (a) 検出: 文字列で届いたことが「古いスキーマの証拠」として警告経路に乗る
			signals := DetectStaleSchemaSignals(bp.toolName, obj(bp.prop, "false"))
			revived := []string{}
			if signals != nil {
				revived = signals.Revived
			}
			expectTrue(t, NewStringSet(revived...).Has(bp.prop), "%s.%s が StaleSchemaArgKindsByTool に無い", bp.toolName, bp.prop)
			// (b) 復元: 実物の正規化器が boolean へ戻す (revivesStaleBoolean の付け忘れ検出)
			normalize, ok := normalizerByTool[bp.toolName]
			expectTrue(t, ok, "normalizerByTool に %s の行が無い — 後付け boolean を足したらここへも1行", bp.toolName)
			normalized, err := normalize(obj(bp.prop, "false"))
			expectNoError(t, err)
			expectEqual(t, normalized.Value(bp.prop), false)
		})
	}
}

// ---------------------------------------------------------------------------
// summarizeToolError (payload.go の実物)
// ---------------------------------------------------------------------------
func TestSummarizeToolErrorWrite(t *testing.T) {
	t.Run("includes tool name and error", func(t *testing.T) {
		result := SummarizeToolError("gkill_add_kmemo", "Connection refused", nil)
		mustContain(t, result, "gkill_add_kmemo")
		mustContain(t, result, "Connection refused")
	})

	t.Run("includes field when present", func(t *testing.T) {
		result := SummarizeToolError("gkill_add_kmemo", "Invalid", obj("field", "content"))
		mustContain(t, result, "content")
	})

	t.Run("handles empty tool name", func(t *testing.T) {
		mustContain(t, SummarizeToolError("", "Timeout", nil), "Timeout")
	})
}
