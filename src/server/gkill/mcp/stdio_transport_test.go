package mcp

// StdioTransport の枠組み（NDJSON と Content-Length）の取り出し方。
//
// stdio_e2e_test.go は子プロセスを立てて端から端まで見るが、あちらは「どう届くか」を選べない。
// NDJSON の行と Content-Length 枠が1回の Read にまとまって届くかどうかは OS とタイミング次第で、
// まとまったときだけ壊れる実装だと CI だけが落ちる。ここでは届き方を固定して判定を確かめる。

import (
	"bytes"
	"io"
	"strconv"
	"strings"
	"testing"
)

// oneChunkReader は Read 1回で全部渡し、次の Read で EOF を返す。
// 「行と枠が同じチャンクで届く」を再現するためのもの。
type oneChunkReader struct {
	data []byte
	done bool
}

func (r *oneChunkReader) Read(p []byte) (int, error) {
	if r.done {
		return 0, io.EOF
	}
	n := copy(p, r.data)
	if n < len(r.data) {
		r.data = r.data[n:]
		return n, nil
	}
	r.done = true
	return n, nil
}

// runStdioInput は入力を流し、stdout へ出た NDJSON の行と stderr を返す。
func runStdioInput(t *testing.T, input string, oneChunk bool) ([]string, string) {
	t.Helper()
	// ping はサーバへ往復しないので、gkill へは繋がらない構成でよい。
	client := NewGkillClient(ClientConfig{BaseURL: "http://127.0.0.1:1", UserID: "testuser", PasswordSha256: goldenPasswordSha256, Locale: "ja"})
	server := NewReadServer(client, nil)
	var out, errOut bytes.Buffer
	var in io.Reader
	if oneChunk {
		in = &oneChunkReader{data: []byte(input)}
	} else {
		in = strings.NewReader(input)
	}
	transport := NewStdioTransportWith(server, in, &out, &errOut)
	if err := transport.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	lines := []string{}
	for _, line := range strings.Split(out.String(), "\n") {
		if strings.TrimSpace(line) != "" {
			lines = append(lines, line)
		}
	}
	return lines, errOut.String()
}

func lspFrame(body string) string {
	return "Content-Length: " + strconv.Itoa(len(body)) + "\r\n\r\n" + body
}

// NDJSON の通知のうしろに Content-Length 枠が続き、同じチャンクで届いても両方処理される。
// 先に "\r\n\r\n" を探す実装だと、通知の行ごと1つのヘッダ塊と見なして両方落としていた
// （stderr に invalid content-length header が出て、ping の応答が返らない）。
func TestStdioOnDataHandlesNdjsonFollowedByContentLengthInOneChunk(t *testing.T) {
	input := `{"jsonrpc":"2.0","method":"notifications/initialized"}` + "\n" +
		lspFrame(`{"jsonrpc":"2.0","id":"c-1","method":"ping"}`)

	for _, oneChunk := range []bool{true, false} {
		lines, stderr := runStdioInput(t, input, oneChunk)
		if strings.Contains(stderr, "invalid content-length header") {
			t.Errorf("oneChunk=%v: ヘッダの取り違えが起きた: %s", oneChunk, stderr)
		}
		if len(lines) != 1 {
			t.Fatalf("oneChunk=%v: 応答は1件のはず: %v（stderr: %s）", oneChunk, lines, stderr)
		}
		if !strings.Contains(lines[0], `"c-1"`) {
			t.Errorf("oneChunk=%v: ping の応答が返っていない: %s", oneChunk, lines[0])
		}
	}
}

// Content-Length 枠が続けて2つ来ても、通知を挟んでも、順に処理される。
func TestStdioOnDataHandlesMixedFramingSequence(t *testing.T) {
	input := lspFrame(`{"jsonrpc":"2.0","id":1,"method":"ping"}`) +
		`{"jsonrpc":"2.0","method":"notifications/initialized"}` + "\n" +
		lspFrame(`{"jsonrpc":"2.0","id":2,"method":"ping"}`) +
		`{"jsonrpc":"2.0","id":3,"method":"ping"}` + "\n"

	lines, stderr := runStdioInput(t, input, true)
	if len(lines) != 3 {
		t.Fatalf("応答は3件のはず: %v（stderr: %s）", lines, stderr)
	}
	for _, want := range []string{`"id":1`, `"id":2`, `"id":3`} {
		found := false
		for _, line := range lines {
			if strings.Contains(line, want) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("%s の応答が無い: %v", want, lines)
		}
	}
}

// JSON でもヘッダでもない行は捨てて警告し、次の要求は普通に処理する。
func TestStdioOnDataDropsJunkLineAndKeepsGoing(t *testing.T) {
	input := "ただのゴミ行\n" + `{"jsonrpc":"2.0","id":9,"method":"ping"}` + "\n"
	lines, stderr := runStdioInput(t, input, true)
	if len(lines) != 1 || !strings.Contains(lines[0], `"id":9`) {
		t.Fatalf("ゴミ行のあとの要求が処理されていない: %v（stderr: %s）", lines, stderr)
	}
	if !strings.Contains(stderr, "invalid json line") {
		t.Errorf("ゴミ行の警告が stderr に無い: %s", stderr)
	}
}

// 壊れた Content-Length は警告して捨てる（従来どおり）。
func TestStdioOnDataWarnsOnBrokenContentLength(t *testing.T) {
	input := "Content-Length: いくつか\r\n\r\n{}"
	_, stderr := runStdioInput(t, input, true)
	if !strings.Contains(stderr, "invalid content-length header") {
		t.Errorf("壊れたヘッダの警告が出ていない: %s", stderr)
	}
}
