package mcp

// stdio トランスポートの端から端まで（テストバイナリ自身を子プロセスとして起動する）。
//
// 旧実装には無かった穴: stdio の枠組み（NDJSON / Content-Length）と「stdout に JSON-RPC 以外を出さない」は
// 単体テストでは守れない（StdioTransport は os.Stdout を直接使う）。TestMain が環境変数を見て
// 子プロセス側では読み取りサーバとして振る舞う。

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/mcp/internal/fakegkill"
	"github.com/mt3hr/gkill/src/server/gkill/mcp/jsonobj"
)

const envStdioServerMode = "GKILL_MCP_TEST_STDIO_SERVER"

func TestMain(m *testing.M) {
	if os.Getenv(envStdioServerMode) == "1" {
		runStdioServerForTest()
		return
	}
	os.Exit(m.Run())
}

// runStdioServerForTest は子プロセス側: 環境変数から gkill へ繋ぐ read サーバを stdio で動かす。
func runStdioServerForTest() {
	client := NewGkillClientFromEnv()
	server := NewReadServer(client, nil)
	server.CurrentUserID = client.UserID()
	if err := NewStdioTransport(server).Start(); err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}

func TestStdioTransportEndToEnd(t *testing.T) {
	fake := fakegkill.New()
	upstream := httptest.NewServer(fake)
	t.Cleanup(upstream.Close)

	exe, err := os.Executable()
	expectNoError(t, err)
	cmd := exec.Command(exe, "-test.run=^$")
	cmd.Env = append(os.Environ(),
		envStdioServerMode+"=1",
		"GKILL_BASE_URL="+upstream.URL,
		"GKILL_USER=testuser",
		"GKILL_PASSWORD_SHA256="+goldenPasswordSha256,
	)
	stdin, err := cmd.StdinPipe()
	expectNoError(t, err)
	stdout, err := cmd.StdoutPipe()
	expectNoError(t, err)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	expectNoError(t, cmd.Start())

	lines := make(chan string, 16)
	go func() {
		scanner := bufio.NewScanner(stdout)
		scanner.Buffer(make([]byte, 1024*1024), 16*1024*1024)
		for scanner.Scan() {
			lines <- scanner.Text()
		}
		close(lines)
	}()
	readLine := func() string {
		select {
		case line, ok := <-lines:
			if !ok {
				t.Fatalf("stdout closed early; stderr: %s", stderr.String())
			}
			return line
		case <-time.After(20 * time.Second):
			t.Fatalf("timed out waiting for a response line; stderr: %s", stderr.String())
		}
		return ""
	}

	// 1. NDJSON の initialize
	_, err = io.WriteString(stdin, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`+"\n")
	expectNoError(t, err)
	first := parseObj(t, readLine())
	expectEqual(t, first.Value("id"), 1)
	expectEqual(t, objAt(t, first, "result").Value("protocolVersion"), "2024-11-05")

	// 2. 通知（応答なし）に続けて Content-Length 枠の tools/call（gkill へ往復する）
	_, err = io.WriteString(stdin, `{"jsonrpc":"2.0","method":"notifications/initialized"}`+"\n")
	expectNoError(t, err)
	body := `{"jsonrpc":"2.0","id":"c-1","method":"tools/call","params":{"name":"gkill_get_mi_board_list","arguments":{}}}`
	_, err = io.WriteString(stdin, "Content-Length: "+itoa(len(body))+"\r\n\r\n"+body)
	expectNoError(t, err)
	second := parseObj(t, readLine())
	expectEqual(t, second.Value("id"), "c-1")
	boards := arrAt(t, objAt(t, second, "result", "structuredContent"), "boards")
	expectEqual(t, boards, strs("Inbox", "Work", "買い物"))

	// 3. 壊れた行は捨てられ（stderr に警告）、次の要求は普通に処理される
	_, err = io.WriteString(stdin, "{not json}\n"+`{"jsonrpc":"2.0","id":3,"method":"ping"}`+"\n")
	expectNoError(t, err)
	third := parseObj(t, readLine())
	expectEqual(t, third.Value("id"), 3)

	expectNoError(t, stdin.Close())
	expectNoError(t, cmd.Wait())
	// stdout に JSON-RPC 以外の行が1つも無かった（残りの行があれば全部 JSON として読める）
	for line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var probe map[string]any
		if err := json.Unmarshal([]byte(line), &probe); err != nil {
			t.Fatalf("non JSON-RPC output on stdout: %q", line)
		}
	}
	mustContain(t, stderr.String(), "invalid json line")
	// gkill へは login と get_mi_board_list だけが届いた
	paths := []string{}
	for _, rec := range fake.Drain() {
		paths = append(paths, rec.Path)
	}
	expectEqual(t, jsonobj.Strings(paths...), strs("/api/login", "/api/get_mi_board_list"))
}
