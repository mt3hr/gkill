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
