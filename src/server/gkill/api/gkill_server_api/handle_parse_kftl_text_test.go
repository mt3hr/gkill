package gkill_server_api

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/mt3hr/gkill/src/server/gkill/api/req_res"
)

// parseKFTL は /api/parse_kftl_text を叩き、HTTPステータスと応答を返す。
func parseKFTL(t *testing.T, tsURL, sessionID, kftlText string) (int, req_res.ParseKFTLTextResponse) {
	t.Helper()
	resp := postJSON(t, tsURL+"/api/parse_kftl_text", &req_res.ParseKFTLTextRequest{
		SessionID:  sessionID,
		LocaleName: "en",
		KFTLText:   kftlText,
	})
	defer resp.Body.Close()

	res := req_res.ParseKFTLTextResponse{}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		t.Fatalf("decode parse kftl text response: %v", err)
	}
	return resp.StatusCode, res
}

// /api/parse_kftl_text（ADR-0507）のハンドラ層テスト。
//
// Web のメモ帳は「おかしな行」のピンク表示と未知タグ・未知板名の確認をこの応答に頼るので、
// (1) 書き間違いは 200 のまま invalid_lines に行番号つきで載る、(2) 何も書かれない、
// (3) タグ・板名が列挙される、(4) 空は null ではなく [] で返る、を固定する。
func TestHandleParseKFTLText(t *testing.T) {
	tsURL, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()

	sessionID := loginAndGetSession(t, tsURL, gkillAPI, "admin", regressionTestPasswordHash)

	t.Run("単独の /mood と同じ行の引数は行番号つきで invalid_lines に載り、何も書かれない", func(t *testing.T) {
		const word = "parseKftlInvalidWord"
		// 1行目 kmemo / 2行目 /mood 8（同じ行に引数）/ 3行目 区切り / 4行目 /mood（値の行が無い）
		status, res := parseKFTL(t, tsURL, sessionID, word+"\n/mood 8\n、\n/mood")
		if status != http.StatusOK {
			t.Errorf("status = %d, want 200（書き間違いは解析の成功。errors ではなく invalid_lines）", status)
		}
		if len(res.Errors) != 0 {
			t.Fatalf("errors = %+v, want 空", res.Errors)
		}
		if len(res.InvalidLines) != 2 {
			t.Fatalf("invalid_lines の件数 = %d, want 2: %+v", len(res.InvalidLines), res.InvalidLines)
		}
		if got := res.InvalidLines[0]; got.LineNumber != 2 || got.LineText != "/mood 8" {
			t.Errorf("invalid_lines[0] = %+v, want line 2 \"/mood 8\"", got)
		}
		if got := res.InvalidLines[1]; got.LineNumber != 4 || got.LineText != "/mood" {
			t.Errorf("invalid_lines[1] = %+v, want line 4 \"/mood\"", got)
		}
		// 文面は submit_kftl_text の errors[].error_message と同じ形（行情報 + 多言語の理由）。
		if msg := res.InvalidLines[0].Message; !strings.Contains(msg, `(line 2: "/mood 8")`) {
			t.Errorf("invalid_lines[0].message に行情報が無い: %q", msg)
		}
		if got := countKmemosByContent(t, tsURL, sessionID, word); got != 0 {
			t.Errorf("解析だけなのに kmemo が書かれている: 件数 = %d, want 0", got)
		}
	})

	t.Run("正しい本文ではタグと板名が列挙され、何も書かれない", func(t *testing.T) {
		const word = "parseKftlValidWord"
		// Mi のブロックは「/mi・タイトル・板名・見積開始・見積終了・期限」の6行固定（空行で欄を飛ばす）
		text := strings.Join([]string{
			word,
			"。parseTagA",
			"。parseTagB",
			"、",
			"/mi",
			"parse mi title",
			"parseBoardX",
			"",
			"",
			"",
			"。parseTagA",
			"、",
			"/mi",
			"parse mi title 2",
			"",
		}, "\n")
		status, res := parseKFTL(t, tsURL, sessionID, text)
		if status != http.StatusOK {
			t.Errorf("status = %d, want 200", status)
		}
		if len(res.Errors) != 0 || len(res.InvalidLines) != 0 {
			t.Fatalf("errors = %+v invalid_lines = %+v, want どちらも空", res.Errors, res.InvalidLines)
		}
		if want := []string{"parseTagA", "parseTagB"}; strings.Join(res.Tags, ",") != strings.Join(want, ",") {
			t.Errorf("tags = %v, want %v（重複なし・出現順）", res.Tags, want)
		}
		// 板名は利用者が書いたとおり。2つ目の Mi は板名を書いていないので既定板へ解決せず、列挙もしない
		if want := []string{"parseBoardX"}; strings.Join(res.MiBoardNames, ",") != strings.Join(want, ",") {
			t.Errorf("mi_board_names = %v, want %v", res.MiBoardNames, want)
		}
		if res.RecordCount != 3 {
			t.Errorf("record_count = %d, want 3（kmemo + Mi 2件）", res.RecordCount)
		}
		if got := countKmemosByContent(t, tsURL, sessionID, word); got != 0 {
			t.Errorf("解析だけなのに kmemo が書かれている: 件数 = %d, want 0", got)
		}
	})

	t.Run("空は null ではなく [] で返る", func(t *testing.T) {
		resp := postJSON(t, tsURL+"/api/parse_kftl_text", &req_res.ParseKFTLTextRequest{
			SessionID: sessionID, LocaleName: "en", KFTLText: "plain memo",
		})
		defer resp.Body.Close()
		var raw map[string]json.RawMessage
		if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
			t.Fatalf("decode: %v", err)
		}
		for _, key := range []string{"invalid_lines", "tags", "mi_board_names"} {
			if got := strings.TrimSpace(string(raw[key])); got != "[]" {
				t.Errorf("%s = %s, want [] （クライアントは「空 = 送信してよい」で判定する）", key, got)
			}
		}
	})

	t.Run("繰り返しブロックの書き損じも invalid_lines に載る", func(t *testing.T) {
		// 繰り返しの検査・展開は submit と同じ prepareRequests で行うので、解析でも同じ行エラーが出る
		status, res := parseKFTL(t, tsURL, sessionID, "repeat memo\n？？\n毎日\n？？")
		if status != http.StatusOK {
			t.Errorf("status = %d, want 200", status)
		}
		if len(res.InvalidLines) != 1 {
			t.Fatalf("invalid_lines = %+v, want 1件（回数/終了日の行が無い）", res.InvalidLines)
		}
		if got := res.InvalidLines[0]; got.LineNumber != 4 || got.LineText != "？？" {
			t.Errorf("invalid_lines[0] = %+v, want line 4 \"？？\"", got)
		}
	})
}
