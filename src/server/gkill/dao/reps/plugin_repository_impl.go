package reps

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
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
//
// stdoutの読み取りはプロセスごとに常駐する1本のgoroutine（リーダー）が担当する。
// bufio.Scanner は並行安全ではないので、scanner に触れてよいのはリーダーだけ。
// リクエスト側は respCh 経由で応答を受け取り、レスポンスIDで自分宛てかを判定する。
type pluginProcess struct {
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	scanner *bufio.Scanner
	started bool

	// respCh はリーダーからの受け口。
	// 呼び出しは実行スロットで1件ずつに直列化されるので、打ち切られた呼び出しの
	// 積み残しは高々1件。バッファ1で足りる。
	respCh chan scanResult
	// retired はこのプロセスをもう使わないことを示す。閉じるとリーダーが抜ける。
	retired chan struct{}
	// retireOnce は retired の二重closeを防ぐ。
	retireOnce sync.Once
	// readerDone はリーダーが抜けきったことを示す。
	// stdoutを読んでいる最中に cmd.Wait() を呼ぶのは os/exec が禁じているので、
	// Close はこれを待ってから Wait する。
	readerDone chan struct{}
}

const (
	// pluginCallTimeout はプラグインに応答を待つ既定の期限。
	// この期限を超えたら「プラグインが詰まっている」とみなしてプロセスを回収する。
	pluginCallTimeout = 30 * time.Second

	// maxPluginQueueWait は実行スロットが空くのを待つ上限。
	// ここで待ちきれなかったのは「混んでいる」だけなので、
	// プロセスには手を出さずにビジーとして返す。
	maxPluginQueueWait = 10 * time.Second

	// pluginGPSLogPageSize は get_gps_logs 1回で受け取る点数。
	// 1点はJSONで約95バイト。親の bufio.Scanner は32MBなので、
	// 余裕をみて10万点（約9.5MB）で切る。
	pluginGPSLogPageSize = 100000

	// maxPluginGPSLogPoints は1回の取得で受け取る合計の上限。
	// ロケーション履歴の全件書き出しは数百万点になることがあり、
	// 際限なく受け取るとgkillのヒープを食い潰す。
	maxPluginGPSLogPoints = 2000000
)

// ErrPluginBusy はプラグインの実行スロットが空かずに待ちきれなかったことを表す。
// プラグインが壊れているわけではないので、プロセスは回収しない。
var ErrPluginBusy = errors.New("plugin is busy")

// pluginRepositoryImpl は PluginRepository インターフェースの実装。
// プラグインバイナリをサブプロセスとして起動し、stdio 改行区切りJSONで通信する。
type pluginRepositoryImpl struct {
	// callSlot は容量1のチャネルで、プラグインへの操作を1件ずつに直列化する。
	// ミューテックスではなくチャネルなのは「待つのをやめられる」ようにするため。
	// 待ちを打ち切れないと、行列に並んでいる間に期限を食い潰し、
	// 応答しているだけのプラグインを期限切れとして殺してしまう。
	callSlot     chan struct{}
	callSlotOnce sync.Once

	userID    string // gkillユーザID
	pluginDir string // $GKILL_HOME/plugins/{userID}/{pluginName}/
	manifest  gkill_plugin.PluginManifest

	proc *pluginProcess // nil = 未起動

	// typedIndex は manifest.provides が空でないときだけ作られる型別/付随データの索引。
	// providesを宣言していないプラグイン（chatgpt/claudeai/claudecode/example）ではnilのままで、
	// UpdateCacheも従来どおり何もしない。
	typedIndex *PluginTypedIndex
}

// インターフェース適合確認（コンパイル時チェック）
var _ PluginRepository = (*pluginRepositoryImpl)(nil)
var _ pluginIndexSource = (*pluginRepositoryImpl)(nil)

// NewPluginRepository はプラグインリポジトリを作成する。
// プロセスは初回クエリ時に遅延起動する。
func NewPluginRepository(userID string, pluginDir string, manifest gkill_plugin.PluginManifest) PluginRepository {
	rep := &pluginRepositoryImpl{
		callSlot:  make(chan struct{}, 1),
		userID:    userID,
		pluginDir: pluginDir,
		manifest:  manifest,
	}
	if len(manifest.Provides) != 0 {
		rep.typedIndex = newPluginTypedIndex(rep)
	}
	return rep
}

// TypedIndex は型別データ・付随データの索引を返す。providesが空ならnil。
func (p *pluginRepositoryImpl) TypedIndex() *PluginTypedIndex {
	return p.typedIndex
}

// indexRepName は索引用にリポジトリ表示名を返す。
func (p *pluginRepositoryImpl) indexRepName() string {
	return p.manifest.RepName
}

// indexPluginName は索引用にプラグイン名を返す。
func (p *pluginRepositoryImpl) indexPluginName() string {
	return p.manifest.Name
}

// indexProvidedKinds は索引用に manifest.provides の集合を返す。
func (p *pluginRepositoryImpl) indexProvidedKinds() map[gkill_plugin.PluginProvidedKind]struct{} {
	return p.manifest.ProvidedKinds()
}

// indexFetchAll は索引の材料を1回のfind_kyousで取ってくる。
func (p *pluginRepositoryImpl) indexFetchAll(ctx context.Context) ([]gkill_plugin.PluginKyou, error) {
	return p.findPluginKyous(ctx, &gkill_plugin.PluginQuery{OnlyLatestData: true})
}

// slot は実行スロットのチャネルを返す。
// ゼロ値の pluginRepositoryImpl から使われても動くように遅延初期化する。
func (p *pluginRepositoryImpl) slot() chan struct{} {
	p.callSlotOnce.Do(func() {
		if p.callSlot == nil {
			p.callSlot = make(chan struct{}, 1)
		}
	})
	return p.callSlot
}

// acquireCallSlot は実行スロットを取得する。
// wait を超えても空かなければ ErrPluginBusy を返す。
// ctx のキャンセルでも待つのをやめる（呼び出し元が結果を要らなくなっただけなので
// プロセスには手を出さない。行列が短くなるぶんむしろ望ましい）。
func (p *pluginRepositoryImpl) acquireCallSlot(ctx context.Context, wait time.Duration) (release func(), err error) {
	slot := p.slot()
	select {
	case slot <- struct{}{}:
		return func() { <-slot }, nil
	default:
	}

	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case slot <- struct{}{}:
		return func() { <-slot }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-timer.C:
		return nil, fmt.Errorf("plugin %s: %w", p.manifest.Name, ErrPluginBusy)
	}
}

// ensureStarted はプラグインプロセスが起動していることを保証する。
// 呼び出し側で実行スロットを取得済みであること。
// プロセスの寿命はリクエストコンテキストに依存させないよう context.Background() を使う。
func (p *pluginRepositoryImpl) ensureStarted() error {
	if p.proc != nil && p.proc.started {
		return nil
	}
	// 差し替える前に古いプロセスを片付ける。
	// リーダーが応答の送信でブロックしたまま残らないようにする。
	if p.proc != nil {
		p.retire(p.proc)
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

	proc := &pluginProcess{
		cmd:        cmd,
		stdin:      stdin,
		scanner:    scanner,
		started:    true,
		respCh:     make(chan scanResult, 1),
		retired:    make(chan struct{}),
		readerDone: make(chan struct{}),
	}
	p.proc = proc
	go p.readLoop(proc)

	slog.Info(fmt.Sprintf("plugin started: %q (user=%q)", p.manifest.Name, p.userID))
	return nil
}

// readLoop はプロセスごとに1本だけ走る常駐リーダー。
// scanner を触るのはこのgoroutineだけで、リクエスト側は respCh 経由で受け取る。
// retired が閉じられたら、誰も読んでいなくても抜ける。
func (p *pluginRepositoryImpl) readLoop(proc *pluginProcess) {
	defer close(proc.readerDone)
	for {
		if !proc.scanner.Scan() {
			var result scanResult
			if scanErr := proc.scanner.Err(); scanErr != nil {
				result = scanResult{err: fmt.Errorf("error at read from plugin stdout %s: %w", p.manifest.Name, scanErr)}
			} else {
				result = scanResult{err: fmt.Errorf("plugin %s closed stdout unexpectedly", p.manifest.Name)}
			}
			p.deliver(proc, result)
			return
		}
		b := make([]byte, len(proc.scanner.Bytes()))
		copy(b, proc.scanner.Bytes())
		if !p.deliver(proc, scanResult{data: b}) {
			return
		}
	}
}

// deliver は応答をリクエスト側へ渡す。渡せたら true。
// 誰も読んでいなくても retired で必ず抜けられるようにして、リーダーを残さない。
func (p *pluginRepositoryImpl) deliver(proc *pluginProcess, result scanResult) bool {
	select {
	case proc.respCh <- result:
		return true
	case <-proc.retired:
		return false
	}
}

// retire はプロセスを使用終了にする。リーダーを解放し、プロセスを強制終了する。
// 複数回呼んでも安全。呼び出し側で実行スロットを取得済みであること。
func (p *pluginRepositoryImpl) retire(proc *pluginProcess) {
	proc.retireOnce.Do(func() { close(proc.retired) })
	proc.started = false
	if proc.cmd.Process != nil {
		_ = proc.cmd.Process.Kill()
	}
}

// scanResult は scanner.Scan() の結果を goroutine 間で受け渡すための型。
type scanResult struct {
	data []byte
	err  error
}

// sendRequest は改行区切りJSONでリクエストを送り、自分宛てのレスポンスを受け取る。
// 呼び出し前に実行スロットを取得済みであること。
//
// 打ち切りの契機を2つに分けているのが要点。
//   - callerCtx: HTTPクライアントの切断やサーバ終了。待つのをやめるだけで、
//     プラグインプロセスには手を出さない。遅れて届く応答は次の呼び出しが
//     レスポンスIDの不一致で読み捨てる。
//   - timeoutCtx: gkill自身が定めた期限（既定30秒 / IsAliveの5秒）。
//     応答が返らない＝プラグインが詰まっているということなので、
//     プロセスを回収する。回収しないと以降の呼び出しも全部詰まったままになる。
func (p *pluginRepositoryImpl) sendRequest(callerCtx context.Context, timeoutCtx context.Context, req gkill_plugin.PluginRequest) (*gkill_plugin.PluginResponse, error) {
	// リーダーと共有するのはこのローカル変数だけにする。
	// p.proc を直接参照すると ensureStarted の差し替えと競合する。
	proc := p.proc

	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error at marshal plugin request: %w", err)
	}

	if _, err := fmt.Fprintf(proc.stdin, "%s\n", reqBytes); err != nil {
		proc.started = false
		return nil, fmt.Errorf("error at write to plugin stdin %s: %w", p.manifest.Name, err)
	}

	for {
		select {
		case <-timeoutCtx.Done():
			p.retire(proc)
			return nil, fmt.Errorf("plugin %s request timed out: %w", p.manifest.Name, timeoutCtx.Err())
		case <-callerCtx.Done():
			// 呼び出し元にDeadlineが設定されていた場合、timeoutCtx はその同じ期限を
			// 引き継いでいるので、両方がほぼ同時にDoneになりselectがどちらを選ぶか決まらない。
			// タイミングではなくエラーの種類で判定する。
			//   DeadlineExceeded … 「これ以上待たない」という判断（IsAliveの5秒など）
			//                       なのでプラグインを回収する
			//   Canceled        … 呼び出し元が結果を要らなくなっただけ。プロセスには触らない
			if errors.Is(callerCtx.Err(), context.DeadlineExceeded) {
				p.retire(proc)
				return nil, fmt.Errorf("plugin %s request timed out: %w", p.manifest.Name, callerCtx.Err())
			}
			return nil, fmt.Errorf("plugin %s request canceled: %w", p.manifest.Name, callerCtx.Err())
		case result, ok := <-proc.respCh:
			if !ok {
				proc.started = false
				return nil, fmt.Errorf("plugin %s closed stdout unexpectedly", p.manifest.Name)
			}
			if result.err != nil {
				proc.started = false
				return nil, result.err
			}
			var resp gkill_plugin.PluginResponse
			if err := json.Unmarshal(result.data, &resp); err != nil {
				return nil, fmt.Errorf("error at unmarshal plugin response %s: %w", p.manifest.Name, err)
			}
			// 打ち切った呼び出しの応答。読み捨てて自分の応答を待つ。
			// IDが空のものはSDKのパースエラー応答（writeError(encoder, "", ...)）なので
			// 自分宛てとして扱う。捨ててしまうと不正入力のたびに期限まで待つことになる。
			if resp.ID != "" && resp.ID != req.ID {
				slog.Debug(fmt.Sprintf("plugin %q discarded stale response %q", p.manifest.Name, resp.ID))
				continue
			}
			if len(resp.Errors) > 0 {
				return &resp, fmt.Errorf("plugin %s returned errors: %v", p.manifest.Name, resp.Errors)
			}
			return &resp, nil
		}
	}
}

// callCommand は実行スロットを取り、ensureStarted・sendRequest・クラッシュ時リトライをまとめて実行する。
// スロットで全操作を直列化することで並列リクエストによる競合を防ぐ。
//
// contextは2つに分ける。呼び出し元の ctx はそのまま「呼び出し元が結果を待つのをやめたか」
// を表し、timeoutCtx は「gkill自身がプラグインを見限る期限」を表す。
// 呼び出し元のキャンセル（HTTPクライアントの切断）を後者に混ぜると、
// 画面を操作しただけでプラグインプロセスが回収されてしまう。
//
// 順序が要点。
//  1. まず行列に並ぶ。混んでいるだけならビジーとして返し、プロセスには手を出さない。
//  2. スロットを取ってから、初めてプラグイン自身の期限を張る。
//
// 期限をスロット取得より前に張ると、行列で待っているだけで期限を食い潰し、
// 正常に応答しているプラグインを期限切れとして殺してしまう。
// 一覧の行数ぶんの本文取得が同時に来たときに実際にこれが起きた。
func (p *pluginRepositoryImpl) callCommand(ctx context.Context, req gkill_plugin.PluginRequest) (*gkill_plugin.PluginResponse, error) {
	// 呼び出し元が期限を設定していれば、その「残り時間」を実行予算として引き継ぐ。
	// 期限そのものを引き継ぐと、行列で待っている間に予算を食い潰してしまう。
	executionBudget := pluginCallTimeout
	queueWait := maxPluginQueueWait
	if deadline, hasDeadline := ctx.Deadline(); hasDeadline {
		executionBudget = time.Until(deadline)
		if executionBudget <= 0 {
			return nil, fmt.Errorf("plugin %s request timed out: %w", p.manifest.Name, context.DeadlineExceeded)
		}
		// 期限が短い呼び出し（IsAliveの5秒など）は行列でも長く待たない
		queueWait = min(queueWait, executionBudget)
	}

	release, err := p.acquireCallSlot(ctx, queueWait)
	if err != nil {
		return nil, err
	}
	defer release()

	// 呼び出し元のキャンセルは引き継がず、期限だけをここから張り直す
	timeoutCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), executionBudget)
	defer cancel()

	if err := p.ensureStarted(); err != nil {
		return nil, err
	}

	resp, err := p.sendRequest(ctx, timeoutCtx, req)
	if err != nil {
		// 打ち切り（呼び出し元のキャンセル・期限切れ）はリトライしない
		if ctx.Err() != nil || timeoutCtx.Err() != nil {
			return nil, err
		}
		// プロセスクラッシュ時のみ1回リトライ（自動再起動）
		slog.Warn(fmt.Sprintf("plugin %q error, retrying: %q", p.manifest.Name, err))
		p.retire(p.proc)
		if startErr := p.ensureStarted(); startErr != nil {
			return nil, fmt.Errorf("plugin restart failed %s: %w (original: %v)", p.manifest.Name, startErr, err)
		}
		resp, err = p.sendRequest(ctx, timeoutCtx, req)
		if err != nil {
			return nil, err
		}
	}
	return resp, nil
}

// --- Repository interface 実装 ---

// findPluginKyous はfind_kyousを1回投げて生のPluginKyouを返す。
// FindKyous と索引の再構築(indexFetchAll)で共有する。
func (p *pluginRepositoryImpl) findPluginKyous(ctx context.Context, pq *gkill_plugin.PluginQuery) ([]gkill_plugin.PluginKyou, error) {
	req := gkill_plugin.PluginRequest{
		ID:      uuid.New().String(),
		Command: "find_kyous",
		Query:   pq,
	}
	resp, err := p.callCommand(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.Kyous, nil
}

func (p *pluginRepositoryImpl) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]Kyou, error) {
	pluginKyous, err := p.findPluginKyous(ctx, findQueryToPluginQuery(query))
	if err != nil {
		slog.Error(fmt.Sprintf("plugin find_kyous error %q: %q", p.manifest.Name, err))
		// プラグイン障害で検索全体を落とさない方針は維持しつつ、
		// 「静かな欠落」にならないよう警告として記録する(呼び出し元がメッセージ表示に使う)
		AppendPluginFindWarning(ctx, p.manifest.Name)
		return map[string][]Kyou{p.manifest.RepName: {}}, nil
	}

	kyous := make([]Kyou, 0, len(pluginKyous))
	for _, pk := range pluginKyous {
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

func (p *pluginRepositoryImpl) UpdateCache(ctx context.Context) error {
	// providesを宣言していないプラグインでは索引を持たない。
	// 従来どおり何もしないことで、既存プラグインに新たなfind_kyousを発生させない。
	if p.typedIndex == nil {
		return nil
	}
	return p.typedIndex.Refresh(ctx)
}

// GetPluginGPSLogs は期間に含まれるGPSログをプラグインから取得する。
func (p *pluginRepositoryImpl) GetPluginGPSLogs(ctx context.Context, startTime *time.Time, endTime *time.Time) ([]GPSLog, error) {
	if !p.manifest.ProvidesKind(gkill_plugin.PluginProvidesGPSLog) {
		return nil, fmt.Errorf("plugin %s does not provide gpslog", p.manifest.Name)
	}
	start, end := NormalizeGPSLogPeriod(startTime, endTime)

	gpsLogs := []GPSLog{}
	offset := 0
	for {
		resp, err := p.callCommand(ctx, gkill_plugin.PluginRequest{
			ID:      uuid.New().String(),
			Command: "get_gps_logs",
			GPSLogQuery: &gkill_plugin.PluginGPSLogQuery{
				StartTime: start,
				EndTime:   end,
				Offset:    offset,
				Limit:     pluginGPSLogPageSize,
			},
		})
		if err != nil {
			return nil, err
		}
		for _, pluginGPSLog := range resp.GPSLogs {
			gpsLogs = append(gpsLogs, GPSLog{
				RelatedTime: pluginGPSLog.RelatedTime,
				Longitude:   pluginGPSLog.Longitude,
				Latitude:    pluginGPSLog.Latitude,
			})
		}
		// 「続きがある」と言いながら0件を返すプラグインで無限ループにしない
		if !resp.HasMoreGPSLogs || len(resp.GPSLogs) == 0 {
			break
		}
		offset += len(resp.GPSLogs)
		if len(gpsLogs) >= maxPluginGPSLogPoints {
			slog.Warn(fmt.Sprintf("plugin %q returned more than %d gps logs, truncated", p.manifest.Name, maxPluginGPSLogPoints))
			break
		}
	}
	return gpsLogs, nil
}

func (p *pluginRepositoryImpl) GetLatestDataRepositoryAddress(_ context.Context, _ bool) ([]gkill_cache.LatestDataRepositoryAddress, error) {
	return []gkill_cache.LatestDataRepositoryAddress{}, nil
}

func (p *pluginRepositoryImpl) Close(ctx context.Context) error {
	// シャットダウン時なので実行中の呼び出しが終わるまで待つ。
	// 期限切れの呼び出しはプロセスを回収してスロットを手放すので、待ち続けることはない。
	release, err := p.acquireCallSlot(ctx, pluginCallTimeout)
	if err != nil {
		return fmt.Errorf("error at acquire plugin call slot for close %s: %w", p.manifest.Name, err)
	}
	defer release()

	if p.proc == nil || !p.proc.started {
		return nil
	}

	proc := p.proc

	req := gkill_plugin.PluginRequest{
		ID:      uuid.New().String(),
		Command: "close",
	}
	reqBytes, _ := json.Marshal(req)
	fmt.Fprintf(proc.stdin, "%s\n", reqBytes) //nolint:errcheck

	// stdoutを読んでいる最中に cmd.Wait() を呼ぶのは os/exec が禁じている
	// （Waitがパイプを閉じてしまうため）。プロセス終了→リーダーがEOFで抜ける、
	// の順を待ってから Wait する。
	//
	// 待つ間は respCh を読み捨てる。closeコマンドの応答と、その後のEOF通知を
	// 誰も受け取らないと、リーダーが respCh への送信でブロックしたまま
	// readerDone が閉じられない。
	timeout := time.After(5 * time.Second)
	for waiting := true; waiting; {
		select {
		case <-proc.readerDone:
			waiting = false
		case <-proc.respCh:
			// closeの応答やEOF通知。もう誰も使わないので捨てる。
		case <-timeout:
			slog.Warn(fmt.Sprintf("plugin %q did not exit in time, killing", p.manifest.Name))
			p.retire(proc)
			<-proc.readerDone
			waiting = false
		}
	}
	if err := proc.cmd.Wait(); err != nil {
		slog.Debug(fmt.Sprintf("plugin %q exited with error: %q", p.manifest.Name, err))
	}

	// 以後このプロセスは使わない。リーダーが残らないよう必ず解放する。
	p.retire(proc)
	slog.Info(fmt.Sprintf("plugin closed: %q", p.manifest.Name))
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
//
// Kyouはメタ情報しか持たないので、Tags / Texts / Typed / Notifications は
// ここでは落ちる。それらは pluginTypedIndex が別途 reps.Tag などに組み立て、
// 型別リポジトリのアダプタ経由で配る（manifest.jsonのprovidesに書いた種別だけ）。
//
// ImageSource だけは Kyou に置き場所が無いが、画像かどうかは IsImage で表現できる。
// 空でなければ画像として扱い、一覧の画像表示から漏れないようにする。
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
		IsImage:      pk.ImageSource != "",
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
	if q.HasWordFilter() {
		pq.Words = q.Words
		pq.NotWords = q.NotWords
		pq.WordsAnd = q.WordsAnd
	}
	if q.Tags != nil {
		pq.Tags = q.Tags
		pq.NotTags = q.HideTags
		pq.TagsAnd = q.TagsAnd
	}
	if q.HasCalendarFilter() {
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
	if q.CalendarStartDate != nil && kyou.RelatedTime.Before(*q.CalendarStartDate) {
		return false
	}
	if q.CalendarEndDate != nil && kyou.RelatedTime.After(*q.CalendarEndDate) {
		return false
	}
	// IDフィルタ:
	// findQueryToPluginQueryはIDsをPluginQueryに変換しないため、
	// プラグインはID指定クエリを受けても全件返す。
	// gkill側でIDフィルタを補完することで、
	// textMatchFindByIDQueryでプラグイン全件が混入するのを防ぐ。
	if q.IDs != nil {
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
