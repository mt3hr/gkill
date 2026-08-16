package kftl

import (
	"context"
	"testing"
)

// 支出(「ーん」のブロック)。支払い(品名と金額のペア)1組ごとに1つのリクエストになるので、
// 金額の行のあとに書いたタグ・テキストは「直前の支払い」だけに付く。
// 店名と関連時刻だけがブロック全体で共有される。
// Mirrors: src/client/__tests__/unit/kftl/kftl-request-generation.test.ts の describe('Nlog (ーん)')

// helperNlogRequests はリクエストマップから支払いのリクエストだけを順番に取り出す。
func helperNlogRequests(t *testing.T, requestMap *KFTLRequestMap) []*kftlNlogRequest {
	t.Helper()
	nlogRequests := make([]*kftlNlogRequest, 0)
	for _, request := range requestMap.All() {
		if nlogRequest, ok := request.(*kftlNlogRequest); ok {
			nlogRequests = append(nlogRequests, nlogRequest)
		}
	}
	return nlogRequests
}

func TestNlogBlock_OneRequestPerPayment(t *testing.T) {
	text := "ーん\nコンビニ\nおにぎり\n150\nお茶\n120"
	requestMap := helperApplyToRequestMap(t, text)
	nlogRequests := helperNlogRequests(t, requestMap)
	if len(nlogRequests) != 2 {
		t.Fatalf("expected 2 payments, got %d", len(nlogRequests))
	}
	if nlogRequests[0].title != "おにぎり" || nlogRequests[0].amount.String() != "150" {
		t.Errorf("payment 0: got %q / %q", nlogRequests[0].title, nlogRequests[0].amount)
	}
	if nlogRequests[1].title != "お茶" || nlogRequests[1].amount.String() != "120" {
		t.Errorf("payment 1: got %q / %q", nlogRequests[1].title, nlogRequests[1].amount)
	}
	for i, nlogRequest := range nlogRequests {
		if nlogRequest.block.shop != "コンビニ" {
			t.Errorf("payment %d: shop is %q", i, nlogRequest.block.shop)
		}
	}
	if nlogRequests[0].GetRequestID() == nlogRequests[1].GetRequestID() {
		t.Error("payments must not share a request id")
	}
}

func TestNlogBlock_TagAfterAmountBelongsToThatPayment(t *testing.T) {
	text := "ーん\nコンビニ\nおにぎり\n150\n。食費\nお茶\n120\n。飲み物"
	requestMap := helperApplyToRequestMap(t, text)
	nlogRequests := helperNlogRequests(t, requestMap)
	if len(nlogRequests) != 2 {
		t.Fatalf("expected 2 payments, got %d", len(nlogRequests))
	}
	if len(nlogRequests[0].GetTags()) != 1 || nlogRequests[0].GetTags()[0] != "食費" {
		t.Errorf("payment 0 tags: %v", nlogRequests[0].GetTags())
	}
	if len(nlogRequests[1].GetTags()) != 1 || nlogRequests[1].GetTags()[0] != "飲み物" {
		t.Errorf("payment 1 tags: %v", nlogRequests[1].GetTags())
	}
}

// タグ行でブロックが切れると、以降の品名行が拾われずおかしな行になっていた
func TestNlogBlock_TitleFollowsTagLine(t *testing.T) {
	text := "ーん\nコンビニ\nおにぎり\n150\n。食費\n。朝食\nお茶\n120"
	requestMap := helperApplyToRequestMap(t, text)
	nlogRequests := helperNlogRequests(t, requestMap)
	if len(nlogRequests) != 2 {
		t.Fatalf("expected 2 payments, got %d", len(nlogRequests))
	}
	if len(nlogRequests[0].GetTags()) != 2 {
		t.Errorf("payment 0 tags: %v", nlogRequests[0].GetTags())
	}
	if nlogRequests[1].title != "お茶" {
		t.Errorf("payment 1 title: %q", nlogRequests[1].title)
	}
}

func TestNlogBlock_TextBlockBelongsToThatPayment(t *testing.T) {
	text := "ーん\nコンビニ\nおにぎり\n150\nーー\n朝ごはん用\nーー\nお茶\n120"
	requestMap := helperApplyToRequestMap(t, text)
	nlogRequests := helperNlogRequests(t, requestMap)
	if len(nlogRequests) != 2 {
		t.Fatalf("expected 2 payments, got %d", len(nlogRequests))
	}
	if len(nlogRequests[0].GetTextsMap()) != 1 {
		t.Errorf("payment 0 texts: %v", nlogRequests[0].GetTextsMap())
	}
	for _, text := range nlogRequests[0].GetTextsMap() {
		if text != "朝ごはん用" {
			t.Errorf("payment 0 text: %q", text)
		}
	}
	if len(nlogRequests[1].GetTextsMap()) != 0 {
		t.Errorf("payment 1 texts: %v", nlogRequests[1].GetTextsMap())
	}
	if nlogRequests[1].title != "お茶" {
		t.Errorf("payment 1 title: %q", nlogRequests[1].title)
	}
}

// タグはブロックの中・金額の行のあとに書かせる。前に書かれても黙って捨てない
func TestNlogBlock_TagBeforeBlockIsRejected(t *testing.T) {
	text := "。買い物\nーん\nコンビニ\nおにぎり\n150"
	if _, err := helperApplyToRequestMapAllowError(t, text); err == nil {
		t.Fatal("expected an error for a tag written before the block, got nil")
	}
}

func TestNlogBlock_TextBeforeBlockIsRejected(t *testing.T) {
	text := "ーー\nメモ\nーー\nーん\nコンビニ\nおにぎり\n150"
	if _, err := helperApplyToRequestMapAllowError(t, text); err == nil {
		t.Fatal("expected an error for a text block written before the block, got nil")
	}
}

func TestNlogBlock_TagAtShopOrFirstTitlePositionIsRejected(t *testing.T) {
	if _, err := helperApplyToRequestMapAllowError(t, "ーん\n。食費\nおにぎり\n150"); err == nil {
		t.Error("expected an error for a tag at the shop name position, got nil")
	}
	if _, err := helperApplyToRequestMapAllowError(t, "ーん\nコンビニ\n。食費\n150"); err == nil {
		t.Error("expected an error for a tag at the first title position, got nil")
	}
}

// 関連時刻だけは支払いごとではなくブロック全体。書いた位置より前の支払いにも効く
func TestNlogBlock_RelatedTimeBeforeBlockAppliesToAllPayments(t *testing.T) {
	text := "？2025-01-15 10:00:00\nーん\nコンビニ\nおにぎり\n150\nお茶\n120"
	requestMap := helperApplyToRequestMap(t, text)
	nlogRequests := helperNlogRequests(t, requestMap)
	if len(nlogRequests) != 2 {
		t.Fatalf("expected 2 payments, got %d", len(nlogRequests))
	}
	for i, nlogRequest := range nlogRequests {
		relatedTime := nlogRequest.GetRelatedTime()
		if relatedTime.Year() != 2025 || relatedTime.Month() != 1 || relatedTime.Day() != 15 {
			t.Errorf("payment %d related time: %v", i, relatedTime)
		}
	}
}

func TestNlogBlock_RelatedTimeInsideBlockAppliesToAllPayments(t *testing.T) {
	text := "ーん\nコンビニ\nおにぎり\n150\n？2025-01-15 10:00:00\nお茶\n120"
	requestMap := helperApplyToRequestMap(t, text)
	nlogRequests := helperNlogRequests(t, requestMap)
	if len(nlogRequests) != 2 {
		t.Fatalf("expected 2 payments, got %d", len(nlogRequests))
	}
	for i, nlogRequest := range nlogRequests {
		if nlogRequest.GetRelatedTime().Year() != 2025 {
			t.Errorf("payment %d related time: %v", i, nlogRequest.GetRelatedTime())
		}
	}
}

// 末尾の改行が品名行として解釈されるだけなので、エラーにも支払いにもしない
func TestNlogBlock_TrailingBlankLineIsIgnored(t *testing.T) {
	text := "ーん\nコンビニ\nおにぎり\n150\n"
	requestMap, err := helperApplyToRequestMapAllowError(t, text)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	nlogRequests := helperNlogRequests(t, requestMap)
	blank := nlogRequests[len(nlogRequests)-1]
	if blank.title != "" {
		t.Fatalf("expected a blank payment, got %q", blank.title)
	}
	if err := blank.DoRequest(context.Background()); err != nil {
		t.Fatalf("a blank payment must be ignored, got: %v", err)
	}
}

func TestNlogBlock_DecimalAmountIsAccepted(t *testing.T) {
	text := "ーん\nカフェ\nコーヒー\n-1.5"
	requestMap := helperApplyToRequestMap(t, text)
	nlogRequests := helperNlogRequests(t, requestMap)
	if len(nlogRequests) != 1 {
		t.Fatalf("expected 1 payment, got %d", len(nlogRequests))
	}
	if nlogRequests[0].amount.String() != "-1.5" {
		t.Errorf("amount: %q", nlogRequests[0].amount)
	}
}

func TestNlogBlock_AsciiPrefixes(t *testing.T) {
	text := "/expense\nConvenience store\nRice ball\n150\n#food\nTea\n120\n#drink"
	requestMap := helperApplyToRequestMap(t, text)
	nlogRequests := helperNlogRequests(t, requestMap)
	if len(nlogRequests) != 2 {
		t.Fatalf("expected 2 payments, got %d", len(nlogRequests))
	}
	if len(nlogRequests[0].GetTags()) != 1 || nlogRequests[0].GetTags()[0] != "food" {
		t.Errorf("payment 0 tags: %v", nlogRequests[0].GetTags())
	}
	if len(nlogRequests[1].GetTags()) != 1 || nlogRequests[1].GetTags()[0] != "drink" {
		t.Errorf("payment 1 tags: %v", nlogRequests[1].GetTags())
	}
}

// resume を渡さない経路(＝支出ブロックの外)は今までどおり Kmemo / None へ抜ける
func TestAfterMetaInfoConstructor_NilResumeKeepsLegacyBehavior(t *testing.T) {
	lines := helperGenerateLines(t, "メモ\n。タグ\nつづき")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	if lines[1].GetLabelName() != "tag" {
		t.Errorf("line 1: expected tag, got %s", lines[1].GetLabelName())
	}
	if lines[2].GetLabelName() != "none" {
		t.Errorf("line 2: expected none, got %s", lines[2].GetLabelName())
	}
}
