package find_word

import (
	"slices"
	"testing"
)

func TestMatchLoweredWords(t *testing.T) {
	cases := []struct {
		name     string
		text     string
		id       string
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
		// ID は前方一致だけ。UUID 丸ごとの貼り付けと git の短縮ハッシュはこれで引ける。
		{
			name:     "id_前方一致で一致する",
			text:     "something else",
			id:       "abcdef01-2345-6789-abcd-ef0123456789",
			words:    []string{"abcdef01"},
			wordsAnd: true,
			want:     true,
		},
		{
			name:     "id_途中の部分一致では一致しない",
			text:     "something else",
			id:       "abcdef01-2345-6789-abcd-ef0123456789",
			words:    []string{"2345"},
			wordsAnd: true,
			want:     false,
		},
		{
			name:     "id_and検索でも語ごとにtextかidのどちらかで足りる",
			text:     "github only",
			id:       "abcdef01-2345",
			words:    []string{"github", "abcd"},
			wordsAnd: true,
			want:     true,
		},
		// 除外語は ID を見ない。`-1` で UUID に 1 を含む記録が消えていた事故の再発防止。
		{
			name:     "除外語はidの前方一致でも消えない",
			text:     "note.png",
			id:       "abcdef01-2345",
			notWords: []string{"abc"},
			wordsAnd: false,
			want:     true,
		},
		// id が空なら ID 照合なし（除外語を肯定語として再検索する内部クエリ用）。
		{
			name:     "id空_id照合なし",
			text:     "note.png",
			id:       "",
			words:    []string{"abc"},
			wordsAnd: false,
			want:     false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := MatchLoweredWords(c.text, c.id, c.words, c.notWords, c.wordsAnd)
			if got != c.want {
				t.Errorf("MatchLoweredWords(%q, %q, %v, %v, %v) = %v, want %v", c.text, c.id, c.words, c.notWords, c.wordsAnd, got, c.want)
			}
		})
	}
}

func TestLowerWordsDoesNotMutateSource(t *testing.T) {
	// query は全repで共有されているので、元のスライスを書き換えてはいけない。
	source := []string{"JPG", "PNG"}
	lowered := LowerWords(source)

	if source[0] != "JPG" || source[1] != "PNG" {
		t.Errorf("元のスライスを書き換えてはいけない: got %v", source)
	}
	if lowered[0] != "jpg" || lowered[1] != "png" {
		t.Errorf("小文字化されていない: got %v", lowered)
	}
	if LowerWords(nil) != nil {
		t.Errorf("nilにはnilを返すべき")
	}
	if LowerWords([]string{}) != nil {
		t.Errorf("空スライスにはnilを返すべき")
	}
}

func TestNormalizeWords(t *testing.T) {
	cases := []struct {
		name  string
		words []string
		want  []string
	}{
		{name: "nilはnilのまま", words: nil, want: nil},
		{name: "空スライスは非nilの空のまま", words: []string{}, want: []string{}},
		{name: "空文字だけなら非nilの空になる", words: []string{""}, want: []string{}},
		{name: "半角空白だけの語は捨てる", words: []string{" ", "foo"}, want: []string{"foo"}},
		{name: "全角空白だけの語も捨てる", words: []string{"　", "foo"}, want: []string{"foo"}},
		{name: "前後の空白を落とす", words: []string{" foo ", "　bar　"}, want: []string{"foo", "bar"}},
		{name: "語中の空白は残す", words: []string{"foo bar"}, want: []string{"foo bar"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := NormalizeWords(c.words)
			if (got == nil) != (c.want == nil) {
				t.Fatalf("NormalizeWords(%q) nil性が違う: got %#v, want %#v", c.words, got, c.want)
			}
			if !slices.Equal(got, c.want) {
				t.Errorf("NormalizeWords(%q) = %q, want %q", c.words, got, c.want)
			}
		})
	}
}

func TestNormalizeWordsDoesNotMutateSource(t *testing.T) {
	source := []string{" foo ", ""}
	_ = NormalizeWords(source)
	if source[0] != " foo " || source[1] != "" {
		t.Errorf("元のスライスを書き換えてはいけない: got %q", source)
	}
}
