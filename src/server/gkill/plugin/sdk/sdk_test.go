package sdk

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// runLoop は Run から stdin/stdout を切り出したメッセージループ。
// プラグイン作者はこのループの応答形状に依存しているので、
// コマンドごとの応答とエラー時のフォールバックをここで固定する。

// runLoopWith は1行以上のリクエストをループに流し、返ってきたレスポンスを順に返す。
func runLoopWith(t *testing.T, h Handler, cfg Config, pluginDir string, requests ...pluginRequest) []pluginResponse {
	t.Helper()

	var in bytes.Buffer
	enc := json.NewEncoder(&in)
	for _, req := range requests {
		if err := enc.Encode(req); err != nil {
			t.Fatalf("encode request: %v", err)
		}
	}

	var out bytes.Buffer
	runLoop(h, cfg, pluginDir, "testuser", &in, &out)

	responses := []pluginResponse{}
	dec := json.NewDecoder(&out)
	for dec.More() {
		var resp pluginResponse
		if err := dec.Decode(&resp); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		responses = append(responses, resp)
	}
	return responses
}

// runLoopOnce は1リクエストを流してレスポンスを1件返す。
func runLoopOnce(t *testing.T, h Handler, req pluginRequest) pluginResponse {
	t.Helper()
	responses := runLoopWith(t, h, Config{}, t.TempDir(), req)
	if len(responses) != 1 {
		t.Fatalf("レスポンス件数 = %d, want 1: %+v", len(responses), responses)
	}
	return responses[0]
}

func TestRunLoop_Ping(t *testing.T) {
	resp := runLoopOnce(t, Handler{}, pluginRequest{ID: "req-1", Command: "ping"})

	if resp.ID != "req-1" {
		t.Errorf("ID = %q, want %q（リクエストとレスポンスの対応が取れていない）", resp.ID, "req-1")
	}
	if !resp.Pong {
		t.Error("Pong = false, want true")
	}
}

func TestRunLoop_GetRepName(t *testing.T) {
	resp := runLoopOnce(t, Handler{RepName: "MyPlugin"}, pluginRequest{ID: "req-1", Command: "get_rep_name"})

	if resp.RepName != "MyPlugin" {
		t.Errorf("RepName = %q, want %q", resp.RepName, "MyPlugin")
	}
}

func TestRunLoop_FindKyous(t *testing.T) {
	var gotQuery Query
	h := Handler{
		FindKyous: func(_ context.Context, q Query, _ Config) ([]Kyou, error) {
			gotQuery = q
			return []Kyou{{ID: "kyou-1"}, {ID: "kyou-2"}}, nil
		},
	}

	resp := runLoopOnce(t, h, pluginRequest{
		ID:      "req-1",
		Command: "find_kyous",
		Query:   &pluginQuery{Words: []string{"alpha"}, WordsAnd: true, Limit: 10},
	})

	if len(resp.Kyous) != 2 {
		t.Fatalf("Kyous件数 = %d, want 2", len(resp.Kyous))
	}
	// pluginQuery → Query の変換が効いていること
	if len(gotQuery.Words) != 1 || gotQuery.Words[0] != "alpha" {
		t.Errorf("Words = %v, want [alpha]", gotQuery.Words)
	}
	if !gotQuery.WordsAnd {
		t.Error("WordsAnd = false, want true")
	}
	if gotQuery.Limit != 10 {
		t.Errorf("Limit = %d, want 10", gotQuery.Limit)
	}
}

// TestRunLoop_FindKyousNotImplemented は、FindKyous 未実装のプラグインが
// 落ちずにエラーレスポンスを返すことを確認する。
func TestRunLoop_FindKyousNotImplemented(t *testing.T) {
	resp := runLoopOnce(t, Handler{}, pluginRequest{ID: "req-1", Command: "find_kyous"})

	if len(resp.Errors) == 0 {
		t.Fatal("FindKyous未実装なのにエラーが返っていない")
	}
	if !strings.Contains(resp.Errors[0], "find_kyous") {
		t.Errorf("エラーメッセージ = %q, want find_kyous を含む", resp.Errors[0])
	}
}

// TestRunLoop_FindKyousError は、ハンドラが返したエラーが
// レスポンスのerrorsに載り、ループが継続することを確認する。
func TestRunLoop_FindKyousError(t *testing.T) {
	h := Handler{
		FindKyous: func(_ context.Context, _ Query, _ Config) ([]Kyou, error) {
			return nil, errors.New("外部APIに繋がらない")
		},
	}

	responses := runLoopWith(t, h, Config{}, t.TempDir(),
		pluginRequest{ID: "req-1", Command: "find_kyous"},
		pluginRequest{ID: "req-2", Command: "ping"},
	)

	if len(responses) != 2 {
		t.Fatalf("レスポンス件数 = %d, want 2（エラーでループが止まっている）", len(responses))
	}
	if len(responses[0].Errors) == 0 || !strings.Contains(responses[0].Errors[0], "外部APIに繋がらない") {
		t.Errorf("errors = %v, want ハンドラのエラーメッセージ", responses[0].Errors)
	}
	if !responses[1].Pong {
		t.Error("エラーの後のリクエストが処理されていない")
	}
}

// TestRunLoop_GetKyouFallsBackToFindKyous は、GetKyou 未実装のときに
// FindKyous の結果から該当IDを拾うフォールバックが効くことを確認する。
func TestRunLoop_GetKyouFallsBackToFindKyous(t *testing.T) {
	h := Handler{
		FindKyous: func(_ context.Context, _ Query, _ Config) ([]Kyou, error) {
			return []Kyou{{ID: "kyou-1"}, {ID: "kyou-2"}}, nil
		},
	}

	t.Run("見つかる", func(t *testing.T) {
		resp := runLoopOnce(t, h, pluginRequest{ID: "req-1", Command: "get_kyou", KyouID: "kyou-2"})
		if resp.Kyou == nil {
			t.Fatal("Kyou = nil, want kyou-2")
		}
		if resp.Kyou.ID != "kyou-2" {
			t.Errorf("Kyou.ID = %q, want %q", resp.Kyou.ID, "kyou-2")
		}
	})

	t.Run("見つからない", func(t *testing.T) {
		resp := runLoopOnce(t, h, pluginRequest{ID: "req-1", Command: "get_kyou", KyouID: "unknown"})
		if resp.Kyou != nil {
			t.Errorf("Kyou = %+v, want nil", resp.Kyou)
		}
		if len(resp.Errors) != 0 {
			t.Errorf("見つからないだけでエラーになっている: %v", resp.Errors)
		}
	})
}

// TestRunLoop_GetContentHTMLDefault は、GetContentHTML 未実装でも
// 既定のHTMLが返りエラーにならないことを確認する。
func TestRunLoop_GetContentHTMLDefault(t *testing.T) {
	resp := runLoopOnce(t, Handler{}, pluginRequest{ID: "req-1", Command: "get_content_html", KyouID: "kyou-9"})

	if len(resp.Errors) != 0 {
		t.Fatalf("既定実装なのにエラーになっている: %v", resp.Errors)
	}
	if !strings.Contains(resp.HTML, "kyou-9") {
		t.Errorf("HTML = %q, want kyou-9 を含む", resp.HTML)
	}
}

// TestRunLoop_PostConfigDefaultSavesForm は、PostConfig 未実装のとき
// フォームの内容がそのまま config.json に保存されることを確認する。
func TestRunLoop_PostConfigDefaultSavesForm(t *testing.T) {
	pluginDir := t.TempDir()

	responses := runLoopWith(t, Handler{}, Config{"existing": "keep"}, pluginDir,
		pluginRequest{ID: "req-1", Command: "post_config", FormData: map[string]string{"source_dirs": "/tmp/logs"}},
	)
	if len(responses) != 1 {
		t.Fatalf("レスポンス件数 = %d, want 1", len(responses))
	}
	if len(responses[0].Errors) != 0 {
		t.Fatalf("post_config errors: %v", responses[0].Errors)
	}

	b, err := os.ReadFile(filepath.Join(pluginDir, "config.json"))
	if err != nil {
		t.Fatalf("config.json が書かれていない: %v", err)
	}
	var saved Config
	if err := json.Unmarshal(b, &saved); err != nil {
		t.Fatalf("config.json の内容が不正: %v", err)
	}
	if saved["source_dirs"] != "/tmp/logs" {
		t.Errorf("source_dirs = %v, want /tmp/logs", saved["source_dirs"])
	}
	if saved["existing"] != "keep" {
		t.Errorf("既存の設定値が失われている: %v", saved)
	}
}

// TestRunLoop_UnknownCommand は、知らないコマンドで落ちずに
// エラーを返して次のリクエストを処理できることを確認する。
// プロトコル拡張時に古いプラグインが即死しないための性質。
func TestRunLoop_UnknownCommand(t *testing.T) {
	responses := runLoopWith(t, Handler{}, Config{}, t.TempDir(),
		pluginRequest{ID: "req-1", Command: "no_such_command"},
		pluginRequest{ID: "req-2", Command: "ping"},
	)

	if len(responses) != 2 {
		t.Fatalf("レスポンス件数 = %d, want 2", len(responses))
	}
	if len(responses[0].Errors) == 0 {
		t.Error("未知コマンドでエラーが返っていない")
	}
	if !responses[1].Pong {
		t.Error("未知コマンドの後のリクエストが処理されていない")
	}
}

// TestRunLoop_InvalidJSONContinues は、壊れた1行でループが止まらないことを確認する。
// stdioは1本の接続なので、ここで抜けるとプラグインが黙って死ぬ。
func TestRunLoop_InvalidJSONContinues(t *testing.T) {
	var in bytes.Buffer
	in.WriteString("{壊れたJSON\n")
	in.WriteString(`{"id":"req-2","command":"ping"}` + "\n")

	var out bytes.Buffer
	runLoop(Handler{}, Config{}, t.TempDir(), "testuser", &in, &out)

	responses := []pluginResponse{}
	dec := json.NewDecoder(&out)
	for dec.More() {
		var resp pluginResponse
		if err := dec.Decode(&resp); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		responses = append(responses, resp)
	}

	if len(responses) != 2 {
		t.Fatalf("レスポンス件数 = %d, want 2（壊れた行でループが止まっている）", len(responses))
	}
	if len(responses[0].Errors) == 0 {
		t.Error("壊れたJSONでエラーが返っていない")
	}
	if !responses[1].Pong {
		t.Error("壊れた行の後のリクエストが処理されていない")
	}
}

// TestRunLoop_CloseStopsLoop は、closeコマンドでループを抜け、
// それ以降のリクエストを処理しないことを確認する。
// Run はこの戻り値を見て os.Exit(0) する。
func TestRunLoop_CloseStopsLoop(t *testing.T) {
	var in bytes.Buffer
	enc := json.NewEncoder(&in)
	_ = enc.Encode(pluginRequest{ID: "req-1", Command: "close"})
	_ = enc.Encode(pluginRequest{ID: "req-2", Command: "ping"})

	var out bytes.Buffer
	closed := runLoop(Handler{}, Config{}, t.TempDir(), "testuser", &in, &out)

	if !closed {
		t.Error("closeコマンドなのに false が返っている（Run が os.Exit(0) しない）")
	}
	if strings.Contains(out.String(), "req-2") {
		t.Errorf("close後のリクエストが処理されている: %s", out.String())
	}
}

// TestRunLoop_StdinCloseReturnsFalse は、stdinが閉じただけの終了では
// closeコマンド扱いにならないことを確認する。
func TestRunLoop_StdinCloseReturnsFalse(t *testing.T) {
	closed := runLoop(Handler{}, Config{}, t.TempDir(), "testuser", bytes.NewReader(nil), &bytes.Buffer{})
	if closed {
		t.Error("stdinが閉じただけなのに close 扱いになっている")
	}
}

// TestRunLoop_PassesUserIDToHandler は、gkillから渡されたユーザIDが
// ハンドラのcontextから取り出せることを確認する。
func TestRunLoop_PassesUserIDToHandler(t *testing.T) {
	var gotUserID any
	h := Handler{
		FindKyous: func(ctx context.Context, _ Query, _ Config) ([]Kyou, error) {
			gotUserID = ctx.Value(ctxKeyUserID{})
			return nil, nil
		},
	}

	var in bytes.Buffer
	_ = json.NewEncoder(&in).Encode(pluginRequest{ID: "req-1", Command: "find_kyous"})
	runLoop(h, Config{}, t.TempDir(), "testuser", &in, &bytes.Buffer{})

	if gotUserID != "testuser" {
		t.Errorf("ctxのuserID = %v, want %q", gotUserID, "testuser")
	}
}

// TestRunLoop_PassesConfigToHandler は、読み込まれた設定がハンドラに渡ることを確認する。
func TestRunLoop_PassesConfigToHandler(t *testing.T) {
	var gotConfig Config
	h := Handler{
		FindKyous: func(_ context.Context, _ Query, cfg Config) ([]Kyou, error) {
			gotConfig = cfg
			return nil, nil
		},
	}

	runLoopWith(t, h, Config{"source_dirs": "/var/logs"}, t.TempDir(),
		pluginRequest{ID: "req-1", Command: "find_kyous"})

	if gotConfig["source_dirs"] != "/var/logs" {
		t.Errorf("cfg = %v, want source_dirs=/var/logs", gotConfig)
	}
}
