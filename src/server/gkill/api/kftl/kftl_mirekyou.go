package kftl

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
	"github.com/mt3hr/gkill/src/server/gkill/dao/sqlite3impl"
)

// ─── KFTLMiReKyouRequest ──────────────────────────────────────────────────────

// kftlMiReKyouRequest records a MiReKyou (turns an existing Kyou into a task).
//
// request_mapのキーはMiReKyou自身のid(RequestID)で、タスク化する対象のidはtargetIDに別に持つ。
// 対象のidをキーにすると直前のKmemoリクエストを踏み潰してKFTLRequestMap.Setがエラーになる。
// Mirrors: kftl-mi-re-kyou-request.ts
type kftlMiReKyouRequest struct {
	KFTLRequestBase
	targetID          string
	boardName         string
	limitTime         *time.Time
	estimateStartTime *time.Time
	estimateEndTime   *time.Time
	// requestMap はApplyThisLineToRequestMapで注入される。宙ぶらりんのMiReKyouを書かないための検査に使う
	requestMap *KFTLRequestMap
}

func newKFTLMiReKyouRequest(requestID string, targetID string, ctx *KFTLStatementLineContext) *kftlMiReKyouRequest {
	return &kftlMiReKyouRequest{
		KFTLRequestBase: KFTLRequestBase{
			RequestID:  requestID,
			Ctx:        ctx,
			CreateTime: nowFromCtx(ctx),
		},
		targetID: targetID,
	}
}

func (r *kftlMiReKyouRequest) DoRequest(ctx context.Context) error {
	// タスク化する対象は「同じレコードで書いたKyou」。レコードにKyou本体が無い
	// (タグだけ書いてプロトタイプのまま終わった場合を含む)と、対象が存在しないMiReKyouになる。
	// そういうMiReKyouは検索でターゲット解決に失敗して結果から落ちるので、
	// 画面に出ないのに消せない行がリポジトリに残ってしまう。書く前に弾く
	if r.requestMap == nil {
		return fmt.Errorf("mirekyou request map is not set: id=%s", r.RequestID)
	}
	target, ok := r.requestMap.Get(r.targetID)
	if !ok {
		return fmt.Errorf("not found mirekyou target in this record: target_id=%s", r.targetID)
	}
	if _, isPrototype := target.(*KFTLPrototypeRequest); isPrototype {
		return fmt.Errorf("not found mirekyou target in this record (prototype only): target_id=%s", r.targetID)
	}

	// MiReKyouは後から追加されたrep種別なので、既存の設定DBには書き込み用repが無いことがある。
	// doBaseRequestより前に判定して、このリクエストは何も書かずに終わらせる
	if r.Ctx.Repositories == nil || r.Ctx.Repositories.WriteMiReKyouRep == nil {
		return fmt.Errorf("not exist write mirekyou rep user id = %s device = %s", r.Ctx.UserID, r.Ctx.Device)
	}

	boardName := r.boardName
	if boardName == "" && r.Ctx.ApplicationConfig != nil {
		boardName = r.Ctx.ApplicationConfig.MiDefaultBoard
	}

	// ブロックの中に書いたタグはここでMiReKyou自身に書かれる
	if err := r.doBaseRequest(ctx, r.RequestID); err != nil {
		return err
	}

	now := r.CreateTime
	mirekyou := reps.MiReKyou{
		ID:                r.RequestID,
		TargetID:          r.targetID,
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
	if err := r.Ctx.Repositories.WriteMiReKyouRep.AddMiReKyouInfo(ctx, mirekyou); err != nil {
		return fmt.Errorf("error at add mirekyou info id=%s: %w", r.RequestID, err)
	}
	repName, _ := r.Ctx.Repositories.WriteMiReKyouRep.GetRepName(ctx)
	// ReKyouと同じくTargetIDInDataにリポスト対象のIDを入れる
	// (usecase.updateMiReKyouLatestDataRepositoryAddress と揃える)
	targetIDInData := r.targetID
	updateLatestDataRepositoryAddress(ctx, r.Ctx.Repositories, r.RequestID, &targetIDInData, false, now, repName)
	// キャッシュに書き込み
	_ = r.Ctx.Repositories.WriteThroughMiReKyouCache(ctx, mirekyou)
	return nil
}

// ─── ブロックの次の行を決める先読み ──────────────────────────────────────────

// generateMiReKyouNextConstructor decides the constructor for the next line inside a MiReKyou block.
//
// 閉じる行 > タグ行 > 次の項目行 の順に見る。
// タグ行は項目の位置を消費しないので、タグを挟んでも次の非タグ行が次の項目になる。
// Mirrors: KFTLMiReKyouTagStatementLine.generate_next_constructor
func generateMiReKyouNextConstructor(nextLineText string, req *kftlMiReKyouRequest, prevLineIsMetaInfo bool, nextFieldConstructor StatementLineConstructorFunc) StatementLineConstructorFunc {
	if isMiReKyouSplitter(nextLineText) {
		return func(lt string, c *KFTLStatementLineContext) KFTLStatementLine {
			return newKFTLEndMiReKyouStatementLine(lt, c, prevLineIsMetaInfo)
		}
	}
	if strings.HasPrefix(nextLineText, splitterTag) || strings.HasPrefix(nextLineText, splitterTagAscii) {
		return func(lt string, c *KFTLStatementLineContext) KFTLStatementLine {
			return newKFTLMiReKyouTagStatementLine(lt, c, req, prevLineIsMetaInfo, nextFieldConstructor)
		}
	}
	return nextFieldConstructor
}

// generateMiReKyouAfterLastFieldConstructor decides the constructor for the line after the last field.
// タグ行か閉じる行しか来られない。それ以外の行はタグ行として作られてApply時にエラーになる。
// Mirrors: KFTLMiReKyouTagStatementLine.generate_after_last_field_constructor
func generateMiReKyouAfterLastFieldConstructor(nextLineText string, req *kftlMiReKyouRequest, prevLineIsMetaInfo bool) StatementLineConstructorFunc {
	var stayInBlock StatementLineConstructorFunc
	stayInBlock = func(lt string, c *KFTLStatementLineContext) KFTLStatementLine {
		return newKFTLMiReKyouTagStatementLine(lt, c, req, prevLineIsMetaInfo, stayInBlock)
	}
	return generateMiReKyouNextConstructor(nextLineText, req, prevLineIsMetaInfo, stayInBlock)
}

// ─── Statement lines ──────────────────────────────────────────────────────────

// kftlStartMiReKyouStatementLine opens a MiReKyou block (line == "～～").
//
// 同じレコードで書いたKyouをタスク化するので、バケツリレーされてきたtarget_idを
// そのまま対象として使う(Miと違ってここでtarget_idを作り直さないこと)。
// Mirrors: kftl-start-mi-re-kyou-statement-line.ts
type kftlStartMiReKyouStatementLine struct {
	lineText string
	ctx      *KFTLStatementLineContext
	req      *kftlMiReKyouRequest
}

func newKFTLStartMiReKyouStatementLine(lineText string, ctx *KFTLStatementLineContext, prevLineIsMetaInfo bool) *kftlStartMiReKyouStatementLine {
	targetID := ctx.ThisStatementLineTargetID
	// ブロックはKyou本体ではなく付随情報なので、プロトタイプかどうかを次の行へ伝える。
	// 伝えないと「ブロックを先に書いてあとからメモを書く」並びでメモが別のidを引き当てて、
	// このMiReKyouの対象が消える
	ctx.NextIsPrototype = ctx.ThisIsPrototype
	ctx.NextStatementLineTargetID = &targetID

	req := newKFTLMiReKyouRequest(sqlite3impl.GenerateNewID(), targetID, ctx)
	ctx.NextStatementLineConstructor = generateMiReKyouNextConstructor(ctx.NextStatementLineText, req, prevLineIsMetaInfo,
		func(lt string, c *KFTLStatementLineContext) KFTLStatementLine {
			return newKFTLMiReKyouBoardNameStatementLine(lt, c, req, prevLineIsMetaInfo)
		})
	return &kftlStartMiReKyouStatementLine{lineText: lineText, ctx: ctx, req: req}
}

func (l *kftlStartMiReKyouStatementLine) ApplyThisLineToRequestMap(_ context.Context, requestMap *KFTLRequestMap) error {
	// キーはMiReKyou自身のid。対象のidをキーにすると直前のKyouのリクエストを踏み潰す
	l.req.requestMap = requestMap
	return requestMap.Set(l.req.RequestID, l.req)
}

func (l *kftlStartMiReKyouStatementLine) GetLabelName() string                  { return "mirekyou" }
func (l *kftlStartMiReKyouStatementLine) GetContext() *KFTLStatementLineContext { return l.ctx }
func (l *kftlStartMiReKyouStatementLine) GetStatementLineText() string          { return l.lineText }

// kftlMiReKyouBoardNameStatementLine reads the MiReKyou board name.
// Mirrors: kftl-mi-re-kyou-board-name-statement-line.ts
type kftlMiReKyouBoardNameStatementLine struct {
	lineText string
	ctx      *KFTLStatementLineContext
	req      *kftlMiReKyouRequest
}

func newKFTLMiReKyouBoardNameStatementLine(lineText string, ctx *KFTLStatementLineContext, req *kftlMiReKyouRequest, prevLineIsMetaInfo bool) *kftlMiReKyouBoardNameStatementLine {
	ctx.NextIsPrototype = ctx.ThisIsPrototype
	targetID := ctx.ThisStatementLineTargetID
	ctx.NextStatementLineTargetID = &targetID
	ctx.NextStatementLineConstructor = generateMiReKyouNextConstructor(ctx.NextStatementLineText, req, prevLineIsMetaInfo,
		func(lt string, c *KFTLStatementLineContext) KFTLStatementLine {
			return newKFTLMiReKyouEstimateStartTimeStatementLine(lt, c, req, prevLineIsMetaInfo)
		})
	return &kftlMiReKyouBoardNameStatementLine{lineText: lineText, ctx: ctx, req: req}
}

func (l *kftlMiReKyouBoardNameStatementLine) ApplyThisLineToRequestMap(_ context.Context, _ *KFTLRequestMap) error {
	l.req.boardName = l.lineText
	return nil
}

func (l *kftlMiReKyouBoardNameStatementLine) GetLabelName() string { return "mirekyouBoardName" }
func (l *kftlMiReKyouBoardNameStatementLine) GetContext() *KFTLStatementLineContext {
	return l.ctx
}
func (l *kftlMiReKyouBoardNameStatementLine) GetStatementLineText() string { return l.lineText }

// kftlMiReKyouEstimateStartTimeStatementLine reads the MiReKyou estimate start time (optional).
// 「？」/「?」は付けても付けなくてもよい。
// Mirrors: kftl-mi-re-kyou-estimate-start-time-statement-line.ts
type kftlMiReKyouEstimateStartTimeStatementLine struct {
	lineText string
	ctx      *KFTLStatementLineContext
	req      *kftlMiReKyouRequest
}

func newKFTLMiReKyouEstimateStartTimeStatementLine(lineText string, ctx *KFTLStatementLineContext, req *kftlMiReKyouRequest, prevLineIsMetaInfo bool) *kftlMiReKyouEstimateStartTimeStatementLine {
	ctx.NextIsPrototype = ctx.ThisIsPrototype
	targetID := ctx.ThisStatementLineTargetID
	ctx.NextStatementLineTargetID = &targetID
	ctx.NextStatementLineConstructor = generateMiReKyouNextConstructor(ctx.NextStatementLineText, req, prevLineIsMetaInfo,
		func(lt string, c *KFTLStatementLineContext) KFTLStatementLine {
			return newKFTLMiReKyouEstimateEndTimeStatementLine(lt, c, req, prevLineIsMetaInfo)
		})
	return &kftlMiReKyouEstimateStartTimeStatementLine{lineText: lineText, ctx: ctx, req: req}
}

func (l *kftlMiReKyouEstimateStartTimeStatementLine) ApplyThisLineToRequestMap(_ context.Context, _ *KFTLRequestMap) error {
	t, ok := parseMiReKyouTime(l.lineText)
	if ok {
		l.req.estimateStartTime = &t
	}
	return nil
}

func (l *kftlMiReKyouEstimateStartTimeStatementLine) GetLabelName() string {
	return "mirekyouEstimateStartTime"
}
func (l *kftlMiReKyouEstimateStartTimeStatementLine) GetContext() *KFTLStatementLineContext {
	return l.ctx
}
func (l *kftlMiReKyouEstimateStartTimeStatementLine) GetStatementLineText() string {
	return l.lineText
}

// kftlMiReKyouEstimateEndTimeStatementLine reads the MiReKyou estimate end time (optional).
// Mirrors: kftl-mi-re-kyou-estimate-end-time-statement-line.ts
type kftlMiReKyouEstimateEndTimeStatementLine struct {
	lineText string
	ctx      *KFTLStatementLineContext
	req      *kftlMiReKyouRequest
}

func newKFTLMiReKyouEstimateEndTimeStatementLine(lineText string, ctx *KFTLStatementLineContext, req *kftlMiReKyouRequest, prevLineIsMetaInfo bool) *kftlMiReKyouEstimateEndTimeStatementLine {
	ctx.NextIsPrototype = ctx.ThisIsPrototype
	targetID := ctx.ThisStatementLineTargetID
	ctx.NextStatementLineTargetID = &targetID
	ctx.NextStatementLineConstructor = generateMiReKyouNextConstructor(ctx.NextStatementLineText, req, prevLineIsMetaInfo,
		func(lt string, c *KFTLStatementLineContext) KFTLStatementLine {
			return newKFTLMiReKyouLimitTimeStatementLine(lt, c, req, prevLineIsMetaInfo)
		})
	return &kftlMiReKyouEstimateEndTimeStatementLine{lineText: lineText, ctx: ctx, req: req}
}

func (l *kftlMiReKyouEstimateEndTimeStatementLine) ApplyThisLineToRequestMap(_ context.Context, _ *KFTLRequestMap) error {
	t, ok := parseMiReKyouTime(l.lineText)
	if ok {
		l.req.estimateEndTime = &t
	}
	return nil
}

func (l *kftlMiReKyouEstimateEndTimeStatementLine) GetLabelName() string {
	return "mirekyouEstimateEndTime"
}
func (l *kftlMiReKyouEstimateEndTimeStatementLine) GetContext() *KFTLStatementLineContext {
	return l.ctx
}
func (l *kftlMiReKyouEstimateEndTimeStatementLine) GetStatementLineText() string { return l.lineText }

// kftlMiReKyouLimitTimeStatementLine reads the MiReKyou limit time (optional).
// 項目行はここで終わり。このあとはタグ行を好きなだけ書けて、「～～」で閉じる。
// Mirrors: kftl-mi-re-kyou-limit-time-statement-line.ts
type kftlMiReKyouLimitTimeStatementLine struct {
	lineText string
	ctx      *KFTLStatementLineContext
	req      *kftlMiReKyouRequest
}

func newKFTLMiReKyouLimitTimeStatementLine(lineText string, ctx *KFTLStatementLineContext, req *kftlMiReKyouRequest, prevLineIsMetaInfo bool) *kftlMiReKyouLimitTimeStatementLine {
	ctx.NextIsPrototype = ctx.ThisIsPrototype
	targetID := ctx.ThisStatementLineTargetID
	ctx.NextStatementLineTargetID = &targetID
	ctx.NextStatementLineConstructor = generateMiReKyouAfterLastFieldConstructor(ctx.NextStatementLineText, req, prevLineIsMetaInfo)
	return &kftlMiReKyouLimitTimeStatementLine{lineText: lineText, ctx: ctx, req: req}
}

func (l *kftlMiReKyouLimitTimeStatementLine) ApplyThisLineToRequestMap(_ context.Context, _ *KFTLRequestMap) error {
	t, ok := parseMiReKyouTime(l.lineText)
	if ok {
		l.req.limitTime = &t
	}
	return nil
}

func (l *kftlMiReKyouLimitTimeStatementLine) GetLabelName() string { return "mirekyouLimitTime" }
func (l *kftlMiReKyouLimitTimeStatementLine) GetContext() *KFTLStatementLineContext {
	return l.ctx
}
func (l *kftlMiReKyouLimitTimeStatementLine) GetStatementLineText() string { return l.lineText }

// kftlMiReKyouTagStatementLine handles tag lines inside a MiReKyou block.
//
// ここで足したタグは対象のKyouではなくMiReKyou自身に付く。
// target_idを切り替えるのではなく、掴んでいるリクエストへ直接AddTagする
// (target_idは対象のKyouのままブロックを通り抜けるので、閉じたあとのタグは対象に付く)。
// Mirrors: kftl-mi-re-kyou-tag-statement-line.ts
type kftlMiReKyouTagStatementLine struct {
	lineText string
	ctx      *KFTLStatementLineContext
	req      *kftlMiReKyouRequest
}

func newKFTLMiReKyouTagStatementLine(lineText string, ctx *KFTLStatementLineContext, req *kftlMiReKyouRequest, prevLineIsMetaInfo bool, nextFieldConstructor StatementLineConstructorFunc) *kftlMiReKyouTagStatementLine {
	ctx.NextIsPrototype = ctx.ThisIsPrototype
	targetID := ctx.ThisStatementLineTargetID
	ctx.NextStatementLineTargetID = &targetID
	ctx.NextStatementLineConstructor = generateMiReKyouNextConstructor(ctx.NextStatementLineText, req, prevLineIsMetaInfo, nextFieldConstructor)
	return &kftlMiReKyouTagStatementLine{lineText: lineText, ctx: ctx, req: req}
}

func (l *kftlMiReKyouTagStatementLine) ApplyThisLineToRequestMap(_ context.Context, _ *KFTLRequestMap) error {
	if l.lineText == "" {
		return nil
	}
	if !strings.HasPrefix(l.lineText, splitterTag) && !strings.HasPrefix(l.lineText, splitterTagAscii) {
		// 項目行を全部書き終えたあとに来られるのはタグ行か閉じる行だけ。
		// テキストは飲み込まずにエラーにする。飲み込むと、閉じ忘れたときに
		// メモの本文が丸ごとタグになってしまう
		return fmt.Errorf("mirekyou block is not closed: line=%q", l.lineText)
	}
	tagStr := strings.TrimPrefix(l.lineText, splitterTag)
	tagStr = strings.TrimPrefix(tagStr, splitterTagAscii)
	tags := strings.FieldsFunc(tagStr, func(r rune) bool { return r == '、' || r == ',' })
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag != "" {
			l.req.AddTag(tag)
		}
	}
	return nil
}

func (l *kftlMiReKyouTagStatementLine) GetLabelName() string                  { return "mirekyouTag" }
func (l *kftlMiReKyouTagStatementLine) GetContext() *KFTLStatementLineContext { return l.ctx }
func (l *kftlMiReKyouTagStatementLine) GetStatementLineText() string          { return l.lineText }

// kftlEndMiReKyouStatementLine closes a MiReKyou block (line == "～～").
// ここから先は再びタスク化する対象のKyou側に戻る。
// Mirrors: kftl-end-mi-re-kyou-statement-line.ts
type kftlEndMiReKyouStatementLine struct {
	lineText string
	ctx      *KFTLStatementLineContext
}

func newKFTLEndMiReKyouStatementLine(lineText string, ctx *KFTLStatementLineContext, prevLineIsMetaInfo bool) *kftlEndMiReKyouStatementLine {
	ctx.NextIsPrototype = ctx.ThisIsPrototype
	targetID := ctx.ThisStatementLineTargetID
	ctx.NextStatementLineTargetID = &targetID

	if prevLineIsMetaInfo {
		ctx.NextStatementLineConstructor = ctx.factory.generateKmemoConstructor(ctx.NextStatementLineText)
	} else {
		ctx.NextStatementLineConstructor = ctx.factory.generateNoneConstructor(ctx.NextStatementLineText)
	}

	return &kftlEndMiReKyouStatementLine{lineText: lineText, ctx: ctx}
}

func (l *kftlEndMiReKyouStatementLine) ApplyThisLineToRequestMap(_ context.Context, _ *KFTLRequestMap) error {
	// 閉じるだけ。リクエストへの書き込みは無い
	return nil
}

func (l *kftlEndMiReKyouStatementLine) GetLabelName() string                  { return "endMirekyou" }
func (l *kftlEndMiReKyouStatementLine) GetContext() *KFTLStatementLineContext { return l.ctx }
func (l *kftlEndMiReKyouStatementLine) GetStatementLineText() string          { return l.lineText }

// parseMiReKyouTime parses an optional datetime line.
// 「？」/「?」は付いていてもいなくてもよい。空行やパースできない行は未設定として扱う。
func parseMiReKyouTime(lineText string) (time.Time, bool) {
	s := strings.TrimPrefix(lineText, splitterRelatedTime)
	s = strings.TrimPrefix(s, splitterRelatedTimeAscii)
	if strings.TrimSpace(s) == "" {
		return time.Time{}, false
	}
	t, err := parseDateTime(s)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}
