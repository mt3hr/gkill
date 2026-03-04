package kftl

import (
	"context"
	"fmt"
)

// kftlNoneStatementLine represents a "done" state — no more processing for this entity.
// If a non-empty line arrives, it is an error.
// Mirrors: src/classes/kftl/kftl_none/kftl-none-statement-line.ts
type kftlNoneStatementLine struct {
	lineText string
	ctx      *KFTLStatementLineContext
}

func newKFTLNoneStatementLine(lineText string, ctx *KFTLStatementLineContext) *kftlNoneStatementLine {
	ctx.ThisIsPrototype = true
	ctx.NextIsPrototype = true
	targetID := ctx.ThisStatementLineTargetID
	ctx.NextStatementLineTargetID = &targetID
	// Next line is also None (but re-evaluated by factory for meta-info patterns)
	ctx.NextStatementLineConstructor = ctx.factory.generateNoneConstructor(ctx.NextStatementLineText)
	return &kftlNoneStatementLine{lineText: lineText, ctx: ctx}
}

func (l *kftlNoneStatementLine) ApplyThisLineToRequestMap(_ context.Context, _ *KFTLRequestMap) error {
	if l.lineText != "" {
		return fmt.Errorf("unexpected non-empty line in none-state: %q", l.lineText)
	}
	return nil
}

func (l *kftlNoneStatementLine) GetLabelName() string                 { return "none" }
func (l *kftlNoneStatementLine) GetContext() *KFTLStatementLineContext { return l.ctx }
func (l *kftlNoneStatementLine) GetStatementLineText() string         { return l.lineText }
