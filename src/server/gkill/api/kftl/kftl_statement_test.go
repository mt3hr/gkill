package kftl

import (
	"context"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
	"github.com/mt3hr/gkill/src/server/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/server/gkill/dao/user_config"
)

// helperGenerateLines calls generateKFTLLines with minimal dependencies.
// It does NOT call DoRequest, so no real repositories are needed.
func helperGenerateLines(t *testing.T, text string) []KFTLStatementLine {
	t.Helper()
	stmt := &KFTLStatement{StatementText: text}
	factory := newKFTLFactory()
	factory.reset()
	txID := sqlite3impl.GenerateNewID()
	baseTime := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)

	lines, err := stmt.generateKFTLLines(
		factory,
		txID,
		baseTime,
		&reps.GkillRepositories{},        // not used during line generation
		&user_config.ApplicationConfig{},  // not used during line generation
		"test-user", "test-device", "test-app", "ja",
	)
	if err != nil {
		t.Fatalf("generateKFTLLines error: %v", err)
	}
	return lines
}

// helperApplyToRequestMap generates lines and applies them to a request map.
func helperApplyToRequestMap(t *testing.T, text string) *KFTLRequestMap {
	t.Helper()
	lines := helperGenerateLines(t, text)
	requestMap := NewKFTLRequestMap()
	for _, line := range lines {
		if err := line.ApplyThisLineToRequestMap(context.Background(), requestMap); err != nil {
			t.Fatalf("ApplyThisLineToRequestMap error on line %q: %v",
				line.GetStatementLineText(), err)
		}
	}
	return requestMap
}

func TestStatement_EmptyText(t *testing.T) {
	lines := helperGenerateLines(t, "")
	// An empty string splits into one empty line, which becomes a kmemo line
	if len(lines) != 1 {
		t.Fatalf("expected 1 line for empty text, got %d", len(lines))
	}
	if lines[0].GetLabelName() != "kmemo" {
		t.Errorf("expected kmemo label, got %s", lines[0].GetLabelName())
	}
}

func TestStatement_SingleKmemoLine(t *testing.T) {
	lines := helperGenerateLines(t, "hello world")
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	if lines[0].GetLabelName() != "kmemo" {
		t.Errorf("expected kmemo, got %s", lines[0].GetLabelName())
	}
	if lines[0].GetStatementLineText() != "hello world" {
		t.Errorf("expected 'hello world', got %q", lines[0].GetStatementLineText())
	}
}

func TestStatement_MultilineKmemo(t *testing.T) {
	text := "line one\nline two\nline three"
	lines := helperGenerateLines(t, text)
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	for _, l := range lines {
		if l.GetLabelName() != "kmemo" {
			t.Errorf("expected kmemo, got %s", l.GetLabelName())
		}
	}
	// All lines should share the same target ID (continuation)
	id0 := lines[0].GetContext().ThisStatementLineTargetID
	for i := 1; i < len(lines); i++ {
		if lines[i].GetContext().ThisStatementLineTargetID != id0 {
			t.Errorf("line %d target ID %s != line 0 target ID %s",
				i, lines[i].GetContext().ThisStatementLineTargetID, id0)
		}
	}
}

func TestStatement_KmemoWithTag(t *testing.T) {
	// First line is the kmemo content, tag prefix "。" triggers tag on next line
	text := "。tag1\nhello"
	lines := helperGenerateLines(t, text)
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	if lines[0].GetLabelName() != "tag" {
		t.Errorf("line 0: expected tag, got %s", lines[0].GetLabelName())
	}
	if lines[1].GetLabelName() != "kmemo" {
		t.Errorf("line 1: expected kmemo, got %s", lines[1].GetLabelName())
	}
}

func TestStatement_SplitSeparatesEntities(t *testing.T) {
	// Two kmemos separated by split character "、"
	text := "first memo\n、\nsecond memo"
	lines := helperGenerateLines(t, text)
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	if lines[0].GetLabelName() != "kmemo" {
		t.Errorf("line 0: expected kmemo, got %s", lines[0].GetLabelName())
	}
	if lines[1].GetLabelName() != "split" {
		t.Errorf("line 1: expected split, got %s", lines[1].GetLabelName())
	}
	if lines[2].GetLabelName() != "kmemo" {
		t.Errorf("line 2: expected kmemo, got %s", lines[2].GetLabelName())
	}
	// The two kmemos should have different target IDs
	id0 := lines[0].GetContext().ThisStatementLineTargetID
	id2 := lines[2].GetContext().ThisStatementLineTargetID
	if id0 == id2 {
		t.Error("expected different target IDs after split")
	}
}

func TestStatement_SplitAndNextSecondIncrementsAddSecond(t *testing.T) {
	text := "first\n、、\nsecond"
	lines := helperGenerateLines(t, text)
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	if lines[1].GetLabelName() != "split+1s" {
		t.Errorf("line 1: expected split+1s, got %s", lines[1].GetLabelName())
	}
	// After a split-and-next-second, the AddSecond on the next line's context should be 1
	if lines[2].GetContext().AddSecond != 1 {
		t.Errorf("expected AddSecond=1 after split+1s, got %d", lines[2].GetContext().AddSecond)
	}
}

func TestStatement_SaveCharacterStopsProcessing(t *testing.T) {
	// "！" on a non-first line should stop processing
	text := "memo content\n！\nthis should be ignored"
	lines := helperGenerateLines(t, text)
	// Should only have the first line (kmemo); the "！" line triggers break
	if len(lines) != 1 {
		t.Fatalf("expected 1 line (stop at save character), got %d", len(lines))
	}
	if lines[0].GetStatementLineText() != "memo content" {
		t.Errorf("expected 'memo content', got %q", lines[0].GetStatementLineText())
	}
}

func TestStatement_SaveCharacterOnFirstLineDoesNotStop(t *testing.T) {
	// "！" on the first line should NOT stop processing
	text := "！"
	lines := helperGenerateLines(t, text)
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
}

func TestStatement_TagAppliedToRequestMap(t *testing.T) {
	// Tag line + kmemo content: the tag should be inherited by the kmemo request
	text := "。myTag\nhello"
	requestMap := helperApplyToRequestMap(t, text)

	all := requestMap.All()
	if len(all) != 1 {
		t.Fatalf("expected 1 request, got %d", len(all))
	}

	tags := all[0].GetTags()
	found := false
	for _, tag := range tags {
		if tag == "myTag" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected tag 'myTag' in request, got tags %v", tags)
	}
}

func TestStatement_MultipleTagsParsed(t *testing.T) {
	// Multiple tags separated by "、" on one tag line
	text := "。tagA、tagB、tagC\nhello"
	requestMap := helperApplyToRequestMap(t, text)

	all := requestMap.All()
	if len(all) != 1 {
		t.Fatalf("expected 1 request, got %d", len(all))
	}

	tags := all[0].GetTags()
	expected := []string{"tagA", "tagB", "tagC"}
	if len(tags) != len(expected) {
		t.Fatalf("expected %d tags, got %d: %v", len(expected), len(tags), tags)
	}
	for i, e := range expected {
		if tags[i] != e {
			t.Errorf("tag[%d]: expected %s, got %s", i, e, tags[i])
		}
	}
}

func TestStatement_SplitCreatesMultipleRequests(t *testing.T) {
	text := "first memo\n、\nsecond memo"
	requestMap := helperApplyToRequestMap(t, text)

	all := requestMap.All()
	if len(all) != 2 {
		t.Fatalf("expected 2 requests after split, got %d", len(all))
	}
	if all[0].GetRequestID() == all[1].GetRequestID() {
		t.Error("expected different request IDs for split entities")
	}
}

func TestStatement_EmptyKmemoApplyProducesOneRequest(t *testing.T) {
	text := ""
	requestMap := helperApplyToRequestMap(t, text)

	all := requestMap.All()
	// Empty text produces one kmemo line with empty content
	if len(all) != 1 {
		t.Fatalf("expected 1 request for empty text, got %d", len(all))
	}
}

func TestStatement_TextBlockApplied(t *testing.T) {
	// Start text "ーー", content lines, end text "ーー"
	text := "ーー\ntext content line 1\ntext content line 2\nーー"
	requestMap := helperApplyToRequestMap(t, text)

	all := requestMap.All()
	if len(all) != 1 {
		t.Fatalf("expected 1 request, got %d", len(all))
	}

	textsMap := all[0].GetTextsMap()
	if len(textsMap) == 0 {
		t.Fatal("expected text content in request, got empty texts map")
	}

	// Find the text entry and verify content
	found := false
	for _, content := range textsMap {
		if content == "text content line 1\ntext content line 2" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected multiline text content, got texts map: %v", textsMap)
	}
}

func TestStatement_ConstructionOnly(t *testing.T) {
	// Verify that KFTLStatement can be constructed with StatementText
	stmt := &KFTLStatement{StatementText: "test"}
	if stmt.StatementText != "test" {
		t.Errorf("expected StatementText='test', got %q", stmt.StatementText)
	}
}

// ─── Phase 1: Data type KFTL statement tests (項番12-22) ───────────────────

func TestStatement_LantanaLine(t *testing.T) {
	// "ーら" triggers lantana, next line is mood value (項番13)
	text := "ーら\n5"
	lines := helperGenerateLines(t, text)
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	if lines[0].GetLabelName() != "lantana" {
		t.Errorf("line 0: expected lantana, got %s", lines[0].GetLabelName())
	}
	if lines[1].GetLabelName() != "lantanaMood" {
		t.Errorf("line 1: expected lantanaMood, got %s", lines[1].GetLabelName())
	}
}

func TestStatement_LantanaWithTag(t *testing.T) {
	// Tag + lantana with mood (項番13 variant with tag)
	text := "。myTag\nーら\n7"
	lines := helperGenerateLines(t, text)
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	if lines[0].GetLabelName() != "tag" {
		t.Errorf("line 0: expected tag, got %s", lines[0].GetLabelName())
	}
	if lines[1].GetLabelName() != "lantana" {
		t.Errorf("line 1: expected lantana, got %s", lines[1].GetLabelName())
	}
	if lines[2].GetLabelName() != "lantanaMood" {
		t.Errorf("line 2: expected lantanaMood, got %s", lines[2].GetLabelName())
	}
}

func TestStatement_LantanaWithRelatedTime(t *testing.T) {
	// Related time + lantana (項番13 variant with time)
	text := "？2025-06-01T10:00:00+09:00\nーら\n3"
	lines := helperGenerateLines(t, text)
	if len(lines) < 3 {
		t.Fatalf("expected at least 3 lines, got %d", len(lines))
	}
	if lines[0].GetLabelName() != "relatedTime" {
		t.Errorf("line 0: expected relatedTime, got %s", lines[0].GetLabelName())
	}
	if lines[1].GetLabelName() != "lantana" {
		t.Errorf("line 1: expected lantana, got %s", lines[1].GetLabelName())
	}
}

func TestStatement_MiLine(t *testing.T) {
	// "ーみ" triggers mi, next line is title (項番14)
	text := "ーみ\nテストタスク"
	lines := helperGenerateLines(t, text)
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	if lines[0].GetLabelName() != "mi" {
		t.Errorf("line 0: expected mi, got %s", lines[0].GetLabelName())
	}
	if lines[1].GetLabelName() != "miTitle" {
		t.Errorf("line 1: expected miTitle, got %s", lines[1].GetLabelName())
	}
}

func TestStatement_MiWithTag(t *testing.T) {
	// Tag + mi (項番14 variant with tag)
	text := "。taskTag\nーみ\nマイタスク"
	lines := helperGenerateLines(t, text)
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	if lines[0].GetLabelName() != "tag" {
		t.Errorf("line 0: expected tag, got %s", lines[0].GetLabelName())
	}
	if lines[1].GetLabelName() != "mi" {
		t.Errorf("line 1: expected mi, got %s", lines[1].GetLabelName())
	}
	if lines[2].GetLabelName() != "miTitle" {
		t.Errorf("line 2: expected miTitle, got %s", lines[2].GetLabelName())
	}
}

func TestStatement_NlogLine(t *testing.T) {
	// "ーん" triggers nlog, next lines are shop name then amount (項番15)
	text := "ーん\n500\nテスト店"
	lines := helperGenerateLines(t, text)
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	if lines[0].GetLabelName() != "nlog" {
		t.Errorf("line 0: expected nlog, got %s", lines[0].GetLabelName())
	}
}

func TestStatement_TimeIsLine(t *testing.T) {
	// "ーち" triggers timeis (start+end), next line is title (項番16)
	text := "ーち\n作業テスト"
	lines := helperGenerateLines(t, text)
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	if lines[0].GetLabelName() != "timeIs" {
		t.Errorf("line 0: expected timeIs, got %s", lines[0].GetLabelName())
	}
	if lines[1].GetLabelName() != "timeIsTitle" {
		t.Errorf("line 1: expected timeIsTitle, got %s", lines[1].GetLabelName())
	}
}

func TestStatement_TimeIsStartLine(t *testing.T) {
	// "ーた" triggers timeis start (項番17)
	text := "ーた\n開始作業"
	lines := helperGenerateLines(t, text)
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	if lines[0].GetLabelName() != "timeIsStart" {
		t.Errorf("line 0: expected timeIsStart, got %s", lines[0].GetLabelName())
	}
	if lines[1].GetLabelName() != "timeIsStartTitle" {
		t.Errorf("line 1: expected timeIsStartTitle, got %s", lines[1].GetLabelName())
	}
}

func TestStatement_TimeIsEndLine(t *testing.T) {
	// "ーえ" triggers timeis end by title (項番18)
	text := "ーえ\n終了作業"
	lines := helperGenerateLines(t, text)
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	if lines[0].GetLabelName() != "timeIsEnd" {
		t.Errorf("line 0: expected timeIsEnd, got %s", lines[0].GetLabelName())
	}
	if lines[1].GetLabelName() != "timeIsEndTitle" {
		t.Errorf("line 1: expected timeIsEndTitle, got %s", lines[1].GetLabelName())
	}
}

func TestStatement_TimeIsEndIfExistLine(t *testing.T) {
	// "ーいえ" triggers timeis end if exist by title (項番19)
	text := "ーいえ\n終了作業"
	lines := helperGenerateLines(t, text)
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	if lines[0].GetLabelName() != "timeIsEndIfExist" {
		t.Errorf("line 0: expected timeIsEndIfExist, got %s", lines[0].GetLabelName())
	}
}

func TestStatement_TimeIsEndByTagLine(t *testing.T) {
	// "ーたえ" triggers timeis end by tag (項番20)
	text := "ーたえ\nテストタグ"
	lines := helperGenerateLines(t, text)
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	if lines[0].GetLabelName() != "timeIsEndByTag" {
		t.Errorf("line 0: expected timeIsEndByTag, got %s", lines[0].GetLabelName())
	}
	if lines[1].GetLabelName() != "timeIsEndByTagTag" {
		t.Errorf("line 1: expected timeIsEndByTagTag, got %s", lines[1].GetLabelName())
	}
}

func TestStatement_TimeIsEndByTagIfExistLine(t *testing.T) {
	// "ーいたえ" triggers timeis end by tag if exist (項番21)
	text := "ーいたえ\nテストタグ"
	lines := helperGenerateLines(t, text)
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	if lines[0].GetLabelName() != "timeIsEndByTagIfExist" {
		t.Errorf("line 0: expected timeIsEndByTagIfExist, got %s", lines[0].GetLabelName())
	}
}

func TestStatement_URLogLine(t *testing.T) {
	// "ーう" triggers urlog, next line is URL (項番22)
	text := "ーう\nhttps://example.com"
	lines := helperGenerateLines(t, text)
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	if lines[0].GetLabelName() != "urlog" {
		t.Errorf("line 0: expected urlog, got %s", lines[0].GetLabelName())
	}
	if lines[1].GetLabelName() != "urlogURL" {
		t.Errorf("line 1: expected urlogURL, got %s", lines[1].GetLabelName())
	}
}

func TestStatement_URLogWithTagAndTitle(t *testing.T) {
	// Tag + urlog with title (項番22 variant)
	text := "。linkTag\nーう\nhttps://example.com\nリンクタイトル"
	lines := helperGenerateLines(t, text)
	if len(lines) != 4 {
		t.Fatalf("expected 4 lines, got %d", len(lines))
	}
	if lines[0].GetLabelName() != "tag" {
		t.Errorf("line 0: expected tag, got %s", lines[0].GetLabelName())
	}
	if lines[1].GetLabelName() != "urlog" {
		t.Errorf("line 1: expected urlog, got %s", lines[1].GetLabelName())
	}
	if lines[2].GetLabelName() != "urlogURL" {
		t.Errorf("line 2: expected urlogURL, got %s", lines[2].GetLabelName())
	}
	if lines[3].GetLabelName() != "urlogTitle" {
		t.Errorf("line 3: expected urlogTitle, got %s", lines[3].GetLabelName())
	}
}

func TestStatement_KCLine(t *testing.T) {
	// "ーか" triggers KC, next lines are title then num value
	text := "ーか\nカウンター\n42"
	lines := helperGenerateLines(t, text)
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	if lines[0].GetLabelName() != "kc" {
		t.Errorf("line 0: expected kc, got %s", lines[0].GetLabelName())
	}
	if lines[1].GetLabelName() != "kcTitle" {
		t.Errorf("line 1: expected kcTitle, got %s", lines[1].GetLabelName())
	}
	if lines[2].GetLabelName() != "kcNumValue" {
		t.Errorf("line 2: expected kcNumValue, got %s", lines[2].GetLabelName())
	}
}

func TestStatement_KmemoWithTextBlock(t *testing.T) {
	// Kmemo content + text block attachment (項番12 variant)
	text := "メモ内容\nーー\nテキスト本文\nーー"
	lines := helperGenerateLines(t, text)
	if len(lines) != 4 {
		t.Fatalf("expected 4 lines, got %d", len(lines))
	}
	if lines[0].GetLabelName() != "kmemo" {
		t.Errorf("line 0: expected kmemo, got %s", lines[0].GetLabelName())
	}
	if lines[1].GetLabelName() != "startText" {
		t.Errorf("line 1: expected startText, got %s", lines[1].GetLabelName())
	}
	if lines[2].GetLabelName() != "text" {
		t.Errorf("line 2: expected text, got %s", lines[2].GetLabelName())
	}
	if lines[3].GetLabelName() != "endText" {
		t.Errorf("line 3: expected endText, got %s", lines[3].GetLabelName())
	}
}

// ─── Request map tests for data types ───────────────────────────────────────

func TestStatement_LantanaAppliedToRequestMap(t *testing.T) {
	text := "ーら\n5"
	requestMap := helperApplyToRequestMap(t, text)
	all := requestMap.All()
	if len(all) != 1 {
		t.Fatalf("expected 1 request, got %d", len(all))
	}
}

func TestStatement_MiAppliedToRequestMap(t *testing.T) {
	text := "ーみ\nテストタスク"
	requestMap := helperApplyToRequestMap(t, text)
	all := requestMap.All()
	if len(all) != 1 {
		t.Fatalf("expected 1 request, got %d", len(all))
	}
}

func TestStatement_NlogAppliedToRequestMap(t *testing.T) {
	text := "ーん\n500\nテスト店"
	requestMap := helperApplyToRequestMap(t, text)
	all := requestMap.All()
	if len(all) != 1 {
		t.Fatalf("expected 1 request, got %d", len(all))
	}
}

func TestStatement_TimeIsStartAppliedToRequestMap(t *testing.T) {
	text := "ーた\n開始作業"
	requestMap := helperApplyToRequestMap(t, text)
	all := requestMap.All()
	if len(all) != 1 {
		t.Fatalf("expected 1 request, got %d", len(all))
	}
}

func TestStatement_URLogAppliedToRequestMap(t *testing.T) {
	text := "ーう\nhttps://example.com"
	requestMap := helperApplyToRequestMap(t, text)
	all := requestMap.All()
	if len(all) != 1 {
		t.Fatalf("expected 1 request, got %d", len(all))
	}
}

func TestStatement_KCAppliedToRequestMap(t *testing.T) {
	text := "ーか\nカウンター\n42"
	requestMap := helperApplyToRequestMap(t, text)
	all := requestMap.All()
	if len(all) != 1 {
		t.Fatalf("expected 1 request, got %d", len(all))
	}
}

func TestStatement_LantanaWithTagAppliedToRequestMap(t *testing.T) {
	// Tag should be inherited by the lantana request
	text := "。moodTag\nーら\n8"
	requestMap := helperApplyToRequestMap(t, text)
	all := requestMap.All()
	if len(all) != 1 {
		t.Fatalf("expected 1 request, got %d", len(all))
	}
	tags := all[0].GetTags()
	found := false
	for _, tag := range tags {
		if tag == "moodTag" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected tag 'moodTag' in lantana request, got tags %v", tags)
	}
}

// ─── Phase 2: ASCII prefix tests (P23) ─────────────────────────────────────

func TestStatement_AsciiSaveCharacter(t *testing.T) {
	// ASCII "!" on a non-first line should stop processing
	text := "memo content\n!\nthis should be ignored"
	lines := helperGenerateLines(t, text)
	if len(lines) != 1 {
		t.Fatalf("expected 1 line (stop at ASCII save character), got %d", len(lines))
	}
	if lines[0].GetStatementLineText() != "memo content" {
		t.Errorf("expected 'memo content', got %q", lines[0].GetStatementLineText())
	}
}

func TestStatement_AsciiTag(t *testing.T) {
	text := "#tag1\nhello"
	lines := helperGenerateLines(t, text)
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	if lines[0].GetLabelName() != "tag" {
		t.Errorf("line 0: expected tag, got %s", lines[0].GetLabelName())
	}
	if lines[1].GetLabelName() != "kmemo" {
		t.Errorf("line 1: expected kmemo, got %s", lines[1].GetLabelName())
	}
}

func TestStatement_AsciiSplit(t *testing.T) {
	text := "first memo\n,\nsecond memo"
	lines := helperGenerateLines(t, text)
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	if lines[1].GetLabelName() != "split" {
		t.Errorf("line 1: expected split, got %s", lines[1].GetLabelName())
	}
}

func TestStatement_AsciiSplitNextSecond(t *testing.T) {
	text := "first\n,,\nsecond"
	lines := helperGenerateLines(t, text)
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	if lines[1].GetLabelName() != "split+1s" {
		t.Errorf("line 1: expected split+1s, got %s", lines[1].GetLabelName())
	}
	if lines[2].GetContext().AddSecond != 1 {
		t.Errorf("expected AddSecond=1 after split+1s, got %d", lines[2].GetContext().AddSecond)
	}
}

func TestStatement_AsciiStartText(t *testing.T) {
	text := "memo\n--\nblock content\n--"
	lines := helperGenerateLines(t, text)
	if len(lines) != 4 {
		t.Fatalf("expected 4 lines, got %d", len(lines))
	}
	if lines[0].GetLabelName() != "kmemo" {
		t.Errorf("line 0: expected kmemo, got %s", lines[0].GetLabelName())
	}
	if lines[1].GetLabelName() != "startText" {
		t.Errorf("line 1: expected startText, got %s", lines[1].GetLabelName())
	}
	if lines[2].GetLabelName() != "text" {
		t.Errorf("line 2: expected text, got %s", lines[2].GetLabelName())
	}
	if lines[3].GetLabelName() != "endText" {
		t.Errorf("line 3: expected endText, got %s", lines[3].GetLabelName())
	}
}

func TestStatement_AsciiRelatedTime(t *testing.T) {
	text := "?2025-06-01T10:00:00+09:00\nhello"
	lines := helperGenerateLines(t, text)
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 lines, got %d", len(lines))
	}
	if lines[0].GetLabelName() != "relatedTime" {
		t.Errorf("line 0: expected relatedTime, got %s", lines[0].GetLabelName())
	}
}

func TestStatement_AsciiMi(t *testing.T) {
	text := "/mi\nTask title"
	lines := helperGenerateLines(t, text)
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	if lines[0].GetLabelName() != "mi" {
		t.Errorf("line 0: expected mi, got %s", lines[0].GetLabelName())
	}
	if lines[1].GetLabelName() != "miTitle" {
		t.Errorf("line 1: expected miTitle, got %s", lines[1].GetLabelName())
	}
}

func TestStatement_AsciiLantana(t *testing.T) {
	text := "/mood\n7"
	lines := helperGenerateLines(t, text)
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	if lines[0].GetLabelName() != "lantana" {
		t.Errorf("line 0: expected lantana, got %s", lines[0].GetLabelName())
	}
	if lines[1].GetLabelName() != "lantanaMood" {
		t.Errorf("line 1: expected lantanaMood, got %s", lines[1].GetLabelName())
	}
}

func TestStatement_AsciiNlog(t *testing.T) {
	text := "/expense\n500\nShop"
	lines := helperGenerateLines(t, text)
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	if lines[0].GetLabelName() != "nlog" {
		t.Errorf("line 0: expected nlog, got %s", lines[0].GetLabelName())
	}
}

func TestStatement_AsciiKC(t *testing.T) {
	text := "/num\nTitle\n42"
	lines := helperGenerateLines(t, text)
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	if lines[0].GetLabelName() != "kc" {
		t.Errorf("line 0: expected kc, got %s", lines[0].GetLabelName())
	}
	if lines[1].GetLabelName() != "kcTitle" {
		t.Errorf("line 1: expected kcTitle, got %s", lines[1].GetLabelName())
	}
	if lines[2].GetLabelName() != "kcNumValue" {
		t.Errorf("line 2: expected kcNumValue, got %s", lines[2].GetLabelName())
	}
}

func TestStatement_AsciiURLog(t *testing.T) {
	text := "/url\nhttps://example.com"
	lines := helperGenerateLines(t, text)
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	if lines[0].GetLabelName() != "urlog" {
		t.Errorf("line 0: expected urlog, got %s", lines[0].GetLabelName())
	}
	if lines[1].GetLabelName() != "urlogURL" {
		t.Errorf("line 1: expected urlogURL, got %s", lines[1].GetLabelName())
	}
}

func TestStatement_AsciiTimeIsStart(t *testing.T) {
	text := "/start\nWork"
	lines := helperGenerateLines(t, text)
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	if lines[0].GetLabelName() != "timeIsStart" {
		t.Errorf("line 0: expected timeIsStart, got %s", lines[0].GetLabelName())
	}
	if lines[1].GetLabelName() != "timeIsStartTitle" {
		t.Errorf("line 1: expected timeIsStartTitle, got %s", lines[1].GetLabelName())
	}
}

func TestStatement_AsciiTimeIsEnd(t *testing.T) {
	text := "/end\nWork"
	lines := helperGenerateLines(t, text)
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	if lines[0].GetLabelName() != "timeIsEnd" {
		t.Errorf("line 0: expected timeIsEnd, got %s", lines[0].GetLabelName())
	}
	if lines[1].GetLabelName() != "timeIsEndTitle" {
		t.Errorf("line 1: expected timeIsEndTitle, got %s", lines[1].GetLabelName())
	}
}

func TestStatement_AsciiTimeIs(t *testing.T) {
	text := "/timeis\nMeeting"
	lines := helperGenerateLines(t, text)
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	if lines[0].GetLabelName() != "timeIs" {
		t.Errorf("line 0: expected timeIs, got %s", lines[0].GetLabelName())
	}
	if lines[1].GetLabelName() != "timeIsTitle" {
		t.Errorf("line 1: expected timeIsTitle, got %s", lines[1].GetLabelName())
	}
}

func TestStatement_AsciiTimeIsEndIfExist(t *testing.T) {
	text := "/end?\nWork"
	lines := helperGenerateLines(t, text)
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	if lines[0].GetLabelName() != "timeIsEndIfExist" {
		t.Errorf("line 0: expected timeIsEndIfExist, got %s", lines[0].GetLabelName())
	}
}

func TestStatement_AsciiTimeIsEndByTag(t *testing.T) {
	text := "/endt\n#work"
	lines := helperGenerateLines(t, text)
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	if lines[0].GetLabelName() != "timeIsEndByTag" {
		t.Errorf("line 0: expected timeIsEndByTag, got %s", lines[0].GetLabelName())
	}
	if lines[1].GetLabelName() != "timeIsEndByTagTag" {
		t.Errorf("line 1: expected timeIsEndByTagTag, got %s", lines[1].GetLabelName())
	}
}

func TestStatement_AsciiTimeIsEndByTagIfExist(t *testing.T) {
	text := "/endt?\n#work"
	lines := helperGenerateLines(t, text)
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	if lines[0].GetLabelName() != "timeIsEndByTagIfExist" {
		t.Errorf("line 0: expected timeIsEndByTagIfExist, got %s", lines[0].GetLabelName())
	}
}

// helperApplyToRequestMapAllowError is like helperApplyToRequestMap but returns
// the first error from ApplyThisLineToRequestMap instead of calling t.Fatal.
func helperApplyToRequestMapAllowError(t *testing.T, text string) (*KFTLRequestMap, error) {
	t.Helper()
	lines := helperGenerateLines(t, text)
	requestMap := NewKFTLRequestMap()
	for _, line := range lines {
		if err := line.ApplyThisLineToRequestMap(context.Background(), requestMap); err != nil {
			return requestMap, err
		}
	}
	return requestMap, nil
}

// ─── H4: Mi ASCII ? time fields ─────────────────────────────────────────────

func TestApply_AsciiMiLimitTime(t *testing.T) {
	// /mi + title + board(empty) + ?2025-01-01 → limitTime should be parsed
	text := "/mi\nTest Task\n\n?2025-01-01"
	requestMap := helperApplyToRequestMap(t, text)
	all := requestMap.All()
	if len(all) != 1 {
		t.Fatalf("expected 1 request, got %d", len(all))
	}
	miReq, ok := all[0].(*kftlMiRequest)
	if !ok {
		t.Fatalf("expected *kftlMiRequest, got %T", all[0])
	}
	if miReq.limitTime == nil {
		t.Fatal("expected limitTime to be set, got nil")
	}
}

func TestApply_AsciiMiEstimateStartTime(t *testing.T) {
	// /mi + title + board(empty) + limitTime(empty) + ?2025-06-01 10:00
	text := "/mi\nTest Task\n\n\n?2025-06-01 10:00"
	requestMap := helperApplyToRequestMap(t, text)
	all := requestMap.All()
	if len(all) != 1 {
		t.Fatalf("expected 1 request, got %d", len(all))
	}
	miReq, ok := all[0].(*kftlMiRequest)
	if !ok {
		t.Fatalf("expected *kftlMiRequest, got %T", all[0])
	}
	if miReq.estimateStartTime == nil {
		t.Fatal("expected estimateStartTime to be set, got nil")
	}
}

func TestApply_AsciiMiEstimateEndTime(t *testing.T) {
	// /mi + title + board(empty) + limitTime(empty) + estimateStartTime(empty) + ?2025-06-01 18:00
	text := "/mi\nTest Task\n\n\n\n?2025-06-01 18:00"
	requestMap := helperApplyToRequestMap(t, text)
	all := requestMap.All()
	if len(all) != 1 {
		t.Fatalf("expected 1 request, got %d", len(all))
	}
	miReq, ok := all[0].(*kftlMiRequest)
	if !ok {
		t.Fatalf("expected *kftlMiRequest, got %T", all[0])
	}
	if miReq.estimateEndTime == nil {
		t.Fatal("expected estimateEndTime to be set, got nil")
	}
}

// ─── M10: Nlog title/amount mismatch ────────────────────────────────────────

func TestApply_NlogTitleAmountMismatch(t *testing.T) {
	// 2 titles + 1 amount → should not error, creates 1 nlog (min of counts)
	text := "ーん\nTestShop\nTitle1\n100\nTitle2"
	requestMap, err := helperApplyToRequestMapAllowError(t, text)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	all := requestMap.All()
	if len(all) != 1 {
		t.Fatalf("expected 1 request, got %d", len(all))
	}
}

// ─── M11: Lantana mood range validation ─────────────────────────────────────

func TestApply_LantanaMood0(t *testing.T) {
	text := "ーら\n0"
	_, err := helperApplyToRequestMapAllowError(t, text)
	if err != nil {
		t.Fatalf("mood 0 should be valid, got error: %v", err)
	}
}

func TestApply_LantanaMood10(t *testing.T) {
	text := "ーら\n10"
	_, err := helperApplyToRequestMapAllowError(t, text)
	if err != nil {
		t.Fatalf("mood 10 should be valid, got error: %v", err)
	}
}

func TestApply_LantanaMood5(t *testing.T) {
	text := "ーら\n5"
	_, err := helperApplyToRequestMapAllowError(t, text)
	if err != nil {
		t.Fatalf("mood 5 should be valid, got error: %v", err)
	}
}

func TestApply_LantanaMood11(t *testing.T) {
	text := "ーら\n11"
	_, err := helperApplyToRequestMapAllowError(t, text)
	if err == nil {
		t.Fatal("mood 11 should be rejected, got no error")
	}
}

func TestApply_LantanaMoodNeg1(t *testing.T) {
	text := "ーら\n-1"
	_, err := helperApplyToRequestMapAllowError(t, text)
	if err == nil {
		t.Fatal("mood -1 should be rejected, got no error")
	}
}

func TestStatement_AsciiMixed(t *testing.T) {
	// Japanese tag + ASCII split + ASCII save character
	text := "。myTag\nhello\n,\nworld\n!\nignored"
	lines := helperGenerateLines(t, text)
	// tag + kmemo("hello") + split + kmemo("world") = 4 lines, "!" stops processing
	if len(lines) != 4 {
		t.Fatalf("expected 4 lines (mixed Japanese+ASCII), got %d", len(lines))
	}
	if lines[0].GetLabelName() != "tag" {
		t.Errorf("line 0: expected tag, got %s", lines[0].GetLabelName())
	}
	if lines[1].GetLabelName() != "kmemo" {
		t.Errorf("line 1: expected kmemo, got %s", lines[1].GetLabelName())
	}
	if lines[2].GetLabelName() != "split" {
		t.Errorf("line 2: expected split, got %s", lines[2].GetLabelName())
	}
	if lines[3].GetLabelName() != "kmemo" {
		t.Errorf("line 3: expected kmemo, got %s", lines[3].GetLabelName())
	}
}
