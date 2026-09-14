package gkill_server_api

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/api/req_res"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
)

// /api/commit_tx が「全部書くか、何も書かないか」であることを HTTP 経由で固定する。
//
// 2026-09-15 まで commit_tx は種別ごとの逐次追記で、途中の種別で失敗すると書けた種別だけが残っていた
// （部分確定。外部レビュー #4）。今は書き込み rep のファイルを ATTACH した1つの SQLite トランザクションで
// 確定する（dao/reps/commit_tx.go）。単体は commit_tx_test.go が守り、ここではハンドラの配線
// （エラーコード ERR000419・committed[]・temp の消費・失敗時の discard 経路）を見る。
// documents/adr/0219-commit-tx-is-one-sqlite-transaction.md

func commitTxAtomicTestNow() time.Time { return time.Now().Truncate(time.Second) }

func decodeCommitTx(t *testing.T, resp *http.Response) (int, req_res.CommitTxResponse) {
	t.Helper()
	defer resp.Body.Close()
	var res req_res.CommitTxResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		t.Fatalf("decode commit tx response: %v", err)
	}
	return resp.StatusCode, res
}

// kyouIDsInRange は期間検索で返る Kyou の id 集合を返す。
func kyouIDsInRange(t *testing.T, tsURL, sessionID string, start, end time.Time) map[string]reps.Kyou {
	t.Helper()
	query := &find.FindQuery{CalendarStartDate: &start, CalendarEndDate: &end}
	resp := postJSON(t, tsURL+"/api/get_kyous", &req_res.GetKyousRequest{SessionID: sessionID, LocaleName: "en", Query: query})
	defer resp.Body.Close()
	var res req_res.GetKyousResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		t.Fatalf("decode get kyous response: %v", err)
	}
	if len(res.Errors) > 0 {
		t.Fatalf("get kyous errors: %+v", res.Errors)
	}
	found := map[string]reps.Kyou{}
	for _, kyou := range res.Kyous {
		found[kyou.ID] = kyou
	}
	return found
}

// kc（書ける）と本文が空の kmemo（leaf の INSERT 前検査で落ちる）を同じ tx に積んで commit すると、
// ERR000419 で返り、**kc も残らない**。以前は kc だけが残っていた。
func TestHandleCommitTx_RollsBackWholeTxOnFailure(t *testing.T) {
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
			now := commitTxAtomicTestNow()

			txID := GenerateNewID()
			kcID := GenerateNewID()
			addKC := postJSON(t, tsURL+"/api/add_kc", &req_res.AddKCRequest{
				SessionID: sessionID, LocaleName: "en", TXID: &txID,
				KC: reps.KC{
					ID: kcID, Title: "commit tx atomic kc", NumValue: "1", RelatedTime: now, DataType: "kc",
					CreateTime: now, CreateApp: "test", CreateUser: "admin", UpdateTime: now, UpdateApp: "test", UpdateUser: "admin",
				},
			})
			addKC.Body.Close()
			kmemoID := GenerateNewID()
			addKmemo := postJSON(t, tsURL+"/api/add_kmemo", &req_res.AddKmemoRequest{
				SessionID: sessionID, LocaleName: "en", TXID: &txID,
				Kmemo: reps.Kmemo{
					ID: kmemoID, Content: "   ", RelatedTime: now, DataType: "kmemo",
					CreateTime: now, CreateApp: "test", CreateUser: "admin", UpdateTime: now, UpdateApp: "test", UpdateUser: "admin",
				},
			})
			addKmemo.Body.Close()

			status, res := decodeCommitTx(t, postJSON(t, tsURL+"/api/commit_tx", &req_res.CommitTxRequest{SessionID: sessionID, LocaleName: "en", TXID: txID}))
			if len(res.Errors) != 1 || res.Errors[0].ErrorCode != message.CommitTxRolledBackError {
				t.Fatalf("errors = %+v, want %s 1件", res.Errors, message.CommitTxRolledBackError)
			}
			if status != http.StatusInternalServerError {
				t.Errorf("status = %d, want 500", status)
			}
			if len(res.Committed) != 0 {
				t.Errorf("失敗した commit の committed = %+v, want 空", res.Committed)
			}

			found := kyouIDsInRange(t, tsURL, sessionID, now.Add(-time.Hour), now.Add(time.Hour))
			if _, ok := found[kcID]; ok {
				t.Error("失敗した tx の kc が検索に出ている（部分確定）")
			}
			if _, ok := found[kmemoID]; ok {
				t.Error("失敗した tx の kmemo が検索に出ている")
			}

			// 失敗した tx は discard で捨てられる（temp に残っているので何も無いエラーにはならない）
			discard := postJSON(t, tsURL+"/api/discard_tx", &req_res.DiscardTxRequest{SessionID: sessionID, LocaleName: "en", TXID: txID})
			var discardRes req_res.DiscardTxResponse
			if err := json.NewDecoder(discard.Body).Decode(&discardRes); err != nil {
				t.Fatalf("decode discard tx response: %v", err)
			}
			discard.Body.Close()
			if len(discardRes.Errors) > 0 {
				t.Errorf("discard tx errors: %+v", discardRes.Errors)
			}
		})
	}
}

// 成功した commit は committed[] に全件を返し、temp を消費する。同じ txID をもう一度 commit しても
// 何も書かれない（版が増えない）。
func TestHandleCommitTx_ReturnsCommittedAndConsumesTemp(t *testing.T) {
	tsURL, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()
	sessionID := loginAndGetSession(t, tsURL, gkillAPI, "admin", regressionTestPasswordHash)
	now := commitTxAtomicTestNow()

	txID := GenerateNewID()
	kmemoID := GenerateNewID()
	addKmemo := postJSON(t, tsURL+"/api/add_kmemo", &req_res.AddKmemoRequest{
		SessionID: sessionID, LocaleName: "en", TXID: &txID,
		Kmemo: reps.Kmemo{
			ID: kmemoID, Content: "commit tx atomic kmemo", RelatedTime: now, DataType: "kmemo",
			CreateTime: now, CreateApp: "test", CreateUser: "admin", UpdateTime: now, UpdateApp: "test", UpdateUser: "admin",
		},
	})
	addKmemo.Body.Close()
	tagID := GenerateNewID()
	addTag := postJSON(t, tsURL+"/api/add_tag", &req_res.AddTagRequest{
		SessionID: sessionID, LocaleName: "en", TXID: &txID,
		Tag: reps.Tag{
			ID: tagID, TargetID: kmemoID, Tag: "commit_tx_atomic", RelatedTime: now,
			CreateTime: now, CreateApp: "test", CreateUser: "admin", UpdateTime: now, UpdateApp: "test", UpdateUser: "admin",
		},
	})
	addTag.Body.Close()

	status, res := decodeCommitTx(t, postJSON(t, tsURL+"/api/commit_tx", &req_res.CommitTxRequest{SessionID: sessionID, LocaleName: "en", TXID: txID}))
	if len(res.Errors) > 0 {
		t.Fatalf("commit tx errors: %+v", res.Errors)
	}
	if status != http.StatusOK {
		t.Errorf("status = %d, want 200", status)
	}
	committed := map[string]bool{}
	for _, record := range res.Committed {
		committed[record.DataType+":"+record.ID] = record.Updated
	}
	for _, key := range []string{"kmemo:" + kmemoID, "tag:" + tagID} {
		updated, ok := committed[key]
		if !ok {
			t.Errorf("committed に %s が無い: %+v", key, res.Committed)
		} else if updated {
			t.Errorf("%s は新規作成なのに updated=true", key)
		}
	}

	found := kyouIDsInRange(t, tsURL, sessionID, now.Add(-time.Hour), now.Add(time.Hour))
	if _, ok := found[kmemoID]; !ok {
		t.Fatal("commit した kmemo が検索に出ない")
	}

	// 2回目の commit: temp は消費済みなので何も書かれない
	_, again := decodeCommitTx(t, postJSON(t, tsURL+"/api/commit_tx", &req_res.CommitTxRequest{SessionID: sessionID, LocaleName: "en", TXID: txID}))
	if len(again.Errors) > 0 {
		t.Fatalf("2回目の commit tx errors: %+v", again.Errors)
	}
	if len(again.Committed) != 0 {
		t.Errorf("2回目の commit が何か書いた（temp が消費されていない）: %+v", again.Committed)
	}
	histories := postJSON(t, tsURL+"/api/get_kyou", &req_res.GetKyouRequest{SessionID: sessionID, LocaleName: "en", ID: kmemoID})
	var historiesRes req_res.GetKyouResponse
	if err := json.NewDecoder(histories.Body).Decode(&historiesRes); err != nil {
		t.Fatalf("decode get kyou response: %v", err)
	}
	histories.Body.Close()
	if len(historiesRes.KyouHistories) != 1 {
		t.Errorf("kmemo の版が %d 件（同じ tx の再 commit で二重登録された疑い）", len(historiesRes.KyouHistories))
	}
}

// 既存の TimeIs を tx で更新して discard すると、元の TimeIs はそのまま検索に出る。
//
// 2026-09-15 まで usecase の Update* は tx 中でも最新版アドレス表を新しい版の時刻で進めていたので、
// discard のあと表だけが未来を指し、find_filter.go の「表より古い版は除外」で
// **既存の記録が検索から消えていた**（保存に失敗したのに副作用が残る実例）。
func TestHandleUpdateTimeIsInTx_DiscardKeepsExistingTimeIsVisible(t *testing.T) {
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
			now := commitTxAtomicTestNow()

			timeisID := GenerateNewID()
			original := reps.TimeIs{
				ID: timeisID, Title: "tx discard keeps timeis", StartTime: now, DataType: "timeis",
				CreateTime: now, CreateApp: "test", CreateUser: "admin", UpdateTime: now, UpdateApp: "test", UpdateUser: "admin",
			}
			add := postJSON(t, tsURL+"/api/add_timeis", &req_res.AddTimeIsRequest{SessionID: sessionID, LocaleName: "en", TimeIs: original})
			var addRes req_res.AddTimeIsResponse
			if err := json.NewDecoder(add.Body).Decode(&addRes); err != nil {
				t.Fatalf("decode add timeis response: %v", err)
			}
			add.Body.Close()
			if len(addRes.Errors) > 0 {
				t.Fatalf("add timeis errors: %+v", addRes.Errors)
			}
			if _, ok := kyouIDsInRange(t, tsURL, sessionID, now.Add(-time.Hour), now.Add(time.Hour))[timeisID]; !ok {
				t.Fatal("追加した TimeIs が検索に出ない（前提が崩れている）")
			}

			txID := GenerateNewID()
			endTime := now.Add(10 * time.Minute)
			updated := original
			updated.EndTime = &endTime
			updated.UpdateTime = now.Add(10 * time.Minute)
			update := postJSON(t, tsURL+"/api/update_timeis", &req_res.UpdateTimeisRequest{SessionID: sessionID, LocaleName: "en", TXID: &txID, TimeIs: updated})
			var updateRes req_res.UpdateTimeisResponse
			if err := json.NewDecoder(update.Body).Decode(&updateRes); err != nil {
				t.Fatalf("decode update timeis response: %v", err)
			}
			update.Body.Close()
			if len(updateRes.Errors) > 0 {
				t.Fatalf("update timeis (tx) errors: %+v", updateRes.Errors)
			}
			discard := postJSON(t, tsURL+"/api/discard_tx", &req_res.DiscardTxRequest{SessionID: sessionID, LocaleName: "en", TXID: txID})
			discard.Body.Close()

			found := kyouIDsInRange(t, tsURL, sessionID, now.Add(-time.Hour), now.Add(time.Hour))
			kyou, ok := found[timeisID]
			if !ok {
				t.Fatal("tx を discard したのに既存の TimeIs が検索から消えた（アドレス表だけが進んでいる）")
			}
			if !kyou.UpdateTime.Equal(now) {
				t.Errorf("既存の TimeIs の版が変わっている: update_time = %v, want %v", kyou.UpdateTime, now)
			}
		})
	}
}
