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

// TX(KFTLの送信経路)で追加した記録が、rep名を指定した検索で返ることを見る。
//
// commit_tx は一時リポジトリから読み直した記録をキャッシュへ write-through する。
// GetXxxByTXID は `? AS REP_NAME` に **temp rep の名前**("KmemoTemp" 等)を差し込んで返すので、
// そのまま write-through するとキャッシュ表の REP_NAME 列に実在しないrep名が入る。
// find_filter.go の filterKyousByRepName は「空なら残す / 非空なら指定repに含まれるものだけ残す」
// なので、この合成名は必ず落ちる ―― **KFTLで書いた記録が一覧から丸ごと消える**。
// GUIは常に reps を送るので、通常の検索が毎回この経路になる。
//
// 既存の TestHandleGetKyous_RepFilter_CacheInMemory は add_kmemo(非tx)しか見ておらず、
// 非tx経路はリクエスト由来の記録をそのまま書くので RepName が空 ＝ 落ちない。
// tx経路だけが踏む壊れ方なので、テストも tx で書く必要がある。
func TestHandleGetKyous_TxAddedKyouSurvivesRepFilter(t *testing.T) {
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

			txID := GenerateNewID()
			kmemoID := GenerateNewID()
			addReq := &req_res.AddKmemoRequest{
				SessionID:  sessionID,
				LocaleName: "en",
				TXID:       &txID,
				Kmemo: reps.Kmemo{
					ID:          kmemoID,
					Content:     "tx経由のrep絞り込みテスト",
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

			commitReq := &req_res.CommitTxRequest{
				SessionID:  sessionID,
				LocaleName: "en",
				TXID:       txID,
			}
			commitResp := postJSON(t, tsURL+"/api/commit_tx", commitReq)
			var commitRes req_res.CommitTxResponse
			if err := json.NewDecoder(commitResp.Body).Decode(&commitRes); err != nil {
				t.Fatalf("decode commit tx response: %v", err)
			}
			commitResp.Body.Close()
			if len(commitRes.Errors) > 0 {
				t.Fatalf("commit tx errors: %+v", commitRes.Errors)
			}

			// 実際のkmemo repの名前を取る(GetAllRepNames は UnWrap してから GetRepName する)
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

			if !contains(t, &find.FindQuery{Reps: nil}) {
				t.Fatal("tx で追加した記録が rep未指定でも返っていない(コミット自体が失敗している)")
			}
			if !contains(t, &find.FindQuery{Reps: []string{kmemoRepName}}) {
				t.Error("tx で追加した記録が rep名指定で消えている" +
					"(commit_tx が temp rep の合成rep名をキャッシュへ書いている疑い)")
			}
		})
	}
}
