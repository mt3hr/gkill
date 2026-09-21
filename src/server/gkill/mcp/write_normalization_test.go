package mcp

// write_normalization.go（書き込みツールの引数正規化）の検査。

import (
	"testing"

	"github.com/mt3hr/gkill/src/server/gkill/mcp/jsonobj"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// expectThrowsField は「落ちる、かつ detail.field がその欄」（expectThrowsField）。
func expectThrowsField(t testing.TB, _ *jsonobj.Object, err error, field string) {
	t.Helper()
	expectTrue(t, err != nil, "expected an error for field %s", field)
	got, _ := DetailField(err)
	expectEqual(t, got, field)
}

// ---------------------------------------------------------------------------
// normalizeKmemoArgs
// ---------------------------------------------------------------------------
func TestNormalizeKmemoArgs(t *testing.T) {
	t.Run("accepts valid content", func(t *testing.T) {
		result := mustNormalize(t)(NormalizeKmemoArgs(obj("content", "hello")))
		expectEqual(t, result.Value("content"), "hello")
		expectTrue(t, !result.Defined("related_time"), "related_time set")
	})

	t.Run("normalizes related_time date-only", func(t *testing.T) {
		result := mustNormalize(t)(NormalizeKmemoArgs(obj("content", "test", "related_time", "2026-03-15")))
		mustMatch(t, strAt(t, result, "related_time"), `^2026-03-15T`)
	})

	t.Run("passes through locale_name", func(t *testing.T) {
		result := mustNormalize(t)(NormalizeKmemoArgs(obj("content", "test", "locale_name", "en")))
		expectEqual(t, result.Value("locale_name"), "en")
	})

	t.Run("rejects missing content", func(t *testing.T) {
		r, err := NormalizeKmemoArgs(obj())
		expectThrowsField(t, r, err, "content")
	})

	t.Run("rejects non-string content", func(t *testing.T) {
		r, err := NormalizeKmemoArgs(obj("content", 123))
		expectThrowsField(t, r, err, "content")
	})

	t.Run("rejects unknown keys", func(t *testing.T) {
		_, err := NormalizeKmemoArgs(obj("content", "x", "foo", "bar"))
		expectTrue(t, err != nil, "accepted")
	})

	t.Run("rejects non-object args", func(t *testing.T) {
		_, err := NormalizeKmemoArgs("string")
		expectTrue(t, err != nil, "accepted a string")
		_, err = NormalizeKmemoArgs(nil)
		expectTrue(t, err != nil, "accepted null")
	})
}

// ---------------------------------------------------------------------------
// normalizeUrlogArgs
// ---------------------------------------------------------------------------
func TestNormalizeUrlogArgs(t *testing.T) {
	t.Run("accepts url only", func(t *testing.T) {
		result := mustNormalize(t)(NormalizeUrlogArgs(obj("url", "https://example.com")))
		expectEqual(t, result.Value("url"), "https://example.com")
		expectTrue(t, !result.Defined("title"), "title set")
	})

	t.Run("accepts url with title", func(t *testing.T) {
		result := mustNormalize(t)(NormalizeUrlogArgs(obj("url", "https://example.com", "title", "Example")))
		expectEqual(t, result.Value("title"), "Example")
	})

	t.Run("rejects missing url", func(t *testing.T) {
		r, err := NormalizeUrlogArgs(obj())
		expectThrowsField(t, r, err, "url")
	})
}

// ---------------------------------------------------------------------------
// normalizeNlogArgs
// ---------------------------------------------------------------------------
func TestNormalizeNlogArgs(t *testing.T) {
	t.Run("accepts title and amount", func(t *testing.T) {
		result := mustNormalize(t)(NormalizeNlogArgs(obj("title", "lunch", "amount", 1500)))
		expectEqual(t, result.Value("title"), "lunch")
		expectEqual(t, result.Value("amount"), 1500)
	})

	t.Run("accepts optional shop", func(t *testing.T) {
		result := mustNormalize(t)(NormalizeNlogArgs(obj("title", "lunch", "amount", 1500, "shop", "cafe")))
		expectEqual(t, result.Value("shop"), "cafe")
	})

	t.Run("accepts negative amounts", func(t *testing.T) {
		result := mustNormalize(t)(NormalizeNlogArgs(obj("title", "refund", "amount", -500)))
		expectEqual(t, result.Value("amount"), -500)
	})

	t.Run("rejects missing amount", func(t *testing.T) {
		r, err := NormalizeNlogArgs(obj("title", "lunch"))
		expectThrowsField(t, r, err, "amount")
	})

	t.Run("rejects non-number amount", func(t *testing.T) {
		r, err := NormalizeNlogArgs(obj("title", "lunch", "amount", "abc"))
		expectThrowsField(t, r, err, "amount")
	})

	t.Run("rejects missing title", func(t *testing.T) {
		r, err := NormalizeNlogArgs(obj("amount", 100))
		expectThrowsField(t, r, err, "title")
	})
}

// ---------------------------------------------------------------------------
// normalizeLantanaArgs
// ---------------------------------------------------------------------------
func TestNormalizeLantanaArgs(t *testing.T) {
	t.Run("accepts mood 0", func(t *testing.T) {
		expectEqual(t, mustNormalize(t)(NormalizeLantanaArgs(obj("mood", 0))).Value("mood"), 0)
	})

	t.Run("accepts mood 10", func(t *testing.T) {
		expectEqual(t, mustNormalize(t)(NormalizeLantanaArgs(obj("mood", 10))).Value("mood"), 10)
	})

	t.Run("accepts mood 5", func(t *testing.T) {
		expectEqual(t, mustNormalize(t)(NormalizeLantanaArgs(obj("mood", 5))).Value("mood"), 5)
	})

	t.Run("rejects mood < 0", func(t *testing.T) {
		r, err := NormalizeLantanaArgs(obj("mood", -1))
		expectThrowsField(t, r, err, "mood")
	})

	t.Run("rejects mood > 10", func(t *testing.T) {
		r, err := NormalizeLantanaArgs(obj("mood", 11))
		expectThrowsField(t, r, err, "mood")
	})

	t.Run("rejects non-integer mood", func(t *testing.T) {
		r, err := NormalizeLantanaArgs(obj("mood", 5.5))
		expectThrowsField(t, r, err, "mood")
	})

	t.Run("rejects missing mood", func(t *testing.T) {
		r, err := NormalizeLantanaArgs(obj())
		expectThrowsField(t, r, err, "mood")
	})
}

// ---------------------------------------------------------------------------
// normalizeTimeIsArgs
// ---------------------------------------------------------------------------
func TestNormalizeTimeIsArgs(t *testing.T) {
	t.Run("accepts title only", func(t *testing.T) {
		result := mustNormalize(t)(NormalizeTimeIsArgs(obj("title", "coding")))
		expectEqual(t, result.Value("title"), "coding")
		expectTrue(t, !result.Defined("start_time"), "start_time set")
		expectTrue(t, !result.Defined("end_time"), "end_time set")
	})

	t.Run("accepts start_time and end_time", func(t *testing.T) {
		result := mustNormalize(t)(NormalizeTimeIsArgs(obj(
			"title", "meeting",
			"start_time", "2026-03-15T09:00:00+09:00",
			"end_time", "2026-03-15T10:00:00+09:00",
		)))
		expectEqual(t, result.Value("start_time"), "2026-03-15T09:00:00+09:00")
		expectEqual(t, result.Value("end_time"), "2026-03-15T10:00:00+09:00")
	})

	t.Run("rejects missing title", func(t *testing.T) {
		r, err := NormalizeTimeIsArgs(obj())
		expectThrowsField(t, r, err, "title")
	})
}

// ---------------------------------------------------------------------------
// normalizeMiArgs
// ---------------------------------------------------------------------------
func TestNormalizeMiArgs(t *testing.T) {
	t.Run("accepts required fields", func(t *testing.T) {
		result := mustNormalize(t)(NormalizeMiArgs(obj("title", "fix bug", "board_name", "dev")))
		expectEqual(t, result.Value("title"), "fix bug")
		expectEqual(t, result.Value("board_name"), "dev")
		expectEqual(t, result.Value("is_checked"), false)
	})

	t.Run("accepts is_checked true", func(t *testing.T) {
		result := mustNormalize(t)(NormalizeMiArgs(obj("title", "done", "board_name", "dev", "is_checked", true)))
		expectEqual(t, result.Value("is_checked"), true)
	})

	t.Run("accepts optional time fields", func(t *testing.T) {
		result := mustNormalize(t)(NormalizeMiArgs(obj(
			"title", "task",
			"board_name", "dev",
			"limit_time", "2026-04-01",
			"estimate_start_time", "2026-03-20T09:00:00+09:00",
		)))
		mustMatch(t, strAt(t, result, "limit_time"), `^2026-04-01T`)
		expectEqual(t, result.Value("estimate_start_time"), "2026-03-20T09:00:00+09:00")
	})

	t.Run("rejects missing title", func(t *testing.T) {
		r, err := NormalizeMiArgs(obj("board_name", "dev"))
		expectThrowsField(t, r, err, "title")
	})

	t.Run("allows omitting board_name (the handler fills in the default board)", func(t *testing.T) {
		// スキーマは optional と宣言しているので省略できなければならない。
		// 以前はここで型エラーになり、既定板へ入れる意図の呼び出しが必ず失敗していた
		result := mustNormalize(t)(NormalizeMiArgs(obj("title", "task")))
		expectTrue(t, !result.Defined("board_name"), "board_name set")
		expectEqual(t, result.Value("title"), "task")
	})

	t.Run("still rejects a non-string board_name", func(t *testing.T) {
		r, err := NormalizeMiArgs(obj("title", "task", "board_name", 1))
		expectThrowsField(t, r, err, "board_name")
	})

	t.Run("rejects non-boolean is_checked", func(t *testing.T) {
		r, err := NormalizeMiArgs(obj("title", "t", "board_name", "b", "is_checked", "yes"))
		expectThrowsField(t, r, err, "is_checked")
	})
}

// ---------------------------------------------------------------------------
// normalizeKcArgs
// ---------------------------------------------------------------------------
func TestNormalizeKcArgs(t *testing.T) {
	t.Run("accepts title and num_value", func(t *testing.T) {
		result := mustNormalize(t)(NormalizeKcArgs(obj("title", "steps", "num_value", 10000)))
		expectEqual(t, result.Value("title"), "steps")
		expectEqual(t, result.Value("num_value"), 10000)
	})

	t.Run("accepts float num_value", func(t *testing.T) {
		result := mustNormalize(t)(NormalizeKcArgs(obj("title", "temp", "num_value", 36.5)))
		expectEqual(t, result.Value("num_value"), 36.5)
	})

	t.Run("rejects missing num_value", func(t *testing.T) {
		r, err := NormalizeKcArgs(obj("title", "x"))
		expectThrowsField(t, r, err, "num_value")
	})
}

// ---------------------------------------------------------------------------
// normalizeTagArgs
// ---------------------------------------------------------------------------
func TestNormalizeTagArgs(t *testing.T) {
	t.Run("accepts tag and target_id", func(t *testing.T) {
		result := mustNormalize(t)(NormalizeTagArgs(obj("tag", "important", "target_id", "uuid-123")))
		expectEqual(t, result.Value("tag"), "important")
		expectEqual(t, result.Value("target_id"), "uuid-123")
	})

	t.Run("rejects missing tag", func(t *testing.T) {
		r, err := NormalizeTagArgs(obj("target_id", "id"))
		expectThrowsField(t, r, err, "tag")
	})

	t.Run("rejects missing target_id", func(t *testing.T) {
		r, err := NormalizeTagArgs(obj("tag", "t"))
		expectThrowsField(t, r, err, "target_id")
	})
}

// ---------------------------------------------------------------------------
// normalizeTextArgs
// ---------------------------------------------------------------------------
func TestNormalizeTextArgs(t *testing.T) {
	t.Run("accepts text and target_id", func(t *testing.T) {
		result := mustNormalize(t)(NormalizeTextArgs(obj("text", "annotation", "target_id", "uuid-456")))
		expectEqual(t, result.Value("text"), "annotation")
		expectEqual(t, result.Value("target_id"), "uuid-456")
	})

	t.Run("rejects missing text", func(t *testing.T) {
		r, err := NormalizeTextArgs(obj("target_id", "id"))
		expectThrowsField(t, r, err, "text")
	})

	t.Run("rejects missing target_id", func(t *testing.T) {
		r, err := NormalizeTextArgs(obj("text", "t"))
		expectThrowsField(t, r, err, "target_id")
	})
}

// ---------------------------------------------------------------------------
// normalizeKftlArgs
// ---------------------------------------------------------------------------
func TestNormalizeKftlArgs(t *testing.T) {
	t.Run("accepts kftl_text", func(t *testing.T) {
		result := mustNormalize(t)(NormalizeKftlArgs(obj("kftl_text", "/mi Buy milk")))
		expectEqual(t, result.Value("kftl_text"), "/mi Buy milk")
	})

	t.Run("rejects missing kftl_text", func(t *testing.T) {
		r, err := NormalizeKftlArgs(obj())
		expectThrowsField(t, r, err, "kftl_text")
	})

	t.Run("rejects empty kftl_text", func(t *testing.T) {
		r, err := NormalizeKftlArgs(obj("kftl_text", "  "))
		expectThrowsField(t, r, err, "kftl_text")
	})
}

// ---------------------------------------------------------------------------
// normalizeDeleteArgs
// ---------------------------------------------------------------------------
func TestNormalizeDeleteArgs(t *testing.T) {
	t.Run("accepts valid id and data_type", func(t *testing.T) {
		result := mustNormalize(t)(NormalizeDeleteArgs(obj("id", "uuid-789", "data_type", "kmemo")))
		expectEqual(t, result.Value("batch"), false)
		expectEqual(t, result.Value("targets"), arr(obj("id", "uuid-789", "data_type", "kmemo")))
	})

	t.Run("accepts all valid data_types", func(t *testing.T) {
		for _, dt := range DeleteDataTypes.Values() {
			result := mustNormalize(t)(NormalizeDeleteArgs(obj("id", "id", "data_type", dt)))
			expectEqual(t, objAt(t, arrAt(t, result, "targets")[0]).Value("data_type"), dt)
		}
	})

	t.Run("rejects invalid data_type", func(t *testing.T) {
		r, err := NormalizeDeleteArgs(obj("id", "id", "data_type", "invalid"))
		expectThrowsField(t, r, err, "data_type")
	})

	t.Run("rejects missing id", func(t *testing.T) {
		r, err := NormalizeDeleteArgs(obj("data_type", "kmemo"))
		expectThrowsField(t, r, err, "id")
	})

	t.Run("rejects missing data_type", func(t *testing.T) {
		r, err := NormalizeDeleteArgs(obj("id", "id"))
		expectThrowsField(t, r, err, "data_type")
	})
}

// ---------------------------------------------------------------------------
// DELETE_DATA_TYPES constant
// ---------------------------------------------------------------------------
func TestDeleteDataTypes(t *testing.T) {
	t.Run("contains 12 data types", func(t *testing.T) {
		expectEqual(t, DeleteDataTypes.Len(), 12)
	})

	t.Run("contains expected types", func(t *testing.T) {
		// rekyou / mirekyou / notification は rep が実在するのに削除も履歴取得もできなかった。
		// 作成(add)は今も無いが、他の経路が作ったものを消す・戻すことはできる
		expected := []string{"kmemo", "urlog", "nlog", "lantana", "timeis", "mi", "kc", "tag", "text",
			"rekyou", "mirekyou", "notification"}
		for _, dt := range expected {
			expectTrue(t, DeleteDataTypes.Has(dt), "missing %s", dt)
		}
	})
}

func TestNormalizeUrlogArgsURLScheme(t *testing.T) {
	// スキームが無いと gkill はページ取得すら試みず、title が空のまま保存される。
	// エラーは出ないので「タイトルの自動補完が効かなかった」としか見えない。
	t.Run("rejects a URL without a scheme", func(t *testing.T) {
		_, err := NormalizeUrlogArgs(obj("url", "example.com/foo"))
		expectErrorMatches(t, err, `must include a scheme`)
		_, err = NormalizeUrlogArgs(obj("url", "//example.com/foo"))
		expectErrorMatches(t, err, `must include a scheme`)
	})

	t.Run("accepts http/https and other schemes", func(t *testing.T) {
		expectEqual(t, mustNormalize(t)(NormalizeUrlogArgs(obj("url", "https://example.com/foo"))).Value("url"), "https://example.com/foo")
		expectEqual(t, mustNormalize(t)(NormalizeUrlogArgs(obj("url", "http://example.com/"))).Value("url"), "http://example.com/")
		expectEqual(t, mustNormalize(t)(NormalizeUrlogArgs(obj("url", "file://host/path"))).Value("url"), "file://host/path")
	})
}

func TestTimeIsOrdering(t *testing.T) {
	t.Run("rejects end_time before start_time instead of creating a negative interval", func(t *testing.T) {
		_, err := NormalizeTimeIsArgs(obj(
			"title", "x",
			"start_time", "2026-08-24T12:00:00+09:00",
			"end_time", "2026-08-24T11:00:00+09:00",
		))
		expectErrorMatches(t, err, `negative length`)
	})

	t.Run("applies to update as well, but only when both ends are supplied", func(t *testing.T) {
		_, err := NormalizeUpdateTimeIsArgs(obj(
			"id", "t1",
			"start_time", "2026-08-24T12:00:00+09:00",
			"end_time", "2026-08-24T11:00:00+09:00",
		))
		expectErrorMatches(t, err, `negative length`)
		// 片側だけの patch は既存値と突き合わせられないので通す
		_, err = NormalizeUpdateTimeIsArgs(obj("id", "t1", "end_time", "2026-08-24T11:00:00+09:00"))
		expectNoError(t, err)
	})

	t.Run("compares instants, not strings", func(t *testing.T) {
		// NormalizeDateTimeString は妥当な RFC3339 をそのまま返すので、offset が混ざりうる。
		// 文字列順と実際の前後が食い違う組み合わせでだけ、この違いが表に出る。

		// 文字列順では start < end に見えるが、実際は start(18:00Z) が end(16:00Z) より後
		_, err := NormalizeTimeIsArgs(obj(
			"title", "x",
			"start_time", "2026-08-23T18:00:00Z",
			"end_time", "2026-08-24T01:00:00+09:00",
		))
		expectErrorMatches(t, err, `negative length`)

		// 逆に文字列順では start > end に見えるが、実際は正しい順序なので通す
		_, err = NormalizeTimeIsArgs(obj(
			"title", "x",
			"start_time", "2026-08-24T02:00:00+09:00",
			"end_time", "2026-08-23T18:00:00Z",
		))
		expectNoError(t, err)

		// 同じ瞬間は長さ0であって負ではない
		_, err = NormalizeTimeIsArgs(obj(
			"title", "x",
			"start_time", "2026-08-24T02:00:00+09:00",
			"end_time", "2026-08-23T17:00:00Z",
		))
		expectNoError(t, err)
	})
}

func TestReopenFinishedTimeIs(t *testing.T) {
	// 以前は end_time:"" が「空文字は不可」で弾かれ、null は未指定と同じ扱いだったので、
	// 一度終わらせた TimeIs を MCP から二度と進行中に戻せなかった。
	t.Run("null means clear, and is distinct from omitting the field", func(t *testing.T) {
		result := mustNormalize(t)(NormalizeUpdateTimeIsArgs(obj("id", "t1", "end_time", nil)))
		expectTrue(t, result.Has("end_time") && result.Value("end_time") == nil, "end_time is not null")
		expectTrue(t, !mustNormalize(t)(NormalizeUpdateTimeIsArgs(obj("id", "t1"))).Defined("end_time"), "end_time set")
	})

	t.Run("a null end_time skips the ordering check", func(t *testing.T) {
		_, err := NormalizeUpdateTimeIsArgs(obj("id", "t1", "start_time", "2026-08-24T12:00:00+09:00", "end_time", nil))
		expectNoError(t, err)
	})

	t.Run("an empty string is still rejected — it is not a way to clear", func(t *testing.T) {
		_, err := NormalizeUpdateTimeIsArgs(obj("id", "t1", "end_time", ""))
		expectErrorMatches(t, err, `must not be empty`)
	})

	t.Run("add_timeis is unaffected: omitting end_time already means ongoing", func(t *testing.T) {
		expectTrue(t, !mustNormalize(t)(NormalizeTimeIsArgs(obj("title", "x", "start_time", "2026-08-24T12:00:00+09:00"))).Defined("end_time"), "end_time set")
	})
}

// ---------------------------------------------------------------------------
// 日付だけを渡したときの展開先
//
// 締切と見積終了は「その日じゅう」の意味なので、その日の終わりへ展開する。
// 00:00:00 に丸めると「8/25締切」が25日の開始時点で期限切れになる。
// 読み取り側は calendar_end_date を 23:59:59 へ展開すると明記しており、
// 書き込み側だけが一律 00:00:00 のままだった（実利用レビュー）。
// ---------------------------------------------------------------------------

func TestDateOnlyExpansionOnWrite(t *testing.T) {
	t.Run("limit_time expands to the end of the day", func(t *testing.T) {
		result := mustNormalize(t)(NormalizeMiArgs(obj("title", "t", "board_name", "b", "limit_time", "2026-08-25")))
		mustMatch(t, strAt(t, result, "limit_time"), `^2026-08-25T23:59:59`)
	})

	t.Run("estimate_end_time expands to the end of the day", func(t *testing.T) {
		result := mustNormalize(t)(NormalizeMiArgs(obj("title", "t", "board_name", "b", "estimate_end_time", "2026-08-25")))
		mustMatch(t, strAt(t, result, "estimate_end_time"), `^2026-08-25T23:59:59`)
	})

	// 開始側は「その日の始まり」のままでなければならない。
	// 両端を終わりへ寄せると、開始 > 終了 の順序検査に引っかかる形が生まれる。
	t.Run("estimate_start_time still expands to the start of the day", func(t *testing.T) {
		result := mustNormalize(t)(NormalizeMiArgs(obj("title", "t", "board_name", "b", "estimate_start_time", "2026-08-25")))
		mustMatch(t, strAt(t, result, "estimate_start_time"), `^2026-08-25T00:00:00`)
	})

	t.Run("a full datetime is untouched", func(t *testing.T) {
		result := mustNormalize(t)(NormalizeMiArgs(obj("title", "t", "board_name", "b", "limit_time", "2026-08-25T09:00:00+09:00")))
		expectEqual(t, result.Value("limit_time"), "2026-08-25T09:00:00+09:00")
	})

	t.Run("update_mi follows the same rule", func(t *testing.T) {
		result := mustNormalize(t)(NormalizeUpdateMiArgs(obj("id", "abc", "limit_time", "2026-08-25")))
		mustMatch(t, strAt(t, result, "limit_time"), `^2026-08-25T23:59:59`)
	})
}

// ---------------------------------------------------------------------------
// 射影名とエンティティ種別の2語彙
// ---------------------------------------------------------------------------

func firstTargetDataType(t *testing.T) func(*jsonobj.Object, error) any {
	return func(result *jsonobj.Object, err error) any {
		t.Helper()
		expectNoError(t, err)
		return objAt(t, arrAt(t, result, "targets")[0]).Value("data_type")
	}
}

func TestProjectionDataTypeIsAcceptedByDeleteRestore(t *testing.T) {
	t.Run("mi_create folds to mi", func(t *testing.T) {
		expectEqual(t, firstTargetDataType(t)(NormalizeDeleteArgs(obj("id", "abc", "data_type", "mi_create"))), "mi")
		expectEqual(t, firstTargetDataType(t)(NormalizeRestoreArgs(obj("id", "abc", "data_type", "mi_create"))), "mi")
	})

	t.Run("timeis_start / timeis_end fold to timeis", func(t *testing.T) {
		expectEqual(t, firstTargetDataType(t)(NormalizeDeleteArgs(obj("id", "abc", "data_type", "timeis_start"))), "timeis")
		expectEqual(t, firstTargetDataType(t)(NormalizeDeleteArgs(obj("id", "abc", "data_type", "timeis_end"))), "timeis")
	})

	t.Run("mirekyou projections fold to mirekyou, not mi", func(t *testing.T) {
		// 接頭辞で判定すると mirekyou_* が mi になる（prefix 判定は mirekyou を先に見る必要がある）
		expectEqual(t, firstTargetDataType(t)(NormalizeDeleteArgs(obj("id", "abc", "data_type", "mirekyou_check"))), "mirekyou")
	})

	t.Run("entity types still pass through unchanged", func(t *testing.T) {
		expectEqual(t, firstTargetDataType(t)(NormalizeDeleteArgs(obj("id", "abc", "data_type", "mi"))), "mi")
		expectEqual(t, firstTargetDataType(t)(NormalizeDeleteArgs(obj("id", "abc", "data_type", "kmemo"))), "kmemo")
	})

	t.Run("unknown values are still rejected", func(t *testing.T) {
		r, err := NormalizeDeleteArgs(obj("id", "abc", "data_type", "bogus"))
		expectThrowsField(t, r, err, "data_type")
		// idf は Kyou 側の data_type だが、エンティティとしての削除は未対応なので通さない
		r, err = NormalizeDeleteArgs(obj("id", "abc", "data_type", "idf"))
		expectThrowsField(t, r, err, "data_type")
	})
}

func TestDeleteRestoreBatchForm(t *testing.T) {
	t.Run("accepts targets and marks the call as batch", func(t *testing.T) {
		result := mustNormalize(t)(NormalizeDeleteArgs(obj(
			"targets", arr(obj("id", "a", "data_type", "kmemo"), obj("id", "b", "data_type", "mi_create")),
		)))
		expectEqual(t, result.Value("batch"), true)
		// 射影名は一括側でもエンティティ種別へ寄せる
		expectEqual(t, result.Value("targets"), arr(
			obj("id", "a", "data_type", "kmemo"),
			obj("id", "b", "data_type", "mi"),
		))
	})

	t.Run("rejects mixing the two forms", func(t *testing.T) {
		r, err := NormalizeDeleteArgs(obj("id", "a", "data_type", "kmemo", "targets", arr(obj("id", "b", "data_type", "kmemo"))))
		expectThrowsField(t, r, err, "targets")
	})

	t.Run("rejects an empty targets list", func(t *testing.T) {
		r, err := NormalizeDeleteArgs(obj("targets", arr()))
		expectThrowsField(t, r, err, "targets")
	})

	// gkill_submit_kftl の created[] をそのまま渡すと updated / related_time が未知キーになる。
	// 汎用の「is not supported」ではなく、変換の仕方と updated:true を消してはいけない理由を言う。
	t.Run("explains how to pass gkill_submit_kftl created[] entries", func(t *testing.T) {
		created := arr(obj("id", "a", "data_type", "kmemo", "updated", false, "related_time", "2026-09-19T10:00:00+09:00"))
		r, err := NormalizeDeleteArgs(obj("targets", created))
		expectThrowsField(t, r, err, "targets[0]")
		mustContain(t, err.Error(), "created.filter(c => !c.updated)")
		mustContain(t, err.Error(), "updated:true")
		expectEqual(t, mustNormalize(t)(NormalizeRestoreArgs(obj("targets", arr(obj("id", "a", "data_type", "kmemo"))))).Value("targets"), arr(obj("id", "a", "data_type", "kmemo")))
	})

	t.Run("reports which entry is malformed", func(t *testing.T) {
		r, err := NormalizeDeleteArgs(obj("targets", arr(obj("id", "a", "data_type", "kmemo"), obj("id", "b", "data_type", "bogus"))))
		expectThrowsField(t, r, err, "targets[1].data_type")
	})

	// 古いスキーマを掴んだクライアントは未知の引数を正規JSON文字列で送ってくる。
	// 表に載せ忘れると boolean/number/配列の新引数は既存の全クライアントから型エラーになる。
	t.Run("revives targets sent as a canonical JSON string", func(t *testing.T) {
		result := mustNormalize(t)(NormalizeDeleteArgs(obj("targets", `[{"id":"a","data_type":"kmemo"}]`)))
		expectEqual(t, result.Value("targets"), arr(obj("id", "a", "data_type", "kmemo")))
	})

	// restore も同じ経路 (normalizeDeleteTargets) を通ることを restore 側の実関数で1本固定する。
	t.Run("restore batch form works the same way and folds projections", func(t *testing.T) {
		result := mustNormalize(t)(NormalizeRestoreArgs(obj("targets", arr(obj("id", "a", "data_type", "mi_create")))))
		expectEqual(t, result.Value("batch"), true)
		expectEqual(t, result.Value("targets"), arr(obj("id", "a", "data_type", "mi")))
	})

	// ツール説明は「単件・一括のどちらも無い呼び出しは実行時に拒否される」と AI へ約束している。
	// その分岐 (verb 入りの専用文言) はここでしか到達しない。
	t.Run("rejects a call with no target at all, naming the batch alternative with the right verb", func(t *testing.T) {
		r, err := NormalizeDeleteArgs(obj())
		expectThrowsField(t, r, err, "id")
		expectErrorMatches(t, err, `delete several entries`)
		r, err = NormalizeRestoreArgs(obj())
		expectThrowsField(t, r, err, "id")
		expectErrorMatches(t, err, `restore several entries`)
	})
}

func TestAddAndUpdateShareTheFieldTable(t *testing.T) {
	// 手書き18本だった頃、assertUrlWithScheme は**追加側の1箇所でしか呼ばれていなかった**。
	// スキームの無いURLで更新すると gkill はページ取得を試みず、title が空のまま
	// エラーも出さずに保存される。表にして種別の定義を1箇所にした（ADR-0611）。
	t.Run("update_urlog もスキームの無いURLを弾く", func(t *testing.T) {
		_, err := NormalizeUpdateUrlogArgs(obj("id", "u1", "url", "example.com/page"))
		expectErrorMatches(t, err, `must include a scheme`)
	})

	t.Run("add_urlog も同じ文言で弾く", func(t *testing.T) {
		_, err := NormalizeUrlogArgs(obj("url", "example.com/page"))
		expectErrorMatches(t, err, `must include a scheme`)
	})

	t.Run("add と update でURLの検証が一致する", func(t *testing.T) {
		_, addErr := NormalizeUrlogArgs(obj("url", "example.com"))
		_, updateErr := NormalizeUpdateUrlogArgs(obj("id", "u1", "url", "example.com"))
		expectTrue(t, addErr != nil && updateErr != nil, "one side accepted")
		expectEqual(t, addErr.Error(), updateErr.Error())
	})

	t.Run("スキーム付きは add / update とも通る", func(t *testing.T) {
		expectEqual(t, mustNormalize(t)(NormalizeUrlogArgs(obj("url", "https://example.com/p"))).Value("url"), "https://example.com/p")
		expectEqual(t, mustNormalize(t)(NormalizeUpdateUrlogArgs(obj("id", "u1", "url", "https://example.com/p"))).Value("url"), "https://example.com/p")
	})

	// 表にしても patch セマンティクスは変わらない（update は id 以外すべて任意）。
	t.Run("update は id 以外を省略できる", func(t *testing.T) {
		expectEqual(t, mustNormalize(t)(NormalizeUpdateKmemoArgs(obj("id", "k1"))), obj(
			"id", "k1",
			"content", jsonobj.Undefined,
			"related_time", jsonobj.Undefined,
			"locale_name", jsonobj.Undefined,
		))
	})

	// target_id は add でしか受け付けない（付け替えはできない）。
	t.Run("update_tag / update_text は target_id を受け付けない", func(t *testing.T) {
		_, err := NormalizeUpdateTagArgs(obj("id", "t1", "target_id", "k1"))
		expectErrorMatches(t, err, `is not supported`)
		_, err = NormalizeUpdateTextArgs(obj("id", "t1", "target_id", "k1"))
		expectErrorMatches(t, err, `is not supported`)
	})

	// add の必須欄は未指定でもその欄の名前で落ちる（従来の文言のまま）。
	t.Run("add の必須欄は欄名つきで落ちる", func(t *testing.T) {
		_, err := NormalizeKmemoArgs(obj())
		expectErrorMatches(t, err, `'content'`)
		_, err = NormalizeNlogArgs(obj("title", "t"))
		expectErrorMatches(t, err, `'amount'`)
		_, err = NormalizeKcArgs(obj("title", "t"))
		expectErrorMatches(t, err, `'num_value'`)
		_, err = NormalizeTagArgs(obj("tag", "a"))
		expectErrorMatches(t, err, `'target_id'`)
	})

	// 範囲つきの欄は add / update の両方で同じ範囲。
	t.Run("mood の範囲は add / update で同じ", func(t *testing.T) {
		_, err := NormalizeLantanaArgs(obj("mood", 11))
		expectErrorMatches(t, err, `less than or equal to 10`)
		_, err = NormalizeUpdateLantanaArgs(obj("id", "l1", "mood", 11))
		expectErrorMatches(t, err, `less than or equal to 10`)
	})
}

// ---------------------------------------------------------------------------
// 後付けフラグ (MCPレビュー): urlog の fetch_metadata / fetch_favicon、
// mi の allow_create_board。既定値・addOnly・古いスキーマからの文字列復元を固定する。
// ---------------------------------------------------------------------------
func TestLateFlagNormalization(t *testing.T) {
	t.Run("urlog: fetch_metadata / fetch_favicon の既定は true (従来どおり取得)", func(t *testing.T) {
		normalized := mustNormalize(t)(NormalizeUrlogArgs(obj("url", "https://example.com")))
		expectEqual(t, normalized.Value("fetch_metadata"), true)
		expectEqual(t, normalized.Value("fetch_favicon"), true)
	})

	t.Run("urlog: false 指定は素直に通る", func(t *testing.T) {
		normalized := mustNormalize(t)(NormalizeUrlogArgs(obj("url", "https://example.com", "fetch_metadata", false, "fetch_favicon", false)))
		expectEqual(t, normalized.Value("fetch_metadata"), false)
		expectEqual(t, normalized.Value("fetch_favicon"), false)
	})

	t.Run("urlog: fetch_* は add 専用 (update は patch で補完自体が働かないので受けない)", func(t *testing.T) {
		_, err := NormalizeUpdateUrlogArgs(obj("id", "u1", "title", "t", "fetch_metadata", false))
		expectErrorMatches(t, err, `fetch_metadata`)
	})

	t.Run(`urlog: 正規JSON文字列 "false" は boolean へ復元される (古スキーマ救済)`, func(t *testing.T) {
		normalized := mustNormalize(t)(NormalizeUrlogArgs(obj("url", "https://example.com", "fetch_metadata", "false", "fetch_favicon", "true")))
		expectEqual(t, normalized.Value("fetch_metadata"), false)
		expectEqual(t, normalized.Value("fetch_favicon"), true)
	})

	t.Run("urlog: 復元対象外の文字列は従来どおり型エラー", func(t *testing.T) {
		_, err := NormalizeUrlogArgs(obj("url", "https://example.com", "fetch_metadata", "yes"))
		expectErrorMatches(t, err, `boolean`)
	})

	t.Run("urlog: 前後に空白の付いた正規JSON文字列も復元される (検出器と同じ受理範囲)", func(t *testing.T) {
		// 検出器 (DetectStaleSchemaSignals → parseCanonicalJSONValue) は trim してから JSON.parse
		// する。正規化側の受理範囲がそれより狭いと「stale 警告は出るのに型エラーで落ちる」
		// 入力が生まれるので、両者は同じ trim 済み比較で揃える。
		normalized := mustNormalize(t)(NormalizeUrlogArgs(obj("url", "https://example.com", "fetch_metadata", " false ")))
		expectEqual(t, normalized.Value("fetch_metadata"), false)
	})

	t.Run("mi: allow_create_board の既定は add=true / update=undefined (未指定=許可)", func(t *testing.T) {
		expectEqual(t, mustNormalize(t)(NormalizeMiArgs(obj("title", "t"))).Value("allow_create_board"), true)
		expectTrue(t, !mustNormalize(t)(NormalizeUpdateMiArgs(obj("id", "m1", "title", "t"))).Defined("allow_create_board"), "allow_create_board set")
	})

	t.Run("mi: allow_create_board は add / update の両方で false を受ける", func(t *testing.T) {
		expectEqual(t, mustNormalize(t)(NormalizeMiArgs(obj("title", "t", "allow_create_board", false))).Value("allow_create_board"), false)
		expectEqual(t, mustNormalize(t)(NormalizeUpdateMiArgs(obj("id", "m1", "board_name", "b", "allow_create_board", false))).Value("allow_create_board"), false)
	})

	t.Run(`mi: 正規JSON文字列 "false" は boolean へ復元される (古スキーマ救済。add / update とも)`, func(t *testing.T) {
		expectEqual(t, mustNormalize(t)(NormalizeMiArgs(obj("title", "t", "allow_create_board", "false"))).Value("allow_create_board"), false)
		expectEqual(t, mustNormalize(t)(NormalizeUpdateMiArgs(obj("id", "m1", "board_name", "b", "allow_create_board", "false"))).Value("allow_create_board"), false)
	})
}
