package sdk

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/api/gkill_plugin"
)

// gkill 本体が組んだ PluginQuery（find_query・words_skip_id_match 付き）を JSON で受けると、
// Query の FindQuery と写しの欄（IDs / Reps / IncludeEndTimeIs / WordsSkipIDMatch）に入る（ADR-0318）。
// 本体側の型から JSON を作るので、欄の名前が両側でずれたら落ちる。
func TestRunLoop_FindKyousReceivesFindQuery(t *testing.T) {
	var gotQuery Query
	h := Handler{
		FindKyous: func(_ context.Context, q Query, _ Config) ([]Kyou, error) {
			gotQuery = q
			return nil, nil
		},
	}

	wire, err := json.Marshal(gkill_plugin.PluginQuery{
		Words:            []string{"alpha"},
		WordsSkipIDMatch: true,
		FindQuery: &find.FindQuery{
			Words:            []string{"alpha"},
			IDs:              []string{"kyou-1"},
			Reps:             []string{},
			IncludeEndTimeIs: true,
		},
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var query pluginQuery
	if err := json.Unmarshal(wire, &query); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	runLoopOnce(t, h, pluginRequest{ID: "req-1", Command: "find_kyous", Query: &query})

	if gotQuery.FindQuery == nil {
		t.Fatal("FindQuery = nil, want gkill の検索条件")
	}
	if len(gotQuery.IDs) != 1 || gotQuery.IDs[0] != "kyou-1" {
		t.Errorf("IDs = %v, want [kyou-1]", gotQuery.IDs)
	}
	if gotQuery.Reps == nil || len(gotQuery.Reps) != 0 {
		t.Errorf("Reps = %#v, want []（0件。nil と取り違えない）", gotQuery.Reps)
	}
	if !gotQuery.IncludeEndTimeIs {
		t.Error("IncludeEndTimeIs = false, want true")
	}
	if !gotQuery.WordsSkipIDMatch {
		t.Error("WordsSkipIDMatch = false, want true")
	}
}

// 古い gkill は find_query を送らない。そのときは FindQuery が nil で、写しの欄はゼロ値（IDs・Reps は nil）。
func TestRunLoop_FindKyousWithoutFindQuery(t *testing.T) {
	var gotQuery Query
	h := Handler{
		FindKyous: func(_ context.Context, q Query, _ Config) ([]Kyou, error) {
			gotQuery = q
			return nil, nil
		},
	}
	var query pluginQuery
	if err := json.Unmarshal([]byte(`{"words":["alpha"],"words_and":false,"only_latest_data":true,"limit":0}`), &query); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	runLoopOnce(t, h, pluginRequest{ID: "req-1", Command: "find_kyous", Query: &query})

	if gotQuery.FindQuery != nil || gotQuery.IDs != nil || gotQuery.Reps != nil || gotQuery.IncludeEndTimeIs || gotQuery.WordsSkipIDMatch {
		t.Errorf("Query = %+v, want FindQuery なし・写しの欄はゼロ値", gotQuery)
	}
	if !gotQuery.MatchID("any") || !gotQuery.MatchRep("any") {
		t.Error("IDs・Reps の指定が無いのに MatchID / MatchRep が偽")
	}
}

// MatchID / IDMatcher / MatchRep は nil（指定なし）を常に真、[]（0件）を常に偽にする。
func TestQueryMatchIDAndRep(t *testing.T) {
	cases := []struct {
		name  string
		list  []string
		value string
		want  bool
	}{
		{"nil は指定なし", nil, "a", true},
		{"[] は0件", []string{}, "a", false},
		{"入っている", []string{"a", "b"}, "b", true},
		{"入っていない", []string{"a", "b"}, "c", false},
	}
	for _, c := range cases {
		if got := (Query{IDs: c.list}).MatchID(c.value); got != c.want {
			t.Errorf("%s: MatchID = %v, want %v", c.name, got, c.want)
		}
		if got := (Query{IDs: c.list}).IDMatcher().Match(c.value); got != c.want {
			t.Errorf("%s: IDMatcher = %v, want %v", c.name, got, c.want)
		}
		if got := (Query{Reps: c.list}).MatchRep(c.value); got != c.want {
			t.Errorf("%s: MatchRep = %v, want %v", c.name, got, c.want)
		}
	}
}

// WordsSkipIDMatch が立っていたら、肯定語で ID の前方一致を見ない（本文だけで判定する）。
// 除外語を肯定語として引き直すとき、ID が除外語で始まる記録まで落とさないため。
func TestQueryMatchTextSkipIDMatch(t *testing.T) {
	const id = "abcdefgh-0000"
	if !(Query{Words: []string{"abcdefgh"}}).MatchText("本文", id) {
		t.Fatal("前提: 7文字以上の語は ID の前方一致で当たる")
	}
	if (Query{Words: []string{"abcdefgh"}, WordsSkipIDMatch: true}).MatchText("本文", id) {
		t.Error("WordsSkipIDMatch なのに ID の前方一致で当たった")
	}
	if !(Query{Words: []string{"本文"}, WordsSkipIDMatch: true}).MatchText("本文", id) {
		t.Error("WordsSkipIDMatch でも本文には当たるべき")
	}
}

// GetKyou の既定実装は、FindKyous に IDs で1件だけを頼む。
func TestRunLoop_GetKyouFallbackAsksByID(t *testing.T) {
	var gotIDs []string
	h := Handler{
		FindKyous: func(_ context.Context, q Query, _ Config) ([]Kyou, error) {
			gotIDs = q.IDs
			return []Kyou{{ID: "kyou-1"}, {ID: "kyou-2"}}, nil
		},
	}
	resp := runLoopOnce(t, h, pluginRequest{ID: "req-1", Command: "get_kyou", KyouID: "kyou-2"})
	if len(gotIDs) != 1 || gotIDs[0] != "kyou-2" {
		t.Errorf("FindKyous に渡った IDs = %v, want [kyou-2]", gotIDs)
	}
	if resp.Kyou == nil || resp.Kyou.ID != "kyou-2" {
		t.Errorf("Kyou = %+v, want kyou-2", resp.Kyou)
	}
}
