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

// parseScheduleFieldTime は Mi / MiReKyou の予定日時欄(見積開始・見積終了・期限)の1行を読む。
//
// 行頭の「？」「?」は**入力エラー**にする。以前は関連時刻と同じ接頭辞として黙って剥がしていたが、
// 剥がしたあとにパースへ失敗しても未設定として握り潰す作りなので、「？18:00」の打ち間違いも
// 「？？」(繰り返しブロック)の書き損じも、エラーも警告も出ないまま日付だけが入らない形で
// 落ちていた。剥がす前に弾く。Mi は related_time 列を持たないので、この欄に関連時刻の
// 接頭辞を書けること自体に意味が無い。
//
// 空行は「未設定」（空行で項目の位置を送る書き方のため）。空でないのに日時として読めない行は
// **入力エラー**にする。2026-09-15 までは未設定として握り潰していて（ADR-0505 はそこを据え置いた）、
// `ーみ`,`タイトル`,`板`,`abc` の `abc` も、6行を埋めずに `、` で次の記録へ移ろうとした `、` も、
// エラーも警告も出ないまま日付だけが入らず、`、` の後ろの本文まで残りの欄に食われていた。
// タグ行・テキスト開始行・`？？` は generateMiBlockNextConstructor / generateMiReKyouNextConstructor が
// 先読みで拾って項目の位置を消費しないので、ここへは来ない（ADR-0508）。
func parseScheduleFieldTime(lineText string, base time.Time) (time.Time, bool, error) {
	if strings.HasPrefix(lineText, splitterRelatedTime) || strings.HasPrefix(lineText, splitterRelatedTimeAscii) {
		return time.Time{}, false, newKFTLInputError("KFTL_SCHEDULE_TIME_PREFIX_NOT_ALLOWED_MESSAGE_TITLE",
			fmt.Errorf("related time prefix is not allowed in a schedule datetime field: %q", lineText))
	}
	if strings.TrimSpace(lineText) == "" {
		return time.Time{}, false, nil
	}
	t, err := parseDateTime(lineText, base)
	if err != nil {
		return time.Time{}, false, newKFTLInputError("KFTL_TIMEIS_INVALID_PARSE_TIME_ERROR_MESSAGE_TITLE",
			fmt.Errorf("invalid schedule datetime %q: %w", lineText, err))
	}
	return t, true, nil
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
		return newKFTLInputError("KFTL_INVALID_PARSE_RELATED_TIME_ERROR_MESSAGE_TITLE",
			fmt.Errorf("invalid related time %q: %w", dateStr, err))
	}
	req.SetRelatedTime(t)
	return nil
}

func (l *kftlRelatedTimeStatementLine) GetLabelName() string                  { return "relatedTime" }
func (l *kftlRelatedTimeStatementLine) GetContext() *KFTLStatementLineContext { return l.ctx }
func (l *kftlRelatedTimeStatementLine) GetStatementLineText() string          { return l.lineText }
