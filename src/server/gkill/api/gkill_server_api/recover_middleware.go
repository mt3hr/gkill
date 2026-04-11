package gkill_server_api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
)

func (g *GkillServerAPI) recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				stack := debug.Stack()
				slog.Log(r.Context(), gkill_log.Error, "panic recovered",
					"panic", fmt.Sprintf("%v", rec),
					"stack", string(stack),
					"method", r.Method,
					"path", r.URL.Path,
				)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]any{
					"errors": []map[string]any{
						{"error_code": "INTERNAL_SERVER_ERROR", "error_message": "内部エラーが発生しました"},
					},
					"messages": []any{},
				})
			}
		}()
		next.ServeHTTP(w, r)
	})
}
