package kftl

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// dateFormats lists time formats tried when parsing KFTL related-time strings.
// Mirrors: moment() parsing in the TypeScript implementation.
var dateFormats = []string{
	"2006-01-02 15:04:05",
	"2006-01-02T15:04:05",
	"2006/01/02 15:04:05",
	"2006/01/02T15:04:05",
	"2006-01-02 15:04",
	"2006/01/02 15:04",
	"2006-01-02",
	"2006/01/02",
	"01/02 15:04",
	"1/2 15:04",
	"15:04:05",
	"15:04",
}

// parseDateTime attempts to parse a date string using multiple formats.
func parseDateTime(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	now := time.Now()
	for _, fmt_ := range dateFormats {
		t, err := time.ParseInLocation(fmt_, s, time.Local)
		if err == nil {
			// Fill in missing year/month/day from now
			if t.Year() == 0 {
				t = t.AddDate(now.Year(), 0, 0)
			}
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse date: %q", s)
}

// kftlRelatedTimeStatementLine handles "？datetime" lines.
// Mirrors: src/classes/kftl/kftl_related_time/kftl-related-time-statement-line.ts
type kftlRelatedTimeStatementLine struct {
	lineText string
	ctx      *KFTLStatementLineContext
}

func newKFTLRelatedTimeStatementLine(lineText string, ctx *KFTLStatementLineContext, prevLineIsMetaInfo bool) *kftlRelatedTimeStatementLine {
	ctx.NextIsPrototype = ctx.ThisIsPrototype
	targetID := ctx.ThisStatementLineTargetID
	ctx.NextStatementLineTargetID = &targetID

	if prevLineIsMetaInfo {
		ctx.NextStatementLineConstructor = ctx.factory.generateKmemoConstructor(ctx.NextStatementLineText)
	} else {
		ctx.NextStatementLineConstructor = ctx.factory.generateNoneConstructor(ctx.NextStatementLineText)
	}

	return &kftlRelatedTimeStatementLine{lineText: lineText, ctx: ctx}
}

func (l *kftlRelatedTimeStatementLine) ApplyThisLineToRequestMap(_ context.Context, requestMap *KFTLRequestMap) error {
	targetID := l.ctx.ThisStatementLineTargetID

	req, ok := requestMap.Get(targetID)
	if !ok {
		proto := newKFTLPrototypeRequest(targetID, l.ctx)
		if err := requestMap.Set(targetID, proto); err != nil {
			return err
		}
		req, _ = requestMap.Get(targetID)
	}

	// Parse the date (remove "？" prefix)
	dateStr := strings.TrimPrefix(l.lineText, splitterRelatedTime)
	t, err := parseDateTime(dateStr)
	if err != nil {
		return fmt.Errorf("invalid related time %q: %w", dateStr, err)
	}
	req.SetRelatedTime(t)
	return nil
}

func (l *kftlRelatedTimeStatementLine) GetLabelName() string                 { return "relatedTime" }
func (l *kftlRelatedTimeStatementLine) GetContext() *KFTLStatementLineContext { return l.ctx }
func (l *kftlRelatedTimeStatementLine) GetStatementLineText() string         { return l.lineText }
