package main

import (
	"testing"

	sdk "github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
)

// FindKyous のワード判定は SDK の Query.MatchText（gkill 本体と同じ規則）で、
// 対象は search_text とスレッド名。gkill 本体は再判定しないので、ここが唯一の判定。
func TestKyousOfRows_WordFilter(t *testing.T) {
	rows := []kyouRow{
		{ID: "r-1", Title: "雑談", SearchText: "本文に Go の話", RelatedUnix: 1000},
		{ID: "r-2", Title: "Go の質問", SearchText: "本文には無い", RelatedUnix: 2000},
		{ID: "r-3", Title: "料理", SearchText: "関係ない話", RelatedUnix: 3000},
	}
	ids := func(kyous []sdk.Kyou) map[string]bool {
		m := map[string]bool{}
		for _, k := range kyous {
			m[k.ID] = true
		}
		return m
	}

	if got := ids(kyousOfRows(rows, sdk.Query{Words: []string{"go"}})); !got["r-1"] || !got["r-2"] || got["r-3"] {
		t.Errorf("肯定語は本文とスレッド名の両方に当たる: got %v", got)
	}
	if got := ids(kyousOfRows(rows, sdk.Query{NotWords: []string{"go"}})); got["r-1"] || got["r-2"] || !got["r-3"] {
		t.Errorf("除外語は本文とスレッド名の両方で消す: got %v", got)
	}
	if got := ids(kyousOfRows(rows, sdk.Query{Words: []string{"r-3"}})); len(got) != 1 || !got["r-3"] {
		t.Errorf("ID 前方一致: got %v", got)
	}
	if got := ids(kyousOfRows(rows, sdk.Query{Words: []string{"-3"}})); len(got) != 0 {
		t.Errorf("ID の途中の部分一致で当たってはいけない: got %v", got)
	}
	if got := kyousOfRows(rows, sdk.Query{Words: []string{"go"}, Limit: 1}); len(got) != 1 {
		t.Errorf("LIMIT は絞った後: got %d件", len(got))
	}
}
