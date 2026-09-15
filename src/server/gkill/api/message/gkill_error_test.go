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
	got := EnsureNotEmpty(nil, "ERR000410", "検索に失敗しました", nil)
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

	got := EnsureNotEmpty(original, "ERR000410", "検索に失敗しました", nil)

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

// ワイヤの形（2026-09-15、ADR-0710）: errors / messages は成功時も []、
// 各エラーには error_kind が必ず、reason は分類できたときだけ載る。Cause は出ない。

func TestGkillErrorMarshalJSONAddsKindAndReason(t *testing.T) {
	gkillError := &GkillError{
		ErrorCode:    AddKmemoError,
		ErrorMessage: "メモ追加に失敗しました",
		Cause:        &ReasonError{Msg: "write repository for kmemo is not configured", Reason: ReasonWriteRepMissing},
	}
	encoded, err := json.Marshal(gkillError)
	if err != nil {
		t.Fatal(err)
	}
	got := string(encoded)
	want := `{"error_code":"ERR000023","error_message":"メモ追加に失敗しました","error_kind":"server","reason":"write_rep_missing"}`
	if got != want {
		t.Errorf("JSON = %s\nwant   %s", got, want)
	}
	if strings.Contains(got, "not configured") {
		t.Errorf("Cause の文面がレスポンスへ漏れている: %s", got)
	}
}

func TestGkillErrorMarshalJSONOmitsReasonWhenUnknown(t *testing.T) {
	encoded, err := json.Marshal(&GkillError{ErrorCode: AlreadyExistKmemoError, ErrorMessage: "x"})
	if err != nil {
		t.Fatal(err)
	}
	got := string(encoded)
	if strings.Contains(got, `"reason"`) {
		t.Errorf("分類できないのに reason が出ている: %s", got)
	}
	if !strings.Contains(got, `"error_kind":"conflict"`) {
		t.Errorf("error_kind が無い（必ず載る）: %s", got)
	}
}

// 成功時の errors / messages は null ではなく []。
// 消費者（Web 約180箇所・MCP・CLI・Wear）がそれぞれ null を吸収する必要が無いように境界で揃える。
func TestGkillErrorsAndMessagesMarshalNilAsEmptyArray(t *testing.T) {
	type response struct {
		Messages GkillMessages `json:"messages"`
		Errors   GkillErrors   `json:"errors"`
	}
	encoded, err := json.Marshal(&response{})
	if err != nil {
		t.Fatal(err)
	}
	if got := string(encoded); got != `{"messages":[],"errors":[]}` {
		t.Errorf("成功時の形 = %s, want {\"messages\":[],\"errors\":[]}", got)
	}

	// 要素があるときは各要素の MarshalJSON（伏せ処理・kind）が効く
	withErrors := &response{
		Errors:   GkillErrors{{ErrorCode: NotFoundKmemoError, ErrorMessage: `C:\Users\username\x`}},
		Messages: GkillMessages{{MessageCode: AddKmemoSuccessMessage, Message: "ok"}},
	}
	encoded, err = json.Marshal(withErrors)
	if err != nil {
		t.Fatal(err)
	}
	got := string(encoded)
	if strings.Contains(got, "username") || !strings.Contains(got, `"error_kind":"not_found"`) {
		t.Errorf("要素の MarshalJSON が効いていない: %s", got)
	}
	if !strings.Contains(got, `"level":"info"`) {
		t.Errorf("GkillMessage の level 既定が info になっていない: %s", got)
	}

	// 受け取る側（CLI の HTTP クライアント）は従来どおり読める
	var decoded response
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded.Errors) != 1 || decoded.Errors[0].ErrorCode != NotFoundKmemoError {
		t.Errorf("復号結果が違う: %+v", decoded.Errors)
	}
}

func TestGkillMessageMarshalJSONKeepsWarningLevel(t *testing.T) {
	encoded, err := json.Marshal(&GkillMessage{MessageCode: FindKyousRepLoadWarningMessage, Message: "x", Level: MessageLevelWarning})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"level":"warning"`) {
		t.Errorf("warning が info に潰れている: %s", encoded)
	}
}
