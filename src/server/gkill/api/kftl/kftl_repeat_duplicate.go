package kftl

import (
	"context"
	"fmt"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
)

// 繰り返しブロック「？？」の3行目が no（既定）のときの既存判定。
//
// **型ごとの「同じ記録」の定義をここに集める。** 複製（CloneForRepeat）は
// フィールドの隣に置いてあるが、こちらは型をまたいだ抜けのほうが問題になる
// ―― 1つ実装し忘れると、その型だけ既定の冪等性が黙って効かなくなる。
//
// 返すのは「既にある記録のアンカー時刻」の集合（Unix秒）で、展開はその時刻の回を飛ばす。
// 絞るのは候補全体を覆う期間だけで、同一性の判定はメモリ上で行う
// （型ごとに定義が違うので SQL 側では表現しない）。**回数ぶん検索はしない。**
//
// Mirrors: src/client/classes/kftl/kftl_repeat/kftl-repeat-duplicate.ts

// newRepeatDuplicateQuery は related_time が主軸の型のための検索条件。
func newRepeatDuplicateQuery(from, to time.Time) *find.FindQuery {
	return &find.FindQuery{
		CalendarStartDate: &from,
		CalendarEndDate:   &to,
		OnlyLatestData:    true,
	}
}

// newRepeatDuplicateScheduleQuery は Mi / MiReKyou のための検索条件。
// 予定日時のどれかが期間に入るものを拾うので、**射影を全部有効にする**
// （1つでも落とすと、その日時軸を持つ記録を取りこぼして重複を作る）。
func newRepeatDuplicateScheduleQuery(from, to time.Time) *find.FindQuery {
	query := newRepeatDuplicateQuery(from, to)
	query.ForMi = true
	query.IncludeCreateMi = true
	query.IncludeCheckMi = true
	query.IncludeLimitMi = true
	query.IncludeStartMi = true
	query.IncludeEndMi = true
	return query
}

// scheduleAnchorOf は Mi / MiReKyou のアンカー（最初に埋まっている予定日時）。
// **AnchorTimeForRepeat と同じ順序で見ること。** ずれると既存判定が噛み合わなくなる。
func scheduleAnchorOf(estimateStart, estimateEnd, limit *time.Time) (time.Time, bool) {
	for _, t := range []*time.Time{estimateStart, estimateEnd, limit} {
		if t != nil {
			return *t, true
		}
	}
	return time.Time{}, false
}

// repositoriesOf は検索に使うリポジトリ。テストのように未設定なら nil を返し、
// 呼び出し側は「既存なし」として扱う。
func repositoriesOf(base *KFTLRequestBase) bool {
	return base.Ctx != nil && base.Ctx.Repositories != nil
}

// ─── related_time が主軸の型 ─────────────────────────────────────────────────

func (r *kftlKmemoRequest) FindExistingForRepeat(ctx context.Context, from, to time.Time) (map[int64]struct{}, error) {
	if !repositoriesOf(&r.KFTLRequestBase) {
		return nil, nil
	}
	found, err := r.Ctx.Repositories.KmemoReps.FindKmemo(ctx, newRepeatDuplicateQuery(from, to))
	if err != nil {
		return nil, fmt.Errorf("error at find kmemo for repeat: %w", err)
	}
	content := joinLines(r.contentLines)
	existing := map[int64]struct{}{}
	for _, kmemo := range found {
		if kmemo.IsDeleted || kmemo.Content != content {
			continue
		}
		existing[kmemo.RelatedTime.Unix()] = struct{}{}
	}
	return existing, nil
}

func (r *kftlKCRequest) FindExistingForRepeat(ctx context.Context, from, to time.Time) (map[int64]struct{}, error) {
	if !repositoriesOf(&r.KFTLRequestBase) {
		return nil, nil
	}
	found, err := r.Ctx.Repositories.KCReps.FindKC(ctx, newRepeatDuplicateQuery(from, to))
	if err != nil {
		return nil, fmt.Errorf("error at find kc for repeat: %w", err)
	}
	existing := map[int64]struct{}{}
	for _, kc := range found {
		if kc.IsDeleted || kc.Title != r.title {
			continue
		}
		existing[kc.RelatedTime.Unix()] = struct{}{}
	}
	return existing, nil
}

// 気分値にはタイトルが無いので、同じ時刻に気分の記録があれば「既にある」とみなす。
// 値そのものは見ない（同じ時刻に別の気分値を2つ置く使い方は想定しない）。
func (r *kftlLantanaRequest) FindExistingForRepeat(ctx context.Context, from, to time.Time) (map[int64]struct{}, error) {
	if !repositoriesOf(&r.KFTLRequestBase) {
		return nil, nil
	}
	found, err := r.Ctx.Repositories.LantanaReps.FindLantana(ctx, newRepeatDuplicateQuery(from, to))
	if err != nil {
		return nil, fmt.Errorf("error at find lantana for repeat: %w", err)
	}
	existing := map[int64]struct{}{}
	for _, lantana := range found {
		if lantana.IsDeleted {
			continue
		}
		existing[lantana.RelatedTime.Unix()] = struct{}{}
	}
	return existing, nil
}

func (r *kftlURLogRequest) FindExistingForRepeat(ctx context.Context, from, to time.Time) (map[int64]struct{}, error) {
	if !repositoriesOf(&r.KFTLRequestBase) {
		return nil, nil
	}
	found, err := r.Ctx.Repositories.URLogReps.FindURLog(ctx, newRepeatDuplicateQuery(from, to))
	if err != nil {
		return nil, fmt.Errorf("error at find urlog for repeat: %w", err)
	}
	existing := map[int64]struct{}{}
	for _, urlog := range found {
		if urlog.IsDeleted || urlog.URL != r.url {
			continue
		}
		existing[urlog.RelatedTime.Unix()] = struct{}{}
	}
	return existing, nil
}

// 支出は支払いごとに見る。**ブロック単位で見てはいけない** ――
// 同じ買い物でも品名が違えば別の記録なので、片方だけ既にある状態がふつうに起きる。
func (r *kftlNlogRequest) FindExistingForRepeat(ctx context.Context, from, to time.Time) (map[int64]struct{}, error) {
	if !repositoriesOf(&r.KFTLRequestBase) {
		return nil, nil
	}
	found, err := r.Ctx.Repositories.NlogReps.FindNlog(ctx, newRepeatDuplicateQuery(from, to))
	if err != nil {
		return nil, fmt.Errorf("error at find nlog for repeat: %w", err)
	}
	existing := map[int64]struct{}{}
	for _, nlog := range found {
		if nlog.IsDeleted || nlog.Title != r.title || nlog.Shop != r.block.shop {
			continue
		}
		existing[nlog.RelatedTime.Unix()] = struct{}{}
	}
	return existing, nil
}

// ─── 開始時刻が主軸の型 ───────────────────────────────────────────────────────

func (r *kftlTimeIsRequest) FindExistingForRepeat(ctx context.Context, from, to time.Time) (map[int64]struct{}, error) {
	if !repositoriesOf(&r.KFTLRequestBase) {
		return nil, nil
	}
	found, err := r.Ctx.Repositories.TimeIsReps.FindTimeIs(ctx, newRepeatDuplicateQuery(from, to))
	if err != nil {
		return nil, fmt.Errorf("error at find timeis for repeat: %w", err)
	}
	existing := map[int64]struct{}{}
	for _, timeIs := range found {
		if timeIs.IsDeleted || timeIs.Title != r.title {
			continue
		}
		existing[timeIs.StartTime.Unix()] = struct{}{}
	}
	return existing, nil
}

// ─── 予定日時が主軸の型 ───────────────────────────────────────────────────────

func (r *kftlMiRequest) FindExistingForRepeat(ctx context.Context, from, to time.Time) (map[int64]struct{}, error) {
	if !repositoriesOf(&r.KFTLRequestBase) {
		return nil, nil
	}
	found, err := r.Ctx.Repositories.MiReps.FindMi(ctx, newRepeatDuplicateScheduleQuery(from, to))
	if err != nil {
		return nil, fmt.Errorf("error at find mi for repeat: %w", err)
	}
	boardName := r.resolvedBoardName()
	existing := map[int64]struct{}{}
	for _, mi := range found {
		if mi.IsDeleted || mi.Title != r.title || mi.BoardName != boardName {
			continue
		}
		if anchor, ok := scheduleAnchorOf(mi.EstimateStartTime, mi.EstimateEndTime, mi.LimitTime); ok {
			existing[anchor.Unix()] = struct{}{}
		}
	}
	return existing, nil
}

// リポストタスクはタイトルを持たないので、対象の記録と板名で見る。
func (r *kftlMiReKyouRequest) FindExistingForRepeat(ctx context.Context, from, to time.Time) (map[int64]struct{}, error) {
	if !repositoriesOf(&r.KFTLRequestBase) {
		return nil, nil
	}
	found, err := r.Ctx.Repositories.MiReKyouReps.FindMiReKyou(ctx, newRepeatDuplicateScheduleQuery(from, to))
	if err != nil {
		return nil, fmt.Errorf("error at find mirekyou for repeat: %w", err)
	}
	boardName := r.resolvedBoardName()
	existing := map[int64]struct{}{}
	for _, mirekyou := range found {
		if mirekyou.IsDeleted || mirekyou.TargetID != r.targetID || mirekyou.BoardName != boardName {
			continue
		}
		if anchor, ok := scheduleAnchorOf(mirekyou.EstimateStartTime, mirekyou.EstimateEndTime, mirekyou.LimitTime); ok {
			existing[anchor.Unix()] = struct{}{}
		}
	}
	return existing, nil
}

// ─── 繰り返せない型 ───────────────────────────────────────────────────────────
//
// SetRepeatSpec が断るのでここへは来ないが、インタフェースの実装は要る
// （基底に既定を置くと、実装を忘れた型が黙って「既存なし」になる）。

func (r *kftlTimeIsStartRequest) FindExistingForRepeat(_ context.Context, _, _ time.Time) (map[int64]struct{}, error) {
	return nil, nil
}

func (r *kftlTimeIsEndByTitleRequest) FindExistingForRepeat(_ context.Context, _, _ time.Time) (map[int64]struct{}, error) {
	return nil, nil
}

func (r *kftlTimeIsEndByTagRequest) FindExistingForRepeat(_ context.Context, _, _ time.Time) (map[int64]struct{}, error) {
	return nil, nil
}

func (r *KFTLPrototypeRequest) FindExistingForRepeat(_ context.Context, _, _ time.Time) (map[int64]struct{}, error) {
	return nil, nil
}
