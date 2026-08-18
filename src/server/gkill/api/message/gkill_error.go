package message

type GkillError struct {
	ErrorCode string `json:"error_code"`

	ErrorMessage string `json:"error_message"`
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
