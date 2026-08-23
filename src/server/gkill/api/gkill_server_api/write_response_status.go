package gkill_server_api

import (
	"encoding/json"
	"net/http"

	"github.com/mt3hr/gkill/src/server/gkill/api/message"
)

// writeErrorStatus は response.Errors の内容から決まる HTTP ステータスを書きます。
//
// **必ず json.NewEncoder(w).Encode(response) より前に呼ぶこと。**
// net/http は本文が1バイトでも書かれた時点で 200 を確定させるので、あとから
// WriteHeader を呼んでも "superfluous response.WriteHeader" がログに出るだけで
// ステータスは 200 のまま返ります。errors 配列には正しくエラーが入っていて
// 画面も普段どおり動くため、**壊れたことに気付けません**。
//
// エラーが無いとき(＝200)は WriteHeader を呼びません。呼ぶとそこでヘッダが確定し、
// 以降の w.Header().Set() が無視されるためです。200 は暗黙のままにしておきます。
//
// ステータスの割り当ての正本は message.HTTPStatusOf / message.HTTPStatusForErrors。
func writeErrorStatus(w http.ResponseWriter, errs []*message.GkillError) {
	status := message.HTTPStatusForErrors(errs)
	if status == http.StatusOK {
		return
	}
	w.WriteHeader(status)
}

// writeGkillErrorResponse は「レスポンス構造体を持たない経路」用に、
// ステータスと {"errors":[...]} の本文をまとめて書きます。
//
// 認証ミドルウェアとローカル限定アクセスのフィルタが使います。どちらもハンドラより手前で
// 打ち切るので response 構造体が無く、以前は本文だけ(または本文すら無しで)返していました。
// **本文を必ず JSON で返すのが重要です** —— クライアント(gkill-api.ts)は
// ステータスを見ずに res.json() するので、本文が空だとそこで例外になり、
// ログイン画面では「証明書が必要です」という無関係な文言が出ます。
func writeGkillErrorResponse(w http.ResponseWriter, gkillError *message.GkillError) {
	w.Header().Set("Content-Type", "application/json")
	if gkillError == nil {
		// 呼び出し側が「失敗したのにGkillErrorを作らないまま」ここへ来た場合の受け皿。
		// nilのまま積むと errors:[null] になり、ステータスも200に戻って
		// 呼び出し側からは成功と区別が付かなくなる(message.EnsureNotEmpty と同じ趣旨)。
		gkillError = &message.GkillError{
			ErrorCode:    message.InternalServerPanicError,
			ErrorMessage: "internal server error",
		}
	}
	errs := []*message.GkillError{gkillError}
	writeErrorStatus(w, errs)
	_ = json.NewEncoder(w).Encode(struct {
		Errors []*message.GkillError `json:"errors"`
	}{Errors: errs})
}
