package gkill_log

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_options"
)

// TestInitNamedUsesPrefixForEveryFile は InitNamed が接頭辞で統合ファイルとレベル別ファイルの全部を開き、
// 静的フィールド app と追加フィールドを全行に付けることを固定する。
//
// MCP サブコマンドはこの経路で logs/gkill_mcp_<kind>*.log へ書く。接頭辞の付け忘れが1ファイルでもあると、
// gkill 本体の gkill_error.log へ MCP の行が混ざり、どちらの障害か読めなくなる。
func TestInitNamedUsesPrefixForEveryFile(t *testing.T) {
	tmpDir := t.TempDir()
	originalLogDir := gkill_options.LogDir
	originalLevel := LogLevelFromCmd
	originalRouter := router
	originalLogger := slog.Default()

	gkill_options.LogDir = tmpDir
	LogLevelFromCmd = "trace_sql"
	InitNamed("gkill_mcp_read", "gkill_mcp", "kind", "read")
	initializedRouter := router
	t.Cleanup(func() {
		slog.SetDefault(originalLogger)
		_ = initializedRouter.Close()
		gkill_options.LogDir = originalLogDir
		LogLevelFromCmd = originalLevel
		router = originalRouter
	})

	levels := []struct {
		level  slog.Level
		suffix string
	}{
		{TraceSQL, "_trace_sql.log"},
		{Trace, "_trace.log"},
		{Debug, "_debug.log"},
		{Access, "_access.log"},
		{Info, "_info.log"},
		{Warn, "_warn.log"},
		{Error, "_error.log"},
	}
	for _, entry := range levels {
		slog.Log(t.Context(), entry.level, "named routing test", "suffix", entry.suffix)
	}
	if err := Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	for _, entry := range levels {
		name := "gkill_mcp_read" + entry.suffix
		data, err := os.ReadFile(filepath.Join(tmpDir, name))
		if err != nil {
			t.Fatalf("ReadFile(%s) failed: %v", name, err)
		}
		content := string(data)
		if !strings.Contains(content, "named routing test") {
			t.Errorf("%s にログ本文が無い: %s", name, content)
		}
		if !strings.Contains(content, `"app":"gkill_mcp"`) || !strings.Contains(content, `"kind":"read"`) {
			t.Errorf("%s に静的フィールドが無い: %s", name, content)
		}
	}
	merged, err := os.ReadFile(filepath.Join(tmpDir, "gkill_mcp_read.log"))
	if err != nil {
		t.Fatalf("ReadFile(gkill_mcp_read.log) failed: %v", err)
	}
	if got := strings.Count(string(merged), "named routing test"); got != len(levels) {
		t.Errorf("統合ファイルの行数 = %d, want %d", got, len(levels))
	}
	// 本体の gkill*.log は1つも作られない（接頭辞の付け忘れの検出）。
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), "gkill_mcp_read") {
			t.Errorf("接頭辞の無いファイルが作られた: %s", entry.Name())
		}
	}
}

// TestCloseWithoutInitIsNoop は Init 前の Close が何もしないことを固定する（テストと早期終了の経路）。
func TestCloseWithoutInitIsNoop(t *testing.T) {
	originalRouter := router
	router = nil
	t.Cleanup(func() { router = originalRouter })
	if err := Close(); err != nil {
		t.Fatalf("Close before Init: %v", err)
	}
}
