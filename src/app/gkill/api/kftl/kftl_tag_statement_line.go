package kftl

import (
	"context"
	"strings"
)

// kftlTagStatementLine handles tag lines (starting with "。").
// Mirrors: src/classes/kftl/kftl_tag/kftl-tag-statement-line.ts
type kftlTagStatementLine struct {
	lineText string
	ctx      *KFTLStatementLineContext
}

func newKFTLTagStatementLine(lineText string, ctx *KFTLStatementLineContext, prevLineIsMetaInfo bool) *kftlTagStatementLine {
	// next target is same as this; prototype flag is inherited
	ctx.NextIsPrototype = ctx.ThisIsPrototype
	targetID := ctx.ThisStatementLineTargetID
	ctx.NextStatementLineTargetID = &targetID

	// Use factory to decide whether next line is Kmemo or None
	if prevLineIsMetaInfo {
		ctx.NextStatementLineConstructor = ctx.factory.generateKmemoConstructor(ctx.NextStatementLineText)
	} else {
		ctx.NextStatementLineConstructor = ctx.factory.generateNoneConstructor(ctx.NextStatementLineText)
	}

	return &kftlTagStatementLine{lineText: lineText, ctx: ctx}
}

func (l *kftlTagStatementLine) ApplyThisLineToRequestMap(_ context.Context, requestMap *KFTLRequestMap) error {
	targetID := l.ctx.ThisStatementLineTargetID

	// Get or create prototype request for this target
	req, ok := requestMap.Get(targetID)
	if !ok {
		proto := newKFTLPrototypeRequest(targetID, l.ctx)
		if err := requestMap.Set(targetID, proto); err != nil {
			return err
		}
		req, _ = requestMap.Get(targetID)
	}

	// Parse tags: remove "。" prefix, split by "、"
	tagStr := strings.TrimPrefix(l.lineText, splitterTag)
	tags := strings.Split(tagStr, "、")
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag != "" {
			req.AddTag(tag)
		}
	}
	return nil
}

func (l *kftlTagStatementLine) GetLabelName() string                 { return "tag" }
func (l *kftlTagStatementLine) GetContext() *KFTLStatementLineContext { return l.ctx }
func (l *kftlTagStatementLine) GetStatementLineText() string         { return l.lineText }
