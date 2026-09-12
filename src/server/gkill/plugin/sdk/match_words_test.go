package sdk

import "testing"

// Query.MatchText は本体（api/find_word）と同じ規則で判定する。
// 細かい組み合わせは api/find_word/match_words_test.go が持つので、ここは SDK 側で足している
// 正規化（空語・前後空白）と大小無視、ID 前方一致、語なしの素通しを見る。
func TestQueryMatchText(t *testing.T) {
	cases := []struct {
		name string
		q    Query
		text string
		id   string
		want bool
	}{
		{name: "語なしは常に真", q: Query{}, text: "anything", id: "x", want: true},
		{name: "非nil空でも真", q: Query{Words: []string{}, NotWords: []string{}}, text: "anything", id: "x", want: true},
		{name: "肯定語_大小無視の部分一致", q: Query{Words: []string{"HELLO"}}, text: "say hello world", id: "x", want: true},
		{name: "肯定語_含まない", q: Query{Words: []string{"bye"}}, text: "say hello world", id: "x", want: false},
		{name: "肯定語_ID前方一致", q: Query{Words: []string{"abc"}}, text: "no", id: "ABCDEF", want: true},
		{name: "肯定語_IDの途中は当たらない", q: Query{Words: []string{"cde"}}, text: "no", id: "ABCDEF", want: false},
		{name: "AND_全語", q: Query{Words: []string{"hello", "world"}, WordsAnd: true}, text: "hello world", id: "x", want: true},
		{name: "AND_片方だけ", q: Query{Words: []string{"hello", "bye"}, WordsAnd: true}, text: "hello world", id: "x", want: false},
		{name: "OR_片方だけで真", q: Query{Words: []string{"hello", "bye"}, WordsAnd: false}, text: "hello world", id: "x", want: true},
		{name: "除外語_含む", q: Query{NotWords: []string{"world"}}, text: "hello world", id: "x", want: false},
		{name: "除外語_IDは見ない", q: Query{NotWords: []string{"abc"}}, text: "hello", id: "abcdef", want: true},
		{name: "空文字の肯定語は無視して素通し", q: Query{Words: []string{""}}, text: "hello", id: "x", want: true},
		{name: "空白だけの除外語は無視", q: Query{NotWords: []string{" ", "　"}}, text: "hello", id: "x", want: true},
		{name: "前後の空白は落として照合", q: Query{Words: []string{" hello "}}, text: "hello", id: "x", want: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.q.MatchText(c.text, c.id); got != c.want {
				t.Errorf("MatchText(%q, %q) with %+v = %v, want %v", c.text, c.id, c.q, got, c.want)
			}
			// Matcher 経由でも同じ結果
			if got := c.q.Matcher().MatchText(c.text, c.id); got != c.want {
				t.Errorf("Matcher().MatchText(%q, %q) with %+v = %v, want %v", c.text, c.id, c.q, got, c.want)
			}
		})
	}
}

// Matcher は元の Query のスライスを書き換えない（query は呼び出しの間で共有されうる）。
func TestQueryMatcherDoesNotMutateQuery(t *testing.T) {
	q := Query{Words: []string{" HELLO "}, NotWords: []string{"BYE"}}
	m := q.Matcher()
	if !m.HasWords() {
		t.Fatalf("語があるのに HasWords が偽")
	}
	if q.Words[0] != " HELLO " || q.NotWords[0] != "BYE" {
		t.Errorf("元の Query を書き換えてはいけない: %+v", q)
	}
	if (Query{}).Matcher().HasWords() {
		t.Errorf("語が無ければ HasWords は偽")
	}
	if (Query{Words: []string{"", " "}}).Matcher().HasWords() {
		t.Errorf("空語だけなら HasWords は偽")
	}
}
