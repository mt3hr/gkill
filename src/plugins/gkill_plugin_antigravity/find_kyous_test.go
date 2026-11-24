package main

import (
	"testing"

	sdk "github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
)

// FindKyous のワード判定は SDK の Query.MatchText（gkill 本体と同じ規則）で、
// 対象は search_text と会話タイトル・プロジェクト名。
// gkill 本体はプラグインが返した Kyou を再判定しないので、ここが唯一の判定。
func TestKyousOfRows_WordFilter(t *testing.T) {
	rows := []kyouRow{
		{ID: "row-0001", Title: "雑談", Project: "repo-beta", SearchText: "本文に Go の話", RelatedUnix: 1000},
		{ID: "row-0002", Title: "Go の質問", Project: "repo-beta", SearchText: "本文には無い", RelatedUnix: 2000},
		{ID: "row-0003", Title: "料理", Project: "repo-alpha", SearchText: "関係ない話", RelatedUnix: 3000},
	}
	ids := func(kyous []sdk.Kyou) map[string]bool {
		m := map[string]bool{}
		for _, k := range kyous {
			m[k.ID] = true
		}
		return m
	}

	if got := ids(kyousOfRows(rows, sdk.Query{Words: []string{"go"}})); !got["row-0001"] || !got["row-0002"] || got["row-0003"] {
		t.Errorf("肯定語は本文と会話タイトルの両方に当たる: got %v", got)
	}
	if got := ids(kyousOfRows(rows, sdk.Query{Words: []string{"repo-alpha"}})); len(got) != 1 || !got["row-0003"] {
		t.Errorf("プロジェクト名にも当たる: got %v", got)
	}
	if got := ids(kyousOfRows(rows, sdk.Query{NotWords: []string{"go"}})); got["row-0001"] || got["row-0002"] || !got["row-0003"] {
		t.Errorf("除外語は本文と会話タイトルの両方で消す: got %v", got)
	}
	if got := ids(kyousOfRows(rows, sdk.Query{Words: []string{"row-0003"}})); len(got) != 1 || !got["row-0003"] {
		t.Errorf("ID 前方一致: got %v", got)
	}
	if got := ids(kyousOfRows(rows, sdk.Query{Words: []string{"ow-0003"}})); len(got) != 0 {
		t.Errorf("ID の途中の部分一致で当たってはいけない: got %v", got)
	}
	if got := kyousOfRows(rows, sdk.Query{Words: []string{"go"}, Limit: 1}); len(got) != 1 {
		t.Errorf("LIMIT は絞った後: got %d件", len(got))
	}
	if got := kyousOfRows(rows, sdk.Query{Words: []string{""}}); len(got) != 3 {
		t.Errorf("空の語で全件が消えてはいけない: got %d件", len(got))
	}
}

// 返す Kyou の欄は rep 名・data_type・時刻・アプリ名。タグは付けない。
func TestKyouOfFields(t *testing.T) {
	kyou := kyouOf(kyouRow{ID: "row-0001", RelatedUnix: 1000, UpdateUnix: 0})
	if kyou.RepName != repName || kyou.DataType != dataType {
		t.Errorf("rep名/data_type = %q/%q", kyou.RepName, kyou.DataType)
	}
	if kyou.CreateApp != appName || kyou.UpdateApp != appName {
		t.Errorf("アプリ名 = %q/%q", kyou.CreateApp, kyou.UpdateApp)
	}
	if !kyou.UpdateTime.Equal(kyou.RelatedTime) {
		t.Errorf("update_unix が 0 のとき related_time へ倒していない: %v", kyou.UpdateTime)
	}
	if len(kyou.Tags) != 0 {
		t.Errorf("タグを付けている: %v", kyou.Tags)
	}
}
