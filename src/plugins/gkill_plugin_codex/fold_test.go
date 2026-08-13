package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// buildGroupFromFiles はファイルを読んでスレッド木を組み立てる。
// cache.go の畳み直しと同じ手順(ファイル名のuuidをスレッドID、親を辿ってルートを決める)。
func buildGroupFromFiles(t *testing.T, titles map[string]string, paths ...string) threadGroup {
	t.Helper()

	files := map[string]scannedFile{}
	items := map[string][]threadItem{}
	parents := map[string]string{}
	known := map[string]struct{}{}

	for _, path := range paths {
		parsed := mustParse(t, path)
		threadID := threadIDFromFileName(path)
		parsed.Meta.ThreadID = threadID
		files[threadID] = scannedFile{Path: path, Kind: kindRollout, Meta: parsed.Meta}
		items[threadID] = parsed.Items
		parents[threadID] = parsed.Meta.ParentThreadID
		known[threadID] = struct{}{}
	}

	rootID := rootOf(threadIDFromFileName(paths[0]), parents, known)
	var children []string
	for threadID := range known {
		if threadID != rootID && rootOf(threadID, parents, known) == rootID {
			children = append(children, threadID)
		}
	}
	if titles == nil {
		titles = map[string]string{}
	}
	return threadGroup{RootID: rootID, Files: files, Items: items, Titles: titles, Children: children}
}

func TestRootOf(t *testing.T) {
	known := map[string]struct{}{"a": {}, "b": {}, "c": {}}

	if got := rootOf("c", map[string]string{"c": "b", "b": "a"}, known); got != "a" {
		t.Errorf("親をたどれていない: %q", got)
	}
	// 親のファイルが手元に無い子は自分自身がルート。
	// そうしないと「親を消したせいでサブエージェントの記録が丸ごと消える」ことになる。
	if got := rootOf("b", map[string]string{"b": "missing"}, known); got != "b" {
		t.Errorf("親が居ないときは自分がルートであるべき: %q", got)
	}
	// 循環しても止まる
	if got := rootOf("a", map[string]string{"a": "b", "b": "a"}, known); got == "" {
		t.Error("循環で空を返してはいけない")
	}
	if got := rootOf("a", map[string]string{}, known); got != "a" {
		t.Errorf("親が無いときは自分がルート: %q", got)
	}
}

func TestFoldSplitsHumanAndResponseRuns(t *testing.T) {
	group := buildGroupFromFiles(t, map[string]string{parentThreadID: "バグ修正スレッド"},
		parentFixture(), subAgentFixture())
	messages := foldGroup(group, subagentModeFold)

	if len(messages) != 4 {
		t.Fatalf("Kyou が %d 件。人間2件 + 応答2件であるべき", len(messages))
	}

	wantRoles := []string{roleHuman, roleAssistant, roleHuman, roleAssistant}
	wantOrdinals := []int64{0, 1, 1, 2}
	for i, m := range messages {
		if m.Role != wantRoles[i] {
			t.Errorf("[%d] Role = %q, want %q", i, m.Role, wantRoles[i])
		}
		if m.Ordinal != wantOrdinals[i] {
			t.Errorf("[%d] Ordinal = %d, want %d", i, m.Ordinal, wantOrdinals[i])
		}
		if m.ThreadID != parentThreadID {
			t.Errorf("[%d] ThreadID = %q (サブエージェントはKyouにしない)", i, m.ThreadID)
		}
		if m.Title != "バグ修正スレッド" {
			t.Errorf("[%d] Title = %q", i, m.Title)
		}
		if m.Project != "myproj" || m.Branch != "main" {
			t.Errorf("[%d] Project=%q Branch=%q", i, m.Project, m.Branch)
		}
		if m.Model != "gpt-5.3-codex" {
			t.Errorf("[%d] Model = %q", i, m.Model)
		}
		if m.Originator != "codex_vscode" {
			t.Errorf("[%d] Originator = %q", i, m.Originator)
		}
	}

	// 人間の発言は本文を持ち、応答は本文を持たない(中身は Items)
	if !strings.Contains(messages[0].Text, "バグを直してください") {
		t.Errorf("人間の本文 = %q", messages[0].Text)
	}
	if messages[0].IDEContext == nil || messages[0].IDEContext.ActiveFile != "main.go" {
		t.Errorf("IDEのコンテキストが取れていない: %+v", messages[0].IDEContext)
	}
	if messages[1].Text != "" {
		t.Errorf("応答の Text = %q, want empty", messages[1].Text)
	}
	if len(messages[1].Items) == 0 {
		t.Error("応答の Items が空")
	}
}

func TestFoldAssistantRunBlocksAndTimes(t *testing.T) {
	group := buildGroupFromFiles(t, nil, parentFixture(), subAgentFixture())
	messages := foldGroup(group, subagentModeFold)
	run := messages[1]

	var kinds []string
	for _, block := range run.Items {
		kinds = append(kinds, block.Kind)
	}
	want := []string{blockThinking, blockText, blockTools, blockPatch, blockTools, blockPlan}
	if strings.Join(kinds, ",") != strings.Join(want, ",") {
		t.Errorf("ブロックの並び = %v, want %v", kinds, want)
	}

	// 連続したツールは1つにまとまる。あいだに patch が入ると分かれる
	if len(run.Items[2].Tools) != 2 {
		t.Errorf("1つ目のツール群 = %d件, want 2", len(run.Items[2].Tools))
	}
	if len(run.Items[4].Tools) != 1 {
		t.Errorf("2つ目のツール群 = %d件, want 1", len(run.Items[4].Tools))
	}

	// RelatedTime は最初のレコード、UpdateTime は最後のレコード
	wantRelated := time.Date(2026, 1, 2, 1, 0, 4, 0, time.UTC)
	wantUpdate := time.Date(2026, 1, 2, 1, 1, 0, 0, time.UTC)
	if !run.RelatedTime.Equal(wantRelated) {
		t.Errorf("RelatedTime = %v, want %v", run.RelatedTime, wantRelated)
	}
	if !run.UpdateTime.Equal(wantUpdate) {
		t.Errorf("UpdateTime = %v, want %v", run.UpdateTime, wantUpdate)
	}
	if run.DurationMs != 58000 {
		t.Errorf("DurationMs = %d, want 58000", run.DurationMs)
	}

	// 人間の発言は RelatedTime = UpdateTime
	if !messages[0].RelatedTime.Equal(messages[0].UpdateTime) {
		t.Error("人間の発言の時刻がずれている")
	}
}

func TestFoldFoldsSubAgentIntoParent(t *testing.T) {
	group := buildGroupFromFiles(t, nil, parentFixture(), subAgentFixture())
	messages := foldGroup(group, subagentModeFold)

	var found *subAgent
	for _, block := range messages[1].Items {
		for _, tool := range block.Tools {
			if tool.Agent != nil {
				if tool.Name != "spawn_agent" {
					t.Errorf("サブエージェントが %q にぶら下がっている", tool.Name)
				}
				if tool.CallID != "call_spawn_1" {
					t.Errorf("突き合わせた call_id = %q", tool.CallID)
				}
				found = tool.Agent
			}
		}
		if block.Kind == blockSpawn {
			t.Error("call_id で突き合わせられるのに独立ブロックになっている")
		}
	}
	if found == nil {
		t.Fatal("サブエージェントが親に畳み込まれていない")
	}
	if found.Nickname != "Singer" || found.AgentPath != "/root/explore" {
		t.Errorf("エージェント情報 = %+v", found)
	}
	if found.Prompt != "調べてください。" {
		t.Errorf("Prompt = %q (子の最初の発言であるべき)", found.Prompt)
	}
	if len(found.Items) == 0 {
		t.Error("サブエージェントの中身が空")
	}

	// 子は Kyou を作らない
	for _, m := range messages {
		if m.ThreadID == subThreadID {
			t.Error("サブエージェントが独立したKyouになっている")
		}
	}
}

func TestFoldSubAgentWithoutMatchingCallID(t *testing.T) {
	// 実データでは sub_agent_activity(started) が61件あるのにロールアウトファイルは
	// 13件しかない。突き合わせに失敗しても記録を落とさないこと。
	group := buildGroupFromFiles(t, nil, parentFixture(), subAgentFixture())
	for threadID, items := range group.Items {
		if threadID != parentThreadID {
			continue
		}
		for i := range items {
			if items[i].Kind == itemSpawn {
				items[i].Extra = "call_不一致"
			}
		}
	}
	messages := foldGroup(group, subagentModeFold)

	standalone := false
	for _, m := range messages {
		for _, block := range m.Items {
			if block.Kind == blockSpawn && block.Agent != nil && block.Agent.Nickname == "Singer" {
				standalone = true
			}
		}
	}
	if !standalone {
		t.Error("突き合わせに失敗したサブエージェントは独立ブロックとして残すべき")
	}
}

func TestFoldOrphanSubAgentBecomesOwnRoot(t *testing.T) {
	// 親のファイルが無い子は自分がルートになり、独立してKyouになる
	group := buildGroupFromFiles(t, nil, subAgentFixture())
	if group.RootID != subThreadID {
		t.Fatalf("RootID = %q, want %q", group.RootID, subThreadID)
	}
	messages := foldGroup(group, subagentModeFold)
	if len(messages) == 0 {
		t.Fatal("親が居ない子の記録が丸ごと消えている")
	}
	if messages[0].Role != roleHuman || messages[0].Text != "調べてください。" {
		t.Errorf("messages[0] = %+v", messages[0])
	}
}

func TestFoldSubagentModeOwnKyou(t *testing.T) {
	group := buildGroupFromFiles(t, nil, parentFixture(), subAgentFixture())
	messages := foldGroup(group, subagentModeOwnKyou)

	sub := 0
	for _, m := range messages {
		if m.ThreadID == subThreadID {
			sub++
		}
		// own_kyou では親に畳み込まない(同じ内容が2箇所に出ると検索が二重になる)
		for _, block := range m.Items {
			if block.Agent != nil {
				t.Error("own_kyou なのに親へ畳み込まれている")
			}
			for _, tool := range block.Tools {
				if tool.Agent != nil {
					t.Error("own_kyou なのにツールへ畳み込まれている")
				}
			}
		}
	}
	if sub == 0 {
		t.Error("own_kyou でサブエージェントがKyouになっていない")
	}
}

func TestKyouIDIsDeterministic(t *testing.T) {
	// 名前空間を固定する。値を変えると全KyouのIDが変わり、
	// ユーザが付けたタグやテキストが全部迷子になる。
	cases := map[string]string{
		kyouIDOf(parentThreadID, roleHuman, 0):     "09cce2fe-9f90-5e59-8474-da7e2630c341",
		kyouIDOf(parentThreadID, roleAssistant, 1): "d5d34551-5834-562e-b1d5-a22d71bee042",
	}
	for got, want := range cases {
		if got != want {
			t.Errorf("KyouID = %q, want %q", got, want)
		}
	}
	// ロールと連番が違えば必ず別ID
	seen := map[string]struct{}{}
	for _, role := range []string{roleHuman, roleAssistant} {
		for ordinal := range int64(4) {
			id := kyouIDOf(parentThreadID, role, ordinal)
			if _, dup := seen[id]; dup {
				t.Fatalf("IDが衝突した: %s", id)
			}
			seen[id] = struct{}{}
		}
	}
	// スレッドが違えば別ID(サブエージェントは親のsession_idを持つので、
	// ここがファイル名のuuid由来でないと親子が衝突する)
	if kyouIDOf(parentThreadID, roleHuman, 0) == kyouIDOf(subThreadID, roleHuman, 0) {
		t.Error("別スレッドで同じIDになっている")
	}
}

func TestKyouIDIsStableAcrossAppend(t *testing.T) {
	// ロールアウトは追記のみ。追記しても既存のKyouIDが動かないことを確かめる。
	// ここが崩れると、ユーザが付けたタグやテキストが更新のたびに迷子になる。
	dir := t.TempDir()
	name := "rollout-2026-01-02T10-00-00-" + parentThreadID + ".jsonl"
	target := filepath.Join(dir, name)

	original, err := os.ReadFile(parentFixture())
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	if err := os.WriteFile(target, original, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	before := foldGroup(buildGroupFromFiles(t, nil, target), subagentModeFold)
	if len(before) == 0 {
		t.Fatal("追記前のKyouが空")
	}

	var appended strings.Builder
	appended.Write(original)
	for i := range 20 {
		appended.WriteString(`{"timestamp":"2026-01-02T02:00:0` + string(rune('0'+i%10)) +
			`.000Z","type":"event_msg","payload":{"type":"agent_message","message":"追記された応答"}}` + "\n")
	}
	if err := os.WriteFile(target, []byte(appended.String()), 0o600); err != nil {
		t.Fatalf("append: %v", err)
	}

	after := foldGroup(buildGroupFromFiles(t, nil, target), subagentModeFold)
	if len(after) < len(before) {
		t.Fatalf("追記でKyouが減った: %d -> %d", len(before), len(after))
	}
	for i, m := range before {
		if after[i].ID != m.ID {
			t.Errorf("[%d] KyouID が変わった: %s -> %s", i, m.ID, after[i].ID)
		}
		if after[i].Role != m.Role || after[i].Ordinal != m.Ordinal {
			t.Errorf("[%d] role/ordinal が変わった", i)
		}
	}
}

func TestSearchTextIncludesEverything(t *testing.T) {
	group := buildGroupFromFiles(t, map[string]string{parentThreadID: "バグ修正スレッド"},
		parentFixture(), subAgentFixture())
	messages := foldGroup(group, subagentModeFold)

	human := searchTextOf(messages[0])
	for _, want := range []string{"バグを直してください", "myproj", "main", "gpt-5.3-codex", "src/util.go", "codex_vscode"} {
		if !strings.Contains(human, want) {
			t.Errorf("人間の検索テキストに %q が無い", want)
		}
	}

	assistant := searchTextOf(messages[1])
	for _, want := range []string{
		"調べます。",         // 本文
		"原因を切り分ける",      // 思考
		"shell_command", // ツール名
		"go test ./...", // ツールの要約
		"src/main.go",   // 変更ファイル
		"## 計画",         // 計画
		"Singer",        // サブエージェント
		"/root/explore",
		"調べてください。", // サブエージェントへの指示
	} {
		if !strings.Contains(assistant, want) {
			t.Errorf("応答の検索テキストに %q が無い", want)
		}
	}

	// ツールの実行結果は入れない。それが実データのバイトの94.7%
	if strings.Contains(assistant, "ツールの実行結果") {
		t.Error("ツールの実行結果が検索テキストに入っている")
	}
	// スレッド名は入れない(session_index.jsonl は名前が付くたび書き換わるので、
	// 焼き込むと畳み直しが要る。照合時に別カラムと連結する)
	if strings.Contains(assistant, "バグ修正スレッド") {
		t.Error("スレッド名が検索テキストに焼き込まれている")
	}
}

func TestSearchTextIsBounded(t *testing.T) {
	// 実データでは1件だけ5.4MBに達するKyouがある(サブエージェント9本を畳み込んだ回)。
	// 上限が無いと単語検索のたびにその1行を読むことになる。
	huge := message{Role: roleAssistant}
	for range 5000 {
		huge.Items = append(huge.Items, turnItem{Kind: blockText, Text: strings.Repeat("x", 1000)})
	}
	got := searchTextOf(huge)
	if len(got) > maxSearchTextBytes {
		t.Errorf("検索テキストが %d バイト。上限 %d を超えている", len(got), maxSearchTextBytes)
	}
	if len(got) < maxSearchTextBytes/2 {
		t.Errorf("検索テキストが %d バイトしかない。上限まで詰めるべき", len(got))
	}
}
