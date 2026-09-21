package mcp

// stdio (標準入出力) の JSON-RPC トランスポート。3つの MCP サーバで共有する（旧 stdio-transport.mjs）。
//
// LSP 形式 (Content-Length ヘッダ) と NDJSON (1行1メッセージ) の両方を受け、出力は NDJSON のみ。
// stdout はプロトコルのチャネルなので、ログは stderr と gkill_log へ出す。

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/mt3hr/gkill/src/server/gkill/mcp/jsonobj"
)

// StdioTransport は stdin から JSON-RPC を読み、stdout へ NDJSON で書く。
type StdioTransport struct {
	server *Server
	in     io.Reader
	out    io.Writer
	errOut io.Writer
	buffer []byte
	outMu  sync.Mutex
	wg     sync.WaitGroup
}

// NewStdioTransport は os.Stdin / os.Stdout / os.Stderr を使うトランスポート。
func NewStdioTransport(server *Server) *StdioTransport {
	return NewStdioTransportWith(server, os.Stdin, os.Stdout, os.Stderr)
}

// NewStdioTransportWith は入出力を差し替えられる（テストとゴールデン採取用）。
func NewStdioTransportWith(server *Server, in io.Reader, out io.Writer, errOut io.Writer) *StdioTransport {
	// stdio で話す相手はこのマシン上のプロセスなので、ファイルの絶対パスを渡してよい
	server.IsLocalTransport = true
	return &StdioTransport{server: server, in: in, out: out, errOut: errOut}
}

// Start は stdin を読み切るまでブロックし、処理中の要求を待ってから返る。
func (t *StdioTransport) Start() error {
	chunk := make([]byte, 64*1024)
	for {
		n, err := t.in.Read(chunk)
		if n > 0 {
			t.onData(chunk[:n])
		}
		if err != nil {
			if err != io.EOF {
				t.logError("stdin error", err)
			}
			break
		}
	}
	t.wg.Wait()
	return nil
}

// **重さで呼び分けること。** stdin が壊れた（プロセスが続けられない）のと、
// 相手が壊れた1行を送ってきた（こちらは動き続ける）のを同じ見た目で出すと、
// ログから深刻度が読めない。
func (t *StdioTransport) logError(message string, err any) { t.log("error", message, err) }

func (t *StdioTransport) logWarn(message string, err any) { t.log("warn", message, err) }

func (t *StdioTransport) log(level string, message string, err any) {
	text := jsString(err)
	if e, ok := err.(error); ok {
		text = e.Error()
	}
	fmt.Fprintf(t.errOut, "%s: %s\n", message, text)
	event := strings.ReplaceAll(message, " ", "_")
	if level == "error" {
		t.server.Log.Error(event, "error", text)
	} else {
		t.server.Log.Warn(event, "error", text)
	}
}

// WriteMessage は1メッセージを NDJSON で書く。
func (t *StdioTransport) WriteMessage(message any) {
	t.outMu.Lock()
	defer t.outMu.Unlock()
	_, _ = io.WriteString(t.out, jsonobj.MarshalString(message)+"\n")
}

func (t *StdioTransport) dispatch(message any) {
	t.wg.Add(1)
	go func() {
		defer t.wg.Done()
		response := t.dispatchSafely(message)
		if response != nil {
			t.WriteMessage(response)
		}
	}()
}

func (t *StdioTransport) dispatchSafely(message any) (response any) {
	defer func() {
		if recovered := recover(); recovered != nil {
			t.logError("unhandled request error", fmt.Errorf("panic: %v", recovered))
			if o, ok := message.(*jsonobj.Object); ok && o != nil && o.Has("id") {
				response = jsonobj.Obj("jsonrpc", "2.0", "id", o.Value("id"), "error", jsonobj.Obj("code", -32603, "message", "Internal error"))
			} else {
				response = nil
			}
		}
	}()
	return t.server.HandlePayload(context.Background(), message, nil)
}

// onData は届いたぶんをバッファへ足し、取り出せるメッセージを順に処理する。
//
// **枠組みはバッファの先頭1行で決めること。** 先に "\r\n\r\n" を探す書き方だと、
// NDJSON の行のうしろに Content-Length 枠が続き、両方が1回の Read にまとまって届いたときに、
// 行の中身ごと1つのヘッダ塊と見なしてしまう。ヘッダ名を切り出す ":" が JSON の中の
// `"jsonrpc":` に当たるので content-length が見つからず、「invalid content-length header」で
// バッファを丸ごと捨てる。行と枠が別々の Read で届けば通るので、**届き方で結果が変わる**。
func (t *StdioTransport) onData(chunk []byte) {
	t.buffer = append(t.buffer, chunk...)

	for {
		t.trimLeadingBlanks()
		if len(t.buffer) == 0 {
			return
		}
		lf := bytes.IndexByte(t.buffer, '\n')

		// NDJSON-style framing: one JSON-RPC message per line.
		if t.startsWithJSON() {
			if lf == -1 {
				return
			}
			line := strings.TrimSpace(string(t.buffer[:lf]))
			t.buffer = append([]byte{}, t.buffer[lf+1:]...)
			if line == "" {
				continue
			}
			message, err := jsonobj.Unmarshal([]byte(line))
			if err != nil {
				// 不正な行を黙って捨てると、クライアント側の枠組みの不具合を追えない
				t.logWarn("invalid json line", err)
				continue
			}
			t.dispatch(message)
			continue
		}

		// LSP-style framing: "Content-Length: N\r\n\r\n{...}"
		if t.startsWithHeaderLine(lf) {
			headerEnd := bytes.Index(t.buffer, []byte("\r\n\r\n"))
			if headerEnd == -1 {
				return // ヘッダの終わりがまだ届いていない
			}
			headerText := string(t.buffer[:headerEnd])
			contentLength := -1
			for _, line := range strings.Split(headerText, "\r\n") {
				idx := strings.Index(line, ":")
				if idx == -1 {
					continue
				}
				key := strings.ToLower(strings.TrimSpace(line[:idx]))
				value := strings.TrimSpace(line[idx+1:])
				if key == "content-length" {
					n, err := strconv.Atoi(value)
					if err != nil {
						contentLength = -1
					} else {
						contentLength = n
					}
				}
			}
			if contentLength < 0 {
				t.logWarn("invalid content-length header", headerText)
				t.buffer = nil
				return
			}
			totalLength := headerEnd + 4 + contentLength
			if len(t.buffer) < totalLength {
				return
			}
			body := t.buffer[headerEnd+4 : totalLength]
			t.buffer = append([]byte{}, t.buffer[totalLength:]...)
			message, err := jsonobj.Unmarshal(body)
			if err != nil {
				t.logWarn("invalid json body", err)
				continue
			}
			t.dispatch(message)
			continue
		}

		// JSON でもヘッダでもない行。捨てて警告する（枠組みの不具合を追えるように）。
		if lf == -1 {
			return
		}
		junk := strings.TrimSpace(string(t.buffer[:lf]))
		t.buffer = append([]byte{}, t.buffer[lf+1:]...)
		if junk != "" {
			t.logWarn("invalid json line", fmt.Errorf("JSON-RPC のメッセージでも Content-Length ヘッダでもない行: %q", junk))
		}
	}
}

// trimLeadingBlanks はバッファ先頭の空白・空行を落とす（枠組みの判定を先頭1バイトで行うため）。
func (t *StdioTransport) trimLeadingBlanks() {
	i := 0
	for i < len(t.buffer) {
		c := t.buffer[i]
		if c == ' ' || c == '\t' || c == '\r' || c == '\n' {
			i++
			continue
		}
		break
	}
	if i > 0 {
		t.buffer = append([]byte{}, t.buffer[i:]...)
	}
}

// startsWithJSON はバッファの先頭が JSON-RPC メッセージ（NDJSON の1行）かを見る。
func (t *StdioTransport) startsWithJSON() bool {
	return len(t.buffer) > 0 && (t.buffer[0] == '{' || t.buffer[0] == '[')
}

// startsWithHeaderLine は先頭行が "Name: value" 形のヘッダかを見る。
// lf は先頭の改行位置で、-1 は「行がまだ完結していない」。
func (t *StdioTransport) startsWithHeaderLine(lf int) bool {
	end := lf
	if end == -1 {
		end = len(t.buffer)
	}
	line := string(t.buffer[:end])
	idx := strings.Index(line, ":")
	if idx <= 0 {
		// 行が未完なら、続きがヘッダになりうるものとして待つ
		return lf == -1
	}
	for _, c := range line[:idx] {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '-' || c == '_' {
			continue
		}
		return false
	}
	return true
}
