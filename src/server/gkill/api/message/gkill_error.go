package message

import "encoding/json"

// GkillError はAPIレスポンスの errors 配列の1要素。
//
// 利用者へ見せるのは ErrorCode と ErrorMessage で、これに加えて marshal 時に
// 「種類」(error_kind。誰の問題か) と「理由」(reason。何が起きたか) を機械語のトークンで載せる。
// 文言の粒度は操作単位（「メモ追加に失敗しました」）のまま変えず、
// 何をすれば直るかは種類と理由から消費者側（Web の error-hints.ts 等）が引く。
// 経緯と却下案は documents/adr/0710-error-kind-and-reason-on-the-wire.md。
type GkillError struct {
	ErrorCode string `json:"error_code"`

	ErrorMessage string `json:"error_message"`

	// Cause は ErrorMessage の裏にある Go の error。JSON には出さない。
	//
	// **`if err != nil` の中で GkillError を組み立てるときは必ず `Cause: err` を付けること。**
	// ここから ReasonOf が理由(reason)を分類し、writeErrorStatus が gkill_error.log の
	// 1行に cause を載せる。付け忘れは gkill_error_cause_scan_test.go（ソース走査）が落とす。
	// 利用者向けの文言に err.Error() を混ぜる代わりにここへ入れる（文言は伏せ処理を通るが、
	// 端末固有のパスは形が残るので、自由文には載せないほうがよい）。
	Cause error `json:"-"`
}

// gkillErrorJSON はワイヤに出る形。GkillError からメソッドを引き継がない別の型にしてある。
type gkillErrorJSON struct {
	ErrorCode    string `json:"error_code"`
	ErrorMessage string `json:"error_message"`
	// ErrorKind は誰の問題か（input / auth / permission / not_found / conflict / too_large /
	// rate_limit / config / server）。エラーコードから KindOf が表で決める。必ず載る。
	ErrorKind string `json:"error_kind"`
	// Reason は何が起きたか（write_rep_missing / storage_unavailable / db_busy / timeout ...）。
	// Cause から ReasonOf が分類する。分類できなければ省略。
	Reason string `json:"reason,omitempty"`
}

// MarshalJSON はレスポンスへ出る直前に ErrorMessage から端末固有の情報を伏せ、
// error_kind と reason を付けます。
//
// **伏せるのは生成側ではなくここです。** ErrorMessage の代入は684箇所あり、
// そのうち err.Error() を埋めているのは4箇所ですが、将来のハンドラが同じ書き方を
// したときに自動で載ることが唯一の再発防止になります。生成側で1つずつ包む方式は
// 必ず足し忘れます（詳細は documents/adr/0707-redact-environment-specific-strings.md）。
// error_kind / reason も同じ理由でここで付けます —— 生成側の633箇所は従来どおり
// ErrorCode と ErrorMessage（と Cause）だけを書けばよい。
//
// 値レシーバなので、レスポンスが持つ []*GkillError からでも呼ばれます。
// 受け取る側（CLIのHTTPクライアント）は従来どおりなので UnmarshalJSON は足しません
// （未知のキーは encoding/json が読み飛ばす）。
func (e GkillError) MarshalJSON() ([]byte, error) {
	return json.Marshal(gkillErrorJSON{
		ErrorCode:    e.ErrorCode,
		ErrorMessage: RedactEnvironmentSpecific(e.ErrorMessage),
		ErrorKind:    KindOf(e.ErrorCode),
		Reason:       reasonForError(e.ErrorCode, e.Cause),
	})
}

// GkillErrors はレスポンス構造体の Errors フィールドの型。
//
// **成功時（nil）でも JSON では `[]` になる。** 2026-09-15 まで `[]*GkillError` のまま
// `json:"errors"`（omitempty 無し）で、ハンドラは失敗時だけ append するので成功時は
// `null` になっていた。消費者（Web 約180箇所・MCP・CLI・Wear）がそれぞれ null を吸収する
// 状態は「サーバの偶然の実装詳細を全クライアントが防御している」形なので、境界で `[]` に揃える。
// Web 側の `res.errors ?? []` / `&& length` のガードは**残す**（古い PWA が新旧どちらのサーバと
// 話しても壊れないため。書き換える理由は無い）。
type GkillErrors []*GkillError

// MarshalJSON は nil を `[]` として出します。要素の伏せ処理は各 GkillError の MarshalJSON が行います。
func (errs GkillErrors) MarshalJSON() ([]byte, error) {
	if errs == nil {
		return []byte("[]"), nil
	}
	return json.Marshal([]*GkillError(errs))
}

// EnsureNotEmpty は「失敗したのにGkillErrorが1つも無い」状態を潰します。
//
// gkillErrorsが空でなければそのまま返し、空のときだけ指定のエラーを1件足します。
//
// 内部のerrorだけがあってGkillErrorが無いまま返すと、レスポンスは
// HTTP 200 + errors:[] + 結果0件になり、**呼び出し側からは
// 「成功・該当0件」と区別が付きません**。内部のerrorはDebugログにしか出ないので、
// 通常の運用ログにも残らず、静かに検索結果が消えます。
// 2026-08-18に /api/get_kyous_mcp と /api/get_kyous で実際に踏みました
// (query.idsを6553件以上渡すとMi検索のバインド変数がSQLiteの上限を超えて失敗し、
// エラーではなく空の結果に見えていた)。
//
// localizedMessageは訳し済みの文言を渡してください
// (messageパッケージはapiパッケージをimportできないため)。
// cause には手元の err を渡してください（reason の分類とログの1行に使う。nil 可）。
func EnsureNotEmpty(gkillErrors []*GkillError, errorCode string, localizedMessage string, cause error) []*GkillError {
	if len(gkillErrors) != 0 {
		return gkillErrors
	}
	return append(gkillErrors, &GkillError{
		ErrorCode:    errorCode,
		ErrorMessage: localizedMessage,
		Cause:        cause,
	})
}
