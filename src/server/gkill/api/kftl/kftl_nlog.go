package kftl

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
	"github.com/mt3hr/gkill/src/server/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_options"
)

// ─── KFTLNlogRequest ──────────────────────────────────────────────────────────

// kftlNlogRequest accumulates shop + multiple (title, amount) pairs and writes
// one Nlog row per pair.
// Mirrors: kftl-nlog-request.ts
type kftlNlogRequest struct {
	KFTLRequestBase
	shop    string
	titles  []string
	amounts []json.Number
}

func newKFTLNlogRequest(requestID string, ctx *KFTLStatementLineContext) *kftlNlogRequest {
	return &kftlNlogRequest{
		KFTLRequestBase: KFTLRequestBase{
			RequestID:  requestID,
			Ctx:        ctx,
			CreateTime: nowFromCtx(ctx),
		},
	}
}

func (r *kftlNlogRequest) DoRequest(ctx context.Context) error {
	if err := r.doBaseRequest(ctx, r.RequestID); err != nil {
		return err
	}
	relatedTime := r.GetRelatedTime()
	now := r.CreateTime

	count := len(r.titles)
	if len(r.amounts) < count {
		count = len(r.amounts)
	}
	if len(r.titles) != len(r.amounts) {
		slog.Log(ctx, gkill_log.Warn, "nlog title/amount count mismatch",
			"titles", len(r.titles), "amounts", len(r.amounts), "using", count)
	}

	// Note: Transaction safety is provided by the KFTL submit flow's temp repositories.
	// Each nlog is inserted independently; if any fails, the entire KFTL submit rolls back.
	for i := range count {
		id := r.RequestID
		if i > 0 {
			id = sqlite3impl.GenerateNewID()
		}
		nlog := reps.Nlog{
			ID:           id,
			Shop:         r.shop,
			Title:        r.titles[i],
			Amount:       r.amounts[i],
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
		if err := r.Ctx.Repositories.WriteNlogRep.AddNlogInfo(ctx, nlog); err != nil {
			return fmt.Errorf("error at add nlog info id=%s: %w", id, err)
		}
		repName, _ := r.Ctx.Repositories.WriteNlogRep.GetRepName(ctx)
		updateLatestDataRepositoryAddress(ctx, r.Ctx.Repositories, id, nil, false, now, repName)
		// キャッシュに書き込み
		if len(r.Ctx.Repositories.NlogReps) == 1 && *gkill_options.CacheNlogReps {
			_ = r.Ctx.Repositories.NlogReps[0].AddNlogInfo(ctx, nlog)
		}
	}
	return nil
}

// ─── Statement lines ──────────────────────────────────────────────────────────

// kftlStartNlogStatementLine handles "ーん".
// Mirrors: kftl-start-nlog-statement-line.ts
type kftlStartNlogStatementLine struct {
	lineText string
	ctx      *KFTLStatementLineContext
	req      *kftlNlogRequest
}

func newKFTLStartNlogStatementLine(lineText string, ctx *KFTLStatementLineContext) *kftlStartNlogStatementLine {
	prevLine := ctx.GetPrevLine()
	var targetID string
	if prevLine != nil && prevLine.GetContext().ThisIsPrototype {
		targetID = prevLine.GetContext().ThisStatementLineTargetID
	} else {
		targetID = sqlite3impl.GenerateNewID()
	}
	ctx.ThisStatementLineTargetID = targetID
	ctx.NextStatementLineTargetID = &targetID

	req := newKFTLNlogRequest(targetID, ctx)
	ctx.NextStatementLineConstructor = func(lt string, c *KFTLStatementLineContext) KFTLStatementLine {
		return newKFTLNlogShopNameStatementLine(lt, c, req)
	}
	return &kftlStartNlogStatementLine{lineText: lineText, ctx: ctx, req: req}
}

func (l *kftlStartNlogStatementLine) ApplyThisLineToRequestMap(_ context.Context, requestMap *KFTLRequestMap) error {
	return requestMap.Set(l.ctx.ThisStatementLineTargetID, l.req)
}
func (l *kftlStartNlogStatementLine) GetLabelName() string                 { return "nlog" }
func (l *kftlStartNlogStatementLine) GetContext() *KFTLStatementLineContext { return l.ctx }
func (l *kftlStartNlogStatementLine) GetStatementLineText() string         { return l.lineText }

// kftlNlogShopNameStatementLine reads the shop name.
// Mirrors: kftl-nlog-shop-name-statement-line.ts
type kftlNlogShopNameStatementLine struct {
	lineText string
	ctx      *KFTLStatementLineContext
	req      *kftlNlogRequest
}

func newKFTLNlogShopNameStatementLine(lineText string, ctx *KFTLStatementLineContext, req *kftlNlogRequest) *kftlNlogShopNameStatementLine {
	targetID := ctx.ThisStatementLineTargetID
	ctx.NextStatementLineTargetID = &targetID
	ctx.NextStatementLineConstructor = func(lt string, c *KFTLStatementLineContext) KFTLStatementLine {
		return newKFTLNlogTitleStatementLine(lt, c, req)
	}
	return &kftlNlogShopNameStatementLine{lineText: lineText, ctx: ctx, req: req}
}

func (l *kftlNlogShopNameStatementLine) ApplyThisLineToRequestMap(_ context.Context, _ *KFTLRequestMap) error {
	l.req.shop = l.lineText
	return nil
}
func (l *kftlNlogShopNameStatementLine) GetLabelName() string                 { return "nlogShop" }
func (l *kftlNlogShopNameStatementLine) GetContext() *KFTLStatementLineContext { return l.ctx }
func (l *kftlNlogShopNameStatementLine) GetStatementLineText() string         { return l.lineText }

// kftlNlogTitleStatementLine reads a title for one Nlog item.
// Mirrors: kftl-nlog-title-statement-line.ts
type kftlNlogTitleStatementLine struct {
	lineText string
	ctx      *KFTLStatementLineContext
	req      *kftlNlogRequest
}

func newKFTLNlogTitleStatementLine(lineText string, ctx *KFTLStatementLineContext, req *kftlNlogRequest) *kftlNlogTitleStatementLine {
	targetID := ctx.ThisStatementLineTargetID
	ctx.NextStatementLineTargetID = &targetID
	ctx.NextStatementLineConstructor = func(lt string, c *KFTLStatementLineContext) KFTLStatementLine {
		return newKFTLNlogAmountStatementLine(lt, c, req)
	}
	return &kftlNlogTitleStatementLine{lineText: lineText, ctx: ctx, req: req}
}

func (l *kftlNlogTitleStatementLine) ApplyThisLineToRequestMap(_ context.Context, _ *KFTLRequestMap) error {
	l.req.titles = append(l.req.titles, l.lineText)
	return nil
}
func (l *kftlNlogTitleStatementLine) GetLabelName() string                 { return "nlogTitle" }
func (l *kftlNlogTitleStatementLine) GetContext() *KFTLStatementLineContext { return l.ctx }
func (l *kftlNlogTitleStatementLine) GetStatementLineText() string         { return l.lineText }

// kftlNlogAmountStatementLine reads the amount for one Nlog item.
// After reading, next constructor is generateNlogConstructor to allow more items.
// Mirrors: kftl-nlog-amount-statement-line.ts
type kftlNlogAmountStatementLine struct {
	lineText string
	ctx      *KFTLStatementLineContext
	req      *kftlNlogRequest
}

func newKFTLNlogAmountStatementLine(lineText string, ctx *KFTLStatementLineContext, req *kftlNlogRequest) *kftlNlogAmountStatementLine {
	targetID := ctx.ThisStatementLineTargetID
	ctx.NextStatementLineTargetID = &targetID
	// After amount: next line is either another title (via nlog constructor) or done
	ctx.NextStatementLineConstructor = ctx.factory.generateNlogConstructor(req)
	return &kftlNlogAmountStatementLine{lineText: lineText, ctx: ctx, req: req}
}

func (l *kftlNlogAmountStatementLine) ApplyThisLineToRequestMap(_ context.Context, _ *KFTLRequestMap) error {
	if l.lineText == "" {
		return fmt.Errorf("nlog amount is empty")
	}
	num := json.Number(l.lineText)
	if _, err := num.Float64(); err != nil {
		return fmt.Errorf("invalid nlog amount %q: %w", l.lineText, err)
	}
	l.req.amounts = append(l.req.amounts, num)
	return nil
}
func (l *kftlNlogAmountStatementLine) GetLabelName() string                 { return "nlogAmount" }
func (l *kftlNlogAmountStatementLine) GetContext() *KFTLStatementLineContext { return l.ctx }
func (l *kftlNlogAmountStatementLine) GetStatementLineText() string         { return l.lineText }
