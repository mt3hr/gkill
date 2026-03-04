package kftl

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mt3hr/gkill/src/app/gkill/dao/reps"
	"github.com/mt3hr/gkill/src/app/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_options"
)

// ─── KFTLMiRequest ────────────────────────────────────────────────────────────

// kftlMiRequest records a Mi (TODO) entry.
// Mirrors: kftl-mi-request.ts
type kftlMiRequest struct {
	KFTLRequestBase
	title             string
	boardName         string
	limitTime         *time.Time
	estimateStartTime *time.Time
	estimateEndTime   *time.Time
}

func newKFTLMiRequest(requestID string, ctx *KFTLStatementLineContext) *kftlMiRequest {
	return &kftlMiRequest{
		KFTLRequestBase: KFTLRequestBase{
			RequestID:  requestID,
			Ctx:        ctx,
			CreateTime: nowFromCtx(ctx),
		},
	}
}

func (r *kftlMiRequest) DoRequest(ctx context.Context) error {
	if r.title == "" {
		return nil // skip blank Mi
	}

	boardName := r.boardName
	if boardName == "" && r.Ctx.ApplicationConfig != nil {
		boardName = r.Ctx.ApplicationConfig.MiDefaultBoard
	}

	if err := r.doBaseRequest(ctx, r.RequestID); err != nil {
		return err
	}

	now := r.CreateTime
	mi := reps.Mi{
		ID:                r.RequestID,
		Title:             r.title,
		BoardName:         boardName,
		IsChecked:         false,
		LimitTime:         r.limitTime,
		EstimateStartTime: r.estimateStartTime,
		EstimateEndTime:   r.estimateEndTime,
		CreateTime:        now,
		CreateApp:         r.Ctx.ApplicationName,
		CreateDevice:      r.Ctx.Device,
		CreateUser:        r.Ctx.UserID,
		UpdateTime:        now,
		UpdateApp:         r.Ctx.ApplicationName,
		UpdateDevice:      r.Ctx.Device,
		UpdateUser:        r.Ctx.UserID,
	}
	if err := r.Ctx.Repositories.WriteMiRep.AddMiInfo(ctx, mi); err != nil {
		return fmt.Errorf("error at add mi info id=%s: %w", r.RequestID, err)
	}
	repName, _ := r.Ctx.Repositories.WriteMiRep.GetRepName(ctx)
	updateLatestDataRepositoryAddress(ctx, r.Ctx.Repositories, r.RequestID, nil, false, now, repName)
	// キャッシュに書き込み
	if len(r.Ctx.Repositories.MiReps) == 1 && *gkill_options.CacheMiReps {
		_ = r.Ctx.Repositories.MiReps[0].AddMiInfo(ctx, mi)
	}
	return nil
}

// ─── Statement lines ──────────────────────────────────────────────────────────

// kftlStartMiStatementLine handles "ーみ".
// Mirrors: kftl-start-mi-statement-line.ts
type kftlStartMiStatementLine struct {
	lineText string
	ctx      *KFTLStatementLineContext
	req      *kftlMiRequest
}

func newKFTLStartMiStatementLine(lineText string, ctx *KFTLStatementLineContext) *kftlStartMiStatementLine {
	prevLine := ctx.GetPrevLine()
	var targetID string
	if prevLine != nil && prevLine.GetContext().ThisIsPrototype {
		targetID = prevLine.GetContext().ThisStatementLineTargetID
	} else {
		targetID = sqlite3impl.GenerateNewID()
	}
	ctx.ThisStatementLineTargetID = targetID
	ctx.NextStatementLineTargetID = &targetID

	req := newKFTLMiRequest(targetID, ctx)
	ctx.NextStatementLineConstructor = func(lt string, c *KFTLStatementLineContext) KFTLStatementLine {
		return newKFTLMiTitleStatementLine(lt, c, req)
	}
	return &kftlStartMiStatementLine{lineText: lineText, ctx: ctx, req: req}
}

func (l *kftlStartMiStatementLine) ApplyThisLineToRequestMap(_ context.Context, requestMap *KFTLRequestMap) error {
	return requestMap.Set(l.ctx.ThisStatementLineTargetID, l.req)
}
func (l *kftlStartMiStatementLine) GetLabelName() string                 { return "mi" }
func (l *kftlStartMiStatementLine) GetContext() *KFTLStatementLineContext { return l.ctx }
func (l *kftlStartMiStatementLine) GetStatementLineText() string         { return l.lineText }

// kftlMiTitleStatementLine reads the Mi title.
// Mirrors: kftl-mi-title-statement-line.ts
type kftlMiTitleStatementLine struct {
	lineText string
	ctx      *KFTLStatementLineContext
	req      *kftlMiRequest
}

func newKFTLMiTitleStatementLine(lineText string, ctx *KFTLStatementLineContext, req *kftlMiRequest) *kftlMiTitleStatementLine {
	targetID := ctx.ThisStatementLineTargetID
	ctx.NextStatementLineTargetID = &targetID
	ctx.NextStatementLineConstructor = func(lt string, c *KFTLStatementLineContext) KFTLStatementLine {
		return newKFTLMiBoardNameStatementLine(lt, c, req)
	}
	return &kftlMiTitleStatementLine{lineText: lineText, ctx: ctx, req: req}
}

func (l *kftlMiTitleStatementLine) ApplyThisLineToRequestMap(_ context.Context, _ *KFTLRequestMap) error {
	l.req.title = l.lineText
	return nil
}
func (l *kftlMiTitleStatementLine) GetLabelName() string                 { return "miTitle" }
func (l *kftlMiTitleStatementLine) GetContext() *KFTLStatementLineContext { return l.ctx }
func (l *kftlMiTitleStatementLine) GetStatementLineText() string         { return l.lineText }

// kftlMiBoardNameStatementLine reads the Mi board name.
// Mirrors: kftl-mi-board-name-statement-line.ts
type kftlMiBoardNameStatementLine struct {
	lineText string
	ctx      *KFTLStatementLineContext
	req      *kftlMiRequest
}

func newKFTLMiBoardNameStatementLine(lineText string, ctx *KFTLStatementLineContext, req *kftlMiRequest) *kftlMiBoardNameStatementLine {
	targetID := ctx.ThisStatementLineTargetID
	ctx.NextStatementLineTargetID = &targetID
	ctx.NextStatementLineConstructor = func(lt string, c *KFTLStatementLineContext) KFTLStatementLine {
		return newKFTLMiLimitTimeStatementLine(lt, c, req)
	}
	return &kftlMiBoardNameStatementLine{lineText: lineText, ctx: ctx, req: req}
}

func (l *kftlMiBoardNameStatementLine) ApplyThisLineToRequestMap(_ context.Context, _ *KFTLRequestMap) error {
	l.req.boardName = l.lineText
	return nil
}
func (l *kftlMiBoardNameStatementLine) GetLabelName() string                 { return "miBoardName" }
func (l *kftlMiBoardNameStatementLine) GetContext() *KFTLStatementLineContext { return l.ctx }
func (l *kftlMiBoardNameStatementLine) GetStatementLineText() string         { return l.lineText }

// kftlMiLimitTimeStatementLine reads the Mi limit time (optional, may be empty).
// Mirrors: kftl-mi-limit-time-statement-line.ts
type kftlMiLimitTimeStatementLine struct {
	lineText string
	ctx      *KFTLStatementLineContext
	req      *kftlMiRequest
}

func newKFTLMiLimitTimeStatementLine(lineText string, ctx *KFTLStatementLineContext, req *kftlMiRequest) *kftlMiLimitTimeStatementLine {
	targetID := ctx.ThisStatementLineTargetID
	ctx.NextStatementLineTargetID = &targetID
	ctx.NextStatementLineConstructor = func(lt string, c *KFTLStatementLineContext) KFTLStatementLine {
		return newKFTLMiEstimateStartTimeStatementLine(lt, c, req)
	}
	return &kftlMiLimitTimeStatementLine{lineText: lineText, ctx: ctx, req: req}
}

func (l *kftlMiLimitTimeStatementLine) ApplyThisLineToRequestMap(_ context.Context, _ *KFTLRequestMap) error {
	s := strings.TrimPrefix(l.lineText, "？")
	if s == "" {
		return nil // optional
	}
	t, err := parseDateTime(s)
	if err != nil {
		return nil // invalid → skip silently (mirrors TS: isNaN check)
	}
	l.req.limitTime = &t
	return nil
}
func (l *kftlMiLimitTimeStatementLine) GetLabelName() string                 { return "miLimitTime" }
func (l *kftlMiLimitTimeStatementLine) GetContext() *KFTLStatementLineContext { return l.ctx }
func (l *kftlMiLimitTimeStatementLine) GetStatementLineText() string         { return l.lineText }

// kftlMiEstimateStartTimeStatementLine reads the Mi estimate start time (optional).
// Mirrors: kftl-mi-estimate-start-time-statement-line.ts
type kftlMiEstimateStartTimeStatementLine struct {
	lineText string
	ctx      *KFTLStatementLineContext
	req      *kftlMiRequest
}

func newKFTLMiEstimateStartTimeStatementLine(lineText string, ctx *KFTLStatementLineContext, req *kftlMiRequest) *kftlMiEstimateStartTimeStatementLine {
	targetID := ctx.ThisStatementLineTargetID
	ctx.NextStatementLineTargetID = &targetID
	ctx.NextStatementLineConstructor = func(lt string, c *KFTLStatementLineContext) KFTLStatementLine {
		return newKFTLMiEstimateEndTimeStatementLine(lt, c, req)
	}
	return &kftlMiEstimateStartTimeStatementLine{lineText: lineText, ctx: ctx, req: req}
}

func (l *kftlMiEstimateStartTimeStatementLine) ApplyThisLineToRequestMap(_ context.Context, _ *KFTLRequestMap) error {
	s := strings.TrimPrefix(l.lineText, "？")
	if s == "" {
		return nil
	}
	t, err := parseDateTime(s)
	if err != nil {
		return nil
	}
	l.req.estimateStartTime = &t
	return nil
}
func (l *kftlMiEstimateStartTimeStatementLine) GetLabelName() string { return "miEstimateStartTime" }
func (l *kftlMiEstimateStartTimeStatementLine) GetContext() *KFTLStatementLineContext {
	return l.ctx
}
func (l *kftlMiEstimateStartTimeStatementLine) GetStatementLineText() string { return l.lineText }

// kftlMiEstimateEndTimeStatementLine reads the Mi estimate end time (optional).
// After this, next constructor reverts to None.
// Mirrors: kftl-mi-estimate-end-time-statement-line.ts
type kftlMiEstimateEndTimeStatementLine struct {
	lineText string
	ctx      *KFTLStatementLineContext
	req      *kftlMiRequest
}

func newKFTLMiEstimateEndTimeStatementLine(lineText string, ctx *KFTLStatementLineContext, req *kftlMiRequest) *kftlMiEstimateEndTimeStatementLine {
	targetID := ctx.ThisStatementLineTargetID
	ctx.NextStatementLineTargetID = &targetID
	ctx.NextStatementLineConstructor = ctx.factory.generateNoneConstructor(ctx.NextStatementLineText)
	return &kftlMiEstimateEndTimeStatementLine{lineText: lineText, ctx: ctx, req: req}
}

func (l *kftlMiEstimateEndTimeStatementLine) ApplyThisLineToRequestMap(_ context.Context, _ *KFTLRequestMap) error {
	s := strings.TrimPrefix(l.lineText, "？")
	if s == "" {
		return nil
	}
	t, err := parseDateTime(s)
	if err != nil {
		return nil
	}
	l.req.estimateEndTime = &t
	return nil
}
func (l *kftlMiEstimateEndTimeStatementLine) GetLabelName() string { return "miEstimateEndTime" }
func (l *kftlMiEstimateEndTimeStatementLine) GetContext() *KFTLStatementLineContext {
	return l.ctx
}
func (l *kftlMiEstimateEndTimeStatementLine) GetStatementLineText() string { return l.lineText }
