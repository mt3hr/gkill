package reps

import (
	"strings"
	"sync"
)

// pluginStderrRingSize はプラグインstderrの保持量。
// 直近のエラー1〜2件が読めれば診断には足りる（全量は従来どおり本体のstderrへ流れる）。
const pluginStderrRingSize = 4 * 1024

// pluginStderrRing はプラグインプロセスの stderr 末尾を保持するリングバッファ。
//
// 以前は cmd.Stderr = os.Stderr の直結で、プラグインのビルドエラー
// （"build error: conversations.json が見つかりません" 等）はサーバのコンソールにしか
// 出ず、get_plugin_list からは「is_alive=true なのに0件」の理由が一切診断できなかった
// （外部監査 D2）。io.MultiWriter でここへも写し、PluginInfo.last_error として返す。
//
// io.Writer として cmd.Stderr に渡されるため、書き込みはプラグインプロセスの
// 出力goroutineから並行に来る。mutexで守る。
type pluginStderrRing struct {
	mu  sync.Mutex
	buf []byte
	// full はリングが一周したか（Tail の組み立て順に効く）。
	full bool
	// next は次に書く位置。
	next int
}

func newPluginStderrRing() *pluginStderrRing {
	return &pluginStderrRing{buf: make([]byte, pluginStderrRingSize)}
}

// Write は io.Writer 実装。常に全量を受け取ったことにする（stderrを詰まらせない）。
func (r *pluginStderrRing) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	written := len(p)
	// リングより大きい書き込みは末尾だけ意味を持つ
	if len(p) > len(r.buf) {
		p = p[len(p)-len(r.buf):]
	}
	for len(p) > 0 {
		n := copy(r.buf[r.next:], p)
		p = p[n:]
		r.next += n
		if r.next == len(r.buf) {
			r.next = 0
			r.full = true
		}
	}
	return written, nil
}

// Tail は保持している末尾を文字列で返す。空なら""。
// 一周している場合は途中で切れた先頭行を捨てて、行境界から返す。
func (r *pluginStderrRing) Tail() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	var tail string
	if !r.full {
		tail = string(r.buf[:r.next])
	} else {
		tail = string(r.buf[r.next:]) + string(r.buf[:r.next])
		// 一周ぶんの先頭は行の途中から始まっている可能性が高いので、最初の改行まで捨てる
		if idx := strings.IndexByte(tail, '\n'); idx >= 0 && idx+1 < len(tail) {
			tail = tail[idx+1:]
		}
	}
	return strings.TrimSpace(tail)
}
