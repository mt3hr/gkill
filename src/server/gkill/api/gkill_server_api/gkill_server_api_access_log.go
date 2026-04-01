package gkill_server_api

import (
	"context"
	"log/slog"
	"net/http"
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
type accessLogInfo struct {
	UserID string
}

func newAccessLogContext(ctx context.Context) (context.Context, *accessLogInfo) {
	info := &accessLogInfo{}
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

// ---------------------------------------------------------------------------
// Middleware
// ---------------------------------------------------------------------------

func (g *GkillServerAPI) accessLogMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		ctx, info := newAccessLogContext(r.Context())
		rec := newResponseRecorder(w)

		next.ServeHTTP(rec, r.WithContext(ctx))

		slog.Log(ctx, gkill_log.Access, "access",
			"remote_addr", extractIP(r.RemoteAddr),
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.statusCode,
			"duration", time.Since(start).String(),
			"user_id", info.UserID,
		)
	})
}
