package main

import (
	"fmt"
	"os"
	"runtime"
	"sort"
	"strconv"
	"sync"
	"time"

	sdk "github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
)

// ビルダはプロセス内に1本だけ常駐する goroutine。
// ハンドラは kick を投げて、即座に「今キャッシュにあるもの」を返す。
//
// gkillのハンドラ期限は「30秒以内なら良い」ではなく「数十ミリ秒で返れ」を意味する。
//   - IsAlive の期限は5秒で、超えるとプロセスが殺される
//   - 一覧は find_kyous 1回のあとに行数ぶんの get_content_html が
//     1本のスロットに並ぶので、3秒かかるハンドラでも4件目から順番待ちが破綻する
//
// 実データ（約1GB・約1,900ファイル・約2,400万行）の初回構築は
// 環境によって10秒〜2分かかる。同期でやると必ず殺される。

const (
	// buildBatchFiles は1トランザクションで取り込むファイル数。
	// 途中で殺されても、このぶんだけやり直せばよい。
	buildBatchFiles = 64

	// foldBatchDays は1回の畳み直しで処理する日数。
	foldBatchDays = 2000

	// builderIdleInterval は何も無くても様子を見に行く間隔。
	builderIdleInterval = 5 * time.Minute
)

// builder はバックグラウンドでキャッシュを作り続ける常駐 goroutine。
type builder struct {
	// kick はバッファ1。ノンブロッキング送信で「作り直して」と伝える。
	kick chan struct{}

	startOnce sync.Once
}

var globalBuilder = &builder{kick: make(chan struct{}, 1)}

// EnsureStarted はビルダを起動する。何度呼んでも1本しか起きない。
func (b *builder) EnsureStarted(pluginDir string, configOf func() pluginConfig) {
	b.startOnce.Do(func() {
		go b.loop(pluginDir, configOf)
	})
}

// Kick は作り直しを促す。待たない。
func (b *builder) Kick() {
	select {
	case b.kick <- struct{}{}:
	default:
	}
}

func (b *builder) loop(pluginDir string, configOf func() pluginConfig) {
	ticker := time.NewTicker(builderIdleInterval)
	defer ticker.Stop()

	// 起動直後に1回走らせる
	b.runOnce(pluginDir, configOf())

	for {
		select {
		case <-b.kick:
		case <-ticker.C:
		}
		b.runOnce(pluginDir, configOf())
	}
}

// runOnce は走査→取り込み→畳み直しを1周する。
//
// os.Stdout には絶対に書かない。あれはプロトコルのチャネルで、
// 1行でも混ざるとJSONストリームが壊れる。ログはstderrに出す。
func (b *builder) runOnce(pluginDir string, config pluginConfig) {
	if err := globalCache.build(pluginDir, config); err != nil {
		fmt.Fprintf(os.Stderr, "gkill_plugin_fitbit: build error: %v\n", err)
	}
}

// build はキャッシュを最新にする。
func (c *cache) build(pluginDir string, config pluginConfig) error {
	// 構築どうしだけを直列化する。読み取りは待たせない
	// （初回構築は実データで数十秒かかるので、待たせるとハンドラが全部詰まる）。
	c.buildMu.Lock()
	defer c.buildMu.Unlock()

	if err := c.openDB(pluginDir); err != nil {
		return err
	}
	if err := c.resetIfGenerationChanged(config.Timezone); err != nil {
		return err
	}
	loc, err := loadLocation(config.Timezone)
	if err != nil {
		c.setMeta("build_state", "error")
		c.setMeta("build_error", fmt.Sprintf("タイムゾーン %q を読めませんでした: %v", config.Timezone, err))
		return fmt.Errorf("error at load location %s: %w", config.Timezone, err)
	}

	c.setMeta("build_state", "scanning")
	c.setMeta("build_error", "")

	// ZIPは読み終わるまで開いたままにする必要がある。
	sources, scanErr := openSources(config.Patterns)
	defer func() { _ = sources.Close() }()

	known, err := c.loadFileCache()
	if err != nil {
		return err
	}

	// 取り込む指標が絞られていれば、対象外の接頭辞のエントリは読まない
	enabled := config.enabledMetrics()
	targets := []sdk.SourceEntry{}
	for _, entry := range sources.Entries() {
		prefix, ok := metricPrefixOf(entry.Name)
		if !ok || len(defsForPrefix(prefix, enabled)) == 0 {
			continue
		}
		targets = append(targets, entry)
	}

	// 新しい順に処理する。直近のデータが数秒で見えるようにするため。
	sort.Slice(targets, func(i, j int) bool { return targets[i].Path > targets[j].Path })

	c.setMeta("target_file_count", strconv.Itoa(len(targets)))
	c.storeSourceProblems(sources.Problems())

	// 世代の反映は取り込みより先。ここで積まれた dirty_day も同じ周回で畳み直す。
	if err := c.syncExports(sources.Exports()); err != nil {
		return err
	}

	// 差分判定は (CRC32, Size)。更新時刻は使えない ――
	// Takeout は書き出し時刻を全エントリに同じ値で入れるので、
	// 中身が変わってもエントリの更新時刻は動かない。
	changed := []sdk.SourceEntry{}
	current := map[string]struct{}{}
	for _, entry := range targets {
		current[entry.Path] = struct{}{}
		previous, exist := known[entry.Path]
		if exist && previous.CRC32 == entry.CRC32 && previous.Size == entry.Size {
			continue
		}
		changed = append(changed, entry)
	}
	removed := []string{}
	for path := range known {
		if _, exist := current[path]; !exist {
			removed = append(removed, path)
		}
	}

	c.setMeta("build_total_files", strconv.Itoa(len(changed)))
	c.setMeta("build_done_files", "0")

	if len(changed) != 0 || len(removed) != 0 {
		c.setMeta("build_state", "ingesting")
	}

	if len(removed) != 0 {
		tx, err := c.db.Begin()
		if err != nil {
			return fmt.Errorf("error at begin remove tx: %w", err)
		}
		for _, path := range removed {
			if err := c.removeFileFromCache(tx, path); err != nil {
				_ = tx.Rollback()
				return err
			}
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("error at commit remove tx: %w", err)
		}
	}

	workers := config.ScanWorkers
	if workers <= 0 {
		workers = min(max(runtime.NumCPU()/2, 1), 4)
	}

	done := 0
	for start := 0; start < len(changed); start += buildBatchFiles {
		end := min(start+buildBatchFiles, len(changed))
		batch := changed[start:end]

		// 解析は並列、DB書き込みは1本にまとめる
		results := make([][]partialDaily, len(batch))
		var wg sync.WaitGroup
		semaphore := make(chan struct{}, workers)
		for i, entry := range batch {
			wg.Add(1)
			go func(i int, entry sdk.SourceEntry) {
				defer wg.Done()
				semaphore <- struct{}{}
				defer func() { <-semaphore }()
				prefix, _ := metricPrefixOf(entry.Name)
				partials, err := ingestEntry(entry, defsForPrefix(prefix, enabled), loc)
				if err != nil {
					fmt.Fprintf(os.Stderr, "gkill_plugin_fitbit: ingest %s: %v\n", entry.Path, err)
					return
				}
				results[i] = partials
			}(i, entry)
		}
		wg.Wait()

		tx, err := c.db.Begin()
		if err != nil {
			return fmt.Errorf("error at begin ingest tx: %w", err)
		}
		for i, entry := range batch {
			if err := c.ingestFileIntoCache(tx, scannedFileOf(entry), results[i]); err != nil {
				_ = tx.Rollback()
				return err
			}
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("error at commit ingest tx: %w", err)
		}

		done += len(batch)
		c.setMeta("build_done_files", strconv.Itoa(done))
	}

	c.setMeta("build_state", "folding")
	for {
		processed, err := c.foldDirtyDays(loc, foldBatchDays)
		if err != nil {
			return err
		}
		if processed == 0 {
			break
		}
	}

	c.setMeta("build_state", "idle")
	c.setMeta("last_scan_unix", strconv.FormatInt(time.Now().Unix(), 10))
	return scanErr
}

// defsForPrefix は接頭辞に対応する定義のうち、有効なものだけを返す。
// enabled が空なら全部有効。
func defsForPrefix(prefix string, enabled map[string]struct{}) []metricDef {
	defs := metricsByPrefix[prefix]
	if len(enabled) == 0 {
		return defs
	}
	filtered := make([]metricDef, 0, len(defs))
	for _, def := range defs {
		if _, ok := enabled[def.Key]; ok {
			filtered = append(filtered, def)
		}
	}
	return filtered
}
