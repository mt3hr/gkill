package threads

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func init() {
	// Ensure the semaphore is initialized for tests.
	sem = make(chan struct{}, 4)
}

func TestInitAndAcquireRelease(t *testing.T) {
	release, err := Acquire(context.Background())
	if err != nil {
		t.Fatalf("Acquire returned error: %v", err)
	}
	release()
}

func TestGoExecutesFunctionAndWaitGroupCompletes(t *testing.T) {
	var called atomic.Bool
	var wg sync.WaitGroup

	err := Go(context.Background(), &wg, func() {
		called.Store(true)
	})
	if err != nil {
		t.Fatalf("Go returned error: %v", err)
	}

	wg.Wait()

	if !called.Load() {
		t.Fatal("expected function to be called")
	}
}

func TestAcquireContextCancellation(t *testing.T) {
	// Fill the semaphore completely so the next Acquire must block.
	capacity := cap(sem)
	for range capacity {
		sem <- struct{}{}
	}
	// Drain after test.
	defer func() {
		for range capacity {
			<-sem
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := Acquire(ctx)
	if err == nil {
		t.Fatal("expected error from cancelled context, got nil")
	}
	if err != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestMultipleConcurrentGoCalls(t *testing.T) {
	const n = 10
	var counter atomic.Int64
	var wg sync.WaitGroup

	for i := range n {
		err := Go(context.Background(), &wg, func() {
			counter.Add(1)
			// Simulate some work.
			time.Sleep(1 * time.Millisecond)
		})
		if err != nil {
			t.Fatalf("Go call %d returned error: %v", i, err)
		}
	}

	wg.Wait()

	if got := counter.Load(); got != int64(n) {
		t.Fatalf("expected counter=%d, got %d", n, got)
	}
}

// withPool はテストの間だけプールを差し替えます。
// Initはsync.Onceなのでテストからサイズを変えられないため、semを直接入れ替えます。
//
// フォールバックの閾値も短縮します。本番の閾値(30秒)は
// 「混んでいる」と「入れ子で永久に空かない」を見分けるためのもので、
// テストでそのまま待つと時間がかかりすぎるだけです。
func withPool(t *testing.T, size int) {
	t.Helper()
	oldSem := sem
	oldSaturated := saturatedUntilUnixNano.Load()
	oldWait := acquireWaitBeforeInline
	sem = make(chan struct{}, size)
	saturatedUntilUnixNano.Store(0)
	acquireWaitBeforeInline = 100 * time.Millisecond
	t.Cleanup(func() {
		sem = oldSem
		saturatedUntilUnixNano.Store(oldSaturated)
		acquireWaitBeforeInline = oldWait
	})
}

// fillPool はプールを満杯にします。解放用の関数を返します。
func fillPool(t *testing.T, size int) func() {
	t.Helper()
	releases := make([]func(), 0, size)
	for range size {
		release, err := Acquire(context.Background())
		if err != nil {
			t.Fatalf("acquire: %v", err)
		}
		releases = append(releases, release)
	}
	return func() {
		for _, release := range releases {
			release()
		}
	}
}

// プールが枯渇していても Go は必ず前に進む。
// 修正前は Acquire が無期限にブロックして恒久ハングしていた。
func TestGoRunsInlineWhenPoolExhausted(t *testing.T) {
	withPool(t, 2)
	release := fillPool(t, 2)
	defer release()

	before := InlineFallbackCount()

	var wg sync.WaitGroup
	executed := make(chan struct{}, 1)

	done := make(chan error, 1)
	go func() {
		done <- Go(context.Background(), &wg, func() { executed <- struct{}{} })
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Go returned error: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Go did not return: goroutine pool deadlock")
	}

	wg.Wait()

	select {
	case <-executed:
	default:
		t.Fatal("fn was not executed")
	}

	if InlineFallbackCount() <= before {
		t.Fatal("inline fallback was not counted")
	}
}

// 入れ子(スロットを保持したまま子のスロットを要求する)でも止まらない。
func TestGoNestedDoesNotDeadlock(t *testing.T) {
	withPool(t, 1)

	var outerWg sync.WaitGroup
	innerExecuted := make(chan struct{}, 1)

	done := make(chan struct{})
	go func() {
		defer close(done)
		err := Go(context.Background(), &outerWg, func() {
			var innerWg sync.WaitGroup
			if err := Go(context.Background(), &innerWg, func() { innerExecuted <- struct{}{} }); err != nil {
				t.Errorf("inner Go returned error: %v", err)
			}
			innerWg.Wait()
		})
		if err != nil {
			t.Errorf("outer Go returned error: %v", err)
		}
		outerWg.Wait()
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("nested Go did not return: goroutine pool deadlock")
	}

	select {
	case <-innerExecuted:
	default:
		t.Fatal("inner fn was not executed")
	}
}

// キャンセル済みctxではfnを実行せずエラーを返す。
func TestGoReturnsErrorForCanceledContext(t *testing.T) {
	withPool(t, 2)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var wg sync.WaitGroup
	var executed atomic.Bool
	err := Go(ctx, &wg, func() { executed.Store(true) })
	if err == nil {
		t.Fatal("expected error for canceled context")
	}
	wg.Wait()
	if executed.Load() {
		t.Fatal("fn should not be executed for canceled context")
	}
}

// 空きがあるときはinlineに落ちず、ちゃんと並列に走る。
// フォールバックを入れたせいで常時逐次化していないことの担保。
func TestGoKeepsParallelismWhenPoolHasRoom(t *testing.T) {
	withPool(t, 4)

	var current atomic.Int64
	var maxConcurrent atomic.Int64

	var wg sync.WaitGroup
	for range 4 {
		err := Go(context.Background(), &wg, func() {
			running := current.Add(1)
			for {
				peak := maxConcurrent.Load()
				if running <= peak || maxConcurrent.CompareAndSwap(peak, running) {
					break
				}
			}
			time.Sleep(50 * time.Millisecond)
			current.Add(-1)
		})
		if err != nil {
			t.Fatalf("Go returned error: %v", err)
		}
	}
	wg.Wait()

	if maxConcurrent.Load() < 2 {
		t.Fatalf("expected parallel execution, max concurrent = %d", maxConcurrent.Load())
	}
}

// 枯渇を1回検知したら猶予期間の間は待たずにinlineへ倒す。
// fan-outループの要素ごとに acquireWaitBeforeInline を払わないことの担保。
func TestGoDoesNotPayTimeoutPerCallWhileSaturated(t *testing.T) {
	withPool(t, 1)
	release := fillPool(t, 1)
	defer release()

	var wg sync.WaitGroup
	var executed atomic.Int64

	start := time.Now()
	const callCount = 10
	for range callCount {
		if err := Go(context.Background(), &wg, func() { executed.Add(1) }); err != nil {
			t.Fatalf("Go returned error: %v", err)
		}
	}
	elapsed := time.Since(start)
	wg.Wait()

	if executed.Load() != callCount {
		t.Fatalf("executed = %d, want %d", executed.Load(), callCount)
	}
	// 毎回待つと callCount * acquireWaitBeforeInline かかる。
	// 1回ぶんの待ち + 余裕を上限にする。
	limit := acquireWaitBeforeInline * 3
	if elapsed > limit {
		t.Fatalf("elapsed = %v, want <= %v (timeout paid per call?)", elapsed, limit)
	}
}
