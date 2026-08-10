package api

// selectMatchRepsFromQuery の回帰テスト。
//
// 1. タイプフィルタ（mi板 / 画像のみ / Plaing / rep種別指定）のときに
//    インメモリキャッシュのrepをUnWrap()して生のディスクrepに戻してしまわないこと。
//    UnWrap()するとキャッシュを丸ごとバイパスし、同一ファイルの端末別重複登録ぶん
//    ディスクを舐めることになる（実データで312rep中263個が重複登録だったケースがある）。
// 2. Reps / RepTypes の nil と非nil空の区別（非nil空 = 候補0件 = 検索結果0件）。
//    len()で有効判定すると、非nil空が「未指定」に化けて全rep検索になる。
// 3. タイプフィルタ同士が和集合になること（以前はif/else ifで、
//    ForMiとrep種別指定を併用するとRepTypesが無視されていた）。

import (
	"context"
	"slices"
	"testing"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
)

// stubIDFRep は reps.IDFKyouRepository を埋め込んで必要なメソッドだけ差し替えたスタブ。
// 埋め込みが nil なので、テストで呼ばない他のメソッドを実装する必要がない。
type stubIDFRep struct {
	reps.IDFKyouRepository
	repName     string
	unwrapCalls *int
	// UnWrap()の戻り。実際のcached repは配下の生repを返すので、それを模す。
	unwrapped reps.Repository
}

func (s *stubIDFRep) GetRepName(_ context.Context) (string, error) {
	return s.repName, nil
}

func (s *stubIDFRep) UnWrap() ([]reps.Repository, error) {
	*s.unwrapCalls++
	return []reps.Repository{s.unwrapped}, nil
}

type stubRawRep struct {
	reps.IDFKyouRepository
	repName string
}

func (s *stubRawRep) GetRepName(_ context.Context) (string, error) {
	return s.repName, nil
}

func (s *stubRawRep) UnWrap() ([]reps.Repository, error) {
	return []reps.Repository{s}, nil
}

// 画像のみ検索（IsImageOnly = タイプフィルタあり / rep名指定なし）で
// キャッシュrepがそのまま使われること。
func TestSelectMatchRepsFromQuery_ImageOnlyKeepsCachedRep(t *testing.T) {
	ctx := context.Background()

	unwrapCalls := 0
	raw := &stubRawRep{repName: "RawIDFRep"}
	cached := &stubIDFRep{repName: "CachedIDFReps", unwrapCalls: &unwrapCalls, unwrapped: raw}

	findCtx := &FindKyouContext{
		MatchReps: map[string]reps.Repository{},
		Repositories: &reps.GkillRepositories{
			IDFKyouReps: reps.IDFKyouRepositories{cached},
		},
		ParsedFindQuery: &find.FindQuery{
			IsImageOnly: true,
		},
	}

	f := &FindFilter{}
	if _, err := f.selectMatchRepsFromQuery(ctx, findCtx); err != nil {
		t.Fatalf("selectMatchRepsFromQuery failed: %v", err)
	}

	if unwrapCalls != 0 {
		t.Errorf("UnWrap() が %d 回呼ばれた。タイプフィルタのみの検索でキャッシュをバイパスしてはいけない", unwrapCalls)
	}
	if len(findCtx.MatchReps) != 1 {
		t.Fatalf("MatchReps の件数 = %d, want 1 (%v)", len(findCtx.MatchReps), findCtx.MatchReps)
	}
	got, ok := findCtx.MatchReps["CachedIDFReps"]
	if !ok {
		t.Fatalf("MatchReps にキャッシュrepが入っていない: %v", findCtx.MatchReps)
	}
	if got != reps.Repository(cached) {
		t.Errorf("MatchReps に入っているのがキャッシュrepではない: %#v", got)
	}
}

// rep名指定ありのときは、個々のrep名を解決する必要があるので
// UnWrap() されること（こちらは意図した挙動）。
func TestSelectMatchRepsFromQuery_RepsSpecifiedStillUnwraps(t *testing.T) {
	ctx := context.Background()

	unwrapCalls := 0
	raw := &stubRawRep{repName: "RawIDFRep"}
	cached := &stubIDFRep{repName: "CachedIDFReps", unwrapCalls: &unwrapCalls, unwrapped: raw}

	findCtx := &FindKyouContext{
		MatchReps: map[string]reps.Repository{},
		Repositories: &reps.GkillRepositories{
			IDFKyouReps: reps.IDFKyouRepositories{cached},
		},
		ParsedFindQuery: &find.FindQuery{
			IsImageOnly: true,
			Reps:        []string{"RawIDFRep"},
		},
	}

	f := &FindFilter{}
	if _, err := f.selectMatchRepsFromQuery(ctx, findCtx); err != nil {
		t.Fatalf("selectMatchRepsFromQuery failed: %v", err)
	}

	if unwrapCalls == 0 {
		t.Error("rep名指定ありのときは名前解決のため UnWrap() が必要")
	}
	if _, ok := findCtx.MatchReps["RawIDFRep"]; !ok {
		t.Errorf("指定したrep名が MatchReps に入っていない: %v", findCtx.MatchReps)
	}
}

// stubMiRep / stubMiReKyouRep / stubKmemoRep はタイプ別候補の組み立てだけを見るためのスタブ。
// stubIDFRep と同じく埋め込みが nil なので、テストで呼ばないメソッドは実装しなくてよい。
type stubMiRep struct {
	reps.MiRepository
	repName string
}

func (s *stubMiRep) GetRepName(_ context.Context) (string, error) {
	return s.repName, nil
}

func (s *stubMiRep) UnWrap() ([]reps.Repository, error) {
	return []reps.Repository{s}, nil
}

type stubMiReKyouRep struct {
	reps.MiReKyouRepository
	repName string
}

func (s *stubMiReKyouRep) GetRepName(_ context.Context) (string, error) {
	return s.repName, nil
}

func (s *stubMiReKyouRep) UnWrap() ([]reps.Repository, error) {
	return []reps.Repository{s}, nil
}

type stubKmemoRep struct {
	reps.KmemoRepository
	repName string
}

func (s *stubKmemoRep) GetRepName(_ context.Context) (string, error) {
	return s.repName, nil
}

func (s *stubKmemoRep) UnWrap() ([]reps.Repository, error) {
	return []reps.Repository{s}, nil
}

// newStubRepositories は Mi / MiReKyou / kmemo を1つずつ持つ GkillRepositories を作る。
// Reps（全rep）にも同じ実体を入れておく。
func newStubRepositories() *reps.GkillRepositories {
	miRep := &stubMiRep{repName: "MiRep"}
	miReKyouRep := &stubMiReKyouRep{repName: "MiReKyouRep"}
	kmemoRep := &stubKmemoRep{repName: "KmemoRep"}
	return &reps.GkillRepositories{
		Reps:      reps.Repositories{miRep, miReKyouRep, kmemoRep},
		MiReps:    reps.MiRepositories{miRep},
		KmemoReps: reps.KmemoRepositories{kmemoRep},
		MiReKyouReps: reps.MiReKyouRepositories{
			MiReKyouRepositories: []reps.MiReKyouRepository{miReKyouRep},
		},
	}
}

func matchRepNames(findCtx *FindKyouContext) []string {
	names := make([]string, 0, len(findCtx.MatchReps))
	for name := range findCtx.MatchReps {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

// 非nilの空スライスは「候補0件」であって「未指定」ではないこと。
// len()で判定すると全rep検索に化け、0件指定のつもりが全件返る。
func TestSelectMatchRepsFromQuery_EmptySliceMeansZeroCandidates(t *testing.T) {
	ctx := context.Background()

	for _, c := range []struct {
		name  string
		query *find.FindQuery
	}{
		{
			// RepTypes: [] → タイプ候補0件。rep名指定は無いのでタイプ候補がそのまま結果になる
			name:  "RepTypesが非nil空",
			query: &find.FindQuery{RepTypes: []string{}},
		},
		{
			// Reps: [] → rep名候補0件。タイプフィルタが無いので全repが候補だが、名前で全部落ちる
			name:  "Repsが非nil空",
			query: &find.FindQuery{Reps: []string{}},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			findCtx := &FindKyouContext{
				MatchReps:       map[string]reps.Repository{},
				Repositories:    newStubRepositories(),
				ParsedFindQuery: c.query,
			}

			f := &FindFilter{}
			if _, err := f.selectMatchRepsFromQuery(ctx, findCtx); err != nil {
				t.Fatalf("selectMatchRepsFromQuery failed: %v", err)
			}

			if len(findCtx.MatchReps) != 0 {
				t.Errorf("非nil空は候補0件のはず: %v", matchRepNames(findCtx))
			}
		})
	}
}

// タイプフィルタは和集合になること。
// 以前はif/else ifだったため、mi板検索とrep種別指定を併用するとRepTypesが無視されていた。
func TestSelectMatchRepsFromQuery_TypeFiltersAreUnioned(t *testing.T) {
	ctx := context.Background()

	for _, c := range []struct {
		name     string
		query    *find.FindQuery
		wantReps []string
	}{
		{
			// rep名指定なし + rep種別指定 → 種別で絞られる（rep名指定に依存しない）
			name:     "rep名指定なしでもrep種別指定が効く",
			query:    &find.FindQuery{Reps: nil, RepTypes: []string{"kmemo"}},
			wantReps: []string{"KmemoRep"},
		},
		{
			// mi板 + rep種別指定 → Mi/MiReKyou と kmemo の和集合
			name:     "ForMiとrep種別指定は和集合",
			query:    &find.FindQuery{ForMi: true, RepTypes: []string{"kmemo"}},
			wantReps: []string{"KmemoRep", "MiReKyouRep", "MiRep"},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			findCtx := &FindKyouContext{
				MatchReps:       map[string]reps.Repository{},
				Repositories:    newStubRepositories(),
				ParsedFindQuery: c.query,
			}

			f := &FindFilter{}
			if _, err := f.selectMatchRepsFromQuery(ctx, findCtx); err != nil {
				t.Fatalf("selectMatchRepsFromQuery failed: %v", err)
			}

			if got := matchRepNames(findCtx); !slices.Equal(got, c.wantReps) {
				t.Errorf("MatchReps = %v, want %v", got, c.wantReps)
			}
		})
	}
}
