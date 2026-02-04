package gkill_log

import (
	"log/slog"
)

var (
	TraceSQL slog.Level = Trace - 4
	Trace    slog.Level = slog.LevelDebug - 4
	Debug    slog.Level = slog.LevelDebug
	Info     slog.Level = slog.LevelInfo
	Warn     slog.Level = slog.LevelWarn
	Error    slog.Level = slog.LevelError
	None     slog.Level = slog.LevelError + 100
)

func LevelName(l slog.Level) string {
	switch {
	case l == TraceSQL:
		return "TRACE_SQL"
	case l == Trace:
		return "TRACE"
	case l == Debug:
		return "DEBUG"
	case l == Info:
		return "INFO"
	case l == Warn:
		return "WARN"
	case l == Error:
		return "ERROR"
	case l >= None:
		return "NONE"
	default:
		return l.String()
	}
}
