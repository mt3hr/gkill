package main

import (
	"bufio"
	"bytes"
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// maxLineBytes は1行の上限。超えた行は読み捨てて数える。
// 1行はステップ1つで、ツールの出力が丸ごと入ることがあるため上限を置く。
const maxLineBytes = 8 * 1024 * 1024

// parsedTranscript は transcript_full.jsonl 1本を読んだ結果。
type parsedTranscript struct {
	ConversationID string
	Messages       []message
	StepCount      int
	// DroppedLines は JSON として読めなかった行と長すぎた行の数。設定画面に出す。
	DroppedLines int
}

// parseTranscript は1ファイルを読んで発言の列にする。
func parseTranscript(file scannedFile) (parsedTranscript, error) {
	handle, err := os.Open(file.Path)
	if err != nil {
		return parsedTranscript{}, fmt.Errorf("error at open transcript %s: %w", file.Path, err)
	}
	defer func() { _ = handle.Close() }()

	parsed := parsedTranscript{ConversationID: file.ConversationID}
	reader := bufio.NewReaderSize(handle, 64*1024)
	records := make([]stepRecord, 0, 256)
	for {
		line, dropped, readErr := readLineLimited(reader)
		if dropped {
			parsed.DroppedLines++
		}
		if len(line) != 0 {
			var record stepRecord
			if json.Unmarshal(line, &record) != nil {
				// 読めない行は数えるだけにして、ファイルごと捨てない
				parsed.DroppedLines++
			} else if record.Type != "" {
				records = append(records, record)
			}
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				break
			}
			return parsed, fmt.Errorf("error at read transcript %s: %w", file.Path, readErr)
		}
	}

	parsed.StepCount = len(records)
	parsed.Messages = buildMessages(file.ConversationID, records)
	return parsed, nil
}

// readLineLimited は1行を読む。maxLineBytes を超えた行は捨てて dropped=true を返し、
// 残りはメモリに起こさず読み捨てる。
func readLineLimited(reader *bufio.Reader) (line []byte, dropped bool, err error) {
	var buffer []byte
	for {
		chunk, readErr := reader.ReadSlice('\n')
		if !dropped {
			if len(buffer)+len(chunk) > maxLineBytes {
				dropped = true
				buffer = nil
			} else {
				buffer = append(buffer, chunk...)
			}
		}
		if errors.Is(readErr, bufio.ErrBufferFull) {
			continue
		}
		return trimEOL(buffer), dropped, readErr
	}
}

// trimEOL は行末の改行(CRLF も LF も)を落とす。
func trimEOL(line []byte) []byte {
	line = bytes.TrimSuffix(line, []byte("\n"))
	return bytes.TrimSuffix(line, []byte("\r"))
}

// buildMessages はステップの列を「人間の発言1件 + それに続く応答1件」の並びにする。
//
// 粒度は ChatGPT / Claude.ai / Claude Code / Codex プラグインと同じ。
// 最初の USER_INPUT より前のステップ(引き継がれた会話履歴・システム発の案内など)は
// 最初の応答の経過の先頭へ入れる —— 黙って捨てない。
// USER_INPUT が1件も無い会話も応答1件にする(0件にはしない)。
func buildMessages(conversationID string, records []stepRecord) []message {
	ordered := slices.Clone(records)
	slices.SortStableFunc(ordered, func(a, b stepRecord) int {
		return cmp.Compare(a.StepIndex, b.StepIndex)
	})

	messages := make([]message, 0, len(ordered)/2+1)
	var leading []stepRecord
	var run []stepRecord
	seenUser := false
	firstReply := true

	flushRun := func() {
		if len(run) == 0 {
			return
		}
		var lead []stepRecord
		if firstReply {
			lead = leading
			leading = nil
		}
		if reply, ok := replyMessage(conversationID, lead, run); ok {
			messages = append(messages, reply)
			firstReply = false
		}
		run = nil
	}

	for _, record := range ordered {
		if record.Type == stepUserInput {
			flushRun()
			messages = append(messages, humanMessage(conversationID, record))
			seenUser = true
			continue
		}
		if !seenUser {
			leading = append(leading, record)
			continue
		}
		run = append(run, record)
	}
	flushRun()

	// 応答が1件も無かった会話(USER_INPUT だけ、あるいは USER_INPUT が無い)。
	// 残ったステップを応答1件にして取りこぼさない。
	if len(leading) != 0 {
		if reply, ok := replyMessage(conversationID, nil, leading); ok {
			messages = append(messages, reply)
		}
	}
	return messages
}

// humanMessage は USER_INPUT 1件を人間の Kyou にする。
func humanMessage(conversationID string, record stepRecord) message {
	text, preamble := splitUserRequest(record.Content)
	return message{
		ID:             kyouIDOf(conversationID, roleHuman, record.StepIndex),
		Role:           roleHuman,
		ConversationID: conversationID,
		StepIndex:      record.StepIndex,
		Text:           text,
		Preamble:       preamble,
		RelatedTime:    record.CreatedAt,
		UpdateTime:     record.CreatedAt,
	}
}

// splitUserRequest は人間の発言から、利用者が書いた依頼と前置きを取り出す。
//
// Antigravity は利用者の入力をブロックで包んで記録する。実データでは例外が無く、
// 並びも固定だった。
//
//	<USER_REQUEST>…実際の依頼…</USER_REQUEST>
//	<ADDITIONAL_METADATA>The current local time is: …</ADDITIONAL_METADATA>
//	<USER_SETTINGS_CHANGE>…自動で付く定型文…</USER_SETTINGS_CHANGE>   ← 付かない発言もある
//
// 包みごと本文にすると、一覧のどの行も `<USER_REQUEST>` の行から始まり、
// タイトルの代用（titleOrFallback）もその文字列になり、検索も定型文の語で全発言に当たる。
// そこで依頼だけを本文にする（gkill_plugin_codex が IDE の前置きを剥がすのと同じ型）。
//
//   - ADDITIONAL_METADATA は捨てる。中身は現在時刻の1行で、同じ時刻は created_at にある
//   - USER_SETTINGS_CHANGE は前置き（詳細画面の折りたたみ）へ。自動の定型文だが出来事ではある
//   - **知らない形は何も剥がさずそのまま本文にする**。USER_REQUEST が無い・閉じていない・
//     ブロックの外に文字がある・中身が空、のどれか。形が変わった日に発言が空になるのを防ぐ
func splitUserRequest(content string) (text string, preamble []string) {
	trimmed := strings.TrimSpace(content)
	request, ok := blockContent(trimmed, tagUserRequest)
	if !ok || request == "" {
		return trimmed, nil
	}

	// 知っているブロックを取り除いて何か残るなら、知らない形。剥がさない
	rest := trimmed
	for _, tag := range []string{tagUserRequest, tagAdditionalMetadata, tagUserSettingsChange} {
		rest = removeBlocks(rest, tag)
	}
	if strings.TrimSpace(rest) != "" {
		return trimmed, nil
	}

	if settings, ok := blockContent(trimmed, tagUserSettingsChange); ok && settings != "" {
		preamble = append(preamble, settings)
	}
	return request, preamble
}

// blockContent は <tag>…</tag> の中身を返す。開きと閉じが揃っていなければ偽。
func blockContent(s, tag string) (string, bool) {
	open, closing := "<"+tag+">", "</"+tag+">"
	start := strings.Index(s, open)
	if start < 0 {
		return "", false
	}
	start += len(open)
	end := strings.Index(s[start:], closing)
	if end < 0 {
		return "", false
	}
	return strings.TrimSpace(s[start : start+end]), true
}

// removeBlocks は <tag>…</tag> を全部取り除く。
// 閉じが無いものは取り除かない（残った開きタグが「知らない形」の判定に効く）。
func removeBlocks(s, tag string) string {
	open, closing := "<"+tag+">", "</"+tag+">"
	for {
		start := strings.Index(s, open)
		if start < 0 {
			return s
		}
		end := strings.Index(s[start+len(open):], closing)
		if end < 0 {
			return s
		}
		s = s[:start] + s[start+len(open)+end+len(closing):]
	}
}

// replyMessage は人間の発言のあとに続いたステップを応答1件の Kyou にする。
//
// 本文は描画のときに「最後のテキスト」を取り出すので、ここでは時系列のまま Items に並べる。
// 見せるものが1つも無い応答(ツールの実行結果だけで切れた場合)は作らない。
func replyMessage(conversationID string, leading, run []stepRecord) (message, bool) {
	appender := &itemAppender{}
	for _, record := range leading {
		appender.appendRecord(record)
	}
	for _, record := range run {
		appender.appendRecord(record)
	}
	if len(appender.items) == 0 {
		return message{}, false
	}

	anchor := run
	if len(anchor) == 0 {
		anchor = leading
	}
	if len(anchor) == 0 {
		return message{}, false
	}

	model := ""
	// 時刻はゼロ値を飛ばして最初と最後を採る。
	// ゼロのまま返すと 1970 年の記録になり、期間で絞った検索から丸ごと漏れる。
	var related, updated time.Time
	for _, record := range anchor {
		if record.Type == stepPlannerResponse && record.Model != "" {
			model = record.Model
		}
		if record.CreatedAt.IsZero() {
			continue
		}
		if related.IsZero() {
			related = record.CreatedAt
		}
		updated = record.CreatedAt
	}

	return message{
		ID:             kyouIDOf(conversationID, roleAssistant, anchor[0].StepIndex),
		Role:           roleAssistant,
		ConversationID: conversationID,
		StepIndex:      anchor[0].StepIndex,
		Model:          model,
		RelatedTime:    related,
		UpdateTime:     updated,
		Items:          appender.items,
	}, true
}

// itemAppender は応答の経過を組み立てる。連続する同種の要素はまとめる。
type itemAppender struct {
	items []turnItem
	// seenNotice は同じ本文の注記を1件に畳むための控え。
	// Antigravity は同じ定型文の EPHEMERAL_MESSAGE を1会話で何度も書くので、
	// 畳まないと経過が定型文で埋まる。
	seenNotice map[string]struct{}
}

// appendRecord は1ステップを経過へ足す。
//
// ツールの実行結果の種別は content を保持しない(ツールの出力そのもので、
// ファイル閲覧1件で KB 級から数十 KB になる。gkill_plugin_codex と同じ方針)。
// エラーだけは1行の注記にする。
func (a *itemAppender) appendRecord(record stepRecord) {
	switch {
	case record.Type == stepPlannerResponse:
		a.addThinking(record.Thinking)
		a.addText(record.Content)
		a.addTools(toolCallsOf(record.ToolCalls))
	case isToolResultStep(record.Type):
		a.addNotice(noticeLabelOf(record.Type), record.Error)
	default:
		// システム発の注記と、知らない種別。捨てないこと
		// (捨てるとログの形が変わった日に会話が静かに消える)。
		// ERROR_MESSAGE は content に見出し・error に詳細が入るので両方残す。
		a.addNotice(noticeLabelOf(record.Type), joinNotice(record.Content, record.Error))
	}
}

func (a *itemAppender) addText(text string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	if n := len(a.items); n != 0 && a.items[n-1].Kind == itemText {
		a.items[n-1].Text += "\n\n" + text
		return
	}
	a.items = append(a.items, turnItem{Kind: itemText, Text: text})
}

func (a *itemAppender) addThinking(text string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	if n := len(a.items); n != 0 && a.items[n-1].Kind == itemThinking {
		a.items[n-1].Thinking = append(a.items[n-1].Thinking, text)
		return
	}
	a.items = append(a.items, turnItem{Kind: itemThinking, Thinking: []string{text}})
}

func (a *itemAppender) addTools(calls []toolCall) {
	if len(calls) == 0 {
		return
	}
	if n := len(a.items); n != 0 && a.items[n-1].Kind == itemTools {
		a.items[n-1].Tools = append(a.items[n-1].Tools, calls...)
		return
	}
	a.items = append(a.items, turnItem{Kind: itemTools, Tools: calls})
}

func (a *itemAppender) addNotice(label, text string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	line := truncateRunes(text, maxNoticeRunes)
	if label != "" {
		line = label + ": " + line
	}
	if a.seenNotice == nil {
		a.seenNotice = map[string]struct{}{}
	}
	if _, exist := a.seenNotice[line]; exist {
		return
	}
	a.seenNotice[line] = struct{}{}
	a.items = append(a.items, turnItem{Kind: itemNotice, Text: line})
}

// joinNotice は注記の本文と詳細をつなぐ。
// 片方が空なら残りだけ、詳細が本文に含まれているなら本文だけを返す。
func joinNotice(content, detail string) string {
	content = strings.TrimSpace(content)
	detail = strings.TrimSpace(detail)
	switch {
	case detail == "":
		return content
	case content == "":
		return detail
	case strings.Contains(content, detail):
		return content
	}
	return content + " / " + detail
}

func isToolResultStep(stepType string) bool {
	_, exist := toolResultStepTypes[stepType]
	return exist
}

// noticeLabelOf はレコード種別の見出しを返す。
// 知らない種別はそのままの名前を出す(隠すより、見えているほうが形の変化に気づける)。
func noticeLabelOf(stepType string) string {
	switch stepType {
	case "SYSTEM_MESSAGE":
		return "システム"
	case "CHECKPOINT":
		return "チェックポイント"
	case "CONVERSATION_HISTORY":
		return "会話履歴の引き継ぎ"
	case "EPHEMERAL_MESSAGE":
		return "一時メッセージ"
	case "ERROR_MESSAGE":
		return "エラー"
	}
	if isToolResultStep(stepType) {
		return "ツールのエラー"
	}
	return stepType
}

// toolCallsOf はツール呼び出しを要約つきで写す。入力そのものは保持しない。
func toolCallsOf(records []toolCallRecord) []toolCall {
	if len(records) == 0 {
		return nil
	}
	calls := make([]toolCall, 0, len(records))
	for _, record := range records {
		calls = append(calls, toolCall{
			Name:    strings.TrimSpace(record.Name),
			Summary: summarizeToolArgs(record.Args),
		})
	}
	return calls
}

// toolArgSummaryKeys は要約に使う args のキー。先に書いたものを優先する。
//
// Antigravity は多くのツール呼び出しに自分で作った読める要約 toolSummary を入れてくるので、
// それがあれば最優先で使う。無いものだけ下の欄から拾う。
// ファイルの中身が入る欄(Content / CodeContent / Replacement)は長いので意図的に入れない。
var toolArgSummaryKeys = []string{
	"toolSummary",
	"CommandLine",
	"Path",
	"AbsolutePath",
	"TargetFile",
	"Target",
	"DirectoryPath",
	"SearchPath",
	"Query",
	"query",
	"ImageName",
	"Recipient",
	"TypeName",
	"Action",
	"kind",
}

// summarizeToolArgs は args から1行の要約を作る。
func summarizeToolArgs(args json.RawMessage) string {
	if len(args) == 0 {
		return ""
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(args, &fields) != nil {
		return truncateRunes(compactJSON(args), maxToolSummaryRunes)
	}

	for _, key := range toolArgSummaryKeys {
		if value := stringFieldOf(fields, key); value != "" {
			return truncateRunes(value, maxToolSummaryRunes)
		}
	}
	// MCP の呼び出しは サーバ名/ツール名 が一番読みやすい
	server := stringFieldOf(fields, "ServerName")
	tool := stringFieldOf(fields, "ToolName")
	switch {
	case server != "" && tool != "":
		return truncateRunes(server+"/"+tool, maxToolSummaryRunes)
	case tool != "":
		return truncateRunes(tool, maxToolSummaryRunes)
	}
	return truncateRunes(compactJSON(args), maxToolSummaryRunes)
}

// stringFieldOf は文字列の欄を取り出す。数値や真偽値は要約にしないので空を返す。
func stringFieldOf(fields map[string]json.RawMessage, key string) string {
	raw, exist := fields[key]
	if !exist {
		return ""
	}
	value := ""
	if json.Unmarshal(raw, &value) != nil {
		return ""
	}
	return strings.TrimSpace(value)
}

// compactJSON は JSON の空白を詰めて1行にする。
func compactJSON(raw json.RawMessage) string {
	return strings.Join(strings.Fields(string(raw)), " ")
}

// truncateRunes は rune 数で切る。切ったことが分かるように印を付ける。
func truncateRunes(text string, limit int) string {
	if limit <= 0 || utf8.RuneCountInString(text) <= limit {
		return text
	}
	runes := []rune(text)
	return string(runes[:limit]) + "…"
}

// kyouIDOf は Kyou ID を作る。
//
// transcript_full.jsonl のレコードには id が無いので合成する。
// 材料は (会話ID, 発言者, その単位の先頭レコードの step_index) の3つ。
// step_index は Antigravity が振る追記のみの通番なので、会話が伸びても
// 既存の Kyou の ID が動かない(動くと利用者が付けたタグやテキストが迷子になる)。
// 発言者を材料に混ぜているので、人間の発言と応答の ID が衝突しない。
func kyouIDOf(conversationID, role string, stepIndex int) string {
	return uuidV5(antigravityNamespace, conversationID+"/"+role+"/"+strconv.Itoa(stepIndex))
}

// searchTextOf はワード検索の対象テキストを組み立てる。
//
// gkill 本体はプラグインが返した Kyou の本文を持たないので、ここが唯一の判定材料になる。
// 会話タイトルとプロジェクト名は焼き込まない —— conversation_summaries.db が
// 更新されたときに全会話を読み直さずに済ませるため、照合のときに連結する(main.go)。
func searchTextOf(t message) string {
	builder := &strings.Builder{}
	write := func(text string) {
		if text == "" || builder.Len() > maxSearchTextBytes {
			return
		}
		builder.WriteString(text)
		builder.WriteString("\n")
	}

	write(t.Text)
	write(t.Model)
	for _, item := range t.Items {
		switch item.Kind {
		case itemText, itemNotice:
			write(item.Text)
		case itemThinking:
			for _, thinking := range item.Thinking {
				write(thinking)
			}
		case itemTools:
			for _, call := range item.Tools {
				write(call.Name)
				write(call.Summary)
			}
		}
	}

	text := builder.String()
	if len(text) > maxSearchTextBytes {
		text = text[:maxSearchTextBytes]
	}
	return text
}
