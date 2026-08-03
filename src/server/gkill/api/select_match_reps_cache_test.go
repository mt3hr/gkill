package api

// selectMatchRepsFromQuery が、タイプフィルタ（mi板 / 画像のみ / Plaing / rep種別指定）の
// ときにインメモリキャッシュのrepをUnWrap()して生のディスクrepに戻してしまわないことを確認する。
//
// UnWrap()するとキャッシュを丸ごとバイパスし、同一ファイルの端末別重複登録ぶん
// ディスクを舐めることになる（実データで312rep中263個が重複登録だったケースがある）。

import (
	"context"
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
			UseReps:     false,
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
func TestSelectMatchRepsFromQuery_UseRepsStillUnwraps(t *testing.T) {
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
			UseReps:     true,
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
