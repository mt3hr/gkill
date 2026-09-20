package mcp

// MCP のログ出力（旧 access-log.mjs）。
//
// Node 版は自前で JSON 行を書いていたが、Go 版は gkill_log（slog）へ流す薄い包み。
// 呼び口（Info("tool_call", "tool", name, ...)）は Node の log.info("tool_call", {...}) と同じ形を保つ。
// ファイルを開くのは gkill_log.InitNamed（main/common/mcp.go）で、ここでは開かない。
//
// レベルは gkill_log の語彙（none / error / warn / info / access / debug / trace / trace_sql）。
// Node の MCP_LOG は「不明な値を黙って info へ落とす」だったが、Go は gkill_server と同じく
// 起動を止める（ParseMcpLogLevel がエラーを返す）。既定は access —— Node の既定 info で
// 見えていた http_request / tool_call を、Go では ACCESS レベルで出すため。
//
// stdout へは書かない: stdio モードの stdout は JSON-RPC 専用で、1行でも混ざると
// クライアントが全ツールを黙って失敗させる。

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"strings"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
)

// DefaultMcpLogLevelName は MCP_LOG / 設定 log_level / --log の既定。
const DefaultMcpLogLevelName = "access"

// McpLogLevelNames は受理するレベル名（エラー文の並び）。
var McpLogLevelNames = []string{"none", "error", "warn", "info", "access", "debug", "trace", "trace_sql"}

var mcpLogLevels = map[string]slog.Level{
	"none":      gkill_log.None,
	"error":     gkill_log.Error,
	"warn":      gkill_log.Warn,
	"info":      gkill_log.Info,
	"access":    gkill_log.Access,
	"debug":     gkill_log.Debug,
	"trace":     gkill_log.Trace,
	"trace_sql": gkill_log.TraceSQL,
}

// ParseMcpLogLevel はレベル名を gkill_log のレベルにする。大小は無視、空は既定（access）。
// 未知の値はエラー（黙って既定へ落とさない）。
func ParseMcpLogLevel(str string) (slog.Level, error) {
	s := strings.ToLower(strings.TrimSpace(str))
	if s == "" {
		s = DefaultMcpLogLevelName
	}
	level, ok := mcpLogLevels[s]
	if !ok {
		return 0, fmt.Errorf("invalid MCP log level %q. log level [%s]", str, strings.Join(McpLogLevelNames, ", "))
	}
	return level, nil
}

// Logger は slog の薄い包み。nil でも呼べる（何も出さない）。
type Logger struct {
	l *slog.Logger
}

// NewLogger は slog.Logger を包む。nil を渡すと slog.Default() を使う。
func NewLogger(l *slog.Logger) *Logger {
	if l == nil {
		l = slog.Default()
	}
	return &Logger{l: l}
}

// log は呼び出し元の位置（source）を保ったまま1件出す。
// slog.Logger.Log を経由すると source がこのファイルになるので、Record を自分で組む。
func (g *Logger) log(level slog.Level, msg string, kv ...any) {
	if g == nil || g.l == nil {
		return
	}
	ctx := context.Background()
	handler := g.l.Handler()
	if !handler.Enabled(ctx, level) {
		return
	}
	var pcs [1]uintptr
	// runtime.Callers, log, Info/Warn/... の3段を飛ばして呼び出し元
	runtime.Callers(3, pcs[:])
	record := slog.NewRecord(time.Now(), level, msg, pcs[0])
	record.Add(kv...)
	_ = handler.Handle(ctx, record)
}

// Error は障害（運用者が見るべきもの）。
func (g *Logger) Error(msg string, kv ...any) { g.log(gkill_log.Error, msg, kv...) }

// Warn は利用者側の誤り・拒否など、常態化しうるもの。
func (g *Logger) Warn(msg string, kv ...any) { g.log(gkill_log.Warn, msg, kv...) }

// Info は起動・接続など。
func (g *Logger) Info(msg string, kv ...any) { g.log(gkill_log.Info, msg, kv...) }

// Access は要求1件ごとの記録（http_request / tool_call）。
func (g *Logger) Access(msg string, kv ...any) { g.log(gkill_log.Access, msg, kv...) }

// Debug は診断用。
func (g *Logger) Debug(msg string, kv ...any) { g.log(gkill_log.Debug, msg, kv...) }

// Trace は最も細かい記録。
func (g *Logger) Trace(msg string, kv ...any) { g.log(gkill_log.Trace, msg, kv...) }
