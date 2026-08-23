package gkill_server_api

// get_kyous_mcp v2（複合カーソル・厳密上限・count_only/group_by・リクエストレベルフィルタ・
// 未知値警告）の回帰テスト。契約: documents/adr/0053-mcp-composite-cursor-strict-limits.md

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/req_res"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
)

// parseMCPCursor / encodeMCPCursor のラウンドトリップと形式の受理・拒否を固定する。
// プラグインIDは任意文字列なので「ID側に :: が含まれる」ケースが必ず要る
// （区切りの探索は左端一致。RFC3339Nanoにコロン2連は現れない）。
func TestParseMCPCursor(t *testing.T) {
	at := time.Date(2026, 8, 19, 12, 34, 56, 789000000, time.Local)

	t.Run("複合形式のラウンドトリップ", func(t *testing.T) {
		for _, id := range []string{"plain-id", "id::with::separators", "3f9e40c1"} {
			encoded := encodeMCPCursor(at, id)
			parsed, err := parseMCPCursor(encoded)
			if err != nil {
				t.Fatalf("parse %q failed: %v", encoded, err)
			}
			if !parsed.hasID || parsed.id != id || !parsed.time.Equal(at) {
				t.Errorf("roundtrip mismatch: %q -> %+v", encoded, parsed)
			}
		}
	})

	t.Run("旧形式(RFC3339)を受理", func(t *testing.T) {
		parsed, err := parseMCPCursor(at.Format(time.RFC3339Nano))
		if err != nil {
			t.Fatalf("parse failed: %v", err)
		}
		if parsed.hasID || !parsed.time.Equal(at) {
			t.Errorf("旧形式の解釈がおかしい: %+v", parsed)
		}
	})

	t.Run("旧形式(日付のみ)を受理", func(t *testing.T) {
		parsed, err := parseMCPCursor("2026-08-19")
		if err != nil {
			t.Fatalf("parse failed: %v", err)
		}
		want := time.Date(2026, 8, 19, 0, 0, 0, 0, time.Local)
		if parsed.hasID || !parsed.time.Equal(want) {
			t.Errorf("日付のみの解釈がおかしい: %+v", parsed)
		}
	})

	t.Run("壊れた文字列はエラー", func(t *testing.T) {
		for _, cursor := range []string{"not-a-time", "not-a-time::id", "::id", "2026/08/19"} {
			if _, err := parseMCPCursor(cursor); err == nil {
				t.Errorf("%q がエラーにならなかった", cursor)
			}
		}
	})
}

// count_only は件数だけを返す（kyous無し・total_count有り・remaining=0）。
func TestHandleGetKyousMCP_CountOnly(t *testing.T) {
	tsURL, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()

	sessionID := loginAndGetSession(t, tsURL, gkillAPI, "admin", mcpTestPasswordHash)
	for i := range 3 {
		addTestKmemo(t, tsURL, sessionID, fmt.Sprintf("count_only用メモ%d", i))
	}

	res := getKyousMCP(t, tsURL, sessionID, map[string]any{}, map[string]any{"count_only": true})
	if res.TotalCount == nil {
		t.Fatal("count_onlyなのにtotal_countが無い")
	}
	if *res.TotalCount < 3 {
		t.Errorf("total_count = %d, want >= 3", *res.TotalCount)
	}
	if len(res.Kyous) != 0 {
		t.Errorf("count_onlyなのにkyousが%d件返っている", len(res.Kyous))
	}
	if res.HasMore || res.RemainingCount != 0 {
		t.Errorf("count_onlyの応答はhas_more=false/remaining=0のはず: %+v", res)
	}

	// limit:1 のデータ応答の total_count と一致する(検算経路の同値性)
	one := getKyousMCP(t, tsURL, sessionID, map[string]any{}, map[string]any{"limit": 1})
	if one.TotalCount == nil || *one.TotalCount != *res.TotalCount {
		t.Errorf("count_onlyとlimit:1のtotal_countが食い違う: %v vs %v", res.TotalCount, one.TotalCount)
	}
}

// count_only / group_by と cursor の併用はエラー（黙って片方を無視しない）。未知のgroup_by値もエラー。
func TestHandleGetKyousMCP_CountOnlyAndGroupByRejectCursor(t *testing.T) {
	tsURL, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()

	sessionID := loginAndGetSession(t, tsURL, gkillAPI, "admin", mcpTestPasswordHash)

	postExpectError := func(extra map[string]any, what string) {
		t.Helper()
		body := map[string]any{"session_id": sessionID, "locale_name": "en", "query": map[string]any{}}
		for key, value := range extra {
			body[key] = value
		}
		resp := postJSON(t, tsURL+"/api/get_kyous_mcp", body)
		defer resp.Body.Close()
		var mcpResp req_res.GetKyousMCPResponse
		if err := json.NewDecoder(resp.Body).Decode(&mcpResp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(mcpResp.Errors) == 0 {
			t.Errorf("%s がエラーにならなかった", what)
		}
	}

	nowCursor := time.Now().Format(time.RFC3339Nano)
	postExpectError(map[string]any{"count_only": true, "cursor": nowCursor}, "count_only+cursor")
	postExpectError(map[string]any{"group_by": "month", "cursor": nowCursor}, "group_by+cursor")
	postExpectError(map[string]any{"group_by": "unknown_axis"}, "未知のgroup_by値")
}

// group_by の集計: バケットの合計が total_count と一致し、時刻系キーは昇順で返る。
func TestHandleGetKyousMCP_GroupByDayAndDataType(t *testing.T) {
	tsURL, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()

	sessionID := loginAndGetSession(t, tsURL, gkillAPI, "admin", mcpTestPasswordHash)

	day1 := time.Date(2026, 8, 10, 12, 0, 0, 0, time.Local)
	day2 := time.Date(2026, 8, 11, 12, 0, 0, 0, time.Local)
	addTestKmemoWithRelatedTime(t, tsURL, sessionID, "8/10のメモ1", day1)
	addTestKmemoWithRelatedTime(t, tsURL, sessionID, "8/10のメモ2", day1.Add(time.Hour))
	addTestKmemoWithRelatedTime(t, tsURL, sessionID, "8/11のメモ", day2)

	query := map[string]any{
		"calendar_start_date": "2026-08-10T00:00:00+09:00",
		"calendar_end_date":   "2026-08-11T23:59:59+09:00",
	}

	res := getKyousMCP(t, tsURL, sessionID, query, map[string]any{"group_by": "day"})
	if res.TotalCount == nil {
		t.Fatal("group_byなのにtotal_countが無い")
	}
	sum := 0
	byKey := map[string]int{}
	for _, bucket := range res.Buckets {
		sum += bucket.Count
		byKey[bucket.Key] = bucket.Count
	}
	if sum != *res.TotalCount {
		t.Errorf("Σbuckets(%d) != total_count(%d)", sum, *res.TotalCount)
	}
	if byKey["2026-08-10"] != 2 || byKey["2026-08-11"] != 1 {
		t.Errorf("日別バケットが想定と違う: %v", byKey)
	}
	if len(res.Kyous) != 0 {
		t.Errorf("group_byなのにkyousが返っている: %d件", len(res.Kyous))
	}
	for i := 1; i < len(res.Buckets); i++ {
		if res.Buckets[i-1].Key > res.Buckets[i].Key {
			t.Errorf("dayバケットが昇順でない: %v", res.Buckets)
		}
	}

	byType := getKyousMCP(t, tsURL, sessionID, query, map[string]any{"group_by": "data_type"})
	sum = 0
	for _, bucket := range byType.Buckets {
		sum += bucket.Count
	}
	if byType.TotalCount == nil || sum != *byType.TotalCount {
		t.Errorf("data_typeバケットの合計が総数と一致しない: %d vs %v", sum, byType.TotalCount)
	}
}

// data_types はDTOのdata_type文字列の許可リストで、未知値は警告になる（エラーにも0件黙殺にもしない）。
func TestHandleGetKyousMCP_DataTypesFilterAndUnknownWarns(t *testing.T) {
	tsURL, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()

	sessionID := loginAndGetSession(t, tsURL, gkillAPI, "admin", mcpTestPasswordHash)
	addTestKmemo(t, tsURL, sessionID, "data_types用メモ")

	res := getKyousMCP(t, tsURL, sessionID, map[string]any{}, map[string]any{"data_types": []string{"kmemo", "typo_type"}})
	if len(res.Kyous) == 0 {
		t.Fatal("kmemoが返っていない")
	}
	for _, kyou := range res.Kyous {
		if kyou.DataType != "kmemo" {
			t.Errorf("data_types:[kmemo]なのに %q が混ざっている", kyou.DataType)
		}
	}
	foundWarn := false
	for _, warning := range res.Warnings {
		if strings.Contains(warning, "typo_type") {
			foundWarn = true
		}
	}
	if !foundWarn {
		t.Errorf("未知のdata_type値が警告されていない: %v", res.Warnings)
	}
}

// 未知のフィルタ値(rep_types / tags)の警告。綴り違いが黙って0件になる監査S7への防御。
func TestHandleGetKyousMCP_UnknownFilterValueWarns(t *testing.T) {
	tsURL, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()

	sessionID := loginAndGetSession(t, tsURL, gkillAPI, "admin", mcpTestPasswordHash)
	kmemoID := addTestKmemo(t, tsURL, sessionID, "警告用メモ")
	addTestTagTo(t, tsURL, sessionID, kmemoID, "ぢ")

	res := getKyousMCP(t, tsURL, sessionID, map[string]any{
		"rep_types": []string{"Kmemo"},        // 正準値はkmemo(小文字)
		"tags":      []string{"ち", "no tags"}, // 「ぢ」の綴り違い。no tagsは正当な仮想タグ
	}, map[string]any{"count_only": true})

	for _, want := range []string{"Kmemo", "ち"} {
		found := false
		for _, warning := range res.Warnings {
			if strings.Contains(warning, want) {
				found = true
			}
		}
		if !found {
			t.Errorf("未知値 %q が警告されていない: %v", want, res.Warnings)
		}
	}
	for _, warning := range res.Warnings {
		if strings.Contains(warning, "no tags") {
			t.Errorf("正当な仮想タグ no tags が警告されている: %v", res.Warnings)
		}
	}
}

// num_min / num_max は kc.num_value / nlog.amount / lantana.mood に効き、
// 数値を持たない種別は結果から外れる。範囲は両端を含む。
func TestHandleGetKyousMCP_NumFilter(t *testing.T) {
	tsURL, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()

	sessionID := loginAndGetSession(t, tsURL, gkillAPI, "admin", mcpTestPasswordHash)

	now := time.Now().Truncate(time.Second)
	addTestKmemo(t, tsURL, sessionID, "数値を持たないメモ")
	addKC := func(id, title string, num json.Number) {
		resp := postJSON(t, tsURL+"/api/add_kc", map[string]any{
			"session_id": sessionID, "locale_name": "en",
			"kc": map[string]any{
				"id": id, "title": title, "num_value": num,
				"related_time": now.Format(time.RFC3339), "data_type": "kc",
				"create_time": now.Format(time.RFC3339), "create_app": "test", "create_user": "admin", "create_device": "test",
				"update_time": now.Format(time.RFC3339), "update_app": "test", "update_user": "admin", "update_device": "test",
			},
		})
		defer resp.Body.Close()
		var addResp map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&addResp); err != nil {
			t.Fatalf("decode add kc: %v", err)
		}
		if errs, ok := addResp["errors"].([]any); ok && len(errs) != 0 {
			t.Fatalf("add kc errors: %v", errs)
		}
	}
	lowID := GenerateNewID()
	highID := GenerateNewID()
	addKC(lowID, "低い値", json.Number("5"))
	addKC(highID, "高い値", json.Number("15"))

	res := getKyousMCP(t, tsURL, sessionID, map[string]any{}, map[string]any{"num_min": 10})
	ids := map[string]bool{}
	for _, kyou := range res.Kyous {
		ids[kyou.ID] = true
	}
	if !ids[highID] {
		t.Error("num_min=10でnum_value=15が返っていない")
	}
	if ids[lowID] {
		t.Error("num_min=10なのにnum_value=5が返っている")
	}
	for _, kyou := range res.Kyous {
		if kyou.DataType == "kmemo" {
			t.Error("数値フィルタ有効時にkmemoが混ざっている(数値を持たない種別は外れるはず)")
		}
	}

	both := getKyousMCP(t, tsURL, sessionID, map[string]any{}, map[string]any{"num_min": 5, "num_max": 15})
	ids = map[string]bool{}
	for _, kyou := range both.Kyous {
		ids[kyou.ID] = true
	}
	if !ids[lowID] || !ids[highID] {
		t.Errorf("num_min=5/num_max=15(両端含む)で両方返るはず: %v", ids)
	}
}

// git payload: addition/deletion の0が消えず、commit_hash が入る（omitempty除去の回帰）。
func TestHandleGetKyousMCP_GitPayloadZeroDiffAndHash(t *testing.T) {
	dto := req_res.GitPayloadMCPDTO{
		Kind:          "git_commit_log",
		CommitHash:    "3f9e40c1",
		CommitMessage: "empty diff commit",
		Addition:      0,
		Deletion:      0,
	}
	encoded, err := json.Marshal(dto)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, key := range []string{"addition", "deletion", "commit_hash"} {
		if _, ok := decoded[key]; !ok {
			t.Errorf("%q がJSONから消えている(omitempty除去の回帰): %s", key, encoded)
		}
	}
}

// idf payload: is_zip がfalseでも常に出力され、file_size は未要求なら出ない。
func TestHandleGetKyousMCP_IDFIsZipAlwaysPresent(t *testing.T) {
	dto := req_res.IDFPayloadMCPDTO{Kind: "idf", FileName: "a.png", IsImage: true}
	encoded, err := json.Marshal(dto)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := decoded["is_zip"]; !ok {
		t.Errorf("is_zip がJSONから消えている(omitempty除去の回帰): %s", encoded)
	}
	if _, ok := decoded["file_size"]; ok {
		t.Errorf("file_size は未要求なら出ないはず: %s", encoded)
	}
}

// remaining_count / total_count のページング健全性:
// 1ページ目 returned+remaining=total、cursorページに total_count 無し、最終ページ remaining=0。
func TestHandleGetKyousMCP_RemainingCountSemantics(t *testing.T) {
	tsURL, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()

	sessionID := loginAndGetSession(t, tsURL, gkillAPI, "admin", mcpTestPasswordHash)

	base := time.Now().Truncate(time.Second).Add(-1 * time.Hour)
	for i := range 5 {
		addTestKmemoWithRelatedTime(t, tsURL, sessionID, fmt.Sprintf("remaining用メモ%d", i), base.Add(time.Duration(i)*time.Minute))
	}

	first := getKyousMCP(t, tsURL, sessionID, map[string]any{}, map[string]any{"limit": 2})
	if first.TotalCount == nil {
		t.Fatal("1ページ目にtotal_countが無い")
	}
	if first.ReturnedCount+first.RemainingCount != *first.TotalCount {
		t.Errorf("returned(%d)+remaining(%d) != total(%d)", first.ReturnedCount, first.RemainingCount, *first.TotalCount)
	}
	if !first.HasMore || first.NextCursor == "" {
		t.Fatal("2ページ目があるはず")
	}

	second := getKyousMCP(t, tsURL, sessionID, map[string]any{}, map[string]any{"limit": 1000, "cursor": first.NextCursor})
	if second.TotalCount != nil {
		t.Errorf("cursorページにtotal_countが入っている(v2ではcursor無し応答のみ): %d", *second.TotalCount)
	}
	if second.RemainingCount != 0 || second.HasMore {
		t.Errorf("最終ページはremaining=0/has_more=falseのはず: remaining=%d has_more=%v", second.RemainingCount, second.HasMore)
	}
	if first.ReturnedCount+second.ReturnedCount != *first.TotalCount {
		t.Errorf("2ページの合計(%d)が総数(%d)と一致しない", first.ReturnedCount+second.ReturnedCount, *first.TotalCount)
	}
}

// 2026-08-24 の再監査 事象7: 「MCP で書いた記録」だけを絞る手段が無かった。
// create_app は全レコードに入っているのに、引く口だけが無かった。
func TestApplyMCPCreateAppsFilter(t *testing.T) {
	kyous := []reps.Kyou{
		{ID: "a", CreateApp: "gkill_mcp_readwrite", UpdateApp: "gkill_mcp_readwrite"},
		{ID: "b", CreateApp: "gkill", UpdateApp: "gkill_kftl"},
		{ID: "c", CreateApp: "gkill_kftl", UpdateApp: "gkill_kftl"},
	}

	// nil は未使用（絞らない）
	if got := applyMCPCreateAppsFilter(append([]reps.Kyou(nil), kyous...), nil); len(got) != 3 {
		t.Errorf("nil は絞らないはず: got %d件", len(got))
	}

	// 非nil空は「明示的な0件」
	if got := applyMCPCreateAppsFilter(append([]reps.Kyou(nil), kyous...), []string{}); len(got) != 0 {
		t.Errorf("[] は0件のはず: got %d件", len(got))
	}

	got := applyMCPCreateAppsFilter(append([]reps.Kyou(nil), kyous...), []string{"gkill_mcp_readwrite", "gkill_kftl"})
	if len(got) != 2 || got[0].ID != "a" || got[1].ID != "c" {
		t.Errorf("許可リストで絞れていない: %+v", got)
	}
}

func TestApplyMCPUpdateAppsFilter(t *testing.T) {
	// create ではなく「最後に更新したアプリ」で絞る
	kyous := []reps.Kyou{
		{ID: "a", CreateApp: "gkill", UpdateApp: "gkill_mcp_readwrite"},
		{ID: "b", CreateApp: "gkill_mcp_readwrite", UpdateApp: "gkill"},
	}
	got := applyMCPUpdateAppsFilter(append([]reps.Kyou(nil), kyous...), []string{"gkill_mcp_readwrite"})
	if len(got) != 1 || got[0].ID != "a" {
		t.Errorf("update_app で絞れていない: %+v", got)
	}
}
