package gkill_server_api

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/api/req_res"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
)

// rep名での絞り込みを**キャッシュ有無の両方**で見る。
//
// 本番の既定は --cache_in_memory=true。そのとき repositories.Reps は型ごとの
// キャッシュrep1個に畳まれ、1つのキャッシュ表に全repの行が REP_NAME 付きで同居する。
// 絞り込みを「検索対象repを選ぶ」から「検索結果を絞る」へ移した(Kyou.RepName で判定)ので、
// **効いているかを見るのはこちらの経路**。
//
// 見るのは次の4つ。かつては gkill_server_api_test.go に「キャッシュOFFで、
// 存在しないrep名を渡すと0件」だけを見るテストがあったが、
//   - 絞り込みが効きすぎて、選んだrepの記録まで消える
//   - キャッシュrepの Kyou.RepName が実rep名になっておらず、全部消える
//
// のどちらも検出できなかったのでこちらへ統合した（重複するので旧テストは削除済み）。
//
// **UpdateCache の前後で2回見るのが要点。** 追加直後はキャッシュ表の REP_NAME が空で、
// filterKyousByRepName の「空の行は残す」分岐で全部素通りするため、
// 追加直後だけだと許可リスト側の分岐を一度も検証できない。
func TestHandleGetKyous_RepFilter_CacheInMemory(t *testing.T) {
	for _, cacheInMemory := range []bool{false, true} {
		name := "cache_in_memory=false"
		if cacheInMemory {
			name = "cache_in_memory=true"
		}
		t.Run(name, func(t *testing.T) {
			if cacheInMemory {
				useCacheInMemory(t)
			}
			tsURL, gkillAPI, cleanup := setupTestRouterWithRepos(t)
			defer cleanup()

			sessionID := loginAndGetSession(t, tsURL, gkillAPI, "admin", regressionTestPasswordHash)
			now := time.Now().Truncate(time.Second)

			kmemoID := GenerateNewID()
			addReq := &req_res.AddKmemoRequest{
				SessionID:  sessionID,
				LocaleName: "en",
				Kmemo: reps.Kmemo{
					ID:          kmemoID,
					Content:     "rep絞り込みテスト",
					RelatedTime: now,
					DataType:    "kmemo",
					CreateTime:  now,
					CreateApp:   "test",
					CreateUser:  "admin",
					UpdateTime:  now,
					UpdateApp:   "test",
					UpdateUser:  "admin",
				},
			}
			addResp := postJSON(t, tsURL+"/api/add_kmemo", addReq)
			addResp.Body.Close()

			// 実際のkmemo repの名前を取る。キャッシュONでもここは実rep名を返す
			// (GetAllRepNames は UnWrap してから GetRepName する)
			repNamesReq := &req_res.GetAllRepNamesRequest{SessionID: sessionID, LocaleName: "en"}
			repNamesResp := postJSON(t, tsURL+"/api/get_all_rep_names", repNamesReq)
			var repNames req_res.GetAllRepNamesResponse
			if err := json.NewDecoder(repNamesResp.Body).Decode(&repNames); err != nil {
				t.Fatalf("decode get all rep names response: %v", err)
			}
			repNamesResp.Body.Close()
			kmemoRepName := ""
			for _, repName := range repNames.RepNames {
				if strings.Contains(strings.ToLower(repName), "kmemo") {
					kmemoRepName = repName
					break
				}
			}
			if kmemoRepName == "" {
				t.Fatalf("kmemo repの名前が見つからない: %v", repNames.RepNames)
			}

			startDate := now.Add(-time.Hour)
			endDate := now.Add(time.Hour)

			contains := func(t *testing.T, query *find.FindQuery) bool {
				t.Helper()
				query.CalendarStartDate = &startDate
				query.CalendarEndDate = &endDate
				getReq := &req_res.GetKyousRequest{SessionID: sessionID, LocaleName: "en", Query: query}
				resp := postJSON(t, tsURL+"/api/get_kyous", getReq)
				defer resp.Body.Close()
				var getResp req_res.GetKyousResponse
				if err := json.NewDecoder(resp.Body).Decode(&getResp); err != nil {
					t.Fatalf("decode get kyous response: %v", err)
				}
				if len(getResp.Errors) > 0 {
					t.Fatalf("get kyous errors: %+v", getResp.Errors)
				}
				for _, kyou := range getResp.Kyous {
					if kyou.ID == kmemoID {
						return true
					}
				}
				return false
			}

			repNameOfAddedKyou := func(t *testing.T) string {
				t.Helper()
				getReq := &req_res.GetKyousRequest{SessionID: sessionID, LocaleName: "en",
					Query: &find.FindQuery{CalendarStartDate: &startDate, CalendarEndDate: &endDate}}
				resp := postJSON(t, tsURL+"/api/get_kyous", getReq)
				defer resp.Body.Close()
				var getResp req_res.GetKyousResponse
				if err := json.NewDecoder(resp.Body).Decode(&getResp); err != nil {
					t.Fatalf("decode get kyous response: %v", err)
				}
				for _, kyou := range getResp.Kyous {
					if kyou.ID == kmemoID {
						return kyou.RepName
					}
				}
				t.Fatalf("追加した kmemo が返ってこない")
				return ""
			}

			assertRepFilter := func(t *testing.T) {
				t.Helper()
				if !contains(t, &find.FindQuery{Reps: []string{kmemoRepName}}) {
					t.Error("選んだrepの記録が返っていない(絞り込みが効きすぎている)")
				}
				if contains(t, &find.FindQuery{Reps: []string{"nonexistent_rep_name_xyz"}}) {
					t.Error("選んでいないrep名で記録が返っている")
				}
				if !contains(t, &find.FindQuery{Reps: nil}) {
					t.Error("rep未指定で記録が返っていない(nilを0件指定と取り違えている)")
				}
				if contains(t, &find.FindQuery{Reps: []string{}}) {
					t.Error("非nil空は0件指定のはず")
				}
			}

			// --- 追加直後 ---
			// キャッシュONのとき、write-through は呼び出し側の値をそのまま INSERT するので
			// キャッシュ表の REP_NAME は**空**（実rep名が入るのは次の UpdateCache）。
			// filterKyousByRepName が「RepName が空の行は残す」のはこのため。
			// 落とすと**いま追加した記録が最大1分間一覧から消える**。
			addedRepName := repNameOfAddedKyou(t)
			if cacheInMemory && addedRepName != "" {
				t.Logf("追加直後の RepName が %q。空を想定していたので、"+
					"下の UpdateCache 後のケースと同じ経路を検証していることになる", addedRepName)
			}
			assertRepFilter(t)

			// --- UpdateCache 後 ---
			// ここで初めてキャッシュ表の REP_NAME に実rep名が入る。
			// **この段があって初めて filterKyousByRepName の許可リスト側の分岐を通る。**
			// 追加直後だけだと「RepName が空だから残った」で全部通ってしまい、
			// 許可リストを一度も検証しないまま緑になる（実際そうなっていた）。
			updateResp := postJSON(t, tsURL+"/api/update_cache", &req_res.UpdateCacheRequest{
				SessionID: sessionID, UserIDs: []string{"admin"}, LocaleName: "en"})
			var updateCacheResponse req_res.UpdateCacheResponse
			if err := json.NewDecoder(updateResp.Body).Decode(&updateCacheResponse); err != nil {
				t.Fatalf("decode update cache response: %v", err)
			}
			updateResp.Body.Close()
			if len(updateCacheResponse.Errors) > 0 {
				t.Fatalf("update cache errors: %+v", updateCacheResponse.Errors)
			}

			if got := repNameOfAddedKyou(t); got != kmemoRepName {
				t.Fatalf("UpdateCache 後の RepName = %q, want %q。"+
					"実rep名が入らないと、以降のアサーションは"+
					"「RepName が空の行は残す」分岐で素通りし許可リストを検証しない", got, kmemoRepName)
			}
			assertRepFilter(t)
		})
	}
}
