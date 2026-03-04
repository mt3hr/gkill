package kftl

import (
	"context"

	"github.com/mt3hr/gkill/src/server/gkill/dao/sqlite3impl"
)

// kftlStartTextStatementLine opens a text block (line == "ーー").
// Generates a new textID and sets the next constructor to KFTLTextStatementLine.
// Mirrors: src/classes/kftl/kftl_text/kftl-start-text-statement-line.ts
type kftlStartTextStatementLine struct {
	lineText string
	ctx      *KFTLStatementLineContext
	textID   string
}

func newKFTLStartTextStatementLine(lineText string, ctx *KFTLStatementLineContext, prevLineIsMetaInfo bool) *kftlStartTextStatementLine {
	ctx.NextIsPrototype = ctx.ThisIsPrototype
	targetID := ctx.ThisStatementLineTargetID
	ctx.NextStatementLineTargetID = &targetID

	textID := sqlite3impl.GenerateNewID()
	ctx.NextStatementLineConstructor = func(lt string, c *KFTLStatementLineContext) KFTLStatementLine {
		return newKFTLTextStatementLine(lt, c, textID, prevLineIsMetaInfo)
	}

	return &kftlStartTextStatementLine{lineText: lineText, ctx: ctx, textID: textID}
}

func (l *kftlStartTextStatementLine) ApplyThisLineToRequestMap(_ context.Context, requestMap *KFTLRequestMap) error {
	// Ensure a prototype exists so the text can be attached later
	targetID := l.ctx.ThisStatementLineTargetID
	_, ok := requestMap.Get(targetID)
	if !ok {
		proto := newKFTLPrototypeRequest(targetID, l.ctx)
		return requestMap.Set(targetID, proto)
	}
	return nil
}

func (l *kftlStartTextStatementLine) GetLabelName() string                 { return "startText" }
func (l *kftlStartTextStatementLine) GetContext() *KFTLStatementLineContext { return l.ctx }
func (l *kftlStartTextStatementLine) GetStatementLineText() string         { return l.lineText }

// kftlTextStatementLine accumulates lines into a text block on the request.
// Mirrors: src/classes/kftl/kftl_text/kftl-text-statement-line.ts
type kftlTextStatementLine struct {
	lineText           string
	ctx                *KFTLStatementLineContext
	textID             string
	prevLineIsMetaInfo bool
}

func newKFTLTextStatementLine(lineText string, ctx *KFTLStatementLineContext, textID string, prevLineIsMetaInfo bool) *kftlTextStatementLine {
	ctx.NextIsPrototype = ctx.ThisIsPrototype
	targetID := ctx.ThisStatementLineTargetID
	ctx.NextStatementLineTargetID = &targetID

	// If next line is "ーー", it ends the text block; otherwise continue accumulating
	if ctx.NextStatementLineText == splitterStartText {
		ctx.NextStatementLineConstructor = func(lt string, c *KFTLStatementLineContext) KFTLStatementLine {
			return newKFTLEndTextStatementLine(lt, c, prevLineIsMetaInfo)
		}
	} else {
		ctx.NextStatementLineConstructor = func(lt string, c *KFTLStatementLineContext) KFTLStatementLine {
			return newKFTLTextStatementLine(lt, c, textID, prevLineIsMetaInfo)
		}
	}

	return &kftlTextStatementLine{lineText: lineText, ctx: ctx, textID: textID, prevLineIsMetaInfo: prevLineIsMetaInfo}
}

func (l *kftlTextStatementLine) ApplyThisLineToRequestMap(_ context.Context, requestMap *KFTLRequestMap) error {
	targetID := l.ctx.ThisStatementLineTargetID
	req, ok := requestMap.Get(targetID)
	if !ok {
		proto := newKFTLPrototypeRequest(targetID, l.ctx)
		if err := requestMap.Set(targetID, proto); err != nil {
			return err
		}
		req, _ = requestMap.Get(targetID)
	}
	req.AddTextLine(l.textID, l.lineText)
	return nil
}

func (l *kftlTextStatementLine) GetLabelName() string                 { return "text" }
func (l *kftlTextStatementLine) GetContext() *KFTLStatementLineContext { return l.ctx }
func (l *kftlTextStatementLine) GetStatementLineText() string         { return l.lineText }

// kftlEndTextStatementLine closes a text block (line == "ーー").
// Mirrors: src/classes/kftl/kftl_text/kftl-end-text-statement-line.ts
type kftlEndTextStatementLine struct {
	lineText string
	ctx      *KFTLStatementLineContext
}

func newKFTLEndTextStatementLine(lineText string, ctx *KFTLStatementLineContext, prevLineIsMetaInfo bool) *kftlEndTextStatementLine {
	ctx.NextIsPrototype = ctx.ThisIsPrototype
	targetID := ctx.ThisStatementLineTargetID
	ctx.NextStatementLineTargetID = &targetID

	if prevLineIsMetaInfo {
		ctx.NextStatementLineConstructor = ctx.factory.generateKmemoConstructor(ctx.NextStatementLineText)
	} else {
		ctx.NextStatementLineConstructor = ctx.factory.generateNoneConstructor(ctx.NextStatementLineText)
	}

	return &kftlEndTextStatementLine{lineText: lineText, ctx: ctx}
}

func (l *kftlEndTextStatementLine) ApplyThisLineToRequestMap(_ context.Context, requestMap *KFTLRequestMap) error {
	// Close the current text block on the request (set current_text_id = nil)
	targetID := l.ctx.ThisStatementLineTargetID
	if req, ok := requestMap.Get(targetID); ok {
		req.SetCurrentTextID(nil)
	}
	return nil
}

func (l *kftlEndTextStatementLine) GetLabelName() string                 { return "endText" }
func (l *kftlEndTextStatementLine) GetContext() *KFTLStatementLineContext { return l.ctx }
func (l *kftlEndTextStatementLine) GetStatementLineText() string         { return l.lineText }
