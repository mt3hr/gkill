package kftl

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
	"github.com/mt3hr/gkill/src/server/gkill/dao/sqlite3impl"
)

// ─── KFTLNlogBlock ────────────────────────────────────────────────────────────

// kftlNlogBlock は支出ブロック(`ーん`)で支払いをまたいで共有する状態。
//
// 支払い(品名と金額のペア)は1組ずつ別の kftlNlogRequest になるので、タグとテキストは
// 「直前の支払い」だけに付く。全支払いで共有するのは店名と関連時刻の2つだけ。
// Mirrors: KFTLNlogBlock (kftl-nlog-block.ts)
type kftlNlogBlock struct {
	// ブロックの入口の target_id。`ーん` の前に書かれたメタ情報行を見つけるために持つ
	blockTargetID string
	// 全支払いで共有する店名
	shop string
	// `？`行で指定された関連時刻。ブロックの中のどこに書いてもブロック全体に効く
	relatedTime *time.Time
}

// generateNlogBlockNextConstructor は支出ブロックの中の「次の行」を決める先読み。
//
// タグ行・テキスト開始行は汎用の行をそのまま使い、「ブロックへ復帰する次行の決め方」だけを渡す。
// 渡さないとブロックの途中にタグを書いた時点でブロックが切れて、以降の品名行が拾われなくなる。
// 関連時刻だけは付け先が支払いではなくブロックなので専用の行にする。
// Mirrors: generate_nlog_block_next_constructor (kftl-nlog-block.ts)
func generateNlogBlockNextConstructor(nextLineText string, block *kftlNlogBlock, factory *kftlFactory) StatementLineConstructorFunc {
	resume := func(lineText string) StatementLineConstructorFunc {
		return generateNlogBlockNextConstructor(lineText, block, factory)
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
	case strings.HasPrefix(nextLineText, splitterRelatedTime) || strings.HasPrefix(nextLineText, splitterRelatedTimeAscii):
		return func(lineText string, ctx *KFTLStatementLineContext) KFTLStatementLine {
			return newKFTLNlogRelatedTimeStatementLine(lineText, ctx, block)
		}
	}
	return factory.generateNlogConstructor(nextLineText, block)
}

// assertIsNotMetaInfoLine は店名行・最初の品名行がタグ行やテキスト開始行になっていないか検査する。
//
// この2箇所は次の行が固定なので、`。タグ` と書くと店名や品名がその文字列になってしまう。
// 支出のタグは「直前の支払い」に付ける仕様で、この位置には直前の支払いがまだ無い。
// Mirrors: assert_is_not_meta_info_line (kftl-nlog-block.ts)
func assertIsNotMetaInfoLine(lineText string) error {
	if strings.HasPrefix(lineText, splitterTag) || strings.HasPrefix(lineText, splitterTagAscii) ||
		lineText == splitterStartText || lineText == splitterStartTextAscii {
		return fmt.Errorf("nlog tags and texts must be written after the amount line: %q", lineText)
	}
	return nil
}

// ─── KFTLNlogRequest ──────────────────────────────────────────────────────────

// kftlNlogRequest は支払い1件(品名と金額のペア1組)ぶんのリクエスト。
//
// 1つの `ーん` ブロックからは支払いの数だけこのリクエストが出る。RequestID をそのまま
// Nlog の ID にしているので、doBaseRequest が書くタグ・テキストがその支払いを正しく指す
// (以前はブロックで1つのリクエストにまとめており、2件目以降の支払いにはタグが付かなかった)。
// Mirrors: kftl-nlog-request.ts
type kftlNlogRequest struct {
	KFTLRequestBase
	block     *kftlNlogBlock
	title     string
	amount    json.Number
	hasAmount bool
}

func newKFTLNlogRequest(requestID string, ctx *KFTLStatementLineContext, block *kftlNlogBlock) *kftlNlogRequest {
	return &kftlNlogRequest{
		KFTLRequestBase: KFTLRequestBase{
			RequestID:  requestID,
			Ctx:        ctx,
			CreateTime: nowFromCtx(ctx),
		},
		block: block,
	}
}

// GetRelatedTime は関連時刻をブロック全体で共有する。
// `？`行をブロックの中のどこに書いても、そのブロックの全支払いが同じ時刻になる。
// Mirrors: KFTLNlogRequest.get_related_time()
func (r *kftlNlogRequest) GetRelatedTime() time.Time {
	if r.block.relatedTime != nil {
		return *r.block.relatedTime
	}
	return r.KFTLRequestBase.GetRelatedTime()
}

func (r *kftlNlogRequest) DoRequest(ctx context.Context) error {
	// 末尾の改行が品名行として解釈されただけの空の支払い。エラーにせず、支払いも作らない
	if r.title == "" && !r.hasAmount {
		return nil
	}
	// 品名だけ書いて金額行が無い。取りこぼしになるので黙って切り詰めずエラーにする
	if !r.hasAmount {
		return fmt.Errorf("nlog title has no amount: %q", r.title)
	}

	if err := r.doBaseRequest(ctx, r.RequestID); err != nil {
		return err
	}
	relatedTime := r.GetRelatedTime()
	now := r.CreateTime

	// Note: Transaction safety is provided by the KFTL submit flow's temp repositories.
	// Each nlog is inserted independently; if any fails, the entire KFTL submit rolls back.
	nlog := reps.Nlog{
		ID:           r.RequestID,
		Shop:         r.block.shop,
		Title:        r.title,
		Amount:       r.amount,
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
		return fmt.Errorf("error at add nlog info id=%s: %w", r.RequestID, err)
	}
	repName, repNameErr := r.Ctx.Repositories.WriteNlogRep.GetRepName(ctx)
	logGetRepNameFailure(ctx, "nlog", nlog.ID, repNameErr)
	updateLatestDataRepositoryAddress(ctx, r.Ctx.Repositories, r.RequestID, nil, false, now, repName)
	// キャッシュに書き込み
	logWriteThroughCacheFailure(ctx, "nlog", nlog.ID, r.Ctx.Repositories.WriteThroughNlogCache(ctx, nlog))
	return nil
}

// ─── Statement lines ──────────────────────────────────────────────────────────

// kftlStartNlogStatementLine handles "ーん".
// Mirrors: kftl-start-nlog-statement-line.ts
type kftlStartNlogStatementLine struct {
	lineText string
	ctx      *KFTLStatementLineContext
	block    *kftlNlogBlock
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

	block := &kftlNlogBlock{blockTargetID: targetID}
	ctx.NextStatementLineConstructor = func(lt string, c *KFTLStatementLineContext) KFTLStatementLine {
		return newKFTLNlogShopNameStatementLine(lt, c, block)
	}
	return &kftlStartNlogStatementLine{lineText: lineText, ctx: ctx, block: block}
}

// ApplyThisLineToRequestMap は開始行ではリクエストを作らない。支払いは品名行が1組ずつ作る。
//
// ここでやるのは「`ーん` より前に書かれたメタ情報行」の検査だけ。支出のタグとテキストは
// 直前の支払いに付ける仕様なので、ブロックの前に書かれていたら黙って捨てずにエラーにする。
// 関連時刻だけはブロック全体に効くので取り込む。
func (l *kftlStartNlogStatementLine) ApplyThisLineToRequestMap(_ context.Context, requestMap *KFTLRequestMap) error {
	prevRequest, ok := requestMap.Get(l.ctx.ThisStatementLineTargetID)
	if !ok {
		return nil
	}
	proto, isProto := prevRequest.(*KFTLPrototypeRequest)
	if !isProto {
		// 区切らずにメモの直後へ書いた場合。従来 KFTLRequestMap.Set が返していたのと同じエラー
		return fmt.Errorf("request id=%s is already set and is not a prototype", l.ctx.ThisStatementLineTargetID)
	}
	if 0 < len(proto.GetTags()) || 0 < len(proto.GetTextsMap()) {
		return fmt.Errorf("nlog tags and texts must be written after the amount line")
	}
	l.block.relatedTime = proto.relatedTime
	return nil
}
func (l *kftlStartNlogStatementLine) GetLabelName() string                  { return "nlog" }
func (l *kftlStartNlogStatementLine) GetContext() *KFTLStatementLineContext { return l.ctx }
func (l *kftlStartNlogStatementLine) GetStatementLineText() string          { return l.lineText }

// kftlNlogShopNameStatementLine reads the shop name.
// Mirrors: kftl-nlog-shop-name-statement-line.ts
type kftlNlogShopNameStatementLine struct {
	lineText string
	ctx      *KFTLStatementLineContext
	block    *kftlNlogBlock
}

func newKFTLNlogShopNameStatementLine(lineText string, ctx *KFTLStatementLineContext, block *kftlNlogBlock) *kftlNlogShopNameStatementLine {
	targetID := ctx.ThisStatementLineTargetID
	ctx.NextStatementLineTargetID = &targetID
	ctx.NextStatementLineConstructor = func(lt string, c *KFTLStatementLineContext) KFTLStatementLine {
		return newKFTLNlogTitleStatementLine(lt, c, block)
	}
	return &kftlNlogShopNameStatementLine{lineText: lineText, ctx: ctx, block: block}
}

func (l *kftlNlogShopNameStatementLine) ApplyThisLineToRequestMap(_ context.Context, _ *KFTLRequestMap) error {
	if err := assertIsNotMetaInfoLine(l.lineText); err != nil {
		return err
	}
	l.block.shop = l.lineText
	return nil
}
func (l *kftlNlogShopNameStatementLine) GetLabelName() string                  { return "nlogShop" }
func (l *kftlNlogShopNameStatementLine) GetContext() *KFTLStatementLineContext { return l.ctx }
func (l *kftlNlogShopNameStatementLine) GetStatementLineText() string          { return l.lineText }

// kftlNlogTitleStatementLine reads a title for one Nlog item.
// ここが1件の支払いの始まりで、支払いごとに target_id を採番し直す。
// Mirrors: kftl-nlog-title-statement-line.ts
type kftlNlogTitleStatementLine struct {
	lineText string
	ctx      *KFTLStatementLineContext
	block    *kftlNlogBlock
}

func newKFTLNlogTitleStatementLine(lineText string, ctx *KFTLStatementLineContext, block *kftlNlogBlock) *kftlNlogTitleStatementLine {
	paymentID := sqlite3impl.GenerateNewID()
	ctx.ThisStatementLineTargetID = paymentID
	ctx.NextStatementLineTargetID = &paymentID
	ctx.NextStatementLineConstructor = func(lt string, c *KFTLStatementLineContext) KFTLStatementLine {
		return newKFTLNlogAmountStatementLine(lt, c, block)
	}
	return &kftlNlogTitleStatementLine{lineText: lineText, ctx: ctx, block: block}
}

func (l *kftlNlogTitleStatementLine) ApplyThisLineToRequestMap(_ context.Context, requestMap *KFTLRequestMap) error {
	// 店名の次は品名行で固定なので、ここへタグ行やテキスト開始行が来ると
	// その文字列が品名になってしまう。飲み込まずにエラーにする
	if err := assertIsNotMetaInfoLine(l.lineText); err != nil {
		return err
	}
	req := newKFTLNlogRequest(l.ctx.ThisStatementLineTargetID, l.ctx, l.block)
	req.title = l.lineText
	return requestMap.Set(l.ctx.ThisStatementLineTargetID, req)
}
func (l *kftlNlogTitleStatementLine) GetLabelName() string                  { return "nlogTitle" }
func (l *kftlNlogTitleStatementLine) GetContext() *KFTLStatementLineContext { return l.ctx }
func (l *kftlNlogTitleStatementLine) GetStatementLineText() string          { return l.lineText }

// kftlNlogAmountStatementLine reads the amount for one Nlog item.
// After reading, next constructor stays inside the block so that tags and text
// blocks can be attached to this payment before the next item starts.
// Mirrors: kftl-nlog-amount-statement-line.ts
type kftlNlogAmountStatementLine struct {
	lineText string
	ctx      *KFTLStatementLineContext
	block    *kftlNlogBlock
}

func newKFTLNlogAmountStatementLine(lineText string, ctx *KFTLStatementLineContext, block *kftlNlogBlock) *kftlNlogAmountStatementLine {
	targetID := ctx.ThisStatementLineTargetID
	ctx.NextStatementLineTargetID = &targetID
	ctx.NextStatementLineConstructor = generateNlogBlockNextConstructor(ctx.NextStatementLineText, block, ctx.factory)
	return &kftlNlogAmountStatementLine{lineText: lineText, ctx: ctx, block: block}
}

func (l *kftlNlogAmountStatementLine) ApplyThisLineToRequestMap(_ context.Context, requestMap *KFTLRequestMap) error {
	if l.lineText == "" {
		return fmt.Errorf("nlog amount is empty")
	}
	num := json.Number(l.lineText)
	if _, err := num.Float64(); err != nil {
		return fmt.Errorf("invalid nlog amount %q: %w", l.lineText, err)
	}
	req, ok := requestMap.Get(l.ctx.ThisStatementLineTargetID)
	if !ok {
		return fmt.Errorf("nlog request not found id=%s", l.ctx.ThisStatementLineTargetID)
	}
	nlogRequest, ok := req.(*kftlNlogRequest)
	if !ok {
		return fmt.Errorf("request id=%s is not an nlog request", l.ctx.ThisStatementLineTargetID)
	}
	nlogRequest.amount = num
	nlogRequest.hasAmount = true
	return nil
}
func (l *kftlNlogAmountStatementLine) GetLabelName() string                  { return "nlogAmount" }
func (l *kftlNlogAmountStatementLine) GetContext() *KFTLStatementLineContext { return l.ctx }
func (l *kftlNlogAmountStatementLine) GetStatementLineText() string          { return l.lineText }

// kftlNlogRelatedTimeStatementLine は支出ブロックの中の関連時刻行。
//
// 付け先は直前の支払いではなくブロックで、ブロックの中のどこに書いても全支払いに効く。
// リクエストの実行は行の解釈が全部終わったあとなので、自分より前に作られた支払いにも効く。
// Mirrors: kftl-nlog-related-time-statement-line.ts
type kftlNlogRelatedTimeStatementLine struct {
	lineText string
	ctx      *KFTLStatementLineContext
	block    *kftlNlogBlock
}

func newKFTLNlogRelatedTimeStatementLine(lineText string, ctx *KFTLStatementLineContext, block *kftlNlogBlock) *kftlNlogRelatedTimeStatementLine {
	ctx.NextIsPrototype = ctx.ThisIsPrototype
	targetID := ctx.ThisStatementLineTargetID
	ctx.NextStatementLineTargetID = &targetID
	ctx.NextStatementLineConstructor = generateNlogBlockNextConstructor(ctx.NextStatementLineText, block, ctx.factory)
	return &kftlNlogRelatedTimeStatementLine{lineText: lineText, ctx: ctx, block: block}
}

func (l *kftlNlogRelatedTimeStatementLine) ApplyThisLineToRequestMap(_ context.Context, _ *KFTLRequestMap) error {
	timeStr := strings.TrimPrefix(l.lineText, splitterRelatedTime)
	timeStr = strings.TrimPrefix(timeStr, splitterRelatedTimeAscii)
	parsed, err := parseDateTime(timeStr)
	if err != nil {
		return fmt.Errorf("invalid nlog related time %q: %w", l.lineText, err)
	}
	l.block.relatedTime = &parsed
	return nil
}
func (l *kftlNlogRelatedTimeStatementLine) GetLabelName() string { return "relatedTime" }
func (l *kftlNlogRelatedTimeStatementLine) GetContext() *KFTLStatementLineContext {
	return l.ctx
}
func (l *kftlNlogRelatedTimeStatementLine) GetStatementLineText() string { return l.lineText }
