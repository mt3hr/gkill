package mcp

// 読み取り専用の gkill MCP サーバ（旧 gkill-read-server.mjs）。
//
// ディスパッチ（plugin → read → write）と起動処理は3サーバ共通で、
// このファイルが持つのは「どのツールを載せ、どの名前とポートで名乗るか」という表の1行だけ。
// 正本: server_base.go（handleToolCall）と bootstrap.go（起動）。

import "github.com/mt3hr/gkill/src/server/gkill/mcp/jsonobj"

// composeTools はサーバに載せるツール一覧を順序どおりに連結する。
func composeTools(lists ...[]*jsonobj.Object) []*jsonobj.Object {
	out := []*jsonobj.Object{}
	for _, list := range lists {
		out = append(out, list...)
	}
	return out
}

// filterTools は names に含まれる名前のツールだけを元の順序で残す。
func filterTools(tools []*jsonobj.Object, names *StringSet) []*jsonobj.Object {
	out := []*jsonobj.Object{}
	for _, tool := range tools {
		if name, _ := tool.String("name"); names.Has(name) {
			out = append(out, tool)
		}
	}
	return out
}

// newNameSet はツール名の集合（verify_docs が数える印）。
func newNameSet(names ...string) *StringSet { return NewStringSet(names...) }

// readServerTools は READ_TOOLS を全部載せる。書き込みツールは持たない。
func readServerTools() []*jsonobj.Object {
	return composeTools(ReadTools, PluginTools)
}

// NewReadServer は読み取り専用サーバ。
func NewReadServer(client GkillAPI, log *Logger) *Server {
	return NewServer(client, log, ServerOptions{
		ServerName:    "gkill-read-mcp",
		ServerKind:    "read",
		ServerVersion: ServerVersion,
		Tools:         readServerTools(),
		ReadToolNames: nil,
		WriteAppName:  "",
	})
}

// ReadStartSpec は読み取りサーバの宣言（意味は bootstrap.go の StartSpec）。
var ReadStartSpec = StartSpec{
	Kind:               "read",
	ServerName:         "gkill-read-mcp",
	LogPrefix:          "gkill_mcp_read",
	OAuthStateFileName: "mcp_oauth_read_state.json",
	DefaultPort:        8808,
	Scope:              "gkill:read",
	EnableFileLinks:    true,
}
