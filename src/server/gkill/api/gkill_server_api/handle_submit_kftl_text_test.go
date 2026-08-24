package gkill_server_api

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/api/req_res"
)

// submitKFTL は /api/submit_kftl_text を叩き、応答を返す。
func submitKFTL(t *testing.T, tsURL, sessionID, kftlText, idempotencyKey string) req_res.SubmitKFTLTextResponse {
	t.Helper()
	_, res := submitKFTLWithStatus(t, tsURL, sessionID, kftlText, idempotencyKey)
	return res
}

// submitKFTLWithStatus は /api/submit_kftl_text を叩き、HTTPステータスと応答を返す。
// 利用者の書き間違い(ERR000416)は 400 で返るため、ステータスも検証対象になる。
func submitKFTLWithStatus(t *testing.T, tsURL, sessionID, kftlText, idempotencyKey string) (int, req_res.SubmitKFTLTextResponse) {
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
	return resp.StatusCode, res
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

// SubmitKFTLTextResponse.Created(2026-08-24 追加)のハンドラ層テスト。
//
// kftl パッケージ側(kftl_statement_test.go)は KFTLCreatedRecord を直接見るが、
// 「応答DTOへ写して返す」配線(toSubmitKFTLTextCreated と response.Created への代入)は
// ハンドラにしかなく、落としてもコンパイルは通る。実際にHTTPで叩いて
// created が応答に載ることをここで固定する。
func TestHandleSubmitKFTLText_CreatedRecords(t *testing.T) {
	tsURL, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()

	sessionID := loginAndGetSession(t, tsURL, gkillAPI, "admin", regressionTestPasswordHash)

	// 1つのテキストから複数種別を書くと、created に種別とIDが全件載る。
	t.Run("複数種別を書くと created に全件が載る", func(t *testing.T) {
		text := "createdMultiMemoWord\n、\nーら\n5\n、\nーん\ncreatedMultiShop\ncreatedMultiItem\n100\n、\nーみ\ncreatedMultiTask"
		status, res := submitKFTLWithStatus(t, tsURL, sessionID, text, "")
		if len(res.Errors) > 0 {
			t.Fatalf("submit kftl text errors: %+v", res.Errors)
		}
		if status != http.StatusOK {
			t.Errorf("status = %d, want 200", status)
		}
		if len(res.Created) != 4 {
			t.Fatalf("created の件数 = %d, want 4: %+v", len(res.Created), res.Created)
		}
		typeCounts := map[string]int{}
		for i, created := range res.Created {
			if created.ID == "" {
				t.Errorf("created[%d] の id が空: %+v", i, created)
			}
			if created.Updated {
				t.Errorf("created[%d] は新規作成なのに updated=true: %+v", i, created)
			}
			typeCounts[created.DataType]++
		}
		for _, dataType := range []string{"kmemo", "lantana", "nlog", "mi"} {
			if typeCounts[dataType] != 1 {
				t.Errorf("data_type %q の件数 = %d, want 1 (created=%+v)", dataType, typeCounts[dataType], res.Created)
			}
		}
	})

	// 打刻の終了は新規作成ではなく既存レコードの更新なので、updated=true と
	// 開始時のIDで返る(リクエストを事前に並べるだけでは分からない情報)。
	t.Run("打刻の終了は updated=true で開始時のIDが載る", func(t *testing.T) {
		started := submitKFTL(t, tsURL, sessionID, "ーた\ncreatedTimeIsEndWork", "")
		if len(started.Errors) > 0 {
			t.Fatalf("打刻開始でエラー: %+v", started.Errors)
		}
		if len(started.Created) != 1 || started.Created[0].DataType != "timeis" {
			t.Fatalf("打刻開始の created = %+v, want timeis 1件", started.Created)
		}
		if started.Created[0].Updated {
			t.Fatalf("打刻開始は新規作成なのに updated=true: %+v", started.Created)
		}

		ended := submitKFTL(t, tsURL, sessionID, "ーえ\ncreatedTimeIsEndWork", "")
		if len(ended.Errors) > 0 {
			t.Fatalf("打刻終了でエラー: %+v", ended.Errors)
		}
		if len(ended.Created) != 1 {
			t.Fatalf("打刻終了の created の件数 = %d, want 1: %+v", len(ended.Created), ended.Created)
		}
		if ended.Created[0].DataType != "timeis" || !ended.Created[0].Updated {
			t.Errorf("打刻終了の created = %+v, want data_type=timeis / updated=true", ended.Created[0])
		}
		if ended.Created[0].ID != started.Created[0].ID {
			t.Errorf("終了した打刻のID = %q, want 開始時のID %q", ended.Created[0].ID, started.Created[0].ID)
		}
	})

	// 実行フェーズの途中失敗では、そこまでに書けたぶんが created に載る。
	// KFTLはDBトランザクションを使わないので部分保存が残る——何が残ったか
	// 分からないと利用者は後始末ができない。
	t.Run("途中失敗でも書けたぶんは created に載る", func(t *testing.T) {
		const word = "createdPartialMemoWord"
		// kmemo は先に書けるが、実在しない打刻の終了(ーえ)が実行フェーズで失敗する。
		status, res := submitKFTLWithStatus(t, tsURL, sessionID, word+"\n、\nーえ\ncreatedPartialNoSuchTimeIs", "")
		if status != http.StatusBadRequest {
			t.Errorf("status = %d, want 400 (終了対象なしは利用者入力の問題)", status)
		}
		if len(res.Errors) != 1 || res.Errors[0].ErrorCode != message.SubmitKFTLTextInvalidInputError {
			t.Fatalf("errors = %+v, want %s 1件", res.Errors, message.SubmitKFTLTextInvalidInputError)
		}
		if len(res.Created) != 1 || res.Created[0].DataType != "kmemo" {
			t.Errorf("失敗前に書けた kmemo が created に載っていない: %+v", res.Created)
		}
		if got := countKmemosByContent(t, tsURL, sessionID, word); got != 1 {
			t.Errorf("部分保存された kmemo の件数 = %d, want 1(ロールバックはされない)", got)
		}
	})

	// 冪等キーで畳んだ再送は再実行していないので created は空。
	t.Run("冪等キーで畳んだ再送は created が空", func(t *testing.T) {
		const word = "createdIdemResendWord"
		res1 := submitKFTL(t, tsURL, sessionID, word, "created-resend-key-1")
		if len(res1.Errors) > 0 {
			t.Fatalf("1回目でエラー: %+v", res1.Errors)
		}
		if len(res1.Created) != 1 {
			t.Fatalf("1回目の created の件数 = %d, want 1: %+v", len(res1.Created), res1.Created)
		}

		res2 := submitKFTL(t, tsURL, sessionID, word, "created-resend-key-1")
		if len(res2.Errors) > 0 {
			t.Fatalf("2回目でエラー: %+v", res2.Errors)
		}
		if len(res2.Messages) == 0 {
			t.Fatal("畳まれた再送も成功で返すべき")
		}
		if len(res2.Created) != 0 {
			t.Errorf("畳んだ再送の created = %+v, want 空(再実行していないので何も書いていない)", res2.Created)
		}
	})
}

// 入力ミス行を含む本文の応答のハンドラ層テスト(2026-08-24 の再監査対応の固定)。
//
// 2026-08-24 までは気分値の打ち間違いもDB障害も同じ ERR000351(500) + 定型文1本に
// 畳まれていた。kftl パッケージ側は行番号つきで全行ぶん集めることを固定済みだが、
// 「HTTP 400 へ落とす・不正行ごとに ERR000416 を積む・多言語メッセージへ行情報を挟む」
// 配線はハンドラ(CollectKFTLInputErrors の分岐と formatKFTLInputErrorMessage)にしかない。
func TestHandleSubmitKFTLText_InvalidInputLines(t *testing.T) {
	tsURL, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()

	sessionID := loginAndGetSession(t, tsURL, gkillAPI, "admin", regressionTestPasswordHash)

	// 4行目(範囲外の気分値)と7行目(数値でない気分値)の2箇所が不正。
	const word = "invalidInputMemoWord"
	status, res := submitKFTLWithStatus(t, tsURL, sessionID, word+"\n、\n/mood\n99\n、\n/mood\nabc", "")

	if status != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 (利用者の書き間違いはサーバ障害と分ける)", status)
	}
	if len(res.Errors) != 2 {
		t.Fatalf("errors の件数 = %d, want 2 (不正行ごとに1件): %+v", len(res.Errors), res.Errors)
	}
	for i, gkillError := range res.Errors {
		if gkillError.ErrorCode != message.SubmitKFTLTextInvalidInputError {
			t.Errorf("errors[%d].ErrorCode = %s, want %s", i, gkillError.ErrorCode, message.SubmitKFTLTextInvalidInputError)
		}
	}
	// 行番号と行テキストが挟まっていて、何行目の何が悪いのか応答だけで分かる。
	if msg := res.Errors[0].ErrorMessage; !strings.Contains(msg, `(line 4: "99")`) {
		t.Errorf("1件目のメッセージに行情報が無い: %q", msg)
	}
	if msg := res.Errors[1].ErrorMessage; !strings.Contains(msg, `(line 7: "abc")`) {
		t.Errorf("2件目のメッセージに行情報が無い: %q", msg)
	}
	// MessageID から locale(en) の文言が引き当てられている(定型文への畳み込みではない)。
	if msg := res.Errors[0].ErrorMessage; !strings.Contains(msg, "Mood value is out of range") {
		t.Errorf("1件目のメッセージが多言語文言を引いていない: %q", msg)
	}
	if msg := res.Errors[1].ErrorMessage; !strings.Contains(msg, "Mood value is invalid") {
		t.Errorf("2件目のメッセージが多言語文言を引いていない: %q", msg)
	}
	// 行の解釈フェーズの失敗は1バイトも書く前なので、created は空で、
	// 同じ本文の正しい行(kmemo)も保存されない。
	if len(res.Created) != 0 {
		t.Errorf("created = %+v, want 空(書き込み前に失敗している)", res.Created)
	}
	if got := countKmemosByContent(t, tsURL, sessionID, word); got != 0 {
		t.Errorf("不正行を含む本文の正しい行が保存されている: 件数 = %d, want 0", got)
	}
}
