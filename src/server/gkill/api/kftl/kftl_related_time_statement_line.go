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

// timeOnlyFormats は時刻だけの書式（年月日を持たない）。base の年月日で補完する。
var timeOnlyFormats = map[string]struct{}{
	"15:04:05": {},
	"15:04":    {},
}

// parseDateTime attempts to parse a date string using multiple formats.
// base（呼び出し元の ctx.BaseTime）を「今日」の補完に使う。time.Now() を直接使わないのは
// テスト可能にするためと、1リクエスト内で時刻を一貫させるため。
func parseDateTime(s string, base time.Time) (time.Time, error) {
	s = strings.TrimSpace(s)
	for _, format := range dateFormats {
		t, err := time.ParseInLocation(format, s, time.Local)
		if err != nil {
			continue
		}
		if _, timeOnly := timeOnlyFormats[format]; timeOnly {
			// 時刻のみ → base の年月日に時刻を載せる。
			// 以前は年だけ補完して月日がゼロ値（1月1日）のまま保存されていた。
			return time.Date(base.Year(), base.Month(), base.Day(), t.Hour(), t.Minute(), t.Second(), 0, time.Local), nil
		}
		if t.Year() == 0 {
			// 「01/02 15:04」等の年だけ省略 → 年のみ base から補完（月日は入力を尊重する。
			// ここで月日まで上書きすると年省略入力が壊れるので分岐を分けている）。
			t = t.AddDate(base.Year(), 0, 0)
		}
		return t, nil
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

	// Parse the date (remove "？" or "?" prefix)
	dateStr := strings.TrimPrefix(l.lineText, splitterRelatedTime)
	dateStr = strings.TrimPrefix(dateStr, splitterRelatedTimeAscii)
	t, err := parseDateTime(dateStr, l.ctx.BaseTime)
	if err != nil {
		return fmt.Errorf("invalid related time %q: %w", dateStr, err)
	}
	req.SetRelatedTime(t)
	return nil
}

func (l *kftlRelatedTimeStatementLine) GetLabelName() string                  { return "relatedTime" }
func (l *kftlRelatedTimeStatementLine) GetContext() *KFTLStatementLineContext { return l.ctx }
func (l *kftlRelatedTimeStatementLine) GetStatementLineText() string          { return l.lineText }
