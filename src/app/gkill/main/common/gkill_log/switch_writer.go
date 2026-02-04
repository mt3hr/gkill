package gkill_log

import (
	"io"
	"sync"
	"sync/atomic"
)

// 出力先を実行中に差し替えできるWriter。
// mutexで1回のWriteを直列化するので、統合ログでも行が混ざりにくい。
type SwitchWriter struct {
	w  atomic.Value // stores io.Writer
	mu sync.Mutex
}

func NewSwitchWriter(initial io.Writer) *SwitchWriter {
	sw := &SwitchWriter{}
	sw.w.Store(initial)
	return sw
}

func (sw *SwitchWriter) Set(w io.Writer) {
	sw.w.Store(w)
}

func (sw *SwitchWriter) Write(p []byte) (int, error) {
	w := sw.w.Load().(io.Writer)
	sw.mu.Lock()
	defer sw.mu.Unlock()
	return w.Write(p)
}
