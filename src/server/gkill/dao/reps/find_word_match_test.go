package reps

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMatchFindWords(t *testing.T) {
	cases := []struct {
		name     string
		text     string
		words    []string
		notWords []string
		wordsAnd bool
		want     bool
	}{
		// 除外語だけを指定した検索。rykvで「-.jpg」と入力するとこの形になる。
		// 以前はここが必ず不一致になっていた（除外語の判定が代入になっていたため）。
		{
			name:     "除外語だけ_含まないので一致する_or",
			text:     "148161896_p0.png",
			notWords: []string{".jpg"},
			wordsAnd: false,
			want:     true,
		},
		{
			name:     "除外語だけ_含むので一致しない_or",
			text:     "hplzhz_agaakfv1.jpg",
			notWords: []string{".jpg"},
			wordsAnd: false,
			want:     false,
		},
		{
			name:     "除外語だけ_含まないので一致する_and",
			text:     "148161896_p0.png",
			notWords: []string{".jpg"},
			wordsAnd: true,
			want:     true,
		},
		// 肯定語が空のOR検索。以前はループが回らず必ず不一致だった。
		{
			name:     "肯定語も除外語も空_or_一致する",
			text:     "148161896_p0.png",
			wordsAnd: false,
			want:     true,
		},
		{
			name:     "肯定語も除外語も空_and_一致する",
			text:     "148161896_p0.png",
			wordsAnd: true,
			want:     true,
		},
		// 肯定語と除外語の併用。以前は除外語のループが肯定側の結果を上書きしていた。
		{
			name:     "肯定語と除外語の併用_肯定を満たし除外に触れない",
			text:     "148161896_p0.png",
			words:    []string{"p0"},
			notWords: []string{".jpg"},
			wordsAnd: true,
			want:     true,
		},
		{
			name:     "肯定語と除外語の併用_除外語を含む",
			text:     "148113820_p0.jpg",
			words:    []string{"p0"},
			notWords: []string{".jpg"},
			wordsAnd: true,
			want:     false,
		},
		{
			name:     "肯定語と除外語の併用_肯定語を含まない",
			text:     "148161896_x1.png",
			words:    []string{"p0"},
			notWords: []string{".jpg"},
			wordsAnd: true,
			want:     false,
		},
		// 除外語は複数指定でき、どれか1つでも含めば不一致。
		{
			name:     "除外語が複数_2つ目を含む",
			text:     "movie.mp4",
			notWords: []string{".jpg", ".mp4"},
			wordsAnd: false,
			want:     false,
		},
		{
			name:     "除外語が複数_どれも含まない",
			text:     "note.png",
			notWords: []string{".jpg", ".mp4"},
			wordsAnd: false,
			want:     true,
		},
		// and / or の基本。
		{
			name:     "and_すべて含む",
			text:     "github gkill",
			words:    []string{"github", "gkill"},
			wordsAnd: true,
			want:     true,
		},
		{
			name:     "and_片方しか含まない",
			text:     "github only",
			words:    []string{"github", "gkill"},
			wordsAnd: true,
			want:     false,
		},
		{
			name:     "or_片方だけ含めば一致する",
			text:     "github only",
			words:    []string{"github", "gkill"},
			wordsAnd: false,
			want:     true,
		},
		{
			name:     "or_どれも含まない",
			text:     "something else",
			words:    []string{"github", "gkill"},
			wordsAnd: false,
			want:     false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := matchFindWords(c.text, c.words, c.notWords, c.wordsAnd)
			if got != c.want {
				t.Errorf("matchFindWords(%q, %v, %v, %v) = %v, want %v", c.text, c.words, c.notWords, c.wordsAnd, got, c.want)
			}
		})
	}
}

func TestLowerFindWordsDoesNotMutateSource(t *testing.T) {
	// query は全repで共有されているので、元のスライスを書き換えてはいけない。
	source := []string{"JPG", "PNG"}
	lowered := lowerFindWords(source)

	if source[0] != "JPG" || source[1] != "PNG" {
		t.Errorf("元のスライスを書き換えてはいけない: got %v", source)
	}
	if lowered[0] != "jpg" || lowered[1] != "png" {
		t.Errorf("小文字化されていない: got %v", lowered)
	}
	if lowerFindWords(nil) != nil {
		t.Errorf("nilにはnilを返すべき")
	}
	if lowerFindWords([]string{}) != nil {
		t.Errorf("空スライスにはnilを返すべき")
	}
}

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
	got := findWordTextOfGitCommit("Fix Search", "ABCDEF0123")
	if !strings.Contains(got, "fix search") || !strings.Contains(got, "abcdef0123") {
		t.Errorf("メッセージとコミットIDの両方が小文字で含まれるべき: got %q", got)
	}
	// 境界をまたいだ語が誤ってヒットしないこと。
	if matchFindWords(got, []string{"searchabcdef"}, nil, true) {
		t.Errorf("メッセージとコミットIDの境界をまたいだ語が一致してはいけない: got %q", got)
	}
}
