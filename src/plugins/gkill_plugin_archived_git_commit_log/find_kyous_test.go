package main

import (
	"testing"

	sdk "github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
)

// FindKyous のワード判定は SDK の Query.MatchText（gkill 本体と同じ規則）で、
// 対象はコミットメッセージ・リポジトリ名・author 名。gkill 本体は再判定しないので、ここが唯一の判定。
// ID（ハッシュ）は前方一致だけ。
func TestKyousOfRows_WordFilter(t *testing.T) {
	rows := []commitRow{
		{Hash: "2aa9a829e303ad135ca6c32575a8ab678c9eb268", RepName: "racoonboard", AuthorName: "Test Author", Message: "to github\n", CommitterUnix: 3000},
		{Hash: "b66fb9148bb13ba73a1271c1de4e5431ef448acc", RepName: "racoonboard", AuthorName: "Test Author", Message: "Initial commit", CommitterUnix: 2000},
		{Hash: "747bb1c24add3fea53c835be9f8626c7ec97ebed", RepName: "ocha", AuthorName: "Someone Else", Message: "update go mod\n", CommitterUnix: 1000},
	}
	ids := func(kyous []sdk.Kyou) map[string]bool {
		m := map[string]bool{}
		for _, k := range kyous {
			m[k.ID] = true
		}
		return m
	}

	if got := ids(kyousOfRows(rows, sdk.Query{Words: []string{"github"}})); len(got) != 1 || !got["2aa9a829e303ad135ca6c32575a8ab678c9eb268"] {
		t.Errorf("肯定語はメッセージに当たる: got %v", got)
	}
	if got := ids(kyousOfRows(rows, sdk.Query{Words: []string{"RACOONBOARD"}})); len(got) != 2 || got["747bb1c24add3fea53c835be9f8626c7ec97ebed"] {
		t.Errorf("肯定語はリポジトリ名にも当たる（大小無視）: got %v", got)
	}
	if got := ids(kyousOfRows(rows, sdk.Query{Words: []string{"someone"}})); len(got) != 1 || !got["747bb1c24add3fea53c835be9f8626c7ec97ebed"] {
		t.Errorf("肯定語は author 名にも当たる: got %v", got)
	}
	if got := ids(kyousOfRows(rows, sdk.Query{NotWords: []string{"racoonboard"}})); len(got) != 1 || !got["747bb1c24add3fea53c835be9f8626c7ec97ebed"] {
		t.Errorf("除外語はリポジトリ名でも消す: got %v", got)
	}
	if got := ids(kyousOfRows(rows, sdk.Query{Words: []string{"b66fb914"}})); len(got) != 1 || !got["b66fb9148bb13ba73a1271c1de4e5431ef448acc"] {
		t.Errorf("ID（ハッシュ）前方一致: got %v", got)
	}
	if got := ids(kyousOfRows(rows, sdk.Query{Words: []string{"9148bb13"}})); len(got) != 0 {
		t.Errorf("ハッシュの途中の部分一致で当たってはいけない: got %v", got)
	}
	if got := ids(kyousOfRows(rows, sdk.Query{Words: []string{""}})); len(got) != 3 {
		t.Errorf("空語は無視して全件: got %v", got)
	}
	if got := kyousOfRows(rows, sdk.Query{Words: []string{"commit", "github"}, Limit: 1}); len(got) != 1 {
		t.Errorf("LIMIT は絞った後: got %d件", len(got))
	}
	if got := kyousOfRows(rows, sdk.Query{Words: []string{"commit", "github"}, WordsAnd: true}); len(got) != 0 {
		t.Errorf("AND 検索: got %d件", len(got))
	}

	// HasWords は「単語で絞るか」の判定。LIMIT を SQL へ押し込むかを決める
	if (sdk.Query{Words: []string{""}}).Matcher().HasWords() {
		t.Error("空語だけなら HasWords は偽")
	}
	if !(sdk.Query{NotWords: []string{"x"}}).Matcher().HasWords() {
		t.Error("除外語だけでも HasWords は真")
	}
}
