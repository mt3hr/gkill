package main

import (
	"testing"

	sdk "github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
)

// gkill の FindQuery.IDs（ADR-0318）が来たら、指定の ID の記録だけを返す。
// nil は指定なし（今までどおり全件）、[] は0件。
func TestKyousOfMessages_IDs(t *testing.T) {
	messages := []cachedMessage{
		{MsgID: "msg-0001", Text: "一つ目", RelatedTimeUnix: 1000},
		{MsgID: "msg-0002", Text: "二つ目", RelatedTimeUnix: 2000},
		{MsgID: "msg-0003", Text: "三つ目", RelatedTimeUnix: 3000},
	}
	if got := kyousOfMessages(messages, sdk.Query{}); len(got) != 3 {
		t.Errorf("IDs が nil: %d件, want 3件", len(got))
	}
	got := kyousOfMessages(messages, sdk.Query{IDs: []string{"msg-0002", "msg-unknown"}})
	if len(got) != 1 || got[0].ID != "msg-0002" {
		t.Errorf("IDs 指定: %+v, want msg-0002 だけ", got)
	}
	if got := kyousOfMessages(messages, sdk.Query{IDs: []string{}}); len(got) != 0 {
		t.Errorf("IDs が []: %d件, want 0件", len(got))
	}
}
