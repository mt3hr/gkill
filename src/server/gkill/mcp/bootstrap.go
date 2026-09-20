package mcp

// 編集前に読む: .claude/skills/gkill-mcp/SKILL.md
//
// 3つの MCP サーバの起動ブロック（旧 mcp-server-bootstrap.mjs）。
//
// read / write / readwrite は、起動処理が**表の1行以外まったく同じ**
// （ログ名 / サーバ名 / 既定ポート / OAuth 状態ファイル名 / scope / file-link の可否）。
// 同じ手順が3形態あると、片方だけ直して静かにずれる。
// ここが正本で、3サーバは表の1行だけを持つ（ADR-0611）。
//
// ここへ足すもの: 全サーバで同じでなければならない起動時の手順。
// ここへ足さないもの: サーバごとに違う「どのツールを持つか」（それは ServerOptions）。

import (
	"context"
	"errors"
	"os"
	"strconv"
	"time"
)

// StartSpec は起動 spec の静的な部分。scope / ポート / file-link 可否は「このサーバが何者か」の宣言で、
// OAuth の scope はこの1値から metadata・認可既定値・トークン発行・受理検証まで全部が生成される。
// テストが宣言値そのものを固定できるよう公開する
// —— ReadWrite サーバが gkill:read を広告していた事故は、テストが自前の値で
// OAuthServer を組み立てていて、ここの宣言を誰も読んでいなかったために漏れた。
type StartSpec struct {
	// Kind は read / write / readwrite。
	Kind string
	// ServerName は initialize の serverInfo.name（gkill-read-mcp など）。
	ServerName string
	// LogPrefix は gkill_log のファイル接頭辞（gkill_mcp_read → logs/gkill_mcp_read*.log）。
	LogPrefix string
	// OAuthStateFileName は $GKILL_HOME/configs/ 配下のファイル名（HTTP のときだけ使う）。
	OAuthStateFileName string
	// DefaultPort は MCP_PORT 未指定時のポート。
	DefaultPort int
	// Scope は OAuth のスコープ名。
	Scope string
	// EnableFileLinks は file-link URL を発行するか（書き込み専用サーバは false）。
	EnableFileLinks bool
}

// StartSpecFor は種別の宣言を値で返す（呼び出し側が書き換えても宣言は動かない）。
func StartSpecFor(kind string) StartSpec {
	switch kind {
	case "read":
		return ReadStartSpec
	case "write":
		return WriteStartSpec
	case "readwrite":
		return ReadWriteStartSpec
	}
	return StartSpec{}
}

// startInfo は server_start ログに載せる「このプロセスがどの世代のツール一覧を配るか」。
// 「ソースは直っているのに AI からは古い」の切り分けは、まずここと gkill_status の
// schema_revision を見比べる（プロセスが古いのか、クライアントの一覧が古いのか）。
func startInfo(server *Server) []any {
	return []any{"pid", os.Getpid(), "schema_revision", server.SchemaRevision, "tool_count", len(server.Tools)}
}

// StartOptions は Start に渡す解決済みの値（config.go の Settings と main/common/mcp.go が組む）。
type StartOptions struct {
	// Transport は stdio か http。
	Transport string
	// Client は gkill 本体へのクライアント。
	Client GkillAPI
	// Log は gkill_log 上のロガー（nil なら何も書かない）。
	Log *Logger
	// LogLevelName は server_start ログに載せるレベル名。
	LogLevelName string
	// Port は HTTP の待ち受けポート。0 なら spec.DefaultPort。
	Port int
	// BindAddr は HTTP の待ち受けアドレス。空なら MCP_BIND_ADDR → 0.0.0.0。
	BindAddr string
	// OAuthIssuer は OAuth の公開 URL。空なら http://localhost:<port>。
	OAuthIssuer string
	// OAuthStatePath はリフレッシュトークンと DCR クライアントの永続化先（HTTP のときだけ使う）。
	OAuthStatePath string
}

// Start はサーバを作り、トランスポートを起動する。
// stdio は stdin が閉じるまで、http は ctx が終わるまでブロックする。
func Start(ctx context.Context, spec StartSpec, opts StartOptions) error {
	if opts.Client == nil {
		return errors.New("mcp.Start requires a gkill client")
	}
	log := opts.Log
	server := NewServerForKind(spec.Kind, opts.Client, log)
	if server == nil {
		return errors.New("unknown MCP server kind: " + spec.Kind)
	}

	if opts.Transport == "http" {
		port := opts.Port
		if port == 0 {
			port = spec.DefaultPort
		}
		issuer := opts.OAuthIssuer
		if issuer == "" {
			issuer = "http://localhost:" + strconv.Itoa(port)
		}
		oauth, err := NewOAuthServer(OAuthServerOptions{
			Issuer: issuer,
			// scope の正本はここ1箇所。metadata・認可既定値・トークン発行・Bearer 受理の全てがこの値から生成される。
			Scope:            spec.Scope,
			PersistPath:      opts.OAuthStatePath,
			Log:              log,
			AuthenticateUser: MakeOAuthAuthenticateUser(opts.Client, log),
		})
		if err != nil {
			return err
		}
		defer oauth.Close()
		log.Info("server_start", append([]any{"transport", "http", "log_level", opts.LogLevelName, "port", port}, startInfo(server)...)...)
		transport, err := NewHttpTransport(server, port, oauth, HttpTransportOptions{EnableFileLinks: spec.EnableFileLinks, BindAddr: opts.BindAddr})
		if err != nil {
			return err
		}
		if err := transport.Start(); err != nil {
			return err
		}
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return transport.Stop(shutdownCtx)
	}

	server.CurrentUserID = opts.Client.UserID()
	log.Info("server_start", append([]any{"transport", "stdio", "log_level", opts.LogLevelName}, startInfo(server)...)...)
	return NewStdioTransport(server).Start()
}
