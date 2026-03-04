package kftl

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
	"github.com/mt3hr/gkill/src/server/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_options"
)

// ─── KFTLTimeIsRequest (ーち): full TimeIs with start + optional end ──────────

// kftlTimeIsRequest records a TimeIs with title, start_time, and optional end_time.
// Mirrors: kftl-time-is-request.ts
type kftlTimeIsRequest struct {
	KFTLRequestBase
	title     string
	startTime time.Time
	endTime   *time.Time
}

func newKFTLTimeIsRequest(requestID string, ctx *KFTLStatementLineContext) *kftlTimeIsRequest {
	return &kftlTimeIsRequest{
		KFTLRequestBase: KFTLRequestBase{
			RequestID:  requestID,
			Ctx:        ctx,
			CreateTime: nowFromCtx(ctx),
		},
	}
}

func (r *kftlTimeIsRequest) DoRequest(ctx context.Context) error {
	if r.title == "" {
		return nil
	}
	if err := r.doBaseRequest(ctx, r.RequestID); err != nil {
		return err
	}
	relatedTime := r.GetRelatedTime()
	if !r.startTime.IsZero() {
		relatedTime = r.startTime
	}
	now := r.CreateTime
	timeis := reps.TimeIs{
		ID:           r.RequestID,
		Title:        r.title,
		StartTime:    relatedTime,
		EndTime:      r.endTime,
		CreateTime:   now,
		CreateApp:    r.Ctx.ApplicationName,
		CreateDevice: r.Ctx.Device,
		CreateUser:   r.Ctx.UserID,
		UpdateTime:   now,
		UpdateApp:    r.Ctx.ApplicationName,
		UpdateDevice: r.Ctx.Device,
		UpdateUser:   r.Ctx.UserID,
	}
	if err := r.Ctx.Repositories.WriteTimeIsRep.AddTimeIsInfo(ctx, timeis); err != nil {
		return err
	}
	repName, _ := r.Ctx.Repositories.WriteTimeIsRep.GetRepName(ctx)
	updateLatestDataRepositoryAddress(ctx, r.Ctx.Repositories, r.RequestID, nil, false, now, repName)
	// キャッシュに書き込み
	if len(r.Ctx.Repositories.TimeIsReps) == 1 && *gkill_options.CacheTimeIsReps {
		_ = r.Ctx.Repositories.TimeIsReps[0].AddTimeIsInfo(ctx, timeis)
	}
	return nil
}

// kftlStartTimeIsStatementLine handles "ーち".
// Line sequence: ーち → title → start_time → end_time
// Mirrors: kftl-start-time-is-statement-line.ts
type kftlStartTimeIsStatementLine struct {
	lineText string
	ctx      *KFTLStatementLineContext
	req      *kftlTimeIsRequest
}

func newKFTLStartTimeIsStatementLine(lineText string, ctx *KFTLStatementLineContext) *kftlStartTimeIsStatementLine {
	prevLine := ctx.GetPrevLine()
	var targetID string
	if prevLine != nil && prevLine.GetContext().ThisIsPrototype {
		targetID = prevLine.GetContext().ThisStatementLineTargetID
	} else {
		targetID = sqlite3impl.GenerateNewID()
	}
	ctx.ThisStatementLineTargetID = targetID
	ctx.NextStatementLineTargetID = &targetID

	req := newKFTLTimeIsRequest(targetID, ctx)
	ctx.NextStatementLineConstructor = func(lt string, c *KFTLStatementLineContext) KFTLStatementLine {
		return newKFTLTimeIsTitleStatementLine(lt, c, req)
	}
	return &kftlStartTimeIsStatementLine{lineText: lineText, ctx: ctx, req: req}
}

func (l *kftlStartTimeIsStatementLine) ApplyThisLineToRequestMap(_ context.Context, requestMap *KFTLRequestMap) error {
	return requestMap.Set(l.ctx.ThisStatementLineTargetID, l.req)
}
func (l *kftlStartTimeIsStatementLine) GetLabelName() string                  { return "timeIs" }
func (l *kftlStartTimeIsStatementLine) GetContext() *KFTLStatementLineContext { return l.ctx }
func (l *kftlStartTimeIsStatementLine) GetStatementLineText() string          { return l.lineText }

// kftlTimeIsTitleStatementLine reads the title for a ーち TimeIs.
type kftlTimeIsTitleStatementLine struct {
	lineText string
	ctx      *KFTLStatementLineContext
	req      *kftlTimeIsRequest
}

func newKFTLTimeIsTitleStatementLine(lineText string, ctx *KFTLStatementLineContext, req *kftlTimeIsRequest) *kftlTimeIsTitleStatementLine {
	targetID := ctx.ThisStatementLineTargetID
	ctx.NextStatementLineTargetID = &targetID
	ctx.NextStatementLineConstructor = func(lt string, c *KFTLStatementLineContext) KFTLStatementLine {
		return newKFTLTimeIsStartTimeStatementLine(lt, c, req)
	}
	return &kftlTimeIsTitleStatementLine{lineText: lineText, ctx: ctx, req: req}
}

func (l *kftlTimeIsTitleStatementLine) ApplyThisLineToRequestMap(_ context.Context, _ *KFTLRequestMap) error {
	l.req.title = l.lineText
	return nil
}
func (l *kftlTimeIsTitleStatementLine) GetLabelName() string                  { return "timeIsTitle" }
func (l *kftlTimeIsTitleStatementLine) GetContext() *KFTLStatementLineContext { return l.ctx }
func (l *kftlTimeIsTitleStatementLine) GetStatementLineText() string          { return l.lineText }

// kftlTimeIsStartTimeStatementLine reads start_time for a ーち TimeIs.
type kftlTimeIsStartTimeStatementLine struct {
	lineText string
	ctx      *KFTLStatementLineContext
	req      *kftlTimeIsRequest
}

func newKFTLTimeIsStartTimeStatementLine(lineText string, ctx *KFTLStatementLineContext, req *kftlTimeIsRequest) *kftlTimeIsStartTimeStatementLine {
	targetID := ctx.ThisStatementLineTargetID
	ctx.NextStatementLineTargetID = &targetID
	ctx.NextStatementLineConstructor = func(lt string, c *KFTLStatementLineContext) KFTLStatementLine {
		return newKFTLTimeIsEndTimeStatementLine(lt, c, req)
	}
	return &kftlTimeIsStartTimeStatementLine{lineText: lineText, ctx: ctx, req: req}
}

func (l *kftlTimeIsStartTimeStatementLine) ApplyThisLineToRequestMap(_ context.Context, _ *KFTLRequestMap) error {
	t, err := parseDateTime(l.lineText)
	if err != nil {
		return fmt.Errorf("invalid timeis start_time %q: %w", l.lineText, err)
	}
	l.req.startTime = t
	l.req.SetRelatedTime(t)
	return nil
}
func (l *kftlTimeIsStartTimeStatementLine) GetLabelName() string                  { return "timeIsStartTime" }
func (l *kftlTimeIsStartTimeStatementLine) GetContext() *KFTLStatementLineContext { return l.ctx }
func (l *kftlTimeIsStartTimeStatementLine) GetStatementLineText() string          { return l.lineText }

// kftlTimeIsEndTimeStatementLine reads end_time for a ーち TimeIs.
type kftlTimeIsEndTimeStatementLine struct {
	lineText string
	ctx      *KFTLStatementLineContext
	req      *kftlTimeIsRequest
}

func newKFTLTimeIsEndTimeStatementLine(lineText string, ctx *KFTLStatementLineContext, req *kftlTimeIsRequest) *kftlTimeIsEndTimeStatementLine {
	targetID := ctx.ThisStatementLineTargetID
	ctx.NextStatementLineTargetID = &targetID
	ctx.NextStatementLineConstructor = ctx.factory.generateNoneConstructor(ctx.NextStatementLineText)
	return &kftlTimeIsEndTimeStatementLine{lineText: lineText, ctx: ctx, req: req}
}

func (l *kftlTimeIsEndTimeStatementLine) ApplyThisLineToRequestMap(_ context.Context, _ *KFTLRequestMap) error {
	t, err := parseDateTime(l.lineText)
	if err != nil {
		return fmt.Errorf("invalid timeis end_time %q: %w", l.lineText, err)
	}
	l.req.endTime = &t
	return nil
}
func (l *kftlTimeIsEndTimeStatementLine) GetLabelName() string                  { return "timeIsEndTime" }
func (l *kftlTimeIsEndTimeStatementLine) GetContext() *KFTLStatementLineContext { return l.ctx }
func (l *kftlTimeIsEndTimeStatementLine) GetStatementLineText() string          { return l.lineText }

// ─── TimeIs Start-only (ーた) ─────────────────────────────────────────────────

// kftlTimeIsStartRequest records only start_time (no end_time).
// Mirrors: kftl-time-is-start-request.ts
type kftlTimeIsStartRequest struct {
	KFTLRequestBase
	title string
}

func newKFTLTimeIsStartRequest(requestID string, ctx *KFTLStatementLineContext) *kftlTimeIsStartRequest {
	return &kftlTimeIsStartRequest{
		KFTLRequestBase: KFTLRequestBase{
			RequestID:  requestID,
			Ctx:        ctx,
			CreateTime: nowFromCtx(ctx),
		},
	}
}

func (r *kftlTimeIsStartRequest) DoRequest(ctx context.Context) error {
	if r.title == "" {
		return nil
	}
	if err := r.doBaseRequest(ctx, r.RequestID); err != nil {
		return err
	}
	relatedTime := r.GetRelatedTime()
	now := r.CreateTime
	timeis := reps.TimeIs{
		ID:           r.RequestID,
		Title:        r.title,
		StartTime:    relatedTime,
		EndTime:      nil,
		CreateTime:   now,
		CreateApp:    r.Ctx.ApplicationName,
		CreateDevice: r.Ctx.Device,
		CreateUser:   r.Ctx.UserID,
		UpdateTime:   now,
		UpdateApp:    r.Ctx.ApplicationName,
		UpdateDevice: r.Ctx.Device,
		UpdateUser:   r.Ctx.UserID,
	}
	if err := r.Ctx.Repositories.WriteTimeIsRep.AddTimeIsInfo(ctx, timeis); err != nil {
		return err
	}
	repName, _ := r.Ctx.Repositories.WriteTimeIsRep.GetRepName(ctx)
	updateLatestDataRepositoryAddress(ctx, r.Ctx.Repositories, r.RequestID, nil, false, now, repName)
	// キャッシュに書き込み
	if len(r.Ctx.Repositories.TimeIsReps) == 1 && *gkill_options.CacheTimeIsReps {
		_ = r.Ctx.Repositories.TimeIsReps[0].AddTimeIsInfo(ctx, timeis)
	}
	return nil
}

// kftlStartTimeIsStartStatementLine handles "ーた".
// Mirrors: kftl-start-time-is-start-statement-line.ts
type kftlStartTimeIsStartStatementLine struct {
	lineText string
	ctx      *KFTLStatementLineContext
	req      *kftlTimeIsStartRequest
}

func newKFTLStartTimeIsStartStatementLine(lineText string, ctx *KFTLStatementLineContext) *kftlStartTimeIsStartStatementLine {
	prevLine := ctx.GetPrevLine()
	var targetID string
	if prevLine != nil && prevLine.GetContext().ThisIsPrototype {
		targetID = prevLine.GetContext().ThisStatementLineTargetID
	} else {
		targetID = sqlite3impl.GenerateNewID()
	}
	ctx.ThisStatementLineTargetID = targetID
	ctx.NextStatementLineTargetID = &targetID

	req := newKFTLTimeIsStartRequest(targetID, ctx)
	ctx.NextStatementLineConstructor = func(lt string, c *KFTLStatementLineContext) KFTLStatementLine {
		return newKFTLTimeIsStartTitleStatementLine(lt, c, req)
	}
	return &kftlStartTimeIsStartStatementLine{lineText: lineText, ctx: ctx, req: req}
}

func (l *kftlStartTimeIsStartStatementLine) ApplyThisLineToRequestMap(_ context.Context, requestMap *KFTLRequestMap) error {
	return requestMap.Set(l.ctx.ThisStatementLineTargetID, l.req)
}
func (l *kftlStartTimeIsStartStatementLine) GetLabelName() string                  { return "timeIsStart" }
func (l *kftlStartTimeIsStartStatementLine) GetContext() *KFTLStatementLineContext { return l.ctx }
func (l *kftlStartTimeIsStartStatementLine) GetStatementLineText() string          { return l.lineText }

// kftlTimeIsStartTitleStatementLine reads the title for a ーた TimeIs.
type kftlTimeIsStartTitleStatementLine struct {
	lineText string
	ctx      *KFTLStatementLineContext
	req      *kftlTimeIsStartRequest
}

func newKFTLTimeIsStartTitleStatementLine(lineText string, ctx *KFTLStatementLineContext, req *kftlTimeIsStartRequest) *kftlTimeIsStartTitleStatementLine {
	targetID := ctx.ThisStatementLineTargetID
	ctx.NextStatementLineTargetID = &targetID
	ctx.NextStatementLineConstructor = ctx.factory.generateNoneConstructor(ctx.NextStatementLineText)
	return &kftlTimeIsStartTitleStatementLine{lineText: lineText, ctx: ctx, req: req}
}

func (l *kftlTimeIsStartTitleStatementLine) ApplyThisLineToRequestMap(_ context.Context, _ *KFTLRequestMap) error {
	l.req.title = l.lineText
	return nil
}
func (l *kftlTimeIsStartTitleStatementLine) GetLabelName() string                  { return "timeIsStartTitle" }
func (l *kftlTimeIsStartTitleStatementLine) GetContext() *KFTLStatementLineContext { return l.ctx }
func (l *kftlTimeIsStartTitleStatementLine) GetStatementLineText() string          { return l.lineText }

// ─── TimeIs End by title (ーえ / ーいえ) ──────────────────────────────────────

// kftlTimeIsEndByTitleRequest finds a playing TimeIs by title and sets its end_time.
// Mirrors: kftl-time-is-end-by-title-request.ts
type kftlTimeIsEndByTitleRequest struct {
	KFTLRequestBase
	title                   string
	errorWhenTargetNotExist bool
}

func newKFTLTimeIsEndByTitleRequest(requestID string, ctx *KFTLStatementLineContext, errorWhenNotExist bool) *kftlTimeIsEndByTitleRequest {
	return &kftlTimeIsEndByTitleRequest{
		KFTLRequestBase: KFTLRequestBase{
			RequestID:  requestID,
			Ctx:        ctx,
			CreateTime: nowFromCtx(ctx),
		},
		errorWhenTargetNotExist: errorWhenNotExist,
	}
}

func (r *kftlTimeIsEndByTitleRequest) DoRequest(ctx context.Context) error {
	if r.title == "" {
		return fmt.Errorf("timeis end: title is empty")
	}
	if err := r.doBaseRequest(ctx, r.RequestID); err != nil {
		return err
	}
	endTime := r.GetRelatedTime()

	query := &find.FindQuery{
		UsePlaing:      true,
		PlaingTime:     time.Now(),
		OnlyLatestData: true,
	}
	playingEntries, err := r.Ctx.Repositories.TimeIsReps.FindTimeIs(ctx, query)
	if err != nil {
		return fmt.Errorf("error finding playing timeis: %w", err)
	}

	var target *reps.TimeIs
	for i := range playingEntries {
		if playingEntries[i].Title == r.title {
			t := playingEntries[i]
			target = &t
			break
		}
	}

	if target == nil {
		if r.errorWhenTargetNotExist {
			return fmt.Errorf("no playing timeis with title=%q", r.title)
		}
		return nil
	}

	now := r.CreateTime
	updated := *target
	updated.EndTime = &endTime
	updated.UpdateTime = now
	updated.UpdateApp = r.Ctx.ApplicationName
	updated.UpdateDevice = r.Ctx.Device
	updated.UpdateUser = r.Ctx.UserID
	if err := r.Ctx.Repositories.WriteTimeIsRep.AddTimeIsInfo(ctx, updated); err != nil {
		return err
	}
	repName, _ := r.Ctx.Repositories.WriteTimeIsRep.GetRepName(ctx)
	updateLatestDataRepositoryAddress(ctx, r.Ctx.Repositories, target.ID, nil, false, now, repName)
	// キャッシュに書き込み
	if len(r.Ctx.Repositories.TimeIsReps) == 1 && *gkill_options.CacheTimeIsReps {
		_ = r.Ctx.Repositories.TimeIsReps[0].AddTimeIsInfo(ctx, updated)
	}
	return nil
}

// kftlStartTimeIsEndStatementLine handles "ーえ" (error if not found).
// Mirrors: kftl-start-time-is-end-statement-line.ts
type kftlStartTimeIsEndStatementLine struct {
	lineText string
	ctx      *KFTLStatementLineContext
	req      *kftlTimeIsEndByTitleRequest
}

func newKFTLStartTimeIsEndStatementLine(lineText string, ctx *KFTLStatementLineContext) *kftlStartTimeIsEndStatementLine {
	targetID := sqlite3impl.GenerateNewID()
	ctx.ThisStatementLineTargetID = targetID
	ctx.NextStatementLineTargetID = &targetID
	ctx.ThisIsPrototype = true

	req := newKFTLTimeIsEndByTitleRequest(targetID, ctx, true)
	ctx.NextStatementLineConstructor = func(lt string, c *KFTLStatementLineContext) KFTLStatementLine {
		return newKFTLTimeIsEndTitleStatementLine(lt, c, req)
	}
	return &kftlStartTimeIsEndStatementLine{lineText: lineText, ctx: ctx, req: req}
}

func (l *kftlStartTimeIsEndStatementLine) ApplyThisLineToRequestMap(_ context.Context, requestMap *KFTLRequestMap) error {
	return requestMap.Set(l.ctx.ThisStatementLineTargetID, l.req)
}
func (l *kftlStartTimeIsEndStatementLine) GetLabelName() string                  { return "timeIsEnd" }
func (l *kftlStartTimeIsEndStatementLine) GetContext() *KFTLStatementLineContext { return l.ctx }
func (l *kftlStartTimeIsEndStatementLine) GetStatementLineText() string          { return l.lineText }

// kftlTimeIsEndTitleStatementLine reads the title to find the playing TimeIs.
type kftlTimeIsEndTitleStatementLine struct {
	lineText string
	ctx      *KFTLStatementLineContext
	req      *kftlTimeIsEndByTitleRequest
}

func newKFTLTimeIsEndTitleStatementLine(lineText string, ctx *KFTLStatementLineContext, req *kftlTimeIsEndByTitleRequest) *kftlTimeIsEndTitleStatementLine {
	targetID := ctx.ThisStatementLineTargetID
	ctx.NextStatementLineTargetID = &targetID
	ctx.NextStatementLineConstructor = ctx.factory.generateNoneConstructor(ctx.NextStatementLineText)
	return &kftlTimeIsEndTitleStatementLine{lineText: lineText, ctx: ctx, req: req}
}

func (l *kftlTimeIsEndTitleStatementLine) ApplyThisLineToRequestMap(_ context.Context, _ *KFTLRequestMap) error {
	l.req.title = l.lineText
	return nil
}
func (l *kftlTimeIsEndTitleStatementLine) GetLabelName() string                  { return "timeIsEndTitle" }
func (l *kftlTimeIsEndTitleStatementLine) GetContext() *KFTLStatementLineContext { return l.ctx }
func (l *kftlTimeIsEndTitleStatementLine) GetStatementLineText() string          { return l.lineText }

// kftlStartTimeIsEndIfExistStatementLine handles "ーいえ" (no error if not found).
// Mirrors: kftl-start-time-is-end-if-exist-statement-line.ts
type kftlStartTimeIsEndIfExistStatementLine struct {
	lineText string
	ctx      *KFTLStatementLineContext
	req      *kftlTimeIsEndByTitleRequest
}

func newKFTLStartTimeIsEndIfExistStatementLine(lineText string, ctx *KFTLStatementLineContext) *kftlStartTimeIsEndIfExistStatementLine {
	targetID := sqlite3impl.GenerateNewID()
	ctx.ThisStatementLineTargetID = targetID
	ctx.NextStatementLineTargetID = &targetID
	ctx.ThisIsPrototype = true

	req := newKFTLTimeIsEndByTitleRequest(targetID, ctx, false)
	ctx.NextStatementLineConstructor = func(lt string, c *KFTLStatementLineContext) KFTLStatementLine {
		return newKFTLTimeIsEndTitleStatementLine(lt, c, req)
	}
	return &kftlStartTimeIsEndIfExistStatementLine{lineText: lineText, ctx: ctx, req: req}
}

func (l *kftlStartTimeIsEndIfExistStatementLine) ApplyThisLineToRequestMap(_ context.Context, requestMap *KFTLRequestMap) error {
	return requestMap.Set(l.ctx.ThisStatementLineTargetID, l.req)
}
func (l *kftlStartTimeIsEndIfExistStatementLine) GetLabelName() string {
	return "timeIsEndIfExist"
}
func (l *kftlStartTimeIsEndIfExistStatementLine) GetContext() *KFTLStatementLineContext {
	return l.ctx
}
func (l *kftlStartTimeIsEndIfExistStatementLine) GetStatementLineText() string { return l.lineText }

// ─── TimeIs End by tag (ーたえ / ーいたえ) ─────────────────────────────────────

// kftlTimeIsEndByTagRequest finds a playing TimeIs by tags and sets its end_time.
// Mirrors: kftl-time-is-end-by-tag-request.ts
type kftlTimeIsEndByTagRequest struct {
	KFTLRequestBase
	searchTags              []string
	errorWhenTargetNotExist bool
}

func newKFTLTimeIsEndByTagRequest(requestID string, ctx *KFTLStatementLineContext, errorWhenNotExist bool) *kftlTimeIsEndByTagRequest {
	return &kftlTimeIsEndByTagRequest{
		KFTLRequestBase: KFTLRequestBase{
			RequestID:  requestID,
			Ctx:        ctx,
			CreateTime: nowFromCtx(ctx),
		},
		errorWhenTargetNotExist: errorWhenNotExist,
	}
}

// Override AddTag so searchTags also gets the value.
func (r *kftlTimeIsEndByTagRequest) AddTag(tag string) {
	r.KFTLRequestBase.AddTag(tag)
	r.searchTags = append(r.searchTags, tag)
}

func (r *kftlTimeIsEndByTagRequest) DoRequest(ctx context.Context) error {
	if err := r.doBaseRequest(ctx, r.RequestID); err != nil {
		return err
	}
	endTime := r.GetRelatedTime()

	query := &find.FindQuery{
		UsePlaing:      true,
		PlaingTime:     time.Now(),
		OnlyLatestData: true,
	}
	playingEntries, err := r.Ctx.Repositories.TimeIsReps.FindTimeIs(ctx, query)
	if err != nil {
		return fmt.Errorf("error finding playing timeis for tag-end: %w", err)
	}

	var target *reps.TimeIs
outer:
	for i := range playingEntries {
		entryTags, err := r.Ctx.Repositories.GetTagsByTargetID(ctx, playingEntries[i].ID)
		if err != nil {
			continue
		}
		for _, et := range entryTags {
			for _, wantTag := range r.searchTags {
				if et.Tag == wantTag {
					t := playingEntries[i]
					target = &t
					break outer
				}
			}
		}
	}

	if target == nil {
		if r.errorWhenTargetNotExist {
			return fmt.Errorf("no playing timeis with tags=%v", r.searchTags)
		}
		return nil
	}

	now := r.CreateTime
	updated := *target
	updated.EndTime = &endTime
	updated.UpdateTime = now
	updated.UpdateApp = r.Ctx.ApplicationName
	updated.UpdateDevice = r.Ctx.Device
	updated.UpdateUser = r.Ctx.UserID
	if err := r.Ctx.Repositories.WriteTimeIsRep.AddTimeIsInfo(ctx, updated); err != nil {
		return err
	}
	repName, _ := r.Ctx.Repositories.WriteTimeIsRep.GetRepName(ctx)
	updateLatestDataRepositoryAddress(ctx, r.Ctx.Repositories, target.ID, nil, false, now, repName)
	// キャッシュに書き込み
	if len(r.Ctx.Repositories.TimeIsReps) == 1 && *gkill_options.CacheTimeIsReps {
		_ = r.Ctx.Repositories.TimeIsReps[0].AddTimeIsInfo(ctx, updated)
	}
	return nil
}

// kftlStartTimeIsEndByTagStatementLine handles "ーたえ" (error if not found).
type kftlStartTimeIsEndByTagStatementLine struct {
	lineText string
	ctx      *KFTLStatementLineContext
	req      *kftlTimeIsEndByTagRequest
}

func newKFTLStartTimeIsEndByTagStatementLine(lineText string, ctx *KFTLStatementLineContext) *kftlStartTimeIsEndByTagStatementLine {
	targetID := sqlite3impl.GenerateNewID()
	ctx.ThisStatementLineTargetID = targetID
	ctx.NextStatementLineTargetID = &targetID
	ctx.ThisIsPrototype = true

	req := newKFTLTimeIsEndByTagRequest(targetID, ctx, true)
	ctx.NextStatementLineConstructor = func(lt string, c *KFTLStatementLineContext) KFTLStatementLine {
		return newKFTLTimeIsEndByTagTagStatementLine(lt, c, req)
	}
	return &kftlStartTimeIsEndByTagStatementLine{lineText: lineText, ctx: ctx, req: req}
}

func (l *kftlStartTimeIsEndByTagStatementLine) ApplyThisLineToRequestMap(_ context.Context, requestMap *KFTLRequestMap) error {
	return requestMap.Set(l.ctx.ThisStatementLineTargetID, l.req)
}
func (l *kftlStartTimeIsEndByTagStatementLine) GetLabelName() string { return "timeIsEndByTag" }
func (l *kftlStartTimeIsEndByTagStatementLine) GetContext() *KFTLStatementLineContext {
	return l.ctx
}
func (l *kftlStartTimeIsEndByTagStatementLine) GetStatementLineText() string { return l.lineText }

// kftlStartTimeIsEndByTagIfExistStatementLine handles "ーいたえ" (no error if not found).
type kftlStartTimeIsEndByTagIfExistStatementLine struct {
	lineText string
	ctx      *KFTLStatementLineContext
	req      *kftlTimeIsEndByTagRequest
}

func newKFTLStartTimeIsEndByTagIfExistStatementLine(lineText string, ctx *KFTLStatementLineContext) *kftlStartTimeIsEndByTagIfExistStatementLine {
	targetID := sqlite3impl.GenerateNewID()
	ctx.ThisStatementLineTargetID = targetID
	ctx.NextStatementLineTargetID = &targetID
	ctx.ThisIsPrototype = true

	req := newKFTLTimeIsEndByTagRequest(targetID, ctx, false)
	ctx.NextStatementLineConstructor = func(lt string, c *KFTLStatementLineContext) KFTLStatementLine {
		return newKFTLTimeIsEndByTagTagStatementLine(lt, c, req)
	}
	return &kftlStartTimeIsEndByTagIfExistStatementLine{lineText: lineText, ctx: ctx, req: req}
}

func (l *kftlStartTimeIsEndByTagIfExistStatementLine) ApplyThisLineToRequestMap(_ context.Context, requestMap *KFTLRequestMap) error {
	return requestMap.Set(l.ctx.ThisStatementLineTargetID, l.req)
}
func (l *kftlStartTimeIsEndByTagIfExistStatementLine) GetLabelName() string {
	return "timeIsEndByTagIfExist"
}
func (l *kftlStartTimeIsEndByTagIfExistStatementLine) GetContext() *KFTLStatementLineContext {
	return l.ctx
}
func (l *kftlStartTimeIsEndByTagIfExistStatementLine) GetStatementLineText() string {
	return l.lineText
}

// kftlTimeIsEndByTagTagStatementLine reads the tag name (plain text, no 。 prefix)
// for ーたえ / ーいたえ. Multiple tags can be separated by 、.
type kftlTimeIsEndByTagTagStatementLine struct {
	lineText string
	ctx      *KFTLStatementLineContext
	req      *kftlTimeIsEndByTagRequest
}

func newKFTLTimeIsEndByTagTagStatementLine(lineText string, ctx *KFTLStatementLineContext, req *kftlTimeIsEndByTagRequest) *kftlTimeIsEndByTagTagStatementLine {
	targetID := ctx.ThisStatementLineTargetID
	ctx.NextStatementLineTargetID = &targetID
	ctx.NextStatementLineConstructor = ctx.factory.generateNoneConstructor(ctx.NextStatementLineText)
	return &kftlTimeIsEndByTagTagStatementLine{lineText: lineText, ctx: ctx, req: req}
}

func (l *kftlTimeIsEndByTagTagStatementLine) ApplyThisLineToRequestMap(_ context.Context, _ *KFTLRequestMap) error {
	for _, tag := range strings.Split(l.lineText, "、") {
		tag = strings.TrimSpace(tag)
		if tag != "" {
			l.req.AddTag(tag)
		}
	}
	return nil
}
func (l *kftlTimeIsEndByTagTagStatementLine) GetLabelName() string { return "timeIsEndByTagTag" }
func (l *kftlTimeIsEndByTagTagStatementLine) GetContext() *KFTLStatementLineContext {
	return l.ctx
}
func (l *kftlTimeIsEndByTagTagStatementLine) GetStatementLineText() string { return l.lineText }
