package message

import "net/http"

// error_kind（誰の問題か）はエラーコードから決める。
//
// HTTP ステータス（http_status.go の表）を粗い分類として使い、同じ 500 の中で
// 「利用者が設定画面で直せる不備」だけを config として切り出す。
// 名前の規則からは導かない（ADR-0706 と同じ理由。Invalid* が 400 と 500 に跨る）。
//
// 消費者は kind をヒント文の既定に使う（Web: error-hints.ts、MCP: formatErrors）。
// reason（何が起きたか）が付いていればそちらが優先される。

// error_kind の語彙。順序は資料（documents/reverse/error-handling-and-security.md）の表と同じ。
const (
	// ErrorKindInput は 400。入力・送信内容の誤り。画面から操作していて出るならアプリの更新が要る。
	ErrorKindInput = "input"
	// ErrorKindAuth は 401。ログインし直す（Web の check_auth がログイン画面へ飛ばす）。
	ErrorKindAuth = "auth"
	// ErrorKindPermission は 403。認証は通っているが許可されない（管理者権限・無効化・ローカル限定）。
	ErrorKindPermission = "permission"
	// ErrorKindNotFound は 404。対象の記録が無い（別の端末で削除・更新された可能性）。
	ErrorKindNotFound = "not_found"
	// ErrorKindConflict は 409。同じ記録が既にある（二重送信）。
	ErrorKindConflict = "conflict"
	// ErrorKindTooLarge は 413。送信データが大きすぎる。
	ErrorKindTooLarge = "too_large"
	// ErrorKindRateLimit は 429。しばらく待つ。
	ErrorKindRateLimit = "rate_limit"
	// ErrorKindConfig は 500 のうち、サーバ設定・保存先設定の不備で、利用者が設定画面や CLI で直せるもの。
	ErrorKindConfig = "config"
	// ErrorKindServer は 500。サーバ側の障害。reason が付いていればそれが次の一手を示す。
	ErrorKindServer = "server"
)

// errorKinds は error_kind の語彙の一覧（資料の件数検査と、テストの網羅確認に使う）。
var errorKinds = []string{
	ErrorKindInput,
	ErrorKindAuth,
	ErrorKindPermission,
	ErrorKindNotFound,
	ErrorKindConflict,
	ErrorKindTooLarge,
	ErrorKindRateLimit,
	ErrorKindConfig,
	ErrorKindServer,
}

// errorCodeKindOverride はステータスから導く既定の kind を上書きするコード。
//
// 500 のうち「サーバのプロセスや DB の障害ではなく、設定を直せば消える」ものを config にする。
// 利用者に「サーバ側の問題です。ログを確認してください」ではなく「設定画面のここを直す」と
// 案内するため。**足すのは設定画面か CLI で利用者自身が直せるものだけ。**
var errorCodeKindOverride = map[string]string{
	NotFoundTLSCertFileError: ErrorKindConfig, // ERR000346 サーバ設定の TLS 証明書ファイルが無い
	NotFoundTLSKeyFileError:  ErrorKindConfig, // ERR000347 同 秘密鍵
	WriteRepMissingError:     ErrorKindConfig, // ERR000422 その種別の書き込み先 rep が未設定
}

// kindForHTTPStatus はステータスから既定の kind を引く。
func kindForHTTPStatus(status int) string {
	switch status {
	case http.StatusBadRequest:
		return ErrorKindInput
	case http.StatusUnauthorized:
		return ErrorKindAuth
	case http.StatusForbidden:
		return ErrorKindPermission
	case http.StatusNotFound:
		return ErrorKindNotFound
	case http.StatusConflict:
		return ErrorKindConflict
	case http.StatusRequestEntityTooLarge:
		return ErrorKindTooLarge
	case http.StatusTooManyRequests:
		return ErrorKindRateLimit
	default:
		// 500 と、表に無いコード（HTTPStatusForErrors が 500 として扱うのと揃える）
		return ErrorKindServer
	}
}

// KindOf は GkillError のエラーコードに対応する error_kind を返します。
//
// 表に無いコードは server（HTTPStatusForErrors が未分類を 500 として扱うのと同じ）。
// 空文字を返すことはありません。
func KindOf(errorCode string) string {
	if kind, ok := errorCodeKindOverride[errorCode]; ok {
		return kind
	}
	return kindForHTTPStatus(HTTPStatusOf(errorCode))
}
