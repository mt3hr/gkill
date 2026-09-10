package kftl

import (
	"context"
	"fmt"

	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
	"github.com/mt3hr/gkill/src/server/gkill/dao/sqlite3impl"
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
	// 「既知のプレフィックス＋同じ行に引数」はここへ落ちてくる。
	// プレフィックスの判定は完全一致なので `/mood 8` は本文扱いになり、
	// 気分記録のつもりが本文「/mood 8」のメモ1件になっていた（エラーも警告も無し）。
	// 本文として正当な行と区別できるのはこの1点だけなので、ここで捕まえる。
	if prefix, ok := prefixWrittenWithArgument(l.lineText); ok {
		return newKFTLInputError("KFTL_PREFIX_MUST_BE_ALONE_ON_LINE_MESSAGE_TITLE",
			fmt.Errorf("prefix %q must be alone on its line; put the value on the next line", prefix))
	}

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

func (l *kftlKmemoStatementLine) GetLabelName() string                  { return "kmemo" }
func (l *kftlKmemoStatementLine) GetContext() *KFTLStatementLineContext { return l.ctx }
func (l *kftlKmemoStatementLine) GetStatementLineText() string          { return l.lineText }

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

	if err := r.doBaseRequest(ctx, r.RequestID, r.GetRelatedTime()); err != nil {
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
	repName, repNameErr := r.Ctx.Repositories.WriteKmemoRep.GetRepName(ctx)
	logGetRepNameFailure(ctx, "kmemo", kmemo.ID, repNameErr)
	r.recordCreated("kmemo", kmemo.ID)
	updateLatestDataRepositoryAddress(ctx, r.Ctx.Repositories, r.RequestID, nil, false, now, repName)
	// キャッシュに書き込み
	logWriteThroughCacheFailure(ctx, "kmemo", kmemo.ID, r.Ctx.Repositories.WriteThroughKmemoCache(ctx, kmemo))
	return nil
}

// CloneForRepeat は繰り返しの1回ぶんを作る。日時は related_time だけなので基底に任せる。
func (r *kftlKmemoRequest) CloneForRepeat(newRequestID string, dayShift int) KFTLRequest {
	c := *r
	c.KFTLRequestBase = r.cloneBase(newRequestID, dayShift)
	c.contentLines = append([]string(nil), r.contentLines...)
	return &c
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
