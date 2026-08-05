package reps

import (
	"context"
	sqllib "database/sql"
	"path/filepath"
	"sync"
	"testing"
	"time"

	gorilla_mux "github.com/gorilla/mux"
	"github.com/mt3hr/gkill/src/server/gkill/api/find"
)

// newIDFRepForCacheTest は指定ディレクトリにIDFリポジトリを作ります。
func newIDFRepForCacheTest(t *testing.T, dir string) IDFKyouRepository {
	t.Helper()

	dbFile := filepath.Join(dir, "gkill_id.db")
	ignorePatterns := []string{}
	repo, err := NewIDFDirRep(context.Background(), "", dir, dbFile, true, gorilla_mux.NewRouter(), false, &ignorePatterns, nil)
	if err != nil {
		t.Fatalf("NewIDFDirRep() error: %v", err)
	}
	t.Cleanup(func() { repo.Close(context.Background()) })
	return repo
}

// addIDFKyouForCacheTest はIDFKyouを1件追記します。
func addIDFKyouForCacheTest(t *testing.T, rep IDFKyouRepository, id string, targetFile string) {
	t.Helper()

	now := time.Now()
	err := rep.AddIDFKyouInfo(context.Background(), IDFKyou{
		IsDeleted:    false,
		ID:           id,
		TargetFile:   targetFile,
		RelatedTime:  now,
		CreateTime:   now,
		CreateApp:    "test",
		CreateDevice: "test",
		CreateUser:   "test",
		UpdateTime:   now,
		UpdateApp:    "test",
		UpdateDevice: "test",
		UpdateUser:   "test",
	})
	if err != nil {
		t.Fatalf("AddIDFKyouInfo() error: %v", err)
	}
}

// IDFにキャッシュ変更検知が入っていることの回帰テスト。
//
// IDFは全rep種別の中で最も行数が多い（実データで50万行超）のに、
// LastUpdateCacheChanged() が true 固定で、cached実装も
// commitCacheRebuildIfSupported を呼んでいなかった。
// そのため update_cache のたびに全件のフルリビルドが走っていた。
//
// 修正後は2回目以降「変更なし」になり、再構築が飛ばされる。
func TestIDFCachedSkipsRebuildWhenUnchanged(t *testing.T) {
	ctx := context.Background()

	dir := t.TempDir()
	idfRep := newIDFRepForCacheTest(t, dir)
	addIDFKyouForCacheTest(t, idfRep, "idf-1", "a.txt")

	cacheDB, err := sqllib.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open cache db: %v", err)
	}
	t.Cleanup(func() { cacheDB.Close() })

	aggregated := IDFKyouRepositories{idfRep}
	cachedRep, err := NewIDFCachedRep(ctx, aggregated, cacheDB, &sync.RWMutex{}, "IDF_GRANULAR_CACHE_TEST")
	if err != nil {
		t.Fatalf("NewIDFCachedRep() error: %v", err)
	}

	// 1回目は基準が無いので必ず再構築される。
	// 構築されたことはキャッシュの中身で確認する
	if err := cachedRep.UpdateCache(ctx); err != nil {
		t.Fatalf("1回目のUpdateCache() error: %v", err)
	}

	kyous, err := cachedRep.FindKyous(ctx, &find.FindQuery{})
	if err != nil {
		t.Fatalf("FindKyous() error: %v", err)
	}
	if len(kyous) != 1 {
		t.Fatalf("キャッシュ件数 = %d, want 1（1回目の構築が行われていない）", len(kyous))
	}

	// 2回目はファイルが変わっていないので再構築を飛ばす
	if err := aggregated.UpdateCache(ctx); err != nil {
		t.Fatalf("2回目のUpdateCache() error: %v", err)
	}
	if aggregated.LastUpdateCacheChanged() {
		t.Fatal("ファイルが変わっていないのに「変更あり」と判定されている（フルリビルドが毎回走る）")
	}
}

// 変わったrepがあれば再構築され、キャッシュに反映される。
// 変更検知を入れたせいで更新を取りこぼしていないことの担保。
func TestIDFCachedRebuildsWhenChanged(t *testing.T) {
	ctx := context.Background()

	dir := t.TempDir()
	idfRep := newIDFRepForCacheTest(t, dir)
	addIDFKyouForCacheTest(t, idfRep, "idf-1", "a.txt")

	cacheDB, err := sqllib.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open cache db: %v", err)
	}
	t.Cleanup(func() { cacheDB.Close() })

	aggregated := IDFKyouRepositories{idfRep}
	cachedRep, err := NewIDFCachedRep(ctx, aggregated, cacheDB, &sync.RWMutex{}, "IDF_GRANULAR_CACHE_REBUILD_TEST")
	if err != nil {
		t.Fatalf("NewIDFCachedRep() error: %v", err)
	}

	if err := cachedRep.UpdateCache(ctx); err != nil {
		t.Fatalf("1回目のUpdateCache() error: %v", err)
	}

	// mtimeの分解能で取りこぼさないよう、書き込み前に少し待つ
	time.Sleep(20 * time.Millisecond)
	addIDFKyouForCacheTest(t, idfRep, "idf-2", "b.txt")

	if err := cachedRep.UpdateCache(ctx); err != nil {
		t.Fatalf("2回目のUpdateCache() error: %v", err)
	}

	kyous, err := cachedRep.FindKyous(ctx, &find.FindQuery{})
	if err != nil {
		t.Fatalf("FindKyous() error: %v", err)
	}
	if len(kyous) != 2 {
		t.Fatalf("キャッシュ件数 = %d, want 2（変更が反映されていない）", len(kyous))
	}
}
