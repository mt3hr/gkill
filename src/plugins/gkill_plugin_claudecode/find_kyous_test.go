package main

import (
	"testing"

	sdk "github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
)

// FindKyous のワード判定は SDK の Query.MatchText（gkill 本体と同じ規則）で、
// 対象は search_text。gkill 本体は再判定しないので、ここが唯一の判定。
func TestKyousOfMessages_WordFilter(t *testing.T) {
	messages := []messageSummary{
		{MessageID: "msg-0001", SearchText: "本文に Go の話 project=gkill", RelatedTimeUnix: 1000},
		{MessageID: "msg-0002", SearchText: "関係ない話", RelatedTimeUnix: 2000},
		{MessageID: "msg-0003", SearchText: "go と rust", RelatedTimeUnix: 3000},
	}
	ids := func(kyous []sdk.Kyou) map[string]bool {
		m := map[string]bool{}
		for _, k := range kyous {
			m[k.ID] = true
		}
		return m
	}

	if got := ids(kyousOfMessages(messages, sdk.Query{Words: []string{"GO"}})); !got["msg-0001"] || !got["msg-0003"] || got["msg-0002"] {
		t.Errorf("肯定語（大小無視）: got %v", got)
	}
	if got := ids(kyousOfMessages(messages, sdk.Query{NotWords: []string{"go"}})); got["msg-0001"] || got["msg-0003"] || !got["msg-0002"] {
		t.Errorf("除外語: got %v", got)
	}
	if got := ids(kyousOfMessages(messages, sdk.Query{Words: []string{"go", "rust"}, WordsAnd: true})); len(got) != 1 || !got["msg-0003"] {
		t.Errorf("AND: got %v", got)
	}
	if got := ids(kyousOfMessages(messages, sdk.Query{Words: []string{"msg-0002"}})); len(got) != 1 || !got["msg-0002"] {
		t.Errorf("ID 前方一致: got %v", got)
	}
	if got := ids(kyousOfMessages(messages, sdk.Query{Words: []string{"sg-0002"}})); len(got) != 0 {
		t.Errorf("ID の途中の部分一致で当たってはいけない: got %v", got)
	}
	if got := kyousOfMessages(messages, sdk.Query{}); len(got) != 3 {
		t.Errorf("語なしは全件: got %d件", len(got))
	}
	if got := kyousOfMessages(messages, sdk.Query{Words: []string{"go"}, Limit: 1}); len(got) != 1 {
		t.Errorf("LIMIT は絞った後: got %d件", len(got))
	}
}
