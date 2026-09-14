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
		{Hash: "a1b2c3d4e5f6a7b8c9d0a1b2c3d4e5f6a7b8c9d0", RepName: "repo-beta", AuthorName: "Test Author", Message: "publish to remote\n", CommitterUnix: 3000},
		{Hash: "b1c2d3e4f5a6b7c8d9e0b1c2d3e4f5a6b7c8d9e0", RepName: "repo-beta", AuthorName: "Test Author", Message: "Initial commit", CommitterUnix: 2000},
		{Hash: "c1d2e3f4a5b6c7d8e9f0c1d2e3f4a5b6c7d8e9f0", RepName: "repo-alpha", AuthorName: "Someone Else", Message: "update dependencies\n", CommitterUnix: 1000},
	}
	ids := func(kyous []sdk.Kyou) map[string]bool {
		m := map[string]bool{}
		for _, k := range kyous {
			m[k.ID] = true
		}
		return m
	}

	if got := ids(kyousOfRows(rows, sdk.Query{Words: []string{"remote"}})); len(got) != 1 || !got["a1b2c3d4e5f6a7b8c9d0a1b2c3d4e5f6a7b8c9d0"] {
		t.Errorf("肯定語はメッセージに当たる: got %v", got)
	}
	if got := ids(kyousOfRows(rows, sdk.Query{Words: []string{"REPO-BETA"}})); len(got) != 2 || got["c1d2e3f4a5b6c7d8e9f0c1d2e3f4a5b6c7d8e9f0"] {
		t.Errorf("肯定語はリポジトリ名にも当たる（大小無視）: got %v", got)
	}
	if got := ids(kyousOfRows(rows, sdk.Query{Words: []string{"someone"}})); len(got) != 1 || !got["c1d2e3f4a5b6c7d8e9f0c1d2e3f4a5b6c7d8e9f0"] {
		t.Errorf("肯定語は author 名にも当たる: got %v", got)
	}
	if got := ids(kyousOfRows(rows, sdk.Query{NotWords: []string{"repo-beta"}})); len(got) != 1 || !got["c1d2e3f4a5b6c7d8e9f0c1d2e3f4a5b6c7d8e9f0"] {
		t.Errorf("除外語はリポジトリ名でも消す: got %v", got)
	}
	if got := ids(kyousOfRows(rows, sdk.Query{Words: []string{"b1c2d3e4"}})); len(got) != 1 || !got["b1c2d3e4f5a6b7c8d9e0b1c2d3e4f5a6b7c8d9e0"] {
		t.Errorf("ID（ハッシュ）前方一致: got %v", got)
	}
	if got := ids(kyousOfRows(rows, sdk.Query{Words: []string{"3e4f5a6b"}})); len(got) != 0 {
		t.Errorf("ハッシュの途中の部分一致で当たってはいけない: got %v", got)
	}
	if got := ids(kyousOfRows(rows, sdk.Query{Words: []string{""}})); len(got) != 3 {
		t.Errorf("空語は無視して全件: got %v", got)
	}
	if got := kyousOfRows(rows, sdk.Query{Words: []string{"commit", "remote"}, Limit: 1}); len(got) != 1 {
		t.Errorf("LIMIT は絞った後: got %d件", len(got))
	}
	if got := kyousOfRows(rows, sdk.Query{Words: []string{"commit", "remote"}, WordsAnd: true}); len(got) != 0 {
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
