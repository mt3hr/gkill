package mcp

// 3つの MCP サーバの StartSpec 宣言の検査。
//
// scope の正本は各サーバの StartSpec 1箇所で、bootstrap がそこから
// OAuth の metadata・認可既定値・トークン発行・受理検証まで全部を生成する
// （bootstrap.go）。ReadWrite サーバが gkill:read を広告していた事故
// （2026-08-30 修正）は、テストが自前の値で OAuthServer を組み立てていて
// エントリスクリプトの宣言値を誰も読んでいなかったために出荷まで漏れた。
// ここでは宣言値そのものを固定する。

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/mt3hr/gkill/src/server/gkill/mcp/jsonobj"
)

type startSpecExpectation struct {
	name            string
	spec            StartSpec
	scope           string
	defaultPort     int
	enableFileLinks bool
}

func startSpecExpectations() []startSpecExpectation {
	return []startSpecExpectation{
		{"read", ReadStartSpec, "gkill:read", 8808, true},
		{"write", WriteStartSpec, "gkill:write", 8809, false},
		{"readwrite", ReadWriteStartSpec, "gkill:readwrite", 8810, true},
	}
}

func TestServerStartSpecDeclarations(t *testing.T) {
	for _, expected := range startSpecExpectations() {
		t.Run(expected.name+" server declares its own scope / port / file-link policy", func(t *testing.T) {
			spec := expected.spec
			expectEqual(t, spec.Scope, expected.scope)
			expectEqual(t, spec.DefaultPort, expected.defaultPort)
			// 書き込み専用サーバだけ file-link URL を発行しない、という宣言もここが正本。
			expectEqual(t, spec.EnableFileLinks, expected.enableFileLinks)
			// 起動後にどこかが書き換える形へ戻さない（宣言は不変）。
			// Go では StartSpecFor が値のコピーを返すので、戻り値を書き換えても宣言は動かない。
			copied := StartSpecFor(expected.name)
			copied.Scope = "tampered"
			expectEqual(t, StartSpecFor(expected.name).Scope, expected.scope)
		})
	}

	t.Run("three servers never share a scope, port, script name, log file, or OAuth state file", func(t *testing.T) {
		specs := startSpecExpectations()
		columns := map[string]func(StartSpec) string{
			"scope":              func(s StartSpec) string { return s.Scope },
			"defaultPort":        func(s StartSpec) string { return jsonobj.MarshalString(s.DefaultPort) },
			"serverName":         func(s StartSpec) string { return s.ServerName },
			"logPrefix":          func(s StartSpec) string { return s.LogPrefix },
			"oauthStateFileName": func(s StartSpec) string { return s.OAuthStateFileName },
		}
		for key, column := range columns {
			seen := NewStringSet()
			for _, expected := range specs {
				seen.Add(column(expected.spec))
			}
			expectTrue(t, seen.Len() == len(specs), "duplicated %s", key)
		}
	})

	t.Run("bootstrap forwards spec.scope to OAuthServer (single source of scope)", func(t *testing.T) {
		// Start はポートを開きログファイルを作るため直接は呼ばない。
		// かわりに「OAuthServer へ spec.Scope を渡す」行の存在をソースで固定する
		// —— この行が消えると3サーバ全部の scope が宣言と無関係になり、
		// 宣言値のテスト（上）が緑のまま実挙動だけずれる。
		bootstrapSource := readSourceFile(t, "bootstrap.go")
		mustMatch(t, bootstrapSource, `Scope:\s+spec\.Scope`)
	})
}

// server_start ログの世代情報。「ソースは直っているのに AI からは古い」の切り分けは
// ここと gkill_status の schema_revision を見比べて行う（プロセスが古いのか、クライアントの
// 一覧が古いのか）。schema_revision 自体は status_tool_test.go が固定する。
func TestServerStartLogCarriesTheToolListGeneration(t *testing.T) {
	t.Run("startInfo reports pid, schema_revision and tool_count of the running server", func(t *testing.T) {
		server := &Server{SchemaRevision: "abc123def456", Tools: []*jsonobj.Object{obj(), obj(), obj()}}
		info := obj(startInfo(server)...)
		expectEqual(t, info, obj("pid", os.Getpid(), "schema_revision", "abc123def456", "tool_count", 3))
	})

	t.Run("both transports log server_start with startInfo (stdio and http)", func(t *testing.T) {
		bootstrapSource := readSourceFile(t, "bootstrap.go")
		lines := []string{}
		for _, line := range strings.Split(bootstrapSource, "\n") {
			if strings.Contains(line, `"server_start"`) {
				lines = append(lines, line)
			}
		}
		expectTrue(t, len(lines) == 2, "stdio と http の2箇所で server_start を出す（%d 箇所）", len(lines))
		for _, line := range lines {
			expectTrue(t, regexp.MustCompile(`startInfo\(server\)`).MatchString(line), "line lacks startInfo(server): %s", line)
		}
	})
}
