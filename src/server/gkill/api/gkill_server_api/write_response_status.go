package gkill_server_api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
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
//
// **失敗したリクエストの1行を残すのもここです。** 深部のエラーログは Debug に置いてあり、
// 既定のログレベルでは1行も出ません。そのままだと「500 が返るが理由がどこにも出ない」に
// なるので、応答を書くこの1箇所で、ステータスから決まるレベルの要約を出します
// （5xx=Error / 401・403・429=Warn / その他の4xx=Debug）。
// ctx を取るのはそのためです。呼び出し側は r.Context() を渡してください。
func writeErrorStatus(ctx context.Context, w http.ResponseWriter, errs []*message.GkillError) {
	status := message.HTTPStatusForErrors(errs)
	if status == http.StatusOK {
		return
	}
	logResponseFailure(ctx, status, errs)
	w.WriteHeader(status)
}

// logResponseFailure は失敗した応答の要約を1行出します。
//
// レベルの根拠はステータスです。
//   - 5xx はサーバ側の障害。運用者がいま知るべきなので Error（gkill_error.log）。
//   - 401 / 403 / 429 は利用者・攻撃者由来だが記録に残すので Warn。
//   - それ以外の 4xx は入力の誤りで、直す先が利用者側にあるので Debug。
//
// エラーコードを載せるのは、これが「どのAPIがどの理由で落ちたか」を1行で示す唯一の行に
// なるためです。自由文の error_message は載せません（i18n 済みの利用者向け文面で、
// 原因の特定には使えない）。代わりに **reason（分類）と cause（GkillError.Cause の文面）** を載せます。
// 2026-09-15 まで原因は深部の Debug ログにしかなく、既定のログレベル（error）では
// 「500 が返るが理由がどこにも出ない」状態でした。cause はサーバのログなので伏せません
// （ADR-0707: 伏せるのはレスポンスだけ）。
func logResponseFailure(ctx context.Context, status int, errs []*message.GkillError) {
	level := gkill_log.Debug
	switch {
	case status >= http.StatusInternalServerError:
		level = gkill_log.Error
	case status == http.StatusUnauthorized, status == http.StatusForbidden, status == http.StatusTooManyRequests:
		level = gkill_log.Warn
	}
	// 呼び出し側が中断した（Web の AbortController 等。reason=canceled）だけの 500 はサーバの障害ではない。
	// 検索欄を打ち直すたびに出るので、Error のままだと gkill_error.log が利用者の操作で埋まる。
	if level == gkill_log.Error && allErrorsCanceled(errs) {
		level = gkill_log.Debug
	}

	// レベルが無効なら組み立てない。Go は引数を呼び出し前に評価するので、
	// ここを素に書くと既定レベル(error)でも 4xx のたびに文字列を作って捨てることになる
	// (gkill_log.TraceSQLEnabled と同じ趣旨)。
	if !slog.Default().Enabled(ctx, level) {
		return
	}

	errorCodes := make([]string, 0, len(errs))
	reasons := make([]string, 0, len(errs))
	causes := make([]string, 0, len(errs))
	for _, gkillError := range errs {
		if gkillError == nil {
			continue
		}
		errorCodes = append(errorCodes, gkillError.ErrorCode)
		if reason := message.ReasonOf(gkillError.Cause); reason != "" {
			reasons = append(reasons, reason)
		}
		if gkillError.Cause != nil {
			causes = append(causes, gkillError.Cause.Error())
		}
	}

	args := []any{"status", status, "error_codes", fmt.Sprintf("%q", errorCodes)}
	if len(reasons) != 0 {
		args = append(args, "reasons", fmt.Sprintf("%q", reasons))
	}
	if len(causes) != 0 {
		args = append(args, "causes", fmt.Sprintf("%q", causes))
	}
	if info := accessLogInfoFromContext(ctx); info != nil {
		args = append(args,
			"method", fmt.Sprintf("%q", info.Method),
			"path", fmt.Sprintf("%q", info.Path),
			"user_id", fmt.Sprintf("%q", info.UserID))
	}
	slog.Log(ctx, level, "request failed", args...)
}

// writeGkillErrorResponse は「レスポンス構造体を持たない経路」用に、
// ステータスと {"errors":[...]} の本文をまとめて書きます。
//
// 認証ミドルウェアとローカル限定アクセスのフィルタが使います。どちらもハンドラより手前で
// 打ち切るので response 構造体が無く、以前は本文だけ(または本文すら無しで)返していました。
// **本文を必ず JSON で返すのが重要です** —— クライアント(gkill-api.ts)は
// ステータスを見ずに res.json() するので、本文が空だとそこで例外になり、
// ログイン画面では「証明書が必要です」という無関係な文言が出ます。
func writeGkillErrorResponse(ctx context.Context, w http.ResponseWriter, gkillError *message.GkillError) {
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
	writeErrorStatus(ctx, w, errs)
	// messages も載せる（常に [] の契約。無いと消費者が undefined を吸収することになる）
	_ = json.NewEncoder(w).Encode(struct {
		Messages message.GkillMessages `json:"messages"`
		Errors   message.GkillErrors   `json:"errors"`
	}{Errors: errs})
}

// allErrorsCanceled は errs の全件が「呼び出し側の中断」に分類されるときだけ true。
// 1件でも中断以外（または Cause 無し）が混ざっていれば false。
func allErrorsCanceled(errs []*message.GkillError) bool {
	found := false
	for _, gkillError := range errs {
		if gkillError == nil {
			continue
		}
		if message.ReasonOf(gkillError.Cause) != message.ReasonCanceled {
			return false
		}
		found = true
	}
	return found
}
