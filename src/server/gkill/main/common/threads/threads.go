package threads

import (
	"context"
	"sync"

	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_options"
)

var (
	sem  chan struct{}
	once sync.Once
)

// mainから明示的に呼び出してください
func Init() {
	once.Do(func() {
		n := gkill_options.GoroutinePool
		if n <= 0 {
			n = 1
		}
		sem = make(chan struct{}, n)
	})
}

func Acquire(ctx context.Context) (release func(), err error) {
	if sem == nil {
		Init()
	}
	select {
	case sem <- struct{}{}:
		return func() { <-sem }, nil
	case <-ctx.Done():
		return func() {}, ctx.Err()
	}
}

func Go(ctx context.Context, wg *sync.WaitGroup, fn func()) error {
	release, err := Acquire(ctx)
	if err != nil {
		return err
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer release()
		fn()
	}()
	return nil
}
