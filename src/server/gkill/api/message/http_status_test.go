package message

import (
	"net/http"
	"testing"
)

// エラーコードごとの HTTP ステータス割り当ては、
// 「新しいコードを足したのに分類し忘れる」と静かに壊れる。
//
// 分類漏れのコードは HTTPStatusOf が 0 を返し、HTTPStatusForErrors はそれを 500 として
// 扱うので、目の前ではエラーにならない。「500 で返ってはいるが、本当は 404 だった」という
// ずれだけが残り、呼び出し側は「サーバ障害」と「指定ミス」を取り違え続ける。
//
// そこで error_codes.go をソース走査して、全定数が表に載っていることを機械的に確認する。
// message_test.go の parseConstStrings を使い回しているので、
// 定数を足せばテスト側を触らずに検査対象へ入る。

// TestHTTPStatusOf_CoversEveryErrorCode は error_codes.go の全定数が
// errorCodeHTTPStatus に載っていることを確認する。
//
// **落ちたら、エラーコードを足したのに分類していない。**
// http_status.go の該当ステータスの節へ1行足すこと。迷ったら 500。
func TestHTTPStatusOf_CoversEveryErrorCode(t *testing.T) {
	consts := parseConstStrings(t, "error_codes.go")

	allowed := map[int]bool{}
	for _, status := range statusPriority {
		allowed[status] = true
	}

	missing := []string{}
	for name, code := range consts {
		status := HTTPStatusOf(code)
		if status == 0 {
			missing = append(missing, name+" ("+code+")")
			continue
		}
		if !allowed[status] {
			t.Errorf("%s (%s) に statusPriority に無いステータス %d が割り当てられている", name, code, status)
		}
	}
	for _, name := range missing {
		t.Errorf("%s が errorCodeHTTPStatus に無い。http_status.go へ分類を足すこと", name)
	}

	// 表にあってソースに無いコード（定数を消したのに表を消し忘れた）も拾う。
	byCode := map[string]bool{}
	for _, code := range consts {
		byCode[code] = true
	}
	for code := range errorCodeHTTPStatus {
		if !byCode[code] {
			t.Errorf("errorCodeHTTPStatus に error_codes.go へ無いコード %q が残っている", code)
		}
	}
}

// TestErrorCodeHTTPStatus_Distribution はステータスごとの件数を固定する。
//
// 網羅テストだけだと「とりあえず 500 にしておく」で全部通ってしまうので、
// 内訳が動いたら気付けるようにしてある。資料（message/README.md・
// documents/reverse/error-handling-and-security.md）に書いた数と同じ。
// **コードを足して落ちたら、期待値と資料の両方を更新すること。**
func TestErrorCodeHTTPStatus_Distribution(t *testing.T) {
	want := map[int]int{
		http.StatusBadRequest:          103,
		http.StatusUnauthorized:        4,
		http.StatusForbidden:           12,
		http.StatusNotFound:            18,
		http.StatusConflict:            16,
		http.StatusTooManyRequests:     1,
		http.StatusInternalServerError: 260,
	}

	got := map[int]int{}
	for _, status := range errorCodeHTTPStatus {
		got[status]++
	}

	total := 0
	for status, wantCount := range want {
		if got[status] != wantCount {
			t.Errorf("ステータス %d の件数 = %d, want %d", status, got[status], wantCount)
		}
		total += wantCount
	}
	if len(errorCodeHTTPStatus) != total {
		t.Errorf("表の総数 = %d, want %d", len(errorCodeHTTPStatus), total)
	}
}

// TestHTTPStatusOf_KnownAssignments は「取り違えると実害が出る」割り当てを名指しで固定する。
//
// どれも名前の規則からは導けないもの。規則ベースへ書き換えようとした人がここで止まる。
func TestHTTPStatusOf_KnownAssignments(t *testing.T) {
	cases := []struct {
		code string
		name string
		want int
		why  string
	}{
		{AccountSessionNotFoundError, "AccountSessionNotFoundError", http.StatusUnauthorized, "名前は NotFound だが認証の失敗"},
		{AccountSessionExpiredError, "AccountSessionExpiredError", http.StatusUnauthorized, "セッション期限切れ"},
		{AccountNotFoundError, "AccountNotFoundError", http.StatusUnauthorized, "自分のセッションのアカウントが消えている"},
		{TargetAccountNotFoundError, "TargetAccountNotFoundError", http.StatusNotFound, "操作対象が居ない。AccountNotFoundError と混ぜない"},
		{AccountNotHasAdminError, "AccountNotHasAdminError", http.StatusForbidden, "権限不足"},
		{AccountDisabledError, "AccountDisabledError", http.StatusForbidden, "無効化済み"},
		{LocalOnlyAccessDeniedError, "LocalOnlyAccessDeniedError", http.StatusForbidden, "ローカル限定アクセス違反"},
		{GetIDFFilePathNotLocalRequestError, "GetIDFFilePathNotLocalRequestError", http.StatusForbidden, "同上（別経路）"},
		{LoginRateLimitError, "LoginRateLimitError", http.StatusTooManyRequests, "レート制限"},
		{AlreadyExistKmemoError, "AlreadyExistKmemoError", http.StatusConflict, "同じIDが既にある"},
		{NotFoundTLSCertFileError, "NotFoundTLSCertFileError", http.StatusInternalServerError, "名前は NotFound だがサーバの設定不備"},
		{NotFoundTLSKeyFileError, "NotFoundTLSKeyFileError", http.StatusInternalServerError, "同上"},
		{InvalidStatusGetRepNameError, "InvalidStatusGetRepNameError", http.StatusInternalServerError, "名前は Invalid だが GetRepName の失敗"},
		{InvalidGetMiSharedTaskRequest, "InvalidGetMiSharedTaskRequest", http.StatusInternalServerError, "保存済み共有設定のJSONが壊れている"},
		{AccountInvalidAddKmemoRequestDataError, "AccountInvalidAddKmemoRequestDataError", http.StatusBadRequest, "リクエストJSONのパース失敗"},
		{FindKyousError, "FindKyousError", http.StatusInternalServerError, "検索の失敗"},
		{InternalServerPanicError, "InternalServerPanicError", http.StatusInternalServerError, "panic 回収"},
	}

	for _, c := range cases {
		if got := HTTPStatusOf(c.code); got != c.want {
			t.Errorf("HTTPStatusOf(%s=%s) = %d, want %d（%s）", c.name, c.code, got, c.want, c.why)
		}
	}
}

// TestHTTPStatusForErrors はエラー配列からステータスを決める部分の挙動を固定する。
func TestHTTPStatusForErrors(t *testing.T) {
	e := func(code string) *GkillError { return &GkillError{ErrorCode: code} }

	cases := []struct {
		name string
		errs []*GkillError
		want int
	}{
		{"nil は 200", nil, http.StatusOK},
		{"空は 200", []*GkillError{}, http.StatusOK},
		{"nil要素だけなら 200", []*GkillError{nil}, http.StatusOK},
		{"1件ならそのまま", []*GkillError{e(AccountNotHasAdminError)}, http.StatusForbidden},
		{"500 は 4xx より優先", []*GkillError{e(AccountInvalidAddKmemoRequestDataError), e(AddKmemoError)}, http.StatusInternalServerError},
		{"401 は 403 より優先", []*GkillError{e(AccountNotHasAdminError), e(AccountSessionExpiredError)}, http.StatusUnauthorized},
		{"403 は 404 より優先", []*GkillError{e(NotFoundKmemoError), e(AccountNotHasAdminError)}, http.StatusForbidden},
		{"409 は 400 より優先", []*GkillError{e(AccountInvalidAddKmemoRequestDataError), e(AlreadyExistKmemoError)}, http.StatusConflict},
		{"順序を入れ替えても同じ", []*GkillError{e(AlreadyExistKmemoError), e(AccountInvalidAddKmemoRequestDataError)}, http.StatusConflict},
		{"未知のコードは 500 扱い", []*GkillError{e("ERR999999")}, http.StatusInternalServerError},
		{"nil要素は飛ばす", []*GkillError{nil, e(NotFoundKmemoError)}, http.StatusNotFound},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := HTTPStatusForErrors(c.errs); got != c.want {
				t.Errorf("HTTPStatusForErrors = %d, want %d", got, c.want)
			}
		})
	}
}
