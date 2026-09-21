package gkill_server_api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
)

// ---------------------------------------------------------------------------
// Access log context — pointer pattern
// ---------------------------------------------------------------------------

type accessLogContextKeyType struct{}

var accessLogContextKey = accessLogContextKeyType{}

// accessLogInfo is stored as a pointer in the request context.
// The middleware creates it before calling next.ServeHTTP.
// Handlers (via getAccountFromSessionIDWithApplicationName) write UserID into it.
//
// Method / Path はミドルウェアが立てます。writeErrorStatus が「どのAPIが失敗したか」を
// 1行に載せるために読みます（あそこには *http.Request が無い）。
type accessLogInfo struct {
	UserID string
	Method string
	Path   string
}

func newAccessLogContext(ctx context.Context, method string, path string) (context.Context, *accessLogInfo) {
	info := &accessLogInfo{Method: method, Path: path}
	return context.WithValue(ctx, accessLogContextKey, info), info
}

func accessLogInfoFromContext(ctx context.Context) *accessLogInfo {
	info, _ := ctx.Value(accessLogContextKey).(*accessLogInfo)
	return info
}

// ---------------------------------------------------------------------------
// responseRecorder — captures the HTTP status code
// ---------------------------------------------------------------------------

type responseRecorder struct {
	http.ResponseWriter
	statusCode int
}

func newResponseRecorder(w http.ResponseWriter) *responseRecorder {
	return &responseRecorder{ResponseWriter: w, statusCode: http.StatusOK}
}

func (rr *responseRecorder) WriteHeader(code int) {
	rr.statusCode = code
	rr.ResponseWriter.WriteHeader(code)
}

// Unwrap は http.ResponseController がラッパ越しに SetReadDeadline 等へ届くための口です。
// これが無いと wrapNoAuthCapped の読み取り期限が本番の全経路で静かに効かなくなります
// (auth_middleware_capped_test.go が検査)。
func (rr *responseRecorder) Unwrap() http.ResponseWriter {
	return rr.ResponseWriter
}

// 静的検査用: Unwrap を満たしていること。http.ResponseController が Unwrap の鎖で
// 底の *http.response へ届く必要があり、欠けると読み取り期限が本番経路でだけ静かに無効になる。
var _ interface{ Unwrap() http.ResponseWriter } = (*responseRecorder)(nil)

// ---------------------------------------------------------------------------
// Middleware
// ---------------------------------------------------------------------------

func (g *GkillServerAPI) accessLogMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		ctx, info := newAccessLogContext(r.Context(), r.Method, r.URL.Path)
		rec := newResponseRecorder(w)

		next.ServeHTTP(rec, r.WithContext(ctx))

		// 利用者由来の値（接続元・メソッド・利用者ID）は改行を落としてから載せる（ログ注入対策）。
		// path は %q で引用する。どちらも「この関数の中で」行うこと: ヘルパ関数に包むと
		// 静的解析（CodeQL go/log-injection）がサニタイザと認識せず、同じ指摘が再発する。
		remoteAddr := strings.ReplaceAll(strings.ReplaceAll(extractIP(r.RemoteAddr), "\r", ""), "\n", "")
		method := strings.ReplaceAll(strings.ReplaceAll(r.Method, "\r", ""), "\n", "")
		userID := strings.ReplaceAll(strings.ReplaceAll(info.UserID, "\r", ""), "\n", "")
		slog.Log(ctx, gkill_log.Access, "access",
			"remote_addr", remoteAddr,
			"method", method,
			"path", fmt.Sprintf("%q", r.URL.Path),
			"status", rec.statusCode,
			"duration", time.Since(start).String(),
			"user_id", userID,
		)
	})
}
