package api

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
)

// stubKyouSearchRep は findKyous の rep名フィルタだけを見るためのスタブ。
// 埋め込みが nil なので、テストで呼ばないメソッドは実装しなくてよい。
// findKyous は検索対象repの選び方には関与しないので、MatchReps へ直接入れて使う。
type stubKyouSearchRep struct {
	reps.Repository
	byQuery func(query *find.FindQuery) map[string][]reps.Kyou
}

func (s *stubKyouSearchRep) FindKyous(_ context.Context, query *find.FindQuery) (map[string][]reps.Kyou, error) {
	return s.byQuery(query), nil
}

func (s *stubKyouSearchRep) GetRepName(_ context.Context) (string, error) {
	return "stub-search-rep", nil
}

func (s *stubKyouSearchRep) UnWrap() ([]reps.Repository, error) {
	return []reps.Repository{s}, nil
}

func kyouInRep(id string, repName string, updateTime time.Time) reps.Kyou {
	return reps.Kyou{ID: id, RepName: repName, DataType: "kmemo", RelatedTime: updateTime, UpdateTime: updateTime}
}

// rep名での絞り込みは**検索対象repではなく検索結果**(Kyou.RepName)で行う。
// 検索対象repのほうで絞ろうとするとキャッシュrepを UnWrap() することになり、
// 生のディスクrepへ戻ってキャッシュを丸ごとバイパスする
// (理由は selectMatchRepsFromQuery のコメント。実データでgitだけで20.7秒/窓)。
//
// 落とし穴が3つあるので全部固定する。
//   - textヒット由来の2本目の検索(matchTextFindByIDQuery)にも同じ絞り込みが要る
//   - 全部落ちたIDはキーごと消す(空スライスを残すと kyous[0] を見る後段が panic する)
//   - Reps == nil は「未指定」。len() で判定すると全件消える
func TestFindKyous_FiltersResultsByRepName(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 8, 19, 12, 0, 0, 0, time.Local)

	// 本番の全経路で OnlyLatestData は true 固定(usecase/kyou.go)。
	// false にすると Repositories.findKyous のマージキーが id+UpdateTime になり、
	// このテストが見たいものとずれる
	newFindCtx := func(query *find.FindQuery, rep reps.Repository, matchTexts map[string]reps.Text) *FindKyouContext {
		return &FindKyouContext{
			ParsedFindQuery: query,
			MatchReps:       map[string]reps.Repository{"stub-search-rep": rep},
			MatchTexts:      matchTexts,
		}
	}
	resultIDs := func(findCtx *FindKyouContext) []string {
		ids := []string{}
		for id := range findCtx.MatchKyousCurrent {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		return ids
	}

	t.Run("指定したrepの記録だけが残る", func(t *testing.T) {
		rep := &stubKyouSearchRep{byQuery: func(_ *find.FindQuery) map[string][]reps.Kyou {
			return map[string][]reps.Kyou{
				"in":  {kyouInRep("in", "rep-a", now)},
				"out": {kyouInRep("out", "rep-b", now)},
			}
		}}
		findCtx := newFindCtx(&find.FindQuery{Reps: []string{"rep-a"}, OnlyLatestData: true}, rep, nil)

		f := &FindFilter{}
		if _, err := f.findKyous(ctx, findCtx); err != nil {
			t.Fatalf("findKyous failed: %v", err)
		}

		if got := resultIDs(findCtx); len(got) != 1 || got[0] != "in" {
			t.Errorf("指定したrepの記録だけが残るはず: %v", got)
		}
	})

	t.Run("全部落ちたIDはキーごと消える", func(t *testing.T) {
		rep := &stubKyouSearchRep{byQuery: func(_ *find.FindQuery) map[string][]reps.Kyou {
			return map[string][]reps.Kyou{"out": {kyouInRep("out", "rep-b", now)}}
		}}
		findCtx := newFindCtx(&find.FindQuery{Reps: []string{"rep-a"}, OnlyLatestData: true}, rep, nil)

		f := &FindFilter{}
		if _, err := f.findKyous(ctx, findCtx); err != nil {
			t.Fatalf("findKyous failed: %v", err)
		}

		if kyous, exist := findCtx.MatchKyousCurrent["out"]; exist {
			t.Errorf("空になったIDのキーが残っている(後段が kyous[0] で panic する): len=%d", len(kyous))
		}
	})

	t.Run("Repsがnilなら何も落とさない", func(t *testing.T) {
		rep := &stubKyouSearchRep{byQuery: func(_ *find.FindQuery) map[string][]reps.Kyou {
			return map[string][]reps.Kyou{
				"a": {kyouInRep("a", "rep-a", now)},
				"b": {kyouInRep("b", "rep-b", now)},
			}
		}}
		findCtx := newFindCtx(&find.FindQuery{Reps: nil, OnlyLatestData: true}, rep, nil)

		f := &FindFilter{}
		if _, err := f.findKyous(ctx, findCtx); err != nil {
			t.Fatalf("findKyous failed: %v", err)
		}

		if got := resultIDs(findCtx); len(got) != 2 {
			t.Errorf("Reps未指定では絞り込まないはず: %v", got)
		}
	})

	t.Run("RepNameが空の行は残す", func(t *testing.T) {
		// キャッシュrepへの write-through は呼び出し側の値をそのまま INSERT するので、
		// 追加直後の行は REP_NAME が空のまま(実rep名が入るのは次の UpdateCache)。
		// ここで落とすと**いま追加した記録が最大1分間一覧から消える**。
		rep := &stubKyouSearchRep{byQuery: func(_ *find.FindQuery) map[string][]reps.Kyou {
			return map[string][]reps.Kyou{"just-added": {kyouInRep("just-added", "", now)}}
		}}
		findCtx := newFindCtx(&find.FindQuery{Reps: []string{"rep-a"}, OnlyLatestData: true}, rep, nil)

		f := &FindFilter{}
		if _, err := f.findKyous(ctx, findCtx); err != nil {
			t.Fatalf("findKyous failed: %v", err)
		}

		if got := resultIDs(findCtx); len(got) != 1 {
			t.Errorf("RepNameが空の行を落としている(追加直後の記録が消える): %v", got)
		}
	})

	t.Run("textヒット由来のID検索にも効く", func(t *testing.T) {
		// 1本目は0件、2本目(IDs指定)だけがチェックしていないrepの行を返す
		rep := &stubKyouSearchRep{byQuery: func(query *find.FindQuery) map[string][]reps.Kyou {
			if query.IDs == nil {
				return map[string][]reps.Kyou{}
			}
			return map[string][]reps.Kyou{"text-hit": {kyouInRep("text-hit", "rep-b", now)}}
		}}
		matchTexts := map[string]reps.Text{"text-1": {ID: "text-1", TargetID: "text-hit"}}
		findCtx := newFindCtx(&find.FindQuery{Reps: []string{"rep-a"}, OnlyLatestData: true}, rep, matchTexts)

		f := &FindFilter{}
		if _, err := f.findKyous(ctx, findCtx); err != nil {
			t.Fatalf("findKyous failed: %v", err)
		}

		if got := resultIDs(findCtx); len(got) != 0 {
			t.Errorf("2本目の検索にも絞り込みが要る: %v", got)
		}
	})
}
