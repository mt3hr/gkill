package reps

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
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
	// behaviorGate はリクエストを受け取った事実を記録してから、
	// テストが解放ファイルを作るまで応答を止める。
	// 「一定時間sleepしてから応答する」だと、テスト側のキャンセルより先に
	// 応答が届いてしまう可能性があり、壊れたコードでもテストが通る
	// （偽グリーンになる）。ファイルなら順序を確実に作れる。
	behaviorGate = "gate"
	// behaviorTyped は型別データ・付随データ・GPSログを返す。
	// リクエストのたびにコマンド名を state ファイルに追記するので、
	// 「アダプタが1件ずつプラグインへ往復していないか」を回数で検査できる。
	behaviorTyped = "typed"
)

// fakeTypedKyouCount は behaviorTyped の偽プラグインが返すKyouの件数。
const fakeTypedKyouCount = 3

// fakeGPSLogTotal は behaviorTyped の偽プラグインが返すGPSログの総点数。
// pluginGPSLogPageSize より小さいので1ページで返るが、
// テストは Limit を小さくして複数ページを作る。
const fakeGPSLogTotal = 250

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
		// close はテストの後片付け（t.Cleanup の Close）で使うので止めない。
		if behavior == behaviorGate && req.Command != "close" {
			statePath := os.Getenv(envPluginStateFile)
			if f, err := os.OpenFile(statePath+".received", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600); err == nil {
				fmt.Fprintln(f, req.ID)
				_ = f.Close()
			}
			for {
				if _, err := os.Stat(statePath + ".release"); err == nil {
					break
				}
				time.Sleep(5 * time.Millisecond)
			}
		}

		// typed のときは、どのコマンドが何回来たかを記録する。
		// アダプタがプラグインへ往復していないことを回数で検査するために使う。
		if behavior == behaviorTyped {
			if statePath := os.Getenv(envPluginStateFile); statePath != "" {
				if f, err := os.OpenFile(statePath+".commands", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600); err == nil {
					fmt.Fprintln(f, req.Command)
					_ = f.Close()
				}
			}
		}

		resp := gkill_plugin.PluginResponse{ID: req.ID}
		if behavior == behaviorTyped {
			switch req.Command {
			case "find_kyous":
				resp.Kyous = fakeTypedPluginKyous()
				_ = encoder.Encode(resp)
				continue
			case "get_gps_logs":
				offset, limit := 0, fakeGPSLogTotal
				if req.GPSLogQuery != nil {
					offset = req.GPSLogQuery.Offset
					if req.GPSLogQuery.Limit > 0 {
						limit = req.GPSLogQuery.Limit
					}
				}
				base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
				for i := offset; i < offset+limit && i < fakeGPSLogTotal; i++ {
					resp.GPSLogs = append(resp.GPSLogs, gkill_plugin.PluginGPSLog{
						RelatedTime: base.Add(time.Duration(i) * time.Minute),
						Latitude:    35.0 + float64(i)/10000.0,
						Longitude:   139.0 + float64(i)/10000.0,
					})
				}
				resp.HasMoreGPSLogs = offset+len(resp.GPSLogs) < fakeGPSLogTotal
				_ = encoder.Encode(resp)
				continue
			}
		}
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

// fakeTypedPluginKyous は型別データ・付随データ付きのKyouを返す。
// KC / Kmemo と、タグ・テキスト・通知を混ぜてある。
func fakeTypedPluginKyous() []gkill_plugin.PluginKyou {
	now := time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)
	kyous := []gkill_plugin.PluginKyou{}
	for i := range fakeTypedKyouCount {
		kyou := fakePluginKyou(fmt.Sprintf("typed-%d", i))
		kyou.RelatedTime = now.AddDate(0, 0, -i)
		kyou.CreateTime = kyou.RelatedTime
		kyou.UpdateTime = kyou.RelatedTime
		kyou.DataType = "kc"
		kyou.Typed = &gkill_plugin.PluginTypedData{
			KC: &gkill_plugin.PluginKC{
				Title:    fmt.Sprintf("歩数%d", i),
				NumValue: json.Number(fmt.Sprintf("%d", 1000+i)),
			},
		}
		kyou.Tags = []string{"fitbit", fmt.Sprintf("歩数%d", i)}
		kyou.Texts = []string{fmt.Sprintf("メモ%d", i)}
		kyou.Notifications = []gkill_plugin.PluginNotification{
			{Content: fmt.Sprintf("通知%d", i), NotificationTime: kyou.RelatedTime},
		}
		kyous = append(kyous, kyou)
	}
	return kyous
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
	return newFakePluginRepositoryWithProvides(t, behavior, nil)
}

// newFakePluginRepositoryWithProvides は provides 付きのmanifestで偽プラグインを組み立てる。
// 型別データ・付随データ・GPSログのアダプタを検証するときに使う。
func newFakePluginRepositoryWithProvides(t *testing.T, behavior string, provides []gkill_plugin.PluginProvidedKind) fakePlugin {
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
		Provides:        provides,
	}

	rep := NewPluginRepository("testuser", pluginDir, manifest)
	t.Cleanup(func() { _ = rep.Close(context.Background()) })
	return fakePlugin{rep: rep, startLog: startLog, statePath: statePath}
}

// waitPluginReceived は behaviorGate の偽プラグインが want 件のリクエストを
// 受け取るまで待つ。「リクエストは届いたがまだ応答していない」状態を確実に作るために使う。
func waitPluginReceived(t *testing.T, statePath string, want int) {
	t.Helper()
	receivedLog := statePath + ".received"
	deadline := time.Now().Add(10 * time.Second)
	for {
		b, err := os.ReadFile(receivedLog)
		if err == nil && strings.Count(string(b), "\n") >= want {
			return
		}
		if err != nil && !os.IsNotExist(err) {
			t.Fatalf("read received log: %v", err)
		}
		if time.Now().After(deadline) {
			t.Fatalf("プラグインが %d 件目のリクエストを受け取らなかった", want)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// releasePlugin は behaviorGate で止めている応答を解放する。
func releasePlugin(t *testing.T, statePath string) {
	t.Helper()
	if err := os.WriteFile(statePath+".release", []byte("go"), 0o600); err != nil {
		t.Fatalf("解放ファイルの作成に失敗した: %v", err)
	}
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
		query := &find.FindQuery{Words: []string{"alpha", "beta"}}
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
// gkill側がIDs指定で絞り込むことを確認する。
// findQueryToPluginQueryがIDsを渡さない設計なので、ここが抜けると
// ID指定検索にプラグインの全件が混入する。
func TestPluginRepository_FindKyousAppliesIDFilter(t *testing.T) {
	fp := newFakePluginRepository(t, behaviorNormal)

	query := &find.FindQuery{
		Words: []string{"alpha"},
		IDs:   []string{"word:alpha"},
	}
	got, err := fp.rep.FindKyous(context.Background(), query)
	if err != nil {
		t.Fatalf("FindKyous failed: %v", err)
	}
	kyous := got["fake_plugin_rep"]
	for _, k := range kyous {
		if k.ID != "word:alpha" {
			t.Errorf("IDsで指定していないKyou %q が混入している", k.ID)
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

// TestPluginRepository_DeadlineAbortsRequest は、応答しないプラグインに対して
// Deadlineでリクエストが打ち切られることを確認する。
// ここが効かないと固まったプラグイン1つで画面全体の読み込みが終わらなくなる。
// あわせて、打ち切りはリトライ対象外（起動は1回きり）であることも見る。
//
// 「呼び出し元のキャンセル」ではなく「Deadline」であることが重要。
// 前者ではプラグインプロセスに手を出さない (TestPluginRepository_ClientCancelDoesNotKillPlugin)。
func TestPluginRepository_DeadlineAbortsRequest(t *testing.T) {
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
		t.Fatal("Deadlineでリクエストが打ち切られなかった")
	}

	if n := countPluginStarts(t, fp.startLog); n != 1 {
		t.Errorf("起動回数 = %d, want 1", n)
	}
}

// TestPluginRepository_ClientCancelDoesNotKillPlugin は、HTTPクライアントの切断
// （＝リクエストcontextのキャンセル）でプラグインプロセスが道連れにされないことを確認する。
//
// フロントは全リクエストにAbortControllerを張っていて、ダッシュボードは再取得のたびに
// 前のget_kyousをabortする（use-dashboard-page.ts）。その切断がプラグインまで伝わって
// プロセスをKillしていると、画面の絞り込みを変えるだけでユーザのプラグインが落ちる。
//
// 起動回数だけを見てもKillは検出できない（Killしても即座に再起動はしないため）。
// キャンセル後にもう一度呼び出し、同じプロセスで応答が返ることで生存を確認する。
func TestPluginRepository_ClientCancelDoesNotKillPlugin(t *testing.T) {
	fp := newFakePluginRepository(t, behaviorGate)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		_, err := fp.rep.GetKyou(ctx, "canceled-call", nil)
		done <- err
	}()

	// リクエストがプラグインに届き、まだ応答していない状態を確実に作ってから切断する
	waitPluginReceived(t, fp.statePath, 1)
	cancel()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("キャンセルしたのにエラーにならなかった")
		}
	case <-time.After(30 * time.Second):
		t.Fatal("キャンセルしても呼び出しが返ってこなかった")
	}

	// 止めていた応答を解放する。壊れたコードではこの時点で既にプロセスが殺されている。
	releasePlugin(t, fp.statePath)

	kyou, err := fp.rep.GetKyou(context.Background(), "after-cancel", nil)
	if err != nil {
		t.Fatalf("キャンセル後のGetKyouに失敗した: %v", err)
	}
	if kyou == nil || kyou.ID != "after-cancel" {
		t.Fatalf("キャンセル後のレスポンスが不正: %+v", kyou)
	}
	if n := countPluginStarts(t, fp.startLog); n != 1 {
		t.Errorf("起動回数 = %d, want 1（呼び出し元のキャンセルでプラグインプロセスが殺されている）", n)
	}
}

// TestPluginRepository_StaleResponseIsDiscarded は、打ち切った呼び出しの応答が
// 後続の呼び出しに混入しないことを確認する。
//
// 呼び出し元のキャンセルでプロセスを殺さなくなったぶん、打ち切った要求の応答が
// 遅れて届くようになった。レスポンスIDの突き合わせで読み捨てられなければ、
// 「別の記録の中身が返ってくる」という最悪の壊れ方をする。
func TestPluginRepository_StaleResponseIsDiscarded(t *testing.T) {
	fp := newFakePluginRepository(t, behaviorGate)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		_, err := fp.rep.GetKyou(ctx, "stale-response", nil)
		done <- err
	}()

	waitPluginReceived(t, fp.statePath, 1)
	cancel()
	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("キャンセルしても呼び出しが返ってこなかった")
	}

	// ここで打ち切った要求の応答がプラグインから流れてくる
	releasePlugin(t, fp.statePath)

	kyou, err := fp.rep.GetKyou(context.Background(), "fresh-response", nil)
	if err != nil {
		t.Fatalf("後続のGetKyouに失敗した: %v", err)
	}
	if kyou == nil {
		t.Fatal("後続のGetKyouがnilを返した")
	}
	if kyou.ID != "fresh-response" {
		t.Errorf("KyouID = %q, want %q（打ち切った呼び出しの応答が混入している）", kyou.ID, "fresh-response")
	}
}

// TestPluginRepository_DeadlineReapsPlugin は、応答しないプラグインがDeadlineで
// 回収され、次の呼び出しで起動し直されることを確認する。
// 回収しないと詰まったプラグインが復帰できず、以降の呼び出しが全部Deadline待ちになる。
func TestPluginRepository_DeadlineReapsPlugin(t *testing.T) {
	fp := newFakePluginRepository(t, behaviorGate)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	if _, err := fp.rep.GetKyou(ctx, "wedged", nil); err == nil {
		t.Fatal("応答しないプラグインなのにエラーにならなかった")
	}
	if n := countPluginStarts(t, fp.startLog); n != 1 {
		t.Fatalf("Deadline直後の起動回数 = %d, want 1", n)
	}

	// 回収済みなので、次の呼び出しは新しいプロセスで動く
	releasePlugin(t, fp.statePath)
	if _, err := fp.rep.GetKyou(context.Background(), "after-reap", nil); err != nil {
		t.Fatalf("回収後のGetKyouに失敗した: %v", err)
	}
	if n := countPluginStarts(t, fp.startLog); n != 2 {
		t.Errorf("起動回数 = %d, want 2（Deadlineで回収されていない）", n)
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

// TestPluginRepository_QueueTimeoutDoesNotReapProcess は、実行スロットの
// 順番待ちで待ちきれなかった呼び出しがプラグインプロセスを殺さないことを確認する。
//
// かつては期限をスロット取得より前に張っていたため、行列に並んでいるだけで
// 期限を食い潰し、正常に応答しているプラグインを期限切れとして回収していた。
// 一覧の行数ぶんの本文取得が同時に来ると、この回収と再起動が延々と繰り返され、
// 待ち行列がまったく消化されなくなる。
func TestPluginRepository_QueueTimeoutDoesNotReapProcess(t *testing.T) {
	fp := newFakePluginRepository(t, behaviorGate)

	// 1本目にスロットを掴ませたまま応答を止める
	firstDone := make(chan error, 1)
	go func() {
		_, err := fp.rep.GetKyou(context.Background(), "first", nil)
		firstDone <- err
	}()
	waitPluginReceived(t, fp.statePath, 1)

	// 2本目は行列で待ちきれずビジーになる。
	// 期限を短くすると順番待ちの上限もそこまでに縮まる。
	busyCtx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	_, err := fp.rep.GetKyou(busyCtx, "second", nil)
	if err == nil {
		t.Fatal("行列で待たされた呼び出しが成功してしまった")
	}
	if !errors.Is(err, ErrPluginBusy) {
		t.Fatalf("err = %v, want ErrPluginBusy", err)
	}

	// 混んでいただけなのでプロセスは回収されていない
	if got := countPluginStarts(t, fp.startLog); got != 1 {
		t.Fatalf("起動回数 = %d, want 1（順番待ちの打ち切りでプロセスを殺している）", got)
	}

	// 1本目を解放すれば普通に完了する
	releasePlugin(t, fp.statePath)
	select {
	case err := <-firstDone:
		if err != nil {
			t.Fatalf("1本目の呼び出しが失敗した: %v", err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("1本目の呼び出しが返ってこない")
	}
	if got := countPluginStarts(t, fp.startLog); got != 1 {
		t.Fatalf("起動回数 = %d, want 1", got)
	}
}

// TestPluginRepository_FindKyousFailureIsWarningNotError は、プラグイン検索が
// 失敗しても検索全体をエラーにせず、警告コレクタへプラグイン名だけを残すことを確認する。
//
// プラグインが1本壊れているだけで検索がエラーになると、
// クライアントは他repの結果まで丸ごと捨ててしまう。かといって黙って0件にすると
// 「静かな欠落」になるので、errはnil・結果は空・警告に名前を残す、の3点セットが結線。
func TestPluginRepository_FindKyousFailureIsWarningNotError(t *testing.T) {
	// 実行できない実行ファイルを指すマニフェスト。起動に失敗して検索が失敗する
	manifest := gkill_plugin.PluginManifest{
		ProtocolVersion: "1",
		Name:            "broken_plugin",
		Version:         "1.0.0",
		DataType:        "broken_plugin_kyou",
		RepName:         "broken_plugin_rep",
		Executable:      "no_such_plugin_executable",
	}
	rep := NewPluginRepository("testuser", t.TempDir(), manifest)
	t.Cleanup(func() { _ = rep.Close(context.Background()) })

	ctx := WithFindWarnings(context.Background())
	got, err := rep.FindKyous(ctx, &find.FindQuery{})
	if err != nil {
		t.Fatalf("プラグイン障害はエラーにせず警告にするはず: err = %v", err)
	}

	kyouCount := 0
	for _, kyous := range got {
		kyouCount += len(kyous)
	}
	if kyouCount != 0 {
		t.Errorf("失敗時の結果は空のはず: got %d件", kyouCount)
	}

	warnings := PluginFindWarnings(ctx)
	if len(warnings) != 1 || warnings[0] != "broken_plugin" {
		t.Errorf("警告にプラグイン名が残るはず: got %v", warnings)
	}
}

// TestPluginRepository_FindKyousSuccessLeavesNoWarning は対照。
// 正常に応答したプラグインで警告が立つと、毎回「取得できませんでした」が出てしまう。
func TestPluginRepository_FindKyousSuccessLeavesNoWarning(t *testing.T) {
	fp := newFakePluginRepository(t, behaviorNormal)

	ctx := WithFindWarnings(context.Background())
	if _, err := fp.rep.FindKyous(ctx, &find.FindQuery{}); err != nil {
		t.Fatalf("FindKyous failed: %v", err)
	}

	if warnings := PluginFindWarnings(ctx); len(warnings) != 0 {
		t.Errorf("正常時に警告が立ってはいけない: got %v", warnings)
	}
}
