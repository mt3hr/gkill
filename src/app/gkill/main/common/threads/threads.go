package threads

import "github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_options"

var (
	threadPool = make(chan struct{}, gkill_options.ThreadPool)
)

func init() {
	for i := 0; i < gkill_options.ThreadPool; i++ {
		threadPool <- struct{}{}
	}
}

func AllocateThread() func() {
	<-threadPool
	return func() {
		threadPool <- struct{}{}
	}
}
