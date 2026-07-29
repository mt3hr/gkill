package main

import (
	"encoding/json"
	"time"
)

const repName = "ClaudeCode"
const dataType = "claude_code_turn"

// probeRecord はファイル種別判定に使う最小限のフィールド。
// history.jsonl のようにtimestampが数値のファイルも混ざりうるので、
// 判定時は時刻フィールドを持たない構造体でパースする。
type probeRecord struct {
	Type        string `json:"type"`
	SessionID   string `json:"sessionId"`
	IsSidechain bool   `json:"isSidechain"`
	AgentID     string `json:"agentId"`
}

// logRecord はJSONLの1行のうち、このプラグインが必要とするフィールドだけを取り出したもの。
// Claude Codeのトランスクリプトは多数のレコード種別を含むが、大半は無視する。
type logRecord struct {
	Type          string          `json:"type"`
	UUID          string          `json:"uuid"`
	Timestamp     time.Time       `json:"timestamp"`
	SessionID     string          `json:"sessionId"`
	Cwd           string          `json:"cwd"`
	GitBranch     string          `json:"gitBranch"`
	IsSidechain   bool            `json:"isSidechain"`
	AgentID       string          `json:"agentId"`
	PromptSource  string          `json:"promptSource"`
	AITitle       string          `json:"aiTitle"`
	Message       *logMessage     `json:"message"`
	ToolUseResult json.RawMessage `json:"toolUseResult"`
}

// logMessage は user/assistant レコードのmessage部。
// contentは文字列の場合とブロック配列の場合があるのでRawMessageで受ける。
type logMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

// contentBlock は message.content が配列だったときの1要素。
type contentBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text"`
	Thinking  string          `json:"thinking"`
	Name      string          `json:"name"`
	ID        string          `json:"id"`
	Input     json.RawMessage `json:"input"`
	ToolUseID string          `json:"tool_use_id"`
}

// toolUseResultAgent は Agent ツールの結果からサブエージェントIDを取り出すための構造体。
type toolUseResultAgent struct {
	AgentID string `json:"agentId"`
}

// agentMeta は agent-<ID>.meta.json の内容。
// toolUseId が親トランスクリプトの tool_use.id と直接対応する。
type agentMeta struct {
	AgentType   string `json:"agentType"`
	Description string `json:"description"`
	ToolUseID   string `json:"toolUseId"`
	SpawnDepth  int    `json:"spawnDepth"`
}

// 発言者。
const (
	roleHuman     = "human"
	roleAssistant = "assistant"
)

// message は1つのKyouになる単位。
// ChatGPT / Claude.ai プラグインと粒度を揃えており、次の2種類がある。
//
//	roleHuman     : 人間の発言1つ。本文は Text に入る
//	roleAssistant : その発言に対する一連の応答。次の人間の発言までをまとめて1つにする。
//	                テキスト・thinking・ツール実行・システム通知が時系列で Items に入る
//
// Claude の応答はツール実行を挟んで何度も分かれるが、まとめて1件として扱う。
type message struct {
	ID           string     `json:"id"`
	Role         string     `json:"role"`
	SessionID    string     `json:"session_id"`
	SessionTitle string     `json:"session_title"`
	Project      string     `json:"project"`
	Branch       string     `json:"branch"`
	Text         string     `json:"text"`
	RelatedTime  time.Time  `json:"related_time"`
	UpdateTime   time.Time  `json:"update_time"`
	Items        []turnItem `json:"items"`
}

// turnItem は発言に付随する時系列要素。連続する同種の要素はまとめられている。
type turnItem struct {
	// Kind は "text"(Claudeのテキスト応答) / "thinking" / "tools" / "notice"(システム通知)。
	Kind     string     `json:"kind"`
	Text     string     `json:"text,omitempty"`
	Thinking []string   `json:"thinking,omitempty"`
	Tools    []toolCall `json:"tools,omitempty"`
}

// toolCall は1回のツール実行。入力は要約のみ保持し、実行結果は保持しない。
type toolCall struct {
	Name    string    `json:"name"`
	Summary string    `json:"summary"`
	Agent   *subAgent `json:"agent,omitempty"`
}

// subAgent は Agent ツールで起動されたサブエージェントの会話。
type subAgent struct {
	AgentID     string     `json:"agent_id"`
	AgentType   string     `json:"agent_type"`
	Description string     `json:"description"`
	Prompt      string     `json:"prompt"`
	Items       []turnItem `json:"items"`
}
