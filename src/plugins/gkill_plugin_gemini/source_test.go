package main

import (
	"path/filepath"
	"testing"

	"github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
)

// エントリは**名前ではなく位置**で選ぶ。Takeout はフォルダ名もファイル名もアカウントの
// 言語で訳すので、"MyActivity.json" を探すと日本語環境で黙って0件になる。
// 訳されないのは製品名の "Gemini" だけ。
func TestIsGeminiActivityEntry(t *testing.T) {
	for _, c := range []struct {
		name  string
		entry string
		want  bool
	}{
		{"日本語の配置", jaActivityDir + jaActivityCSV, true},
		{"英語の配置", enActivityDir + enActivityCSV, true},
		{"英語フォルダに日本語ファイル名が来ても拾う", enActivityDir + jaActivityCSV, true},
		{"HTML も拾う（形式の警告を出すため）", jaActivityDir + "マイアクティビティ.html", true},
		{"トップレベルの Gemini フォルダ", "Takeout/Gemini/gemini_gems_data.html", true},
		{"別製品のアクティビティ", "Takeout/マイ アクティビティ/Chrome/マイアクティビティ.json", false},
		{"Gemini フォルダの添付（音声）", jaActivityDir + "00000000-0000-0000-0000-000000000001.wav", false},
		{"Gemini フォルダの添付（ZIP）", jaActivityDir + "00000000-0000-0000-0000-000000000002.zip", false},
		{"Fitbit の CSV", "Takeout/Google Health/Physical Activity_GoogleData/steps_2024-01-01.csv", false},
		{"タイムライン", "Takeout/タイムライン/Timeline Edits.json", false},
		{"フォルダ直下のファイル", "MyActivity.json", false},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := isGeminiActivityEntry(c.entry); got != c.want {
				t.Errorf("isGeminiActivityEntry(%q) = %v, want %v", c.entry, got, c.want)
			}
		})
	}
}

// relevantProblems は設定画面に出す走査の問題を絞る:
//   - ZIP の中の ZIP は出さない（Gemini の添付が ZIP で入っていて、実物では百件規模になる）
//   - マッチしない指定は出さない（設定画面が展開結果から直接出すので二重になる）
//   - 展開済みフォルダは、プラグイン自身のフォルダなら出さず、それ以外は文言を差し替える
//   - **混在した書き出しは残す**（Takeout では世代が並ぶのが普通で、利用者が知りたい情報）
//   - それ以外（壊れた ZIP など）はそのまま
func TestRelevantProblemsKeepsMixedExportsAndDropsNestedZip(t *testing.T) {
	own := t.TempDir()
	problems := []sdk.SourceProblem{
		{Kind: sdk.ProblemNestedZip, Path: "a.zip!/x.zip", Message: "nested"},
		{Kind: sdk.ProblemMissingPattern, Path: "x/*.zip", Message: "missing"},
		{Kind: sdk.ProblemMixedExports, Path: "dir", Message: "mixed"},
		{Kind: sdk.ProblemExtractedFolder, Path: own, Message: "orig"},
		{Kind: sdk.ProblemExtractedFolder, Path: filepath.Join(own, "other"), Message: "orig"},
		{Kind: sdk.ProblemBrokenArchive, Path: "b.zip", Message: "broken"},
	}
	kept := relevantProblems(problems, own)
	if len(kept) != 3 {
		t.Fatalf("kept = %+v, want 3件", kept)
	}
	if kept[0].Kind != sdk.ProblemMixedExports || kept[0].Message != "mixed" {
		t.Errorf("混在した書き出し = %+v, want そのまま残る", kept[0])
	}
	if kept[1].Kind != sdk.ProblemExtractedFolder || kept[1].Message != extractedFolderMessage {
		t.Errorf("展開済みフォルダ = %+v, want 文言を差し替えたもの", kept[1])
	}
	if kept[2].Kind != sdk.ProblemBrokenArchive || kept[2].Message != "broken" {
		t.Errorf("壊れた ZIP = %+v, want そのまま", kept[2])
	}
	// pluginDir が空なら展開済みフォルダは全部残す（自分のフォルダを判定できない）
	if got := relevantProblems(problems, ""); len(got) != 4 {
		t.Errorf("pluginDir 無し = %d 件, want 4", len(got))
	}
}

// sourceSignature は (Path, CRC32, Size) の集合の署名。列挙順に依存せず、中身が変われば変わる。
// 同じ署名なら取り込み直さない（mtime は Takeout で当てにならないので使わない。ADR-0303）。
func TestSourceSignatureIsOrderIndependentAndContentSensitive(t *testing.T) {
	a := sdk.SourceEntry{Path: "x.zip!/" + jaActivityDir + jaActivityCSV, CRC32: 0x11, Size: 10}
	b := sdk.SourceEntry{Path: "y.zip!/" + enActivityDir + enActivityCSV, CRC32: 0x22, Size: 20}
	if sourceSignature([]sdk.SourceEntry{a, b}) != sourceSignature([]sdk.SourceEntry{b, a}) {
		t.Error("列挙順で署名が変わる")
	}
	changedCRC := a
	changedCRC.CRC32 = 0x33
	if sourceSignature([]sdk.SourceEntry{changedCRC, b}) == sourceSignature([]sdk.SourceEntry{a, b}) {
		t.Error("CRC32 が変わっても署名が同じ")
	}
	changedSize := a
	changedSize.Size = 11
	if sourceSignature([]sdk.SourceEntry{changedSize, b}) == sourceSignature([]sdk.SourceEntry{a, b}) {
		t.Error("Size が変わっても署名が同じ")
	}
	if sourceSignature(nil) != "" {
		t.Error("空の署名が空文字でない")
	}
}

// Kyou ID は (会話 ID, 時刻, 役割, 連番) から決定的に作る。
// 書き出しにメッセージ ID が無いので、ここが同じなら何度取り込んでも同じ ID になる。
// **この導出を変えると全 Kyou の ID が変わり、利用者が付けたタグが全部外れる。**
func TestKyouIDOfIsStableAndDistinct(t *testing.T) {
	base := kyouIDOf(testConvA, testTimeA, senderUser, 0)
	if base != kyouIDOf(testConvA, testTimeA, senderUser, 0) {
		t.Error("同じ材料から違う ID が出ている")
	}
	for _, other := range []string{
		kyouIDOf(testConvB, testTimeA, senderUser, 0),
		kyouIDOf(testConvA, testTimeB, senderUser, 0),
		kyouIDOf(testConvA, testTimeA, senderModel, 0),
		kyouIDOf(testConvA, testTimeA, senderUser, 1),
	} {
		if other == base {
			t.Errorf("材料が違うのに同じ ID: %s", other)
		}
	}
	// UUID の形（36文字・ハイフン4本・version 5）であること。
	if len(base) != 36 || base[8] != '-' || base[13] != '-' || base[14] != '5' || base[18] != '-' || base[23] != '-' {
		t.Errorf("UUIDv5 の形になっていない: %s", base)
	}
}
