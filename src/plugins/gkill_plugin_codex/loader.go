package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"slices"
	"sort"
	"strconv"
	"strings"
)

// 保持するレコードの種別。ここに無いものは行ごと読み捨てる。
//
// 実データではバイトの9割超がツールの実行結果で、それは1バイトも要らない。
// 保持するのは会話・思考・ツールの「呼び出し」・変更ファイル・計画だけ。
var keepPayloadKinds = map[string]struct{}{
	outerEventMsg + "/user_message":       {},
	outerEventMsg + "/agent_message":      {},
	outerEventMsg + "/agent_reasoning":    {},
	outerEventMsg + "/patch_apply_end":    {},
	outerEventMsg + "/sub_agent_activity": {},
	outerEventMsg + "/task_complete":      {},
	outerEventMsg + "/task_started":       {},
	outerEventMsg + "/item_completed":     {},
	outerEventMsg + "/turn_aborted":       {},
	outerEventMsg + "/context_compacted":  {},

	outerResponseItem + "/function_call":    {},
	outerResponseItem + "/custom_tool_call": {},
	outerResponseItem + "/web_search_call":  {},
	outerResponseItem + "/tool_search_call": {},
}

// 中身を見るまでもなく捨ててよい外側の種別。
//
// compacted は圧縮前の履歴をまるごと抱えていて、拾うと会話が二重になる。
// world_state は環境のスナップショット、inter_agent_communication_metadata は
// フラグ1つしかない。token_usage_record はトークン数(event_msg/token_count と同じく要らない)。
var skipOuterKinds = map[string]struct{}{
	"compacted":                          {},
	"world_state":                        {},
	"inter_agent_communication_metadata": {},
	"token_usage_record":                 {},
}

// keepForBuild は行を保持するかどうかを先頭だけを見て決める。
//
// 判定できなかったものは「捨てる」ではなく「拾う」。
// キー順や種別名が変わった日に会話が静かに消えるのを防ぐためで、
// 劣化しても maxRecordBytes までしか持たないので上限は効いている。
func keepForBuild(h recordHead) bool {
	switch h.Outer {
	case "":
		// 外側すら取り出せなかった。空行かもしれないが捨てない
		return true
	case outerSessionMeta, outerTurnContext:
		// payload.type を持たない。常に必要
		return true
	case outerEventMsg, outerResponseItem:
		if h.Payload == "" {
			// 先頭 headScanBytes 内に payload.type が無かった。本パースに委ねる
			return true
		}
		if _, ok := keepPayloadKinds[h.Outer+"/"+h.Payload]; !ok {
			return false
		}
		return keepCompletedItem(h)
	}
	if _, skip := skipOuterKinds[h.Outer]; skip {
		return false
	}
	// 知らない外側の種別。将来 Codex が増やしたものかもしれないので拾い、
	// 何が来たかは UnknownKinds に控えて設定画面に出す。
	return true
}

// skipCompletedItems は item_completed のうち、中身を見るまでもなく捨ててよい項目。
//
// 0.151 以降はツールの実行結果もここに載り、item_completed のバイトの大半を占める。
// ツールの「呼び出し」は response_item 側(custom_tool_call / function_call)から拾っているので、
// これらは1バイトも要らない。
var skipCompletedItems = map[string]struct{}{
	"CommandExecution": {}, // aggregated_output にコマンドの出力を丸ごと持つ
	"McpToolCall":      {}, // result に MCP ツールの応答を持つ
	"ImageView":        {},
}

// keepCompletedItem は item_completed の行を項目の種類で選ぶ。item_completed 以外は常に拾う。
//
// 判定できなかったもの(item.type や kind が先頭に無い、知らない種類)は拾う。
// Extension は Web 検索だけを拾う。画像生成は1行に画像そのものを抱えていて 2MiB を超える。
func keepCompletedItem(h recordHead) bool {
	if h.Kind() != outerEventMsg+"/item_completed" {
		return true
	}
	if _, skip := skipCompletedItems[h.Item]; skip {
		return false
	}
	if h.Item == completedExtension && h.ItemKind != "" && h.ItemKind != extensionKindWebSearch {
		return false
	}
	return true
}

// rolloutMeta は1つのロールアウトファイルから分かるスレッドの素性。
type rolloutMeta struct {
	ThreadID       string
	ParentThreadID string
	IsSubAgent     bool
	AgentPath      string
	AgentNickname  string
	Cwd            string
	Branch         string
	RepoURL        string
	Originator     string
	CLIVersion     string
}

// parsedRollout は1ファイルを読み切った結果。
type parsedRollout struct {
	Meta         rolloutMeta
	Items        []threadItem
	Dropped      int
	UnknownKinds []string
}

// parseRollout はロールアウトJSONLを1回のストリームで読み、
// スレッドの素性と正規化した要素列を返す。
//
// session_meta が1つも無いファイルは Meta.ThreadID が空で返る(＝対象外)。
func parseRollout(filePath string) (parsedRollout, error) {
	result := parsedRollout{}

	file, err := os.Open(filePath)
	if err != nil {
		return result, fmt.Errorf("error at open %s: %w", filePath, err)
	}
	defer func() { _ = file.Close() }()

	reader := newLineReader(file)
	unknown := map[string]struct{}{}
	metaSeen := false
	currentModel := ""
	seq := int64(-1)
	// currentTurn は今のターンの turn_id(task_started / turn_context から追う)。
	// legacyTurns は旧レーンの会話が出たターン。そのターンでは新レーンの会話を捨てる。
	currentTurn := ""
	legacyTurns := map[string]struct{}{}

	for {
		head, data, dropped, readErr := reader.next(keepForBuild)
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				break
			}
			return result, fmt.Errorf("error at read %s: %w", filePath, readErr)
		}
		seq++

		if dropped {
			result.Dropped++
			// 落としたのが人間の発言なら、必ず場所だけは残す。
			// 抜けると応答の境目がずれて ordinal が動き、KyouIDが変わってしまう。
			// 新レーンでは item.type まで見ないと人間の発言か分からない。
			legacyUser := head.Kind() == outerEventMsg+"/user_message"
			itemUserMessage := head.Kind() == outerEventMsg+"/item_completed" && head.Item == completedUserMessage
			if legacyUser || itemUserMessage {
				result.Items = append(result.Items, threadItem{
					Seq:  seq,
					Kind: itemUser,
					Text: "(この発言は大きすぎるため省略しました)",
				})
			}
			if legacyUser {
				legacyTurns[currentTurn] = struct{}{}
			}
			continue
		}
		if len(data) == 0 {
			continue
		}

		var envelope recordEnvelope
		if err := json.Unmarshal(data, &envelope); err != nil {
			// 壊れた行1つでファイルを諦めない
			continue
		}

		switch envelope.Type {
		case outerSessionMeta:
			applySessionMeta(&result.Meta, envelope.Payload, !metaSeen)
			metaSeen = true
			continue
		case outerTurnContext:
			var payload turnContextPayload
			if json.Unmarshal(envelope.Payload, &payload) != nil {
				continue
			}
			if payload.Cwd != "" && result.Meta.Cwd == "" {
				result.Meta.Cwd = payload.Cwd
			}
			if payload.TurnID != "" {
				currentTurn = payload.TurnID
			}
			// モデルが変わったときだけ印を置く。fold が現在のモデルを追える
			if payload.Model != "" && payload.Model != currentModel {
				currentModel = payload.Model
				result.Items = append(result.Items, threadItem{
					Seq:   seq,
					TS:    envelope.Timestamp.UTC(),
					Kind:  itemTurn,
					Name:  payload.Model,
					RefID: payload.TurnID,
				})
			}
			continue
		case outerEventMsg, outerResponseItem:
		default:
			if _, skip := skipOuterKinds[envelope.Type]; !skip && envelope.Type != "" {
				unknown[envelope.Type] = struct{}{}
			}
			continue
		}

		item, ok := itemFromPayload(envelope, seq, result.Meta.Cwd)
		if !ok {
			continue
		}
		switch {
		case item.Kind == itemTurnStart:
			if item.RefID != "" {
				currentTurn = item.RefID
			}
			continue
		case item.lane == laneLegacy:
			legacyTurns[currentTurn] = struct{}{}
		}
		result.Items = append(result.Items, item)
	}
	result.Items = dropItemLaneInLegacyTurns(result.Items, legacyTurns)

	if len(unknown) != 0 {
		result.UnknownKinds = make([]string, 0, len(unknown))
		for kind := range unknown {
			result.UnknownKinds = append(result.UnknownKinds, kind)
		}
		sort.Strings(result.UnknownKinds)
	}
	return result, nil
}

// dropItemLaneInLegacyTurns は、旧レーンの会話が出たターンから新レーンの会話を除く。
//
// 0.151 以降は新レーンにしか書かれないので、ふだんは何も落とさない。
// 両方を書く版が挟まっていたときや、旧版のセッションを新版で resume したファイルでも、
// ターンごとに片方だけを使えば会話が二重にならない。ファイル単位で選ぶと resume した後半が消える。
// 新レーンの項目の RefID には item_completed の turn_id が入っている。
func dropItemLaneInLegacyTurns(items []threadItem, legacyTurns map[string]struct{}) []threadItem {
	if len(legacyTurns) == 0 {
		return items
	}
	kept := items[:0]
	for _, item := range items {
		if item.lane == laneItem {
			if _, legacy := legacyTurns[item.RefID]; legacy {
				continue
			}
		}
		kept = append(kept, item)
	}
	return kept
}

// applySessionMeta は session_meta を素性へ反映する。
//
// identity(ID・出自・親)は1回目のものだけを使う。
// サブエージェントのファイルは2つ目に「親の」session_meta を持っているので、
// マージすると自分が親にすり替わる。
// environment(cwd・git・originator)は逆に、1つ目が空で2つ目に入っている例があるため
// 全 occurrence をマージして最初の非空を採る。
func applySessionMeta(meta *rolloutMeta, raw json.RawMessage, isFirst bool) {
	var payload sessionMetaPayload
	if json.Unmarshal(raw, &payload) != nil {
		return
	}

	if isFirst {
		meta.ThreadID = payload.ID
		meta.IsSubAgent = payload.ThreadSource == threadSourceSubAgent
		meta.ParentThreadID = payload.ParentThreadID
		if meta.ParentThreadID == "" {
			meta.ParentThreadID = payload.ForkedFromID
		}
		meta.AgentPath = payload.AgentPath
		meta.AgentNickname = payload.AgentNickname

		if spawn := parseSubagentSource(payload.Source); spawn != nil {
			meta.IsSubAgent = true
			if meta.ParentThreadID == "" {
				meta.ParentThreadID = spawn.ParentThreadID
			}
			if meta.AgentPath == "" {
				meta.AgentPath = spawn.AgentPath
			}
			if meta.AgentNickname == "" {
				meta.AgentNickname = spawn.AgentNickname
			}
		}
		// 古い版には thread_source も parent_thread_id も無い。
		// session_id が自分と違うならそれが親を指している
		if meta.ParentThreadID == "" && payload.SessionID != "" && payload.SessionID != payload.ID {
			meta.ParentThreadID = payload.SessionID
		}
	}

	setIfEmpty(&meta.Cwd, payload.Cwd)
	setIfEmpty(&meta.Originator, payload.Originator)
	setIfEmpty(&meta.CLIVersion, payload.CLIVersion)
	if payload.Git != nil {
		setIfEmpty(&meta.Branch, payload.Git.Branch)
		setIfEmpty(&meta.RepoURL, payload.Git.RepositoryURL)
	}
}

func setIfEmpty(target *string, value string) {
	if *target == "" && value != "" {
		*target = value
	}
}

type threadSpawn struct {
	ParentThreadID string
	AgentPath      string
	AgentNickname  string
}

// parseSubagentSource は session_meta.source からサブエージェントの生成情報を取り出す。
// source は文字列("vscode")のこともあるので、オブジェクトのときだけ見る。
func parseSubagentSource(raw json.RawMessage) *threadSpawn {
	if len(raw) == 0 || raw[0] != '{' {
		return nil
	}
	var source sourceSubagent
	if json.Unmarshal(raw, &source) != nil {
		return nil
	}
	if source.Subagent == nil || source.Subagent.ThreadSpawn == nil {
		return nil
	}
	spawn := source.Subagent.ThreadSpawn
	return &threadSpawn{
		ParentThreadID: spawn.ParentThreadID,
		AgentPath:      spawn.AgentPath,
		AgentNickname:  spawn.AgentNickname,
	}
}

// itemFromPayload は event_msg / response_item を正規化した1要素にする。
func itemFromPayload(envelope recordEnvelope, seq int64, cwd string) (threadItem, bool) {
	base := threadItem{Seq: seq, TS: envelope.Timestamp.UTC()}

	var kind struct {
		Type string `json:"type"`
	}
	if json.Unmarshal(envelope.Payload, &kind) != nil {
		return base, false
	}

	switch envelope.Type + "/" + kind.Type {
	case outerEventMsg + "/user_message":
		var payload userMessagePayload
		if json.Unmarshal(envelope.Payload, &payload) != nil {
			return base, false
		}
		return userItemOf(base, payload.Message, laneLegacy), true

	case outerEventMsg + "/agent_message":
		var payload agentMessagePayload
		if json.Unmarshal(envelope.Payload, &payload) != nil {
			return base, false
		}
		if strings.TrimSpace(payload.Message) == "" {
			return base, false
		}
		base.Kind = itemAssistant
		base.Name = payload.Phase
		base.Text = truncateRunes(strings.TrimSpace(payload.Message), maxTextRunes)
		base.lane = laneLegacy
		return base, true

	case outerEventMsg + "/agent_reasoning":
		var payload agentReasoningPayload
		if json.Unmarshal(envelope.Payload, &payload) != nil {
			return base, false
		}
		if strings.TrimSpace(payload.Text) == "" {
			return base, false
		}
		base.Kind = itemThinking
		base.Text = truncateRunes(strings.TrimSpace(payload.Text), maxTextRunes)
		base.lane = laneLegacy
		return base, true

	case outerResponseItem + "/function_call":
		var payload functionCallPayload
		if json.Unmarshal(envelope.Payload, &payload) != nil {
			return base, false
		}
		// Codex から人間への質問は、ツール一覧の1行(200字で切れる)ではなく会話として出す。
		// 回答は次の人間の発言として届く
		if strings.HasPrefix(payload.Name, "request_user_input") {
			if question := formatQuestions(payload.Arguments); question != "" {
				base.Kind = itemQuestion
				base.Name = payload.Name
				base.Text = question
				base.RefID = payload.CallID
				return base, true
			}
		}
		base.Kind = itemTool
		base.Name = payload.Name
		base.Text = summarizeFunctionCall(payload.Name, payload.Arguments)
		base.RefID = payload.CallID
		return base, true

	case outerResponseItem + "/custom_tool_call":
		var payload customToolCallPayload
		if json.Unmarshal(envelope.Payload, &payload) != nil {
			return base, false
		}
		base.Kind = itemTool
		base.Name = payload.Name
		base.Text = summarizeCustomToolCall(payload.Name, payload.Input)
		base.RefID = payload.CallID
		return base, true

	case outerResponseItem + "/web_search_call":
		var payload webSearchCallPayload
		if json.Unmarshal(envelope.Payload, &payload) != nil {
			return base, false
		}
		base.Kind = itemTool
		base.Name = "web_search"
		base.Text = truncateRunes(oneLine(stringFieldOf(payload.Action, "query")), maxToolSummaryRunes)
		base.lane = laneLegacy
		return base, true

	case outerResponseItem + "/tool_search_call":
		var payload struct {
			CallID    string          `json:"call_id"`
			Arguments json.RawMessage `json:"arguments"`
		}
		if json.Unmarshal(envelope.Payload, &payload) != nil {
			return base, false
		}
		base.Kind = itemTool
		base.Name = "tool_search"
		base.Text = truncateRunes(oneLine(stringFieldOf(payload.Arguments, "query")), maxToolSummaryRunes)
		base.RefID = payload.CallID
		return base, true

	case outerEventMsg + "/patch_apply_end":
		var payload patchApplyEndPayload
		if json.Unmarshal(envelope.Payload, &payload) != nil {
			return base, false
		}
		files := patchFilesOf(payload.Changes, cwd)
		if len(files) == 0 {
			return base, false
		}
		encoded, err := json.Marshal(files)
		if err != nil {
			return base, false
		}
		base.Kind = itemPatch
		base.RefID = payload.CallID
		base.Extra = string(encoded)
		base.lane = laneLegacy
		return base, true

	case outerEventMsg + "/item_completed":
		var payload itemCompletedPayload
		if json.Unmarshal(envelope.Payload, &payload) != nil {
			return base, false
		}
		return itemFromCompleted(base, payload, cwd)

	case outerEventMsg + "/task_started":
		var payload struct {
			TurnID string `json:"turn_id"`
		}
		if json.Unmarshal(envelope.Payload, &payload) != nil {
			return base, false
		}
		base.Kind = itemTurnStart
		base.RefID = payload.TurnID
		return base, true

	case outerEventMsg + "/sub_agent_activity":
		var payload subAgentActivityPayload
		if json.Unmarshal(envelope.Payload, &payload) != nil {
			return base, false
		}
		if payload.Kind != "started" || payload.AgentThreadID == "" {
			return base, false
		}
		base.Kind = itemSpawn
		base.Name = payload.AgentPath
		base.RefID = payload.AgentThreadID
		base.Extra = payload.EventID // 親のツール呼び出しの call_id
		return base, true

	case outerEventMsg + "/task_complete":
		var payload struct {
			TurnID     string `json:"turn_id"`
			DurationMs int64  `json:"duration_ms"`
		}
		if json.Unmarshal(envelope.Payload, &payload) != nil || payload.DurationMs <= 0 {
			return base, false
		}
		base.Kind = itemTaskDone
		base.RefID = payload.TurnID
		base.Text = strconv.FormatInt(payload.DurationMs, 10)
		return base, true

	case outerEventMsg + "/turn_aborted":
		var payload turnAbortedPayload
		if json.Unmarshal(envelope.Payload, &payload) != nil {
			return base, false
		}
		base.Kind = itemNotice
		base.Text = "中断しました"
		if payload.Reason != "" {
			base.Text += " (" + truncateRunes(oneLine(payload.Reason), maxNoticeRunes) + ")"
		}
		return base, true

	case outerEventMsg + "/context_compacted":
		base.Kind = itemNotice
		base.Text = noticeContextCompacted
		base.lane = laneLegacy
		return base, true
	}

	return base, false
}

// noticeContextCompacted はコンテキスト圧縮の通知文。旧レーンと新レーンで揃える。
const noticeContextCompacted = "コンテキストを圧縮しました"

// itemFromCompleted は item_completed の項目を正規化した1要素にする。
//
// 0.151 以降の Codex は会話をここにしか書かない(旧レーンの user_message などは出ない)。
// 計画(Plan)だけは旧版のころからここに載っていたので、旧レーンとの重複判定の対象にしない。
// 新レーンの項目の RefID には turn_id を入れる(重複判定に使う)。
func itemFromCompleted(base threadItem, payload itemCompletedPayload, cwd string) (threadItem, bool) {
	var head completedItemHead
	if json.Unmarshal(payload.Item, &head) != nil {
		return base, false
	}
	base.RefID = payload.TurnID

	switch head.Type {
	case completedUserMessage:
		var item completedMessageItem
		if json.Unmarshal(payload.Item, &item) != nil {
			return base, false
		}
		return userItemOf(base, joinContentText(item.Content), laneItem), true

	case completedAgentMessage:
		var item completedMessageItem
		if json.Unmarshal(payload.Item, &item) != nil {
			return base, false
		}
		text := strings.TrimSpace(joinContentText(item.Content))
		if text == "" {
			return base, false
		}
		base.Kind = itemAssistant
		base.Name = item.Phase
		base.Text = truncateRunes(text, maxTextRunes)
		base.lane = laneItem
		return base, true

	case completedReasoning:
		var item completedReasoningItem
		if json.Unmarshal(payload.Item, &item) != nil {
			return base, false
		}
		text := strings.TrimSpace(strings.Join(item.SummaryText, "\n\n"))
		if text == "" {
			return base, false
		}
		base.Kind = itemThinking
		base.Text = truncateRunes(text, maxTextRunes)
		base.lane = laneItem
		return base, true

	case completedFileChange:
		var item completedFileChangeItem
		if json.Unmarshal(payload.Item, &item) != nil {
			return base, false
		}
		files := patchFilesOf(item.Changes, cwd)
		if len(files) == 0 {
			return base, false
		}
		encoded, err := json.Marshal(files)
		if err != nil {
			return base, false
		}
		base.Kind = itemPatch
		base.Extra = string(encoded)
		base.lane = laneItem
		return base, true

	case completedContextCompaction:
		base.Kind = itemNotice
		base.Text = noticeContextCompacted
		base.lane = laneItem
		return base, true

	case completedExtension:
		var item completedExtensionItem
		if json.Unmarshal(payload.Item, &item) != nil || item.Kind != extensionKindWebSearch {
			return base, false
		}
		query := item.Query
		if query == "" {
			query = stringFieldOf(item.Action, "query")
		}
		base.Kind = itemTool
		base.Name = "web_search"
		base.Text = truncateRunes(oneLine(query), maxToolSummaryRunes)
		base.lane = laneItem
		return base, true
	}

	if strings.EqualFold(head.Type, completedPlan) {
		var item completedPlanItem
		if json.Unmarshal(payload.Item, &item) != nil || strings.TrimSpace(item.Text) == "" {
			return base, false
		}
		base.Kind = itemPlan
		base.Text = truncateRunes(strings.TrimSpace(item.Text), maxPlanRunes)
		return base, true
	}
	return base, false
}

// imageMarker は発言に添付された画像の置き場所の印。画像そのものは持たない。
const imageMarker = "(画像)"

// joinContentText は UserMessage / AgentMessage の本文の片を1つの文字列にする。
// type は UserMessage では text、AgentMessage では Text なので大小を無視して比べる。
func joinContentText(parts []completedContentPart) string {
	var lines []string
	for _, part := range parts {
		switch {
		case strings.EqualFold(part.Type, "text"):
			lines = append(lines, part.Text)
		case strings.Contains(strings.ToLower(part.Type), "image"):
			lines = append(lines, imageMarker)
		}
	}
	return strings.Join(lines, "\n")
}

// userItemOf は人間の発言1つを要素にする。旧レーンと新レーンで同じ扱いにする。
func userItemOf(base threadItem, raw string, lane string) threadItem {
	body, preamble, isReply := normalizeUserText(raw)
	base.Kind = itemUser
	base.Text = truncateRunes(strings.TrimSpace(body), maxTextRunes)
	base.lane = lane
	if isReply {
		base.Name = userKindQuestionReply
	}
	if preamble != nil {
		if encoded, err := json.Marshal(preamble); err == nil {
			base.Extra = string(encoded)
		}
	}
	return base
}

// 発言を包むタグ。
const (
	// questionReplyTag は Codex の質問(request_user_input)への回答を丸ごと包む。
	questionReplyTag = "send_user_message_question_reply"
	// browserContextTag はアプリ内ブラウザが発言の先頭に付ける、表示中のページの情報。
	browserContextTag = "in-app-browser-context"
)

// normalizeUserText は発言から前置きと包みを剥がす。
//
// rykv は一覧の行に詳細HTMLをそのまま描くので、前置きを本文に残すとどの行も定型文で埋まる。
// 剥がした前置きは詳細ビューの折りたたみへ入れる。
func normalizeUserText(raw string) (body string, preamble *ideContext, isReply bool) {
	body = raw
	if inner, rest, ok := stripWrapperTag(body, questionReplyTag); ok {
		isReply = true
		body = answersOfQuestionReply(inner)
		if rest != "" {
			body += "\n" + rest
		}
	}

	browser := ""
	if inner, rest, ok := stripWrapperTag(body, browserContextTag); ok {
		browser = inner
		body = rest
	}

	body, preamble = stripIDEContext(body)
	if browser != "" {
		if preamble == nil {
			preamble = &ideContext{}
		}
		preamble.BrowserContext = truncateRunes(browser, maxPreambleRunes)
	}
	return body, preamble, isReply
}

// answersOfQuestionReply は質問への回答の包みの中身を、人が読む回答文にする。
//
// 中身は [{"questionItemId", "question", "answer"}] の JSON 配列。設問は直前の返事に出ているので、
// 設問が1つなら回答だけを本文にする。複数なら「設問 → 回答」を並べる。
// JSON でない・回答が1つも無いときは、中身をそのまま返す(知らない形で本文を消さない)。
func answersOfQuestionReply(inner string) string {
	var replies []struct {
		Question string          `json:"question"`
		Answer   json.RawMessage `json:"answer"`
	}
	if json.Unmarshal([]byte(inner), &replies) != nil || len(replies) == 0 {
		return inner
	}
	if len(replies) == 1 {
		if answer := answerTextOf(replies[0].Answer); answer != "" {
			return answer
		}
		return inner
	}
	var parts []string
	for _, reply := range replies {
		answer := answerTextOf(reply.Answer)
		if answer == "" {
			continue
		}
		parts = append(parts, strings.TrimSpace(reply.Question)+"\n→ "+answer)
	}
	if len(parts) == 0 {
		return inner
	}
	return strings.Join(parts, "\n\n")
}

// answerTextOf は回答1つを文字列にする。文字列のほか、複数選択の配列にも備える。
func answerTextOf(raw json.RawMessage) string {
	var text string
	if json.Unmarshal(raw, &text) == nil {
		return strings.TrimSpace(text)
	}
	var list []string
	if json.Unmarshal(raw, &list) == nil {
		return strings.TrimSpace(strings.Join(list, "、"))
	}
	return strings.TrimSpace(string(raw))
}

// stripWrapperTag は <name>…</name> で始まる発言から、タグの中身と後ろの残りを取り出す。
// 閉じタグが無いなど知らない形なら ok=false で、何も剥がさない。
func stripWrapperTag(msg, name string) (inner string, rest string, ok bool) {
	trimmed := strings.TrimLeft(msg, " \t\r\n")
	open := "<" + name
	if !strings.HasPrefix(trimmed, open) || len(trimmed) == len(open) {
		return "", msg, false
	}
	// <name-foo> のような別のタグを取り違えない
	if next := trimmed[len(open)]; next != '>' && next != ' ' {
		return "", msg, false
	}
	openEnd := strings.IndexByte(trimmed, '>')
	closeTag := "</" + name + ">"
	closeAt := strings.Index(trimmed, closeTag)
	if openEnd < 0 || closeAt < openEnd {
		return "", msg, false
	}
	inner = strings.TrimSpace(trimmed[openEnd+1 : closeAt])
	rest = strings.TrimSpace(trimmed[closeAt+len(closeTag):])
	return inner, rest, true
}

// formatQuestions は request_user_input の引数を、人が読む質問文にする。
// 形が分からなければ空を返す(呼び出し側はふつうのツール呼び出しとして扱う)。
func formatQuestions(arguments string) string {
	var args struct {
		Questions []struct {
			Title    string            `json:"title"`
			Question string            `json:"question"`
			Options  []json.RawMessage `json:"options"`
		} `json:"questions"`
	}
	if json.Unmarshal([]byte(arguments), &args) != nil {
		return ""
	}

	var builder strings.Builder
	for _, question := range args.Questions {
		title := strings.TrimSpace(question.Title)
		if title == "" {
			title = strings.TrimSpace(question.Question)
		}
		var options []string
		for _, raw := range question.Options {
			if label := optionLabelOf(raw); label != "" {
				options = append(options, label)
			}
		}
		if title == "" && len(options) == 0 {
			continue
		}
		if builder.Len() != 0 {
			builder.WriteString("\n\n")
		}
		builder.WriteString(title)
		for _, option := range options {
			builder.WriteString("\n・")
			builder.WriteString(option)
		}
	}
	return truncateRunes(builder.String(), maxTextRunes)
}

// optionLabelOf は選択肢1つの表示名を返す。文字列のこともオブジェクトのこともある。
func optionLabelOf(raw json.RawMessage) string {
	var label string
	if json.Unmarshal(raw, &label) == nil {
		return strings.TrimSpace(label)
	}
	var object map[string]any
	if json.Unmarshal(raw, &object) != nil {
		return ""
	}
	for _, key := range []string{"label", "title", "text", "value"} {
		if value, ok := object[key].(string); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

// stringFieldOf は JSON オブジェクトから文字列フィールドを1つ取り出す。
func stringFieldOf(raw json.RawMessage, key string) string {
	if len(raw) == 0 {
		return ""
	}
	var object map[string]any
	if json.Unmarshal(raw, &object) != nil {
		return ""
	}
	if value, ok := object[key].(string); ok {
		return value
	}
	return ""
}

const (
	ideContextPrefix = "# Context from my IDE setup:"
	ideRequestMarker = "## My request for Codex:"
	ideActiveMarker  = "## Active file:"
	ideTabsMarker    = "## Open tabs:"
)

// stripIDEContext は VSCode拡張が付ける前置きを本文から剥がす。
//
// 実データでは6割ほどにこれが付いている。
// rykv は一覧の行に詳細HTMLをそのまま描くので、前置きを本文の先頭に残すと
// どの行も「開いているタブ一覧」で埋まって読めなくなる。
//
// 形は3通り(Active+タブ / タブのみ / Activeのみ)あるが、
// いずれも "## My request for Codex:" を持つ。マーカーが無いときは何もしない。
func stripIDEContext(msg string) (string, *ideContext) {
	if !strings.HasPrefix(msg, ideContextPrefix) {
		return msg, nil
	}
	markerAt := strings.Index(msg, ideRequestMarker)
	if markerAt < 0 {
		// 知らない形。捨てずにそのまま返す
		return msg, nil
	}

	body := strings.TrimPrefix(msg[markerAt+len(ideRequestMarker):], "\n")
	header := msg[:markerAt]

	result := &ideContext{}
	inTabs := false
	for line := range strings.SplitSeq(header, "\n") {
		line = strings.TrimSuffix(line, "\r")
		switch {
		case strings.HasPrefix(line, ideActiveMarker):
			result.ActiveFile = strings.TrimSpace(strings.TrimPrefix(line, ideActiveMarker))
			inTabs = false
		case strings.HasPrefix(line, ideTabsMarker):
			inTabs = true
		case inTabs && strings.HasPrefix(line, "- "):
			entry := strings.TrimPrefix(line, "- ")
			if name, filePath, found := strings.Cut(entry, ": "); found {
				result.OpenTabs = append(result.OpenTabs, ideTab{Name: name, Path: filePath})
			} else {
				result.OpenTabs = append(result.OpenTabs, ideTab{Name: entry})
			}
		}
	}

	if result.ActiveFile == "" && len(result.OpenTabs) == 0 {
		return body, nil
	}
	return body, result
}

// ツール引数から代表的な1フィールドを選ぶときの優先順。
var toolSummaryFields = []string{
	"command", "cmd", "description", "agent_path", "prompt", "message",
	"file_path", "path", "notebook_path", "pattern", "query", "url",
	"plan", "text", "chars", "name", "skill", "args",
}

// summarizeFunctionCall は function_call の引数を1行に潰す。
// arguments は JSON を文字列にしたものなので、もう一段デコードが要る。
// ツール名は呼び出し側が別に表示するので、ここでは要約に含めない。
func summarizeFunctionCall(_ string, arguments string) string {
	var object map[string]any
	if json.Unmarshal([]byte(arguments), &object) != nil {
		return truncateRunes(oneLine(arguments), maxToolSummaryRunes)
	}
	if summary := pickSummaryField(object); summary != "" {
		return truncateRunes(oneLine(summary), maxToolSummaryRunes)
	}
	return truncateRunes(oneLine(arguments), maxToolSummaryRunes)
}

// summarizeCustomToolCall は custom_tool_call の入力を1行に潰す。
// input は JSON ではなく生の文字列(スクリプトやパッチ)。
func summarizeCustomToolCall(name, input string) string {
	if name == "apply_patch" {
		if summary := summarizeApplyPatch(input); summary != "" {
			return summary
		}
	}
	return truncateRunes(oneLine(input), maxToolSummaryRunes)
}

// summarizeApplyPatch は apply_patch の本文からファイル一覧を作る。
// 生のまま切ると "*** Begin Patch" しか見えないので、対象ファイルを拾う。
func summarizeApplyPatch(input string) string {
	markers := map[string]string{
		"*** Update File: ": "更新",
		"*** Add File: ":    "追加",
		"*** Delete File: ": "削除",
	}
	var files []string
	for line := range strings.SplitSeq(input, "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		for marker := range markers {
			if strings.HasPrefix(line, marker) {
				files = append(files, path.Base(strings.ReplaceAll(strings.TrimPrefix(line, marker), `\`, "/")))
				break
			}
		}
	}
	if len(files) == 0 {
		return ""
	}
	return truncateRunes(fmt.Sprintf("%dファイル: %s", len(files), strings.Join(files, ", ")), maxToolSummaryRunes)
}

func pickSummaryField(object map[string]any) string {
	for _, key := range toolSummaryFields {
		value, ok := object[key]
		if !ok {
			continue
		}
		switch typed := value.(type) {
		case string:
			if typed != "" {
				return typed
			}
		case []any:
			parts := make([]string, 0, len(typed))
			for _, element := range typed {
				if s, ok := element.(string); ok {
					parts = append(parts, s)
				}
			}
			if len(parts) != 0 {
				return strings.Join(parts, " ")
			}
		}
	}
	return ""
}

// patchFilesOf は changes(旧レーンの patch_apply_end、新レーンの FileChange)を
// 「変更したファイル一覧」に潰す。
//
// unified diff も新しいファイルの本文も保存しない。実データでは changes のJSONが中央値で十数KB・最大で数十KBあり、
// 日記として要るのは「どのファイルを触ったか」だけ。
func patchFilesOf(changes map[string]patchChange, cwd string) []patchFile {
	if len(changes) == 0 {
		return nil
	}
	paths := make([]string, 0, len(changes))
	for filePath := range changes {
		paths = append(paths, filePath)
	}
	sort.Strings(paths)
	if len(paths) > maxPatchFiles {
		paths = paths[:maxPatchFiles]
	}

	files := make([]patchFile, 0, len(paths))
	for _, filePath := range paths {
		change := changes[filePath]
		added, removed := changeCounts(change)
		files = append(files, patchFile{
			Path:    relativizePath(filePath, cwd),
			Type:    change.Type,
			Added:   added,
			Removed: removed,
		})
	}
	return files
}

// changeCounts は1ファイルぶんの増減行数を数える。
// unified diff があればそれを、無ければ(新レーンの add / delete)本文の行数を数える。
func changeCounts(change patchChange) (added int, removed int) {
	if change.UnifiedDiff != "" {
		return diffCounts(change.UnifiedDiff)
	}
	lines := lineCount(change.Content)
	switch change.Type {
	case "add":
		return lines, 0
	case "delete":
		return 0, lines
	}
	return 0, 0
}

// lineCount は行数を数える。末尾に改行が無い最後の行も1行と数える。
func lineCount(s string) int {
	if s == "" {
		return 0
	}
	count := strings.Count(s, "\n")
	if !strings.HasSuffix(s, "\n") {
		count++
	}
	return count
}

// diffCounts は unified diff の増減行数を数える。
func diffCounts(unified string) (added int, removed int) {
	for line := range strings.SplitSeq(unified, "\n") {
		switch {
		case strings.HasPrefix(line, "+++"), strings.HasPrefix(line, "---"):
			continue
		case strings.HasPrefix(line, "+"):
			added++
		case strings.HasPrefix(line, "-"):
			removed++
		}
	}
	return added, removed
}

// relativizePath は絶対パスを cwd からの相対にする。
// Codex のログには "c:\..." と "C:\..." が混在するので大小を無視して比較する。
func relativizePath(absolute, cwd string) string {
	normalized := strings.ReplaceAll(absolute, `\`, "/")
	if cwd == "" {
		return normalized
	}
	base := strings.TrimRight(strings.ReplaceAll(cwd, `\`, "/"), "/")
	if base == "" {
		return normalized
	}
	if len(normalized) > len(base) && strings.EqualFold(normalized[:len(base)], base) {
		return strings.TrimLeft(normalized[len(base):], "/")
	}
	return normalized
}

// oneLine は改行とタブを空白に潰し、連続する空白を1つにする。
func oneLine(s string) string {
	replaced := strings.NewReplacer("\r\n", " ", "\r", " ", "\n", " ", "\t", " ").Replace(s)
	return strings.Join(strings.Fields(replaced), " ")
}

// truncateRunes は文字数で切り、切ったら末尾に … を付ける。
func truncateRunes(s string, limit int) string {
	if limit <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= limit {
		return s
	}
	return string(runes[:limit]) + "…"
}

// lastPathElement はパスの末尾要素を返す。
//
// filepath.Base を使わないのは、Windows で書かれたログを Linux で読むと
// 区切りが解釈されず、パス全体がプロジェクト名になってしまうため。
func lastPathElement(p string) string {
	trimmed := strings.TrimRight(p, `/\`)
	if trimmed == "" {
		return ""
	}
	if i := strings.LastIndexAny(trimmed, `/\`); i >= 0 {
		return trimmed[i+1:]
	}
	return trimmed
}

// dedupeStrings は空文字を除いて重複を落とす。順序は保つ。
func dedupeStrings(values []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return slices.Clip(result)
}
