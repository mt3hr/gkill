package main

import (
	"testing"

	sdk "github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
)

// FindKyous のワード判定は SDK の Query.MatchText（gkill 本体と同じ規則）で、
// 対象は発言本文・会話タイトル・添付名。gkill 本体は再判定しないので、ここが唯一の判定。
func TestKyousOfMessages_WordFilter(t *testing.T) {
	msgs := []cachedMessage{
		{MsgID: "msg-body", ConvID: "c1", Text: "本文に Go の話", ConvTitle: "雑談", RelatedTimeUnix: 1000, CreateTimeUnix: 1000},
		{MsgID: "msg-title", ConvID: "c2", Text: "本文には無い", ConvTitle: "Go の質問", RelatedTimeUnix: 2000, CreateTimeUnix: 2000},
		{MsgID: "msg-attach", ConvID: "c4", Text: "添付つき", ConvTitle: "雑談", Attachments: "go_report.wav", RelatedTimeUnix: 4000, CreateTimeUnix: 4000},
		{MsgID: "msg-none", ConvID: "c3", Text: "関係ない話", ConvTitle: "料理", RelatedTimeUnix: 3000, CreateTimeUnix: 3000},
	}
	ids := func(kyous []sdk.Kyou) map[string]bool {
		m := map[string]bool{}
		for _, k := range kyous {
			m[k.ID] = true
		}
		return m
	}

	t.Run("肯定語は本文・会話タイトル・添付名に当たる（大小無視）", func(t *testing.T) {
		got := ids(kyousOfMessages(msgs, sdk.Query{Words: []string{"go"}}))
		if !got["msg-body"] || !got["msg-title"] || !got["msg-attach"] || got["msg-none"] {
			t.Errorf("got %v", got)
		}
	})
	t.Run("除外語は同じ対象で消す", func(t *testing.T) {
		got := ids(kyousOfMessages(msgs, sdk.Query{NotWords: []string{"go"}}))
		if got["msg-body"] || got["msg-title"] || got["msg-attach"] || !got["msg-none"] {
			t.Errorf("got %v", got)
		}
	})
	t.Run("AND は全語", func(t *testing.T) {
		got := ids(kyousOfMessages(msgs, sdk.Query{Words: []string{"go", "質問"}, WordsAnd: true}))
		if got["msg-body"] || !got["msg-title"] || got["msg-none"] {
			t.Errorf("got %v", got)
		}
	})
	t.Run("ID は前方一致", func(t *testing.T) {
		got := ids(kyousOfMessages(msgs, sdk.Query{Words: []string{"msg-none"}}))
		if len(got) != 1 || !got["msg-none"] {
			t.Errorf("got %v", got)
		}
		if got := ids(kyousOfMessages(msgs, sdk.Query{Words: []string{"g-none"}})); len(got) != 0 {
			t.Errorf("ID の途中の部分一致で当たってはいけない: %v", got)
		}
	})
	t.Run("語なしは全件", func(t *testing.T) {
		if got := kyousOfMessages(msgs, sdk.Query{Words: []string{}, NotWords: []string{}}); len(got) != 4 {
			t.Errorf("got %d件", len(got))
		}
	})
	t.Run("LIMIT は絞った後に掛かる", func(t *testing.T) {
		got := ids(kyousOfMessages(msgs, sdk.Query{Words: []string{"go"}, Limit: 1}))
		if len(got) != 1 {
			t.Errorf("got %v", got)
		}
	})
}

// Kyou の欄は rep 名・data_type・時刻・作成アプリが manifest と揃っていること。
func TestKyousOfMessages_KyouFields(t *testing.T) {
	got := kyousOfMessages([]cachedMessage{
		{MsgID: "msg-1", ConvID: "c1", Sender: senderModel, Text: "こたえ", RelatedTimeUnix: 1000, CreateTimeUnix: 1000},
	}, sdk.Query{})
	if len(got) != 1 {
		t.Fatalf("got %d件", len(got))
	}
	k := got[0]
	if k.RepName != repName || k.DataType != dataType {
		t.Errorf("RepName = %q / DataType = %q, want %q / %q", k.RepName, k.DataType, repName, dataType)
	}
	if k.CreateApp != appName || k.UpdateApp != appName {
		t.Errorf("CreateApp = %q / UpdateApp = %q, want %q", k.CreateApp, k.UpdateApp, appName)
	}
	if k.RelatedTime.Unix() != 1000 || k.UpdateTime != k.CreateTime {
		t.Errorf("時刻が合っていない: %+v", k)
	}
}
