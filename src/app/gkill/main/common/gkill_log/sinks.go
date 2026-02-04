package gkill_log

import (
	"fmt"
	"io"
	"os"
	"sync"
)

type FileSink struct {
	sw   *SwitchWriter
	mu   sync.Mutex
	cur  io.Closer
	path string
}

func NewFileSink(initial io.Writer) *FileSink {
	return &FileSink{sw: NewSwitchWriter(initial)}
}

func (fs *FileSink) Writer() io.Writer { return fs.sw }

func (fs *FileSink) SetFile(path string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}

	// 切替を先に行い、その後に旧ファイルをClose（欠け/競合を減らす）
	fs.sw.Set(f)
	if fs.cur != nil {
		_ = fs.cur.Close()
	}
	fs.cur = f
	fs.path = path
	return nil
}

func (fs *FileSink) Close() error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	if fs.cur != nil {
		err := fs.cur.Close()
		fs.cur = nil
		fs.path = ""
		return err
	}
	return nil
}

func (fs *FileSink) Path() string {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	return fs.path
}
