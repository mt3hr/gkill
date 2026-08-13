package main

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

// decodePatchFiles は thread_item.extra に入れた変更ファイル一覧を戻す。
func decodePatchFiles(extra string) []patchFile {
	if extra == "" {
		return nil
	}
	var files []patchFile
	if json.Unmarshal([]byte(extra), &files) != nil {
		return nil
	}
	return files
}

// decodeIDEContext は thread_item.extra に入れたIDEの前置きを戻す。
func decodeIDEContext(extra string) *ideContext {
	if extra == "" {
		return nil
	}
	var decoded ideContext
	if json.Unmarshal([]byte(extra), &decoded) != nil {
		return nil
	}
	if decoded.ActiveFile == "" && len(decoded.OpenTabs) == 0 {
		return nil
	}
	return &decoded
}

// 詳細ビューに描くブロックの種類。threadItem.Kind とは別の名前空間。
const (
	blockText     = "text"
	blockThinking = "thinking"
	blockTools    = "tools"
	blockPatch    = "patch"
	blockPlan     = "plan"
	blockNotice   = "notice"
	blockSpawn    = "spawn"
)

// サブエージェントの扱い。
const (
	subagentModeFold    = "fold"     // 親の応答に畳み込む(既定)
	subagentModeOwnKyou = "own_kyou" // 独立したKyouにする
)

// maxThreadDepth は親をたどる上限。循環と、壊れたログでの無限ループを防ぐ。
const maxThreadDepth = 8

// threadGroup は1つのルートスレッドと、そこにぶら下がるサブエージェントの束。
type threadGroup struct {
	RootID string
	// Files はスレッドID -> ファイルの素性。ルート自身も含む。
	Files map[string]scannedFile
	// Items はスレッドID -> Seq昇順の要素列。
	Items map[string][]threadItem
	// Titles はスレッドID -> スレッド名(session_index.jsonl 由来)。
	Titles map[string]string
	// Children はルート以外のスレッドID。
	Children []string
}

// rootOf は親をたどってルートスレッドを返す。
//
// 親のファイルが手元に無い子は自分自身がルートになる。
// そうしないと「親を消したせいでサブエージェントの記録が丸ごと消える」ことになる。
func rootOf(threadID string, parents map[string]string, known map[string]struct{}) string {
	current := threadID
	seen := map[string]struct{}{current: {}}
	for range maxThreadDepth {
		parent, ok := parents[current]
		if !ok || parent == "" || parent == current {
			return current
		}
		if _, exist := known[parent]; !exist {
			return current
		}
		if _, loop := seen[parent]; loop {
			return current
		}
		seen[parent] = struct{}{}
		current = parent
	}
	return current
}

// foldGroup はスレッド木を Kyou の列にする。
//
// 1Kyou = 人間の発言1つ、または「次の人間の発言までの応答一式」。
func foldGroup(group threadGroup, mode string) []message {
	root, ok := group.Files[group.RootID]
	if !ok {
		return nil
	}

	subAgents := map[string]*subAgent{}
	if mode != subagentModeOwnKyou {
		for _, childID := range group.Children {
			if built := buildSubAgent(childID, group, 0); built != nil {
				subAgents[childID] = built
			}
		}
	}

	messages := foldThread(group.RootID, root, group, subAgents)

	if mode == subagentModeOwnKyou {
		for _, childID := range group.Children {
			child, exist := group.Files[childID]
			if !exist {
				continue
			}
			messages = append(messages, foldThread(childID, child, group, nil)...)
		}
	}
	return messages
}

// foldThread は1スレッドの要素列を Kyou の列にする。
func foldThread(threadID string, file scannedFile, group threadGroup, subAgents map[string]*subAgent) []message {
	items := group.Items[threadID]
	if len(items) == 0 {
		return nil
	}

	header := message{
		ThreadID:     threadID,
		RootThreadID: group.RootID,
		Title:        group.Titles[threadID],
		Project:      lastPathElement(file.Meta.Cwd),
		Branch:       file.Meta.Branch,
		Originator:   file.Meta.Originator,
		Cwd:          file.Meta.Cwd,
	}
	if header.Title == "" && threadID != group.RootID {
		header.Title = group.Titles[group.RootID]
	}

	var messages []message
	var run *runBuilder
	usersSeen := int64(0)
	model := ""

	flush := func() {
		if run == nil {
			return
		}
		if built, ok := run.build(); ok {
			messages = append(messages, built)
		}
		run = nil
	}
	ensureRun := func(item threadItem) *runBuilder {
		if run == nil {
			run = newRunBuilder(header, threadID, usersSeen, model, item.TS)
		}
		run.touch(item.TS)
		return run
	}

	for _, item := range items {
		switch item.Kind {
		case itemTurn:
			// モデルの切り替わり。ブロックにはしないが、以降のKyouのチップに載る
			model = item.Name
			if run != nil && run.message.Model == "" {
				run.message.Model = model
			}
			continue

		case itemUser:
			flush()
			messages = append(messages, buildHumanMessage(header, threadID, usersSeen, model, item))
			usersSeen++
			continue

		case itemTaskDone:
			if run != nil {
				if ms, err := strconv.ParseInt(item.Text, 10, 64); err == nil {
					run.message.DurationMs += ms
				}
				run.touch(item.TS)
			}
			continue

		case itemAssistant:
			ensureRun(item).addText(item.Text)
		case itemThinking:
			ensureRun(item).addThinking(item.Text)
		case itemTool:
			ensureRun(item).addTool(toolCall{Name: item.Name, Summary: item.Text, CallID: item.RefID})
		case itemPatch:
			ensureRun(item).addPatch(decodePatchFiles(item.Extra))
		case itemPlan:
			ensureRun(item).addPlan(item.Text)
		case itemNotice:
			ensureRun(item).addNotice(item.Text)
		case itemSpawn:
			builder := ensureRun(item)
			if subAgents != nil {
				if child, ok := subAgents[strings.ToLower(item.RefID)]; ok {
					builder.attachAgent(item.Extra, child)
				}
			}
		}
	}
	flush()
	return messages
}

// buildHumanMessage は人間の発言1つを Kyou にする。
func buildHumanMessage(header message, threadID string, ordinal int64, model string, item threadItem) message {
	built := header
	built.Role = roleHuman
	built.Ordinal = ordinal
	built.ID = kyouIDOf(threadID, roleHuman, ordinal)
	built.Model = model
	built.Text = item.Text
	built.IDEContext = decodeIDEContext(item.Extra)
	built.RelatedTime = item.TS
	built.UpdateTime = item.TS
	return built
}

// runBuilder は「次の人間の発言までの応答一式」を1Kyouに組み立てる。
type runBuilder struct {
	message message
	blocks  []turnItem
}

func newRunBuilder(header message, threadID string, ordinal int64, model string, start time.Time) *runBuilder {
	built := header
	built.Role = roleAssistant
	built.Ordinal = ordinal
	built.ID = kyouIDOf(threadID, roleAssistant, ordinal)
	built.Model = model
	built.RelatedTime = start
	built.UpdateTime = start
	return &runBuilder{message: built}
}

func (r *runBuilder) touch(ts time.Time) {
	if ts.IsZero() {
		return
	}
	if r.message.RelatedTime.IsZero() {
		r.message.RelatedTime = ts
	}
	if ts.After(r.message.UpdateTime) {
		r.message.UpdateTime = ts
	}
}

// last は末尾のブロックが指定の種類ならそれを返す。連続する思考やツールをまとめるため。
func (r *runBuilder) last(kind string) *turnItem {
	if len(r.blocks) == 0 {
		return nil
	}
	tail := &r.blocks[len(r.blocks)-1]
	if tail.Kind != kind {
		return nil
	}
	return tail
}

func (r *runBuilder) addText(text string) {
	if text == "" {
		return
	}
	r.blocks = append(r.blocks, turnItem{Kind: blockText, Text: text})
}

func (r *runBuilder) addThinking(text string) {
	if text == "" {
		return
	}
	if tail := r.last(blockThinking); tail != nil {
		tail.Thinking = append(tail.Thinking, text)
		return
	}
	r.blocks = append(r.blocks, turnItem{Kind: blockThinking, Thinking: []string{text}})
}

func (r *runBuilder) addTool(call toolCall) {
	if tail := r.last(blockTools); tail != nil {
		tail.Tools = append(tail.Tools, call)
		return
	}
	r.blocks = append(r.blocks, turnItem{Kind: blockTools, Tools: []toolCall{call}})
}

func (r *runBuilder) addPatch(files []patchFile) {
	if len(files) == 0 {
		return
	}
	if tail := r.last(blockPatch); tail != nil {
		tail.Files = append(tail.Files, files...)
		return
	}
	r.blocks = append(r.blocks, turnItem{Kind: blockPatch, Files: files})
}

func (r *runBuilder) addPlan(text string) {
	if text == "" {
		return
	}
	r.blocks = append(r.blocks, turnItem{Kind: blockPlan, Text: text})
}

func (r *runBuilder) addNotice(text string) {
	if text == "" {
		return
	}
	r.blocks = append(r.blocks, turnItem{Kind: blockNotice, Text: text})
}

// attachAgent はサブエージェントを、それを起こしたツール呼び出しにぶら下げる。
//
// 突き合わせは call_id。見つからなければ独立したブロックとして置く ――
// 実データでは sub_agent_activity(started) が61件あるのに対しロールアウトファイルは
// 13件しかなく、親だけ残っている子が普通にあるので、この経路は必ず要る。
func (r *runBuilder) attachAgent(callID string, agent *subAgent) {
	if callID != "" {
		for blockIndex := range r.blocks {
			if r.blocks[blockIndex].Kind != blockTools {
				continue
			}
			for toolIndex := range r.blocks[blockIndex].Tools {
				if r.blocks[blockIndex].Tools[toolIndex].CallID == callID {
					r.blocks[blockIndex].Tools[toolIndex].Agent = agent
					return
				}
			}
		}
	}
	r.blocks = append(r.blocks, turnItem{Kind: blockSpawn, Agent: agent})
}

// build は組み立てた Kyou を返す。中身が1つも無いなら Kyou にしない。
func (r *runBuilder) build() (message, bool) {
	if len(r.blocks) == 0 {
		return message{}, false
	}
	r.message.Items = r.blocks
	return r.message, true
}

// buildSubAgent は子スレッドを、親に畳み込む形へまとめる。
func buildSubAgent(threadID string, group threadGroup, depth int) *subAgent {
	if depth >= maxThreadDepth {
		return nil
	}
	file, ok := group.Files[threadID]
	if !ok {
		return nil
	}
	items := group.Items[threadID]
	if len(items) == 0 {
		return nil
	}

	built := &subAgent{
		ThreadID:  threadID,
		AgentPath: file.Meta.AgentPath,
		Nickname:  file.Meta.AgentNickname,
	}

	runner := &runBuilder{}
	promptTaken := false
	for _, item := range items {
		switch item.Kind {
		case itemUser:
			if !promptTaken {
				built.Prompt = item.Text
				promptTaken = true
				continue
			}
			runner.addText(item.Text)
		case itemAssistant:
			runner.addText(item.Text)
		case itemThinking:
			runner.addThinking(item.Text)
		case itemTool:
			runner.addTool(toolCall{Name: item.Name, Summary: item.Text, CallID: item.RefID})
		case itemPatch:
			runner.addPatch(decodePatchFiles(item.Extra))
		case itemPlan:
			runner.addPlan(item.Text)
		case itemNotice:
			runner.addNotice(item.Text)
		case itemSpawn:
			if grandChild := buildSubAgent(strings.ToLower(item.RefID), group, depth+1); grandChild != nil {
				runner.attachAgent(item.Extra, grandChild)
			}
		}
		if len(runner.blocks) >= maxSubAgentItems {
			break
		}
	}
	built.Items = runner.blocks
	return built
}

// maxSearchTextBytes は1Kyouぶんの検索用テキストの上限。
//
// 実データでは1件だけ5.4MBに達するKyouがあり(サブエージェント9本を畳み込んだ回)、
// 上限が無いと単語検索のたびにその1行を読むことになる。
// 打ち切っても50万文字ぶんは引けるので実用上は困らない。
const maxSearchTextBytes = 512 * 1024

// searchTextOf は単語検索の対象になるテキストを組み立てる。
//
// ツールの「実行結果」は入れない。それが実データのバイトの94.7%で、
// 入れるとキャッシュが実ログと同じ大きさになる。
// スレッド名も入れない —— session_index.jsonl は名前が付くたびに書き換わるので、
// 焼き込むと畳み直しが要る。照合時に別カラムと連結する。
func searchTextOf(m message) string {
	builder := &boundedBuilder{limit: maxSearchTextBytes}

	builder.line(m.Text)
	builder.line(m.Project)
	builder.line(m.Branch)
	builder.line(m.Model)
	builder.line(m.Originator)
	if m.IDEContext != nil {
		builder.line(m.IDEContext.ActiveFile)
		for _, tab := range m.IDEContext.OpenTabs {
			builder.line(tab.Name)
			builder.line(tab.Path)
		}
	}
	writeBlocks(builder, m.Items, 0)
	return builder.String()
}

// boundedBuilder は上限に達したら黙って書き捨てる文字列ビルダ。
type boundedBuilder struct {
	builder strings.Builder
	limit   int
	full    bool
}

func (b *boundedBuilder) line(value string) {
	if value == "" || b.full {
		return
	}
	if b.builder.Len()+len(value) > b.limit {
		b.full = true
		return
	}
	b.builder.WriteString(value)
	b.builder.WriteByte('\n')
}

func (b *boundedBuilder) String() string { return b.builder.String() }

func writeBlocks(builder *boundedBuilder, blocks []turnItem, depth int) {
	if depth > maxThreadDepth {
		return
	}
	for _, block := range blocks {
		if builder.full {
			return
		}
		builder.line(block.Text)
		for _, thinking := range block.Thinking {
			builder.line(thinking)
		}
		for _, tool := range block.Tools {
			builder.line(tool.Name + " " + tool.Summary)
			writeAgent(builder, tool.Agent, depth)
		}
		for _, file := range block.Files {
			builder.line(file.Path)
		}
		writeAgent(builder, block.Agent, depth)
	}
}

func writeAgent(builder *boundedBuilder, agent *subAgent, depth int) {
	if agent == nil {
		return
	}
	builder.line(agent.AgentPath)
	builder.line(agent.Nickname)
	builder.line(agent.Prompt)
	writeBlocks(builder, agent.Items, depth+1)
}

// kyouIDOf は Kyou の ID を作る。
//
// ロールアウトは追記のみなので、スレッド内でのロール別連番は安定している。
// 同じセッションを2箇所(実ログと集約コピー)から拾っても同じIDになるので重複しない。
//
// 追記のみの前提が破れた(過去の行が消された)場合、それ以降のKyouは別IDになる。
// 検出は cache 側の thread_state(前回のサイズ)で行い、自動修復はしない。
// どうしても耐える必要が出たら turn_id を混ぜる形へ移行できるが、
// そのときは一度だけ全KyouのIDが変わる。
func kyouIDOf(threadID, role string, ordinal int64) string {
	return uuidV5(codexNamespace, appName+"|"+threadID+"|"+role+"|"+strconv.FormatInt(ordinal, 10))
}
