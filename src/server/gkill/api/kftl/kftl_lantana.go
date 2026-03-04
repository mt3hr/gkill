package kftl

import (
	"context"
	"fmt"
	"strconv"

	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
	"github.com/mt3hr/gkill/src/server/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_options"
)

// ─── KFTLLantanaRequest ───────────────────────────────────────────────────────

// kftlLantanaRequest records a mood value (0-10).
// Mirrors: kftl-lantana-request.ts
type kftlLantanaRequest struct {
	KFTLRequestBase
	mood int
}

func newKFTLLantanaRequest(requestID string, ctx *KFTLStatementLineContext) *kftlLantanaRequest {
	return &kftlLantanaRequest{
		KFTLRequestBase: KFTLRequestBase{
			RequestID:  requestID,
			Ctx:        ctx,
			CreateTime: nowFromCtx(ctx),
		},
	}
}

func (r *kftlLantanaRequest) DoRequest(ctx context.Context) error {
	if err := r.doBaseRequest(ctx, r.RequestID); err != nil {
		return err
	}
	relatedTime := r.GetRelatedTime()
	now := r.CreateTime
	lantana := reps.Lantana{
		ID:           r.RequestID,
		Mood:         r.mood,
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
	if err := r.Ctx.Repositories.WriteLantanaRep.AddLantanaInfo(ctx, lantana); err != nil {
		return err
	}
	repName, _ := r.Ctx.Repositories.WriteLantanaRep.GetRepName(ctx)
	updateLatestDataRepositoryAddress(ctx, r.Ctx.Repositories, r.RequestID, nil, false, now, repName)
	// キャッシュに書き込み
	if len(r.Ctx.Repositories.LantanaReps) == 1 && *gkill_options.CacheLantanaReps {
		_ = r.Ctx.Repositories.LantanaReps[0].AddLantanaInfo(ctx, lantana)
	}
	return nil
}

// ─── Statement lines ──────────────────────────────────────────────────────────

// kftlStartLantanaStatementLine handles "ーら".
// Mirrors: kftl-start-lantana-statement-line.ts
type kftlStartLantanaStatementLine struct {
	lineText string
	ctx      *KFTLStatementLineContext
	req      *kftlLantanaRequest
}

func newKFTLStartLantanaStatementLine(lineText string, ctx *KFTLStatementLineContext) *kftlStartLantanaStatementLine {
	prevLine := ctx.GetPrevLine()
	var targetID string
	if prevLine != nil && prevLine.GetContext().ThisIsPrototype {
		targetID = prevLine.GetContext().ThisStatementLineTargetID
	} else {
		targetID = sqlite3impl.GenerateNewID()
	}
	ctx.ThisStatementLineTargetID = targetID
	ctx.NextStatementLineTargetID = &targetID

	req := newKFTLLantanaRequest(targetID, ctx)
	ctx.NextStatementLineConstructor = func(lt string, c *KFTLStatementLineContext) KFTLStatementLine {
		return newKFTLLantanaMoodStatementLine(lt, c, req)
	}
	return &kftlStartLantanaStatementLine{lineText: lineText, ctx: ctx, req: req}
}

func (l *kftlStartLantanaStatementLine) ApplyThisLineToRequestMap(_ context.Context, requestMap *KFTLRequestMap) error {
	return requestMap.Set(l.ctx.ThisStatementLineTargetID, l.req)
}
func (l *kftlStartLantanaStatementLine) GetLabelName() string                 { return "lantana" }
func (l *kftlStartLantanaStatementLine) GetContext() *KFTLStatementLineContext { return l.ctx }
func (l *kftlStartLantanaStatementLine) GetStatementLineText() string         { return l.lineText }

// kftlLantanaMoodStatementLine reads the mood integer (0–10).
// Mirrors: kftl-lantana-mood-statement-line.ts
type kftlLantanaMoodStatementLine struct {
	lineText string
	ctx      *KFTLStatementLineContext
	req      *kftlLantanaRequest
}

func newKFTLLantanaMoodStatementLine(lineText string, ctx *KFTLStatementLineContext, req *kftlLantanaRequest) *kftlLantanaMoodStatementLine {
	targetID := ctx.ThisStatementLineTargetID
	ctx.NextStatementLineTargetID = &targetID
	ctx.NextStatementLineConstructor = ctx.factory.generateNoneConstructor(ctx.NextStatementLineText)
	return &kftlLantanaMoodStatementLine{lineText: lineText, ctx: ctx, req: req}
}

func (l *kftlLantanaMoodStatementLine) ApplyThisLineToRequestMap(_ context.Context, _ *KFTLRequestMap) error {
	n, err := strconv.Atoi(l.lineText)
	if err != nil {
		return fmt.Errorf("invalid lantana mood %q: %w", l.lineText, err)
	}
	l.req.mood = n
	return nil
}
func (l *kftlLantanaMoodStatementLine) GetLabelName() string                 { return "lantanaMood" }
func (l *kftlLantanaMoodStatementLine) GetContext() *KFTLStatementLineContext { return l.ctx }
func (l *kftlLantanaMoodStatementLine) GetStatementLineText() string         { return l.lineText }
