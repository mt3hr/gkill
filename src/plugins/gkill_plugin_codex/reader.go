package main

import (
	"bufio"
	"bytes"
	"io"
)

// Codexのロールアウトログは1行が極端に大きくなる。
//
// 実データ(52ファイル・254,737,956バイト・24,234行)の実測:
//   - 最大の1行は 19,912,604 バイト(response_item/custom_tool_call_output)
//   - 1MBを超える行が77行
//   - しかし保持したい種別の行は合計 13.50MB / 254.75MB = 5.30% しかない
//   - 保持対象の最大行は 260,614 バイト
//
// bufio.Scanner が使えないのは当然として、bufio.Reader.ReadString('\n') でも
// 捨てるだけの19.9MBを文字列に起こしてしまう。ビルダは並列に回るので、
// これをそのままやるとメモリが跳ねる。
//
// そこで「行の先頭だけ覗いて種別を決め、要らない行は改行まで溜めずに読み捨てる」。
const (
	// readBufBytes は bufio.Reader のバッファ。
	// Peek で先頭を覗くので headScanBytes より確実に大きくする。
	readBufBytes = 128 * 1024

	// headScanBytes は種別判定に見る先頭バイト数。
	// 実データでは外側の "type" は必ず40バイト目にあり、payload の "type" は
	// その直後に来る(24,218行中23,293行)ので、512もあれば桁で足りる。
	headScanBytes = 512

	// maxRecordBytes は保持する1レコードの上限。
	// 実データで保持対象の最大行は260,614バイトなので8倍の余裕がある。
	// 超えた行は dropped として数え、設定画面に出す。
	maxRecordBytes = 2 << 20
)

// recordHead は行の先頭だけから分かること。
type recordHead struct {
	// Outer は外側の type。取り出せなかったときは空。
	Outer string
	// Payload は payload.type。その種別が payload.type を持たないとき、
	// および先頭 headScanBytes 内に見つからなかったときは空。
	Payload string
}

// Kind は "外側/payload" の形にした種別。判定に使う。
func (h recordHead) Kind() string {
	if h.Payload == "" {
		return h.Outer
	}
	return h.Outer + "/" + h.Payload
}

// lineReader は改行区切りのレコードを、要らない行を溜め込まずに読む。
type lineReader struct {
	reader *bufio.Reader
	buf    []byte
}

func newLineReader(r io.Reader) *lineReader {
	return &lineReader{reader: bufio.NewReaderSize(r, readBufBytes)}
}

// next は次のレコードを返す。
//
// keep が false を返した行は改行まで読み捨てて data に nil を返す。
// data は次に next を呼ぶまでしか有効でない。持ち回すならその場で複製すること。
// dropped は maxRecordBytes を超えたので中身を捨てたことを表す(head は有効)。
// 末尾では err に io.EOF を返す。
func (lr *lineReader) next(keep func(recordHead) bool) (head recordHead, data []byte, dropped bool, err error) {
	chunk, peekErr := lr.reader.Peek(headScanBytes)
	if len(chunk) == 0 {
		if peekErr == nil {
			peekErr = io.EOF
		}
		return recordHead{}, nil, false, peekErr
	}

	head = classifyHead(chunk)
	if keep != nil && !keep(head) {
		return head, nil, false, lr.drainLine()
	}

	lr.buf = lr.buf[:0]
	for {
		fragment, readErr := lr.reader.ReadSlice('\n')
		if room := maxRecordBytes - len(lr.buf); room > 0 {
			if len(fragment) > room {
				lr.buf = append(lr.buf, fragment[:room]...)
				dropped = true
			} else {
				lr.buf = append(lr.buf, fragment...)
			}
		} else if len(fragment) > 0 {
			dropped = true
		}

		if readErr == bufio.ErrBufferFull {
			// 行がバッファより長い。まだ続きがある
			continue
		}
		if readErr != nil {
			if readErr == io.EOF && len(lr.buf) != 0 {
				// 最終行に改行が無いだけ。次の呼び出しで EOF を返す
				break
			}
			return head, nil, dropped, readErr
		}
		break
	}

	return head, bytes.TrimRight(lr.buf, "\r\n"), dropped, nil
}

// drainLine は改行まで1バイトも溜めずに読み飛ばす。
func (lr *lineReader) drainLine() error {
	for {
		_, err := lr.reader.ReadSlice('\n')
		if err == bufio.ErrBufferFull {
			continue
		}
		if err == io.EOF {
			// 最終行に改行が無いだけ。次の Peek が EOF を返す
			return nil
		}
		return err
	}
}

// bufferCap は保持用バッファの実際の確保量を返す。テスト用。
func (lr *lineReader) bufferCap() int { return cap(lr.buf) }

var (
	keyType    = []byte(`"type":"`)
	keyPayload = []byte(`"payload":`)
)

// classifyHead は行の先頭チャンクから (外側type, payload.type) を取り出す。
//
// 完全なJSONパースはしない。19.9MBの行を捨てるかどうかを決めるためだけの判定なので、
// 先頭 headScanBytes しか見ない。
// 取り出せなかったときは空文字を返す —— 呼び出し側は「分からなかったら拾う」こと。
// 分からないものを捨てると、キー順が変わった日に会話が静かに消える。
func classifyHead(chunk []byte) recordHead {
	limit := len(chunk)
	if limit > headScanBytes {
		limit = headScanBytes
	}
	if i := bytes.IndexByte(chunk[:limit], '\n'); i >= 0 {
		limit = i
	}
	head := chunk[:limit]

	var result recordHead

	payloadAt := bytes.Index(head, keyPayload)
	typeAt := bytes.Index(head, keyType)

	// payload より前にある "type" だけが外側のもの。
	// payload が先に来る並びでは外側を確定できないので空のままにする。
	if typeAt >= 0 && (payloadAt < 0 || typeAt < payloadAt) {
		result.Outer = jsonStringAt(head, typeAt+len(keyType))
	}

	if payloadAt < 0 {
		return result
	}
	inner := head[payloadAt+len(keyPayload):]
	if innerType := bytes.Index(inner, keyType); innerType >= 0 {
		result.Payload = jsonStringAt(inner, innerType+len(keyType))
	}
	return result
}

// jsonStringAt は開始位置から閉じ引用符までを返す。
// 種別名にエスケープは現れないので、最初の " で切る。
func jsonStringAt(b []byte, start int) string {
	if start >= len(b) {
		return ""
	}
	end := bytes.IndexByte(b[start:], '"')
	if end < 0 {
		return ""
	}
	return string(b[start : start+end])
}
