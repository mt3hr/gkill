package gkill_server_api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/api/req_res"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
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
	return submitKFTLWithCreateApp(t, tsURL, sessionID, kftlText, idempotencyKey, "")
}

// submitKFTLWithCreateApp は create_app を指定して /api/submit_kftl_text を叩く。
// 空なら要求から create_app キーが落ちる(omitempty)ので「無指定」の経路になる。
func submitKFTLWithCreateApp(t *testing.T, tsURL, sessionID, kftlText, idempotencyKey, createApp string) (int, req_res.SubmitKFTLTextResponse) {
	t.Helper()
	resp := postJSON(t, tsURL+"/api/submit_kftl_text", &req_res.SubmitKFTLTextRequest{
		SessionID:      sessionID,
		LocaleName:     "en",
		KFTLText:       kftlText,
		IdempotencyKey: idempotencyKey,
		CreateApp:      createApp,
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

// KFTL送信のサーバ側冪等キー(指摘 S3-wear)の end-to-end テスト。
//
// ストア単体(kftl_idempotency_test.go)は markDone を直接呼ぶので通ってしまうが、
// ハンドラが成功後に markDone を配線し忘れると冪等が no-op になる。この壊れ方は
// ビルドも vet も素通しする(変数は alreadyDone 分岐で使われるので未使用にならない)ので、
// 実際に2回叩いて「2回目が畳まれる」ことをここで固定する。
func TestHandleSubmitKFTLText_IdempotencyKey(t *testing.T) {
	tsURL, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()

	sessionID := loginAndGetSession(t, tsURL, gkillAPI, "admin", regressionTestPasswordHash)

	// 同じ冪等キー・同じ本文の再送は1回に畳まれ、元の created[] が replayed:true で返る。
	t.Run("同じ冪等キーの再送は登録が1回に畳まれ、元の created を replayed:true で返す", func(t *testing.T) {
		const word = "idemkeyDedupWord"
		res1 := submitKFTL(t, tsURL, sessionID, word, "wear-key-1")
		if len(res1.Errors) > 0 {
			t.Fatalf("1回目でエラー: %+v", res1.Errors)
		}
		if res1.Replayed {
			t.Error("1回目の送信が replayed=true になっている")
		}
		if len(res1.Created) != 1 {
			t.Fatalf("1回目の created = %+v, want 1件", res1.Created)
		}
		if got := countKmemosByContent(t, tsURL, sessionID, word); got != 1 {
			t.Fatalf("1回目の登録後の件数 = %d, want 1", got)
		}

		// 同じキー・同じ本文でもう一度。成功で返るが登録は増えず、created は1回目の控え。
		res2 := submitKFTL(t, tsURL, sessionID, word, "wear-key-1")
		if len(res2.Errors) > 0 {
			t.Fatalf("2回目でエラー: %+v", res2.Errors)
		}
		if len(res2.Messages) == 0 {
			t.Fatal("2回目が成功メッセージを返していない(畳まれた再送も成功で返すべき)")
		}
		if !res2.Replayed {
			t.Error("畳んだ再送が replayed=true でない（呼び出し側が「今回書いた」と誤読する）")
		}
		if len(res2.Created) != 1 || res2.Created[0].ID != res1.Created[0].ID || res2.Created[0].DataType != res1.Created[0].DataType {
			t.Errorf("畳んだ再送の created = %+v, want 1回目と同じ %+v（応答を失った呼び出し側が ID を回収する経路）", res2.Created, res1.Created)
		}
		if got := countKmemosByContent(t, tsURL, sessionID, word); got != 1 {
			t.Errorf("同じ冪等キーの再送後の件数 = %d, want 1(markDone 未配線なら2になる)", got)
		}
	})

	// 同じ冪等キーで別の本文は衝突。何も書かず 409 で返す（2026-09-19 まで「成功・created 空」で返し、
	// 保存されていない本文にも成功メッセージが返っていた）。
	t.Run("同じ冪等キーで別の本文は 409 で何も書かない", func(t *testing.T) {
		const wordA, wordB = "idemkeyConflictWordA", "idemkeyConflictWordB"
		res1 := submitKFTL(t, tsURL, sessionID, wordA, "wear-key-conflict")
		if len(res1.Errors) > 0 {
			t.Fatalf("1回目でエラー: %+v", res1.Errors)
		}

		status, res2 := submitKFTLWithStatus(t, tsURL, sessionID, wordB, "wear-key-conflict")
		if status != http.StatusConflict {
			t.Errorf("status = %d, want 409", status)
		}
		if len(res2.Errors) != 1 || res2.Errors[0].ErrorCode != message.SubmitKFTLTextIdempotencyKeyConflictError {
			t.Fatalf("errors = %+v, want %s 1件", res2.Errors, message.SubmitKFTLTextIdempotencyKeyConflictError)
		}
		if res2.Replayed || len(res2.Created) != 0 {
			t.Errorf("衝突した送信の replayed / created = %v / %+v, want false / 空", res2.Replayed, res2.Created)
		}
		if got := countKmemosByContent(t, tsURL, sessionID, wordB); got != 0 {
			t.Errorf("衝突した本文が書かれている: %d 件, want 0", got)
		}
		if got := countKmemosByContent(t, tsURL, sessionID, wordA); got != 1 {
			t.Errorf("元の本文の件数 = %d, want 1（衝突が元の記録に影響してはいけない）", got)
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

	// 実行フェーズの途中失敗では何も残らず、created は空になる。
	// 2026-09-15 まで KFTL は実 rep へ直書きで、失敗した行より前の kmemo が残り、
	// 利用者が created[] を見て後始末する設計だった。今は temp rep に積んで最後に
	// 1つの SQLite トランザクションで確定するので、失敗したら**何も残らない**。
	t.Run("途中失敗では何も残らず created は空", func(t *testing.T) {
		const word = "createdPartialMemoWord"
		// kmemo は先に積まれるが、実在しない打刻の終了(ーえ)が実行フェーズで失敗する。
		status, res := submitKFTLWithStatus(t, tsURL, sessionID, word+"\n、\nーえ\ncreatedPartialNoSuchTimeIs", "")
		if status != http.StatusBadRequest {
			t.Errorf("status = %d, want 400 (終了対象なしは利用者入力の問題)", status)
		}
		if len(res.Errors) != 1 || res.Errors[0].ErrorCode != message.SubmitKFTLTextInvalidInputError {
			t.Fatalf("errors = %+v, want %s 1件", res.Errors, message.SubmitKFTLTextInvalidInputError)
		}
		if len(res.Created) != 0 {
			t.Errorf("失敗した送信の created = %+v, want 空(何も残っていない)", res.Created)
		}
		if got := countKmemosByContent(t, tsURL, sessionID, word); got != 0 {
			t.Errorf("失敗した送信の kmemo が残っている: %d 件, want 0(部分確定)", got)
		}
	})

	// 冪等キーで畳んだ再送が元の created を replayed:true で返すことは
	// TestHandleSubmitKFTLText_IdempotencyKey の1つ目のサブテストが固定している（ここには重ねない）。

	// related_time は保存層と同じ秒精度。丸めないと Windows の時計解像度で 7 桁の小数秒が
	// 応答にだけ載り、保存値と一致しない（2026-09-18 の MCP 実利用報告）。
	t.Run("created の related_time は秒精度", func(t *testing.T) {
		res := submitKFTL(t, tsURL, sessionID, "createdSecondPrecisionWord", "")
		if len(res.Errors) > 0 || len(res.Created) != 1 {
			t.Fatalf("errors=%+v created=%+v", res.Errors, res.Created)
		}
		if res.Created[0].RelatedTime.Nanosecond() != 0 {
			t.Errorf("related_time = %s, want 秒精度（小数秒なし）", res.Created[0].RelatedTime.Format(time.RFC3339Nano))
		}
	})
}

// 失敗時の created は JSON で `[]`（`null` ではない）。
// 説明文は「失敗時は created[] が空」と言っているのに、2026-09-19 まで実応答は `"created": null` だった。
func TestHandleSubmitKFTLText_CreatedIsEmptyArrayOnFailure(t *testing.T) {
	tsURL, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()

	sessionID := loginAndGetSession(t, tsURL, gkillAPI, "admin", regressionTestPasswordHash)

	resp := postJSON(t, tsURL+"/api/submit_kftl_text", &req_res.SubmitKFTLTextRequest{
		SessionID:  sessionID,
		LocaleName: "en",
		KFTLText:   "/mood\n99",
	})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
	var raw map[string]json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		t.Fatalf("decode raw response: %v", err)
	}
	if got := strings.TrimSpace(string(raw["created"])); got != "[]" {
		t.Errorf("失敗時の created = %s, want [] (null ではない)", got)
	}
}

// KFTL の `ーみ` で書いたタスクの create_device / create_user が入れ替わらないこと（2026-09-19）。
//
// kftl パッケージは正しく Ctx.Device / Ctx.UserID を入れていたが、Mi の temp rep の
// GetMisByTXID が SELECT を表の列順（CREATE_USER, CREATE_DEVICE）で書き、Scan は
// (CreateDevice, CreateUser) の順で受けていたため、temp rep → CommitTx を通る KFTL の Mi だけ
// 利用者名が端末名として確定していた（Web の add_mi は temp rep を通らないので正常）。
// 同じ送信の kmemo（temp rep の列順が正しい）と突き合わせることで、端末名の実値を知らずに固定する。
func TestHandleSubmitKFTLText_MiAuditFieldsAreNotSwapped(t *testing.T) {
	tsURL, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()

	sessionID := loginAndGetSession(t, tsURL, gkillAPI, "admin", regressionTestPasswordHash)

	res := submitKFTL(t, tsURL, sessionID, "auditSwapMemoWord\n、\nーみ\nauditSwapTaskTitle", "")
	if len(res.Errors) > 0 {
		t.Fatalf("submit kftl text errors: %+v", res.Errors)
	}
	var kmemoID, miID string
	for _, created := range res.Created {
		switch created.DataType {
		case "kmemo":
			kmemoID = created.ID
		case "mi":
			miID = created.ID
		}
	}
	if kmemoID == "" || miID == "" {
		t.Fatalf("created = %+v, want kmemo と mi が1件ずつ", res.Created)
	}

	kmemoResp := postJSON(t, tsURL+"/api/get_kmemo", &req_res.GetKmemoRequest{SessionID: sessionID, ID: kmemoID, LocaleName: "en"})
	defer kmemoResp.Body.Close()
	var kmemoRes req_res.GetKmemoResponse
	if err := json.NewDecoder(kmemoResp.Body).Decode(&kmemoRes); err != nil {
		t.Fatalf("decode get kmemo response: %v", err)
	}
	if len(kmemoRes.Errors) > 0 || len(kmemoRes.KmemoHistories) == 0 {
		t.Fatalf("get kmemo: errors=%+v histories=%d", kmemoRes.Errors, len(kmemoRes.KmemoHistories))
	}
	miResp := postJSON(t, tsURL+"/api/get_mi", &req_res.GetMiRequest{SessionID: sessionID, ID: miID, LocaleName: "en"})
	defer miResp.Body.Close()
	var miRes req_res.GetMiResponse
	if err := json.NewDecoder(miResp.Body).Decode(&miRes); err != nil {
		t.Fatalf("decode get mi response: %v", err)
	}
	if len(miRes.Errors) > 0 || len(miRes.MiHistories) == 0 {
		t.Fatalf("get mi: errors=%+v histories=%d", miRes.Errors, len(miRes.MiHistories))
	}

	kmemo, mi := kmemoRes.KmemoHistories[0], miRes.MiHistories[0]
	if kmemo.CreateUser != "admin" {
		t.Fatalf("kmemo の create_user = %q, want admin（比較の基準が壊れている）", kmemo.CreateUser)
	}
	if mi.CreateUser != kmemo.CreateUser {
		t.Errorf("mi の create_user = %q, want %q（kmemo と同じ利用者）", mi.CreateUser, kmemo.CreateUser)
	}
	if mi.CreateDevice != kmemo.CreateDevice {
		t.Errorf("mi の create_device = %q, want %q（kmemo と同じ端末。利用者名が入っていれば temp rep の列順がずれている）", mi.CreateDevice, kmemo.CreateDevice)
	}
	if mi.UpdateUser != kmemo.UpdateUser || mi.UpdateDevice != kmemo.UpdateDevice {
		t.Errorf("mi の update_user / update_device = %q / %q, want %q / %q", mi.UpdateUser, mi.UpdateDevice, kmemo.UpdateUser, kmemo.UpdateDevice)
	}
}

// 入力ミス行を含む本文の応答のハンドラ層テスト(2巡目の指摘への対応の固定)。
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

// 繰り返し「？？」で実際に書き込まれる時刻の年チェック（Wear / MCP が通る Go 経路）。
//
// 2026-09-10 に Web の打刻（ーち）が 2083 年で登録された事故は、kftl パッケージ側の
// テストが「件数」しか見ていなかったので素通りした。ここでは HTTP で送って型別 API で
// 引き直し、**書き込まれた値**を年まで見る。時刻のみの行は送信日の日付で補完されるが、
// 起点 2026-09-07 から暦日数でずらすので、いつ実行しても同じ日付に着地する。
func TestHandleSubmitKFTLText_RepeatWritesShiftedTimes(t *testing.T) {
	tsURL, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()

	sessionID := loginAndGetSession(t, tsURL, gkillAPI, "admin", regressionTestPasswordHash)

	t.Run("打刻の開始と終了が起点からの日付で書かれ、年が変わらない", func(t *testing.T) {
		text := "ーち\nrepeatWiredWork\n08:30\n17:30\n？？\n毎日\n3\n\n2026-09-07\n？？"
		res := submitKFTL(t, tsURL, sessionID, text, "")
		if len(res.Errors) > 0 {
			t.Fatalf("submit kftl text errors: %+v", res.Errors)
		}
		if len(res.Created) != 3 {
			t.Fatalf("created の件数 = %d, want 3: %+v", len(res.Created), res.Created)
		}
		gotStarts := map[int64]bool{}
		for i, created := range res.Created {
			if created.DataType != "timeis" {
				t.Fatalf("created[%d].DataType = %q, want timeis", i, created.DataType)
			}
			resp := postJSON(t, tsURL+"/api/get_timeis", &req_res.GetTimeisRequest{SessionID: sessionID, ID: created.ID, LocaleName: "en"})
			var got req_res.GetTimeisResponse
			if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
				t.Fatalf("decode get timeis response: %v", err)
			}
			resp.Body.Close()
			if len(got.Errors) > 0 || len(got.TimeisHistories) == 0 {
				t.Fatalf("get timeis id=%s: errors=%+v histories=%d", created.ID, got.Errors, len(got.TimeisHistories))
			}
			timeIs := got.TimeisHistories[0]
			if timeIs.StartTime.Year() != 2026 {
				t.Errorf("[%d] 開始の年 = %d, want 2026 (2083 年の再発シグネチャ)", i, timeIs.StartTime.Year())
			}
			if timeIs.EndTime == nil {
				t.Fatalf("[%d] 終了が無い", i)
			}
			wantEnd := time.Date(2026, 9, timeIs.StartTime.Day(), 17, 30, 0, 0, time.Local)
			if !timeIs.EndTime.Equal(wantEnd) {
				t.Errorf("[%d] 終了 = %v, want %v (開始と同じ日数だけずれる)", i, timeIs.EndTime, wantEnd)
			}
			gotStarts[timeIs.StartTime.Unix()] = true
		}
		for day := 7; day <= 9; day++ {
			want := time.Date(2026, 9, day, 8, 30, 0, 0, time.Local)
			if !gotStarts[want.Unix()] {
				t.Errorf("開始 %v が書かれていない (書かれた開始: %v)", want, gotStarts)
			}
		}
	})

	// 支出ブロックの `？`行の時刻はタグにも乗る。doBaseRequest が埋め込み基底の GetRelatedTime を
	// 引いていた頃は Nlog の override（ブロック共有の時刻）が効かず、タグだけ「今」で書かれていた
	// （TS は override に届くので Web と Wear / MCP で結果が違っていた）。
	t.Run("支出の関連時刻がタグにも乗る", func(t *testing.T) {
		text := "ーん\nrepeatWiredShop\nrepeatWiredItem\n200\n。repeatWiredTag\n？2026-09-07 12:00"
		res := submitKFTL(t, tsURL, sessionID, text, "")
		if len(res.Errors) > 0 {
			t.Fatalf("submit kftl text errors: %+v", res.Errors)
		}
		if len(res.Created) != 1 || res.Created[0].DataType != "nlog" {
			t.Fatalf("created = %+v, want nlog 1件", res.Created)
		}
		resp := postJSON(t, tsURL+"/api/get_tags_by_id", &req_res.GetTagsByTargetIDRequest{SessionID: sessionID, TargetID: res.Created[0].ID, LocaleName: "en"})
		defer resp.Body.Close()
		var got req_res.GetTagsByTargetIDResponse
		if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
			t.Fatalf("decode get tags response: %v", err)
		}
		if len(got.Errors) > 0 || len(got.Tags) != 1 {
			t.Fatalf("get tags: errors=%+v tags=%+v, want 1件", got.Errors, got.Tags)
		}
		want := time.Date(2026, 9, 7, 12, 0, 0, 0, time.Local)
		if !got.Tags[0].RelatedTime.Equal(want) {
			t.Errorf("タグの関連時刻 = %v, want %v (ブロックの `？`行の時刻。基底で引くと「今」になる)", got.Tags[0].RelatedTime, want)
		}
	})
}

// getKmemoApps は /api/get_kmemo で最新版を引き、書き込まれた create_app / update_app を返す。
func getKmemoApps(t *testing.T, tsURL, sessionID, id string) (string, string) {
	t.Helper()
	resp := postJSON(t, tsURL+"/api/get_kmemo", &req_res.GetKmemoRequest{SessionID: sessionID, ID: id, LocaleName: "en"})
	defer resp.Body.Close()
	var got req_res.GetKmemoResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode get kmemo response: %v", err)
	}
	if len(got.Errors) > 0 || len(got.KmemoHistories) == 0 {
		t.Fatalf("get kmemo id=%s: errors=%+v histories=%d", id, got.Errors, len(got.KmemoHistories))
	}
	return got.KmemoHistories[0].CreateApp, got.KmemoHistories[0].UpdateApp
}

// 要求の create_app が書き込まれた記録の create_app / update_app に載ること（2026-09-11）。
//
// Wear companion は "gkill_wear" を送って手打ちのメモ帳と区別する。2026-09-11 までは
// ハンドラが "gkill_kftl" を固定で渡していて、ウォッチからつけた記録を後から絞る手段が無かった。
// 無指定（MCP と旧 companion）は従来どおり "gkill_kftl" に落ちる —— ここが崩れると
// 既存の記録と新しい記録の値が割れるので、既定値も同時に固定する。
func TestHandleSubmitKFTLText_CreateApp(t *testing.T) {
	tsURL, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()

	sessionID := loginAndGetSession(t, tsURL, gkillAPI, "admin", regressionTestPasswordHash)

	t.Run("create_app を指定するとその値で書かれる", func(t *testing.T) {
		_, res := submitKFTLWithCreateApp(t, tsURL, sessionID, "createAppWearWord", "", "gkill_wear")
		if len(res.Errors) > 0 {
			t.Fatalf("submit kftl text errors: %+v", res.Errors)
		}
		if len(res.Created) != 1 || res.Created[0].DataType != "kmemo" {
			t.Fatalf("created = %+v, want kmemo 1件", res.Created)
		}
		createApp, updateApp := getKmemoApps(t, tsURL, sessionID, res.Created[0].ID)
		if createApp != "gkill_wear" {
			t.Errorf("create_app = %q, want gkill_wear", createApp)
		}
		if updateApp != "gkill_wear" {
			t.Errorf("update_app = %q, want gkill_wear", updateApp)
		}
	})

	t.Run("create_app が無ければ従来どおり gkill_kftl になる", func(t *testing.T) {
		_, res := submitKFTLWithCreateApp(t, tsURL, sessionID, "createAppDefaultWord", "", "")
		if len(res.Errors) > 0 {
			t.Fatalf("submit kftl text errors: %+v", res.Errors)
		}
		if len(res.Created) != 1 || res.Created[0].DataType != "kmemo" {
			t.Fatalf("created = %+v, want kmemo 1件", res.Created)
		}
		createApp, updateApp := getKmemoApps(t, tsURL, sessionID, res.Created[0].ID)
		if createApp != "gkill_kftl" {
			t.Errorf("create_app = %q, want gkill_kftl (無指定の既定値)", createApp)
		}
		if updateApp != "gkill_kftl" {
			t.Errorf("update_app = %q, want gkill_kftl (無指定の既定値)", updateApp)
		}
	})

	// 空白だけの指定は無指定と同じ扱い。" " のような値が create_app に書かれると
	// 絞り込みで見つからない記録になる。
	t.Run("空白だけの create_app は無指定と同じ", func(t *testing.T) {
		_, res := submitKFTLWithCreateApp(t, tsURL, sessionID, "createAppBlankWord", "", "  ")
		if len(res.Errors) > 0 {
			t.Fatalf("submit kftl text errors: %+v", res.Errors)
		}
		if len(res.Created) != 1 {
			t.Fatalf("created = %+v, want 1件", res.Created)
		}
		createApp, _ := getKmemoApps(t, tsURL, sessionID, res.Created[0].ID)
		if createApp != "gkill_kftl" {
			t.Errorf("create_app = %q, want gkill_kftl (空白は無指定扱い)", createApp)
		}
	})
}

// countKyousByDataType は全件検索して data_type が前方一致する Kyou の件数を返す
// （打刻は "timeis_start" / "timeis_end" の2種で出るので前方一致）。
func countKyousByDataType(t *testing.T, tsURL, sessionID, dataType string) int {
	t.Helper()
	res := getKyousWithQuery(t, tsURL, sessionID, &find.FindQuery{})
	if len(res.Errors) > 0 {
		t.Fatalf("get kyous errors: %+v", res.Errors)
	}
	count := 0
	for _, kyou := range res.Kyous {
		if strings.HasPrefix(kyou.DataType, dataType) {
			count++
		}
	}
	return count
}

// 保存マーカー「！」で保存したときの、内容の無い記録（ADR-0508）。
//
// Web のメモ帳はマーカー行を含めたまま送る。2026-09-15 まで generateKFTLLines がマーカー行を
// 「次の行」に入れたまま break していたので、`ーち`+「！」は requireNextLineText を素通りして
// タイトル空のまま DoRequest に届き、200「保存しました」でタブが閉じて何も残らなかった
// （`ーら`+「！」は**気分値0が1件書かれ**、`ーか`+「！」は 500 だった）。
// 利用者の報告（「`ーち` で内容を入力しなくてもエラーにならない」）の再現そのもの。
func TestHandleSubmitKFTLText_SaveMarkerAfterBarePrefixIsInputError(t *testing.T) {
	tsURL, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()

	sessionID := loginAndGetSession(t, tsURL, gkillAPI, "admin", regressionTestPasswordHash)

	t.Run("ーち + マーカーは parse でも submit でも行1の入力エラーで、何も書かれない", func(t *testing.T) {
		before := countKyousByDataType(t, tsURL, sessionID, "timeis")

		pStatus, pRes := parseKFTL(t, tsURL, sessionID, "ーち\n！\n")
		if pStatus != http.StatusOK || len(pRes.InvalidLines) != 1 {
			t.Fatalf("parse: status = %d, invalid_lines = %+v, want 200 と 1件", pStatus, pRes.InvalidLines)
		}
		if got := pRes.InvalidLines[0]; got.LineNumber != 1 || got.LineText != "ーち" || !strings.Contains(got.Message, "Write the value on the next line") {
			t.Errorf("parse invalid_lines[0] = %+v, want 行1 ーち + 値の行の文言", got)
		}

		sStatus, sRes := submitKFTLWithStatus(t, tsURL, sessionID, "ーち\n！\n", "marker-timeis")
		if sStatus != http.StatusBadRequest {
			t.Errorf("submit: status = %d, want 400（2026-09-15 までは 200 で「保存しました」だった）", sStatus)
		}
		if len(sRes.Errors) != 1 || sRes.Errors[0].ErrorCode != message.SubmitKFTLTextInvalidInputError {
			t.Fatalf("submit: errors = %+v, want ERR000416 1件", sRes.Errors)
		}
		if !strings.Contains(sRes.Errors[0].ErrorMessage, `(line 1: "ーち")`) {
			t.Errorf("submit: error_message = %q, want 行1の行情報", sRes.Errors[0].ErrorMessage)
		}
		if len(sRes.Created) != 0 {
			t.Errorf("submit: created = %+v, want 空", sRes.Created)
		}
		if after := countKyousByDataType(t, tsURL, sessionID, "timeis"); after != before {
			t.Errorf("打刻の件数が %d → %d に増えた（タイトル空の打刻が書かれている）", before, after)
		}
	})

	t.Run("ーら + マーカーで気分値0が書かれない", func(t *testing.T) {
		before := countKyousByDataType(t, tsURL, sessionID, "lantana")
		sStatus, sRes := submitKFTLWithStatus(t, tsURL, sessionID, "ーら\n！\n", "marker-lantana")
		if sStatus != http.StatusBadRequest || len(sRes.Created) != 0 {
			t.Errorf("status = %d, created = %+v, want 400 と 空", sStatus, sRes.Created)
		}
		if after := countKyousByDataType(t, tsURL, sessionID, "lantana"); after != before {
			t.Errorf("気分記録の件数が %d → %d に増えた（気分値0が書かれている。ADR-0503 の事故の再発）", before, after)
		}
	})

	t.Run("ーか + マーカーは 500 ではなく行別の 400", func(t *testing.T) {
		sStatus, sRes := submitKFTLWithStatus(t, tsURL, sessionID, "ーか\n！\n", "marker-kc")
		if sStatus != http.StatusBadRequest {
			t.Errorf("status = %d, want 400（2026-09-15 までは rep の生 error で 500 だった）", sStatus)
		}
		if len(sRes.Errors) != 1 || sRes.Errors[0].ErrorCode != message.SubmitKFTLTextInvalidInputError {
			t.Errorf("errors = %+v, want ERR000416 1件", sRes.Errors)
		}
	})

	t.Run("値があれば今までどおり保存される", func(t *testing.T) {
		before := countKyousByDataType(t, tsURL, sessionID, "timeis")
		sStatus, sRes := submitKFTLWithStatus(t, tsURL, sessionID, "ーち\nmarkerOkTitle\n！\n", "marker-ok")
		if sStatus != http.StatusOK || len(sRes.Errors) != 0 || len(sRes.Created) != 1 {
			t.Fatalf("status = %d, errors = %+v, created = %+v", sStatus, sRes.Errors, sRes.Created)
		}
		if after := countKyousByDataType(t, tsURL, sessionID, "timeis"); after != before+1 {
			t.Errorf("打刻の件数 = %d, want %d", after, before+1)
		}
	})
}

// 打刻終了（ーえ / ーたえ 系）の**対象の決め方**のハンドラ層テスト（利用者報告。ADR-0509）。
//
// 2026-09-15 の Go 寄せ（ADR-0507）まで Web は get_kyous の並び（開始時刻降順・削除済み除外）に乗って
// 常に最新の1件を終えていたが、Go は TimeIsReps.FindTimeIs の不定順（map 由来）の先頭を取っていた。
// 同じタグを持つ終え忘れが N 件あると、いま走っている1件に当たる確率が 1/N で、削除済みの打刻に
// 終了を書くこともあった。既存の TimeIsEndByTagIfExist_WithMatch は「エラーが出ない」しか見ておらず、
// 実行中が1件しか無い環境では不定順でも当たるので捕まらなかった。ここでは**どの打刻が終わったか**を見る。
// cache_in_memory の ON/OFF で FindTimeIs の経路が変わるので両方で回す。
func TestHandleSubmitKFTLText_TimeIsEndTargetsLatestRunning(t *testing.T) {
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

			// addRunningTimeIs は終了時刻の無い打刻を1件足し、その ID を返す。
			addRunningTimeIs := func(t *testing.T, title string, start time.Time) string {
				t.Helper()
				id := GenerateNewID()
				resp := postJSON(t, tsURL+"/api/add_timeis", &req_res.AddTimeIsRequest{
					SessionID: sessionID, LocaleName: "en",
					TimeIs: reps.TimeIs{
						ID: id, Title: title, StartTime: start, EndTime: nil, DataType: "timeis",
						CreateTime: start, CreateApp: "test", CreateUser: "admin",
						UpdateTime: start, UpdateApp: "test", UpdateUser: "admin",
					},
				})
				resp.Body.Close()
				return id
			}
			addTag := func(t *testing.T, targetID, tag string, relatedTime time.Time) {
				t.Helper()
				resp := postJSON(t, tsURL+"/api/add_tag", &req_res.AddTagRequest{
					SessionID: sessionID, LocaleName: "en",
					Tag: reps.Tag{
						ID: GenerateNewID(), TargetID: targetID, Tag: tag, RelatedTime: relatedTime,
						CreateTime: relatedTime, CreateApp: "test", CreateUser: "admin",
						UpdateTime: relatedTime, UpdateApp: "test", UpdateUser: "admin",
					},
				})
				resp.Body.Close()
			}
			// markDeleted は打刻を削除済みにする（版を1つ足す。終了時刻は無いまま）。
			markDeleted := func(t *testing.T, id, title string, start time.Time) {
				t.Helper()
				resp := postJSON(t, tsURL+"/api/update_timeis", &req_res.UpdateTimeisRequest{
					SessionID: sessionID, LocaleName: "en",
					TimeIs: reps.TimeIs{
						ID: id, Title: title, StartTime: start, EndTime: nil, DataType: "timeis", IsDeleted: true,
						CreateTime: start, CreateApp: "test", CreateUser: "admin",
						UpdateTime: start.Add(time.Second), UpdateApp: "test", UpdateUser: "admin",
					},
				})
				resp.Body.Close()
			}
			// latestTimeIs は最新版を返す。
			latestTimeIs := func(t *testing.T, id string) reps.TimeIs {
				t.Helper()
				resp := postJSON(t, tsURL+"/api/get_timeis", &req_res.GetTimeisRequest{SessionID: sessionID, LocaleName: "en", ID: id})
				defer resp.Body.Close()
				var res req_res.GetTimeisResponse
				if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
					t.Fatalf("decode get timeis response: %v", err)
				}
				if len(res.Errors) != 0 || len(res.TimeisHistories) == 0 {
					t.Fatalf("get timeis %s: errors=%+v histories=%d", id, res.Errors, len(res.TimeisHistories))
				}
				latest := res.TimeisHistories[0]
				for _, history := range res.TimeisHistories[1:] {
					if history.UpdateTime.After(latest.UpdateTime) {
						latest = history
					}
				}
				return latest
			}
			// countTagRowsByName は Tag rep に実際にある行数を名前で数える（Kyou の無い ID に付いた Tag は get_kyous では見えない）。
			countTagRowsByName := func(t *testing.T, tagName string) int {
				t.Helper()
				device, err := gkillAPI.GetDevice()
				if err != nil {
					t.Fatalf("GetDevice: %v", err)
				}
				repositories, err := gkillAPI.GkillDAOManager.GetRepositories("admin", device)
				if err != nil {
					t.Fatalf("GetRepositories: %v", err)
				}
				tags, err := repositories.TagReps.FindTags(context.Background(), &find.FindQuery{Words: []string{tagName}, OnlyLatestData: true})
				if err != nil {
					t.Fatalf("FindTags: %v", err)
				}
				count := 0
				for _, found := range tags {
					if found.Tag == tagName && !found.IsDeleted {
						count++
					}
				}
				return count
			}
			// endTimeOf は終了時刻を "nil" か秒精度の文字列で返す（比較用）。
			endTimeOf := func(timeis reps.TimeIs) string {
				if timeis.EndTime == nil {
					return "nil"
				}
				return timeis.EndTime.Truncate(time.Second).Format(time.RFC3339)
			}

			t.Run("ーたえ は同じタグの実行中のうち開始時刻が最新の1件だけを終える", func(t *testing.T) {
				tag := "endtag_latest_" + GenerateNewID()[:8]
				oldest := addRunningTimeIs(t, "endByTagOldest", now.Add(-3*time.Hour))
				middle := addRunningTimeIs(t, "endByTagMiddle", now.Add(-2*time.Hour))
				newest := addRunningTimeIs(t, "endByTagNewest", now.Add(-1*time.Hour))
				for _, id := range []string{oldest, middle, newest} {
					addTag(t, id, tag, now)
				}

				res := submitKFTL(t, tsURL, sessionID, "ーたえ\n"+tag, "")
				if len(res.Errors) != 0 {
					t.Fatalf("ーたえ でエラー: %+v", res.Errors)
				}
				if len(res.Created) != 1 || res.Created[0].ID != newest || !res.Created[0].Updated {
					t.Errorf("created = %+v, want 最新の打刻 %s の updated=true 1件", res.Created, newest)
				}
				if got := latestTimeIs(t, newest); got.EndTime == nil {
					t.Errorf("最新の打刻が終わっていない: %+v", got)
				}
				for _, id := range []string{oldest, middle} {
					if got := latestTimeIs(t, id); got.EndTime != nil {
						t.Errorf("古い打刻 %s に終了時刻が書かれた: %s（終えるのは最新の1件だけ）", id, endTimeOf(got))
					}
				}
			})

			t.Run("ーえ は同じ題名の実行中のうち開始時刻が最新の1件だけを終える", func(t *testing.T) {
				title := "endByTitle_" + GenerateNewID()[:8]
				oldest := addRunningTimeIs(t, title, now.Add(-3*time.Hour))
				newest := addRunningTimeIs(t, title, now.Add(-1*time.Hour))

				res := submitKFTL(t, tsURL, sessionID, "ーえ\n"+title, "")
				if len(res.Errors) != 0 {
					t.Fatalf("ーえ でエラー: %+v", res.Errors)
				}
				if len(res.Created) != 1 || res.Created[0].ID != newest {
					t.Errorf("created = %+v, want 最新の打刻 %s", res.Created, newest)
				}
				if got := latestTimeIs(t, oldest); got.EndTime != nil {
					t.Errorf("古い打刻に終了時刻が書かれた: %s", endTimeOf(got))
				}
			})

			t.Run("削除済みの実行中は終了の対象にしない", func(t *testing.T) {
				tag := "endtag_deleted_" + GenerateNewID()[:8]
				live := addRunningTimeIs(t, "endByTagLive", now.Add(-2*time.Hour))
				deleted := addRunningTimeIs(t, "endByTagDeleted", now.Add(-1*time.Hour)) // 最新なので、含まれていればこちらが選ばれる
				addTag(t, live, tag, now)
				addTag(t, deleted, tag, now)
				markDeleted(t, deleted, "endByTagDeleted", now.Add(-1*time.Hour))

				res := submitKFTL(t, tsURL, sessionID, "ーたえ\n"+tag, "")
				if len(res.Errors) != 0 {
					t.Fatalf("ーたえ でエラー: %+v", res.Errors)
				}
				if len(res.Created) != 1 || res.Created[0].ID != live {
					t.Errorf("created = %+v, want 生きている打刻 %s", res.Created, live)
				}
				if got := latestTimeIs(t, live); got.EndTime == nil {
					t.Errorf("生きている打刻が終わっていない: %+v", got)
				}
				if got := latestTimeIs(t, deleted); got.EndTime != nil || !got.IsDeleted {
					t.Errorf("削除済みの打刻が触られた: end=%s is_deleted=%v", endTimeOf(got), got.IsDeleted)
				}
			})

			t.Run("検索タグは Tag 行として書かれず parse_kftl_text の tags にも載らない", func(t *testing.T) {
				tag := "endtag_notwritten_" + GenerateNewID()[:8]
				id := addRunningTimeIs(t, "endByTagNoLeak", now.Add(-1*time.Hour))
				addTag(t, id, tag, now)

				parseResp := postJSON(t, tsURL+"/api/parse_kftl_text", &req_res.ParseKFTLTextRequest{SessionID: sessionID, LocaleName: "en", KFTLText: "ーたえ\n" + tag})
				var parsed req_res.ParseKFTLTextResponse
				if err := json.NewDecoder(parseResp.Body).Decode(&parsed); err != nil {
					t.Fatalf("decode parse kftl text response: %v", err)
				}
				parseResp.Body.Close()
				if len(parsed.Tags) != 0 {
					t.Errorf("parse の tags に検索タグが載っている: %v（未知タグ確認が余計に出る）", parsed.Tags)
				}

				before := len(getKyousWithQuery(t, tsURL, sessionID, &find.FindQuery{Tags: []string{tag}}).Kyous)
				res := submitKFTL(t, tsURL, sessionID, "ーたえ\n"+tag, "")
				if len(res.Errors) != 0 {
					t.Fatalf("ーたえ でエラー: %+v", res.Errors)
				}
				// タグ付きの記録は打刻1件のまま（付け先の無い Tag 行は Kyou を出さないので件数では見えない）。
				// Tag rep の行数で見る: 終了前後で増えていないこと。
				after := len(getKyousWithQuery(t, tsURL, sessionID, &find.FindQuery{Tags: []string{tag}}).Kyous)
				if before != after {
					t.Errorf("タグ %q の記録数が %d → %d に変わった", tag, before, after)
				}
				if tagRows := countTagRowsByName(t, tag); tagRows != 1 {
					t.Errorf("タグ %q の Tag 行 = %d, want 1（終了で付け先の無い Tag 行が増えた）", tag, tagRows)
				}
			})

			t.Run("？時刻 を前に書くとその時刻で終わる", func(t *testing.T) {
				endAt := now.Add(-30 * time.Minute)
				endAtText := endAt.Format("2006-01-02 15:04:05")

				tag := "endtag_reltime_" + GenerateNewID()[:8]
				byTag := addRunningTimeIs(t, "endByTagRelTime", now.Add(-2*time.Hour))
				addTag(t, byTag, tag, now)
				res := submitKFTL(t, tsURL, sessionID, "？"+endAtText+"\nーいたえ\n"+tag, "")
				if len(res.Errors) != 0 {
					t.Fatalf("？時刻 + ーいたえ でエラー: %+v（付け先の無いメタ情報になっていないか）", res.Errors)
				}
				if got := endTimeOf(latestTimeIs(t, byTag)); got != endAt.Format(time.RFC3339) {
					t.Errorf("ーいたえ の終了時刻 = %s, want %s（？時刻 が引き継がれていない）", got, endAt.Format(time.RFC3339))
				}

				title := "endByTitleRelTime_" + GenerateNewID()[:8]
				byTitle := addRunningTimeIs(t, title, now.Add(-2*time.Hour))
				res = submitKFTL(t, tsURL, sessionID, "？"+endAtText+"\nーえ\n"+title, "")
				if len(res.Errors) != 0 {
					t.Fatalf("？時刻 + ーえ でエラー: %+v", res.Errors)
				}
				if got := endTimeOf(latestTimeIs(t, byTitle)); got != endAt.Format(time.RFC3339) {
					t.Errorf("ーえ の終了時刻 = %s, want %s", got, endAt.Format(time.RFC3339))
				}
			})
		})
	}
}
