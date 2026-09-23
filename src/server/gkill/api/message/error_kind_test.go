package message

import (
	"net/http"
	"testing"
)

// error_kind はクライアントがヒント文（何をすれば直るか）を引く鍵。
// エラーコードごとに必ず1つ決まり、語彙は errorKinds の中に閉じていることを固定する。

// TestKindOf_CoversEveryErrorCode は error_codes.go の全定数に kind が決まり、
// それが errorKinds の語彙に入っていることを確認する。
func TestKindOf_CoversEveryErrorCode(t *testing.T) {
	consts := parseConstStrings(t, "error_codes.go")

	allowed := map[string]bool{}
	for _, kind := range errorKinds {
		allowed[kind] = true
	}

	for name, code := range consts {
		kind := KindOf(code)
		if kind == "" {
			t.Errorf("%s (%s) の kind が空", name, code)
			continue
		}
		if !allowed[kind] {
			t.Errorf("%s (%s) の kind %q が errorKinds に無い", name, code, kind)
		}
	}
}

// TestKindOf_OverrideCodesExist は上書き表のコードが実在することを確認する
// （定数を消したのに表を消し忘れると、静かに効かない上書きが残る）。
func TestKindOf_OverrideCodesExist(t *testing.T) {
	consts := parseConstStrings(t, "error_codes.go")
	byCode := map[string]bool{}
	for _, code := range consts {
		byCode[code] = true
	}
	for code, kind := range errorCodeKindOverride {
		if !byCode[code] {
			t.Errorf("errorCodeKindOverride に error_codes.go へ無いコード %q が残っている", code)
		}
		if kind != ErrorKindConfig {
			// 今の上書きは「500 だが設定で直せる」だけ。別の用途で使い始めたらこのテストを更新する。
			t.Errorf("errorCodeKindOverride[%q] = %q。上書きは config だけのはず", code, kind)
		}
		if HTTPStatusOf(code) != http.StatusInternalServerError {
			t.Errorf("errorCodeKindOverride[%q] は 500 のコードではない（status=%d）。config は 500 の細分化", code, HTTPStatusOf(code))
		}
	}
	for code := range errorCodeReason {
		if !byCode[code] {
			t.Errorf("errorCodeReason に error_codes.go へ無いコード %q が残っている", code)
		}
	}
}

// TestKindOf_KnownAssignments は「取り違えると案内が逆になる」割り当てを名指しで固定する。
func TestKindOf_KnownAssignments(t *testing.T) {
	cases := []struct {
		code string
		name string
		want string
		why  string
	}{
		{AccountInvalidAddKmemoRequestDataError, "AccountInvalidAddKmemoRequestDataError", ErrorKindInput, "400"},
		{AccountSessionExpiredError, "AccountSessionExpiredError", ErrorKindAuth, "401"},
		{AccountNotHasAdminError, "AccountNotHasAdminError", ErrorKindPermission, "403"},
		{LocalOnlyAccessDeniedError, "LocalOnlyAccessDeniedError", ErrorKindPermission, "403。reason=local_only_access が案内を担う"},
		{NotFoundKmemoError, "NotFoundKmemoError", ErrorKindNotFound, "404"},
		{AlreadyExistKmemoError, "AlreadyExistKmemoError", ErrorKindConflict, "409"},
		{RequestBodyTooLargeError, "RequestBodyTooLargeError", ErrorKindTooLarge, "413"},
		{LoginRateLimitError, "LoginRateLimitError", ErrorKindRateLimit, "429"},
		{NotFoundTLSCertFileError, "NotFoundTLSCertFileError", ErrorKindConfig, "500 だがサーバ設定の不備"},
		{WriteRepMissingError, "WriteRepMissingError", ErrorKindConfig, "500 だが保存先設定の不備"},
		{AddKmemoError, "AddKmemoError", ErrorKindServer, "500"},
		{InternalServerPanicError, "InternalServerPanicError", ErrorKindServer, "panic 回収"},
		{"ERR999999", "未知のコード", ErrorKindServer, "HTTPStatusForErrors の未分類=500 と揃える"},
	}
	for _, c := range cases {
		if got := KindOf(c.code); got != c.want {
			t.Errorf("KindOf(%s=%s) = %q, want %q（%s）", c.name, c.code, got, c.want, c.why)
		}
	}
}

// TestErrorKinds_Distribution は kind ごとの件数を固定する（資料の表と同じ数）。
// **コードを足して落ちたら、期待値と資料の両方を更新すること。**
func TestErrorKinds_Distribution(t *testing.T) {
	consts := parseConstStrings(t, "error_codes.go")
	got := map[string]int{}
	for _, code := range consts {
		got[KindOf(code)]++
	}
	want := map[string]int{
		ErrorKindInput:      107,
		ErrorKindAuth:       4,
		ErrorKindPermission: 6,
		ErrorKindNotFound:   20,
		ErrorKindConflict:   19,
		ErrorKindTooLarge:   1,
		ErrorKindRateLimit:  1,
		ErrorKindConfig:     3,
		ErrorKindServer:     242,
	}
	for kind, wantCount := range want {
		if got[kind] != wantCount {
			t.Errorf("kind %q の件数 = %d, want %d", kind, got[kind], wantCount)
		}
	}
	if len(errorKinds) != len(want) {
		t.Errorf("errorKinds の語彙数 = %d, want %d", len(errorKinds), len(want))
	}
}
