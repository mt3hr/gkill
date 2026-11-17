package main

import (
	"slices"
	"testing"

	sdk "github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
)

// FindKyous のワード判定は SDK の Query.MatchText（gkill 本体と同じ規則）で、
// 対象は search_text。gkill 本体は再判定しないので、ここが唯一の判定。
func TestKyousOfRows_WordFilter(t *testing.T) {
	rows := []recordRow{
		{ID: "row-0001", Kind: kindWorldTimeIs, Title: "WorldAlpha", SearchText: "WorldAlpha\nwrld_x", StartUnix: 1000, EndUnix: 2000, RelatedUnix: 1000},
		{ID: "row-0002", Kind: kindPlayer, Title: "UserB", SearchText: "UserB\nWorldAlpha", StartUnix: 1000, EndUnix: 2000, RelatedUnix: 1000},
		{ID: "row-0003", Kind: kindEvent, Content: "動画: x", SearchText: "動画: x\nWorldBeta", StartUnix: 3000, RelatedUnix: 3000},
	}
	ids := func(kyous []sdk.Kyou) map[string]bool {
		m := map[string]bool{}
		for _, k := range kyous {
			m[k.ID] = true
		}
		return m
	}

	if got := ids(kyousOfRows(rows, sdk.Query{Words: []string{"worldalpha"}})); !got["row-0001"] || !got["row-0002"] || got["row-0003"] {
		t.Errorf("ワールド名で、その部屋にいた人も当たる: got %v", got)
	}
	if got := ids(kyousOfRows(rows, sdk.Query{NotWords: []string{"worldalpha"}})); len(got) != 1 || !got["row-0003"] {
		t.Errorf("除外語: got %v", got)
	}
	if got := ids(kyousOfRows(rows, sdk.Query{Words: []string{"row-0003"}})); len(got) != 1 || !got["row-0003"] {
		t.Errorf("ID 前方一致: got %v", got)
	}
	if got := ids(kyousOfRows(rows, sdk.Query{Words: []string{"ow-0003"}})); len(got) != 0 {
		t.Errorf("ID の途中の部分一致で当たってはいけない: got %v", got)
	}
	if got := kyousOfRows(rows, sdk.Query{Words: []string{"worldalpha"}, Limit: 1}); len(got) != 1 {
		t.Errorf("LIMIT は絞った後: got %d件", len(got))
	}
}

// 種類ごとに data_type と型別データが合っていること。manifest の data_type で上書きしない（ADR-0317）。
func TestKyouOfSetsTypedDataPerKind(t *testing.T) {
	cases := []struct {
		row      recordRow
		dataType string
	}{
		{recordRow{ID: "a", Kind: kindWorldTimeIs, Title: "WorldAlpha", StartUnix: 1000, EndUnix: 2000, RelatedUnix: 1000, UpdateUnix: 2000, Tags: tagWorld}, dataTypeTimeIs},
		{recordRow{ID: "b", Kind: kindPlayer, Title: "UserB", StartUnix: 1000, EndUnix: 2000, RelatedUnix: 1000, UpdateUnix: 2000, Tags: tagRoom}, dataTypeTimeIs},
		{recordRow{ID: "c", Kind: kindWorldURLog, Title: "WorldAlpha", URL: "https://example.com/", StartUnix: 1000, RelatedUnix: 1000, Tags: tagWorld}, dataTypeURLog},
		{recordRow{ID: "d", Kind: kindEvent, Content: "動画: x", StartUnix: 1000, RelatedUnix: 1000, Tags: tagEvent}, dataTypeKmemo},
	}
	for _, c := range cases {
		kyou := kyouOf(c.row)
		if kyou.DataType != c.dataType || kyou.RepName != repName {
			t.Errorf("%s: data_type=%q rep=%q, want %q", c.row.ID, kyou.DataType, kyou.RepName, c.dataType)
		}
		if !slices.Equal(kyou.Tags, []string{c.row.Tags}) {
			t.Errorf("%s: tags = %v", c.row.ID, kyou.Tags)
		}
		typed := kyou.Typed
		switch c.dataType {
		case dataTypeTimeIs:
			if typed == nil || typed.TimeIs == nil || typed.TimeIs.EndTime == nil || typed.TimeIs.EndTime.Unix() != 2000 || typed.TimeIs.Title != c.row.Title {
				t.Errorf("%s: TimeIs = %+v（終了時刻は必ず埋める）", c.row.ID, typed)
			}
			if !kyou.UpdateTime.Equal(*typed.TimeIs.EndTime) {
				t.Errorf("%s: UpdateTime = %v, want 終了時刻", c.row.ID, kyou.UpdateTime)
			}
		case dataTypeURLog:
			if typed == nil || typed.URLog == nil || typed.URLog.URL != "https://example.com/" {
				t.Errorf("%s: URLog = %+v", c.row.ID, typed)
			}
		case dataTypeKmemo:
			if typed == nil || typed.Kmemo == nil || typed.Kmemo.Content != "動画: x" {
				t.Errorf("%s: Kmemo = %+v", c.row.ID, typed)
			}
		}
	}
}
