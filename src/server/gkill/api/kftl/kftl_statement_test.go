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
