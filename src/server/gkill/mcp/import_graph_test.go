package mcp

// package mcp の import 境界。
//
// MCP は起動中の gkill_server への HTTP クライアントであって、本体の API・DAO を直接は使わない
// （gkill-cli-ops スキル）。gkill/api を import すると embed 無しの go test が動かなくなり、
// 循環 import の温床にもなる。ここでソースを走査して固定する。

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMCPDoesNotImportServerAPI(t *testing.T) {
	forbidden := []string{
		"github.com/mt3hr/gkill/src/server/gkill/api",
		"github.com/mt3hr/gkill/src/server/gkill/dao",
		"github.com/mt3hr/gkill/src/server/gkill/usecase",
		"github.com/mt3hr/gkill/src/server/gkill/main/common\"",
	}
	entries, err := os.ReadDir(".")
	expectNoError(t, err)
	fset := token.NewFileSet()
	checked := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		file, err := parser.ParseFile(fset, entry.Name(), nil, parser.ImportsOnly)
		expectNoError(t, err)
		checked++
		for _, imp := range file.Imports {
			path := imp.Path.Value
			for _, bad := range forbidden {
				if strings.Contains(path, bad) {
					t.Errorf("%s imports %s (mcp must stay an HTTP client of gkill_server)", entry.Name(), path)
				}
			}
		}
	}
	expectTrue(t, checked > 30, "only %d files checked", checked)
	// gkill_log は許す（ログの出口）。それ以外の main/common 配下は import しない。
	_ = filepath.Join
}
