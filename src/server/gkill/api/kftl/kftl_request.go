package kftl

import (
	"context"
	"fmt"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
	"github.com/mt3hr/gkill/src/server/gkill/dao/sqlite3impl"
)

// KFTLRequest is the interface implemented by all KFTL request types.
// Mirrors: src/classes/kftl/kftl-request.ts (abstract class)
type KFTLRequest interface {
	DoRequest(ctx context.Context) error
	// ValidateContent は「書く前に分かる、内容の欠け」を入力エラーで返す
	// （本文の無いメモ・タイトルの無い打刻やタスク・値の行を通っていない気分・付け先の無いプロトタイプ）。
	// prepareRequests が全行を適用した後に呼ぶので、Analyze（ピンク）と送信の両方で同じ結果になる。
	// **基底に既定実装を置かない** —— 置くと新しい型で書き忘れても通り、空の記録が黙って書かれるか
	// 黙って0件になる。DoRequest はこれをもう一度呼んで、書き込みフェーズまで来た経路でも書かない（ADR-0508）。
	ValidateContent() error
	GetRequestID() string
	GetTags() []string
	GetTextsMap() map[string]string
	GetRelatedTime() time.Time
	SetRelatedTime(t time.Time)
	SetRelatedTimePtr(t *time.Time)
	AddTag(tag string)
	AddTextLine(textID, line string)
	GetCurrentTextID() *string
	SetCurrentTextID(textID *string)
	GetContext() *KFTLStatementLineContext
	// GetCreatedRecords は DoRequest が実際に書いたものを返す。
	// 実装は KFTLRequestBase に1本だけあり、全具象がそれを埋め込んでいる。
	GetCreatedRecords() []KFTLCreatedRecord

	// ─── 繰り返し（「？？」ブロック）────────────────────────────────────────
	GetRepeatSpec() *repeatSpec
	SetRepeatSpec(spec *repeatSpec) error
	AnchorTimeForRepeat() (time.Time, bool)
	// CloneForRepeat は繰り返しの1回ぶんを作る。**基底に既定実装を置かない** ――
	// 置くと新しい型で override を忘れても通ってしまい、日時のずれない複製が黙って書かれる。
	// 型を足したらコンパイルエラーで気づけるようにしてある。
	CloneForRepeat(newRequestID string, dayShift int) KFTLRequest
	// FindExistingForRepeat は「もう同じ記録があるか」を調べる（3行目が no のときだけ呼ばれる）。
	// 実装は kftl_repeat_duplicate.go にまとめてある。
	FindExistingForRepeat(ctx context.Context, from, to time.Time) (map[int64]struct{}, error)
}

// KFTLRequestBase is the base struct embedded by all concrete request types.
// KFTLCreatedRecord は KFTL が実際に書いた1件。
//
// KFTL は1つのテキストから複数のKyouを作るのに、応答は「記録しました」の1文だけで、
// 件数も種別もIDも返していなかった（2巡目の指摘）。IDそのものは
// リクエストIDと同じ値で最初から手元にあったが、**支払いの後ろの空行が作る空の Nlog は
// 何も書かずに成功し、打刻の終了は既存レコードの更新**なので、リクエストを
// 事前に並べるだけでは「作られたもの」にならない。書いた側が控える
// （本文が空の kmemo / Mi / 打刻は 2026-09-15 から書く前に行別エラーになる。ADR-0508）。
type KFTLCreatedRecord struct {
	ID       string
	DataType string
	// Updated は新規作成ではなく既存レコードの更新であることを表す（打刻の終了）。
	Updated bool
	// RelatedTime は書いた記録の関連時刻（打刻の終了は終了時刻）。Web が「、、」でずらした分を
	// 実行中画面へ渡す（saved_kyou_by_kftl）ために、引き直しを待たず応答から取れるようにする。
	RelatedTime time.Time
}

// Mirrors: src/classes/kftl/kftl-request.ts
type KFTLRequestBase struct {
	RequestID     string
	Tags          []string
	TextsMap      map[string]string // textID → accumulated text content
	CurrentTextID *string
	relatedTime   *time.Time // nil means use addSecond offset
	Ctx           *KFTLStatementLineContext
	CreateTime    time.Time

	// created は DoRequest が実際に書いたもの。recordCreated / recordUpdated だけが積む。
	created []KFTLCreatedRecord

	// repeat は「？？」ブロックの指定。同じポインタを共有しているリクエストが
	// 1つの繰り返しグループになる（支出ブロックの全支払いなど）。
	repeat *repeatSpec
}

// recordCreated は新規作成した1件を控える。**書き込みが成功した直後にだけ呼ぶこと。**
// 関連時刻は呼び出し側が引数で渡す（基底の中で b.GetRelatedTime() を引くと支出ブロックの override が効かない）。
func (b *KFTLRequestBase) recordCreated(dataType, id string, relatedTime time.Time) {
	b.created = append(b.created, KFTLCreatedRecord{ID: id, DataType: dataType, RelatedTime: relatedTime})
}

// recordUpdated は既存レコードを更新した1件を控える（打刻の終了）。
func (b *KFTLRequestBase) recordUpdated(dataType, id string, relatedTime time.Time) {
	b.created = append(b.created, KFTLCreatedRecord{ID: id, DataType: dataType, Updated: true, RelatedTime: relatedTime})
}

// GetCreatedRecords は DoRequest が実際に書いたものを返す。
func (b *KFTLRequestBase) GetCreatedRecords() []KFTLCreatedRecord { return b.created }

func (b *KFTLRequestBase) GetRequestID() string                  { return b.RequestID }
func (b *KFTLRequestBase) GetTags() []string                     { return b.Tags }
func (b *KFTLRequestBase) GetTextsMap() map[string]string        { return b.TextsMap }
func (b *KFTLRequestBase) GetCurrentTextID() *string             { return b.CurrentTextID }
func (b *KFTLRequestBase) SetCurrentTextID(textID *string)       { b.CurrentTextID = textID }
func (b *KFTLRequestBase) GetContext() *KFTLStatementLineContext { return b.Ctx }

// GetRelatedTime returns the effective related time.
// If not explicitly set, returns now + addSecond offset.
// Mirrors: KFTLRequest.get_related_time()
func (b *KFTLRequestBase) GetRelatedTime() time.Time {
	if b.relatedTime != nil {
		return *b.relatedTime
	}
	return time.Now().Add(time.Duration(b.Ctx.AddSecond) * time.Second)
}

func (b *KFTLRequestBase) SetRelatedTime(t time.Time)     { b.relatedTime = &t }
func (b *KFTLRequestBase) SetRelatedTimePtr(t *time.Time) { b.relatedTime = t }

func (b *KFTLRequestBase) AddTag(tag string) {
	b.Tags = append(b.Tags, tag)
}

func (b *KFTLRequestBase) AddTextLine(textID, line string) {
	if b.TextsMap == nil {
		b.TextsMap = make(map[string]string)
	}
	existing, ok := b.TextsMap[textID]
	if !ok || existing == "" {
		b.TextsMap[textID] = line
	} else {
		b.TextsMap[textID] = existing + "\n" + line
	}
}

// ─── 繰り返し（「？？」ブロック）──────────────────────────────────────────────

func (b *KFTLRequestBase) GetRepeatSpec() *repeatSpec { return b.repeat }

// SetRepeatSpec は既定で受け入れる。繰り返しても意味が無い型
// （打刻開始のみ・打刻終了の4種・プロトタイプ）だけが override して弾く。
func (b *KFTLRequestBase) SetRepeatSpec(spec *repeatSpec) error {
	b.repeat = spec
	return nil
}

// AnchorTimeForRepeat は繰り返しの基準になる日時を返す。
//
// **呼ぶと確定させる。** related_time が未設定なら CreateTime を書き込む。
// 確定させないと GetRelatedTime() が time.Now() へ落ちるので、
// 複製した側が DoRequest の時点でそれぞれ現在時刻を引き、全回が同じ日時になる。
//
// 日時欄を複数持つ型（Mi / MiReKyou）と、開始時刻が主軸の型（TimeIs）は override する。
func (b *KFTLRequestBase) AnchorTimeForRepeat() (time.Time, bool) {
	if b.relatedTime == nil {
		t := b.CreateTime
		b.relatedTime = &t
	}
	return *b.relatedTime, true
}

// cloneBase は繰り返し複製の土台。**ここで採り直すものを1つでも落とすと静かに壊れる。**
func (b *KFTLRequestBase) cloneBase(newRequestID string, dayShift int) KFTLRequestBase {
	c := KFTLRequestBase{
		RequestID:  newRequestID,
		Ctx:        b.Ctx,
		CreateTime: b.CreateTime,
		// 複製に繰り返し指定を持たせない。持たせると展開が再帰する
		repeat: nil,
		// created は書いた側が積むので空で始める
	}
	// スライスとマップは実体を作る。シャローのままだとバッキングを共有する
	if b.Tags != nil {
		c.Tags = append([]string(nil), b.Tags...)
	}
	if b.TextsMap != nil {
		c.TextsMap = make(map[string]string, len(b.TextsMap))
		for _, text := range b.TextsMap {
			// **textID を採り直す。** 使い回すと同じIDのテキストを回数ぶん書くことになり、
			// append-only なので最後の1件以外が消える
			c.TextsMap[sqlite3impl.GenerateNewID()] = text
		}
	}
	c.relatedTime = shiftTimePtr(b.relatedTime, dayShift)
	return c
}

// shiftTimePtr / shiftTime は暦日数でずらす。
// duration 加算にしないのは壁時計時刻を保つため（夏時間のある地域で時刻がずれる）。
func shiftTimePtr(t *time.Time, dayShift int) *time.Time {
	if t == nil {
		return nil
	}
	shifted := shiftTime(*t, dayShift)
	return &shifted
}

func shiftTime(t time.Time, dayShift int) time.Time {
	if dayShift == 0 {
		return t
	}
	return t.AddDate(0, 0, dayShift)
}

// daysBetween は暦日の差。truncate を UTC で行うので夏時間の影響を受けない。
func daysBetween(from, to time.Time) int {
	a := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC)
	b := time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, time.UTC)
	return int(b.Sub(a).Hours() / 24)
}

// doBaseRequest adds tags and texts for the given targetID.
// Mirrors: KFTLRequest.do_request() in TS (the tag/text portion).
//
// relatedTime は呼ぶ側（外側の型）が `r.GetRelatedTime()` で渡す。
// **埋め込み基底のメソッドは外側の override を見ない**（Go は埋め込みで仮想ディスパッチしない）ので、
// ここで `b.GetRelatedTime()` を引くと kftlNlogRequest がブロック共有の時刻を返す override が効かず、
// 支出（ーん）の `？`行の時刻がタグ・テキストに乗らないまま「今」で書かれていた
// （TS は `super.do_request` の中の `this.get_related_time()` が override に届くので揃っていなかった）。
func (b *KFTLRequestBase) doBaseRequest(ctx context.Context, targetID string, relatedTime time.Time) error {
	now := b.CreateTime

	// Add tags
	for _, tag := range b.Tags {
		tagObj := reps.Tag{
			ID:           sqlite3impl.GenerateNewID(),
			TargetID:     targetID,
			Tag:          tag,
			RelatedTime:  relatedTime,
			CreateTime:   now,
			CreateApp:    b.Ctx.ApplicationName,
			CreateDevice: b.Ctx.Device,
			CreateUser:   b.Ctx.UserID,
			UpdateTime:   now,
			UpdateApp:    b.Ctx.ApplicationName,
			UpdateDevice: b.Ctx.Device,
			UpdateUser:   b.Ctx.UserID,
		}
		err := b.Ctx.Repositories.TempReps.TagTempRep.AddTagInfo(ctx, tagObj, b.Ctx.TXID, b.Ctx.UserID, b.Ctx.Device)
		if err != nil {
			return fmt.Errorf("error at add tag info target_id=%s tag=%s: %w", targetID, tag, err)
		}
	}

	// Add texts
	for textID, textContent := range b.TextsMap {
		if textContent == "" {
			continue
		}
		textObj := reps.Text{
			ID:           textID,
			TargetID:     targetID,
			Text:         textContent,
			RelatedTime:  relatedTime,
			CreateTime:   now,
			CreateApp:    b.Ctx.ApplicationName,
			CreateDevice: b.Ctx.Device,
			CreateUser:   b.Ctx.UserID,
			UpdateTime:   now,
			UpdateApp:    b.Ctx.ApplicationName,
			UpdateDevice: b.Ctx.Device,
			UpdateUser:   b.Ctx.UserID,
		}
		err := b.Ctx.Repositories.TempReps.TextTempRep.AddTextInfo(ctx, textObj, b.Ctx.TXID, b.Ctx.UserID, b.Ctx.Device)
		if err != nil {
			return fmt.Errorf("error at add text info target_id=%s text_id=%s: %w", targetID, textID, err)
		}
	}

	return nil
}
