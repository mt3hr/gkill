package reps

// 派生キャッシュの一括生成が threads.Go を入れ子にしていないことの回帰テスト。
//
// rep単位のファンアウト（IDFKyouRepositories.GenerateThumbCache）が
// 既に threads.Go のスロットを保持しているので、その内側でファイル単位の並列化に
// threads.Go を使うと入れ子になり、プールが枯渇して恒久ハングする（ADR-0206）。
//
// threads.Go には枯渇時の inline フォールバックが入っているので、
// 「返ってくること」だけでは入れ子の復活を検知できない。
// 内側でスロットを取ろうとしていないことをフォールバック回数で確認する。

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/main/common/threads"
)

func TestGenerateThumbCacheDoesNotExhaustPool(t *testing.T) {
	repo, contentDir, thumbCacheDir := newIDFRepForThumbBatchTest(t)
	ctx := context.Background()

	rels := []string{"a.png", "b.png", "c.png", "sub/d.png"}
	for index, rel := range rels {
		writeTestImage(t, filepath.Join(contentDir, filepath.FromSlash(rel)), index*8)
	}
	if err := repo.IDF(ctx); err != nil {
		t.Fatalf("IDF failed: %v", err)
	}
	wantNames := make([]string, 0, len(rels))
	for _, rel := range rels {
		wantNames = append(wantNames, thumbCacheNameForTest(t, repo, contentDir, rel))
	}

	// スレッドプールを満杯にする（タイムアウト付きAcquireで空きが無くなるまで取得）
	releases := []func(){}
	t.Cleanup(func() {
		for _, release := range releases {
			release()
		}
	})
	for {
		acquireCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
		release, err := threads.Acquire(acquireCtx)
		cancel()
		if err != nil {
			break // 満杯になった
		}
		releases = append(releases, release)
	}
	if len(releases) == 0 {
		t.Fatal("could not acquire any pool slot")
	}

	// 1スロットだけ空け、そのスロットを保持する閉包の中から一括生成を呼ぶ
	releases[len(releases)-1]()
	releases = releases[:len(releases)-1]

	inlineFallbackBefore := threads.InlineFallbackCount()

	done := make(chan struct{})
	go func() {
		defer close(done)
		wg := &sync.WaitGroup{}
		if err := threads.Go(ctx, wg, func() {
			if err := repo.GenerateThumbCache(ctx); err != nil {
				t.Errorf("GenerateThumbCache() error: %v", err)
			}
		}); err != nil {
			t.Errorf("threads.Go() error: %v", err)
		}
		wg.Wait()
	}()

	select {
	case <-done:
	case <-time.After(60 * time.Second):
		t.Fatal("プールが埋まった状態で一括生成が完了しない。threads.Goを入れ子にしている")
	}

	if got := threads.InlineFallbackCount(); got != inlineFallbackBefore {
		t.Errorf("内側でスレッドプールのスロットを取っている: inline fallback %d -> %d", inlineFallbackBefore, got)
	}

	for _, name := range wantNames {
		if _, err := os.Stat(filepath.Join(thumbCacheDir, name)); err != nil {
			t.Errorf("プールが埋まっていてもサムネイルは生成されるはず: %s: %v", name, err)
		}
	}
}
