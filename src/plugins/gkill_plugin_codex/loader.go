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
// 実データではバイトの94.7%がツールの実行結果で、それは1バイトも要らない。
// 保持するのは会話・思考・ツールの「呼び出し」・変更ファイル・計画だけ。
var keepPayloadKinds = map[string]struct{}{
	outerEventMsg + "/user_message":       {},
	outerEventMsg + "/agent_message":      {},
	outerEventMsg + "/agent_reasoning":    {},
	outerEventMsg + "/patch_apply_end":    {},
	outerEventMsg + "/sub_agent_activity": {},
	outerEventMsg + "/task_complete":      {},
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
// フラグ1つしかない。
var skipOuterKinds = map[string]struct{}{
	"compacted":                          {},
	"world_state":                        {},
	"inter_agent_communication_metadata": {},
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
		_, ok := keepPayloadKinds[h.Outer+"/"+h.Payload]
		return ok
	}
	if _, skip := skipOuterKinds[h.Outer]; skip {
		return false
	}
	// 知らない外側の種別。将来 Codex が増やしたものかもしれないので拾い、
	// 何が来たかは UnknownKinds に控えて設定画面に出す。
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
			if head.Kind() == outerEventMsg+"/user_message" {
				result.Items = append(result.Items, threadItem{
					Seq:  seq,
					Kind: itemUser,
					Text: "(この発言は大きすぎるため省略しました)",
				})
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
		result.Items = append(result.Items, item)
	}

	if len(unknown) != 0 {
		result.UnknownKinds = make([]string, 0, len(unknown))
		for kind := range unknown {
			result.UnknownKinds = append(result.UnknownKinds, kind)
		}
		sort.Strings(result.UnknownKinds)
	}
	return result, nil
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
		body, ideCtx := stripIDEContext(payload.Message)
		base.Kind = itemUser
		base.Text = truncateRunes(strings.TrimSpace(body), maxTextRunes)
		if ideCtx != nil {
			if encoded, err := json.Marshal(ideCtx); err == nil {
				base.Extra = string(encoded)
			}
		}
		return base, true

	case outerEventMsg + "/agent_message":
		var payload agentMessagePayload
		if json.Unmarshal(envelope.Payload, &payload) != nil {
			return base, false
		}
		if strings.TrimSpace(payload.Message) == "" {
			return base, false
		}
		base.Kind = itemAssistant
		base.Text = truncateRunes(strings.TrimSpace(payload.Message), maxTextRunes)
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
		return base, true

	case outerResponseItem + "/function_call":
		var payload functionCallPayload
		if json.Unmarshal(envelope.Payload, &payload) != nil {
			return base, false
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
		files := patchFilesOf(payload, cwd)
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
		return base, true

	case outerEventMsg + "/item_completed":
		var payload itemCompletedPayload
		if json.Unmarshal(envelope.Payload, &payload) != nil {
			return base, false
		}
		if !strings.EqualFold(payload.Item.Type, "plan") || strings.TrimSpace(payload.Item.Text) == "" {
			return base, false
		}
		base.Kind = itemPlan
		base.Text = truncateRunes(strings.TrimSpace(payload.Item.Text), maxPlanRunes)
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
		base.Text = "コンテキストを圧縮しました"
		return base, true
	}

	return base, false
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
// 実データでは178件中108件にこれが付いている。
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

// patchFilesOf は patch_apply_end の changes を「変更したファイル一覧」に潰す。
//
// unified diff は保存しない。実データでは changes のJSONが中央値11.5KB・最大87KBあり、
// 日記として要るのは「どのファイルを触ったか」だけ。
func patchFilesOf(payload patchApplyEndPayload, cwd string) []patchFile {
	if len(payload.Changes) == 0 {
		return nil
	}
	paths := make([]string, 0, len(payload.Changes))
	for filePath := range payload.Changes {
		paths = append(paths, filePath)
	}
	sort.Strings(paths)
	if len(paths) > maxPatchFiles {
		paths = paths[:maxPatchFiles]
	}

	files := make([]patchFile, 0, len(paths))
	for _, filePath := range paths {
		change := payload.Changes[filePath]
		added, removed := diffCounts(change.UnifiedDiff)
		files = append(files, patchFile{
			Path:    relativizePath(filePath, cwd),
			Type:    change.Type,
			Added:   added,
			Removed: removed,
		})
	}
	return files
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
