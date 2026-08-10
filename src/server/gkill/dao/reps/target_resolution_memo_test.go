package reps

// ReKyou/MiReKyou のターゲット解決メモの回帰テスト。
//
// 委譲が入れ子になっているので、同じ query で末端repを何度も舐めても
// 結果は変わらない。だから通常のテストでは退行に気づけない。
// 末端repの FindKyous 呼び出し回数を数え、
// 「回数が減ったこと」と「結果集合が変わらないこと」を一緒に固定する。
// 別々のテストにすると片方だけ通ってしまう。

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	gkill_cache "github.com/mt3hr/gkill/src/server/gkill/dao/reps/cache"
)

// countingRepository は FindKyous の呼び出し回数を数える末端repのスタブ。
type countingRepository struct {
	repName string
	kyouIDs []string

	m     sync.Mutex
	calls int
}

func (c *countingRepository) FindKyous(_ context.Context, _ *find.FindQuery) (map[string][]Kyou, error) {
	c.m.Lock()
	c.calls++
	c.m.Unlock()

	matchKyous := map[string][]Kyou{}
	for _, id := range c.kyouIDs {
		matchKyous[id] = []Kyou{{ID: id, RepName: c.repName}}
	}
	return matchKyous, nil
}

func (c *countingRepository) callCount() int {
	c.m.Lock()
	defer c.m.Unlock()
	return c.calls
}

func (c *countingRepository) GetKyou(_ context.Context, _ string, _ *time.Time) (*Kyou, error) {
	return nil, nil
}
func (c *countingRepository) GetKyouHistories(_ context.Context, _ string) ([]Kyou, error) {
	return nil, nil
}
func (c *countingRepository) GetPath(_ context.Context, _ string) (string, error) { return "", nil }
func (c *countingRepository) GetRepName(_ context.Context) (string, error)        { return c.repName, nil }
func (c *countingRepository) UpdateCache(_ context.Context) error                 { return nil }
func (c *countingRepository) GetLatestDataRepositoryAddress(_ context.Context, _ bool) ([]gkill_cache.LatestDataRepositoryAddress, error) {
	return nil, nil
}
func (c *countingRepository) Close(_ context.Context) error      { return nil }
func (c *countingRepository) UnWrap() ([]Repository, error)      { return []Repository{c}, nil }
func newCountingRep(name string, ids ...string) *countingRepository {
	return &countingRepository{repName: name, kyouIDs: ids}
}

func wordQuery() *find.FindQuery {
	return &find.FindQuery{Words: []string{"word"}}
}

func sortedKeys(ids map[string]bool) []string {
	keys := make([]string, 0, len(ids))
	for id := range ids {
		keys = append(keys, id)
	}
	return keys
}

func assertIDSet(t *testing.T, got map[string]bool, want ...string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("ID集合 = %v, want %v", sortedKeys(got), want)
	}
	for _, id := range want {
		if !got[id] {
			t.Errorf("%s が無い (got=%v)", id, sortedKeys(got))
		}
	}
}

// 同じ scope・同じ query なら末端repは1回しか舐められないこと。
// 結果は毎回同じであること。
func TestTargetResolutionMemoSweepsEachRepOnce(t *testing.T) {
	ctx := WithTargetResolutionMemo(context.Background())
	kmemo := newCountingRep("Kmemo", "kmemo-1")
	urlog := newCountingRep("URLog", "urlog-1")
	dataReps := Repositories{kmemo, urlog}
	query := wordQuery()

	for range 3 {
		ids, err := resolveMiReKyouWordMatchTargetIDs(ctx, dataReps, query)
		if err != nil {
			t.Fatalf("resolveMiReKyouWordMatchTargetIDs failed: %v", err)
		}
		assertIDSet(t, ids, "kmemo-1", "urlog-1")
	}

	for _, rep := range []*countingRepository{kmemo, urlog} {
		if got := rep.callCount(); got != 1 {
			t.Errorf("%s の FindKyous が %d 回呼ばれた。同じqueryなら1回に収まるべき", rep.repName, got)
		}
	}
}

// ReKyouの委譲先は「実データrep + MiReKyou」なので、
// 実データrepぶんはMiReKyou側の解決と共有され、結果は両者の和になること。
func TestReKyouTargetResolutionSharesDataSweepWithMiReKyou(t *testing.T) {
	ctx := WithTargetResolutionMemo(context.Background())
	kmemo := newCountingRep("Kmemo", "kmemo-1")
	miReKyou := newCountingRep("MiReKyou", "mirekyou-1")
	dataReps := Repositories{kmemo}
	miReKyouReps := Repositories{miReKyou}
	query := wordQuery()

	// ReKyou の解決
	reKyouIDs, err := resolveReKyouWordMatchTargetIDs(ctx, dataReps, miReKyouReps, query)
	if err != nil {
		t.Fatalf("resolveReKyouWordMatchTargetIDs failed: %v", err)
	}
	assertIDSet(t, reKyouIDs, "kmemo-1", "mirekyou-1")

	// 入れ子になっているMiReKyou自身の解決（実データrepのみ）
	miReKyouTargetIDs, err := resolveMiReKyouWordMatchTargetIDs(ctx, dataReps, query)
	if err != nil {
		t.Fatalf("resolveMiReKyouWordMatchTargetIDs failed: %v", err)
	}
	assertIDSet(t, miReKyouTargetIDs, "kmemo-1")

	// 同じ検索の中でトップレベルのMiReKyouからも呼ばれる
	if _, err := resolveMiReKyouWordMatchTargetIDs(ctx, dataReps, query); err != nil {
		t.Fatalf("resolveMiReKyouWordMatchTargetIDs failed: %v", err)
	}

	if got := kmemo.callCount(); got != 1 {
		t.Errorf("実データrepの FindKyous が %d 回呼ばれた。ReKyouとMiReKyouで共有して1回に収まるべき", got)
	}
	if got := miReKyou.callCount(); got != 1 {
		t.Errorf("MiReKyou repの FindKyous が %d 回呼ばれた。1回に収まるべき", got)
	}
}

// ReKyou側で和をとってもメモに載っているmapを汚さないこと。
// 汚すとMiReKyouの解決にMiReKyou自身のIDが混ざる。
func TestReKyouTargetResolutionDoesNotMutateMemoizedSet(t *testing.T) {
	ctx := WithTargetResolutionMemo(context.Background())
	dataReps := Repositories{newCountingRep("Kmemo", "kmemo-1")}
	miReKyouReps := Repositories{newCountingRep("MiReKyou", "mirekyou-1")}
	query := wordQuery()

	if _, err := resolveReKyouWordMatchTargetIDs(ctx, dataReps, miReKyouReps, query); err != nil {
		t.Fatalf("resolveReKyouWordMatchTargetIDs failed: %v", err)
	}
	dataIDs, err := resolveMiReKyouWordMatchTargetIDs(ctx, dataReps, query)
	if err != nil {
		t.Fatalf("resolveMiReKyouWordMatchTargetIDs failed: %v", err)
	}
	assertIDSet(t, dataIDs, "kmemo-1")
}

// queryが違えば別の解決になること。
// メモがqueryを見ずに使い回すと、絞り込みの違う検索の結果を返してしまう。
func TestTargetResolutionMemoIsPerQuery(t *testing.T) {
	ctx := WithTargetResolutionMemo(context.Background())
	kmemo := newCountingRep("Kmemo", "kmemo-1")
	dataReps := Repositories{kmemo}

	if _, err := resolveMiReKyouWordMatchTargetIDs(ctx, dataReps, wordQuery()); err != nil {
		t.Fatalf("1回目: %v", err)
	}
	if _, err := resolveMiReKyouWordMatchTargetIDs(ctx, dataReps, wordQuery()); err != nil {
		t.Fatalf("2回目: %v", err)
	}

	if got := kmemo.callCount(); got != 2 {
		t.Errorf("FindKyous が %d 回。別のqueryなら別々に解決すべき", got)
	}
}

// contextにメモが無くても今までどおり動くこと。
// repを直接叩く経路や単体テストはメモを載せない。
func TestTargetResolutionWithoutMemoStillResolves(t *testing.T) {
	ctx := context.Background()
	kmemo := newCountingRep("Kmemo", "kmemo-1")
	dataReps := Repositories{kmemo}
	query := wordQuery()

	for range 2 {
		ids, err := resolveMiReKyouWordMatchTargetIDs(ctx, dataReps, query)
		if err != nil {
			t.Fatalf("resolveMiReKyouWordMatchTargetIDs failed: %v", err)
		}
		assertIDSet(t, ids, "kmemo-1")
	}
	if got := kmemo.callCount(); got != 2 {
		t.Errorf("FindKyous が %d 回。メモが無ければ毎回解決するべき", got)
	}
}

// 並行に呼ばれても1回しか解決しないこと。
// find_filter はrepを並列に検索するので、ReKyouとMiReKyouは同時に走りうる。
func TestTargetResolutionMemoIsSafeForConcurrentCalls(t *testing.T) {
	ctx := WithTargetResolutionMemo(context.Background())
	kmemo := newCountingRep("Kmemo", "kmemo-1")
	dataReps := Repositories{kmemo}
	query := wordQuery()

	wg := &sync.WaitGroup{}
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ids, err := resolveMiReKyouWordMatchTargetIDs(ctx, dataReps, query)
			if err != nil || !ids["kmemo-1"] {
				t.Errorf("resolveMiReKyouWordMatchTargetIDs = %v, %v", sortedKeys(ids), err)
			}
		}()
	}
	wg.Wait()

	if got := kmemo.callCount(); got != 1 {
		t.Errorf("並行呼び出しで FindKyous が %d 回。1回に収まるべき", got)
	}
}

// ReKyouの委譲先が「実データrep + MiReKyou」であること。
// メモはこの関係を前提に実データrepぶんをMiReKyouと共有しているので、
// 片方にだけrep種別が足されると、その種別がReKyouのワード検索から漏れる。
func TestCollectNonReKyouRepositoriesIsDataPlusMiReKyou(t *testing.T) {
	repositories := &GkillRepositories{}
	kmemoRep, err := NewKmemoRepositorySQLite3Impl(context.Background(), t.TempDir()+"/kmemo.db", true)
	if err != nil {
		t.Fatalf("failed to create kmemo repo: %v", err)
	}
	t.Cleanup(func() { _ = kmemoRep.Close(context.Background()) })
	repositories.KmemoReps = KmemoRepositories{kmemoRep}

	nonReKyou := repositories.collectNonReKyouRepositories()
	data := repositories.collectTargetDataRepositories()
	miReKyou := repositories.collectMiReKyouRepositories()

	if len(nonReKyou) != len(data)+len(miReKyou) {
		t.Fatalf("ReKyouの委譲先が %d 件。実データ %d 件 + MiReKyou %d 件 と一致すべき",
			len(nonReKyou), len(data), len(miReKyou))
	}
	for i, rep := range append(append(Repositories{}, data...), miReKyou...) {
		if nonReKyou[i] != rep {
			t.Errorf("%d 番目のrepが一致しない", i)
		}
	}
}

// リポジトリ群を辿れないReKyouにワード検索をかけてもnil参照で落ちないこと。
// 集約実装だけ !allowAllTargets の判定が抜けていた。
func TestReKyouRepositoriesFindKyousWithoutRepositoriesDoesNotPanic(t *testing.T) {
	rekyouReps := &ReKyouRepositories{GkillRepositories: nil}
	if _, err := rekyouReps.FindKyous(context.Background(), wordQuery()); err != nil {
		t.Fatalf("FindKyous failed: %v", err)
	}
}

// リポジトリ群を辿れないReKyou repが、ReKyouを1件でも持っていても落ちないこと。
// この経路(allowAllTargets)ではターゲット解決をせず全部通すので、
// 解決に使う repsWithoutRekyou には触ってはいけない。
// TX中の一時repや単体テストがこの形になる。
func TestReKyouRepositoryFindKyousWithoutRepositoriesDoesNotPanic(t *testing.T) {
	ctx := context.Background()
	repo := newTempReKyouRepo(t, nil)
	if err := repo.AddReKyouInfo(ctx, makeReKyou("rekyou-allow-all", "target-allow-all")); err != nil {
		t.Fatalf("AddReKyouInfo failed: %v", err)
	}

	for name, query := range map[string]*find.FindQuery{
		"ワード指定あり": wordQuery(),
		"ワード指定なし": {},
	} {
		matchKyous, err := repo.FindKyous(ctx, query)
		if err != nil {
			t.Fatalf("%s: FindKyous failed: %v", name, err)
		}
		if len(matchKyous) != 1 {
			t.Errorf("%s: ヒット件数 = %d, want 1 (ターゲット解決できないときは全部通す)", name, len(matchKyous))
		}
	}
}

// ワード委譲検索が要るかの判定は「語が1つでもあるか」だけを見ること。
//
// FindQuery のゲート判定 HasWordFilter() は「Words/NotWordsが非nilか」なので、
// 非nil空(明示的に語なし)でも真になる。これをそのまま委譲の要否に使うと、
// 語が1つも無いのに実データrepを全部舐めてターゲット解決を走らせてしまう。
// SQL側は語なしなら条件を足さない（＝素通し）ので、解決した結果で絞る意味も無い。
func TestIsWordFilterEnabledIgnoresEmptyWordSlices(t *testing.T) {
	for name, testCase := range map[string]struct {
		query *find.FindQuery
		want  bool
	}{
		"Words/NotWordsともnil":   {&find.FindQuery{}, false},
		"Wordsが非nil空":           {&find.FindQuery{Words: []string{}}, false},
		"NotWordsが非nil空":        {&find.FindQuery{NotWords: []string{}}, false},
		"Words/NotWordsとも非nil空": {&find.FindQuery{Words: []string{}, NotWords: []string{}}, false},
		"Wordsに語がある":            {&find.FindQuery{Words: []string{"word"}}, true},
		"NotWordsに語がある":         {&find.FindQuery{NotWords: []string{"word"}}, true},
	} {
		if got := isWordFilterEnabled(testCase.query); got != testCase.want {
			t.Errorf("%s: isWordFilterEnabled = %v, want %v", name, got, testCase.want)
		}
	}

	// 非nil空でも HasWordFilter は真。両者を取り違えないことをここで示しておく
	emptyWordsQuery := &find.FindQuery{Words: []string{}}
	if !emptyWordsQuery.HasWordFilter() {
		t.Fatal("前提が変わっている: 非nil空の Words は HasWordFilter では有効のはず")
	}
}
