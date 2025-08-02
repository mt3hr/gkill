package threads

import (
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_log"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_options"
)

var (
	threadPool  = make(chan struct{}, gkill_options.GoroutinePool)
	threadCount = 0
)

func init() {
	for i := 0; i < gkill_options.GoroutinePool; i++ {
		threadPool <- struct{}{}
		threadCount++
	}
}

func AllocateThread() func() {
	gkill_log.Trace.Printf("threadCount: %d\n", threadCount)
	threadCount--
	<-threadPool
	return func() {
		go func() {
			threadCount++
			gkill_log.Trace.Printf("threadCount: %d\n", threadCount)
			threadPool <- struct{}{}
		}()
	}
}
