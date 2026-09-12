package reps

// キーワード判定そのもの（肯定語・除外語・AND/OR・ID 前方一致）のテストは
// api/find_word/match_words_test.go にある。ここは rep 側で検索対象テキストと ID を
// 組み立てるヘルパのテストだけ。

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/api/find_word"
)

func TestFindWordTextOfIDFKyou(t *testing.T) {
	dir := t.TempDir()

	// 本文を持つファイルは、rep内相対パスに続けて本文も検索対象になる。
	mdPath := filepath.Join(dir, "memo.md")
	if err := os.WriteFile(mdPath, []byte("Hello From Body"), os.ModePerm); err != nil {
		t.Fatalf("error at write file: %v", err)
	}
	got := findWordTextOfIDFKyou(context.Background(), "memo.md", mdPath)
	if !strings.Contains(got, "memo.md") || !strings.Contains(got, "hello from body") {
		t.Errorf("相対パスと本文の両方が小文字で含まれるべき: got %q", got)
	}

	// 画像などは本文を読まない。
	pngPath := filepath.Join(dir, "Image.PNG")
	if err := os.WriteFile(pngPath, []byte("binary"), os.ModePerm); err != nil {
		t.Fatalf("error at write file: %v", err)
	}
	got = findWordTextOfIDFKyou(context.Background(), "Image.PNG", pngPath)
	if got != "image.png" {
		t.Errorf("本文を読まない拡張子では相対パスだけを返すべき: got %q", got)
	}

	// 絶対パスは検索対象に含めない。
	// 含めていたころは、除外語にrepのフォルダ名を書くとrepが丸ごと消えていた。
	got = findWordTextOfIDFKyou(context.Background(), "memo.md", mdPath)
	if strings.Contains(got, strings.ToLower(dir)) {
		t.Errorf("絶対パスを検索対象に含めてはいけない: got %q (dir=%q)", got, dir)
	}

	// 読めないファイルでも検索は続行し、相対パスだけで判定する。
	missingPath := filepath.Join(dir, "missing.txt")
	got = findWordTextOfIDFKyou(context.Background(), "missing.txt", missingPath)
	if got != "missing.txt" {
		t.Errorf("読めないファイルは本文なしとして扱うべき: got %q", got)
	}
}

func TestFindWordTextOfGitCommit(t *testing.T) {
	got := findWordTextOfGitCommit("Fix Search")
	if got != "fix search" {
		t.Errorf("メッセージが小文字で返るべき: got %q", got)
	}
	// コミットIDはテキストに連結しない（id 引数で前方一致させる）。
	// 連結していたころは短い hex 語で無関係なコミットが当たっていた。
	if strings.Contains(got, "abcdef") {
		t.Errorf("コミットIDをテキストに含めてはいけない: got %q", got)
	}
	id := findWordIDOf(&find.FindQuery{}, "ABCDEF0123")
	if !find_word.MatchLoweredWords(got, id, []string{"abcdef"}, nil, true) {
		t.Errorf("コミットIDの前方一致で当たるべき")
	}
	if find_word.MatchLoweredWords(got, id, []string{"def0123"}, nil, true) {
		t.Errorf("コミットIDの途中の部分一致で当たってはいけない")
	}
}

// findWordIDOf は WordsSkipIDMatch が立っていれば空（ID 照合なし）、そうでなければ小文字の ID を返す。
func TestFindWordIDOf(t *testing.T) {
	if got := findWordIDOf(&find.FindQuery{}, "ABC-123"); got != "abc-123" {
		t.Errorf("小文字の ID を返すべき: got %q", got)
	}
	if got := findWordIDOf(&find.FindQuery{WordsSkipIDMatch: true}, "ABC-123"); got != "" {
		t.Errorf("WordsSkipIDMatch のときは空を返すべき: got %q", got)
	}
}
