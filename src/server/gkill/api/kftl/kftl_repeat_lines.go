package kftl

import (
	"context"
	"fmt"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/dao/sqlite3impl"
)

// 繰り返しブロック「？？」の行と、展開。仕様の計算そのものは kftl_repeat.go。
//
// ブロックは開始・終了とも `？？` の単独行で、間に4行を順に書く。
// 閉じる行を最優先で見るので、3行目・4行目を省いて早く閉じられる。
//
//	？？
//	金             1行目 繰り返し条件（必須）
//	3              2行目 回数 または 終了日（必須）
//	no             3行目 既存があっても追加するか（Optional・既定 no）
//	2026-09-12     4行目 起点（Optional・既定 BaseTime）
//	？？
//
// Mirrors: src/client/classes/kftl/kftl_repeat/

// 行の位置。索引でしか区別しないので、順序を変えるときはクライアント側と一緒に変えること。
const (
	repeatFieldCondition = iota
	repeatFieldCount
	repeatFieldAddIfExists
	repeatFieldOrigin
	// これより後ろの位置は受け皿。空行は見逃し、テキストは default で弾く
	// （クライアントはここに「**********」の行ラベルを出す）
)

// ─── 開始行 ───────────────────────────────────────────────────────────────────

// kftlStartRepeatStatementLine handles "？？" that opens a repeat block.
//
// タグ行・テキスト開始行と同じく**項目の位置を消費しない**。ブロック（Mi / MiReKyou / 支出）の
// 中に書いても、閉じたあとは resume でブロックへ戻る。
type kftlStartRepeatStatementLine struct {
	lineText string
	ctx      *KFTLStatementLineContext
	spec     *repeatSpec
	// target は付け先を明示するとき。nil なら target_id で request_map から引く。
	//
	// **リポストタスク（`～～`）は必ず渡すこと。** ブロックの中の target_id は
	// 「タスク化される元の記録」を指していて、リポストタスク自身は別のIDで登録されている。
	// 引かせると元の記録のほうが繰り返されてしまう（タグ行が req を持ち回っているのと同じ理由）。
	target KFTLRequest
}

func newKFTLStartRepeatStatementLine(lineText string, ctx *KFTLStatementLineContext, prevLineIsMetaInfo bool, resume resumeConstructorFunc, target KFTLRequest) *kftlStartRepeatStatementLine {
	ctx.NextIsPrototype = ctx.ThisIsPrototype
	targetID := ctx.ThisStatementLineTargetID
	ctx.NextStatementLineTargetID = &targetID

	// 展開でエラーになったとき何行目を指すか。展開は実行ループの直前なので、
	// ここで控えておかないと応答から原因の行が分からない
	spec := &repeatSpec{lineIndex: ctx.LineIndex, lineText: lineText}
	ctx.NextStatementLineConstructor = generateRepeatBlockNextConstructor(
		ctx.NextStatementLineText, spec, repeatFieldCondition, prevLineIsMetaInfo, resume)

	return &kftlStartRepeatStatementLine{lineText: lineText, ctx: ctx, spec: spec, target: target}
}

func (l *kftlStartRepeatStatementLine) ApplyThisLineToRequestMap(_ context.Context, requestMap *KFTLRequestMap) error {
	req := l.target
	if req == nil {
		// **付け先が無ければここで弾く。** タグ行のようにプロトタイプを作ってはいけない ――
		// 作ると「繰り返しの対象が無い」まま展開まで進み、何も作らずに黙って終わる。
		found, ok := requestMap.Get(l.ctx.ThisStatementLineTargetID)
		if !ok {
			return newKFTLInputError("KFTL_REPEAT_NO_TARGET_MESSAGE_TITLE",
				fmt.Errorf("no record to repeat at this position"))
		}
		req = found
	}
	// 繰り返しても意味が無い型（打刻開始のみ・打刻終了・プロトタイプ）はここで断る
	return req.SetRepeatSpec(l.spec)
}

func (l *kftlStartRepeatStatementLine) GetLabelName() string                  { return "repeat" }
func (l *kftlStartRepeatStatementLine) GetContext() *KFTLStatementLineContext { return l.ctx }
func (l *kftlStartRepeatStatementLine) GetStatementLineText() string          { return l.lineText }

// ─── ブロックの中の次の行 ─────────────────────────────────────────────────────

// generateRepeatBlockNextConstructor は「？？」ブロックの中の次の行を決める。
// **閉じる行を最優先で見る**ので、3行目・4行目を省いて早く閉じられる。
func generateRepeatBlockNextConstructor(nextLineText string, spec *repeatSpec, index int, prevLineIsMetaInfo bool, resume resumeConstructorFunc) StatementLineConstructorFunc {
	if isRepeatSplitter(nextLineText) {
		return func(lineText string, ctx *KFTLStatementLineContext) KFTLStatementLine {
			return newKFTLEndRepeatStatementLine(lineText, ctx, spec, prevLineIsMetaInfo, resume)
		}
	}
	return func(lineText string, ctx *KFTLStatementLineContext) KFTLStatementLine {
		return newKFTLRepeatFieldStatementLine(lineText, ctx, spec, index, prevLineIsMetaInfo, resume)
	}
}

// ─── 項目行 ───────────────────────────────────────────────────────────────────

// kftlRepeatFieldStatementLine は「？？」ブロックの中の1行。位置で意味が決まる。
type kftlRepeatFieldStatementLine struct {
	lineText string
	ctx      *KFTLStatementLineContext
	spec     *repeatSpec
	index    int
}

func newKFTLRepeatFieldStatementLine(lineText string, ctx *KFTLStatementLineContext, spec *repeatSpec, index int, prevLineIsMetaInfo bool, resume resumeConstructorFunc) *kftlRepeatFieldStatementLine {
	ctx.NextIsPrototype = ctx.ThisIsPrototype
	targetID := ctx.ThisStatementLineTargetID
	ctx.NextStatementLineTargetID = &targetID
	ctx.NextStatementLineConstructor = generateRepeatBlockNextConstructor(
		ctx.NextStatementLineText, spec, index+1, prevLineIsMetaInfo, resume)
	return &kftlRepeatFieldStatementLine{lineText: lineText, ctx: ctx, spec: spec, index: index}
}

func (l *kftlRepeatFieldStatementLine) ApplyThisLineToRequestMap(_ context.Context, _ *KFTLRequestMap) error {
	switch l.index {
	case repeatFieldCondition:
		cond, err := parseRepeatCondition(l.lineText)
		if err != nil {
			return err
		}
		l.spec.cond = cond
	case repeatFieldCount:
		count, until, err := parseRepeatCountOrUntil(l.lineText, l.ctx.BaseTime)
		if err != nil {
			return err
		}
		l.spec.count, l.spec.until = count, until
	case repeatFieldAddIfExists:
		addIfExists, err := parseRepeatAddIfExists(l.lineText)
		if err != nil {
			return err
		}
		l.spec.addIfExists = addIfExists
	case repeatFieldOrigin:
		origin, err := parseRepeatOrigin(l.lineText, l.ctx.BaseTime)
		if err != nil {
			return err
		}
		l.spec.origin = origin
	default:
		// 4行を書き終えたあとの位置。**空行は見逃す** ―― クライアントはここに「**********」の
		// 行ラベルを出すので、エラーにすると表示と食い違う（閉じる前に行を空けられるようにする）。
		// テキストは飲み込まない。飲み込むと、閉じ忘れたときに本文が繰り返し指定として読まれる
		// （MiReKyou の「閉じたあとはタグ行しか来られない」と同じ扱い）
		if l.lineText == "" || l.lineText == "\n" {
			return nil
		}
		return newKFTLInputError("KFTL_REPEAT_NOT_CLOSED_MESSAGE_TITLE",
			fmt.Errorf("a repeat block takes at most 4 lines and must be closed"))
	}
	return nil
}

func (l *kftlRepeatFieldStatementLine) GetLabelName() string {
	switch l.index {
	case repeatFieldCondition:
		return "repeatCondition"
	case repeatFieldCount:
		return "repeatCount"
	case repeatFieldAddIfExists:
		return "repeatAddIfExists"
	case repeatFieldOrigin:
		return "repeatOrigin"
	}
	// 4行より後ろは受け皿。クライアントは「**********」を出す（TS の get_label_name と対）
	return "none"
}
func (l *kftlRepeatFieldStatementLine) GetContext() *KFTLStatementLineContext { return l.ctx }
func (l *kftlRepeatFieldStatementLine) GetStatementLineText() string          { return l.lineText }

// ─── 終了行 ───────────────────────────────────────────────────────────────────

// kftlEndRepeatStatementLine handles the "？？" that closes a repeat block.
type kftlEndRepeatStatementLine struct {
	lineText string
	ctx      *KFTLStatementLineContext
	spec     *repeatSpec
}

func newKFTLEndRepeatStatementLine(lineText string, ctx *KFTLStatementLineContext, spec *repeatSpec, prevLineIsMetaInfo bool, resume resumeConstructorFunc) *kftlEndRepeatStatementLine {
	ctx.NextIsPrototype = ctx.ThisIsPrototype
	targetID := ctx.ThisStatementLineTargetID
	ctx.NextStatementLineTargetID = &targetID
	// テキストブロックの終了行と同じ戻り方。ブロックの中なら resume、外なら Kmemo / None
	ctx.NextStatementLineConstructor = afterMetaInfoConstructor(ctx.factory, ctx.NextStatementLineText, prevLineIsMetaInfo, resume)
	return &kftlEndRepeatStatementLine{lineText: lineText, ctx: ctx, spec: spec}
}

func (l *kftlEndRepeatStatementLine) ApplyThisLineToRequestMap(_ context.Context, _ *KFTLRequestMap) error {
	// 必須2行の検査。展開の側でも同じ検査をするが（閉じ忘れたブロックはここへ来ない）、
	// 閉じてあるなら**この行の行番号で**返せるほうが直しやすい
	return validateRepeatSpec(l.spec)
}

func (l *kftlEndRepeatStatementLine) GetLabelName() string                  { return "endRepeat" }
func (l *kftlEndRepeatStatementLine) GetContext() *KFTLStatementLineContext { return l.ctx }
func (l *kftlEndRepeatStatementLine) GetStatementLineText() string          { return l.lineText }

// validateRepeatSpec は必須2行が埋まっているかを見る。
func validateRepeatSpec(spec *repeatSpec) error {
	if spec.cond == nil {
		return newKFTLInputError("KFTL_REPEAT_CONDITION_REQUIRED_MESSAGE_TITLE",
			fmt.Errorf("repeat needs a condition on its first line"))
	}
	if spec.count == 0 && spec.until == nil {
		return newKFTLInputError("KFTL_REPEAT_COUNT_REQUIRED_MESSAGE_TITLE",
			fmt.Errorf("repeat needs a count or an end date on its second line"))
	}
	return nil
}

// ─── 展開 ─────────────────────────────────────────────────────────────────────

// expandRepeats は繰り返し指定を持つリクエストを、候補日時のぶんだけ複製する。
//
// **実行ループの直前で呼ぶこと。行の解釈のフェーズでやってはいけない。**
// クライアント側は本文が変わるたびに全行を解釈し直すので、そこで複製すると
// 打鍵1回あたり最大 repeatMaxRecords 件を作ることになる。
// ここでやれば「？？」をブロックのどこに書いても結果が同じになる（行順に依存しない）。
//
// グループは**同じ spec ポインタを共有しているか**で決まる。支出ブロックは
// 全支払いが同じポインタを見るので、ブロックまるごとが1グループになる。
func expandRepeats(ctx context.Context, requestMap *KFTLRequestMap, base time.Time) error {
	type repeatGroup struct {
		spec    *repeatSpec
		members []KFTLRequest
	}
	// slot はグループを「最初の1件が居た位置」に置くための入れ物。挿入順を保つ
	type slot struct {
		req   KFTLRequest
		group *repeatGroup
	}

	var slots []slot
	groups := map[*repeatSpec]*repeatGroup{}
	hasRepeat := false
	for _, req := range requestMap.All() {
		spec := req.GetRepeatSpec()
		if spec == nil {
			slots = append(slots, slot{req: req})
			continue
		}
		hasRepeat = true
		group, ok := groups[spec]
		if !ok {
			group = &repeatGroup{spec: spec}
			groups[spec] = group
			slots = append(slots, slot{group: group})
		}
		group.members = append(group.members, req)
	}
	if !hasRepeat {
		return nil
	}

	var expanded []KFTLRequest
	for _, s := range slots {
		if s.group == nil {
			expanded = append(expanded, s.req)
			continue
		}
		out, err := expandRepeatGroup(ctx, s.group.spec, s.group.members, base)
		if err != nil {
			// 展開は実行ループの直前なので、行の情報はここでしか付けられない
			return withLine(err, s.group.spec.lineIndex+1, s.group.spec.lineText)
		}
		expanded = append(expanded, out...)
	}
	if len(expanded) > repeatMaxRecords {
		return newKFTLInputError("KFTL_REPEAT_LIMIT_EXCEEDED_MESSAGE_TITLE",
			fmt.Errorf("this submission would create %d records (max %d)", len(expanded), repeatMaxRecords))
	}
	requestMap.ReplaceAll(expanded)
	return nil
}

// expandRepeatGroup は1グループを候補日時のぶんだけ複製する。
func expandRepeatGroup(ctx context.Context, spec *repeatSpec, members []KFTLRequest, base time.Time) ([]KFTLRequest, error) {
	if err := validateRepeatSpec(spec); err != nil {
		return nil, err // 閉じ忘れたブロックはここで捕まる
	}
	// アンカーはグループの先頭から取る。支出ブロックは関連時刻をブロックで共有するので、
	// どの支払いから取っても同じ
	anchor, ok := members[0].AnchorTimeForRepeat()
	if !ok {
		return nil, newKFTLInputError("KFTL_REPEAT_NO_DATE_FIELD_MESSAGE_TITLE",
			fmt.Errorf("the record has no datetime field, so there is nowhere to put the repeated dates"))
	}
	occurrences, err := occurrencesOf(spec, anchor, base)
	if err != nil {
		return nil, err
	}
	if len(occurrences) == 0 {
		return nil, newKFTLInputError("KFTL_REPEAT_NO_OCCURRENCE_MESSAGE_TITLE",
			fmt.Errorf("no date matches the repeat condition"))
	}

	// 既存があればその回を飛ばす（3行目の既定が no なので、既定でこちらを通る）。
	//
	// **メンバーごとに引く。** 支出ブロックは支払いごとに品名が違うので、
	// グループでまとめて判定すると「片方だけ既にある」ときに残りも作られなくなる。
	// 引くのはメンバー数ぶんだけで、候補の回数ぶんは引かない。
	existingByMember := make([]map[int64]struct{}, len(members))
	if !spec.addIfExists {
		from := occurrences[0].AddDate(0, 0, -1)
		to := occurrences[len(occurrences)-1].AddDate(0, 0, 1)
		for i, member := range members {
			found, findErr := member.FindExistingForRepeat(ctx, from, to)
			if findErr != nil {
				return nil, findErr
			}
			existingByMember[i] = found
		}
	}

	out := make([]KFTLRequest, 0, len(occurrences)*len(members))
	// 元のIDを引き継ぐのは「実際に作る最初の1件」。
	// 既存で先頭の回が飛ばされても、作られた1件目が引き継ぐ
	idTaken := make([]bool, len(members))
	for _, occurrence := range occurrences {
		// 全日時欄を同じ日数だけずらす。欄どうしの相対差は保たれる
		dayShift := daysBetween(anchor, occurrence)
		for i, member := range members {
			if _, duplicated := existingByMember[i][occurrence.Unix()]; duplicated {
				continue
			}
			id := sqlite3impl.GenerateNewID()
			if !idTaken[i] {
				id = member.GetRequestID()
				idTaken[i] = true
			}
			out = append(out, member.CloneForRepeat(id, dayShift))
		}
	}
	return out, nil
}
