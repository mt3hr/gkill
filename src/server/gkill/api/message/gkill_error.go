package message

import "encoding/json"

type GkillError struct {
	ErrorCode string `json:"error_code"`

	ErrorMessage string `json:"error_message"`
}

// MarshalJSON はレスポンスへ出る直前に ErrorMessage から端末固有の情報を伏せます。
//
// **伏せるのは生成側ではなくここです。** ErrorMessage の代入は684箇所あり、
// そのうち err.Error() を埋めているのは4箇所ですが、将来のハンドラが同じ書き方を
// したときに自動で載ることが唯一の再発防止になります。生成側で1つずつ包む方式は
// 必ず足し忘れます（詳細は documents/adr/0046-redact-environment-specific-strings.md）。
//
// 値レシーバなので、レスポンスが持つ []*GkillError からでも呼ばれます。
// 受け取る側（CLIのHTTPクライアント）は従来どおりなので UnmarshalJSON は足しません。
func (e GkillError) MarshalJSON() ([]byte, error) {
	// メソッドを引き継がない別の型にしてから包む。そうしないと再帰する。
	type gkillErrorJSON GkillError
	return json.Marshal(gkillErrorJSON{
		ErrorCode:    e.ErrorCode,
		ErrorMessage: RedactEnvironmentSpecific(e.ErrorMessage),
	})
}

// EnsureNotEmpty は「失敗したのにGkillErrorが1つも無い」状態を潰します。
//
// gkillErrorsが空でなければそのまま返し、空のときだけ指定のエラーを1件足します。
//
// 内部のerrorだけがあってGkillErrorが無いまま返すと、レスポンスは
// HTTP 200 + errors:null + 結果0件になり、**呼び出し側からは
// 「成功・該当0件」と区別が付きません**。内部のerrorはDebugログにしか出ないので、
// 通常の運用ログにも残らず、静かに検索結果が消えます。
// 2026-08-18に /api/get_kyous_mcp と /api/get_kyous で実際に踏みました
// (query.idsを6553件以上渡すとMi検索のバインド変数がSQLiteの上限を超えて失敗し、
// エラーではなく空の結果に見えていた)。
//
// localizedMessageは訳し済みの文言を渡してください
// (messageパッケージはapiパッケージをimportできないため)。
func EnsureNotEmpty(gkillErrors []*GkillError, errorCode string, localizedMessage string) []*GkillError {
	if len(gkillErrors) != 0 {
		return gkillErrors
	}
	return append(gkillErrors, &GkillError{
		ErrorCode:    errorCode,
		ErrorMessage: localizedMessage,
	})
}
