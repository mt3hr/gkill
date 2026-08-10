package reps

// *find.FindQuery は複数repへ同じポインタのまま渡され、repごとのgoroutineで
// 並列に使われる。rep側がqueryを書き換えると、データ競合になるだけでなく
// 他repの検索条件まで壊れる。ここではその「書き換えないこと」を回帰テストで固定する。

import (
	"context"
	"database/sql"
	"sync"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	gkill_cache "github.com/mt3hr/gkill/src/server/gkill/dao/reps/cache"
)

// IDFのFindKyousは検索語を小文字化して使うが、
// query.Words のスライスを直接書き換えてはいけない。
func TestIDFKyouFindKyous_DoesNotMutateQueryWords(t *testing.T) {
	repo := newTempIDFKyouRepo(t)
	ctx := context.Background()

	if err := repo.AddIDFKyouInfo(ctx, makeIDFKyou("idf-mut-001", "Photo.JPG")); err != nil {
		t.Fatalf("AddIDFKyouInfo failed: %v", err)
	}

	query := &find.FindQuery{
		Words:    []string{"Photo", "MiXeD"},
		NotWords: []string{"NotThis"},
	}

	if _, err := repo.FindKyous(ctx, query); err != nil {
		t.Fatalf("FindKyous failed: %v", err)
	}

	if got := query.Words[0]; got != "Photo" {
		t.Errorf("query.Words[0] が書き換えられた: %q, want %q", got, "Photo")
	}
	if got := query.Words[1]; got != "MiXeD" {
		t.Errorf("query.Words[1] が書き換えられた: %q, want %q", got, "MiXeD")
	}
	if got := query.NotWords[0]; got != "NotThis" {
		t.Errorf("query.NotWords[0] が書き換えられた: %q, want %q", got, "NotThis")
	}
}

// GkillRepositories.FindTags は rep ごとに「最新版のIDの一覧」を
// query へ詰めてから rep に渡すが、呼び出し元のqueryを書き換えてはいけない。
func TestGkillRepositoriesFindTags_DoesNotMutateCallerQuery(t *testing.T) {
	ctx := context.Background()
	tagRep := newTempTagRepo(t)

	addrDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory sqlite: %v", err)
	}
	t.Cleanup(func() { _ = addrDB.Close() })
	addrDAO, err := gkill_cache.NewLatestDataRepositoryAddressSQLite3Impl("testuser", addrDB, &sync.RWMutex{})
	if err != nil {
		t.Fatalf("NewLatestDataRepositoryAddressSQLite3Impl: %v", err)
	}

	repositories := &GkillRepositories{
		TagReps:                        TagRepositories{tagRep},
		LatestDataRepositoryAddressDAO: addrDAO,
	}

	// IDs が nil（フィルタ未使用）のまま渡し、rep 側で ID 一覧が詰められても
	// 呼び出し元の query.IDs が nil のままであることを確認する。
	// ポインタフィールド（MiBoardName/PlaingTime等）は浅いコピーでポインタ先が共有されるので、
	// 「ポインタ先へ書かず、差し替えは必ずポインタごと」の規約もここで固定する。
	boardName := "board-mut"
	plaingTime := time.Date(2024, 4, 1, 12, 0, 0, 0, time.UTC)
	query := &find.FindQuery{
		MiBoardName: &boardName,
		PlaingTime:  &plaingTime,
	}

	if _, err := repositories.FindTags(ctx, query); err != nil {
		t.Fatalf("FindTags failed: %v", err)
	}

	if query.IDs != nil {
		t.Errorf("呼び出し元の query.IDs が書き換えられた: %v", query.IDs)
	}
	if query.MiBoardName == nil || *query.MiBoardName != "board-mut" {
		t.Errorf("呼び出し元の query.MiBoardName（ポインタ先）が書き換えられた: %v", query.MiBoardName)
	}
	if query.PlaingTime == nil || !query.PlaingTime.Equal(plaingTime) {
		t.Errorf("呼び出し元の query.PlaingTime（ポインタ先）が書き換えられた: %v", query.PlaingTime)
	}
}
