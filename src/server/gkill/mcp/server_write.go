package mcp

// 書き込み用の gkill MCP サーバ（旧 gkill-write-server.mjs）。
//
// ディスパッチ（plugin → read → write）と起動処理は3サーバ共通で、
// このファイルが持つのは「どのツールを載せ、どの名前とポートで名乗るか」という表の1行だけ。
// 正本: server_base.go（handleToolCall）と bootstrap.go（起動）。

import "github.com/mt3hr/gkill/src/server/gkill/mcp/jsonobj"

// WriteServerReadToolNames は書き込みサーバにも載せる読み取りツール。
// 前3つは書き込みの前に rep名 / 板名 / タグ名を引くためのもの。
// gkill_get_kyou_history は gkill_delete_kyou / gkill_restore_kyou の相棒で、
// 「今なにを消したのか」「なにを戻そうとしているのか」を同じサーバから確かめられないと
// 削除の取り消しが当てずっぽうになるので、ここに載せる。
//
// これは「載せる分の選抜集合」であって、「read ツールかどうか」の判定ではない
// （判定は read_handlers.go の IsReadToolName が正本。Server が両方を見る）。
var WriteServerReadToolNames = newNameSet(
	// 接続先の確認とツール一覧の世代（schema_revision）の照合。3サーバ全部に載る。
	"gkill_status",
	// ツール説明の本文（KFTL の文法全文など）。説明文は要約なので、これも3サーバ全部に載る（ADR-0622）。
	"gkill_get_mcp_help",
	// 「どのアカウントへ書くのか」を書く前に確かめる手段。
	// 3サーバが別アカウントを向いていることがあり、これが無いと
	// 書き込み専用サーバだけが自分の接続先を答えられなかった
	// （しかも EntityNotFoundMessage はこのツールを名指しで案内していた）。
	"gkill_get_application_config",
	"gkill_get_all_rep_names",
	"gkill_get_mi_board_list",
	"gkill_get_all_tag_names",
	"gkill_get_kyou_history",
	// スキル（ADR-0634）。書き換える前に読んで revision を得る手段なので、書き込み専用サーバにも載せる。
	"gkill_get_skill_list",
	"gkill_get_skill",
)

// writeServerTools は書き込みツール + 選抜した読み取りツール + プラグインツール。
// 読み取りツールの定義は読み取りサーバと同じオブジェクトを使う（以前は description が食い違っていた）。
func writeServerTools() []*jsonobj.Object {
	return composeTools(WriteTools, filterTools(ReadTools, WriteServerReadToolNames), PluginTools)
}

// NewWriteServer は書き込み専用サーバ。
func NewWriteServer(client GkillAPI, log *Logger) *Server {
	return NewServer(client, log, ServerOptions{
		ServerName:    "gkill-write-mcp",
		ServerKind:    "write",
		ServerVersion: ServerVersion,
		Tools:         writeServerTools(),
		ReadToolNames: WriteServerReadToolNames,
		WriteAppName:  "gkill_mcp_write",
	})
}

// WriteStartSpec は書き込みサーバの宣言。宣言値の意味と公開の理由は server_read.go の同名変数を参照。
// EnableFileLinks:false は「書き込み専用サーバは file-link URL を発行しない」の宣言で、
// bootstrap がそのまま HttpTransport へ渡す。
var WriteStartSpec = StartSpec{
	Kind:               "write",
	ServerName:         "gkill-write-mcp",
	LogPrefix:          "gkill_mcp_write",
	OAuthStateFileName: "mcp_oauth_write_state.json",
	DefaultPort:        8809,
	Scope:              "gkill:write",
	EnableFileLinks:    false,
}
