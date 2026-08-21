package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mattn/go-zglob"
)

// ファイル種別。
const (
	kindMain     = "main"     // メインのセッショントランスクリプト
	kindSubAgent = "subagent" // サブエージェントのトランスクリプト
	kindMeta     = "meta"     // agent-<ID>.meta.json
	kindOther    = "other"    // history.jsonl など、対象外のファイル
)

// 表示・保存サイズの上限。
const (
	maxToolSummaryRunes = 200
	maxNoticeRunes      = 200
	maxSubAgentItems    = 200
)

// scannedFile はソースフォルダ走査で見つかった1ファイル。
type scannedFile struct {
	Path      string
	MtimeUnix int64
	Size      int64
	Kind      string
	SessionID string
}

// defaultSourceDir は設定が空のときに使う既定のデータソース。
func defaultSourceDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".claude", "projects")
}

// parseSourcePatterns は設定値をパターンのリストにする。
// 設定は文字列(改行区切り)でも配列でも書ける。
//
//	"source_dirs": "C:\\a\nC:\\b"
//	"source_dirs": ["C:\\a", "C:\\b"]
//
// 空なら既定のフォルダにフォールバックする。
func parseSourcePatterns(value any) []string {
	var patterns []string
	add := func(s string) {
		s = strings.TrimSpace(strings.TrimSuffix(s, "\r"))
		if s == "" {
			return
		}
		patterns = append(patterns, expandHome(os.ExpandEnv(s)))
	}

	switch v := value.(type) {
	case nil:
	case string:
		for line := range strings.SplitSeq(v, "\n") {
			add(line)
		}
	case []string:
		for _, s := range v {
			for line := range strings.SplitSeq(s, "\n") {
				add(line)
			}
		}
	case []any:
		for _, e := range v {
			if s, ok := e.(string); ok {
				for line := range strings.SplitSeq(s, "\n") {
					add(line)
				}
			}
		}
	}

	if len(patterns) == 0 {
		if d := defaultSourceDir(); d != "" {
			patterns = append(patterns, d)
		}
	}
	return patterns
}

// expandHome は先頭の ~ をホームディレクトリに展開する。
// Windowsサービスとして動くgkillでは実行アカウントのホームになる点に注意
// (LocalSystemなら systemprofile)。確実に指定したいときは絶対パスを使う。
func expandHome(path string) string {
	if path != "~" && !strings.HasPrefix(path, "~/") && !strings.HasPrefix(path, `~\`) {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return path
	}
	if path == "~" {
		return home
	}
	return filepath.Join(home, path[2:])
}

// hasGlobMeta はパターンにワイルドカードが含まれるかを判定する。
func hasGlobMeta(pattern string) bool {
	return strings.ContainsAny(pattern, "*?[")
}

// expandedSource はパターンを展開した結果。
type expandedSource struct {
	Dirs    []string // 再帰的に走査するディレクトリ
	Files   []string // 直接対象にするファイル
	Missing []string // 実在しなかった、あるいは何にもマッチしなかったパターン
}

// expandSourcePatterns はパターンを実在するパスへ展開する。
// ワイルドカード(*, **, ?, [])を含むパターンはグロブ展開し、
// マッチしたものがディレクトリなら再帰走査の対象、ファイルならそのまま対象にする。
// 例:
//
//	C:\Users\user\.claude\projects        → そのディレクトリを再帰走査
//	C:\Users\user\PC\ClaudeCode_*     → マッチした各ディレクトリを再帰走査
//	C:\logs\**\*.jsonl                     → マッチした各ファイルを対象
func expandSourcePatterns(patterns []string) expandedSource {
	var result expandedSource
	seen := map[string]bool{}

	classify := func(path string) bool {
		abs, err := filepath.Abs(path)
		if err != nil {
			abs = path
		}
		if seen[abs] {
			return true
		}
		info, err := os.Stat(abs)
		if err != nil {
			return false
		}
		seen[abs] = true
		if info.IsDir() {
			result.Dirs = append(result.Dirs, abs)
		} else {
			result.Files = append(result.Files, abs)
		}
		return true
	}

	for _, pattern := range patterns {
		if pattern == "" {
			continue
		}
		if !hasGlobMeta(pattern) {
			if !classify(pattern) {
				result.Missing = append(result.Missing, pattern)
			}
			continue
		}
		matches, err := zglob.Glob(pattern)
		if err != nil || len(matches) == 0 {
			result.Missing = append(result.Missing, pattern)
			continue
		}
		matched := false
		for _, m := range matches {
			if classify(m) {
				matched = true
			}
		}
		if !matched {
			result.Missing = append(result.Missing, pattern)
		}
	}
	return result
}

// scanSources は展開済みのディレクトリとファイルから対象ファイルの一覧を返す。
// ファイル種別はパスではなく中身で判定するので、フラット配置でも入れ子配置でも動く。
// knownは前回のスキャン結果。mtimeとサイズが変わっていないファイルは中身を読み直さない。
func scanSources(src expandedSource, known map[string]scannedFile) ([]scannedFile, error) {
	var files []scannedFile
	seen := map[string]bool{}
	var errs []error

	appendFile := func(path string, info fs.FileInfo) {
		abs, aerr := filepath.Abs(path)
		if aerr != nil {
			abs = path
		}
		if seen[abs] {
			return
		}
		name := strings.ToLower(filepath.Base(abs))
		isJSONL := strings.HasSuffix(name, ".jsonl")
		isMeta := strings.HasSuffix(name, ".meta.json")
		if !isJSONL && !isMeta {
			return
		}

		f := scannedFile{
			Path:      abs,
			MtimeUnix: info.ModTime().Unix(),
			Size:      info.Size(),
		}
		if prev, ok := known[abs]; ok && prev.MtimeUnix == f.MtimeUnix && prev.Size == f.Size {
			f.Kind = prev.Kind
			f.SessionID = prev.SessionID
		} else if isMeta {
			f.Kind = kindMeta
		} else {
			f.Kind, f.SessionID = probeTranscript(abs)
		}
		seen[abs] = true
		files = append(files, f)
	}

	for _, dir := range src.Dirs {
		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				// 読めないディレクトリ/ファイルはスキップして続行するが、
				// 取りこぼしに気づけるよう errs に載せる(scanErr→LastScanError へ流れる)。
				errs = append(errs, fmt.Errorf("error at walk %s: %w", path, walkErr))
				return nil
			}
			if d.IsDir() {
				return nil
			}
			info, ierr := d.Info()
			if ierr != nil {
				errs = append(errs, fmt.Errorf("error at stat %s: %w", path, ierr))
				return nil
			}
			appendFile(path, info)
			return nil
		})
		if err != nil {
			errs = append(errs, fmt.Errorf("error at walk %s: %w", dir, err))
		}
	}

	for _, path := range src.Files {
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		appendFile(path, info)
	}

	return files, errors.Join(errs...)
}

// probeTranscript はJSONLの先頭数行を読んでファイル種別とセッションIDを判定する。
// type を持つレコードがあればトランスクリプト。isSidechain か agentId があればサブエージェント。
func probeTranscript(path string) (kind string, sessionID string) {
	f, err := os.Open(path)
	if err != nil {
		return kindOther, ""
	}
	defer func() { _ = f.Close() }()

	kind = kindOther
	r := bufio.NewReader(f)
	for range 5 {
		line, err := readLine(r)
		if line != "" {
			var p probeRecord
			if json.Unmarshal([]byte(line), &p) == nil {
				if p.SessionID != "" && sessionID == "" {
					sessionID = p.SessionID
				}
				if p.IsSidechain || p.AgentID != "" {
					return kindSubAgent, sessionID
				}
				if p.Type != "" {
					kind = kindMain
				}
			}
		}
		if err != nil {
			break
		}
	}
	if kind != kindMain {
		return kindOther, ""
	}
	return kind, sessionID
}

// readLine は改行区切りの1行を読む。
// トランスクリプトには100万文字を超える行が実在するため、bufio.Scannerは使えない。
func readLine(r *bufio.Reader) (string, error) {
	line, err := r.ReadString('\n')
	return strings.TrimRight(line, "\r\n"), err
}

// readRecords はJSONLを1行ずつlogRecordとして読む。パースできない行は読み飛ばす。
func readRecords(path string) ([]logRecord, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("error at open %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	var records []logRecord
	r := bufio.NewReader(f)
	for {
		line, rerr := readLine(r)
		if line != "" {
			var rec logRecord
			if json.Unmarshal([]byte(line), &rec) == nil {
				records = append(records, rec)
			}
		}
		if rerr != nil {
			if errors.Is(rerr, io.EOF) {
				break
			}
			return records, fmt.Errorf("error at read %s: %w", path, rerr)
		}
	}
	return records, nil
}

// readAgentMeta は agent-<ID>.meta.json を読む。
func readAgentMeta(path string) (agentID string, meta agentMeta, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", meta, fmt.Errorf("error at read %s: %w", path, err)
	}
	if err := json.Unmarshal(data, &meta); err != nil {
		return "", meta, fmt.Errorf("error at parse %s: %w", path, err)
	}
	return agentIDFromFileName(path), meta, nil
}

// agentIDFromFileName は agent-<ID>.jsonl / agent-<ID>.meta.json からIDを取り出す。
func agentIDFromFileName(path string) string {
	name := filepath.Base(path)
	name = strings.TrimSuffix(name, ".meta.json")
	name = strings.TrimSuffix(name, ".jsonl")
	return strings.TrimPrefix(name, "agent-")
}

// isHumanPrompt は人間が入力したプロンプトのレコードかどうかを判定する。
// task-notification などシステム発の入力はここではfalseになり、直前のターンに吸収される。
func isHumanPrompt(rec logRecord) bool {
	if rec.Type != "user" {
		return false
	}
	switch rec.PromptSource {
	case "typed", "queued", "text":
		return true
	}
	return false
}

// extractBlocks は message.content をブロック配列として取り出す。
// contentが文字列の場合は text ブロック1個として扱う。
func extractBlocks(raw json.RawMessage) []contentBlock {
	if len(raw) == 0 {
		return nil
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		if s == "" {
			return nil
		}
		return []contentBlock{{Type: "text", Text: s}}
	}
	var blocks []contentBlock
	if json.Unmarshal(raw, &blocks) == nil {
		return blocks
	}
	return nil
}

// extractText はブロック配列からテキストだけを連結する。
func extractText(raw json.RawMessage) string {
	var sb strings.Builder
	for _, b := range extractBlocks(raw) {
		if b.Type == "text" && b.Text != "" {
			if sb.Len() > 0 {
				sb.WriteString("\n")
			}
			sb.WriteString(b.Text)
		}
	}
	return sb.String()
}

// summarizeToolInput はツール入力を1行に要約する。実行結果は保持しない。
func summarizeToolInput(name string, input json.RawMessage) string {
	if len(input) == 0 {
		return ""
	}
	var m map[string]any
	if json.Unmarshal(input, &m) != nil {
		return truncateRunes(oneLine(string(input)), maxToolSummaryRunes)
	}

	// ツールごとに代表的なフィールドを選ぶ
	var keys []string
	switch name {
	case "Bash", "PowerShell":
		keys = []string{"command", "description"}
	case "Read", "Write", "NotebookEdit":
		keys = []string{"file_path", "notebook_path"}
	case "Edit":
		keys = []string{"file_path"}
	case "Grep", "Glob":
		keys = []string{"pattern", "query"}
	case "WebFetch":
		keys = []string{"url", "prompt"}
	case "WebSearch":
		keys = []string{"query"}
	case "Agent", "Task", "TaskCreate", "TaskUpdate":
		keys = []string{"description", "subagent_type", "prompt"}
	case "Skill":
		keys = []string{"skill", "args"}
	default:
		keys = []string{"description", "command", "file_path", "path", "query", "pattern", "url", "prompt"}
	}
	for _, k := range keys {
		if v, ok := m[k]; ok {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				return truncateRunes(oneLine(s), maxToolSummaryRunes)
			}
		}
	}
	return truncateRunes(oneLine(string(input)), maxToolSummaryRunes)
}

// oneLine は改行と連続空白を潰して1行にする。
func oneLine(s string) string {
	s = strings.ReplaceAll(s, "\r\n", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\t", " ")
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}
	return strings.TrimSpace(s)
}

// truncateRunes はルーン単位で長さを切り詰める。
func truncateRunes(s string, limit int) string {
	r := []rune(s)
	if len(r) <= limit {
		return s
	}
	return string(r[:limit]) + "…"
}

// itemAppender はturnItemを組み立てるヘルパ。連続する同種の要素をまとめる。
type itemAppender struct {
	items []turnItem
}

func (a *itemAppender) addText(text string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	if n := len(a.items); n > 0 && a.items[n-1].Kind == "text" {
		a.items[n-1].Text += "\n\n" + text
		return
	}
	a.items = append(a.items, turnItem{Kind: "text", Text: text})
}

func (a *itemAppender) addThinking(text string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	if n := len(a.items); n > 0 && a.items[n-1].Kind == "thinking" {
		a.items[n-1].Thinking = append(a.items[n-1].Thinking, text)
		return
	}
	a.items = append(a.items, turnItem{Kind: "thinking", Thinking: []string{text}})
}

func (a *itemAppender) addTool(tc toolCall) {
	if n := len(a.items); n > 0 && a.items[n-1].Kind == "tools" {
		a.items[n-1].Tools = append(a.items[n-1].Tools, tc)
		return
	}
	a.items = append(a.items, turnItem{Kind: "tools", Tools: []toolCall{tc}})
}

func (a *itemAppender) addNotice(text string) {
	text = truncateRunes(oneLine(text), maxNoticeRunes)
	if text == "" {
		return
	}
	a.items = append(a.items, turnItem{Kind: "notice", Text: text})
}

// buildMessages はメイントランスクリプトのレコード列をKyouの単位に分ける。
// 人間の発言が1件、それに続く一連の応答がまとめて1件になる。
// 応答はツール実行を挟んで何度も分かれるが、次の人間の発言までを1つとして扱う。
// agentsByToolUseID / agentsByID はサブエージェントの紐付けに使う。
func buildMessages(records []logRecord, agentsByToolUseID, agentsByID map[string]*subAgent) []message {
	sessionTitle, project, branch := scanSessionMeta(records)
	toolUseIDToAgentID := scanAgentIDsFromResults(records)

	var messages []message
	var cur *message // 組み立て中の応答
	var app *itemAppender

	flush := func() {
		if cur == nil {
			return
		}
		if len(app.items) > 0 {
			cur.Items = app.items
			messages = append(messages, *cur)
		}
		cur = nil
		app = nil
	}
	// 応答のまとまりを開く。IDと開始時刻は最初のレコードのものを使う。
	// システム通知だけのKyouができないよう、assistantレコードでのみ開く。
	open := func(rec logRecord) {
		if cur != nil || rec.UUID == "" {
			return
		}
		cur = &message{
			ID:           rec.UUID,
			Role:         roleAssistant,
			SessionID:    rec.SessionID,
			SessionTitle: sessionTitle,
			Project:      project,
			Branch:       branch,
			RelatedTime:  rec.Timestamp,
			UpdateTime:   rec.Timestamp,
		}
		app = &itemAppender{}
	}
	touch := func(rec logRecord) {
		if cur != nil && !rec.Timestamp.IsZero() && rec.Timestamp.After(cur.UpdateTime) {
			cur.UpdateTime = rec.Timestamp
		}
	}

	for _, rec := range records {
		if isHumanPrompt(rec) {
			flush()
			if rec.UUID == "" {
				// IDが無いレコードはKyouにできないので捨てる
				continue
			}
			messages = append(messages, message{
				ID:           rec.UUID,
				Role:         roleHuman,
				SessionID:    rec.SessionID,
				SessionTitle: sessionTitle,
				Project:      project,
				Branch:       branch,
				Text:         strings.TrimSpace(extractText(rec.Message.contentOrEmpty())),
				RelatedTime:  rec.Timestamp,
				UpdateTime:   rec.Timestamp,
			})
			continue
		}

		switch rec.Type {
		case "user":
			// task-notification などシステム発の入力。応答の一部として注記に入れる
			if rec.PromptSource == "system" && app != nil {
				app.addNotice(extractText(rec.Message.contentOrEmpty()))
				touch(rec)
			}
			// tool_result は表示しない
		case "assistant":
			blocks := extractBlocks(rec.Message.contentOrEmpty())
			if len(blocks) == 0 {
				continue
			}
			open(rec)
			if app == nil {
				continue
			}
			for _, b := range blocks {
				switch b.Type {
				case "text":
					app.addText(b.Text)
				case "thinking":
					app.addThinking(b.Thinking)
				case "tool_use":
					tc := toolCall{
						Name:    b.Name,
						Summary: summarizeToolInput(b.Name, b.Input),
					}
					if agent := lookupSubAgent(b.ID, agentsByToolUseID, agentsByID, toolUseIDToAgentID); agent != nil {
						tc.Agent = agent
					}
					app.addTool(tc)
				}
			}
			touch(rec)
		}
	}
	flush()
	return messages
}

// lookupSubAgent は tool_use.id からサブエージェントを引く。
// agent-<ID>.meta.json の toolUseId を第一手段とし、
// 取れない場合は親の tool_result の toolUseResult.agentId にフォールバックする。
func lookupSubAgent(toolUseID string, byToolUseID, byID map[string]*subAgent, toolUseIDToAgentID map[string]string) *subAgent {
	if toolUseID == "" {
		return nil
	}
	if a, ok := byToolUseID[toolUseID]; ok {
		return a
	}
	if agentID, ok := toolUseIDToAgentID[toolUseID]; ok {
		if a, ok := byID[agentID]; ok {
			return a
		}
	}
	return nil
}

// scanSessionMeta はセッションタイトル・プロジェクト名・ブランチ名を取り出す。
// タイトルは ai-title レコードの最後のものを採用する。
func scanSessionMeta(records []logRecord) (title, project, branch string) {
	for _, rec := range records {
		if rec.Type == "ai-title" && rec.AITitle != "" {
			title = rec.AITitle
		}
		if project == "" && rec.Cwd != "" {
			project = lastPathElement(rec.Cwd)
		}
		if branch == "" && rec.GitBranch != "" {
			branch = rec.GitBranch
		}
	}
	return title, project, branch
}

// lastPathElement はパス文字列の末尾の要素を返す。
//
// filepath.Base を使ってはいけない。ログを書いた環境と読む環境でOSが違うことがあり、
// 区切り文字の解釈がずれるため。実際 Linux 上では
// filepath.Base(`C:\work\myproj`) が区切りを見つけられず文字列全体を返すので、
// Windowsで記録したログをLinuxで読むとプロジェクト名がフルパスになっていた。
// どちらの区切り文字も見て切り出す。
func lastPathElement(p string) string {
	trimmed := strings.TrimRight(p, `/\`)
	if trimmed == "" {
		return p
	}
	if i := strings.LastIndexAny(trimmed, `/\`); i >= 0 {
		return trimmed[i+1:]
	}
	return trimmed
}

// scanAgentIDsFromResults は tool_result の toolUseResult.agentId を集める。
func scanAgentIDsFromResults(records []logRecord) map[string]string {
	result := map[string]string{}
	for _, rec := range records {
		if rec.Type != "user" || len(rec.ToolUseResult) == 0 {
			continue
		}
		var tur toolUseResultAgent
		if json.Unmarshal(rec.ToolUseResult, &tur) != nil || tur.AgentID == "" {
			continue
		}
		for _, b := range extractBlocks(rec.Message.contentOrEmpty()) {
			if b.Type == "tool_result" && b.ToolUseID != "" {
				result[b.ToolUseID] = tur.AgentID
			}
		}
	}
	return result
}

// buildSubAgent はサブエージェントのトランスクリプトを1つの会話にまとめる。
func buildSubAgent(agentID string, meta agentMeta, records []logRecord) *subAgent {
	sa := &subAgent{
		AgentID:     agentID,
		AgentType:   meta.AgentType,
		Description: meta.Description,
	}
	app := &itemAppender{}
	for _, rec := range records {
		switch rec.Type {
		case "user":
			if sa.Prompt == "" && rec.PromptSource != "" && rec.PromptSource != "tool_result" {
				sa.Prompt = strings.TrimSpace(extractText(rec.Message.contentOrEmpty()))
			}
		case "assistant":
			for _, b := range extractBlocks(rec.Message.contentOrEmpty()) {
				switch b.Type {
				case "text":
					app.addText(b.Text)
				case "thinking":
					app.addThinking(b.Thinking)
				case "tool_use":
					app.addTool(toolCall{Name: b.Name, Summary: summarizeToolInput(b.Name, b.Input)})
				}
			}
		}
		if len(app.items) > maxSubAgentItems {
			break
		}
	}
	if sa.Prompt == "" {
		sa.Prompt = meta.Description
	}
	sa.Items = app.items
	return sa
}

// contentOrEmpty はmessageがnilでも安全にcontentを取り出す。
func (m *logMessage) contentOrEmpty() json.RawMessage {
	if m == nil {
		return nil
	}
	return m.Content
}

// searchTextOf は検索対象のテキストを組み立てる。
// 発言本文・ツール名・サブエージェントの説明のほか、
// セッションタイトル・プロジェクト名・ブランチ名も含める。
// これらはKyouのタグにはしない(gkillのタグ一覧にはプラグインのタグが載らないため、
// rykvの既定のタグ絞り込み「no tags」から漏れて何も表示されなくなる)。
// 代わりにワード検索で引けるようにし、表示は詳細HTMLのチップで行う。
func searchTextOf(t message) string {
	var sb strings.Builder
	sb.WriteString(t.Text)
	sb.WriteString("\n")
	sb.WriteString(t.SessionTitle)
	sb.WriteString("\n")
	sb.WriteString(t.Project)
	sb.WriteString(" ")
	sb.WriteString(t.Branch)
	sb.WriteString("\n")
	for _, item := range t.Items {
		switch item.Kind {
		case "text", "notice":
			sb.WriteString(item.Text)
			sb.WriteString("\n")
		case "tools":
			for _, tc := range item.Tools {
				sb.WriteString(tc.Name)
				sb.WriteString(" ")
				sb.WriteString(tc.Summary)
				sb.WriteString("\n")
				if tc.Agent != nil {
					sb.WriteString(tc.Agent.AgentType)
					sb.WriteString(" ")
					sb.WriteString(tc.Agent.Description)
					sb.WriteString("\n")
				}
			}
		}
	}
	return sb.String()
}

// unixToTime はUnixタイムスタンプをtime.Timeに変換する。
func unixToTime(unix int64) time.Time {
	if unix == 0 {
		return time.Time{}
	}
	return time.Unix(unix, 0)
}
