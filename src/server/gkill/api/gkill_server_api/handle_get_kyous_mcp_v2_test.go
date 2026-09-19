package gkill_server_api

// get_kyous_mcp v2（複合カーソル・厳密上限・count_only/group_by・リクエストレベルフィルタ・
// 未知値警告）の回帰テスト。契約: documents/adr/0604-mcp-composite-cursor-strict-limits.md

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/api/gkill_plugin"
	"github.com/mt3hr/gkill/src/server/gkill/api/req_res"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_options"
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

// count_only / group_by と cursor の併用、count_only と group_by の併用はエラー（黙って片方を無視しない）。未知のgroup_by値もエラー。
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
	// count_only + group_by も同じ扱い。以前は count_only の早期 return が group_by を黙って捨て、
	// buckets の無い応答が「集計できた」顔で返っていた（2026-09-18 の実利用報告）。
	postExpectError(map[string]any{"count_only": true, "group_by": "data_type"}, "count_only+group_by")
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

// git payload: addition/deletion の0が消えず（omitempty除去の回帰）、commit_hash は持たない
// （Kyou の id がハッシュそのもの。毎件 40 桁の二重持ちを落とした。ADR-0629）。
func TestHandleGetKyousMCP_GitPayloadZeroDiffAndHash(t *testing.T) {
	dto := req_res.GitPayloadMCPDTO{
		Kind:          "git_commit_log",
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
	for _, key := range []string{"addition", "deletion"} {
		if _, ok := decoded[key]; !ok {
			t.Errorf("%q がJSONから消えている(omitempty除去の回帰): %s", key, encoded)
		}
	}
	if _, ok := decoded["commit_hash"]; ok {
		t.Errorf("commit_hash が復活している（Kyou の id と同値の二重持ち）: %s", encoded)
	}
}

// 未知の mi_board_name は警告で名指しする（tags / reps / data_types と同じ。以前は無警告で 0 件だった）。
func TestHandleGetKyousMCP_UnknownMiBoardNameWarns(t *testing.T) {
	tsURL, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()
	sessionID := loginAndGetSession(t, tsURL, gkillAPI, "admin", mcpTestPasswordHash)
	addTestMiForMCP(t, tsURL, sessionID, "板名警告のタスク", "realboard", nil)

	unknown := getKyousMCP(t, tsURL, sessionID, map[string]any{"for_mi": true, "include_create_mi": true, "mi_board_name": "no_such_board"}, nil)
	if !slices.ContainsFunc(unknown.Warnings, func(w string) bool { return strings.Contains(w, `unknown mi_board_name "no_such_board"`) }) {
		t.Errorf("未知の板名が警告されていない: %v", unknown.Warnings)
	}
	known := getKyousMCP(t, tsURL, sessionID, map[string]any{"for_mi": true, "include_create_mi": true, "mi_board_name": "realboard"}, nil)
	if slices.ContainsFunc(known.Warnings, func(w string) bool { return strings.Contains(w, "unknown mi_board_name") }) {
		t.Errorf("実在する板名に警告が出ている: %v", known.Warnings)
	}
}

// 地図条件の3値が揃わないときは警告で名指しする（欠けると地図条件ごと黙って無視される）。
func TestPartialMapFilterWarning(t *testing.T) {
	lat, lng, radius := 35.0, 135.0, 500.0
	if got := partialMapFilterWarning(&find.FindQuery{}); got != "" {
		t.Errorf("地図条件なしで警告が出ている: %q", got)
	}
	if got := partialMapFilterWarning(&find.FindQuery{MapLatitude: &lat, MapLongitude: &lng, MapRadius: &radius}); got != "" {
		t.Errorf("3値揃いで警告が出ている: %q", got)
	}
	got := partialMapFilterWarning(&find.FindQuery{MapLatitude: &lat})
	for _, want := range []string{"query.map_latitude is set", "map_longitude / map_radius is missing", "meters"} {
		if !strings.Contains(got, want) {
			t.Errorf("警告に %q が無い: %q", want, got)
		}
	}
}

// group_by の week_of_day / hour は 0 件のバケットも並べる（定義域が有限なので省略しない）。
func TestBucketizeMCPKyous_TimeKeyedBucketsAreComplete(t *testing.T) {
	kyous := []reps.Kyou{{ID: "a", RelatedTime: time.Date(2026, 9, 20, 9, 0, 0, 0, time.Local)}} // 日曜 9 時
	week, _, err := bucketizeMCPKyous(context.Background(), nil, kyous, "week_of_day")
	if err != nil {
		t.Fatalf("week_of_day: %v", err)
	}
	if len(week) != 7 || week[0].Key != "sunday" || week[0].Count != 1 || week[6].Key != "saturday" || week[6].Count != 0 {
		t.Errorf("week_of_day のバケット = %+v, want 日〜土の7つ（0件も並ぶ）", week)
	}
	hour, _, err := bucketizeMCPKyous(context.Background(), nil, kyous, "hour")
	if err != nil {
		t.Fatalf("hour: %v", err)
	}
	if len(hour) != 24 || hour[0].Key != "00" || hour[9].Key != "09" || hour[9].Count != 1 || hour[23].Key != "23" {
		t.Errorf("hour のバケット = %+v, want 00〜23 の24個（0件も並ぶ）", hour)
	}
}

// include_attached_ids を立てたときだけ tag_entities / text_entities が組まれる（既定は tags[] だけ）。
func TestHandleGetKyousMCP_AttachedIDsAreOptIn(t *testing.T) {
	tsURL, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()
	sessionID := loginAndGetSession(t, tsURL, gkillAPI, "admin", mcpTestPasswordHash)
	kmemoID := addTestKmemo(t, tsURL, sessionID, "付随IDのメモ")
	addTestTagTo(t, tsURL, sessionID, kmemoID, "attachedidtag")

	plain := getKyousMCP(t, tsURL, sessionID, map[string]any{"ids": []string{kmemoID}}, nil)
	if len(plain.Kyous) != 1 || len(plain.Kyous[0].Tags) != 1 || plain.Kyous[0].TagEntities != nil {
		t.Errorf("既定: tags=%v tag_entities=%v, want tags 1件 / tag_entities 無し", plain.Kyous[0].Tags, plain.Kyous[0].TagEntities)
	}
	withIDs := getKyousMCP(t, tsURL, sessionID, map[string]any{"ids": []string{kmemoID}}, map[string]any{"include_attached_ids": true})
	if len(withIDs.Kyous) != 1 || len(withIDs.Kyous[0].TagEntities) != 1 || withIDs.Kyous[0].TagEntities[0].ID == "" {
		t.Errorf("include_attached_ids: tag_entities=%v, want id 付き1件", withIDs.Kyous[0].TagEntities)
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

// Kyouを1件も出さないプラグイン（GPSログ専用など）の rep名 / data_type を渡したときは、
// 汎用の「綴りを確かめろ」ではなく**読む先を名指しする**ことを固定する。
//
// 2026-08-24 の実利用報告の中心がこれ。manifest の rep_name / data_type は
// プラグイン一覧に出ているので綴りは合っており、汎用文では直しようが無かった。
// さらに data_type 側は既知集合に無条件で入っていたため、**警告すら出ずに必ず0件**だった。
func TestHandleGetKyousMCP_NonKyouPluginValuesGetNamedWarning(t *testing.T) {
	tsURL, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()

	t.Setenv("GKILL_HOME", gkill_options.GkillHomeDir)

	writePluginManifestForTest(t, "admin", "warn_gpslog_plugin", map[string]any{
		"protocol_version": "1",
		"name":             "warn_gpslog_plugin",
		"version":          "1.0.0",
		"description":      "gps only",
		"data_type":        "warn_gpslog_visit",
		"rep_name":         "WarnGPSLogRep",
		"executable":       "no_such_plugin_binary",
		"provides":         []string{"gpslog"},
		"emits_kyou":       false,
	})

	sessionID := loginAndGetSession(t, tsURL, gkillAPI, "admin", mcpTestPasswordHash)

	res := getKyousMCP(t, tsURL, sessionID, map[string]any{
		"reps": []string{"WarnGPSLogRep"},
	}, map[string]any{"data_types": []string{"warn_gpslog_visit"}})

	// rep名・data_type の両方に、そのプラグインがKyouを出さないことと
	// 読む先（get_gps_log）が出ていること
	for _, want := range []string{"WarnGPSLogRep", "warn_gpslog_visit"} {
		found := false
		for _, warning := range res.Warnings {
			if strings.Contains(warning, want) && strings.Contains(warning, "emits no kyou") {
				found = true
			}
		}
		if !found {
			t.Errorf("%q に対する名指しの警告が無い: %v", want, res.Warnings)
		}
	}
	foundRoute := false
	for _, warning := range res.Warnings {
		if strings.Contains(warning, "get_gps_log") {
			foundRoute = true
		}
	}
	if !foundRoute {
		t.Errorf("読む先(get_gps_log)が案内されていない: %v", res.Warnings)
	}
}

// data_types に Mi の射影名を渡したのに for_mi を立てていないときは案内する。
//
// Mi の5射影は for_mi を立てたときだけ残る。立てないと代表1件へ潰され、
// _start 優先→DataType辞書昇順なので mi_check が勝つ。MI.IS_CHECKED は NOT NULL で
// mi_check 行は全 Mi に必ず存在するため、**mi_create は構造的にほぼ絶対に生き残れない**。
// data_types は検索後の後段フィルタで、値としては既知なので、
// 「警告ゼロで必ず0件」という一番たちの悪い形になっていた（2026-08-24 の実利用レビュー）。
func TestHandleGetKyousMCP_MiProjectionWithoutForMiWarns(t *testing.T) {
	tsURL, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()

	sessionID := loginAndGetSession(t, tsURL, gkillAPI, "admin", mcpTestPasswordHash)

	res := getKyousMCP(t, tsURL, sessionID, map[string]any{}, map[string]any{
		"data_types": []string{"mi_create"},
		"count_only": true,
	})

	found := false
	for _, warning := range res.Warnings {
		if strings.Contains(warning, "mi_create") && strings.Contains(warning, "for_mi") {
			found = true
		}
	}
	if !found {
		t.Errorf("for_mi を促す警告が無い: %v", res.Warnings)
	}
}

// for_mi を立てているときは出さない（正しく使っている呼び出しへノイズを出さない）。
func TestHandleGetKyousMCP_MiProjectionWithForMiDoesNotWarn(t *testing.T) {
	tsURL, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()

	sessionID := loginAndGetSession(t, tsURL, gkillAPI, "admin", mcpTestPasswordHash)

	res := getKyousMCP(t, tsURL, sessionID, map[string]any{
		"for_mi":            true,
		"include_create_mi": true,
	}, map[string]any{
		"data_types": []string{"mi_create"},
		"count_only": true,
	})

	for _, warning := range res.Warnings {
		if strings.Contains(warning, "for_mi") {
			t.Errorf("for_mi を立てているのに警告が出ている: %v", res.Warnings)
		}
	}
}

// Mi と無関係な data_types では出さない。
func TestHandleGetKyousMCP_NonMiDataTypeDoesNotWarnAboutForMi(t *testing.T) {
	tsURL, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()

	sessionID := loginAndGetSession(t, tsURL, gkillAPI, "admin", mcpTestPasswordHash)

	res := getKyousMCP(t, tsURL, sessionID, map[string]any{}, map[string]any{
		"data_types": []string{"kmemo"},
		"count_only": true,
	})

	for _, warning := range res.Warnings {
		if strings.Contains(warning, "for_mi") {
			t.Errorf("Mi と無関係なのに for_mi の警告が出ている: %v", res.Warnings)
		}
	}
}

// 登録済みプラグインの data_type だが索引が空、のときは0件の理由を伝える。
//
// 既知集合に載っているので警告の対象外になり、「該当なし」と
// 「プラグインが取り込めていない」が区別できなかった。実際 Claude.ai プラグインが
// データソース欠如で失敗している最中に、data_types:["claude_conversation"] が
// 0件・警告なしで返っていた（2026-08-24 の実利用レビュー）。
//
// HTTP 経由ではなくヘルパを直接見る。実行ファイルの無いプラグインを登録すると
// 検索そのものが内部エラーになり、警告の検査まで到達できないため。
func TestEmptyPluginIndexHint(t *testing.T) {
	newPlugin := func(name, dataType string, provides []string) reps.PluginRepository {
		manifest := gkill_plugin.PluginManifest{
			ProtocolVersion: "1",
			Name:            name,
			DataType:        dataType,
			RepName:         name + "Rep",
			Executable:      "no_such_plugin_binary",
		}
		for _, kind := range provides {
			manifest.Provides = append(manifest.Provides, gkill_plugin.PluginProvidedKind(kind))
		}
		return reps.NewPluginRepository("testuser", t.TempDir(), manifest)
	}

	t.Run("索引が一度も構築されていなければ0件の理由を返す", func(t *testing.T) {
		repositories := &reps.GkillRepositories{
			PluginReps: []reps.PluginRepository{newPlugin("empty_index_plugin", "empty_index_conversation", []string{"tag"})},
		}
		hint := emptyPluginIndexHint(repositories, "empty_index_conversation")
		if !strings.Contains(hint, "matches nothing") {
			t.Errorf("0件になる理由が案内されていない: %q", hint)
		}
		if !strings.Contains(hint, "gkill_get_plugin_list") {
			t.Errorf("読む先が案内されていない: %q", hint)
		}
		// 失敗理由の本文は返さない（利用者の端末の構成を含むため。ADR-0707）
		if strings.Contains(hint, "no_such_plugin_binary") {
			t.Errorf("プラグインの診断文がそのまま載っている: %q", hint)
		}
	})

	t.Run("provides を宣言していないプラグインには索引が無いので断定しない", func(t *testing.T) {
		repositories := &reps.GkillRepositories{
			PluginReps: []reps.PluginRepository{newPlugin("no_provides_plugin", "no_provides_conversation", nil)},
		}
		if hint := emptyPluginIndexHint(repositories, "no_provides_conversation"); hint != "" {
			t.Errorf("取り込み状況を知る手段が無いのに断定している: %q", hint)
		}
	})

	t.Run("関係のない data_type には何も返さない", func(t *testing.T) {
		repositories := &reps.GkillRepositories{
			PluginReps: []reps.PluginRepository{newPlugin("empty_index_plugin", "empty_index_conversation", []string{"tag"})},
		}
		if hint := emptyPluginIndexHint(repositories, "kmemo"); hint != "" {
			t.Errorf("組み込みの data_type に警告が出ている: %q", hint)
		}
	})
}

// for_mi を立てたのに include_*_mi を1つも立てていないときは、0件の理由を伝える。
//
// この形は 0件が返るのに警告が1行も出なかった。実利用の AI は
// 「先週はタスクが無かった」と読み、include フラグを足して19件出るまで気づけなかった
// （2026-08-25 のレビュー）。rep_types の綴り違いには有効値つきの警告が出るのに、
// ここだけ無言なのは非対称でもあった。
func TestHandleGetKyousMCP_ForMiWithoutProjectionFlagsWarns(t *testing.T) {
	tsURL, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()

	sessionID := loginAndGetSession(t, tsURL, gkillAPI, "admin", mcpTestPasswordHash)

	res := getKyousMCP(t, tsURL, sessionID, map[string]any{
		"for_mi": true,
	}, map[string]any{"count_only": true})

	found := false
	for _, warning := range res.Warnings {
		if strings.Contains(warning, "include_create_mi") && strings.Contains(warning, "zero entries") {
			found = true
		}
	}
	if !found {
		t.Errorf("include_*_mi を促す警告が無い: %v", res.Warnings)
	}
}

// include フラグを1つでも立てていれば出さない（正しく使っている呼び出しへノイズを出さない）。
func TestHandleGetKyousMCP_ForMiWithProjectionFlagDoesNotWarn(t *testing.T) {
	tsURL, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()

	sessionID := loginAndGetSession(t, tsURL, gkillAPI, "admin", mcpTestPasswordHash)

	res := getKyousMCP(t, tsURL, sessionID, map[string]any{
		"for_mi":            true,
		"include_create_mi": true,
	}, map[string]any{"count_only": true})

	for _, warning := range res.Warnings {
		if strings.Contains(warning, "include_create_mi") && strings.Contains(warning, "zero entries") {
			t.Errorf("include フラグを立てているのに警告が出ている: %v", res.Warnings)
		}
	}
}

// for_mi を立てていない検索には出さない。
func TestHandleGetKyousMCP_WithoutForMiDoesNotWarnAboutProjectionFlags(t *testing.T) {
	tsURL, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()

	sessionID := loginAndGetSession(t, tsURL, gkillAPI, "admin", mcpTestPasswordHash)

	res := getKyousMCP(t, tsURL, sessionID, map[string]any{}, map[string]any{"count_only": true})

	for _, warning := range res.Warnings {
		if strings.Contains(warning, "include_create_mi") {
			t.Errorf("for_mi 無しなのに警告が出ている: %v", res.Warnings)
		}
	}
}

// 付随 TimeIs は ID と時刻を持つ。
//
// 以前は Title と Tags だけで、同じ題名の打刻が1つの応答に何度並んでも
// 区別も特定もできなかった（実測で lantana 3件に対し付随 TimeIs 90件、
// うち同題名が4回）。「記録時に何が走っていたか」を知る機能なのに時刻が無く、
// 実質「その日に存在した打刻の題名一覧」だった（2026-08-25 の実利用レビュー）。
//
// 組み立て地点（handle_get_kyous_mcp.go）には ti.ID / ti.StartTime / ti.EndTime が
// その場にあり、DTO へ載せていないだけだった。落としてもコンパイルは通るので固定する。
func TestTimeIsMCPDTO_CarriesIDAndTimes(t *testing.T) {
	start := time.Date(2026, 8, 25, 9, 0, 0, 0, time.Local)
	end := time.Date(2026, 8, 25, 10, 30, 0, 0, time.Local)
	dto := req_res.TimeIsMCPDTO{
		ID:        "timeis-1",
		Title:     "大船駅",
		Tags:      []string{"移動"},
		StartTime: start,
		EndTime:   &end,
	}

	encoded, err := json.Marshal(dto)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	for _, key := range []string{"id", "title", "start_time", "end_time"} {
		if _, ok := decoded[key]; !ok {
			t.Errorf("付随TimeIsに %q が無い: %v", key, decoded)
		}
	}
	if decoded["id"] != "timeis-1" {
		t.Errorf("id = %v, want timeis-1", decoded["id"])
	}
}

// 計測中（EndTime が nil）は end_time を出さない。
// ゼロ値の "0001-01-01T00:00:00Z" が出ると「1年に終わった打刻」に見える。
func TestTimeIsMCPDTO_OmitsEndTimeWhileRunning(t *testing.T) {
	dto := req_res.TimeIsMCPDTO{
		ID:        "timeis-2",
		Title:     "作業中",
		StartTime: time.Date(2026, 8, 25, 9, 0, 0, 0, time.Local),
	}
	encoded, err := json.Marshal(dto)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(encoded), "end_time") {
		t.Errorf("計測中なのに end_time が出ている: %s", encoded)
	}
}

// 削除済みの打刻を付随 TimeIs に混ぜない。
//
// FindTimeIs は rep の直叩きで IS_DELETED を見ないため、落とさないと
// 「終了記録ごと消した未終了の打刻」が開始時刻以降のあらゆる記録へ永久に付く。
// 本番実測(2026-08-25): ある kmemo に付いた付随 TimeIs 16件のうち14件が削除済みで、
// 最古は1年前(2025-08-14)の開始。同じ瞬間を playing_time で引くと2件しか返らなかった。
//
// livePlayingTimeIsCandidates の中身を「そのまま返す」に戻すとこのテストが落ちる。
func TestLivePlayingTimeIsCandidates_DropsDeleted(t *testing.T) {
	// 1年前に始めて未終了のまま削除した打刻。playing の意味論では「まだ走っている」に見える。
	deletedGhost := reps.TimeIs{
		IsDeleted: true,
		ID:        "ghost",
		Title:     "カラオケ",
		StartTime: time.Date(2025, 8, 14, 11, 55, 23, 0, time.Local),
	}
	alive := reps.TimeIs{
		ID:        "alive",
		Title:     "覚醒",
		StartTime: time.Date(2026, 8, 24, 8, 27, 38, 0, time.Local),
	}

	live := livePlayingTimeIsCandidates([]reps.TimeIs{deletedGhost, alive})

	if len(live) != 1 {
		t.Fatalf("削除済みが落ちていない: %d件 %+v", len(live), live)
	}
	if live[0].ID != "alive" {
		t.Errorf("残ったのが違う: %q", live[0].ID)
	}

	// 落とさなかった場合に何が起きるかも固定しておく。
	// 幽霊は「今日の記録」を覆ってしまうので、除外が唯一の防御線になる。
	today := time.Date(2026, 8, 24, 17, 30, 0, 0, time.Local)
	if !timeIsCoversMoment(deletedGhost, today) {
		t.Fatal("前提が崩れている: 未終了の打刻は開始時刻以降を覆うはず")
	}
}

// 空でも nil を返さない（呼び出し側が len() で回すだけなので実害は無いが、
// make の容量ヒントごと消す変更を検出する）。
func TestLivePlayingTimeIsCandidates_AllDeleted(t *testing.T) {
	live := livePlayingTimeIsCandidates([]reps.TimeIs{
		{IsDeleted: true, ID: "a"},
		{IsDeleted: true, ID: "b"},
	})
	if len(live) != 0 {
		t.Errorf("全部削除済みなのに残っている: %+v", live)
	}
}

// 「その瞬間に走っていたか」の判定。playing_time の SQL と同じ意味である必要がある
// （START_TIME <= ? AND (? <= END_TIME OR END_TIME IS NULL)）。
func TestTimeIsCoversMoment(t *testing.T) {
	start := time.Date(2026, 8, 25, 9, 0, 0, 0, time.Local)
	end := time.Date(2026, 8, 25, 10, 0, 0, 0, time.Local)
	closed := reps.TimeIs{ID: "closed", StartTime: start, EndTime: &end}
	running := reps.TimeIs{ID: "running", StartTime: start}

	cases := []struct {
		name   string
		timeis reps.TimeIs
		moment time.Time
		want   bool
	}{
		{"終了済み・期間内", closed, start.Add(30 * time.Minute), true},
		{"終了済み・開始前", closed, start.Add(-time.Minute), false},
		{"終了済み・終了後", closed, end.Add(time.Minute), false},
		{"計測中・開始後はいつでも", running, start.Add(400 * 24 * time.Hour), true},
		{"計測中・開始前", running, start.Add(-time.Second), false},
		// 境界ちょうど。SQL が両端を含む(>= / <=)ので、ここも含める。
		// 排他へ戻すと、打刻と同じ時刻に書かれた記録に打刻が付かなくなる。
		{"終了済み・開始ちょうど", closed, start, true},
		{"終了済み・終了ちょうど", closed, end, true},
		{"計測中・開始ちょうど", running, start, true},
	}
	for _, c := range cases {
		if got := timeIsCoversMoment(c.timeis, c.moment); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

// tag / text は ID 付きの別枠でも返す。
//
// gkill_update_text と gkill_delete_kyou(data_type:"text") は text 自身の ID を要求するのに、
// 検索結果は文字列配列しか返しておらず、add_text の応答を持っていない限り
// 後から直すことも消すこともできなかった（2026-08-25 の実利用レビュー）。
// 既存の tags / texts は Web の列と Wear OS が []string を前提にしているので残す。
func TestKyouMCPDTO_CarriesAttachedEntityIDs(t *testing.T) {
	dto := req_res.KyouMCPDTO{
		ID:           "kyou-1",
		Tags:         []string{"仕事"},
		Texts:        []string{"あとで直す"},
		TagEntities:  []req_res.AttachedEntityMCPDTO{{ID: "tag-1", Value: "仕事"}},
		TextEntities: []req_res.AttachedEntityMCPDTO{{ID: "text-1", Value: "あとで直す"}},
	}
	encoded, err := json.Marshal(dto)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// 旧来の文字列配列は残っていること（ワイヤ互換）。
	for _, key := range []string{"tags", "texts"} {
		if _, ok := decoded[key]; !ok {
			t.Errorf("互換フィールド %q が消えている", key)
		}
	}
	for _, key := range []string{"tag_entities", "text_entities"} {
		entities, ok := decoded[key].([]any)
		if !ok || len(entities) != 1 {
			t.Fatalf("%q が無いか件数が違う: %v", key, decoded[key])
		}
		entity, ok := entities[0].(map[string]any)
		if !ok {
			t.Fatalf("%q の要素がオブジェクトでない: %v", key, entities[0])
		}
		if entity["id"] == nil || entity["id"] == "" {
			t.Errorf("%q に id が無い: %v", key, entity)
		}
		if entity["value"] == nil || entity["value"] == "" {
			t.Errorf("%q に value が無い: %v", key, entity)
		}
	}
}

// 通知は ID を返す。data_type:"notification" は delete_kyou が受理するのに、
// これが無いと MCP から通知の id を得る経路が1つも無い
// （notification は data_types フィルタにも group_by のバケットにも出ない）。
func TestNotificationMCPDTO_CarriesID(t *testing.T) {
	encoded, err := json.Marshal(req_res.NotificationMCPDTO{
		ID:      "notification-1",
		Content: "そろそろ出る",
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded["id"] != "notification-1" {
		t.Errorf("通知に id が無い: %v", decoded)
	}
}

// mi_sort_type は「並び順」の名前をしているが、カレンダー範囲・時間帯・曜日を
// 照合する時刻軸も決める。対応する include_*_mi を立てていないと黙って無視され、
// 並び順ではなく **件数** が変わる（実測で同じ1週間が 15件 と 9件 に割れた）。
// 警告を消すとこのテストが落ちる。
func TestMiSortTypeIgnoredWarning(t *testing.T) {
	cases := []struct {
		name     string
		query    find.FindQuery
		wantWarn bool
	}{
		{
			"for_mi が無ければ Mi の話ではないので黙る",
			find.FindQuery{MiSortType: "limit_time"},
			false,
		},
		{
			"mi_sort_type 未指定なら黙る",
			find.FindQuery{ForMi: true, IncludeCreateMi: true},
			false,
		},
		{
			"対応する射影が立っていれば効いているので黙る",
			find.FindQuery{ForMi: true, MiSortType: "limit_time", IncludeLimitMi: true},
			false,
		},
		{
			"別の射影しか立っていないと無視されるので警告する",
			find.FindQuery{ForMi: true, MiSortType: "limit_time", IncludeCreateMi: true},
			true,
		},
		{
			"estimate_start_time も同じ",
			find.FindQuery{ForMi: true, MiSortType: "estimate_start_time", IncludeCreateMi: true},
			true,
		},
	}
	for _, c := range cases {
		query := c.query
		got := miSortTypeIgnoredWarning(&query)
		if (got != "") != c.wantWarn {
			t.Errorf("%s: warning=%q, want warning=%v", c.name, got, c.wantWarn)
		}
	}
}

// addTestCheckedMiWithTimes は作成時刻と更新時刻を分けて指定した完了済み Mi を1件足す。
// mi_check 射影の RelatedTime は UPDATE_TIME なので、create と update を離すと
// 「どの窓で検索するかで代表射影が変わる」記録になる（ADR-0621 の再現材料）。
func addTestCheckedMiWithTimes(t *testing.T, tsURL string, sessionID string, title string, createTime time.Time, updateTime time.Time) string {
	t.Helper()
	id := GenerateNewID()
	resp := postJSON(t, tsURL+"/api/add_mi", &req_res.AddMiRequest{
		SessionID:  sessionID,
		LocaleName: "en",
		Mi: reps.Mi{
			ID:         id,
			Title:      title,
			BoardName:  "Inbox",
			DataType:   "mi",
			IsChecked:  true,
			CreateTime: createTime,
			CreateApp:  "test",
			CreateUser: "admin",
			UpdateTime: updateTime,
			UpdateApp:  "test",
			UpdateUser: "admin",
		},
	})
	defer resp.Body.Close()
	var addResp req_res.AddMiResponse
	if err := json.NewDecoder(resp.Body).Decode(&addResp); err != nil {
		t.Fatalf("decode add mi response: %v", err)
	}
	if len(addResp.Errors) > 0 {
		t.Fatalf("add mi errors: %+v", addResp.Errors)
	}
	return id
}

// walkMCPPages は next_cursor を辿って最後まで読み、各頁の応答を順に返す。
// limits は頁ごとの limit で、足りない頁は最後の値を使う。
func walkMCPPages(t *testing.T, tsURL string, sessionID string, query map[string]any, limits []int) []req_res.GetKyousMCPResponse {
	t.Helper()
	pages := []req_res.GetKyousMCPResponse{}
	cursor := ""
	for i := 0; ; i++ {
		limit := limits[len(limits)-1]
		if i < len(limits) {
			limit = limits[i]
		}
		extra := map[string]any{"limit": limit}
		if cursor != "" {
			extra["cursor"] = cursor
		}
		page := getKyousMCP(t, tsURL, sessionID, query, extra)
		pages = append(pages, page)
		if !page.HasMore {
			return pages
		}
		if page.NextCursor == "" {
			t.Fatalf("page %d: has_more=true なのに next_cursor が空", i+1)
		}
		if i > 1000 {
			t.Fatal("ページングが終わらない")
		}
		cursor = page.NextCursor
	}
}

// assertMCPPagesAreExact は「各頁の remaining は前頁の remaining − returned」「cursor 頁に total_count 無し」
// 「ID の重複ゼロ」「Σreturned = total = 最終頁 remaining 0」を検査する。
func assertMCPPagesAreExact(t *testing.T, pages []req_res.GetKyousMCPResponse, total int) {
	t.Helper()
	if len(pages) == 0 {
		t.Fatal("頁が1つも無い")
	}
	first := pages[0]
	if first.TotalCount == nil || *first.TotalCount != total {
		t.Fatalf("1頁目の total_count = %v, want %d", first.TotalCount, total)
	}
	seen := map[string]int{}
	sum := 0
	prevRemaining := total
	for i, page := range pages {
		sum += page.ReturnedCount
		if page.RemainingCount != prevRemaining-page.ReturnedCount {
			t.Errorf("page %d: remaining=%d, want %d (前頁 remaining %d − returned %d)", i+1, page.RemainingCount, prevRemaining-page.ReturnedCount, prevRemaining, page.ReturnedCount)
		}
		prevRemaining = page.RemainingCount
		if i > 0 && page.TotalCount != nil {
			t.Errorf("page %d: cursor 頁に total_count が入っている: %d", i+1, *page.TotalCount)
		}
		for _, kyou := range page.Kyous {
			seen[kyou.ID]++
		}
	}
	for id, n := range seen {
		if n > 1 {
			t.Errorf("ID %s が %d 回返った（ページをまたいだ重複）", id, n)
		}
	}
	if sum != total {
		t.Errorf("Σreturned = %d, want %d", sum, total)
	}
	if len(seen) != total {
		t.Errorf("返った ID の種類 = %d, want %d", len(seen), total)
	}
	last := pages[len(pages)-1]
	if last.RemainingCount != 0 || last.HasMore {
		t.Errorf("最終頁: remaining=%d has_more=%v", last.RemainingCount, last.HasMore)
	}
}

// 2026-09-18 の実利用報告: 150件を limit 5 → 3 → 3 で読むと remaining_count が 145 → 145 → 143 と
// 単調に減らなかった。正体はカーソルの期間押し下げで Mi の代表射影が変わり
// （1頁目は mi_check=UPDATE_TIME、2頁目以降の狭い窓では mi_create=CREATE_TIME）、
// 返却済みの Mi がカーソルより後ろに別の射影名で再出現していたこと。
// 残件数は副作用で、実害は同じ記録がページをまたいで重複すること（ADR-0621）。
func TestHandleGetKyousMCP_RemainingCountMonotonicAcrossPages(t *testing.T) {
	tsURL, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()
	sessionID := loginAndGetSession(t, tsURL, gkillAPI, "admin", mcpTestPasswordHash)

	base := time.Now().Truncate(time.Second).Add(-24 * time.Hour)
	const kmemoCount = 146
	const miCount = 4
	const total = kmemoCount + miCount
	for i := range kmemoCount {
		addTestKmemoWithRelatedTime(t, tsURL, sessionID, fmt.Sprintf("単調減少用メモ%d", i), base.Add(time.Duration(i)*time.Minute))
	}
	// Mi は CREATE_TIME を古い側（メモ列の末尾と同じ時刻帯）、UPDATE_TIME を全メモより新しい側に置く。
	// 1頁目は mi_check（UPDATE_TIME）として先頭に出て、2頁目以降の押し下げた窓では mi_check が窓外へ落ちる。
	for i := range miCount {
		addTestCheckedMiWithTimes(t, tsURL, sessionID, fmt.Sprintf("単調減少用タスク%d", i),
			base.Add(time.Duration(i)*time.Minute), base.Add(200*time.Minute))
	}

	pages := walkMCPPages(t, tsURL, sessionID, map[string]any{}, []int{5, 3, 3})

	// 前提の確認: 1頁目の先頭 4 件は mi_check（UPDATE_TIME が最新）。
	// これが崩れると以降の検査が「再現していないのに通る」ことになる。
	first := pages[0]
	if len(first.Kyous) != 5 {
		t.Fatalf("1頁目 returned=%d, want 5", len(first.Kyous))
	}
	for i := range miCount {
		if first.Kyous[i].DataType != "mi_check" {
			t.Fatalf("1頁目 %d 件目の data_type=%q, want mi_check（前提が崩れている）", i+1, first.Kyous[i].DataType)
		}
	}

	// 報告どおりの形: 145 → 142 → 139
	for i, want := range []int{145, 142, 139} {
		if len(pages) <= i {
			t.Fatalf("page %d が無い", i+1)
		}
		if pages[i].RemainingCount != want {
			t.Errorf("page %d: remaining=%d, want %d", i+1, pages[i].RemainingCount, want)
		}
	}
	assertMCPPagesAreExact(t, pages, total)
}

// for_mi 検索（RelatedTime が mi_sort_type の射影時刻へ上書きされる経路）でも
// 同じ材料でページをまたいだ重複が出ないこと。
func TestHandleGetKyousMCP_ForMiPagingDoesNotRepeatTasks(t *testing.T) {
	tsURL, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()
	sessionID := loginAndGetSession(t, tsURL, gkillAPI, "admin", mcpTestPasswordHash)

	base := time.Now().Truncate(time.Second).Add(-24 * time.Hour)
	const miCount = 6
	for i := range miCount {
		addTestCheckedMiWithTimes(t, tsURL, sessionID, fmt.Sprintf("for_mi頁送り用タスク%d", i),
			base.Add(time.Duration(i)*time.Minute), base.Add(200*time.Minute))
	}

	query := map[string]any{"for_mi": true, "include_create_mi": true, "include_check_mi": true}
	pages := walkMCPPages(t, tsURL, sessionID, query, []int{1})
	assertMCPPagesAreExact(t, pages, miCount)
}

// expandMCPDataTypes: エンティティ名は全射影へ、nil は nil、空は空、重複は落ちて順序は保つ。
func TestExpandMCPDataTypes(t *testing.T) {
	if got := expandMCPDataTypes(nil); got != nil {
		t.Errorf("nil は nil のまま（未使用）のはず: %v", got)
	}
	if got := expandMCPDataTypes([]string{}); got == nil || len(got) != 0 {
		t.Errorf("空は非nilの空（0件指定）のはず: %#v", got)
	}
	got := expandMCPDataTypes([]string{"mi", "kmemo", "mi_create", "timeis", "zzz_bogus"})
	want := []string{"mi_create", "mi_check", "mi_limit", "mi_start", "mi_end", "kmemo", "timeis_start", "timeis_end", "zzz_bogus"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("expand = %v, want %v", got, want)
	}
	for _, projection := range miProjectionDataTypes {
		if _, ok := knownMCPDataTypes(&reps.GkillRepositories{})[projection]; !ok {
			t.Errorf("射影 %q が knownMCPDataTypes に無い（表が2つに割れている）", projection)
		}
	}
}

// 2026-09-18 の実利用報告: data_types:["timeis"] / ["mi"] が警告なしで0件、
// ["timeis","mi","idf"] が idf 単体と同じ件数。エンティティ名は射影へ展開して受理する（ADR-0623）。
func TestHandleGetKyousMCP_DataTypesAcceptsEntityNames(t *testing.T) {
	tsURL, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()
	sessionID := loginAndGetSession(t, tsURL, gkillAPI, "admin", mcpTestPasswordHash)

	now := time.Now().Truncate(time.Second)
	addTestKmemo(t, tsURL, sessionID, "エンティティ名用メモ")
	endTime := now.Add(-30 * time.Minute)
	addTestTimeIsWithPeriod(t, tsURL, sessionID, "エンティティ名用打刻", now.Add(-time.Hour), &endTime)
	addTestCheckedMiWithTimes(t, tsURL, sessionID, "エンティティ名用タスク", now.Add(-2*time.Hour), now.Add(-time.Hour))

	count := func(dataTypes []string) (int, []string) {
		res := getKyousMCP(t, tsURL, sessionID, map[string]any{}, map[string]any{"count_only": true, "data_types": dataTypes})
		if res.TotalCount == nil {
			t.Fatalf("count_only なのに total_count が無い: %v", dataTypes)
		}
		return *res.TotalCount, res.Warnings
	}
	assertNoUnknownWarning := func(dataTypes []string, warnings []string) {
		t.Helper()
		for _, warning := range warnings {
			if strings.Contains(warning, "unknown data_type") {
				t.Errorf("%v が未知値として警告された: %q", dataTypes, warning)
			}
		}
	}

	timeisCount, warnings := count([]string{"timeis"})
	assertNoUnknownWarning([]string{"timeis"}, warnings)
	if timeisCount == 0 {
		t.Fatal("data_types:[timeis] が0件（エンティティ名が射影へ展開されていない）")
	}
	if projected, _ := count([]string{"timeis_start", "timeis_end"}); projected != timeisCount {
		t.Errorf("timeis(%d) と timeis_start+timeis_end(%d) の件数が違う", timeisCount, projected)
	}

	miCount, warnings := count([]string{"mi"})
	assertNoUnknownWarning([]string{"mi"}, warnings)
	if miCount != 1 {
		t.Errorf("data_types:[mi] = %d, want 1（潰し込み後の代表1件）", miCount)
	}

	kmemoCount, _ := count([]string{"kmemo"})
	sum, warnings := count([]string{"timeis", "mi", "kmemo"})
	assertNoUnknownWarning([]string{"timeis", "mi", "kmemo"}, warnings)
	if sum != timeisCount+miCount+kmemoCount {
		t.Errorf("[timeis,mi,kmemo] = %d, want %d+%d+%d", sum, timeisCount, miCount, kmemoCount)
	}

	// 綴り違いの警告は従来どおり
	if _, warnings := count([]string{"zzz_bogus"}); !slices.ContainsFunc(warnings, func(w string) bool { return strings.Contains(w, "zzz_bogus") }) {
		t.Errorf("未知の data_type が警告されていない: %v", warnings)
	}
}

// num_min / num_max の結果に kc / nlog / lantana が混ざったら警告する（単位の無い1本の軸で比べているため）。
// 1種類だけなら黙る。2026-09-18 の実利用報告: num_min:7 だけで気分・歩数・円が混ざって 20,624 件。
func TestHandleGetKyousMCP_NumFilterWarnsWhenKindsMix(t *testing.T) {
	tsURL, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()
	sessionID := loginAndGetSession(t, tsURL, gkillAPI, "admin", mcpTestPasswordHash)

	addTestKC(t, tsURL, sessionID, "歩数", "10")
	addTestLantana(t, tsURL, sessionID, 8)

	hasMixWarning := func(warnings []string) bool {
		return slices.ContainsFunc(warnings, func(w string) bool { return strings.Contains(w, "unit-less axis") })
	}

	mixed := getKyousMCP(t, tsURL, sessionID, map[string]any{}, map[string]any{"num_min": 7})
	if len(mixed.Kyous) != 2 {
		t.Fatalf("kc 10 と lantana 8 の両方が num_min:7 に当たるはず: %d件", len(mixed.Kyous))
	}
	if !hasMixWarning(mixed.Warnings) {
		t.Errorf("種別混在の警告が無い: %v", mixed.Warnings)
	}
	for _, want := range []string{"1 kc", "1 lantana"} {
		if !slices.ContainsFunc(mixed.Warnings, func(w string) bool { return strings.Contains(w, want) }) {
			t.Errorf("警告に %q が無い: %v", want, mixed.Warnings)
		}
	}

	only := getKyousMCP(t, tsURL, sessionID, map[string]any{}, map[string]any{"num_min": 7, "data_types": []string{"lantana"}})
	if len(only.Kyous) != 1 || only.Kyous[0].DataType != "lantana" {
		t.Fatalf("data_types:[lantana] で lantana 1件のはず: %+v", only.Kyous)
	}
	if hasMixWarning(only.Warnings) {
		t.Errorf("1種類しか無いのに混在警告が出ている: %v", only.Warnings)
	}
}

// query.ids のうち結果に出なかった ID は警告で名指しする（存在しない／削除済み／他条件で落ちた、は区別しない）。
// tags / reps / rep_types / data_types だけが警告され、ids だけ無言だった（2026-09-18 の実利用報告）。
func TestHandleGetKyousMCP_UnmatchedIDsWarn(t *testing.T) {
	tsURL, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()
	sessionID := loginAndGetSession(t, tsURL, gkillAPI, "admin", mcpTestPasswordHash)

	realID := addTestKmemo(t, tsURL, sessionID, "ids用メモ")
	bogusID := "00000000-0000-4000-8000-000000000000"

	res := getKyousMCP(t, tsURL, sessionID, map[string]any{"ids": []string{realID, bogusID}}, map[string]any{"count_only": true})
	if res.TotalCount == nil || *res.TotalCount != 1 {
		t.Fatalf("実在する1件だけが数えられるはず: %v", res.TotalCount)
	}
	idWarnings := []string{}
	for _, warning := range res.Warnings {
		if strings.HasPrefix(warning, "query.ids:") {
			idWarnings = append(idWarnings, warning)
		}
	}
	if len(idWarnings) != 1 {
		t.Fatalf("query.ids の警告が1行のはず: %v", res.Warnings)
	}
	if !strings.Contains(idWarnings[0], bogusID) || !strings.Contains(idWarnings[0], "1 of 2") {
		t.Errorf("不在 ID を名指ししていない: %q", idWarnings[0])
	}
	if strings.Contains(idWarnings[0], realID) {
		t.Errorf("実在する ID まで不一致扱い: %q", idWarnings[0])
	}
	if !strings.Contains(idWarnings[0], "cannot tell these apart") {
		t.Errorf("「区別できない」と言っていない: %q", idWarnings[0])
	}

	ok := getKyousMCP(t, tsURL, sessionID, map[string]any{"ids": []string{realID}}, map[string]any{})
	for _, warning := range ok.Warnings {
		if strings.HasPrefix(warning, "query.ids:") {
			t.Errorf("全 ID が一致しているのに警告が出た: %q", warning)
		}
	}
}
