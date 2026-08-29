package gkill_log

import (
	"fmt"
	"io"
	"os"
	"sync"
)

// FileSink は1つの出力先（レベル別ファイル・統合ファイル・stdout）。
//
// 自身が io.Writer なのは、書き込みの直前にサイズを見て世代を回すため。
// SwitchWriter を直に leaf handler へ渡していた頃は回転する隙が無く、
// ログファイルが上限なく育った（既定レベルが none で誰も気付かなかった）。
type FileSink struct {
	sw   *SwitchWriter
	mu   sync.Mutex
	cur  io.Closer
	path string

	// size は path に対して書いた累計バイト数。SetFile のときに実ファイルから取り直す。
	size int64

	// maxBytes が 0 以下なら回転しない（stdout など、パスを持たない sink を含む）。
	maxBytes int64
	// keep は残す世代数。path.1 .. path.keep。0 以下なら退避せず捨てる。
	keep int
}

func NewFileSink(initial io.Writer) *FileSink {
	return &FileSink{sw: NewSwitchWriter(initial)}
}

// Writer は leaf handler が書き込む先を返す。
// **fs.sw ではなく fs 自身を返すこと。** sw を直接渡すと Write を素通りして回転が効かない。
func (fs *FileSink) Writer() io.Writer { return fs }

// SetRotation は回転の条件を決める。SetFile より前に呼ぶこと。
func (fs *FileSink) SetRotation(maxBytes int64, keep int) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	fs.maxBytes = maxBytes
	fs.keep = keep
}

func (fs *FileSink) Write(p []byte) (int, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if fs.shouldRotateLocked(len(p)) {
		// 回転に失敗しても書き込みは続ける。ログの都合で本体を止めない。
		_ = fs.rotateLocked()
	}
	n, err := fs.sw.Write(p)
	fs.size += int64(n)
	return n, err
}

func (fs *FileSink) shouldRotateLocked(incoming int) bool {
	if fs.path == "" || fs.maxBytes <= 0 {
		return false
	}
	// 空ファイルは回さない。1回の書き込みが上限を超える場合に無限に回るのを防ぐ。
	if fs.size == 0 {
		return false
	}
	return fs.size+int64(incoming) > fs.maxBytes
}

// rotateLocked は path.keep を捨て、古い世代を1つずつ後ろへずらし、path を path.1 にする。
//
// **Windowsでは開いたままのファイルをリネームできない。** 先に Close してから rename し、
// そのあとで開き直すこと。順序を変えると回転のたびに失敗し、ファイルが育ち続ける。
func (fs *FileSink) rotateLocked() error {
	path := fs.path
	if fs.cur != nil {
		_ = fs.cur.Close()
		fs.cur = nil
	}

	if fs.keep <= 0 {
		_ = os.Remove(path)
		return fs.openLocked(path)
	}

	_ = os.Remove(fmt.Sprintf("%s.%d", path, fs.keep))
	for i := fs.keep - 1; i >= 1; i-- {
		_ = os.Rename(fmt.Sprintf("%s.%d", path, i), fmt.Sprintf("%s.%d", path, i+1))
	}
	if err := os.Rename(path, path+".1"); err != nil {
		// リネームできなかったときは、開き直して書き込みを続ける（欠測を作らない）。
		if openErr := fs.openLocked(path); openErr != nil {
			return openErr
		}
		return fmt.Errorf("rotate log file: %w", err)
	}
	return fs.openLocked(path)
}

func (fs *FileSink) SetFile(path string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	return fs.openLocked(path)
}

// openLocked は path を開いて sink の出力先にし、size を実ファイルの大きさへ合わせる。
func (fs *FileSink) openLocked(path string) error {
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

	fs.size = 0
	if stat, statErr := f.Stat(); statErr == nil {
		fs.size = stat.Size()
	}
	return nil
}

func (fs *FileSink) Close() error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	if fs.cur != nil {
		err := fs.cur.Close()
		fs.cur = nil
		fs.path = ""
		fs.size = 0
		return err
	}
	return nil
}

func (fs *FileSink) Path() string {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	return fs.path
}
