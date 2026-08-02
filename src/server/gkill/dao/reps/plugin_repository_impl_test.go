package reps

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/api/gkill_plugin"
)

// プラグインリポジトリはプラグインバイナリをサブプロセスとして起動し、
// 改行区切りJSONで会話する。外部バイナリをビルドせずにこれを検証するため、
// 標準ライブラリ os/exec のテストで使われている「テストバイナリ自身を再execする」
// 方式をとる。TestMain が環境変数を見て、偽プラグインとしてプロトコルを喋る。
//
// pluginRepositoryImpl は実行ファイルを filepath.Join(pluginDir, Executable[+".exe"])
// で解決するので、pluginDir にテストバイナリのあるディレクトリ、Executable に
// テストバイナリ名（Windowsでは .exe を除いたもの）を渡すと自分自身が起動する。

const (
	envPluginMode      = "GKILL_TEST_PLUGIN_MODE"
	envPluginBehavior  = "GKILL_TEST_PLUGIN_BEHAVIOR"
	envPluginStateFile = "GKILL_TEST_PLUGIN_STATE_FILE"
	envPluginStartLog  = "GKILL_TEST_PLUGIN_START_LOG"

	behaviorNormal    = "normal"
	behaviorCrashOnce = "crash_once"
	behaviorHang      = "hang"
)

func TestMain(m *testing.M) {
	if os.Getenv(envPluginMode) != "" {
		runFakePlugin()
		return
	}
	os.Exit(m.Run())
}

// runFakePlugin は偽プラグインとして stdin/stdout でプロトコルを処理する。
// テストバイナリが再execされたときだけ実行される。
func runFakePlugin() {
	// 起動された事実を記録する（遅延起動・再起動の確認に使う）
	if logPath := os.Getenv(envPluginStartLog); logPath != "" {
		f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
		if err == nil {
			fmt.Fprintln(f, "started")
			f.Close()
		}
	}

	behavior := os.Getenv(envPluginBehavior)
	if behavior == "" {
		behavior = behaviorNormal
	}

	// crash_once: 初回起動のプロセスだけ、最初のリクエストを受けた直後に落ちる。
	// 2回目以降の起動では通常動作する（自動再起動の確認用）。
	crashThisRun := false
	if behavior == behaviorCrashOnce {
		statePath := os.Getenv(envPluginStateFile)
		if _, err := os.Stat(statePath); os.IsNotExist(err) {
			_ = os.WriteFile(statePath, []byte("crashed"), 0o600)
			crashThisRun = true
		}
	}

	encoder := json.NewEncoder(os.Stdout)
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for scanner.Scan() {
		var req gkill_plugin.PluginRequest
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
			_ = encoder.Encode(gkill_plugin.PluginResponse{Errors: []string{"invalid request"}})
			continue
		}

		if crashThisRun {
			// レスポンスを返さずにプロセスごと終了する＝クラッシュを模す
			os.Exit(1)
		}
		if behavior == behaviorHang {
			// contextキャンセルでリクエストが打ち切られることを確認するため応答しない。
			// select{} や <-chan だと Go のデッドロック検出でプロセスが落ちてしまい
			// 「クラッシュ→自動再起動」の経路に入ってしまうので、Sleepで待つ。
			time.Sleep(time.Hour)
		}

		resp := gkill_plugin.PluginResponse{ID: req.ID}
		switch req.Command {
		case "find_kyous":
			// クエリのWordsをそのままKyouのIDに詰めて返し、
			// gkill側がPluginQueryへ正しく変換できているかを見えるようにする
			words := []string{}
			if req.Query != nil {
				words = req.Query.Words
			}
			for _, w := range words {
				resp.Kyous = append(resp.Kyous, fakePluginKyou("word:"+w))
			}
			resp.Kyous = append(resp.Kyous, fakePluginKyou("always"))
		case "get_kyou":
			// リクエストのKyouIDをそのまま返す。並行呼び出しでレスポンスが
			// 取り違わっていないことをこれで判定する。
			k := fakePluginKyou(req.KyouID)
			resp.Kyou = &k
		case "get_rep_name":
			resp.RepName = "fake_plugin_rep"
		case "get_content_html":
			resp.HTML = "<p>" + req.KyouID + "</p>"
		case "get_config_html":
			resp.HTML = "<form>fake config</form>"
		case "post_config":
			// 受け取ったフォームを設定ファイルに書き出す代わりに、
			// 呼び出し側から確認できるようテンポラリへ落とす
			if statePath := os.Getenv(envPluginStateFile); statePath != "" {
				b, _ := json.Marshal(req.FormData)
				_ = os.WriteFile(statePath+".posted", b, 0o600)
			}
		case "ping":
			resp.Pong = true
		case "close":
			_ = encoder.Encode(resp)
			return
		default:
			resp.Errors = []string{"unknown command: " + req.Command}
		}
		_ = encoder.Encode(resp)
	}
}

func fakePluginKyou(id string) gkill_plugin.PluginKyou {
	now := time.Now().Truncate(time.Second)
	return gkill_plugin.PluginKyou{
		ID:          id,
		RepName:     "fake_plugin_rep",
		DataType:    "fake_plugin_kyou",
		RelatedTime: now,
		CreateTime:  now,
		UpdateTime:  now,
	}
}

// fakePlugin は偽プラグインを起動するリポジトリと、その観測用ファイルパス。
type fakePlugin struct {
	rep PluginRepository
	// startLog には偽プラグインが起動するたびに1行追記される
	startLog string
	// statePath は偽プラグインの状態ファイル。post_config の内容は
	// statePath+".posted" に書き出される
	statePath string
}

// newFakePluginRepository は自分自身（テストバイナリ）をプラグインとして起動する
// リポジトリを組み立てる。
func newFakePluginRepository(t *testing.T, behavior string) fakePlugin {
	t.Helper()

	exePath, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable failed: %v", err)
	}
	pluginDir := filepath.Dir(exePath)
	execName := strings.TrimSuffix(filepath.Base(exePath), ".exe")

	tmpDir := t.TempDir()
	startLog := filepath.Join(tmpDir, "start.log")
	statePath := filepath.Join(tmpDir, "state")
	t.Setenv(envPluginMode, "1")
	t.Setenv(envPluginBehavior, behavior)
	t.Setenv(envPluginStateFile, statePath)
	t.Setenv(envPluginStartLog, startLog)

	manifest := gkill_plugin.PluginManifest{
		ProtocolVersion: "1",
		Name:            "fake_plugin",
		Version:         "1.0.0",
		DataType:        "fake_plugin_kyou",
		RepName:         "fake_plugin_rep",
		Executable:      execName,
	}

	rep := NewPluginRepository("testuser", pluginDir, manifest)
	t.Cleanup(func() { _ = rep.Close(context.Background()) })
	return fakePlugin{rep: rep, startLog: startLog, statePath: statePath}
}

// countPluginStarts は偽プラグインが何回起動したかを返す。
func countPluginStarts(t *testing.T, startLog string) int {
	t.Helper()
	b, err := os.ReadFile(startLog)
	if os.IsNotExist(err) {
		return 0
	}
	if err != nil {
		t.Fatalf("read start log: %v", err)
	}
	return strings.Count(string(b), "started")
}

// TestPluginRepository_StartsLazily は、リポジトリを作っただけでは
// プロセスが起動せず、最初のクエリで初めて起動することを確認する。
// ユーザが1つもプラグインを使っていない画面で無駄にプロセスが増えないための性質。
func TestPluginRepository_StartsLazily(t *testing.T) {
	fp := newFakePluginRepository(t, behaviorNormal)

	if n := countPluginStarts(t, fp.startLog); n != 0 {
		t.Fatalf("リポジトリ生成だけでプラグインが %d 回起動している", n)
	}

	if _, err := fp.rep.FindKyous(context.Background(), &find.FindQuery{}); err != nil {
		t.Fatalf("FindKyous failed: %v", err)
	}
	if n := countPluginStarts(t, fp.startLog); n != 1 {
		t.Errorf("初回クエリ後の起動回数 = %d, want 1", n)
	}

	// 2回目のクエリでプロセスは再利用される
	if _, err := fp.rep.GetKyou(context.Background(), "id-1", nil); err != nil {
		t.Fatalf("GetKyou failed: %v", err)
	}
	if n := countPluginStarts(t, fp.startLog); n != 1 {
		t.Errorf("2回目のクエリ後の起動回数 = %d, want 1（プロセスが使い回されていない）", n)
	}
}

// TestPluginRepository_RoundTrip はstdio越しのリクエスト/レスポンスが
// 往復することと、FindQuery→PluginQueryの変換が効いていることを確認する。
func TestPluginRepository_RoundTrip(t *testing.T) {
	fp := newFakePluginRepository(t, behaviorNormal)
	ctx := context.Background()

	t.Run("FindKyous", func(t *testing.T) {
		query := &find.FindQuery{UseWords: true, Words: []string{"alpha", "beta"}}
		got, err := fp.rep.FindKyous(ctx, query)
		if err != nil {
			t.Fatalf("FindKyous failed: %v", err)
		}
		kyous, ok := got["fake_plugin_rep"]
		if !ok {
			t.Fatalf("rep_name をキーにした結果が返っていない: %v", got)
		}
		ids := map[string]bool{}
		for _, k := range kyous {
			ids[k.ID] = true
		}
		// 偽プラグインはPluginQuery.Wordsをそのまま返すので、
		// FindQuery.Words が変換されて届いていることがここで分かる
		for _, want := range []string{"word:alpha", "word:beta"} {
			if !ids[want] {
				t.Errorf("%q が結果にない（Wordsがプラグインに渡っていない）: %v", want, ids)
			}
		}
	})

	t.Run("GetKyou", func(t *testing.T) {
		kyou, err := fp.rep.GetKyou(ctx, "kyou-123", nil)
		if err != nil {
			t.Fatalf("GetKyou failed: %v", err)
		}
		if kyou == nil {
			t.Fatal("GetKyou returned nil")
		}
		if kyou.ID != "kyou-123" {
			t.Errorf("ID = %q, want %q", kyou.ID, "kyou-123")
		}
		if kyou.DataType != "fake_plugin_kyou" {
			t.Errorf("DataType = %q, want %q", kyou.DataType, "fake_plugin_kyou")
		}
	})

	t.Run("GetKyouHistories", func(t *testing.T) {
		histories, err := fp.rep.GetKyouHistories(ctx, "kyou-456")
		if err != nil {
			t.Fatalf("GetKyouHistories failed: %v", err)
		}
		if len(histories) != 1 || histories[0].ID != "kyou-456" {
			t.Errorf("histories = %+v, want 1件で ID=kyou-456", histories)
		}
	})

	t.Run("GetRepName", func(t *testing.T) {
		// マニフェスト由来の値を返す（プラグインには問い合わせない）
		name, err := fp.rep.GetRepName(ctx)
		if err != nil {
			t.Fatalf("GetRepName failed: %v", err)
		}
		if name != "fake_plugin_rep" {
			t.Errorf("RepName = %q, want %q", name, "fake_plugin_rep")
		}
	})
}

// TestPluginRepository_ContentAndConfigCommands は、プラグイン詳細ビューと
// 設定ダイアログが使うコマンド群がstdio越しに往復することを確認する。
func TestPluginRepository_ContentAndConfigCommands(t *testing.T) {
	fp := newFakePluginRepository(t, behaviorNormal)
	ctx := context.Background()

	t.Run("GetContentHTML", func(t *testing.T) {
		html, err := fp.rep.GetContentHTML(ctx, "kyou-777")
		if err != nil {
			t.Fatalf("GetContentHTML failed: %v", err)
		}
		if !strings.Contains(html, "kyou-777") {
			t.Errorf("HTML = %q, want kyou-777 を含む", html)
		}
	})

	t.Run("GetConfigHTML", func(t *testing.T) {
		html, err := fp.rep.GetConfigHTML(ctx)
		if err != nil {
			t.Fatalf("GetConfigHTML failed: %v", err)
		}
		if !strings.Contains(html, "fake config") {
			t.Errorf("HTML = %q, want fake config を含む", html)
		}
	})

	t.Run("PostConfig", func(t *testing.T) {
		form := map[string]string{"source_dirs": "/tmp/logs"}
		if err := fp.rep.PostConfig(ctx, form); err != nil {
			t.Fatalf("PostConfig failed: %v", err)
		}
		b, err := os.ReadFile(fp.statePath + ".posted")
		if err != nil {
			t.Fatalf("フォームデータがプラグインに届いていない: %v", err)
		}
		var posted map[string]string
		if err := json.Unmarshal(b, &posted); err != nil {
			t.Fatalf("届いたフォームデータが不正: %v", err)
		}
		if posted["source_dirs"] != "/tmp/logs" {
			t.Errorf("届いたフォーム = %v, want source_dirs=/tmp/logs", posted)
		}
	})

	t.Run("IsAlive", func(t *testing.T) {
		if !fp.rep.IsAlive(ctx) {
			t.Error("IsAlive = false, want true（応答しているプラグインを死んでいると判定している）")
		}
	})
}

// TestPluginRepository_FindKyousAppliesIDFilter は、プラグインが返した全件から
// gkill側がUseIDsで絞り込むことを確認する。
// findQueryToPluginQueryがUseIDsを渡さない設計なので、ここが抜けると
// ID指定検索にプラグインの全件が混入する。
func TestPluginRepository_FindKyousAppliesIDFilter(t *testing.T) {
	fp := newFakePluginRepository(t, behaviorNormal)

	query := &find.FindQuery{
		UseWords: true,
		Words:    []string{"alpha"},
		UseIDs:   true,
		IDs:      []string{"word:alpha"},
	}
	got, err := fp.rep.FindKyous(context.Background(), query)
	if err != nil {
		t.Fatalf("FindKyous failed: %v", err)
	}
	kyous := got["fake_plugin_rep"]
	for _, k := range kyous {
		if k.ID != "word:alpha" {
			t.Errorf("UseIDsで指定していないKyou %q が混入している", k.ID)
		}
	}
	if len(kyous) != 1 {
		t.Errorf("件数 = %d, want 1", len(kyous))
	}
}

// TestPluginRepository_RestartsAfterCrash は、プラグインプロセスが落ちても
// 同じリクエストが自動再起動でリトライされ、呼び出し側にはエラーが見えないことを確認する。
func TestPluginRepository_RestartsAfterCrash(t *testing.T) {
	fp := newFakePluginRepository(t, behaviorCrashOnce)

	kyou, err := fp.rep.GetKyou(context.Background(), "after-crash", nil)
	if err != nil {
		t.Fatalf("クラッシュ後の自動再起動に失敗した: %v", err)
	}
	if kyou == nil || kyou.ID != "after-crash" {
		t.Fatalf("再起動後のレスポンスが不正: %+v", kyou)
	}
	if n := countPluginStarts(t, fp.startLog); n != 2 {
		t.Errorf("起動回数 = %d, want 2（クラッシュ1回 + 再起動1回）", n)
	}
}

// TestPluginRepository_ContextCancelAbortsRequest は、応答しないプラグインに対して
// contextのキャンセルでリクエストが打ち切られることを確認する。
// ここが効かないと固まったプラグイン1つで画面全体の読み込みが終わらなくなる。
// あわせて、タイムアウトはリトライ対象外（起動は1回きり）であることも見る。
func TestPluginRepository_ContextCancelAbortsRequest(t *testing.T) {
	fp := newFakePluginRepository(t, behaviorHang)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		_, err := fp.rep.GetKyou(ctx, "never-answered", nil)
		done <- err
	}()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("応答しないプラグインなのにエラーにならなかった")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("contextのタイムアウトでリクエストが打ち切られなかった")
	}

	if n := countPluginStarts(t, fp.startLog); n != 1 {
		t.Errorf("起動回数 = %d, want 1", n)
	}
}

// TestPluginRepository_SerializesConcurrentCalls は、stdioが1本しかないプラグインに
// 並行リクエストが来てもmutexで直列化され、レスポンスが取り違わらないことを確認する。
// 偽プラグインはリクエストのKyouIDをそのまま返すので、取り違えるとIDが一致しなくなる。
func TestPluginRepository_SerializesConcurrentCalls(t *testing.T) {
	fp := newFakePluginRepository(t, behaviorNormal)

	const n = 20
	var wg sync.WaitGroup
	errs := make(chan error, n)

	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			id := fmt.Sprintf("concurrent-%02d", i)
			kyou, err := fp.rep.GetKyou(context.Background(), id, nil)
			if err != nil {
				errs <- fmt.Errorf("GetKyou(%s): %w", id, err)
				return
			}
			if kyou == nil {
				errs <- fmt.Errorf("GetKyou(%s) returned nil", id)
				return
			}
			if kyou.ID != id {
				errs <- fmt.Errorf("レスポンスが取り違わっている: got %q, want %q", kyou.ID, id)
			}
		}()
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		t.Error(err)
	}
	if got := countPluginStarts(t, fp.startLog); got != 1 {
		t.Errorf("起動回数 = %d, want 1（並行呼び出しでプロセスが増えている）", got)
	}
}
