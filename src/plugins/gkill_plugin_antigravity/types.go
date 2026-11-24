package main

import (
	"encoding/json"
	"time"
)

const repName = "Antigravity"

const dataType = "antigravity_turn"

// appName はstderrログとKyouのCreateApp/UpdateAppに使うプラグイン名。
const appName = "gkill_plugin_antigravity"

// 走査で拾うファイルの種別。ファイル名だけで決める(scan.go の classifyFileName)。
const (
	kindTranscript = "transcript" // brain/<会話ID>/.system_generated/logs/transcript_full.jsonl
	kindSummaries  = "summaries"  // conversation_summaries.db
	kindOther      = "other"      // 対象外
)

// 発言者。
const (
	roleHuman     = "human"
	roleAssistant = "assistant"
)

// transcript_full.jsonl のレコード種別。
//
// 実測した15種を役割で3つに分ける。知らない種別が増えても捨てないこと
// (捨てるとログの形が変わった日に会話が静かに消える)。既定は「システム発の注記」扱いにする。
const (
	stepUserInput       = "USER_INPUT"       // 人間の発言。source は USER_EXPLICIT
	stepPlannerResponse = "PLANNER_RESPONSE" // モデルの発言。content / thinking / tool_calls
)

// toolResultStepTypes はツールの実行結果が入るレコード種別。
//
// PLANNER_RESPONSE の tool_calls の件数とほぼ1対1で続くことを実測で確かめた。
// content はツールの出力そのもので、ファイル閲覧だけで1件 KB 級〜数十 KB になるため
// 保持しない(gkill_plugin_codex と同じ方針)。error があるときだけ1行の注記にする。
var toolResultStepTypes = map[string]struct{}{
	"GENERIC":        {},
	"VIEW_FILE":      {},
	"LIST_DIRECTORY": {},
	"GREP_SEARCH":    {},
	"RUN_COMMAND":    {},
	"SEARCH_WEB":     {},
	"CODE_ACTION":    {},
	"GENERATE_IMAGE": {},
}

// 表示・保存サイズの上限。
const (
	maxToolSummaryRunes = 200
	maxNoticeRunes      = 200
	maxSearchTextBytes  = 512 * 1024
)

// stepRecord は transcript_full.jsonl の1行のうち、このプラグインが必要とする欄だけ。
//
// 全レコードが step_index / type / source / status / created_at を持ち、
// それ以外の欄は種別によって有無が変わる。
type stepRecord struct {
	StepIndex int              `json:"step_index"`
	Type      string           `json:"type"`
	Source    string           `json:"source"`
	Status    string           `json:"status"`
	CreatedAt time.Time        `json:"created_at"`
	Content   string           `json:"content"`
	Thinking  string           `json:"thinking"`
	ToolCalls []toolCallRecord `json:"tool_calls"`
	Model     string           `json:"model"`
	Error     string           `json:"error"`
	// ErrorCode は数値でも文字列でも来うるので生のまま受ける(型を決め打つと行ごと読めなくなる)。
	ErrorCode json.RawMessage `json:"error_code"`
	Media     []mediaRecord   `json:"media"`
}

// toolCallRecord は PLANNER_RESPONSE.tool_calls の1要素。
// args は道具ごとに形が違うので生のまま受け、要約だけを作る。
type toolCallRecord struct {
	Name string          `json:"name"`
	Args json.RawMessage `json:"args"`
}

// mediaRecord は添付(生成画像など)。uri は端末のパスなので保持しない。
type mediaRecord struct {
	MimeType string `json:"mime_type"`
	URI      string `json:"uri"`
}

// conversationMeta は conversation_summaries.db から引くタイトルとワークスペース。
// 無い会話もあるので、どちらも空のことがある。
type conversationMeta struct {
	ConversationID string
	Title          string
	Project        string
}

// message は1つのKyouになる単位。
// ChatGPT / Claude.ai / Claude Code / Codex プラグインと粒度を揃えており、次の2種類がある。
//
//	roleHuman     : 人間の発言1つ(USER_INPUT)。本文は Text に入る
//	roleAssistant : その発言に対する一連の応答。次の USER_INPUT までをまとめて1つにする。
//	                テキスト・thinking・ツール呼び出し・システム発の注記が時系列で Items に入る
type message struct {
	ID             string `json:"id"`
	Role           string `json:"role"`
	ConversationID string `json:"conversation_id"`
	// Title / Project は body_json に焼かない(conversation_summaries.db の更新で
	// 全会話を読み直さずに済ませるため、kyou_cache の別カラムで持つ)。
	Title   string `json:"-"`
	Project string `json:"-"`
	// StepIndex はこの単位の先頭レコードの step_index。Kyou ID の材料。
	StepIndex int    `json:"step_index"`
	Text      string `json:"text"`
	// Model は返事を書いたモデル名(最後の PLANNER_RESPONSE の model)。人間の発言では空。
	Model       string     `json:"model,omitempty"`
	RelatedTime time.Time  `json:"related_time"`
	UpdateTime  time.Time  `json:"update_time"`
	Items       []turnItem `json:"items"`
}

// turnItem は応答に付随する時系列要素。連続する同種の要素はまとめられている。
type turnItem struct {
	// Kind は "text"(モデルのテキスト) / "thinking" / "tools" / "notice"(システム発の注記)。
	Kind     string     `json:"kind"`
	Text     string     `json:"text,omitempty"`
	Thinking []string   `json:"thinking,omitempty"`
	Tools    []toolCall `json:"tools,omitempty"`
}

// turnItem.Kind の値。
const (
	itemText     = "text"
	itemThinking = "thinking"
	itemTools    = "tools"
	itemNotice   = "notice"
)

// toolCall は1回のツール呼び出し。入力の要約だけを持ち、実行結果は持たない。
type toolCall struct {
	Name    string `json:"name"`
	Summary string `json:"summary"`
}
