package mcp

// 編集前に読む: .claude/skills/gkill-mcp/SKILL.md
//
// 3つの MCP サーバに共通する JSON-RPC の受け口（旧 mcp-server-base.mjs）。
//
// handleMessage / handlePayload / constructor は3本とも1文字違わず同じだった。
// buildToolResult も read と readwrite は完全に同一で、write サーバだけが
// file-link 注入と IDF 画像ブロックを欠いた劣化コピーを持っていたので、
// 正しいほう (read / readwrite 版) をここへ引き上げて1本にする。
// handleToolCall も3本とも「plugin → read → write」の同じ順で、違うのは
// **write を持つか**と**read の選抜集合を持つか**の2つの値だけだったので、ここへ畳んだ。
// これで readwrite が read の上位集合であることが構造で担保される。
// 種別ごとに残すのは options の表1行だけ（ADR-0611）。
//
// server.Current* へ書き戻してはいけない理由（並行要求の混線）:
// documents/adr/0601-mcp-request-context-immutable.md

import (
	"context"
	"fmt"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/mcp/jsonobj"
)

// ServerOptions はサーバ種別ごとの「どのツールを載せ、どの名前で名乗るか」。
type ServerOptions struct {
	ServerName    string
	ServerKind    string
	ServerVersion string
	Tools         []*jsonobj.Object
	// ReadToolNames は読み取りツールのうち載せる分の選抜集合。nil なら全部。
	ReadToolNames *StringSet
	// WriteAppName は書き込み時の create_app / update_app。空なら書き込みツールを持たない。
	WriteAppName string
}

// Server はトランスポート非依存の JSON-RPC ハンドラ。
type Server struct {
	Client        GkillAPI
	Log           *Logger
	ServerName    string
	ServerVersion string
	// ServerKind は gkill_status が名乗る種別（read / write / readwrite）。
	ServerKind string
	// SchemaRevision はこのサーバが配るツール一覧の世代。tools/list の中身から決定的に計算し、
	// gkill_status の description 末尾へ焼き込む（クライアントが握っている一覧の世代を
	// AI 自身が応答と比べられる唯一の経路）。静的な ReadTools は書き換えず、
	// このサーバ用に gkill_status だけ差し替えた配列を持つ（status_tool.go）。
	SchemaRevision string
	Tools          []*jsonobj.Object
	// StartedAt はプロセスの世代。「ソースは直っているのに AI からは古い」の切り分けに要る。
	StartedAt time.Time
	// ReadToolNames は nil なら ReadTools 全部。集合ならそのうち載せる分だけ
	// （書き込み専用サーバは数本だけ載せる）。「read ツールか」の判定自体は
	// IsReadToolName が正本で、ここは選抜集合。選抜集合だけで判定すると
	// ReadTools から消えた名前がここに残っていても気づけない。
	ReadToolNames *StringSet
	// WriteAppName は空なら書き込みツールを持たない（読み取り専用サーバ）。
	// 文字列なら書いた記録の create_app に載る値。
	WriteAppName string
	// CurrentSessionID / CurrentUserID / CurrentRemoteAddr は stdio 起動時に一度だけ設定される値
	// （HTTP は要求ごとの RequestContext を引数で流すので、ここには書かない）。
	CurrentSessionID  string
	CurrentUserID     string
	CurrentRemoteAddr string
	// IsLocalTransport は MCP クライアントがこのマシン上で動くとき（stdio）だけ true。
	// 絶対パスはこのときだけクライアントへ出す。
	// トランスポートが明示しない限り false（漏らさない側に倒す）。
	IsLocalTransport bool
	// FileLinkContext は HttpTransport が置く。リモートクライアントにはローカルパスの代わりに公開 URL を渡す。
	// stdio では nil（ローカルクライアントはパスを直接読む）。
	FileLinkContext *FileLinkContext
}

// ServerVersion はサーバが名乗る版。main/common/mcp.go が起動前に本体の版へ差し替える。
var ServerVersion = "0.0.0"

// NewServer はツール一覧に schema_revision を焼き込んだサーバを作る。
func NewServer(client GkillAPI, log *Logger, opts ServerOptions) *Server {
	revision := ComputeSchemaRevision(opts.Tools)
	return &Server{
		Client:         client,
		Log:            log,
		ServerName:     opts.ServerName,
		ServerVersion:  opts.ServerVersion,
		ServerKind:     opts.ServerKind,
		SchemaRevision: revision,
		Tools:          StampSchemaRevision(opts.Tools, revision),
		StartedAt:      timeNow(),
		ReadToolNames:  opts.ReadToolNames,
		WriteAppName:   opts.WriteAppName,
	}
}

// RequestContext は HTTP モードの要求1件の文脈（ADR-0601。値で末端まで流し、共有フィールドへ書かない）。
type RequestContext struct {
	SessionID  string
	UserID     string
	RemoteAddr string
}

// CallContext はツール呼び出し1回分の文脈（旧 ctx）。
type CallContext struct {
	Ctx              context.Context
	Client           GkillAPI
	SID              string
	IsLocalTransport bool
	AppName          string
	SessionID        string
	UserID           string
	RemoteAddr       string
	Log              *Logger
	Server           *Server
}

func (c *CallContext) context() context.Context {
	if c == nil || c.Ctx == nil {
		return context.Background()
	}
	return c.Ctx
}

// timeNow はテストで差し替える現在時刻。
var timeNow = time.Now

// Transport は describeServer の transport（stdio / http）。
func (s *Server) Transport() string {
	if s.IsLocalTransport {
		return "stdio"
	}
	return "http"
}

// summarizeToolPayload は結果の1行要約を返す。plugin → read → write の順に委ねる。
// 各要約器は対象外のツールに「無し」を返すので、持っていないツールの分は素通りする
// (読み取り専用サーバは write の case に一致しない)。
// warnings / partial の1行サマリへの昇格はここ1箇所で全ツールに掛ける
// (要約器ごとに書くと必ず足し忘れる。古スキーマ警告の AppendStaleSchemaWarning と同じ判断)。
func summarizeToolPayload(name string, payload *jsonobj.Object) string {
	return AppendWarningsToSummary(summarizeToolPayloadBody(name, payload), payload)
}

func summarizeToolPayloadBody(name string, payload *jsonobj.Object) string {
	if summary, ok := SummarizePluginToolPayload(name, payload); ok {
		return summary
	}
	if summary, ok := SummarizeReadToolPayload(name, payload); ok {
		return summary
	}
	if summary, ok := SummarizeWriteToolPayload(name, payload); ok {
		return summary
	}
	return "Tool call completed."
}

// requestContextOrCurrent は RequestContext 未指定 (stdio / 単体テストの直接呼び出し) のときだけ
// 起動時に一度設定される Current* のスナップショットへフォールバックする。
// HTTP 経路は必ず RequestContext を渡すので、この分岐には入らない。
func (s *Server) requestContextOrCurrent(ctx *RequestContext) RequestContext {
	if ctx != nil {
		return *ctx
	}
	return RequestContext{SessionID: s.CurrentSessionID, UserID: s.CurrentUserID, RemoteAddr: s.CurrentRemoteAddr}
}

// HandleToolCall は plugin → read → write の順で委譲する。3サーバ共通。
func (s *Server) HandleToolCall(goCtx context.Context, name string, args any, ctx *RequestContext) (*jsonobj.Object, error) {
	rc := s.requestContextOrCurrent(ctx)
	sid := rc.SessionID

	if IsPluginToolName(name) {
		return HandlePluginToolCall(func(pathname string, body *jsonobj.Object) (*jsonobj.Object, error) {
			return s.Client.CallApi(goCtx, pathname, body, true, sid)
		}, name, args)
	}

	if IsReadToolName(name) && (s.ReadToolNames == nil || s.ReadToolNames.Has(name)) {
		return HandleReadToolCall(&CallContext{
			Ctx:              goCtx,
			Client:           s.Client,
			SID:              sid,
			IsLocalTransport: s.IsLocalTransport,
			Server:           s,
			Log:              s.Log,
		}, name, args)
	}

	if s.WriteAppName == "" {
		return nil, NewGkillApiError(UnknownToolMessage(name), nil)
	}
	// 書き込みツールは一覧（WriteTools）に載っているものだけを通す。実装はあっても公開していない
	// ツール（gkill_delete_skill。skill_delete_tool.go）が、振り分けの case を戻しただけで
	// 一覧に無いまま呼べるようになるのを防ぐ。
	if !IsWriteToolName(name) {
		return nil, NewGkillApiError(UnknownToolMessage(name), nil)
	}

	// 書き込みに刻む user は、その要求を認証したアカウントでなければならない。
	// sid（どのアカウントの DB へ書くか）と userId（レコードに刻む名前）は
	// 出どころが別なので、放っておくと静かにずれる。
	// sid がトークン由来（HTTP/OAuth）なのに userId だけ取れないと、環境変数
	// GKILL_USER へ落ちて「別アカウントのセッションへ書いたのに create_user は
	// 手元の名前」という記録ができあがり、あとから誰が書いたのか追えなくなる。
	// stdio は sid を持たず環境変数で接続するので、そちらは従来どおり。
	authenticatedUserID := rc.UserID
	if sid != "" && authenticatedUserID == "" {
		return nil, NewGkillApiError(
			"Cannot determine which user to record as the writer of this entry: "+
				"the session is authenticated but carries no user id, so create_user / update_user "+
				"would be taken from this server's environment instead of the account being written to. "+
				"Reconnect (re-authorize) this MCP server and retry.", nil)
	}
	userID := authenticatedUserID
	if userID == "" && s.Client != nil {
		userID = s.Client.UserID()
	}
	return HandleWriteToolCall(&CallContext{
		Ctx:     goCtx,
		Client:  s.Client,
		SID:     sid,
		UserID:  userID,
		AppName: s.WriteAppName,
	}, name, args)
}

// BuildToolResult はツールの payload を MCP の tools/call 結果にする。
func (s *Server) BuildToolResult(name string, payload *jsonobj.Object, isError bool, ctx *RequestContext) *jsonobj.Object {
	// ローカルクライアントには実パスを渡す。リモートには実パスを渡さず、代わりに期限付きの
	// 公開ファイル URL を注入する —— ただし**呼び出し側が include_file_urls で頼んだときだけ**
	// （ハンドラが MintFileLinksMark の印を立てる。JSON には出ない meta）。頼まれていなければ
	// 実パスを消すだけ。以前は HTTP なら全応答で idf ごとにトークンを2本鋳造していた（ADR-0630）。
	// file-link トークンは ctx.SessionID で鋳造する。ctx 未指定 (単体テスト) のみ CurrentSessionID。
	if !s.IsLocalTransport {
		mint := false
		if payload != nil {
			if v, ok := payload.Meta(MintFileLinksMark); ok && v == true {
				mint = true
			}
		}
		if s.FileLinkContext != nil && !isError && mint {
			sessionID := s.CurrentSessionID
			if ctx != nil {
				sessionID = ctx.SessionID
			}
			ApplyFileLinks(payload, s.FileLinkContext, sessionID)
		} else {
			StripFilePaths(payload)
		}
	}

	var summary string
	if isError {
		errText := "Unknown tool error"
		var detail *jsonobj.Object
		if payload != nil {
			if v := payload.Value("error"); jsTruthy(v) {
				errText = jsString(v)
			}
			if d, ok := payload.Object("detail"); ok {
				detail = d
			}
		}
		summary = SummarizeToolError(name, errText, detail)
	} else {
		summary = summarizeToolPayload(name, payload)
	}

	hasBase64 := (name == "gkill_get_idf_file" || name == "gkill_get_skill") && !isError && payload != nil && jsTruthy(payload.Value("file_content_base64"))
	// 画像は image ブロックでバイト列を届ける
	hasImageBlock := hasBase64 && jsTruthy(payload.Value("is_image"))

	// テキスト表現に base64 は載せない（読めないうえに肥大化するだけ）
	textPayload := payload
	if hasBase64 {
		rest := payload.Clone()
		rest.Delete("file_content_base64")
		// 画像のときは「バイト列は image ブロックへ移してある」の印を残す。
		// structuredContent だけを見ると file_content_base64 が無く、
		// 「画像が返っていない」と誤読された (2026-08-30 レビュー 5.5)。
		if hasImageBlock {
			rest.Set("image_content_attached", true)
		}
		textPayload = rest
	}
	// structuredContent から base64 を落とすのは、image ブロックで既にバイト列を届けている画像のときだけ。
	// 同じデータが1レスポンスに2回入ると、クライアント側のツール結果上限を超えて切り捨てられ、
	// 画像そのものが届かなくなる。非画像 (PDF 等) はここが唯一のバイト列の渡し口なので残す。
	structuredPayload := payload
	if hasImageBlock {
		structuredPayload = textPayload
	}

	text := summary
	if textPayload != nil {
		text = summary + "\n\n" + jsonobj.MarshalIndentString(textPayload, "  ")
	}

	content := jsonobj.Arr(jsonobj.Obj("type", "text", "text", text))
	if hasImageBlock {
		content = append(content, jsonobj.Obj(
			"type", "image",
			"data", payload.Value("file_content_base64"),
			"mimeType", NormalizeMimeType(payload.Value("mime_type")),
		))
	}
	result := jsonobj.Obj("content", content, "isError", isError)
	if structuredPayload != nil {
		result.Set("structuredContent", structuredPayload)
	}
	return result
}

// HandlePayload は JSON-RPC の本文（単体かバッチ）を処理する。
// 戻り値は *jsonobj.Object（単体）、[]any（バッチ）、nil（応答なし）。
//
// ctx は HttpTransport が組む1リクエスト分の不変値 {SessionID, UserID, RemoteAddr}。
// 以前は server.current* 共有フィールドに書いて await をまたいで読んでいたため、
// 並行リクエストで別要求の user/session が混線した。引数で末端まで流すことで混線を構造的に断つ。
func (s *Server) HandlePayload(goCtx context.Context, payload any, ctx *RequestContext) any {
	batch, isArray := jsonobj.AsArray(payload)
	if !isArray {
		if response := s.HandleMessage(goCtx, payload, ctx); response != nil {
			return response
		}
		return nil
	}
	if len(batch) == 0 {
		return jsonobj.Obj("jsonrpc", "2.0", "id", nil, "error", jsonobj.Obj("code", -32600, "message", "Invalid Request"))
	}
	responses := []any{}
	for _, message := range batch {
		if response := s.HandleMessage(goCtx, message, ctx); response != nil {
			responses = append(responses, response)
		}
	}
	if len(responses) == 0 {
		return nil
	}
	return responses
}

// HandleMessage は JSON-RPC のメッセージ1件を処理する。通知には nil を返す。
func (s *Server) HandleMessage(goCtx context.Context, message any, ctx *RequestContext) *jsonobj.Object {
	rc := s.requestContextOrCurrent(ctx)
	msg, isObject := message.(*jsonobj.Object)
	if !isObject || msg == nil || msg.Value("jsonrpc") != "2.0" || !jsTruthy(msg.Value("method")) {
		var id any
		if isObject && msg != nil && msg.Has("id") {
			id = msg.Value("id")
		}
		return jsonobj.Obj("jsonrpc", "2.0", "id", id, "error", jsonobj.Obj("code", -32600, "message", "Invalid Request"))
	}

	hasID := msg.Has("id")
	id := msg.Value("id")
	method := jsString(msg.Value("method"))
	var params any = jsonobj.New()
	if msg.Has("params") {
		params = msg.Value("params")
	}

	if method == "notifications/initialized" {
		return nil
	}

	if method == "initialize" {
		if !hasID {
			return nil
		}
		return jsonobj.Obj("jsonrpc", "2.0", "id", id, "result", jsonobj.Obj(
			"protocolVersion", "2024-11-05",
			"capabilities", jsonobj.Obj("tools", jsonobj.New()),
			// version にツール一覧の世代を添える（semver のビルドメタデータ形式）。
			// クライアントの接続情報画面や initialize のログから、どの世代の一覧を
			// 配ったサーバかが読める。AI 向けの経路は gkill_status の description。
			"serverInfo", jsonobj.Obj("name", s.ServerName, "version", s.ServerVersion+"+schema."+s.SchemaRevision),
		))
	}

	if method == "ping" {
		if !hasID {
			return nil
		}
		return jsonobj.Obj("jsonrpc", "2.0", "id", id, "result", jsonobj.New())
	}

	if method == "tools/list" {
		if !hasID {
			return nil
		}
		return jsonobj.Obj("jsonrpc", "2.0", "id", id, "result", jsonobj.Obj("tools", toolsToAny(s.Tools)))
	}

	if method == "tools/call" {
		if !hasID {
			return nil
		}
		toolStart := timeNow()
		response, toolName, err := s.callTool(goCtx, params, &rc)
		if err == nil {
			s.Log.Access("tool_call",
				"tool", toolName,
				"user_id", nullIfEmpty(rc.UserID),
				"remote_addr", nullIfEmpty(rc.RemoteAddr),
				"duration", durationString(toolStart),
			)
			return jsonobj.Obj("jsonrpc", "2.0", "id", id, "result", s.BuildToolResult(toolName, response, false, &rc))
		}
		var detail *jsonobj.Object
		if apiErr, ok := AsGkillApiError(err); ok {
			detail = apiErr.Detail
		}
		messageText := err.Error()
		// 引数の型違い（errors.go の InvalidArgument）は呼び出し側の誤りで、HTTP でいう 400。
		// サーバ障害と同じ ERROR に混ぜると、ログの ERROR 件数が意味を持たなくなる。
		isCallerError := false
		if detail != nil {
			_, isCallerError = detail.String("field")
		}
		rawName := paramsName(params)
		kv := []any{
			"tool", rawName,
			"user_id", nullIfEmpty(rc.UserID),
			"remote_addr", nullIfEmpty(rc.RemoteAddr),
			"duration", durationString(toolStart),
			"error", messageText,
		}
		if isCallerError {
			s.Log.Warn("tool_call_error", kv...)
		} else {
			s.Log.Error("tool_call_error", kv...)
		}
		var detailValue any
		if detail != nil {
			detailValue = detail
		}
		errorPayload := jsonobj.Obj("error", messageText, "detail", detailValue)
		// 名前が無い（params が object でない・name 未指定）ときは "Tool call failed" にする（旧実装の params.name が undefined の形）。
		errorName := ""
		if rawName != nil {
			errorName = jsString(rawName)
		}
		return jsonobj.Obj("jsonrpc", "2.0", "id", id, "result", s.BuildToolResult(errorName, errorPayload, true, &rc))
	}

	if !hasID {
		return nil
	}
	return jsonobj.Obj("jsonrpc", "2.0", "id", id, "error", jsonobj.Obj("code", -32601, "message", "Method not found: "+method))
}

// callTool は tools/call の params を検証してツールを呼ぶ。戻り値のツール名は検証済みのもの。
func (s *Server) callTool(goCtx context.Context, params any, rc *RequestContext) (*jsonobj.Object, string, error) {
	if !IsPlainObject(params) {
		return nil, "", InvalidArgument("params", "must be an object", params)
	}
	p := params.(*jsonobj.Object)
	toolName, err := AssertTrimmedString(p.Value("name"), "name")
	if err != nil {
		return nil, "", err
	}
	var toolArgs any = jsonobj.New()
	if p.Has("arguments") {
		toolArgs = p.Value("arguments")
	}
	response, err := s.HandleToolCall(goCtx, toolName, toolArgs, rc)
	if err != nil {
		return nil, toolName, err
	}
	return response, toolName, nil
}

// paramsName は params.name をそのまま返す（検証前なので型は問わない。ログとエラー要約用）。
func paramsName(params any) any {
	if p, ok := params.(*jsonobj.Object); ok && p != nil {
		v := p.Value("name")
		if jsonobj.IsUndefined(v) {
			return nil
		}
		return v
	}
	return nil
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func durationString(start time.Time) string {
	return fmt.Sprintf("%dms", timeNow().Sub(start).Milliseconds())
}

// AuthenticateUserFunc は OAuth の利用者認証コールバック。認証に通れば gkill のセッション ID、
// 拒否なら空文字列。error は「gkill へ届かなかった」など呼び出し自体の失敗（Node の throw）。
type AuthenticateUserFunc func(ctx context.Context, userID, passwordSha256 string) (string, error)

// MakeOAuthAuthenticateUser は OAuth の利用者認証コールバックを作る。
//
// gkill のログイン応答はエンベロープ（errors / session_id）なので、HTTP ステータスではなく中身で判定する。
//
// 認証の失敗理由は**呼び出し側へ返さない**（総当たりの手掛かりになる）。
// アクセスログにだけ user_id つきで残す。
func MakeOAuthAuthenticateUser(client GkillAPI, log *Logger) AuthenticateUserFunc {
	return func(ctx context.Context, userID, passwordSha256 string) (string, error) {
		response, err := client.Post(ctx, "/api/login", jsonobj.Obj(
			"user_id", userID,
			"password_sha256", passwordSha256,
			"locale_name", client.DefaultLocale(),
		))
		if err != nil {
			// 応答には理由を返さない（総当たりの手がかりになる）が、ログには残す。
			// 捨てると「gkill へ届かない」のか「パスワードが違う」のかが区別できない。
			log.Warn("auth_failure", "user_id", userID, "reason", "request_failed", "error", err.Error())
			return "", nil
		}
		sessionID, _ := response.String("session_id")
		if client.HasErrors(response) || sessionID == "" {
			log.Warn("auth_failure", "user_id", userID, "reason", "rejected_by_gkill")
			return "", nil
		}
		log.Info("auth_success", "user_id", userID)
		return sessionID, nil
	}
}
