package message

import (
	"encoding/json"
	"strings"
	"testing"
)

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

// ErrorMessage に err.Error() を埋めているハンドラが4箇所ある。
// そこには起動に失敗したプラグインの実行ファイルの絶対パスが乗るので、
// レスポンスへ出る時点で伏せられていないと、AIの文脈へ端末固有のパスが流れ込む。
// 生成側ではなく marshal 側で伏せるのは、将来のハンドラが同じ書き方をしても
// 自動で載るようにするため。
func TestGkillErrorMarshalJSONRedactsEnvironmentSpecific(t *testing.T) {
	gkillError := &GkillError{
		ErrorCode:    "ERR000401",
		ErrorMessage: `プラグインコンテンツHTMLの取得に失敗しました: fork/exec C:\Users\username\gkill\plugins\admin\p\p.exe: not found`,
	}

	encoded, err := json.Marshal(gkillError)
	if err != nil {
		t.Fatalf("json.Marshal に失敗した: %v", err)
	}

	got := string(encoded)
	if strings.Contains(got, "username") {
		t.Errorf("ユーザー名がレスポンスに残っている: %s", got)
	}
	if !strings.Contains(got, "〈ユーザー名〉") {
		t.Errorf("プレースホルダへ置き換わっていない: %s", got)
	}
	// error_code は伏せない。ステータス判定に使う値なので変えてはいけない。
	if !strings.Contains(got, `"error_code":"ERR000401"`) {
		t.Errorf("error_code が壊れている: %s", got)
	}
}
