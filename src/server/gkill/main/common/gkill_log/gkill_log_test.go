package gkill_log

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"
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
