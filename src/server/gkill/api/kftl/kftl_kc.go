package kftl

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mt3hr/gkill/src/app/gkill/dao/reps"
	"github.com/mt3hr/gkill/src/app/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_options"
)

// ─── KFTLKCRequest ────────────────────────────────────────────────────────────

// kftlKCRequest records a KC (key-counter) entry.
// Mirrors: kftl-kc-request.ts
type kftlKCRequest struct {
	KFTLRequestBase
	title    string
	numValue json.Number
}

func newKFTLKCRequest(requestID string, ctx *KFTLStatementLineContext) *kftlKCRequest {
	return &kftlKCRequest{
		KFTLRequestBase: KFTLRequestBase{
			RequestID:  requestID,
			Ctx:        ctx,
			CreateTime: nowFromCtx(ctx),
		},
	}
}

func (r *kftlKCRequest) DoRequest(ctx context.Context) error {
	if err := r.doBaseRequest(ctx, r.RequestID); err != nil {
		return err
	}
	relatedTime := r.GetRelatedTime()
	now := r.CreateTime
	kc := reps.KC{
		ID:           r.RequestID,
		Title:        r.title,
		NumValue:     r.numValue,
		RelatedTime:  relatedTime,
		CreateTime:   now,
		CreateApp:    r.Ctx.ApplicationName,
		CreateDevice: r.Ctx.Device,
		CreateUser:   r.Ctx.UserID,
		UpdateTime:   now,
		UpdateApp:    r.Ctx.ApplicationName,
		UpdateDevice: r.Ctx.Device,
		UpdateUser:   r.Ctx.UserID,
	}
	if err := r.Ctx.Repositories.WriteKCRep.AddKCInfo(ctx, kc); err != nil {
		return err
	}
	repName, _ := r.Ctx.Repositories.WriteKCRep.GetRepName(ctx)
	updateLatestDataRepositoryAddress(ctx, r.Ctx.Repositories, r.RequestID, nil, false, now, repName)
	// キャッシュに書き込み
	if len(r.Ctx.Repositories.KCReps) == 1 && *gkill_options.CacheKCReps {
		_ = r.Ctx.Repositories.KCReps[0].AddKCInfo(ctx, kc)
	}
	return nil
}

// ─── Statement lines ──────────────────────────────────────────────────────────

// kftlStartKCStatementLine handles "ーか".
// Mirrors: kftl-start-kc-statement-line.ts
type kftlStartKCStatementLine struct {
	lineText string
	ctx      *KFTLStatementLineContext
	req      *kftlKCRequest
}

func newKFTLStartKCStatementLine(lineText string, ctx *KFTLStatementLineContext) *kftlStartKCStatementLine {
	prevLine := ctx.GetPrevLine()
	var targetID string
	if prevLine != nil && prevLine.GetContext().ThisIsPrototype {
		targetID = prevLine.GetContext().ThisStatementLineTargetID
	} else {
		targetID = sqlite3impl.GenerateNewID()
	}
	ctx.ThisStatementLineTargetID = targetID
	ctx.NextStatementLineTargetID = &targetID

	req := newKFTLKCRequest(targetID, ctx)
	ctx.NextStatementLineConstructor = func(lt string, c *KFTLStatementLineContext) KFTLStatementLine {
		return newKFTLKCTitleStatementLine(lt, c, req)
	}
	return &kftlStartKCStatementLine{lineText: lineText, ctx: ctx, req: req}
}

func (l *kftlStartKCStatementLine) ApplyThisLineToRequestMap(_ context.Context, requestMap *KFTLRequestMap) error {
	return requestMap.Set(l.ctx.ThisStatementLineTargetID, l.req)
}
func (l *kftlStartKCStatementLine) GetLabelName() string                 { return "kc" }
func (l *kftlStartKCStatementLine) GetContext() *KFTLStatementLineContext { return l.ctx }
func (l *kftlStartKCStatementLine) GetStatementLineText() string         { return l.lineText }

// kftlKCTitleStatementLine reads the KC title.
// Mirrors: kftl-kc-title-statement-line.ts
type kftlKCTitleStatementLine struct {
	lineText string
	ctx      *KFTLStatementLineContext
	req      *kftlKCRequest
}

func newKFTLKCTitleStatementLine(lineText string, ctx *KFTLStatementLineContext, req *kftlKCRequest) *kftlKCTitleStatementLine {
	targetID := ctx.ThisStatementLineTargetID
	ctx.NextStatementLineTargetID = &targetID
	ctx.NextStatementLineConstructor = func(lt string, c *KFTLStatementLineContext) KFTLStatementLine {
		return newKFTLKCNumValueStatementLine(lt, c, req)
	}
	return &kftlKCTitleStatementLine{lineText: lineText, ctx: ctx, req: req}
}

func (l *kftlKCTitleStatementLine) ApplyThisLineToRequestMap(_ context.Context, _ *KFTLRequestMap) error {
	l.req.title = l.lineText
	return nil
}
func (l *kftlKCTitleStatementLine) GetLabelName() string                 { return "kcTitle" }
func (l *kftlKCTitleStatementLine) GetContext() *KFTLStatementLineContext { return l.ctx }
func (l *kftlKCTitleStatementLine) GetStatementLineText() string         { return l.lineText }

// kftlKCNumValueStatementLine reads the KC numeric value.
// Mirrors: kftl-kc-num-value-statement-line.ts
type kftlKCNumValueStatementLine struct {
	lineText string
	ctx      *KFTLStatementLineContext
	req      *kftlKCRequest
}

func newKFTLKCNumValueStatementLine(lineText string, ctx *KFTLStatementLineContext, req *kftlKCRequest) *kftlKCNumValueStatementLine {
	targetID := ctx.ThisStatementLineTargetID
	ctx.NextStatementLineTargetID = &targetID
	ctx.NextStatementLineConstructor = ctx.factory.generateNoneConstructor(ctx.NextStatementLineText)
	return &kftlKCNumValueStatementLine{lineText: lineText, ctx: ctx, req: req}
}

func (l *kftlKCNumValueStatementLine) ApplyThisLineToRequestMap(_ context.Context, _ *KFTLRequestMap) error {
	if l.lineText == "" {
		return fmt.Errorf("kc num_value is empty")
	}
	l.req.numValue = json.Number(l.lineText)
	// Validate it's actually a number
	if _, err := l.req.numValue.Float64(); err != nil {
		return fmt.Errorf("invalid kc num_value %q: %w", l.lineText, err)
	}
	return nil
}
func (l *kftlKCNumValueStatementLine) GetLabelName() string                 { return "kcNumValue" }
func (l *kftlKCNumValueStatementLine) GetContext() *KFTLStatementLineContext { return l.ctx }
func (l *kftlKCNumValueStatementLine) GetStatementLineText() string         { return l.lineText }
