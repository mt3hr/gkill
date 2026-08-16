package kftl

import (
	"strings"
	"testing"
)

// helperMiReKyouRequest picks the single *kftlMiReKyouRequest out of the request map.
func helperMiReKyouRequest(t *testing.T, requestMap *KFTLRequestMap) *kftlMiReKyouRequest {
	t.Helper()
	var found *kftlMiReKyouRequest
	for _, req := range requestMap.All() {
		if mireq, ok := req.(*kftlMiReKyouRequest); ok {
			if found != nil {
				t.Fatalf("expected exactly 1 mirekyou request, got more than 1")
			}
			found = mireq
		}
	}
	if found == nil {
		t.Fatalf("expected a *kftlMiReKyouRequest, got none")
	}
	return found
}

func helperLabelNames(lines []KFTLStatementLine) []string {
	names := make([]string, 0, len(lines))
	for _, line := range lines {
		names = append(names, line.GetLabelName())
	}
	return names
}

func helperAssertLabels(t *testing.T, lines []KFTLStatementLine, expected []string) {
	t.Helper()
	got := helperLabelNames(lines)
	if len(got) != len(expected) {
		t.Fatalf("expected %d lines %v, got %d lines %v", len(expected), expected, len(got), got)
	}
	for i, want := range expected {
		if got[i] != want {
			t.Errorf("line %d: expected %s, got %s (all: %v)", i, want, got[i], got)
		}
	}
}

// ─── 行の並び ────────────────────────────────────────────────────────────────

func TestStatement_MiReKyouLineOrder(t *testing.T) {
	// リポストタスクはMiと違ってタイトル行を持たない(対象の記録をそのまま表示するため)。
	// TS側(kftl-statement.test.ts)と同じ並びなので、崩したら両方直すこと
	text := "牛乳を買う\n～～\n仕事\n2025-03-20\n2025-03-21\n2025-03-22\n。今日中\n～～"
	lines := helperGenerateLines(t, text)
	helperAssertLabels(t, lines, []string{
		"kmemo", "mirekyou", "mirekyouBoardName",
		"mirekyouEstimateStartTime", "mirekyouEstimateEndTime", "mirekyouLimitTime",
		"mirekyouTag", "endMirekyou",
	})
}

func TestStatement_MiReKyouTagBeforeBoardName(t *testing.T) {
	// タグ行は項目の位置を消費しないので、板名の前に書いても次の非タグ行が板名になる
	text := "牛乳を買う\n～～\n。今日中\n。重要\n仕事\n～～"
	lines := helperGenerateLines(t, text)
	helperAssertLabels(t, lines, []string{
		"kmemo", "mirekyou", "mirekyouTag", "mirekyouTag", "mirekyouBoardName", "endMirekyou",
	})
}

func TestStatement_MiReKyouTagBetweenFields(t *testing.T) {
	text := "牛乳を買う\n～～\n仕事\n。今日中\n2025-03-20\n～～"
	lines := helperGenerateLines(t, text)
	helperAssertLabels(t, lines, []string{
		"kmemo", "mirekyou", "mirekyouBoardName", "mirekyouTag", "mirekyouEstimateStartTime", "endMirekyou",
	})
}

func TestStatement_MiReKyouEmptyBlock(t *testing.T) {
	text := "牛乳を買う\n～～\n～～"
	lines := helperGenerateLines(t, text)
	helperAssertLabels(t, lines, []string{"kmemo", "mirekyou", "endMirekyou"})
}

func TestStatement_MiReKyouClosesEarly(t *testing.T) {
	text := "牛乳を買う\n～～\n仕事\n～～\nつづきは無い"
	lines := helperGenerateLines(t, text)
	helperAssertLabels(t, lines, []string{"kmemo", "mirekyou", "mirekyouBoardName", "endMirekyou", "none"})
}

func TestStatement_MiReKyouBeforeKmemoEndsWithKmemo(t *testing.T) {
	// ブロックを先に書く並び。閉じたあとはメモに戻れる
	text := "～～\n仕事\n～～\n牛乳を買う"
	lines := helperGenerateLines(t, text)
	helperAssertLabels(t, lines, []string{"mirekyou", "mirekyouBoardName", "endMirekyou", "kmemo"})
}

func TestStatement_AsciiMiReKyou(t *testing.T) {
	text := "buy milk\n~~\nwork\n。today\n~~"
	lines := helperGenerateLines(t, text)
	helperAssertLabels(t, lines, []string{"kmemo", "mirekyou", "mirekyouBoardName", "mirekyouTag", "endMirekyou"})
}

func TestStatement_MiReKyouWaveDash(t *testing.T) {
	// 「～」はWindowsのIMEがU+FF5E、macOS/iOSのIMEがU+301Cを出す。
	// 正規化を落とすとiOSからだけ記法が効かなくなる
	text := "牛乳を買う\n〜〜\n仕事\n〜〜"
	lines := helperGenerateLines(t, text)
	helperAssertLabels(t, lines, []string{"kmemo", "mirekyou", "mirekyouBoardName", "endMirekyou"})
}

func TestStatement_MiReKyouIsNotText(t *testing.T) {
	// 「～～」と「ーー」は形が似ているので取り違えを見張る
	if isMiReKyouSplitter(splitterStartText) {
		t.Errorf("text splitter %q must not be treated as a mirekyou splitter", splitterStartText)
	}
	if isMiReKyouSplitter(splitterStartTextAscii) {
		t.Errorf("text splitter %q must not be treated as a mirekyou splitter", splitterStartTextAscii)
	}
	lines := helperGenerateLines(t, "メモ\nーー\n本文\nーー")
	helperAssertLabels(t, lines, []string{"kmemo", "startText", "text", "endText"})
}

func TestStatement_MiNotEatenByMiReKyou(t *testing.T) {
	// 既存のMi記法が巻き込まれていないことの回帰テスト
	lines := helperGenerateLines(t, "ーみ\nテストタスク")
	helperAssertLabels(t, lines, []string{"mi", "miTitle"})
}

// ─── リクエストへの反映 ──────────────────────────────────────────────────────

func TestApply_MiReKyouTargetID(t *testing.T) {
	// 対象のidはバケツリレーされてきたもの、MiReKyou自身のidは別に採番される。
	// 作り直すと対象を見失って保存されない
	requestMap := helperApplyToRequestMap(t, "牛乳を買う\n～～\n仕事\n～～")
	all := requestMap.All()
	if len(all) != 2 {
		t.Fatalf("expected 2 requests, got %d", len(all))
	}
	mireq := helperMiReKyouRequest(t, requestMap)

	kmemoReq, ok := all[0].(*kftlKmemoRequest)
	if !ok {
		t.Fatalf("expected the first request to be *kftlKmemoRequest, got %T", all[0])
	}
	if mireq.targetID != kmemoReq.GetRequestID() {
		t.Errorf("expected targetID %q to be the kmemo request id %q", mireq.targetID, kmemoReq.GetRequestID())
	}
	if mireq.GetRequestID() == mireq.targetID {
		t.Errorf("mirekyou request id must differ from its targetID, both were %q", mireq.targetID)
	}
	if mireq.boardName != "仕事" {
		t.Errorf("expected boardName 仕事, got %q", mireq.boardName)
	}
}

func TestApply_MiReKyouBeforeKmemoSharesTargetID(t *testing.T) {
	// ブロックの各行がプロトタイプかどうかを次の行へ伝えていないと、
	// あとから書いたメモが別のidを引き当てて対象が消える
	requestMap := helperApplyToRequestMap(t, "～～\n仕事\n～～\n牛乳を買う")
	mireq := helperMiReKyouRequest(t, requestMap)

	var kmemoReq *kftlKmemoRequest
	for _, req := range requestMap.All() {
		if kreq, ok := req.(*kftlKmemoRequest); ok {
			kmemoReq = kreq
		}
	}
	if kmemoReq == nil {
		t.Fatalf("expected a *kftlKmemoRequest, got none")
	}
	if mireq.targetID != kmemoReq.GetRequestID() {
		t.Errorf("expected targetID %q to be the kmemo request id %q", mireq.targetID, kmemoReq.GetRequestID())
	}
}

func TestApply_MiReKyouTagGoesToMiReKyou(t *testing.T) {
	// ブロックの中のタグは対象ではなくMiReKyou自身に付く。閉じたあとのタグは対象に付く
	requestMap := helperApplyToRequestMap(t, "牛乳を買う\n～～\n。今日中\n仕事\n。重要\n～～\n。買い物")
	mireq := helperMiReKyouRequest(t, requestMap)

	miTags := strings.Join(mireq.GetTags(), ",")
	if miTags != "今日中,重要" {
		t.Errorf("expected mirekyou tags 今日中,重要, got %q", miTags)
	}

	var kmemoReq *kftlKmemoRequest
	for _, req := range requestMap.All() {
		if kreq, ok := req.(*kftlKmemoRequest); ok {
			kmemoReq = kreq
		}
	}
	if kmemoReq == nil {
		t.Fatalf("expected a *kftlKmemoRequest, got none")
	}
	kmemoTags := strings.Join(kmemoReq.GetTags(), ",")
	if kmemoTags != "買い物" {
		t.Errorf("expected kmemo tags 買い物, got %q", kmemoTags)
	}
}

func TestApply_MiReKyouTimesWithoutPrefix(t *testing.T) {
	// Miと同じく日時の前の「？」は要らない
	requestMap := helperApplyToRequestMap(t, "牛乳を買う\n～～\n仕事\n2025-03-20\n\n2025-03-22\n～～")
	mireq := helperMiReKyouRequest(t, requestMap)

	if mireq.estimateStartTime == nil || mireq.estimateStartTime.Day() != 20 {
		t.Errorf("expected estimateStartTime on the 20th, got %v", mireq.estimateStartTime)
	}
	if mireq.estimateEndTime != nil {
		t.Errorf("expected estimateEndTime to stay unset, got %v", mireq.estimateEndTime)
	}
	if mireq.limitTime == nil || mireq.limitTime.Day() != 22 {
		t.Errorf("expected limitTime on the 22nd, got %v", mireq.limitTime)
	}
}

func TestApply_MiReKyouTimesWithPrefix(t *testing.T) {
	requestMap := helperApplyToRequestMap(t, "牛乳を買う\n～～\n仕事\n？2025-03-20\n\n?2025-03-22\n～～")
	mireq := helperMiReKyouRequest(t, requestMap)

	if mireq.estimateStartTime == nil || mireq.estimateStartTime.Day() != 20 {
		t.Errorf("expected estimateStartTime on the 20th, got %v", mireq.estimateStartTime)
	}
	if mireq.limitTime == nil || mireq.limitTime.Day() != 22 {
		t.Errorf("expected limitTime on the 22nd, got %v", mireq.limitTime)
	}
}

func TestApply_MiReKyouNonTagLineAfterFieldsIsError(t *testing.T) {
	// 飲み込むと、閉じ忘れたときにメモの本文が丸ごとタグになってしまう
	_, err := helperApplyToRequestMapAllowError(t, "メモ\n～～\n仕事\n\n\n\nただの文\n～～")
	if err == nil {
		t.Errorf("expected an error for a non-tag line inside the block, got nil")
	}
}

func TestApply_MiReKyouClosedBlockHasNoError(t *testing.T) {
	if _, err := helperApplyToRequestMapAllowError(t, "メモ\n～～\n。今日中\n仕事\n2025-03-20\n\n2025-03-22\n。重要\n～～\n。買い物"); err != nil {
		t.Errorf("expected no error for a properly closed block, got %v", err)
	}
}

// ─── DoRequest のガード ──────────────────────────────────────────────────────

func TestDoRequest_MiReKyouWithoutTargetIsError(t *testing.T) {
	// 対象の無いMiReKyouは検索でターゲット解決に失敗して結果から落ちるので、
	// 画面に出ないのに消せない行が残る。書く前にエラーにする
	requestMap := helperApplyToRequestMap(t, "～～\n仕事\n～～")
	mireq := helperMiReKyouRequest(t, requestMap)
	if err := mireq.DoRequest(t.Context()); err == nil {
		t.Errorf("expected an error when the record has no target kyou, got nil")
	}
}

func TestDoRequest_MiReKyouWithPrototypeOnlyTargetIsError(t *testing.T) {
	// タグだけのレコードはプロトタイプが残るので、存在確認だけでは弾けない
	requestMap := helperApplyToRequestMap(t, "。買い物\n～～\n仕事\n～～")
	mireq := helperMiReKyouRequest(t, requestMap)
	if err := mireq.DoRequest(t.Context()); err == nil {
		t.Errorf("expected an error when the target is a prototype only, got nil")
	}
}

func TestDoRequest_MiReKyouWithoutWriteRepIsError(t *testing.T) {
	// MiReKyouは後から追加されたrep種別なので、既存の設定DBには書き込み用repが無いことがある
	requestMap := helperApplyToRequestMap(t, "牛乳を買う\n～～\n仕事\n～～")
	mireq := helperMiReKyouRequest(t, requestMap)
	err := mireq.DoRequest(t.Context())
	if err == nil {
		t.Fatalf("expected an error when WriteMiReKyouRep is nil, got nil")
	}
	if !strings.Contains(err.Error(), "not exist write mirekyou rep") {
		t.Errorf("expected the write rep error, got %v", err)
	}
}
