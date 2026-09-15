package gkill_server_api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
)

// recoverMiddleware は panic を回収して 500 を返します。
//
// **serve.go では最外層と最内層の両方に登録してあります。** 内側にも要るのは、
// gzipMiddleware の defer gzipWriter.Close() が panic の巻き戻しで先に走ってしまい、
// 空の gzip ストリームを書いて暗黙 200 を確定させるためです。そうなると外側の recover が
// 書く 500 は捨てられ、利用者には **200 + 復号すると空になる本文** が返っていました
// (2026-08 に再現テストで確認)。内側の recover が gzip の Close より先に 500 を書けば、
// 500 + gzip 圧縮された JSON という正しい応答になります。
// 外側を残してあるのは、ミドルウェア自身の panic を拾うためです。
func (g *GkillServerAPI) recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				stack := debug.Stack()
				slog.Log(r.Context(), gkill_log.Error, "panic recovered",
					"panic", fmt.Sprintf("%q", fmt.Sprint(rec)),
					"stack", string(stack),
					"method", r.Method,
					"path", fmt.Sprintf("%q", r.URL.Path),
				)
				// ここは panic の回収中なので、i18n は通さない。
				// MustLocalizeMessage はキーが無いと panic するので、
				// panic ハンドラの中で呼ぶとプロセスごと落ちる。
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]any{
					"errors": []map[string]any{
						{"error_code": message.InternalServerPanicError, "error_message": "内部エラーが発生しました", "error_kind": message.ErrorKindServer},
					},
					"messages": []any{},
				})
			}
		}()
		next.ServeHTTP(w, r)
	})
}
