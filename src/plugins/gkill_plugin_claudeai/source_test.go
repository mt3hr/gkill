package main

import (
	"path/filepath"
	"testing"

	"github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
)

// relevantProblems は設定画面に出す走査の問題を絞る:
//   - ZIP の中の ZIP・混在した書き出し・マッチしない指定は出さない（この形式では意味が無い）
//   - 展開済みフォルダは、プラグイン自身のフォルダなら出さず、それ以外は文言を「ZIP ではない」に差し替える
//   - それ以外（壊れた ZIP など）はそのまま
func TestRelevantProblemsDropsNoiseAndRewritesExtractedFolder(t *testing.T) {
	own := t.TempDir()
	problems := []sdk.SourceProblem{
		{Kind: sdk.ProblemNestedZip, Path: "a.zip", Message: "nested"},
		{Kind: sdk.ProblemMixedExports, Path: "dir", Message: "mixed"},
		{Kind: sdk.ProblemMissingPattern, Path: "x/*.zip", Message: "missing"},
		{Kind: sdk.ProblemExtractedFolder, Path: own, Message: "orig"},
		{Kind: sdk.ProblemExtractedFolder, Path: filepath.Join(own, "other"), Message: "orig"},
		{Kind: sdk.ProblemBrokenArchive, Path: "b.zip", Message: "broken"},
	}
	kept := relevantProblems(problems, own)
	if len(kept) != 2 {
		t.Fatalf("kept = %+v, want 2件", kept)
	}
	if kept[0].Kind != sdk.ProblemExtractedFolder || kept[0].Message != extractedFolderMessage {
		t.Errorf("展開済みフォルダ = %+v, want 文言を差し替えたもの", kept[0])
	}
	if kept[1].Kind != sdk.ProblemBrokenArchive || kept[1].Message != "broken" {
		t.Errorf("壊れた ZIP = %+v, want そのまま", kept[1])
	}
	// pluginDir が空なら展開済みフォルダは全部残す（自分のフォルダを判定できない）
	if got := relevantProblems(problems, ""); len(got) != 3 {
		t.Errorf("pluginDir 無し = %d 件, want 3", len(got))
	}
}

// sourceSignature は (Path, CRC32, Size) の集合の署名。列挙順に依存せず、中身が変われば変わる。
// 同じ署名なら取り込み直さない（mtime は Takeout で当てにならないので使わない）。
func TestSourceSignatureIsOrderIndependentAndContentSensitive(t *testing.T) {
	a := sdk.SourceEntry{Path: "x.zip!/conversations.json", CRC32: 0x11, Size: 10}
	b := sdk.SourceEntry{Path: "y.zip!/conversations-000.json", CRC32: 0x22, Size: 20}
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
