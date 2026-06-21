package reps

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/api/gkill_plugin"
	gkill_cache "github.com/mt3hr/gkill/src/server/gkill/dao/reps/cache"
)

// pluginProcess はプラグインプロセスとのstdio通信状態を管理する。
type pluginProcess struct {
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	scanner *bufio.Scanner
	started bool
}

// pluginRepositoryImpl は PluginRepository インターフェースの実装。
// プラグインバイナリをサブプロセスとして起動し、stdio 改行区切りJSONで通信する。
type pluginRepositoryImpl struct {
	mu        sync.Mutex // すべての操作を直列化するミューテックス
	userID    string     // gkillユーザID
	pluginDir string     // $GKILL_HOME/plugins/{userID}/{pluginName}/
	manifest  gkill_plugin.PluginManifest

	proc *pluginProcess // nil = 未起動
}

// インターフェース適合確認（コンパイル時チェック）
var _ PluginRepository = (*pluginRepositoryImpl)(nil)

// NewPluginRepository はプラグインリポジトリを作成する。
// プロセスは初回クエリ時に遅延起動する。
func NewPluginRepository(userID string, pluginDir string, manifest gkill_plugin.PluginManifest) PluginRepository {
	return &pluginRepositoryImpl{
		userID:    userID,
		pluginDir: pluginDir,
		manifest:  manifest,
	}
}

// ensureStarted はプラグインプロセスが起動していることを保証する。
// 呼び出し側で p.mu をロック済みであること。
// プロセスの寿命はリクエストコンテキストに依存させないよう context.Background() を使う。
func (p *pluginRepositoryImpl) ensureStarted() error {
	if p.proc != nil && p.proc.started {
		return nil
	}

	execName := p.manifest.Executable
	if runtime.GOOS == "windows" {
		execName += ".exe"
	}
	execPath := filepath.Join(p.pluginDir, execName)

	// プロセスはリクエストのキャンセルで終了させないためBackground contextを使う
	cmd := exec.CommandContext(context.Background(),
		execPath,
		"--gkill-plugin-dir", p.pluginDir,
		"--gkill-user-id", p.userID,
		"--gkill-protocol-version", p.manifest.ProtocolVersion,
	)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("error at get stdin pipe for plugin %s: %w", p.manifest.Name, err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("error at get stdout pipe for plugin %s: %w", p.manifest.Name, err)
	}
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("error at start plugin %s (%s): %w", p.manifest.Name, execPath, err)
	}

	scanner := bufio.NewScanner(stdout)
	// 大きな会話HTMLレスポンスに対応するためバッファを32MBに拡大
	scanner.Buffer(make([]byte, 32*1024*1024), 32*1024*1024)

	p.proc = &pluginProcess{
		cmd:     cmd,
		stdin:   stdin,
		scanner: scanner,
		started: true,
	}

	slog.Info(fmt.Sprintf("plugin started: %s (user=%s)", p.manifest.Name, p.userID))
	return nil
}

// scanResult は scanner.Scan() の結果を goroutine 間で受け渡すための型。
type scanResult struct {
	data []byte
	err  error
}

// sendRequest は改行区切りJSONでリクエストを送り、レスポンスを受け取る。
// 呼び出し前に p.mu がロック済みであること。
// ctx がキャンセルされた場合はプロセスを強制終了してエラーを返す。
func (p *pluginRepositoryImpl) sendRequest(ctx context.Context, req gkill_plugin.PluginRequest) (*gkill_plugin.PluginResponse, error) {
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error at marshal plugin request: %w", err)
	}

	if _, err := fmt.Fprintf(p.proc.stdin, "%s\n", reqBytes); err != nil {
		p.proc.started = false
		return nil, fmt.Errorf("error at write to plugin stdin %s: %w", p.manifest.Name, err)
	}

	// bufio.Scanner.Scan() はブロッキングなのでgoroutineで実行し、
	// contextキャンセル（タイムアウト含む）に対応する
	ch := make(chan scanResult, 1)
	go func() {
		if p.proc.scanner.Scan() {
			b := make([]byte, len(p.proc.scanner.Bytes()))
			copy(b, p.proc.scanner.Bytes())
			ch <- scanResult{data: b}
		} else {
			if scanErr := p.proc.scanner.Err(); scanErr != nil {
				ch <- scanResult{err: fmt.Errorf("error at read from plugin stdout %s: %w", p.manifest.Name, scanErr)}
			} else {
				ch <- scanResult{err: fmt.Errorf("plugin %s closed stdout unexpectedly", p.manifest.Name)}
			}
		}
	}()

	select {
	case <-ctx.Done():
		p.proc.started = false
		// goroutineのScan()ブロックを解除するためプロセスを強制終了する。
		// 次のcallCommand呼び出しでensureStarted()が新プロセスを起動する。
		if p.proc.cmd.Process != nil {
			_ = p.proc.cmd.Process.Kill()
		}
		return nil, fmt.Errorf("plugin %s request timed out: %w", p.manifest.Name, ctx.Err())
	case result := <-ch:
		if result.err != nil {
			p.proc.started = false
			return nil, result.err
		}
		var resp gkill_plugin.PluginResponse
		if err := json.Unmarshal(result.data, &resp); err != nil {
			return nil, fmt.Errorf("error at unmarshal plugin response %s: %w", p.manifest.Name, err)
		}
		if len(resp.Errors) > 0 {
			return &resp, fmt.Errorf("plugin %s returned errors: %v", p.manifest.Name, resp.Errors)
		}
		return &resp, nil
	}
}

// callCommand は p.mu でロックし、ensureStarted・sendRequest・クラッシュ時リトライをまとめて実行する。
// p.mu で全操作を直列化することで並列リクエストによる競合を防ぐ。
// ctx にDeadlineが設定されていない場合はデフォルト30秒のタイムアウトを付加する。
func (p *pluginRepositoryImpl) callCommand(ctx context.Context, req gkill_plugin.PluginRequest) (*gkill_plugin.PluginResponse, error) {
	// タイムアウト未設定の場合はデフォルト30秒を付加する
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if err := p.ensureStarted(); err != nil {
		return nil, err
	}

	resp, err := p.sendRequest(ctx, req)
	if err != nil {
		// タイムアウト・キャンセルはリトライしない（リトライしても同じ結果になるため）
		if ctx.Err() != nil {
			return nil, err
		}
		// プロセスクラッシュ時のみ1回リトライ（自動再起動）
		slog.Warn(fmt.Sprintf("plugin %s error, retrying: %v", p.manifest.Name, err))
		p.proc.started = false
		if startErr := p.ensureStarted(); startErr != nil {
			return nil, fmt.Errorf("plugin restart failed %s: %w (original: %v)", p.manifest.Name, startErr, err)
		}
		resp, err = p.sendRequest(ctx, req)
		if err != nil {
			return nil, err
		}
	}
	return resp, nil
}

// --- Repository interface 実装 ---

func (p *pluginRepositoryImpl) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]Kyou, error) {
	pq := findQueryToPluginQuery(query)
	req := gkill_plugin.PluginRequest{
		ID:      uuid.New().String(),
		Command: "find_kyous",
		Query:   pq,
	}

	resp, err := p.callCommand(ctx, req)
	if err != nil {
		slog.Error(fmt.Sprintf("plugin find_kyous error %s: %v", p.manifest.Name, err))
		return map[string][]Kyou{p.manifest.RepName: {}}, nil
	}

	kyous := make([]Kyou, 0, len(resp.Kyous))
	for _, pk := range resp.Kyous {
		k := convertPluginKyouToKyou(pk)
		if pluginKyouMatchesQuery(k, query) {
			kyous = append(kyous, k)
		}
	}
	return map[string][]Kyou{p.manifest.RepName: kyous}, nil
}

func (p *pluginRepositoryImpl) GetKyou(ctx context.Context, id string, updateTime *time.Time) (*Kyou, error) {
	req := gkill_plugin.PluginRequest{
		ID:         uuid.New().String(),
		Command:    "get_kyou",
		KyouID:     id,
		UpdateTime: updateTime,
	}

	resp, err := p.callCommand(ctx, req)
	if err != nil {
		return nil, err
	}
	if resp.Kyou == nil {
		return nil, nil
	}
	kyou := convertPluginKyouToKyou(*resp.Kyou)
	return &kyou, nil
}

func (p *pluginRepositoryImpl) GetKyouHistories(ctx context.Context, id string) ([]Kyou, error) {
	kyou, err := p.GetKyou(ctx, id, nil)
	if err != nil {
		return nil, err
	}
	if kyou == nil {
		return []Kyou{}, nil
	}
	return []Kyou{*kyou}, nil
}

func (p *pluginRepositoryImpl) GetPath(ctx context.Context, _ string) (string, error) {
	return p.pluginDir, nil
}

func (p *pluginRepositoryImpl) GetRepName(_ context.Context) (string, error) {
	return p.manifest.RepName, nil
}

func (p *pluginRepositoryImpl) UpdateCache(_ context.Context) error {
	return nil
}

func (p *pluginRepositoryImpl) GetLatestDataRepositoryAddress(_ context.Context, _ bool) ([]gkill_cache.LatestDataRepositoryAddress, error) {
	return []gkill_cache.LatestDataRepositoryAddress{}, nil
}

func (p *pluginRepositoryImpl) Close(_ context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.proc == nil || !p.proc.started {
		return nil
	}

	req := gkill_plugin.PluginRequest{
		ID:      uuid.New().String(),
		Command: "close",
	}
	reqBytes, _ := json.Marshal(req)
	fmt.Fprintf(p.proc.stdin, "%s\n", reqBytes) //nolint:errcheck

	done := make(chan error, 1)
	go func() { done <- p.proc.cmd.Wait() }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		slog.Warn(fmt.Sprintf("plugin %s did not exit in time, killing", p.manifest.Name))
		p.proc.cmd.Process.Kill() //nolint:errcheck
	}

	p.proc.started = false
	slog.Info(fmt.Sprintf("plugin closed: %s", p.manifest.Name))
	return nil
}

func (p *pluginRepositoryImpl) UnWrap() ([]Repository, error) {
	return []Repository{p}, nil
}

// --- PluginRepository 追加メソッド ---

func (p *pluginRepositoryImpl) GetManifest() gkill_plugin.PluginManifest {
	return p.manifest
}

func (p *pluginRepositoryImpl) GetContentHTML(ctx context.Context, kyouID string) (string, error) {
	req := gkill_plugin.PluginRequest{
		ID:      uuid.New().String(),
		Command: "get_content_html",
		KyouID:  kyouID,
	}
	resp, err := p.callCommand(ctx, req)
	if err != nil {
		return "", err
	}
	return resp.HTML, nil
}

func (p *pluginRepositoryImpl) GetConfigHTML(ctx context.Context) (string, error) {
	req := gkill_plugin.PluginRequest{
		ID:      uuid.New().String(),
		Command: "get_config_html",
	}
	resp, err := p.callCommand(ctx, req)
	if err != nil {
		return "", err
	}
	return resp.HTML, nil
}

func (p *pluginRepositoryImpl) PostConfig(ctx context.Context, formData map[string]string) error {
	req := gkill_plugin.PluginRequest{
		ID:       uuid.New().String(),
		Command:  "post_config",
		FormData: formData,
	}
	_, err := p.callCommand(ctx, req)
	return err
}

func (p *pluginRepositoryImpl) IsAlive(ctx context.Context) bool {
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req := gkill_plugin.PluginRequest{
		ID:      uuid.New().String(),
		Command: "ping",
	}
	resp, err := p.callCommand(pingCtx, req)
	return err == nil && resp.Pong
}

// --- 変換ヘルパー ---

// convertPluginKyouToKyou はPluginKyouをgkill本体のKyouに変換する。
func convertPluginKyouToKyou(pk gkill_plugin.PluginKyou) Kyou {
	return Kyou{
		IsDeleted:    pk.IsDeleted,
		ID:           pk.ID,
		RepName:      pk.RepName,
		RelatedTime:  pk.RelatedTime,
		DataType:     pk.DataType,
		CreateTime:   pk.CreateTime,
		CreateApp:    pk.CreateApp,
		CreateDevice: pk.CreateDevice,
		CreateUser:   pk.CreateUser,
		UpdateTime:   pk.UpdateTime,
		UpdateApp:    pk.UpdateApp,
		UpdateDevice: pk.UpdateDevice,
		UpdateUser:   pk.UpdateUser,
	}
}

// findQueryToPluginQuery はFindQueryをPluginQueryに変換する。
func findQueryToPluginQuery(q *find.FindQuery) *gkill_plugin.PluginQuery {
	if q == nil {
		return &gkill_plugin.PluginQuery{}
	}
	pq := &gkill_plugin.PluginQuery{
		IsDeleted:      q.IsDeleted,
		OnlyLatestData: q.OnlyLatestData,
	}
	if q.UseWords {
		pq.Words = q.Words
		pq.NotWords = q.NotWords
		pq.WordsAnd = q.WordsAnd
	}
	if q.UseTags {
		pq.Tags = q.Tags
		pq.NotTags = q.HideTags
		pq.TagsAnd = q.TagsAnd
	}
	if q.UseCalendar {
		pq.CalendarStartDate = q.CalendarStartDate
		pq.CalendarEndDate = q.CalendarEndDate
	}
	return pq
}

// pluginKyouMatchesQuery はgkill側での追加フィルタリング（プラグイン側フィルタの補完）。
func pluginKyouMatchesQuery(kyou Kyou, q *find.FindQuery) bool {
	if q == nil {
		return true
	}
	if q.UseCalendar {
		if q.CalendarStartDate != nil && kyou.RelatedTime.Before(*q.CalendarStartDate) {
			return false
		}
		if q.CalendarEndDate != nil && kyou.RelatedTime.After(*q.CalendarEndDate) {
			return false
		}
	}
	// UseIDsフィルタ:
	// findQueryToPluginQueryはUseIDsをPluginQueryに変換しないため、
	// プラグインはID指定クエリを受けても全件返す。
	// gkill側でIDフィルタを補完することで、
	// textMatchFindByIDQueryでプラグイン全件が混入するのを防ぐ。
	if q.UseIDs {
		idSet := make(map[string]struct{}, len(q.IDs))
		for _, id := range q.IDs {
			idSet[id] = struct{}{}
		}
		if _, inSet := idSet[kyou.ID]; !inSet {
			return false
		}
	}
	return true
}
