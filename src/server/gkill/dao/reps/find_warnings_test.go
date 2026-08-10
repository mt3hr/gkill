package reps

// 検索警告コレクタ(find_warnings.go)のテスト。
// プラグイン検索の失敗を「エラーで検索全体を落とす」のではなく
// 「警告として回収してメッセージ表示する」ための仕組み。

import (
	"context"
	"sync"
	"testing"
)

func TestFindWarnings_CollectsPluginNames(t *testing.T) {
	ctx := WithFindWarnings(context.Background())

	AppendPluginFindWarning(ctx, "plugin-a")
	AppendPluginFindWarning(ctx, "plugin-b")

	got := PluginFindWarnings(ctx)
	if len(got) != 2 {
		t.Fatalf("警告は2件のはず: got %v", got)
	}
	if got[0] != "plugin-a" || got[1] != "plugin-b" {
		t.Errorf("記録順に返るはず: got %v", got)
	}
}

// コレクタが無いcontextでは何もせず、panicもしないこと(rep直叩き・テスト経路の安全)
func TestFindWarnings_NoCollectorIsNoop(t *testing.T) {
	ctx := context.Background()

	AppendPluginFindWarning(ctx, "plugin-a")

	if got := PluginFindWarnings(ctx); got != nil {
		t.Errorf("コレクタ無しではnilのはず: got %v", got)
	}
}

// 並行追記で競合しないこと(-race で検証)
func TestFindWarnings_ConcurrentAppend(t *testing.T) {
	ctx := WithFindWarnings(context.Background())

	wg := &sync.WaitGroup{}
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			AppendPluginFindWarning(ctx, "plugin")
		}()
	}
	wg.Wait()

	if got := PluginFindWarnings(ctx); len(got) != 10 {
		t.Errorf("並行追記10件が全部残るはず: got %d件", len(got))
	}
}
