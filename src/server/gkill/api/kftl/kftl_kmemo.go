package kftl

import (
	"context"
	"fmt"

	"github.com/mt3hr/gkill/src/app/gkill/dao/reps"
	"github.com/mt3hr/gkill/src/app/gkill/dao/sqlite3impl"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_options"
)

// ─── KFTLKmemoStatementLine ───────────────────────────────────────────────────

// kftlKmemoStatementLine handles plain text (default) lines and accumulates
// multi-line Kmemo content.
// Mirrors: src/classes/kftl/kftl_kmemo/kftl-kmemo-statement-line.ts
type kftlKmemoStatementLine struct {
	lineText string
	ctx      *KFTLStatementLineContext
}

func newKFTLKmemoStatementLine(lineText string, ctx *KFTLStatementLineContext) *kftlKmemoStatementLine {
	// Determine target_id:
	//   - if prev was prototype → reuse its target_id
	//   - if prev was also a Kmemo line → reuse its target_id
	//   - otherwise → generate new UUID
	prevLine := ctx.GetPrevLine()
	var targetID string
	if prevLine != nil && prevLine.GetContext().ThisIsPrototype {
		targetID = prevLine.GetContext().ThisStatementLineTargetID
	} else if prevLineIsKmemo(ctx) {
		targetID = prevLine.GetContext().ThisStatementLineTargetID
	} else {
		targetID = sqlite3impl.GenerateNewID()
	}
	ctx.ThisStatementLineTargetID = targetID
	// next line keeps same target_id (continuation)
	ctx.NextStatementLineTargetID = &targetID

	return &kftlKmemoStatementLine{lineText: lineText, ctx: ctx}
}

// prevLineIsKmemo checks if the previous statement line was a Kmemo line.
func prevLineIsKmemo(ctx *KFTLStatementLineContext) bool {
	lines := ctx.KFTLStatementLines
	if len(lines) >= 1 {
		_, ok := lines[len(lines)-1].(*kftlKmemoStatementLine)
		return ok
	}
	return false
}

func (l *kftlKmemoStatementLine) ApplyThisLineToRequestMap(_ context.Context, requestMap *KFTLRequestMap) error {
	targetID := l.ctx.ThisStatementLineTargetID

	// Mirrors TS try/catch: if the existing entry is not a KmemoRequest (e.g. PrototypeRequest),
	// create a new KmemoRequest; requestMap.Set inherits tags/texts from the prototype.
	var kmemoReq *kftlKmemoRequest
	if existing, ok := requestMap.Get(targetID); ok {
		kmemoReq, _ = existing.(*kftlKmemoRequest)
	}
	if kmemoReq == nil {
		newReq := newKFTLKmemoRequest(targetID, l.ctx)
		if err := requestMap.Set(targetID, newReq); err != nil {
			return err
		}
		found, _ := requestMap.Get(targetID)
		kmemoReq = found.(*kftlKmemoRequest)
	}
	kmemoReq.addKmemoLine(l.lineText)
	return nil
}

func (l *kftlKmemoStatementLine) GetLabelName() string                 { return "kmemo" }
func (l *kftlKmemoStatementLine) GetContext() *KFTLStatementLineContext { return l.ctx }
func (l *kftlKmemoStatementLine) GetStatementLineText() string         { return l.lineText }

// ─── KFTLKmemoRequest ────────────────────────────────────────────────────────

// kftlKmemoRequest accumulates Kmemo content and writes it to the repository.
// Mirrors: src/classes/kftl/kftl_kmemo/kftl-kmemo-request.ts
type kftlKmemoRequest struct {
	KFTLRequestBase
	contentLines []string
}

func newKFTLKmemoRequest(requestID string, ctx *KFTLStatementLineContext) *kftlKmemoRequest {
	return &kftlKmemoRequest{
		KFTLRequestBase: KFTLRequestBase{
			RequestID:  requestID,
			Ctx:        ctx,
			CreateTime: nowFromCtx(ctx),
		},
	}
}

func (r *kftlKmemoRequest) addKmemoLine(line string) {
	r.contentLines = append(r.contentLines, line)
}

func (r *kftlKmemoRequest) DoRequest(ctx context.Context) error {
	content := joinLines(r.contentLines)
	if content == "" {
		return nil // skip blank kmemo
	}

	if err := r.doBaseRequest(ctx, r.RequestID); err != nil {
		return err
	}

	relatedTime := r.GetRelatedTime()
	now := r.CreateTime
	kmemo := reps.Kmemo{
		ID:           r.RequestID,
		Content:      content,
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
	err := r.Ctx.Repositories.WriteKmemoRep.AddKmemoInfo(ctx, kmemo)
	if err != nil {
		return fmt.Errorf("error at add kmemo info id=%s: %w", r.RequestID, err)
	}
	repName, _ := r.Ctx.Repositories.WriteKmemoRep.GetRepName(ctx)
	updateLatestDataRepositoryAddress(ctx, r.Ctx.Repositories, r.RequestID, nil, false, now, repName)
	// キャッシュに書き込み
	if len(r.Ctx.Repositories.KmemoReps) == 1 && *gkill_options.CacheKmemoReps {
		_ = r.Ctx.Repositories.KmemoReps[0].AddKmemoInfo(ctx, kmemo)
	}
	return nil
}

// joinLines joins content lines with newline, mirroring TS add_kmemo_line behaviour.
func joinLines(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	result := lines[0]
	for i := 1; i < len(lines); i++ {
		result += "\n" + lines[i]
	}
	return result
}
