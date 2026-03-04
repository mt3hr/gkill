package kftl

import (
	"context"

	"github.com/mt3hr/gkill/src/app/gkill/dao/sqlite3impl"
)

// kftlSplitStatementLine separates two entities.
// Sets a fresh target_id for the current line and clears next_target_id
// so the next entity gets a new UUID.
// Mirrors: src/classes/kftl/kftl_split/kftl-split-statement-line.ts
type kftlSplitStatementLine struct {
	lineText string
	ctx      *KFTLStatementLineContext
}

func newKFTLSplitStatementLine(lineText string, ctx *KFTLStatementLineContext) *kftlSplitStatementLine {
	ctx.ThisStatementLineTargetID = sqlite3impl.GenerateNewID()
	ctx.ThisIsPrototype = true
	ctx.NextIsPrototype = true
	ctx.NextStatementLineTargetID = nil // next entity gets a fresh UUID
	ctx.NextStatementLineConstructor = ctx.factory.generateKmemoConstructor(ctx.NextStatementLineText)
	return &kftlSplitStatementLine{lineText: lineText, ctx: ctx}
}

func (l *kftlSplitStatementLine) ApplyThisLineToRequestMap(_ context.Context, _ *KFTLRequestMap) error {
	return nil
}

func (l *kftlSplitStatementLine) GetLabelName() string                 { return "split" }
func (l *kftlSplitStatementLine) GetContext() *KFTLStatementLineContext { return l.ctx }
func (l *kftlSplitStatementLine) GetStatementLineText() string         { return l.lineText }

// kftlSplitAndNextSecondStatementLine is like Split but also increments the
// add_second counter so subsequent entities are timestamped one second later.
// Mirrors: src/classes/kftl/kftl_split/kftl-split-and-next-second-statement-line.ts
type kftlSplitAndNextSecondStatementLine struct {
	lineText string
	ctx      *KFTLStatementLineContext
}

func newKFTLSplitAndNextSecondStatementLine(lineText string, ctx *KFTLStatementLineContext) *kftlSplitAndNextSecondStatementLine {
	ctx.ThisStatementLineTargetID = sqlite3impl.GenerateNewID()
	ctx.ThisIsPrototype = true
	ctx.NextIsPrototype = true
	ctx.NextStatementLineTargetID = nil
	ctx.NextStatementLineConstructor = ctx.factory.generateKmemoConstructor(ctx.NextStatementLineText)
	return &kftlSplitAndNextSecondStatementLine{lineText: lineText, ctx: ctx}
}

func (l *kftlSplitAndNextSecondStatementLine) ApplyThisLineToRequestMap(_ context.Context, _ *KFTLRequestMap) error {
	return nil
}

func (l *kftlSplitAndNextSecondStatementLine) GetLabelName() string                 { return "split+1s" }
func (l *kftlSplitAndNextSecondStatementLine) GetContext() *KFTLStatementLineContext { return l.ctx }
func (l *kftlSplitAndNextSecondStatementLine) GetStatementLineText() string         { return l.lineText }
