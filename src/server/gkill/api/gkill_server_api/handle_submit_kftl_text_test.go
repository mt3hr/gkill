package gkill_server_api

import (
	"encoding/json"
	"testing"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/api/req_res"
)

// submitKFTL は /api/submit_kftl_text を叩き、応答を返す。
func submitKFTL(t *testing.T, tsURL, sessionID, kftlText, idempotencyKey string) req_res.SubmitKFTLTextResponse {
	t.Helper()
	resp := postJSON(t, tsURL+"/api/submit_kftl_text", &req_res.SubmitKFTLTextRequest{
		SessionID:      sessionID,
		LocaleName:     "en",
		KFTLText:       kftlText,
		IdempotencyKey: idempotencyKey,
	})
	defer resp.Body.Close()

	res := req_res.SubmitKFTLTextResponse{}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		t.Fatalf("decode submit kftl text response: %v", err)
	}
	return res
}

// countKmemosByContent は本文語句で /api/get_kyous を叩き件数を返す。
func countKmemosByContent(t *testing.T, tsURL, sessionID, word string) int {
	t.Helper()
	res := getKyousWithQuery(t, tsURL, sessionID, &find.FindQuery{Words: []string{word}})
	if len(res.Errors) > 0 {
		t.Fatalf("get kyous errors: %+v", res.Errors)
	}
	return len(res.Kyous)
}

// KFTL送信のサーバ側冪等キー(監査 S3-wear)の end-to-end テスト。
//
// ストア単体(kftl_idempotency_test.go)は markDone を直接呼ぶので通ってしまうが、
// ハンドラが成功後に markDone を配線し忘れると冪等が no-op になる。この壊れ方は
// ビルドも vet も素通しする(変数は alreadyDone 分岐で使われるので未使用にならない)ので、
// 実際に2回叩いて「2回目が畳まれる」ことをここで固定する。
func TestHandleSubmitKFTLText_IdempotencyKey(t *testing.T) {
	tsURL, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()

	sessionID := loginAndGetSession(t, tsURL, gkillAPI, "admin", regressionTestPasswordHash)

	// 同じ冪等キーの再送は1回に畳まれる。
	t.Run("同じ冪等キーの再送は登録が1回に畳まれる", func(t *testing.T) {
		const word = "idemkeyDedupWord"
		res1 := submitKFTL(t, tsURL, sessionID, word, "wear-key-1")
		if len(res1.Errors) > 0 {
			t.Fatalf("1回目でエラー: %+v", res1.Errors)
		}
		if got := countKmemosByContent(t, tsURL, sessionID, word); got != 1 {
			t.Fatalf("1回目の登録後の件数 = %d, want 1", got)
		}

		// 同じキーでもう一度。成功で返るが登録は増えない。
		res2 := submitKFTL(t, tsURL, sessionID, word, "wear-key-1")
		if len(res2.Errors) > 0 {
			t.Fatalf("2回目でエラー: %+v", res2.Errors)
		}
		if len(res2.Messages) == 0 {
			t.Fatal("2回目が成功メッセージを返していない(畳まれた再送も成功で返すべき)")
		}
		if got := countKmemosByContent(t, tsURL, sessionID, word); got != 1 {
			t.Errorf("同じ冪等キーの再送後の件数 = %d, want 1(markDown 未配線なら2になる)", got)
		}
	})

	// 別の冪等キーは意図的な再送として通す(畳まない)。
	t.Run("別の冪等キーは別の登録になる", func(t *testing.T) {
		const word = "idemkeyDistinctWord"
		submitKFTL(t, tsURL, sessionID, word, "wear-key-A")
		submitKFTL(t, tsURL, sessionID, word, "wear-key-B")
		if got := countKmemosByContent(t, tsURL, sessionID, word); got != 2 {
			t.Errorf("別キー2回の後の件数 = %d, want 2", got)
		}
	})

	// 冪等キー無し(空文字)は毎回登録する。
	t.Run("冪等キーが無ければ毎回登録される", func(t *testing.T) {
		const word = "idemkeyNoKeyWord"
		submitKFTL(t, tsURL, sessionID, word, "")
		submitKFTL(t, tsURL, sessionID, word, "")
		if got := countKmemosByContent(t, tsURL, sessionID, word); got != 2 {
			t.Errorf("キー無し2回の後の件数 = %d, want 2", got)
		}
	})
}
