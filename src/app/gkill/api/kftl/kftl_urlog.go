package kftl

import (
	"context"
	"fmt"

	"github.com/mt3hr/gkill/src/app/gkill/dao/reps"
	"github.com/mt3hr/gkill/src/app/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_options"
)

// ─── KFTLURLogRequest ─────────────────────────────────────────────────────────

// kftlURLogRequest records a URLog (URL bookmark) entry.
// Mirrors: kftlur-log-request.ts
type kftlURLogRequest struct {
	KFTLRequestBase
	url   string
	title string
}

func newKFTLURLogRequest(requestID string, ctx *KFTLStatementLineContext) *kftlURLogRequest {
	return &kftlURLogRequest{
		KFTLRequestBase: KFTLRequestBase{
			RequestID:  requestID,
			Ctx:        ctx,
			CreateTime: nowFromCtx(ctx),
		},
	}
}

func (r *kftlURLogRequest) DoRequest(ctx context.Context) error {
	if r.url == "" && r.title == "" {
		return nil // skip blank URLog
	}

	if err := r.doBaseRequest(ctx, r.RequestID); err != nil {
		return err
	}

	relatedTime := r.GetRelatedTime()
	now := r.CreateTime
	urlog := reps.URLog{
		ID:           r.RequestID,
		URL:          r.url,
		Title:        r.title,
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
	if err := r.Ctx.Repositories.WriteURLogRep.AddURLogInfo(ctx, urlog); err != nil {
		return fmt.Errorf("error at add urlog info id=%s: %w", r.RequestID, err)
	}
	repName, _ := r.Ctx.Repositories.WriteURLogRep.GetRepName(ctx)
	updateLatestDataRepositoryAddress(ctx, r.Ctx.Repositories, r.RequestID, nil, false, now, repName)
	// キャッシュに書き込み
	if len(r.Ctx.Repositories.URLogReps) == 1 && *gkill_options.CacheURLogReps {
		_ = r.Ctx.Repositories.URLogReps[0].AddURLogInfo(ctx, urlog)
	}
	return nil
}

// ─── Statement lines ──────────────────────────────────────────────────────────

// kftlStartURLogStatementLine handles "ーう".
// Mirrors: kftl-start-ur-log-statement-line.ts
type kftlStartURLogStatementLine struct {
	lineText string
	ctx      *KFTLStatementLineContext
	req      *kftlURLogRequest
}

func newKFTLStartURLogStatementLine(lineText string, ctx *KFTLStatementLineContext) *kftlStartURLogStatementLine {
	prevLine := ctx.GetPrevLine()
	var targetID string
	if prevLine != nil && prevLine.GetContext().ThisIsPrototype {
		targetID = prevLine.GetContext().ThisStatementLineTargetID
	} else {
		targetID = sqlite3impl.GenerateNewID()
	}
	ctx.ThisStatementLineTargetID = targetID
	ctx.NextStatementLineTargetID = &targetID

	req := newKFTLURLogRequest(targetID, ctx)
	ctx.NextStatementLineConstructor = func(lt string, c *KFTLStatementLineContext) KFTLStatementLine {
		return newKFTLURLogURLStatementLine(lt, c, req)
	}
	return &kftlStartURLogStatementLine{lineText: lineText, ctx: ctx, req: req}
}

func (l *kftlStartURLogStatementLine) ApplyThisLineToRequestMap(_ context.Context, requestMap *KFTLRequestMap) error {
	return requestMap.Set(l.ctx.ThisStatementLineTargetID, l.req)
}
func (l *kftlStartURLogStatementLine) GetLabelName() string                 { return "urlog" }
func (l *kftlStartURLogStatementLine) GetContext() *KFTLStatementLineContext { return l.ctx }
func (l *kftlStartURLogStatementLine) GetStatementLineText() string         { return l.lineText }

// kftlURLogURLStatementLine reads the URL.
// Mirrors: kftlur-log-url-statement-line.ts
type kftlURLogURLStatementLine struct {
	lineText string
	ctx      *KFTLStatementLineContext
	req      *kftlURLogRequest
}

func newKFTLURLogURLStatementLine(lineText string, ctx *KFTLStatementLineContext, req *kftlURLogRequest) *kftlURLogURLStatementLine {
	targetID := ctx.ThisStatementLineTargetID
	ctx.NextStatementLineTargetID = &targetID
	ctx.NextStatementLineConstructor = func(lt string, c *KFTLStatementLineContext) KFTLStatementLine {
		return newKFTLURLogTitleStatementLine(lt, c, req)
	}
	return &kftlURLogURLStatementLine{lineText: lineText, ctx: ctx, req: req}
}

func (l *kftlURLogURLStatementLine) ApplyThisLineToRequestMap(_ context.Context, _ *KFTLRequestMap) error {
	l.req.url = l.lineText
	return nil
}
func (l *kftlURLogURLStatementLine) GetLabelName() string                 { return "urlogURL" }
func (l *kftlURLogURLStatementLine) GetContext() *KFTLStatementLineContext { return l.ctx }
func (l *kftlURLogURLStatementLine) GetStatementLineText() string         { return l.lineText }

// kftlURLogTitleStatementLine reads the URL title.
// After this, next constructor reverts to None.
// Mirrors: kftlur-log-title-statement-line.ts
type kftlURLogTitleStatementLine struct {
	lineText string
	ctx      *KFTLStatementLineContext
	req      *kftlURLogRequest
}

func newKFTLURLogTitleStatementLine(lineText string, ctx *KFTLStatementLineContext, req *kftlURLogRequest) *kftlURLogTitleStatementLine {
	targetID := ctx.ThisStatementLineTargetID
	ctx.NextStatementLineTargetID = &targetID
	ctx.NextStatementLineConstructor = ctx.factory.generateNoneConstructor(ctx.NextStatementLineText)
	return &kftlURLogTitleStatementLine{lineText: lineText, ctx: ctx, req: req}
}

func (l *kftlURLogTitleStatementLine) ApplyThisLineToRequestMap(_ context.Context, _ *KFTLRequestMap) error {
	l.req.title = l.lineText
	return nil
}
func (l *kftlURLogTitleStatementLine) GetLabelName() string                 { return "urlogTitle" }
func (l *kftlURLogTitleStatementLine) GetContext() *KFTLStatementLineContext { return l.ctx }
func (l *kftlURLogTitleStatementLine) GetStatementLineText() string         { return l.lineText }
