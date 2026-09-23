package main

import (
	"testing"

	sdk "github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
)

// FindKyous のワード判定は SDK の Query.MatchText（gkill 本体と同じ規則）で、
// 対象は search_text（指標名・キー・単位・数値・デバイス・日付）。gkill 本体は再判定しないので、ここが唯一の判定。
// 単語で絞るときは SQL の LIMIT を押し込まず、絞った後に切る（絞る前に切ると取りこぼす）。
func TestKyousOfMetrics_WordFilter(t *testing.T) {
	metrics := []dailyMetric{
		{KyouID: "day-0001", Title: "歩数", NumValue: "8000", SearchText: "歩数 steps 8000 歩 Pixel 2026-08-01 Fitbit", RelatedUnix: 1000},
		{KyouID: "day-0002", Title: "体重", NumValue: "65.4", SearchText: "体重 weight 65.4 kg 2026-08-01 Fitbit", RelatedUnix: 2000},
		{KyouID: "day-0003", Title: "歩数", NumValue: "9000", SearchText: "歩数 steps 9000 歩 Pixel 2026-08-02 Fitbit", RelatedUnix: 3000},
	}
	ids := func(kyous []sdk.Kyou) map[string]bool {
		m := map[string]bool{}
		for _, k := range kyous {
			m[k.ID] = true
		}
		return m
	}

	if got := ids(kyousOfMetrics(metrics, sdk.Query{Words: []string{"歩数"}})); !got["day-0001"] || !got["day-0003"] || got["day-0002"] {
		t.Errorf("指標名で当たる: got %v", got)
	}
	if got := ids(kyousOfMetrics(metrics, sdk.Query{Words: []string{"65.4"}})); len(got) != 1 || !got["day-0002"] {
		t.Errorf("数値で当たる: got %v", got)
	}
	if got := ids(kyousOfMetrics(metrics, sdk.Query{NotWords: []string{"pixel"}})); got["day-0001"] || got["day-0003"] || !got["day-0002"] {
		t.Errorf("除外語（大小無視）: got %v", got)
	}
	if got := ids(kyousOfMetrics(metrics, sdk.Query{Words: []string{"day-0002"}})); len(got) != 1 || !got["day-0002"] {
		t.Errorf("ID 前方一致: got %v", got)
	}
	// 空文字の語は無視する（以前は Words=[""] + OR で全件が消えていた）
	if got := kyousOfMetrics(metrics, sdk.Query{Words: []string{""}}); len(got) != 3 {
		t.Errorf("空文字の語は素通し: got %d件", len(got))
	}
	// LIMIT は絞った後に掛かる: 歩数の2件のうち1件
	if got := ids(kyousOfMetrics(metrics, sdk.Query{Words: []string{"歩数"}, Limit: 1})); len(got) != 1 || got["day-0002"] {
		t.Errorf("LIMIT は絞った後: got %v", got)
	}
}

// 単語で絞るときは SDK の Matcher.HasWords が真になり、SQL の LIMIT を押し込まない判断に使う。
func TestFindKyousLimitIsNotPushedDownWithWords(t *testing.T) {
	if !(sdk.Query{Words: []string{"歩数"}, Limit: 1}).Matcher().HasWords() {
		t.Errorf("語があれば HasWords は真（LIMIT を SQL へ押し込まない）")
	}
	if (sdk.Query{Words: []string{""}, Limit: 1}).Matcher().HasWords() {
		t.Errorf("空文字の語だけなら HasWords は偽（LIMIT を SQL へ押し込んでよい）")
	}
}
