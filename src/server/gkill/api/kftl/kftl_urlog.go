package kftl

import (
	"context"
	"fmt"

	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
	"github.com/mt3hr/gkill/src/server/gkill/dao/sqlite3impl"
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

// ValidateContent は URL もタイトルも無いブックマークを入力エラーにする（旧 Web の ERR900020 と同じ文言）。
// 通常は start 行の requireNextLineText が先に止める。ここは保存マーカーの穴（ADR-0508）のような
// 経路で空のまま届いたときの防御線。
func (r *kftlURLogRequest) ValidateContent() error {
	if r.url == "" && r.title == "" {
		return newKFTLInputError("KFTL_URLOG_BLANK_SKIP_SAVE_MESSAGE_TITLE",
			fmt.Errorf("urlog url and title are empty: id=%s", r.RequestID))
	}
	return nil
}

func (r *kftlURLogRequest) DoRequest(ctx context.Context) error {
	if err := r.ValidateContent(); err != nil {
		return err
	}

	if err := r.doBaseRequest(ctx, r.RequestID, r.GetRelatedTime()); err != nil {
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
	if err := r.Ctx.Repositories.TempReps.URLogTempRep.AddURLogInfo(ctx, urlog, r.Ctx.TXID, r.Ctx.UserID, r.Ctx.Device); err != nil {
		return fmt.Errorf("error at add urlog info id=%s: %w", r.RequestID, err)
	}
	r.recordCreated("urlog", urlog.ID, r.GetRelatedTime())
	return nil
}

// CloneForRepeat は繰り返しの1回ぶんを作る。日時は related_time だけなので基底に任せる。
func (r *kftlURLogRequest) CloneForRepeat(newRequestID string, dayShift int) KFTLRequest {
	c := *r
	c.KFTLRequestBase = r.cloneBase(newRequestID, dayShift)
	return &c
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
	// URLの行が無いと無言で0件になり、なぜ作られなかったのか分からない。
	if err := requireNextLineText(l.ctx); err != nil {
		return err
	}
	return requestMap.Set(l.ctx.ThisStatementLineTargetID, l.req)
}
func (l *kftlStartURLogStatementLine) GetLabelName() string                  { return "urlog" }
func (l *kftlStartURLogStatementLine) GetContext() *KFTLStatementLineContext { return l.ctx }
func (l *kftlStartURLogStatementLine) GetStatementLineText() string          { return l.lineText }

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
	// Use generateDefaultConstructor so that separators (「、」「、、」) are
	// recognised and break the URLog; otherwise fall through to title.
	ctx.NextStatementLineConstructor = ctx.factory.generateDefaultConstructor(ctx.NextStatementLineText, func(lt string, c *KFTLStatementLineContext) KFTLStatementLine {
		return newKFTLURLogTitleStatementLine(lt, c, req)
	})
	return &kftlURLogURLStatementLine{lineText: lineText, ctx: ctx, req: req}
}

func (l *kftlURLogURLStatementLine) ApplyThisLineToRequestMap(_ context.Context, _ *KFTLRequestMap) error {
	l.req.url = l.lineText
	return nil
}
func (l *kftlURLogURLStatementLine) GetLabelName() string                  { return "urlogURL" }
func (l *kftlURLogURLStatementLine) GetContext() *KFTLStatementLineContext { return l.ctx }
func (l *kftlURLogURLStatementLine) GetStatementLineText() string          { return l.lineText }

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
func (l *kftlURLogTitleStatementLine) GetLabelName() string                  { return "urlogTitle" }
func (l *kftlURLogTitleStatementLine) GetContext() *KFTLStatementLineContext { return l.ctx }
func (l *kftlURLogTitleStatementLine) GetStatementLineText() string          { return l.lineText }
