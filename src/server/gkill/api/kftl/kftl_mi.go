package kftl

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
	"github.com/mt3hr/gkill/src/server/gkill/dao/sqlite3impl"
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

// MiBoardName は利用者が書いたとおりの板名（空欄なら空）。既定板へは解決しない。
// Analyze が「まだ無い板名」の確認に使う（miBoardNameProvider）。
func (r *kftlMiRequest) MiBoardName() string { return r.boardName }

// resolvedBoardName は書き込みに使う板名。空なら設定の既定板になる。
// **既存判定（FindExistingForRepeat）と書き込みで同じ値を使うため**にここへ出してある。
func (r *kftlMiRequest) resolvedBoardName() string {
	if r.boardName != "" {
		return r.boardName
	}
	if r.Ctx != nil && r.Ctx.ApplicationConfig != nil {
		return r.Ctx.ApplicationConfig.MiDefaultBoard
	}
	return ""
}

// ValidateContent はタイトルの無いタスクを入力エラーにする（旧 Web の ERR900013 と同じ文言）。
// 通常は start 行の requireNextLineText が先に止める。ここは保存マーカーの穴（ADR-0508）のような
// 経路でタイトル空のまま届いたときの防御線。
func (r *kftlMiRequest) ValidateContent() error {
	if r.title == "" {
		return newKFTLInputError("KFTL_MI_TITLE_BLANK_SKIP_SAVE_MESSAGE_TITLE",
			fmt.Errorf("mi title is empty: id=%s", r.RequestID))
	}
	return nil
}

func (r *kftlMiRequest) DoRequest(ctx context.Context) error {
	if err := r.ValidateContent(); err != nil {
		return err
	}

	boardName := r.resolvedBoardName()

	if err := r.doBaseRequest(ctx, r.RequestID, r.GetRelatedTime()); err != nil {
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
	if err := r.Ctx.Repositories.TempReps.MiTempRep.AddMiInfo(ctx, mi, r.Ctx.TXID, r.Ctx.UserID, r.Ctx.Device); err != nil {
		return fmt.Errorf("error at add mi info id=%s: %w", r.RequestID, err)
	}
	r.recordCreated("mi", mi.ID, r.GetRelatedTime())
	return nil
}

// AnchorTimeForRepeat は予定日時のうち最初に埋まっているものを基準にする。
// reps.Mi に RelatedTime 列は無いので、1つも埋まっていなければ繰り返しの入れ先が無い。
func (r *kftlMiRequest) AnchorTimeForRepeat() (time.Time, bool) {
	for _, t := range []*time.Time{r.estimateStartTime, r.estimateEndTime, r.limitTime} {
		if t != nil {
			return *t, true
		}
	}
	return time.Time{}, false
}

// CloneForRepeat は3つの予定日時を**同じ日数だけ**ずらす。
// 欄どうしの相対差（見積開始の2日後が期限、など）はそのまま保たれる。
func (r *kftlMiRequest) CloneForRepeat(newRequestID string, dayShift int) KFTLRequest {
	c := *r
	c.KFTLRequestBase = r.cloneBase(newRequestID, dayShift)
	c.estimateStartTime = shiftTimePtr(r.estimateStartTime, dayShift)
	c.estimateEndTime = shiftTimePtr(r.estimateEndTime, dayShift)
	c.limitTime = shiftTimePtr(r.limitTime, dayShift)
	return &c
}

// ─── Statement lines ──────────────────────────────────────────────────────────

// generateMiBlockNextConstructor は `ーみ` ブロックの中の「次の行」を決める先読み。
//
// タグ行・テキスト開始行は**項目の位置を消費しない**。汎用の行をそのまま使い、
// 「ブロックへ復帰する次行の決め方」だけを渡す(支出ブロックと同じやり方)。
// 渡さないと、ブロックの途中に `。タグ` と書いた時点でその行が板名や見積開始として
// 読まれてしまい、タグは付かないまま板名が "。タグ" になる。
//
// Mi のリクエストは ThisStatementLineTargetID をキーに request_map へ入っている
// (kftlStartMiStatementLine.ApplyThisLineToRequestMap)ので、
// MiReKyou と違って専用のタグ行は要らない。汎用のタグ行がそのまま Mi へタグを付ける。
//
// **`？` はここで拾ってはいけない。** 見積開始・見積終了・期限の3行は
// `？`/`?` で始まる行を入力エラーとして自分で弾く(parseScheduleFieldTime)。
// generateDefaultConstructor へ委譲すると `？` が関連時刻行に化けて、
// 空行で位置を送る既存の書き方が壊れる
// (reps.Mi に RelatedTime 列は無いので、関連時刻としての付け先もそもそも無い)。
//
// 空行も拾わない。空行は今までどおり項目の位置を消費する。
// Mirrors: generate_mi_block_next_constructor (kftl-mi-block.ts)
func generateMiBlockNextConstructor(nextLineText string, nextField StatementLineConstructorFunc) StatementLineConstructorFunc {
	resume := func(lineText string) StatementLineConstructorFunc {
		return generateMiBlockNextConstructor(lineText, nextField)
	}

	switch {
	case strings.HasPrefix(nextLineText, splitterTag) || strings.HasPrefix(nextLineText, splitterTagAscii):
		return func(lineText string, ctx *KFTLStatementLineContext) KFTLStatementLine {
			return newKFTLTagStatementLine(lineText, ctx, false, resume)
		}
	case nextLineText == splitterStartText || nextLineText == splitterStartTextAscii:
		return func(lineText string, ctx *KFTLStatementLineContext) KFTLStatementLine {
			return newKFTLStartTextStatementLine(lineText, ctx, false, resume)
		}
	case isRepeatSplitter(nextLineText):
		// 繰り返しブロックもタグ・テキストと同じく項目の位置を消費しない。
		// 拾わないと `？？` が予定日時の行として読まれ、`？` の禁止に引っかかる
		return func(lineText string, ctx *KFTLStatementLineContext) KFTLStatementLine {
			return newKFTLStartRepeatStatementLine(lineText, ctx, false, resume, nil)
		}
	}
	return nextField
}

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
	ctx.NextStatementLineConstructor = generateMiBlockNextConstructor(ctx.NextStatementLineText, func(lt string, c *KFTLStatementLineContext) KFTLStatementLine {
		return newKFTLMiTitleStatementLine(lt, c, req)
	})
	return &kftlStartMiStatementLine{lineText: lineText, ctx: ctx, req: req}
}

func (l *kftlStartMiStatementLine) ApplyThisLineToRequestMap(_ context.Context, requestMap *KFTLRequestMap) error {
	// タイトルの行が無いと無言で0件になり、なぜ作られなかったのか分からない。
	if err := requireNextLineText(l.ctx); err != nil {
		return err
	}
	return requestMap.Set(l.ctx.ThisStatementLineTargetID, l.req)
}
func (l *kftlStartMiStatementLine) GetLabelName() string                  { return "mi" }
func (l *kftlStartMiStatementLine) GetContext() *KFTLStatementLineContext { return l.ctx }
func (l *kftlStartMiStatementLine) GetStatementLineText() string          { return l.lineText }

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
	ctx.NextStatementLineConstructor = generateMiBlockNextConstructor(ctx.NextStatementLineText, func(lt string, c *KFTLStatementLineContext) KFTLStatementLine {
		return newKFTLMiBoardNameStatementLine(lt, c, req)
	})
	return &kftlMiTitleStatementLine{lineText: lineText, ctx: ctx, req: req}
}

func (l *kftlMiTitleStatementLine) ApplyThisLineToRequestMap(_ context.Context, _ *KFTLRequestMap) error {
	l.req.title = l.lineText
	return nil
}
func (l *kftlMiTitleStatementLine) GetLabelName() string                  { return "miTitle" }
func (l *kftlMiTitleStatementLine) GetContext() *KFTLStatementLineContext { return l.ctx }
func (l *kftlMiTitleStatementLine) GetStatementLineText() string          { return l.lineText }

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
	ctx.NextStatementLineConstructor = generateMiBlockNextConstructor(ctx.NextStatementLineText, func(lt string, c *KFTLStatementLineContext) KFTLStatementLine {
		return newKFTLMiEstimateStartTimeStatementLine(lt, c, req)
	})
	return &kftlMiBoardNameStatementLine{lineText: lineText, ctx: ctx, req: req}
}

func (l *kftlMiBoardNameStatementLine) ApplyThisLineToRequestMap(_ context.Context, _ *KFTLRequestMap) error {
	l.req.boardName = l.lineText
	return nil
}
func (l *kftlMiBoardNameStatementLine) GetLabelName() string                  { return "miBoardName" }
func (l *kftlMiBoardNameStatementLine) GetContext() *KFTLStatementLineContext { return l.ctx }
func (l *kftlMiBoardNameStatementLine) GetStatementLineText() string          { return l.lineText }

// kftlMiLimitTimeStatementLine reads the Mi limit time (optional, may be empty).
// After this, next constructor reverts to None.
// Mirrors: kftl-mi-limit-time-statement-line.ts
type kftlMiLimitTimeStatementLine struct {
	lineText string
	ctx      *KFTLStatementLineContext
	req      *kftlMiRequest
}

func newKFTLMiLimitTimeStatementLine(lineText string, ctx *KFTLStatementLineContext, req *kftlMiRequest) *kftlMiLimitTimeStatementLine {
	targetID := ctx.ThisStatementLineTargetID
	ctx.NextStatementLineTargetID = &targetID
	ctx.NextStatementLineConstructor = ctx.factory.generateNoneConstructor(ctx.NextStatementLineText)
	return &kftlMiLimitTimeStatementLine{lineText: lineText, ctx: ctx, req: req}
}

func (l *kftlMiLimitTimeStatementLine) ApplyThisLineToRequestMap(_ context.Context, _ *KFTLRequestMap) error {
	t, ok, err := parseScheduleFieldTime(l.lineText, l.ctx.BaseTime)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	l.req.limitTime = &t
	return nil
}
func (l *kftlMiLimitTimeStatementLine) GetLabelName() string                  { return "miLimitTime" }
func (l *kftlMiLimitTimeStatementLine) GetContext() *KFTLStatementLineContext { return l.ctx }
func (l *kftlMiLimitTimeStatementLine) GetStatementLineText() string          { return l.lineText }

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
	ctx.NextStatementLineConstructor = generateMiBlockNextConstructor(ctx.NextStatementLineText, func(lt string, c *KFTLStatementLineContext) KFTLStatementLine {
		return newKFTLMiEstimateEndTimeStatementLine(lt, c, req)
	})
	return &kftlMiEstimateStartTimeStatementLine{lineText: lineText, ctx: ctx, req: req}
}

func (l *kftlMiEstimateStartTimeStatementLine) ApplyThisLineToRequestMap(_ context.Context, _ *KFTLRequestMap) error {
	t, ok, err := parseScheduleFieldTime(l.lineText, l.ctx.BaseTime)
	if err != nil {
		return err
	}
	if !ok {
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
// Mirrors: kftl-mi-estimate-end-time-statement-line.ts
type kftlMiEstimateEndTimeStatementLine struct {
	lineText string
	ctx      *KFTLStatementLineContext
	req      *kftlMiRequest
}

func newKFTLMiEstimateEndTimeStatementLine(lineText string, ctx *KFTLStatementLineContext, req *kftlMiRequest) *kftlMiEstimateEndTimeStatementLine {
	targetID := ctx.ThisStatementLineTargetID
	ctx.NextStatementLineTargetID = &targetID
	ctx.NextStatementLineConstructor = generateMiBlockNextConstructor(ctx.NextStatementLineText, func(lt string, c *KFTLStatementLineContext) KFTLStatementLine {
		return newKFTLMiLimitTimeStatementLine(lt, c, req)
	})
	return &kftlMiEstimateEndTimeStatementLine{lineText: lineText, ctx: ctx, req: req}
}

func (l *kftlMiEstimateEndTimeStatementLine) ApplyThisLineToRequestMap(_ context.Context, _ *KFTLRequestMap) error {
	t, ok, err := parseScheduleFieldTime(l.lineText, l.ctx.BaseTime)
	if err != nil {
		return err
	}
	if !ok {
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
