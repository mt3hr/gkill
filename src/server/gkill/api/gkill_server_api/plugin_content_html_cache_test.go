package gkill_server_api

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// 2回目以降はプラグインへ問い合わせずキャッシュから返す。
func TestPluginContentHTMLCacheServesFromCache(t *testing.T) {
	cache := newPluginContentHTMLCache()
	key := pluginContentHTMLCacheKey("user", "rep", "kyou")

	var calls atomic.Int64
	fetch := func(ctx context.Context) (string, error) {
		calls.Add(1)
		return "<p>html</p>", nil
	}

	for range 5 {
		html, err := cache.GetOrFetch(context.Background(), key, fetch)
		if err != nil {
			t.Fatalf("GetOrFetch() error: %v", err)
		}
		if html != "<p>html</p>" {
			t.Fatalf("html = %q, want %q", html, "<p>html</p>")
		}
	}

	if got := calls.Load(); got != 1 {
		t.Fatalf("プラグインへの問い合わせ回数 = %d, want 1", got)
	}
}

// 同じKyouへの同時要求は1回の問い合わせに畳まれる。
// 仮想スクロールで同じ行が一斉に描画されたときにプラグインへ殺到させないための性質。
func TestPluginContentHTMLCacheCollapsesConcurrentRequests(t *testing.T) {
	cache := newPluginContentHTMLCache()
	key := pluginContentHTMLCacheKey("user", "rep", "kyou")

	const n = 20

	var calls atomic.Int64
	// arrived は GetOrFetch を呼ぶ直前まで来たgoroutineの数。
	// 全員が揃うまで問い合わせを終わらせないことで、
	// 「先行の1本が終わってから後続が来た」せいで2回走るのを防ぐ。
	var arrived atomic.Int64
	fetch := func(ctx context.Context) (string, error) {
		calls.Add(1)
		for arrived.Load() < n {
			time.Sleep(time.Millisecond)
		}
		// 全員が揃ってから、待ち手として登録されるまでの猶予を置く
		time.Sleep(50 * time.Millisecond)
		return "<p>html</p>", nil
	}

	var wg sync.WaitGroup
	errs := make(chan error, n)
	for range n {
		wg.Go(func() {
			arrived.Add(1)
			html, err := cache.GetOrFetch(context.Background(), key, fetch)
			if err != nil {
				errs <- err
				return
			}
			if html != "<p>html</p>" {
				errs <- fmt.Errorf("html = %q", html)
			}
		})
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Error(err)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("プラグインへの問い合わせ回数 = %d, want 1", got)
	}
}

// 呼び出し元が打ち切ったら待つのをやめる。
// 代表になった1本が遅くても、後続が道連れで固まらないようにするため。
func TestPluginContentHTMLCacheRespectsCallerCancel(t *testing.T) {
	cache := newPluginContentHTMLCache()
	key := pluginContentHTMLCacheKey("user", "rep", "kyou")

	release := make(chan struct{})
	t.Cleanup(func() { close(release) })
	fetch := func(ctx context.Context) (string, error) {
		<-release
		return "<p>html</p>", nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		_, err := cache.GetOrFetch(ctx, key, fetch)
		done <- err
	}()

	select {
	case err := <-done:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("err = %v, want DeadlineExceeded", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("呼び出し元の打ち切りで抜けられていない")
	}
}

// エラーはキャッシュしない。次の要求でもう一度問い合わせる。
func TestPluginContentHTMLCacheDoesNotCacheErrors(t *testing.T) {
	cache := newPluginContentHTMLCache()
	key := pluginContentHTMLCacheKey("user", "rep", "kyou")

	var calls atomic.Int64
	fetch := func(ctx context.Context) (string, error) {
		calls.Add(1)
		return "", errors.New("plugin error")
	}

	for range 3 {
		if _, err := cache.GetOrFetch(context.Background(), key, fetch); err == nil {
			t.Fatal("expected error")
		}
	}

	if got := calls.Load(); got != 3 {
		t.Fatalf("プラグインへの問い合わせ回数 = %d, want 3（エラーをキャッシュしている）", got)
	}
}

// 設定変更などでユーザ単位に捨てられる。他ユーザのぶんは残る。
func TestPluginContentHTMLCacheInvalidateUser(t *testing.T) {
	cache := newPluginContentHTMLCache()
	targetKey := pluginContentHTMLCacheKey("user", "rep", "kyou")
	otherKey := pluginContentHTMLCacheKey("other_user", "rep", "kyou")

	var calls atomic.Int64
	fetch := func(ctx context.Context) (string, error) {
		calls.Add(1)
		return "<p>html</p>", nil
	}

	if _, err := cache.GetOrFetch(context.Background(), targetKey, fetch); err != nil {
		t.Fatalf("GetOrFetch() error: %v", err)
	}
	if _, err := cache.GetOrFetch(context.Background(), otherKey, fetch); err != nil {
		t.Fatalf("GetOrFetch() error: %v", err)
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("問い合わせ回数 = %d, want 2", got)
	}

	cache.InvalidateUser("user")

	// 捨てられた側は再取得、残っている側はキャッシュヒット
	if _, err := cache.GetOrFetch(context.Background(), targetKey, fetch); err != nil {
		t.Fatalf("GetOrFetch() error: %v", err)
	}
	if _, err := cache.GetOrFetch(context.Background(), otherKey, fetch); err != nil {
		t.Fatalf("GetOrFetch() error: %v", err)
	}
	if got := calls.Load(); got != 3 {
		t.Fatalf("問い合わせ回数 = %d, want 3（無効化の範囲が正しくない）", got)
	}
}

// 件数上限を超えたら古いものから捨てる。
func TestPluginContentHTMLCacheEvictsOldestOverLimit(t *testing.T) {
	cache := newPluginContentHTMLCache()

	fetch := func(ctx context.Context) (string, error) { return "<p>html</p>", nil }

	for i := range pluginContentHTMLCacheMaxEntries + 10 {
		key := pluginContentHTMLCacheKey("user", "rep", fmt.Sprintf("kyou-%d", i))
		if _, err := cache.GetOrFetch(context.Background(), key, fetch); err != nil {
			t.Fatalf("GetOrFetch() error: %v", err)
		}
	}

	cache.mu.Lock()
	entryCount := len(cache.entries)
	orderCount := len(cache.order)
	cache.mu.Unlock()

	if entryCount > pluginContentHTMLCacheMaxEntries {
		t.Fatalf("保持件数 = %d, want <= %d", entryCount, pluginContentHTMLCacheMaxEntries)
	}
	if orderCount != entryCount {
		t.Fatalf("order = %d, entries = %d（挿入順の管理がずれている）", orderCount, entryCount)
	}

	// 最初に入れたものは捨てられている
	if _, ok := cache.lookup(pluginContentHTMLCacheKey("user", "rep", "kyou-0")); ok {
		t.Fatal("古いエントリが捨てられていない")
	}
}
