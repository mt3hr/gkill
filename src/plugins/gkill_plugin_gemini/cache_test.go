package main

import (
	"archive/zip"
	"bytes"
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	sdk "github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
)

// errTestIngest はバッチ取り込みを故意に失敗させるためのエラー。
var errTestIngest = errors.New("ingest failed for test")

// testRecord は合成のマイアクティビティのフィクスチャ。実データは使わない(PII対策)。
type testRecord struct {
	// conv は会話 ID。空なら details を出さない（会話にひも付かない記録）。
	conv string
	// at は RFC 3339 の時刻。
	at string
	// title は書き出しのままの title（接頭辞込み）。
	title string
	// replies は safeHtmlItem[].html。実物は1つのことが多いが2つ以上のこともある。
	replies []string
	// attached は attachedFiles。
	attached []string
	// subtitle は subtitles[0].name。空なら出さない。
	subtitle string
	// header は製品名。空なら日本語の既定にする。
	header string
}

const (
	// 合成の会話 ID と時刻。実物の書き出しから写した値は置かない。
	testConvA = "0123456789abcdef"
	testConvB = "fedcba9876543210"
	testTimeA = "2024-01-01T00:00:00.000Z"
	testTimeB = "2024-01-02T00:00:00.000Z"
	testTimeC = "2024-01-03T00:00:00.000Z"

	// 日本語アカウントと英語アカウントのフォルダ配置。ファイル名まで訳される。
	jaActivityDir = "Takeout/マイ アクティビティ/Gemini アプリ/"
	jaActivityCSV = "マイアクティビティ.json"
	enActivityDir = "Takeout/My Activity/Gemini Apps/"
	enActivityCSV = "MyActivity.json"

	// 日本語の title の接頭辞。英語は "Prompted " のようにコロンを使わない。
	jaPromptPrefix = "送信したメッセージ: "

	testZipName    = "takeout-20240101T000000Z-1-001.zip"
	testZipNameOld = "takeout-20230101T000000Z-1-001.zip"
)

var testZipModified = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

// activityJSON はマイアクティビティの書き出しと同じ形の JSON 配列を作る。
func activityJSON(t *testing.T, records []testRecord) []byte {
	t.Helper()
	arr := make([]map[string]any, 0, len(records))
	for _, r := range records {
		header := r.header
		if header == "" {
			header = "Gemini アプリ"
		}
		record := map[string]any{
			"header":           header,
			"title":            r.title,
			"time":             r.at,
			"products":         []any{header},
			"activityControls": []any{header + " アクティビティ"},
		}
		if r.conv != "" {
			url := "https://gemini.google.com/app/" + r.conv
			record["details"] = []any{map[string]any{"name": url, "url": url}}
		}
		if len(r.replies) != 0 {
			items := make([]any, 0, len(r.replies))
			for _, html := range r.replies {
				items = append(items, map[string]any{"html": html})
			}
			record["safeHtmlItem"] = items
		}
		if len(r.attached) != 0 {
			attached := make([]any, 0, len(r.attached))
			for _, name := range r.attached {
				attached = append(attached, name)
			}
			record["attachedFiles"] = attached
		}
		if r.subtitle != "" {
			record["subtitles"] = []any{map[string]any{"name": r.subtitle}}
		}
		arr = append(arr, record)
	}
	data, err := json.MarshalIndent(arr, "", " ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return data
}

// writeZip は dir/zipName に entries を詰めた ZIP を書く（既にあれば作り直す）。
// バイナリの fixture は commit せず、テストのたびに作る。
func writeZip(t *testing.T, dir, zipName string, entries map[string][]byte, modified time.Time) string {
	t.Helper()
	buffer := &bytes.Buffer{}
	writer := zip.NewWriter(buffer)
	for _, name := range slices.Sorted(maps.Keys(entries)) {
		header := &zip.FileHeader{Name: name, Method: zip.Deflate}
		header.Modified = modified
		entry, err := writer.CreateHeader(header)
		if err != nil {
			t.Fatalf("create zip entry %s: %v", name, err)
		}
		if _, err := entry.Write(entries[name]); err != nil {
			t.Fatalf("write zip entry %s: %v", name, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close zip writer: %v", err)
	}
	zipPath := filepath.Join(dir, zipName)
	if err := os.WriteFile(zipPath, buffer.Bytes(), 0o600); err != nil {
		t.Fatalf("write %s: %v", zipPath, err)
	}
	return zipPath
}

// noiseEntries は実物の Takeout に必ず並ぶ「読まないもの」。
// 毎回同梱して、読み飛ばされることと設定画面を埋めないことを通す。
func noiseEntries(t *testing.T) map[string][]byte {
	t.Helper()
	return map[string][]byte{
		// トップレベルの「Gemini」。Gem と予定アクションだけで、実物も中身が空。
		"Takeout/Gemini/gemini_gems_data.html":              []byte("<div></div>"),
		"Takeout/Gemini/gemini_scheduled_actions_data.html": []byte("<div></div>"),
		// 別製品のマイアクティビティ（同じファイル名で並ぶ）。
		"Takeout/マイ アクティビティ/Chrome/マイアクティビティ.json": activityJSON(t, []testRecord{{
			conv: "", at: testTimeA, title: "閲覧しました: どこかのページ", header: "Chrome",
		}}),
		// Gemini の添付。音声と ZIP が同じフォルダに並ぶ（ZIP の中の ZIP）。
		jaActivityDir + "00000000-0000-0000-0000-000000000001.wav": []byte("RIFF0000WAVE"),
		jaActivityDir + "00000000-0000-0000-0000-000000000002.zip": []byte("PK\x03\x04"),
		// 他の Takeout データ。
		"Takeout/タイムライン/Settings.json": []byte(`{"deviceSettings":[]}`),
	}
}

// writeActivityZipAt は dir に Takeout の ZIP を1つ書く。
func writeActivityZipAt(t *testing.T, dir, zipName, activityDir, activityName string, records []testRecord, modified time.Time) string {
	t.Helper()
	entries := noiseEntries(t)
	entries[activityDir+activityName] = activityJSON(t, records)
	return writeZip(t, dir, zipName, entries, modified)
}

// writeActivityZip は日本語アカウントの配置で既定の ZIP を書く。
func writeActivityZip(t *testing.T, dir string, records []testRecord) string {
	t.Helper()
	return writeActivityZipAt(t, dir, testZipName, jaActivityDir, jaActivityCSV, records, testZipModified)
}

// newTestCache は空のキャッシュとプラグインフォルダを用意する。
func newTestCache(t *testing.T) (*pluginCache, string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("GKILL_HOME", home)
	pluginDir := filepath.Join(home, "plugins", "testuser", "gkill_plugin_gemini")
	if err := os.MkdirAll(pluginDir, os.ModePerm); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	c := &pluginCache{}
	t.Cleanup(func() {
		if c.db != nil {
			_ = c.db.Close()
		}
	})
	return c, pluginDir
}

func countMessages(t *testing.T, c *pluginCache) int {
	t.Helper()
	n := 0
	if err := c.conn().QueryRow(`SELECT COUNT(*) FROM msg_cache`).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	return n
}

// textsBySender はキャッシュの本文を役割ごとに集める。
func textsBySender(t *testing.T, c *pluginCache, pluginDir string, sender string) []string {
	t.Helper()
	msgs, err := c.GetMessages(pluginDir)
	if err != nil {
		t.Fatalf("GetMessages: %v", err)
	}
	var texts []string
	for _, m := range msgs {
		if m.Sender == sender {
			texts = append(texts, m.Text)
		}
	}
	return texts
}

// problemMessages は設定画面に出る問題の文言を集める。
func problemMessages(c *pluginCache) []string {
	var messages []string
	for _, problem := range c.loadSourceProblems() {
		messages = append(messages, problem.Message)
	}
	return messages
}

// 日本語アカウント（フォルダ名もファイル名も訳される）でも英語アカウントでも読める。
// **ファイル名で選ぶと日本語環境で黙って0件になる**ので、ここが最初の防御線。
func TestGeminiReadsLocalizedAndEnglishLayouts(t *testing.T) {
	for _, layout := range []struct {
		name string
		dir  string
		file string
	}{
		{"日本語", jaActivityDir, jaActivityCSV},
		{"英語", enActivityDir, enActivityCSV},
	} {
		t.Run(layout.name, func(t *testing.T) {
			c, pluginDir := newTestCache(t)
			dataDir := t.TempDir()
			writeActivityZipAt(t, dataDir, testZipName, layout.dir, layout.file, []testRecord{{
				conv: testConvA, at: testTimeA,
				title:   jaPromptPrefix + "これは何ですか",
				replies: []string{"<p>これは<strong>テスト</strong>です。</p>"},
			}}, testZipModified)

			if err := c.build(pluginDir, []string{dataDir}); err != nil {
				t.Fatalf("build: %v", err)
			}
			if got := countMessages(t, c); got != 2 {
				t.Fatalf("発言数 = %d, want 2（プロンプトと返答）", got)
			}
			if got := textsBySender(t, c, pluginDir, senderUser); len(got) != 1 || got[0] != "これは何ですか" {
				t.Errorf("プロンプト = %q, want 接頭辞を落としたもの", got)
			}
			if got := textsBySender(t, c, pluginDir, senderModel); len(got) != 1 || got[0] != "これは**テスト**です。" {
				t.Errorf("返答 = %q, want HTML を平文に写したもの", got)
			}
		})
	}
}

// マイアクティビティには製品ごとに同じ名前のファイルが並ぶ。Gemini 以外は読まない。
// Takeout のトップレベルの「Gemini」（Gem と予定アクションの空 HTML）は、
// 警告も出さずに無視する —— 毎回必ず入っているので、警告にすると常に出続ける。
func TestGeminiIgnoresOtherProductsAndGemsPlaceholder(t *testing.T) {
	c, pluginDir := newTestCache(t)
	dataDir := t.TempDir()
	writeActivityZip(t, dataDir, []testRecord{{
		conv: testConvA, at: testTimeA, title: jaPromptPrefix + "ねえ",
		replies: []string{"<p>はい</p>"},
	}})

	if err := c.build(pluginDir, []string{dataDir}); err != nil {
		t.Fatalf("build: %v", err)
	}
	for _, text := range textsBySender(t, c, pluginDir, senderUser) {
		if strings.Contains(text, "どこかのページ") {
			t.Errorf("別製品（Chrome）のアクティビティを読んでいる: %q", text)
		}
	}
	for _, message := range problemMessages(c) {
		t.Errorf("警告を出すべきでない入力で警告が出ている: %q", message)
	}
}

// ZIP の中の ZIP（Gemini の添付）は、実物では1フォルダに多数並ぶ。
// そのまま警告にすると設定画面が埋まって他の警告が読めなくなるので、出さない。
func TestGeminiNestedZipAttachmentsDoNotFloodProblems(t *testing.T) {
	c, pluginDir := newTestCache(t)
	dataDir := t.TempDir()
	entries := noiseEntries(t)
	for i := range 20 {
		entries[fmt.Sprintf("%sattachment-%03d.zip", jaActivityDir, i)] = []byte("PK\x03\x04")
	}
	entries[jaActivityDir+jaActivityCSV] = activityJSON(t, []testRecord{{
		conv: testConvA, at: testTimeA, title: jaPromptPrefix + "添付つき",
		replies: []string{"<p>受け取りました</p>"},
	}})
	writeZip(t, dataDir, testZipName, entries, testZipModified)

	if err := c.build(pluginDir, []string{dataDir}); err != nil {
		t.Fatalf("build: %v", err)
	}
	if got := problemMessages(c); len(got) != 0 {
		t.Errorf("添付の ZIP で警告が %d 件出ている: %v", len(got), got)
	}
}

// HTML 形式で書き出してしまったときは、読まないことを設定画面に出す。
// 黙って0件にすると「プラグインが壊れている」と読まれる（ADR-0310 と同じ考え方）。
func TestGeminiHTMLFormatIsReported(t *testing.T) {
	c, pluginDir := newTestCache(t)
	dataDir := t.TempDir()
	entries := noiseEntries(t)
	entries[jaActivityDir+"マイアクティビティ.html"] = []byte(
		`<html><body><div class="outer-cell"><a href="https://gemini.google.com/app/` + testConvA + `">詳細</a></div></body></html>`)
	writeZip(t, dataDir, testZipName, entries, testZipModified)

	err := c.build(pluginDir, []string{dataDir})
	if err == nil {
		t.Fatal("JSON が1つも無いのにエラーを返していない")
	}
	found := false
	for _, message := range problemMessages(c) {
		if message == htmlFormatMessage {
			found = true
		}
	}
	if !found {
		t.Errorf("HTML 形式の警告が出ていない: %v", problemMessages(c))
	}
}

// プロンプトも返答も無い記録（「アシスタント機能を使用」など）は Kyou を作らない。
// title をそのままプロンプトにすると、操作の記録が人間の発言として一覧に並ぶ。
func TestGeminiRecordWithoutPromptAndReplyIsSkipped(t *testing.T) {
	c, pluginDir := newTestCache(t)
	dataDir := t.TempDir()
	writeActivityZip(t, dataDir, []testRecord{
		{conv: testConvA, at: testTimeA, title: "アシスタント機能を使用"},
		{conv: testConvA, at: testTimeB, title: jaPromptPrefix + "これは残る", replies: []string{"<p>はい</p>"}},
	})

	if err := c.build(pluginDir, []string{dataDir}); err != nil {
		t.Fatalf("build: %v", err)
	}
	if got := countMessages(t, c); got != 2 {
		t.Fatalf("発言数 = %d, want 2（残る往復のぶんだけ）", got)
	}
	for _, text := range textsBySender(t, c, pluginDir, senderUser) {
		if strings.Contains(text, "アシスタント") {
			t.Errorf("操作の記録を人間の発言にしている: %q", text)
		}
	}
}

// safeHtmlItem が複数ある記録は、返答を順に連結する。
func TestGeminiMultipleRepliesAreJoined(t *testing.T) {
	c, pluginDir := newTestCache(t)
	dataDir := t.TempDir()
	writeActivityZip(t, dataDir, []testRecord{{
		conv: testConvA, at: testTimeA, title: jaPromptPrefix + "2つ返して",
		replies: []string{"<p>ひとつ目</p>", "<p>ふたつ目</p>"},
	}})

	if err := c.build(pluginDir, []string{dataDir}); err != nil {
		t.Fatalf("build: %v", err)
	}
	got := textsBySender(t, c, pluginDir, senderModel)
	if len(got) != 1 {
		t.Fatalf("返答 = %d件, want 1件（連結して1つ）", len(got))
	}
	if !strings.Contains(got[0], "ひとつ目") || !strings.Contains(got[0], "ふたつ目") {
		t.Errorf("返答 = %q, want 両方を含む", got[0])
	}
}

// 書き出しに会話の名前が無いので、その会話で最初に送ったプロンプトを見出しにする。
func TestGeminiConversationTitleIsOldestPrompt(t *testing.T) {
	c, pluginDir := newTestCache(t)
	dataDir := t.TempDir()
	// 書き出しは新しい順に並ぶ（実物もそう）。
	writeActivityZip(t, dataDir, []testRecord{
		{conv: testConvA, at: testTimeC, title: jaPromptPrefix + "あとの発言", replies: []string{"<p>b</p>"}},
		{conv: testConvA, at: testTimeA, title: jaPromptPrefix + "さいしょの発言", replies: []string{"<p>a</p>"}},
	})

	if err := c.build(pluginDir, []string{dataDir}); err != nil {
		t.Fatalf("build: %v", err)
	}
	title := ""
	if err := c.conn().QueryRow(`SELECT title FROM conv_cache WHERE conv_id = ?`, testConvA).Scan(&title); err != nil {
		t.Fatalf("conv_cache: %v", err)
	}
	if title != "さいしょの発言" {
		t.Errorf("会話の見出し = %q, want 最も古いプロンプト", title)
	}
}

// 同じ往復が新旧2つの書き出しに入っていても1件。新しい書き出しのほうを採る。
// Takeout は同じフォルダに世代が並ぶのが普通で、Gemini のアクティビティは
// 毎回「その時点までの全履歴」が入るので、畳まないと世代のぶんだけ並ぶ。
func TestGeminiSameExchangeAcrossExportsIsSingle(t *testing.T) {
	c, pluginDir := newTestCache(t)
	dataDir := t.TempDir()
	// 古い書き出し（更新時刻も古くする）。
	writeActivityZipAt(t, dataDir, testZipNameOld, jaActivityDir, jaActivityCSV, []testRecord{{
		conv: testConvA, at: testTimeA, title: jaPromptPrefix + "おなじ質問",
		replies: []string{"<p>ふるい返答</p>"},
	}}, testZipModified.AddDate(-1, 0, 0))
	// 新しい書き出し。同じ往復の返答だけが違う。
	writeActivityZipAt(t, dataDir, testZipName, jaActivityDir, jaActivityCSV, []testRecord{{
		conv: testConvA, at: testTimeA, title: jaPromptPrefix + "おなじ質問",
		replies: []string{"<p>あたらしい返答</p>"},
	}}, testZipModified)

	if err := c.build(pluginDir, []string{dataDir}); err != nil {
		t.Fatalf("build: %v", err)
	}
	if got := countMessages(t, c); got != 2 {
		t.Fatalf("発言数 = %d, want 2（往復1つぶん）", got)
	}
	replies := textsBySender(t, c, pluginDir, senderModel)
	if len(replies) != 1 || replies[0] != "あたらしい返答" {
		t.Errorf("返答 = %q, want 新しい書き出しのほう", replies)
	}
	// 同じフォルダに世代が並んでいることは設定画面に出す（chatgpt と違い、Takeout では意味がある）。
	found := false
	for _, problem := range c.loadSourceProblems() {
		if problem.Kind == "mixed_exports" {
			found = true
		}
	}
	if !found {
		t.Errorf("世代が混ざっている警告が出ていない: %v", problemMessages(c))
	}
}

// 署名(Path:CRC32:Size)が変われば作り直す。
func TestGeminiRebuildOnSourceChange(t *testing.T) {
	c, pluginDir := newTestCache(t)
	dataDir := t.TempDir()
	writeActivityZip(t, dataDir, []testRecord{{
		conv: testConvA, at: testTimeA, title: jaPromptPrefix + "はじめ", replies: []string{"<p>a</p>"},
	}})
	if err := c.build(pluginDir, []string{dataDir}); err != nil {
		t.Fatalf("build: %v", err)
	}
	if got := countMessages(t, c); got != 2 {
		t.Fatalf("1回目 = %d件, want 2件", got)
	}

	writeActivityZip(t, dataDir, []testRecord{
		{conv: testConvA, at: testTimeA, title: jaPromptPrefix + "はじめ", replies: []string{"<p>a</p>"}},
		{conv: testConvB, at: testTimeB, title: jaPromptPrefix + "つぎ", replies: []string{"<p>b</p>"}},
	})
	if err := c.build(pluginDir, []string{dataDir}); err != nil {
		t.Fatalf("build 2回目: %v", err)
	}
	if got := countMessages(t, c); got != 4 {
		t.Errorf("2回目 = %d件, want 4件", got)
	}
}

// 中身が同じで更新時刻だけ違う ZIP では作り直さない。
// Takeout は書き出し時刻を全エントリに入れるので、mtime は差分判定に使えない（ADR-0303）。
func TestGeminiSameContentDifferentMtimeDoesNotRebuild(t *testing.T) {
	c, pluginDir := newTestCache(t)
	dataDir := t.TempDir()
	records := []testRecord{{
		conv: testConvA, at: testTimeA, title: jaPromptPrefix + "はじめ", replies: []string{"<p>a</p>"},
	}}
	writeActivityZipAt(t, dataDir, testZipName, jaActivityDir, jaActivityCSV, records, testZipModified)
	if err := c.build(pluginDir, []string{dataDir}); err != nil {
		t.Fatalf("build: %v", err)
	}
	firstGen := c.getMeta("build_gen")

	writeActivityZipAt(t, dataDir, testZipName, jaActivityDir, jaActivityCSV, records, testZipModified.AddDate(0, 1, 0))
	if err := c.build(pluginDir, []string{dataDir}); err != nil {
		t.Fatalf("build 2回目: %v", err)
	}
	if got := c.getMeta("build_gen"); got != firstGen {
		t.Errorf("build_gen = %q, want %q（mtime だけの違いでは作り直さない）", got, firstGen)
	}
}

// バッチ永続性: 1バッチ目 commit 後に2つ目を故意に失敗させても、1つ目の行は残る。
// 途中で殺されても進捗ゼロに戻らないための性質（ADR-0305）。
func TestGeminiBatchCommitPersistsEarlierBatches(t *testing.T) {
	c, pluginDir := newTestCache(t)
	dataDir := t.TempDir()
	writeActivityZip(t, dataDir, []testRecord{
		{conv: testConvA, at: testTimeA, title: jaPromptPrefix + "ひとつ目", replies: []string{"<p>a</p>"}},
		{conv: testConvB, at: testTimeB, title: jaPromptPrefix + "ふたつ目", replies: []string{"<p>b</p>"}},
	})

	originalBatch, originalHook := buildBatchExchanges, ingestExchangeHook
	t.Cleanup(func() { buildBatchExchanges, ingestExchangeHook = originalBatch, originalHook })
	buildBatchExchanges = 1
	ingestExchangeHook = func(timeKey string) error {
		if strings.HasPrefix(timeKey, "2024-01-02") {
			return errTestIngest
		}
		return nil
	}

	if err := c.build(pluginDir, []string{dataDir}); err == nil {
		t.Fatal("2バッチ目で失敗させたのにエラーが返っていない")
	}
	if got := countMessages(t, c); got != 2 {
		t.Errorf("発言数 = %d, want 2（1バッチ目はコミット済み）", got)
	}
}

// 読み取りは buildMu を取らない。構築中でも「そこまで取り込めたぶん」を即返す。
func TestGeminiReadsDoNotTakeBuildMu(t *testing.T) {
	c, pluginDir := newTestCache(t)
	if err := c.openDB(pluginDir); err != nil {
		t.Fatalf("openDB: %v", err)
	}
	c.buildMu.Lock()
	defer c.buildMu.Unlock()

	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = c.GetMessages(pluginDir)
		_ = c.GetStats(pluginDir)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("構築中に読み取りがブロックされた（buildMu を取っている）")
	}
}

// 展開済みのフォルダ（ZIP ではないもの）を指したら設定画面に出す。黙って0件にしない。
func TestGeminiExtractedFolderIsReported(t *testing.T) {
	c, pluginDir := newTestCache(t)
	dataDir := t.TempDir()
	loose := filepath.Join(dataDir, "Takeout", "マイ アクティビティ", "Gemini アプリ")
	if err := os.MkdirAll(loose, os.ModePerm); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(loose, jaActivityCSV), activityJSON(t, []testRecord{{
		conv: testConvA, at: testTimeA, title: jaPromptPrefix + "展開済み", replies: []string{"<p>a</p>"},
	}}), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	if err := c.build(pluginDir, []string{dataDir}); err == nil {
		t.Fatal("ZIP が無いのにエラーを返していない")
	}
	if got := countMessages(t, c); got != 0 {
		t.Errorf("展開済みのフォルダを読んでいる: %d件", got)
	}
	found := false
	for _, message := range problemMessages(c) {
		if message == extractedFolderMessage {
			found = true
		}
	}
	if !found {
		t.Errorf("展開済みフォルダの警告が出ていない: %v", problemMessages(c))
	}
}

// source_dirs が空のときの既定（プラグインフォルダ自身）では警告を出さない。
// そこには manifest.json / config.json が必ずあるので、SDK は毎回 extracted_folder を立てる。
func TestGeminiPluginDirDefaultDoesNotWarn(t *testing.T) {
	c, pluginDir := newTestCache(t)
	if err := os.WriteFile(filepath.Join(pluginDir, "manifest.json"), []byte("{}"), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := c.build(pluginDir, []string{pluginDir}); err == nil {
		t.Fatal("ZIP が無いのにエラーを返していない")
	}
	if got := problemMessages(c); len(got) != 0 {
		t.Errorf("自分のフォルダで警告が出ている: %v", got)
	}
}

// GetMsgByID は存在する ID を返し、存在しない ID にはエラーを返す。
func TestGeminiGetMsgByIDHitAndMiss(t *testing.T) {
	c, pluginDir := newTestCache(t)
	dataDir := t.TempDir()
	writeActivityZip(t, dataDir, []testRecord{{
		conv: testConvA, at: testTimeA, title: jaPromptPrefix + "単体取得テスト",
		replies: []string{"<p>返答です</p>"},
	}})
	if err := c.build(pluginDir, []string{dataDir}); err != nil {
		t.Fatalf("build: %v", err)
	}
	msgs, err := c.GetMessages(pluginDir)
	if err != nil || len(msgs) != 2 {
		t.Fatalf("GetMessages = %d件, err=%v", len(msgs), err)
	}
	_, got, err := c.GetMsgByID(pluginDir, msgs[0].MsgID)
	if err != nil || got.MsgID != msgs[0].MsgID {
		t.Fatalf("GetMsgByID(hit) = %+v, err=%v", got, err)
	}
	if _, _, err := c.GetMsgByID(pluginDir, "00000000-0000-0000-0000-000000000000"); err == nil {
		t.Errorf("GetMsgByID(miss) expected error, got nil")
	}
}

// 1回の書き出し内で同じ (convID, 時刻) が複数並んでも、ordinal 採番により ID が衝突せず全件残る。
func TestGeminiSameTimestampRecordsGetDistinctIDs(t *testing.T) {
	c, pluginDir := newTestCache(t)
	dataDir := t.TempDir()
	writeActivityZip(t, dataDir, []testRecord{
		{conv: testConvA, at: testTimeA, title: jaPromptPrefix + "1件目", replies: []string{"<p>返答1</p>"}},
		{conv: testConvA, at: testTimeA, title: jaPromptPrefix + "2件目", replies: []string{"<p>返答2</p>"}},
	})
	if err := c.build(pluginDir, []string{dataDir}); err != nil {
		t.Fatalf("build: %v", err)
	}
	if got := countMessages(t, c); got != 4 {
		t.Errorf("同一時刻2往復の発言数 = %d, want 4", got)
	}
}

// マイアクティビティは1往復に時刻が1つしかない。返答をプロンプトの1秒後に置くので、
// gkill 本体と同じ規則（関連時刻の秒の降順 → ID）で並べると、往復ごとに必ず「返答 → プロンプト」の順になる。
// 同じ秒のままだと上下が UUIDv5 のハッシュ次第で、往復ごとにばらばらに入れ替わる。
// ずらすのは時刻だけで、Kyou ID は元の時刻から作る（利用者のタグが外れない）。
func TestGeminiReplyIsOrderedAfterPrompt(t *testing.T) {
	c, pluginDir := newTestCache(t)
	dataDir := t.TempDir()
	const testTimeA2 = "2024-01-01T00:01:00.000Z" // 合成値（テスト由来）
	writeActivityZip(t, dataDir, []testRecord{
		{conv: testConvA, at: testTimeA, title: jaPromptPrefix + "質問1", replies: []string{"<p>返答1</p>"}},
		{conv: testConvA, at: testTimeA2, title: jaPromptPrefix + "質問2", replies: []string{"<p>返答2</p>"}},
	})
	if err := c.build(pluginDir, []string{dataDir}); err != nil {
		t.Fatalf("build: %v", err)
	}
	msgs, err := c.GetMessages(pluginDir)
	if err != nil {
		t.Fatalf("GetMessages: %v", err)
	}

	textByID := map[string]string{}
	for _, m := range msgs {
		textByID[m.MsgID] = m.Text
	}

	// 時刻と ID。ID は元の時刻（ずらす前）から作ったものと一致する。
	for _, at := range []string{testTimeA, testTimeA2} {
		parsed, err := time.Parse(time.RFC3339Nano, at)
		if err != nil {
			t.Fatalf("parse %s: %v", at, err)
		}
		timeKey := parsed.UTC().Format(time.RFC3339Nano)
		userID := kyouIDOf(testConvA, timeKey, senderUser, 0)
		modelID := kyouIDOf(testConvA, timeKey, senderModel, 0)
		var user, model *cachedMessage
		for i := range msgs {
			switch msgs[i].MsgID {
			case userID:
				user = &msgs[i]
			case modelID:
				model = &msgs[i]
			}
		}
		if user == nil || model == nil {
			t.Fatalf("%s: 元の時刻から作った ID の発言が見つからない (user=%v model=%v)", at, user != nil, model != nil)
		}
		if user.RelatedTimeUnix != parsed.Unix() {
			t.Errorf("%s: プロンプトの時刻 = %d, want %d", at, user.RelatedTimeUnix, parsed.Unix())
		}
		if model.RelatedTimeUnix != parsed.Unix()+replyTimeOffsetSeconds || model.CreateTimeUnix != model.RelatedTimeUnix {
			t.Errorf("%s: 返答の時刻 = %d/%d, want %d", at, model.RelatedTimeUnix, model.CreateTimeUnix, parsed.Unix()+replyTimeOffsetSeconds)
		}
	}

	// gkill 本体の一覧と同じ規則で並べる（find_filter.go の sortResultKyousByKey）。
	kyous := kyousOfMessages(msgs, sdk.Query{})
	slices.SortFunc(kyous, func(a, b sdk.Kyou) int {
		if c := cmp.Compare(b.RelatedTime.Unix(), a.RelatedTime.Unix()); c != 0 {
			return c
		}
		return strings.Compare(a.ID, b.ID)
	})
	var got []string
	for _, k := range kyous {
		got = append(got, textByID[k.ID])
	}
	want := []string{"返答2", "質問2", "返答1", "質問1"}
	if !slices.Equal(got, want) {
		t.Errorf("一覧の並び = %v, want %v", got, want)
	}
}
