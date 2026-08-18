package message

import "testing"

// EnsureNotEmpty は「失敗したのにGkillErrorが1つも無い」状態を潰すためのもの。
//
// 内部のerrorだけを持って返すと、レスポンスは HTTP 200 + errors:null + 0件になり、
// 呼び出し側からは「成功・該当0件」と区別が付かない。
// 2026-08-18に /api/get_kyous_mcp で実際に踏んだ（IDを6553件以上渡すと
// Mi検索のバインド変数がSQLiteの上限を超えて失敗し、空の結果に見えていた）。
func TestEnsureNotEmptyAddsErrorWhenNone(t *testing.T) {
	got := EnsureNotEmpty(nil, "ERR000410", "検索に失敗しました")
	if len(got) != 1 {
		t.Fatalf("エラー = %d件, want 1件（失敗が呼び出し側へ伝わらない）", len(got))
	}
	if got[0].ErrorCode != "ERR000410" {
		t.Errorf("error_code = %q, want %q", got[0].ErrorCode, "ERR000410")
	}
	if got[0].ErrorMessage != "検索に失敗しました" {
		t.Errorf("error_message = %q, want %q", got[0].ErrorMessage, "検索に失敗しました")
	}
}

func TestEnsureNotEmptyKeepsExistingErrors(t *testing.T) {
	original := []*GkillError{{ErrorCode: "ERR000001", ErrorMessage: "元のエラー"}}

	got := EnsureNotEmpty(original, "ERR000410", "検索に失敗しました")

	// 既にあるものを置き換えたり増やしたりしない。
	// 元のエラーのほうが原因に近いので、こちらを残す。
	if len(got) != 1 {
		t.Fatalf("エラー = %d件, want 1件", len(got))
	}
	if got[0].ErrorCode != "ERR000001" {
		t.Errorf("error_code = %q, want %q（元のエラーが失われている）", got[0].ErrorCode, "ERR000001")
	}
}
