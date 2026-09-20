package mcp

// 読み書き両用の gkill MCP サーバ（旧 gkill-readwrite-server.mjs）。
//
// ディスパッチ（plugin → read → write）と起動処理は3サーバ共通で、
// このファイルが持つのは「どのツールを載せ、どの名前とポートで名乗るか」という表の1行だけ。
// 正本: server_base.go（handleToolCall）と bootstrap.go（起動）。

import "github.com/mt3hr/gkill/src/server/gkill/mcp/jsonobj"

// readWriteServerTools は読み取り + 書き込み + プラグインの全部。
func readWriteServerTools() []*jsonobj.Object {
	return composeTools(ReadTools, WriteTools, PluginTools)
}

// NewReadWriteServer は読み書き両用サーバ。
func NewReadWriteServer(client GkillAPI, log *Logger) *Server {
	return NewServer(client, log, ServerOptions{
		ServerName:    "gkill-readwrite-mcp",
		ServerKind:    "readwrite",
		ServerVersion: ServerVersion,
		Tools:         readWriteServerTools(),
		ReadToolNames: nil,
		WriteAppName:  "gkill_mcp_readwrite",
	})
}

// NewServerForKind は種別名からサーバを作る（schema-budget サブコマンドとテストが使う）。
func NewServerForKind(kind string, client GkillAPI, log *Logger) *Server {
	switch kind {
	case "read":
		return NewReadServer(client, log)
	case "write":
		return NewWriteServer(client, log)
	case "readwrite":
		return NewReadWriteServer(client, log)
	}
	return nil
}

// ReadWriteStartSpec は読み書きサーバの宣言。宣言値の意味と公開の理由は server_read.go の同名変数を参照。
var ReadWriteStartSpec = StartSpec{
	Kind:               "readwrite",
	ServerName:         "gkill-readwrite-mcp",
	LogPrefix:          "gkill_mcp_readwrite",
	OAuthStateFileName: "mcp_oauth_readwrite_state.json",
	DefaultPort:        8810,
	Scope:              "gkill:readwrite",
	EnableFileLinks:    true,
}
