package main

import (
	"encoding/json"
	"time"
)

const (
	// repName は manifest.json の rep_name と一致させること。
	repName = "Codex"
	// dataType は manifest.json の data_type と一致させること。
	dataType = "codex_turn"
	// appName はログ出力と CreateApp/UpdateApp に使う。
	appName = "gkill_plugin_codex"
)

// 外側の type。
const (
	outerSessionMeta  = "session_meta"
	outerTurnContext  = "turn_context"
	outerEventMsg     = "event_msg"
	outerResponseItem = "response_item"
)

// ファイル種別。
const (
	kindRollout = "rollout" // rollout-<日時>-<uuid>.jsonl
	kindIndex   = "index"   // session_index.jsonl
	kindOther   = "other"   // 対象外
)

// スレッドの出自。session_meta.thread_source の値。
// user 側は「subagentでない」で判定するので定数は持たない。
const (
	threadSourceSubAgent = "subagent"
)

// Kyou のロール。
const (
	roleHuman     = "human"
	roleAssistant = "assistant"
)

// threadItem.Kind。正規化した1要素の種類。
const (
	itemUser      = "user"      // 人間の発言。ここで応答が切れる
	itemAssistant = "assistant" // 応答本文
	itemThinking  = "thinking"  // 思考の要約
	itemTool      = "tool"      // ツール呼び出し(結果は持たない)
	itemPatch     = "patch"     // 変更したファイル
	itemPlan      = "plan"      // 計画
	itemNotice    = "notice"    // 中断・コンテキスト圧縮などの短い通知
	itemSpawn     = "spawn"     // サブエージェントの起動
	itemTurn      = "turn"      // モデルが切り替わった印(描画はしない)
	itemTaskDone  = "task_done" // ターンの所要時間(描画はしない)
)

const (
	maxToolSummaryRunes = 200
	maxNoticeRunes      = 200
	maxPlanRunes        = 4000
	maxTextRunes        = 100000
	maxSubAgentItems    = 200
	maxPatchFiles       = 200
)

// recordEnvelope は全レコード共通の外側。
type recordEnvelope struct {
	Timestamp time.Time       `json:"timestamp"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
}

// sessionMetaPayload は session_meta の中身。
//
// 1ファイルに1〜13回出る(resume のたびに書かれる)。
// identity(ID/ThreadSource/ParentThreadID/...)は必ず1回目のものだけを使うこと。
// サブエージェントのファイルには2回目として「親の」session_meta が入っているので、
// マージすると自分が親にすり替わる。
//
// SessionID は使ってはいけない。52ファイル中23ファイルに存在せず、
// 存在してもサブエージェントでは親のIDが入っている(52ファイルに対し38種しかない)。
// スレッドの同一性は ID(＝ファイル名のuuid)で決める。
type sessionMetaPayload struct {
	ID             string          `json:"id"`
	SessionID      string          `json:"session_id"`
	ParentThreadID string          `json:"parent_thread_id"`
	ForkedFromID   string          `json:"forked_from_id"`
	ThreadSource   string          `json:"thread_source"`
	AgentPath      string          `json:"agent_path"`
	AgentNickname  string          `json:"agent_nickname"`
	Cwd            string          `json:"cwd"`
	Originator     string          `json:"originator"`
	CLIVersion     string          `json:"cli_version"`
	Git            *gitMeta        `json:"git"`
	Source         json.RawMessage `json:"source"` // 文字列("vscode")かオブジェクト(subagent)
}

type gitMeta struct {
	CommitHash    string `json:"commit_hash"`
	Branch        string `json:"branch"`
	RepositoryURL string `json:"repository_url"`
}

// sourceSubagent は session_meta.source がオブジェクトのときの中身。
type sourceSubagent struct {
	Subagent *struct {
		ThreadSpawn *struct {
			ParentThreadID string `json:"parent_thread_id"`
			Depth          int    `json:"depth"`
			AgentPath      string `json:"agent_path"`
			AgentNickname  string `json:"agent_nickname"`
		} `json:"thread_spawn"`
	} `json:"subagent"`
}

type turnContextPayload struct {
	TurnID         string `json:"turn_id"`
	Cwd            string `json:"cwd"`
	Model          string `json:"model"`
	Effort         string `json:"effort"`
	Personality    string `json:"personality"`
	ApprovalPolicy string `json:"approval_policy"`
}

type userMessagePayload struct {
	Message string `json:"message"`
}

type agentMessagePayload struct {
	Message string `json:"message"`
	Phase   string `json:"phase"`
}

type agentReasoningPayload struct {
	Text string `json:"text"`
}

type functionCallPayload struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSONを文字列にしたもの
	CallID    string `json:"call_id"`
	ID        string `json:"id"`
}

type customToolCallPayload struct {
	Name   string `json:"name"`
	Input  string `json:"input"` // 生の文字列(コードやパッチ)
	CallID string `json:"call_id"`
	ID     string `json:"id"`
}

type webSearchCallPayload struct {
	Action json.RawMessage `json:"action"`
	ID     string          `json:"id"`
}

type patchApplyEndPayload struct {
	CallID  string                 `json:"call_id"`
	TurnID  string                 `json:"turn_id"`
	Success bool                   `json:"success"`
	Changes map[string]patchChange `json:"changes"`
}

type patchChange struct {
	Type        string `json:"type"`
	UnifiedDiff string `json:"unified_diff"`
}

type subAgentActivityPayload struct {
	EventID       string `json:"event_id"` // call_id
	OccurredAtMs  int64  `json:"occurred_at_ms"`
	AgentThreadID string `json:"agent_thread_id"`
	AgentPath     string `json:"agent_path"`
	Kind          string `json:"kind"` // started / interacted / interrupted
}

type itemCompletedPayload struct {
	TurnID string `json:"turn_id"`
	Item   struct {
		Type string `json:"type"`
		ID   string `json:"id"`
		Text string `json:"text"`
	} `json:"item"`
}

type turnAbortedPayload struct {
	TurnID string `json:"turn_id"`
	Reason string `json:"reason"`
}

// threadItem は1ファイルを正規化した1要素。thread_item テーブルの1行に対応する。
type threadItem struct {
	Seq   int64
	TS    time.Time
	Kind  string
	Name  string // ツール名 / モデル名 / エージェントパス
	Text  string // 本文、または要約
	RefID string // call_id / turn_id / agent_thread_id
	Extra string // 種別ごとの付加情報のJSON
}

// ideContext は user_message に混ざるIDEの前置き。
type ideContext struct {
	ActiveFile string   `json:"active_file,omitempty"`
	OpenTabs   []ideTab `json:"open_tabs,omitempty"`
}

type ideTab struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// patchFile は変更した1ファイル。unified diff は保存しない。
type patchFile struct {
	Path    string `json:"path"`
	Type    string `json:"type"` // add / update / delete
	Added   int    `json:"added"`
	Removed int    `json:"removed"`
}

// toolCall はツール呼び出し1件。実行結果は持たない。
type toolCall struct {
	Name    string    `json:"name"`
	Summary string    `json:"summary"`
	CallID  string    `json:"call_id,omitempty"`
	Agent   *subAgent `json:"agent,omitempty"`
}

// subAgent は親の応答に畳み込むサブエージェントの会話。
type subAgent struct {
	ThreadID  string     `json:"thread_id"`
	AgentPath string     `json:"agent_path"`
	Nickname  string     `json:"nickname"`
	Prompt    string     `json:"prompt"`
	Items     []turnItem `json:"items"`
}

// turnItem は詳細ビューに描く1ブロック。
type turnItem struct {
	Kind     string      `json:"kind"`
	Text     string      `json:"text,omitempty"`
	Thinking []string    `json:"thinking,omitempty"`
	Tools    []toolCall  `json:"tools,omitempty"`
	Files    []patchFile `json:"files,omitempty"`
	Agent    *subAgent   `json:"agent,omitempty"`
}

// message は1Kyouになる単位。body_json に入る。
type message struct {
	ID           string      `json:"id"`
	Role         string      `json:"role"`
	ThreadID     string      `json:"thread_id"`
	RootThreadID string      `json:"root_thread_id"`
	Ordinal      int64       `json:"ordinal"`
	Title        string      `json:"title"`
	Project      string      `json:"project"`
	Branch       string      `json:"branch"`
	Model        string      `json:"model"`
	Originator   string      `json:"originator"`
	Cwd          string      `json:"cwd"`
	Text         string      `json:"text"`
	IDEContext   *ideContext `json:"ide_context,omitempty"`
	RelatedTime  time.Time   `json:"related_time"`
	UpdateTime   time.Time   `json:"update_time"`
	DurationMs   int64       `json:"duration_ms,omitempty"`
	Items        []turnItem  `json:"items"`
}
