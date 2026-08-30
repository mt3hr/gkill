package gkill_log

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_options"
)

func TestLevelNameReturnsCorrectNames(t *testing.T) {
	tests := []struct {
		level slog.Level
		want  string
	}{
		{TraceSQL, "TRACE_SQL"},
		{Trace, "TRACE"},
		{Debug, "DEBUG"},
		{Info, "INFO"},
		{Warn, "WARN"},
		{Error, "ERROR"},
		{None, "NONE"},
	}
	for _, tt := range tests {
		got := LevelName(tt.level)
		if got != tt.want {
			t.Errorf("LevelName(%v) = %q, want %q", tt.level, got, tt.want)
		}
	}
}

func TestLevelNameNoneAndAbove(t *testing.T) {
	// Any level >= None should return "NONE".
	got := LevelName(None + 50)
	if got != "NONE" {
		t.Errorf("LevelName(None+50) = %q, want %q", got, "NONE")
	}
}

func TestLevelConstants_Order(t *testing.T) {
	// TraceSQL < Trace < Debug < Info < Warn < Error < None
	levels := []slog.Level{TraceSQL, Trace, Debug, Info, Warn, Error, None}
	for i := 1; i < len(levels); i++ {
		if levels[i] <= levels[i-1] {
			t.Errorf("expected level[%d](%v) > level[%d](%v)",
				i, levels[i], i-1, levels[i-1])
		}
	}
}

func TestSplitModeConstants(t *testing.T) {
	if SplitOnly != 0 {
		t.Errorf("SplitOnly = %d, want 0", SplitOnly)
	}
	if MergedOnly != 1 {
		t.Errorf("MergedOnly = %d, want 1", MergedOnly)
	}
	if MergedAndSplit != 2 {
		t.Errorf("MergedAndSplit = %d, want 2", MergedAndSplit)
	}
}

func TestNewRouterReturnsNonNil(t *testing.T) {
	r := NewRouter(Options{
		MinLevel: Info,
	})
	if r == nil {
		t.Fatal("NewRouter returned nil")
	}
	if r.Logger() == nil {
		t.Fatal("Router.Logger() returned nil")
	}
}

func TestNewRouterDefaultTimeFormat(t *testing.T) {
	r := NewRouter(Options{
		MinLevel: Info,
	})
	// When TimeFormat is empty, the router should set a default.
	if r.opts.TimeFormat == "" {
		t.Error("expected default TimeFormat to be set, got empty string")
	}
}

func TestSetMinLevel(t *testing.T) {
	r := NewRouter(Options{
		MinLevel: Info,
	})
	r.SetMinLevel(Debug)
	if r.level.Level() != Debug {
		t.Errorf("expected level Debug(%v), got %v", Debug, r.level.Level())
	}
}

func TestSetMode(t *testing.T) {
	r := NewRouter(Options{
		MinLevel: Info,
		Mode:     SplitOnly,
	})
	r.SetMode(MergedAndSplit)
	r.lock()
	got := r.opts.Mode
	r.unlock()
	if got != MergedAndSplit {
		t.Errorf("expected mode MergedAndSplit, got %v", got)
	}
}

func TestSetSplitFileAndLogOutput(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test_info.log")

	r := NewRouter(Options{
		JSON:     true,
		MinLevel: Info,
		Mode:     SplitOnly,
	})

	err := r.SetSplitFile(Info, logPath)
	if err != nil {
		t.Fatalf("SetSplitFile failed: %v", err)
	}

	// Log a message at Info level.
	r.Logger().Info("test message", "key", "value")

	// Close the sink so the file handle is released before TempDir cleanup.
	r.byLevel[Info].Close()

	// Read the log file and verify it contains the message.
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}
	content := string(data)
	if len(content) == 0 {
		t.Error("expected log file to have content, got empty")
	}
}

func TestSetSplitFileUnknownLevel(t *testing.T) {
	r := NewRouter(Options{
		MinLevel: Info,
	})
	// Use a level that is not in the byLevel map.
	err := r.SetSplitFile(slog.Level(999), filepath.Join(t.TempDir(), "x.log"))
	if err == nil {
		t.Error("expected error for unknown level, got nil")
	}
}

func TestSetMergedFile(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "merged.log")

	r := NewRouter(Options{
		JSON:     true,
		MinLevel: Info,
		Mode:     MergedOnly,
	})

	err := r.SetMergedFile(logPath)
	if err != nil {
		t.Fatalf("SetMergedFile failed: %v", err)
	}

	r.Logger().Warn("merged warning")

	// Close the merged sink so the file handle is released before TempDir cleanup.
	r.merged.Close()

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read merged log: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected merged log to have content")
	}
}

// Init の実配線で、統合ログとレベル別ログの両方へ同じ行が出ることを確認する。
// Router単体が正しくても Init が SplitOnly に戻ると gkill.log は常に空になる。
func TestInitWritesMergedAndSplitLogs(t *testing.T) {
	tmpDir := t.TempDir()
	originalLogDir := gkill_options.LogDir
	originalLevel := LogLevelFromCmd
	originalRouter := router
	originalLogger := slog.Default()

	gkill_options.LogDir = tmpDir
	LogLevelFromCmd = "info"
	Init()
	initializedRouter := router
	t.Cleanup(func() {
		slog.SetDefault(originalLogger)
		for _, sink := range initializedRouter.byLevel {
			_ = sink.Close()
		}
		_ = initializedRouter.merged.Close()
		gkill_options.LogDir = originalLogDir
		LogLevelFromCmd = originalLevel
		router = originalRouter
	})

	slog.Info("init routing test", "test_key", "test_value")
	_ = initializedRouter.byLevel[Info].Close()
	_ = initializedRouter.merged.Close()

	for _, name := range []string{"gkill_info.log", "gkill.log"} {
		data, err := os.ReadFile(filepath.Join(tmpDir, name))
		if err != nil {
			t.Fatalf("ReadFile(%s) failed: %v", name, err)
		}
		content := string(data)
		if !strings.Contains(content, "init routing test") {
			t.Errorf("%s にログ本文が無い: %s", name, content)
		}
		if !strings.Contains(content, `"app":"gkill"`) {
			t.Errorf("%s に静的フィールドが無い: %s", name, content)
		}
	}
}

// TestStaticFieldsAreEmitted は Logger().With(...) で足した静的フィールドが
// 実際に出力へ載ることを確認する。
//
// **落ちたら routingHandler.WithAttrs が属性を捨てている。**
// 以前は WithAttrs/WithGroup が `return h` の空実装で、資料に「{"app":"gkill"} が付く」と
// 書いてあるのに1行も出ていなかった。leaf handler は Record ごとに作り直すので、
// 修飾もそのたびに掛け直す必要がある。
func TestStaticFieldsAreEmitted(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "static_fields.log")

	r := NewRouter(Options{
		JSON:         true,
		MinLevel:     Info,
		Mode:         MergedOnly,
		StaticFields: []any{"app", "gkill"},
	})
	if err := r.SetMergedFile(logPath); err != nil {
		t.Fatalf("SetMergedFile failed: %v", err)
	}

	r.Logger().Info("with static fields")
	r.merged.Close()

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}
	if !strings.Contains(string(data), `"app":"gkill"`) {
		t.Errorf("静的フィールドが出力に無い。routingHandler.WithAttrs が属性を捨てている: %s", string(data))
	}
}

// TestWithGroupIsApplied は WithGroup も leaf handler へ伝わることを確認する。
func TestWithGroupIsApplied(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "with_group.log")

	r := NewRouter(Options{JSON: true, MinLevel: Info, Mode: MergedOnly})
	if err := r.SetMergedFile(logPath); err != nil {
		t.Fatalf("SetMergedFile failed: %v", err)
	}

	r.Logger().WithGroup("g").Info("grouped", "key", "value")
	r.merged.Close()

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}
	if !strings.Contains(string(data), `"g":{"key":"value"}`) {
		t.Errorf("WithGroup が leaf handler へ伝わっていない: %s", string(data))
	}
}

// TestLogRotationKeepsGenerations はサイズ上限を超えたときに世代が回ることを確認する。
//
// **Windowsでは開いたままのファイルをリネームできない。** 回転の実装が Close を先に
// 行っていないと、ここが Windows でだけ落ちる（ファイルは上限なく育ち続ける）。
func TestLogRotationKeepsGenerations(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "rotate.log")

	const maxBytes = 512
	r := NewRouter(Options{
		JSON:           true,
		MinLevel:       Info,
		Mode:           SplitOnly,
		RotateMaxBytes: maxBytes,
		RotateKeep:     2,
	})
	if err := r.SetSplitFile(Info, logPath); err != nil {
		t.Fatalf("SetSplitFile failed: %v", err)
	}

	for i := range 40 {
		r.Logger().Info("rotate me", "index", i, "padding", strings.Repeat("x", 64))
	}
	r.byLevel[Info].Close()

	stat, err := os.Stat(logPath)
	if err != nil {
		t.Fatalf("stat %s: %v", logPath, err)
	}
	if stat.Size() > maxBytes {
		t.Errorf("回転していない。%s が %d バイト（上限 %d）", logPath, stat.Size(), maxBytes)
	}
	if _, err := os.Stat(logPath + ".1"); err != nil {
		t.Errorf("退避した世代 %s.1 が無い: %v", logPath, err)
	}
	// keep=2 なので .3 は残らない
	if _, err := os.Stat(logPath + ".3"); err == nil {
		t.Errorf("keep=2 なのに %s.3 が残っている", logPath)
	}
}

// TestNoRotationWhenDisabled は RotateMaxBytes が0のとき回転しないことを確認する。
func TestNoRotationWhenDisabled(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "no_rotate.log")

	r := NewRouter(Options{JSON: true, MinLevel: Info, Mode: SplitOnly})
	if err := r.SetSplitFile(Info, logPath); err != nil {
		t.Fatalf("SetSplitFile failed: %v", err)
	}
	for i := range 40 {
		r.Logger().Info("no rotate", "index", i, "padding", strings.Repeat("x", 64))
	}
	r.byLevel[Info].Close()

	if _, err := os.Stat(logPath + ".1"); err == nil {
		t.Errorf("回転を無効にしているのに %s.1 ができている", logPath)
	}
}
