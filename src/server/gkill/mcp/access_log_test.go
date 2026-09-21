package mcp

// access_log.go（Logger / ParseMcpLogLevel）の検査。
//
// Node 版（McpAccessLog）は自前でファイルを開いて JSON 行を書いていたので、
// テストもファイルの生成・close・再オープンまで見ていた。Go 版はファイルの寿命を
// gkill_log（Router）が持ち、ここは slog への薄い包みなので、
//   - 「lazy open がディレクトリを作る」「close で fd を離す・多重 close」は
//     gkill_log_test.go（InitNamed / Router.Close）側の検査に移した
//   - 「未知のレベルは info へ落ちる」は「未知のレベルは拒否する」に変えた（gkill_server と同じ）
//   - 「none ならファイルを作らない」は「none なら1行も書かない」に変えた
// それ以外の観点（JSON 行の欄・レベル名・レベルの絞り込み・文脈欄）は同じ。

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/mt3hr/gkill/src/server/gkill/mcp/jsonobj"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// newFileLogger は gkill_log の Router を一時ファイルへ向けた Logger を返す。
func newFileLogger(t *testing.T, minLevel slog.Level) (*Logger, string, *gkill_log.Router) {
	t.Helper()
	logPath := filepath.Join(t.TempDir(), "gkill_mcp_test.log")
	router := gkill_log.NewRouter(gkill_log.Options{
		JSON:         true,
		AddSource:    true,
		MinLevel:     minLevel,
		Mode:         gkill_log.MergedOnly,
		StaticFields: []any{"app", "gkill_mcp"},
	})
	if err := router.SetMergedFile(logPath); err != nil {
		t.Fatalf("SetMergedFile: %v", err)
	}
	t.Cleanup(func() { _ = router.Close() })
	return NewLogger(router.Logger()), logPath, router
}

func readLines(t *testing.T, logPath string) []*jsonobj.Object {
	t.Helper()
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read %s: %v", logPath, err)
	}
	lines := []*jsonobj.Object{}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		lines = append(lines, parseObj(t, line))
	}
	return lines
}

func logMessages(lines []*jsonobj.Object) []string {
	out := []string{}
	for _, line := range lines {
		msg, _ := line.String("msg")
		out = append(out, msg)
	}
	return out
}

// ---------------------------------------------------------------------------
// parseMcpLogLevel
// ---------------------------------------------------------------------------

func TestParseMcpLogLevel(t *testing.T) {
	t.Run("returns known levels as-is", func(t *testing.T) {
		expectations := map[string]slog.Level{
			"none":  gkill_log.None,
			"error": gkill_log.Error,
			"warn":  gkill_log.Warn,
			"info":  gkill_log.Info,
			"debug": gkill_log.Debug,
			"trace": gkill_log.Trace,
			// gkill_log の語彙で Node に無かったもの
			"access":    gkill_log.Access,
			"trace_sql": gkill_log.TraceSQL,
		}
		for name, want := range expectations {
			got, err := ParseMcpLogLevel(name)
			expectNoError(t, err)
			expectTrue(t, got == want, "%s: got %v want %v", name, got, want)
		}
	})

	t.Run("is case-insensitive", func(t *testing.T) {
		got, err := ParseMcpLogLevel("INFO")
		expectNoError(t, err)
		expectTrue(t, got == gkill_log.Info, "INFO -> %v", got)
		got, err = ParseMcpLogLevel("WARN")
		expectNoError(t, err)
		expectTrue(t, got == gkill_log.Warn, "WARN -> %v", got)
	})

	// Node は未知の値を黙って info へ落としていた。Go は gkill_server の --log と同じく起動を止める。
	// 空（未指定）だけが既定（access）になる。
	t.Run("rejects unknown values instead of falling back, and treats empty as the default access", func(t *testing.T) {
		_, err := ParseMcpLogLevel("unknown")
		expectErrorContains(t, err, "invalid MCP log level")
		got, err := ParseMcpLogLevel("")
		expectNoError(t, err)
		expectTrue(t, got == gkill_log.Access, "empty -> %v", got)
		expectEqual(t, DefaultMcpLogLevelName, "access")
	})
}

// ---------------------------------------------------------------------------
// Logger — JSON format
// ---------------------------------------------------------------------------

func TestLoggerJSONFormat(t *testing.T) {
	t.Run("writes valid JSON with required fields", func(t *testing.T) {
		logger, logPath, router := newFileLogger(t, gkill_log.Trace)
		logger.Info("test_msg", "event", "test", "key", "value")
		_ = router.Close()
		lines := readLines(t, logPath)
		expectTrue(t, len(lines) == 1, "%d lines", len(lines))
		entry := lines[0]
		expectTrue(t, entry.Defined("time"), "time missing")
		expectEqual(t, entry.Value("level"), "INFO")
		// source は gkill_log の形（{function, file, line}）。呼び出し元＝このテストファイルを指す
		source := objAt(t, entry, "source")
		mustContain(t, strAt(t, source, "file"), "access_log_test.go")
		expectEqual(t, entry.Value("msg"), "test_msg")
		expectEqual(t, entry.Value("app"), "gkill_mcp")
		expectEqual(t, entry.Value("event"), "test")
		expectEqual(t, entry.Value("key"), "value")
	})

	t.Run("time field is ISO 8601", func(t *testing.T) {
		logger, logPath, router := newFileLogger(t, gkill_log.Trace)
		logger.Info("ts_test")
		_ = router.Close()
		entry := readLines(t, logPath)[0]
		_, err := time.Parse(time.RFC3339Nano, strAt(t, entry, "time"))
		expectNoError(t, err)
	})

	t.Run("writes correct level names", func(t *testing.T) {
		logger, logPath, router := newFileLogger(t, gkill_log.Trace)
		logger.Error("e")
		logger.Warn("w")
		logger.Info("i")
		logger.Access("a")
		logger.Debug("d")
		logger.Trace("t")
		_ = router.Close()
		levels := []string{}
		for _, line := range readLines(t, logPath) {
			levels = append(levels, strAt(t, line, "level"))
		}
		expectEqual(t, levels, []string{"ERROR", "WARN", "INFO", "ACCESS", "DEBUG", "TRACE"})
	})
}

// ---------------------------------------------------------------------------
// Logger — level filtering
// ---------------------------------------------------------------------------

func TestLoggerLevelFiltering(t *testing.T) {
	emitAll := func(logger *Logger) {
		logger.Trace("t")
		logger.Debug("d")
		logger.Access("a")
		logger.Info("i")
		logger.Warn("w")
		logger.Error("e")
	}

	t.Run("info level filters out debug and trace", func(t *testing.T) {
		logger, logPath, router := newFileLogger(t, gkill_log.Info)
		emitAll(logger)
		_ = router.Close()
		expectEqual(t, logMessages(readLines(t, logPath)), []string{"i", "w", "e"})
	})

	t.Run("access level keeps access and filters out debug and trace", func(t *testing.T) {
		logger, logPath, router := newFileLogger(t, gkill_log.Access)
		emitAll(logger)
		_ = router.Close()
		expectEqual(t, logMessages(readLines(t, logPath)), []string{"a", "i", "w", "e"})
	})

	t.Run("warn level filters out info, debug, trace", func(t *testing.T) {
		logger, logPath, router := newFileLogger(t, gkill_log.Warn)
		emitAll(logger)
		_ = router.Close()
		expectEqual(t, logMessages(readLines(t, logPath)), []string{"w", "e"})
	})

	t.Run("error level only includes error", func(t *testing.T) {
		logger, logPath, router := newFileLogger(t, gkill_log.Error)
		emitAll(logger)
		_ = router.Close()
		expectEqual(t, logMessages(readLines(t, logPath)), []string{"e"})
	})
}

// ---------------------------------------------------------------------------
// Logger — none level (nothing written)
// ---------------------------------------------------------------------------

func TestLoggerNoneLevel(t *testing.T) {
	t.Run("does not write anything when level is none", func(t *testing.T) {
		logger, logPath, router := newFileLogger(t, gkill_log.None)
		logger.Info("should_not_appear")
		logger.Error("should_not_appear_either")
		_ = router.Close()
		expectTrue(t, len(readLines(t, logPath)) == 0, "lines were written at level none")
	})
}

// ---------------------------------------------------------------------------
// Logger — nil safety（Node の accessLog 省略時の { info() {}, ... } に相当）
// ---------------------------------------------------------------------------

func TestLoggerNil(t *testing.T) {
	t.Run("a nil Logger and a Logger without a slog.Logger accept every level without panicking", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("nil-safe Logger panicked: %v", r)
			}
		}()
		var nilLogger *Logger
		for _, logger := range []*Logger{nilLogger, NewLogger(nil), {}} {
			logger.Error("e")
			logger.Warn("w")
			logger.Info("i")
			logger.Access("a")
			logger.Debug("d")
			logger.Trace("t")
		}
	})
}

// ---------------------------------------------------------------------------
// Logger — context fields
// ---------------------------------------------------------------------------

func TestLoggerContextFields(t *testing.T) {
	t.Run("includes all context fields in output", func(t *testing.T) {
		logger, logPath, router := newFileLogger(t, gkill_log.Info)
		logger.Info("auth_success",
			"event", "auth_success",
			"user_id", "testuser",
			"remote_addr", "192.0.2.10",
		)
		_ = router.Close()
		entry := readLines(t, logPath)[0]
		expectEqual(t, entry.Value("event"), "auth_success")
		expectEqual(t, entry.Value("user_id"), "testuser")
		expectEqual(t, entry.Value("remote_addr"), "192.0.2.10")
	})
}
