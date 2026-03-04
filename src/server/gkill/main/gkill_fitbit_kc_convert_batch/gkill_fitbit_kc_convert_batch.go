// gkill_fitbit_kc_convert_batch.go
package main

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mt3hr/gkill/src/app/gkill/dao/reps"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_log"
)

// ---------------- Consts ----------------

const (
	appName         = "gkill_fitbit_kc_convert_batch"
	fixedTag        = "watch_log"
	timeLayoutUTC00 = "2006-01-02T15:04:05+00:00" // +00:00 固定
)

// -------------- CLI args ---------------

type Args struct {
	FitbitPath    string
	KCDBPath      string
	TagDBPath     string
	User          string
	Device        string
	SourceTZ      string
	ParseWorkers  int
	FastNoDBCheck bool   // 速攻モード（初回一括投入など）: DB照会なし・常に挿入
	FromUTC       string // 例: 2025-01-01T00:00:00+00:00
	ToUTC         string // 例: 2025-08-01T00:00:00+00:00
}

func parseArgs() Args {
	a := Args{}
	flag.StringVar(&a.FitbitPath, "fitbit", "", "Fitbit export zip or directory (required)")
	flag.StringVar(&a.KCDBPath, "kc_db", "", "KC sqlite3 path (required)")
	flag.StringVar(&a.TagDBPath, "tag_db", "", "TAG sqlite3 path (required)")
	flag.StringVar(&a.User, "user", "", "CREATE/UPDATE_USER (required)")
	flag.StringVar(&a.Device, "device", "PixelWatch2", "CREATE/UPDATE_DEVICE (default PixelWatch2)")
	flag.StringVar(&a.SourceTZ, "source_tz", "Asia/Tokyo", "timezone for naive timestamps in CSV")
	flag.IntVar(&a.ParseWorkers, "parse_workers", runtime.NumCPU(), "number of CSV parse workers")
	flag.BoolVar(&a.FastNoDBCheck, "fast_no_dbcheck", false, "skip DB existence checks (useful for first-time bulk load)")
	flag.StringVar(&a.FromUTC, "from_utc", "", "process records with RELATED_TIME >= this UTC (RFC3339, +00:00)")
	flag.StringVar(&a.ToUTC, "to_utc", "", "process records with RELATED_TIME < this UTC (RFC3339, +00:00)")
	flag.Parse()

	if a.FitbitPath == "" || a.KCDBPath == "" || a.TagDBPath == "" || a.User == "" {
		fmt.Println("usage: --fitbit <zip_or_dir> --kc_db <sqlite> --tag_db <sqlite> --user <name> [--device PixelWatch2] [--source_tz Asia/Tokyo] [--parse_workers N] [--from_utc RFC3339] [--to_utc RFC3339] [--fast_no_dbcheck]")
		os.Exit(2)
	}
	if a.ParseWorkers < 1 {
		a.ParseWorkers = 1
	}
	return a
}

// -------------- File walking ------------

type fileSource interface {
	Walk(func(path string, open func() (io.ReadCloser, error)) error) error
}

type dirSource struct{ root string }

func (d dirSource) Walk(fn func(path string, open func() (io.ReadCloser, error)) error) error {
	return filepath.Walk(d.root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		p := path
		return fn(p, func() (io.ReadCloser, error) { return os.Open(p) })
	})
}

type zipSource struct{ zr *zip.ReadCloser }

func (z zipSource) Walk(fn func(path string, open func() (io.ReadCloser, error)) error) error {
	for _, f := range z.zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		ff := f
		if err := fn(ff.Name, func() (io.ReadCloser, error) {
			rc, err := ff.Open()
			if err != nil {
				return nil, err
			}
			var buf bytes.Buffer
			if _, err := io.Copy(&buf, rc); err != nil {
				rc.Close()
				return nil, err
			}
			rc.Close()
			return io.NopCloser(bytes.NewReader(buf.Bytes())), nil
		}); err != nil {
			return err
		}
	}
	return nil
}

func openSource(path string) (fileSource, func(), error) {
	fi, err := os.Stat(path)
	if err != nil {
		return nil, nil, err
	}
	if fi.IsDir() {
		return dirSource{root: path}, func() {}, nil
	}
	zr, err := zip.OpenReader(path)
	if err == nil {
		return zipSource{zr: zr}, func() { _ = zr.Close() }, nil
	}
	return nil, nil, errors.New("fitbit path must be a directory or a zip")
}

// -------------- Time utils --------------

func fmtUTC00(t time.Time) string { return t.UTC().Format(timeLayoutUTC00) }

type timeParser struct {
	layout     string
	naive      bool // true=オフセットなし（sourceTZで解釈）
	addSeconds bool // true=":ss"を補完
}

func decideTimeParser(samples []string) timeParser {
	tp := timeParser{}
	for _, s := range samples {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		// RFC3339系
		if strings.Contains(s, "T") && (strings.ContainsAny(s, "Z+-")) {
			tp.layout = time.RFC3339
			tp.naive = false
			tp.addSeconds = strings.Count(s, ":") == 2
			return tp
		}
		// 01/02/2006 15:04(:05)?
		if strings.Contains(s, "/") {
			tp.layout = "01/02/2006 15:04:05"
			tp.naive = true
			tp.addSeconds = strings.Count(s, ":") == 1
			return tp
		}
		// 2006-01-02 15:04(:05)?
		if strings.Contains(s, " ") && strings.Count(s, "-") == 2 {
			tp.layout = "2006-01-02 15:04:05"
			tp.naive = true
			tp.addSeconds = strings.Count(s, ":") == 1
			return tp
		}
		// 2006-01-02
		if len(s) == len("2006-01-02") && strings.Count(s, "-") == 2 {
			tp.layout = "2006-01-02"
			tp.naive = true
			return tp
		}
	}
	// フォールバック
	tp.layout = time.RFC3339
	tp.naive = false
	return tp
}

func parseWithTP(s, sourceTZ string, tp timeParser) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, errors.New("empty time string")
	}
	if tp.addSeconds && strings.Count(s, ":") == 1 {
		s = s + ":00"
	}
	if tp.naive {
		loc, err := time.LoadLocation(sourceTZ)
		if err != nil {
			loc = time.FixedZone("source", 9*3600)
		}
		t, err := time.ParseInLocation(tp.layout, s, loc)
		if err != nil {
			return time.Time{}, err
		}
		return t.UTC(), nil
	}
	t, err := time.Parse(tp.layout, s)
	if err != nil {
		return time.Time{}, err
	}
	return t.UTC(), nil
}

// -------------- CSV utils --------------

func newCSVReader(r io.Reader) *csv.Reader {
	cr := csv.NewReader(bufio.NewReader(r))
	cr.FieldsPerRecord = -1
	cr.ReuseRecord = true
	return cr
}

func normHeader(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "_", "")
	return s
}

func findCol(headers []string, candidates ...string) int {
	for i, h := range headers {
		nh := normHeader(h)
		for _, c := range candidates {
			if nh == c {
				return i
			}
		}
	}
	return -1
}

// -------------- Fitbit file detection --------
var (
	// 心拍CSV候補（hrvは除外を別判定にする）
	reHeartCSV = regexp.MustCompile(`(?i)(?:^|[\\/]).*?(heart[_-]?rate|heartrate).*\.csv$`)
	reHRVSub   = regexp.MustCompile(`(?i)(?:^|[\\/]).*?hrv.*`) // HRV を含むパスなら除外

	reStep = regexp.MustCompile(`(?i)(?:^|[\\/]).*?(minute[_-]?steps|steps).*\.csv$`)
	reCal  = regexp.MustCompile(`(?i)(?:^|[\\/]).*?(minute[_-]?calories|calories).*\.csv$`)

	// Fitbit配下だけに絞りたい場合（Windows/Unix両対応）
	reFitbitRoot = regexp.MustCompile(`(?i)(?:^|[\\/])fitbit[\\/]`)
)

type metricDef struct {
	Key       string
	Title     string
	TimeCols  []string
	ValueCols []string
}

func detectMetric(pathLower string) (metricDef, bool) {
	switch {
	case reHeartCSV.MatchString(pathLower) && !reHRVSub.MatchString(pathLower):
		return metricDef{
			Key:      "heart_rate",
			Title:    "心拍",
			TimeCols: []string{"time", "datetime", "timestamp", "date"},
			// Fitbitの主流 "Timestamp, Beats per minute" に対応
			ValueCols: []string{"beatsperminute", "value", "heartrate", "heartrate(bpm)", "heartratebpm", "heart rate", "bpm"},
		}, true
	case reStep.MatchString(pathLower):
		return metricDef{
			Key:       "steps",
			Title:     "歩数",
			TimeCols:  []string{"time", "datetime", "timestamp", "date"},
			ValueCols: []string{"steps", "value"},
		}, true
	case reCal.MatchString(pathLower):
		return metricDef{
			Key:       "calories",
			Title:     "消費カロリー",
			TimeCols:  []string{"time", "datetime", "timestamp", "date"},
			ValueCols: []string{"calories", "value", "kcal"},
		}, true
	default:
		return metricDef{}, false
	}
}

// -------------- Pipeline types --------------

type parseTask struct {
	path string
	open func() (io.ReadCloser, error)
	md   metricDef
}

type rec struct {
	metricKey string
	title     string
	valueStr  string    // 文字列のまま（json.Numberへ入れる）
	related   time.Time // UTC
}

// -------------- Writer (KC/Tag repositories) --------------

type KCWriter struct {
	KCRepo      reps.KCRepository
	Tags        reps.TagRepository
	User        string
	Device      string
	FastNoCheck bool
	RunAt       time.Time // バッチ起動時刻（UTC）

	// RELATED_TIMEベースの重複判定キャッシュ
	// 同一ID（= metric|RELATED_TIME|device のハッシュ）について、
	// 一度DBを引いたら「存在有無」と「最新値」を保持して以後はDB照会しない
	seenKC map[string]struct {
		exists  bool
		lastVal string
	}
	seenTag map[string]struct{}
	mu      sync.RWMutex

	// stats
	inserted map[string]int64
	skipped  map[string]int64
}

func newKCWriter(kc reps.KCRepository, tags reps.TagRepository, user, device string, fast bool, runAt time.Time) *KCWriter {
	return &KCWriter{
		KCRepo:      kc,
		Tags:        tags,
		User:        user,
		Device:      device,
		FastNoCheck: fast,
		RunAt:       runAt,
		seenKC: make(map[string]struct {
			exists  bool
			lastVal string
		}, 1<<14),
		seenTag:  make(map[string]struct{}, 1<<14),
		inserted: map[string]int64{},
		skipped:  map[string]int64{},
	}
}

func (w *KCWriter) makeKCID(metric string, related time.Time) string {
	key := fmt.Sprintf("fitbit|%s|%s", metric, fmtUTC00(related))
	sum := sha256.Sum256([]byte(key))
	return fmt.Sprintf("%x", sum[:])
}

func (w *KCWriter) handle(ctx context.Context, r rec) error {
	id := w.makeKCID(r.metricKey, r.related)

	// Fastモードは無検査で挿入（初回一括投入向け）
	if w.FastNoCheck {
		if err := w.insertKCAndTag(ctx, id, r); err != nil {
			return err
		}
		w.mu.Lock()
		w.inserted[r.metricKey]++
		w.mu.Unlock()
		return nil
	}

	// まずキャッシュを確認
	w.mu.Lock()
	cache, ok := w.seenKC[id]
	w.mu.Unlock()

	needInsert := false
	sameValue := false

	if ok && cache.exists {
		// 既に同RELATED_TIMEのIDが存在
		sameValue = (cache.lastVal == r.valueStr)
		needInsert = !sameValue // 値が変わっていれば「更新」として挿入
	} else {
		// DBを1回だけ確認（存在/最新値）
		histories, err := w.KCRepo.GetKCHistories(ctx, id)
		if err != nil {
			return fmt.Errorf("get kc histories: %w", err)
		}
		if len(histories) == 0 {
			// まだ存在しない ⇒ 新規挿入
			needInsert = true
			w.mu.Lock()
			w.seenKC[id] = struct {
				exists  bool
				lastVal string
			}{exists: true, lastVal: ""}
			w.mu.Unlock()
		} else {
			// 既存あり ⇒ 最新（UPDATE_TIMEが最大）の値を取得して比較
			latest := histories[0]
			for _, h := range histories[1:] {
				if h.UpdateTime.After(latest.UpdateTime) {
					latest = h
				}
			}
			lastVal := string(latest.NumValue)
			sameValue = (lastVal == r.valueStr)
			needInsert = !sameValue

			w.mu.Lock()
			w.seenKC[id] = struct {
				exists  bool
				lastVal string
			}{exists: true, lastVal: lastVal}
			w.mu.Unlock()
		}
	}

	if needInsert {
		if err := w.insertKCAndTag(ctx, id, r); err != nil {
			return err
		}
		w.mu.Lock()
		w.seenKC[id] = struct {
			exists  bool
			lastVal string
		}{exists: true, lastVal: r.valueStr}
		w.inserted[r.metricKey]++
		w.mu.Unlock()
	} else {
		w.mu.Lock()
		w.skipped[r.metricKey]++
		w.mu.Unlock()
	}

	return nil
}

func (w *KCWriter) insertKCAndTag(ctx context.Context, id string, r rec) error {
	runAt := w.RunAt // 起動時刻をCREATE/UPDATEに入れる

	kc := reps.KC{
		IsDeleted:    false,
		ID:           id,
		RelatedTime:  r.related, // サンプル時刻（UTC、+00:00）
		CreateTime:   runAt,     // ★ バッチ起動時刻
		CreateApp:    appName,
		CreateDevice: w.Device,
		CreateUser:   w.User,
		UpdateTime:   runAt, // ★ バッチ起動時刻
		UpdateApp:    appName,
		UpdateDevice: w.Device,
		UpdateUser:   w.User,
		Title:        r.title,
		NumValue:     json.Number(r.valueStr),
	}
	if err := w.KCRepo.AddKCInfo(ctx, kc); err != nil {
		return fmt.Errorf("add kc: %w", err)
	}

	// KCを挿入した時だけ、watch_logタグを（未付与なら）付与
	w.mu.Lock()
	_, tagSeen := w.seenTag[id]
	w.mu.Unlock()
	needTag := !tagSeen

	if needTag && !w.FastNoCheck {
		existing, err := w.Tags.GetTagsByTargetID(ctx, id)
		if err != nil {
			return fmt.Errorf("get tags by target: %w", err)
		}
		for _, t := range existing {
			if strings.EqualFold(t.Tag, fixedTag) {
				needTag = false
				break
			}
		}
	}

	if needTag {
		tagID := sha256.Sum256([]byte(id + "|" + fixedTag + "|" + fmtUTC00(r.related)))
		tag := reps.Tag{
			IsDeleted:    false,
			ID:           fmt.Sprintf("%x", tagID[:]),
			TargetID:     id,
			Tag:          fixedTag,
			RelatedTime:  r.related, // タグの関連時刻はサンプル時刻
			CreateTime:   runAt,     // ★ バッチ起動時刻
			CreateApp:    appName,
			CreateDevice: w.Device,
			CreateUser:   w.User,
			UpdateTime:   runAt, // ★ バッチ起動時刻
			UpdateApp:    appName,
			UpdateDevice: w.Device,
			UpdateUser:   w.User,
		}
		if err := w.Tags.AddTagInfo(ctx, tag); err != nil {
			return fmt.Errorf("add tag: %w", err)
		}
		w.mu.Lock()
		w.seenTag[id] = struct{}{}
		w.mu.Unlock()
	}
	return nil
}

// -------------- Main runner --------------

func run(args Args) error {
	ctx := context.Background()

	// バッチ起動時刻（UTC固定）
	runAt := time.Now().UTC()

	// 期間フィルタ
	var fromPtr, toPtr *time.Time
	if args.FromUTC != "" {
		t, err := time.Parse(time.RFC3339, args.FromUTC)
		if err != nil {
			return fmt.Errorf("--from_utc parse error: %w", err)
		}
		u := t.UTC()
		fromPtr = &u
	}
	if args.ToUTC != "" {
		t, err := time.Parse(time.RFC3339, args.ToUTC)
		if err != nil {
			return fmt.Errorf("--to_utc parse error: %w", err)
		}
		u := t.UTC()
		toPtr = &u
	}

	// KC / Tag の本番リポジトリ
	kcRepo, err := reps.NewKCRepositorySQLite3Impl(ctx, args.KCDBPath, true)
	if err != nil {
		return err
	}
	defer kcRepo.Close(ctx)

	tagRepo, err := reps.NewTagRepositorySQLite3Impl(ctx, args.TagDBPath, true)
	if err != nil {
		return err
	}
	defer tagRepo.Close(ctx)

	writer := newKCWriter(kcRepo, tagRepo, args.User, args.Device, args.FastNoDBCheck, runAt)

	// 入力（Fitbitのみ）
	src, closer, err := openSource(args.FitbitPath)
	if err != nil {
		return err
	}
	if closer != nil {
		defer closer()
	}

	// 対象CSVリストアップ（Fitbit配下のみ）
	var tasks []parseTask
	err = src.Walk(func(path string, open func() (io.ReadCloser, error)) error {
		if !reFitbitRoot.MatchString(path) {
			return nil
		}
		lower := strings.ToLower(path)
		md, ok := detectMetric(lower)
		if !ok {
			return nil
		}
		tasks = append(tasks, parseTask{path: path, open: open, md: md})
		return nil
	})
	if err != nil {
		return err
	}
	if len(tasks) == 0 {
		fmt.Fprintln(os.Stderr, "no target CSVs found (heart_rate / steps / calories) under Fitbit")
		return nil
	}

	// パイプライン
	recCh := make(chan rec, 8192)
	errCh := make(chan error, len(tasks)+4)

	// Writer goroutine（1本）
	var wgWrite sync.WaitGroup
	wgWrite.Add(1)
	go func() {
		defer wgWrite.Done()
		for r := range recCh {
			if err := writer.handle(ctx, r); err != nil {
				errCh <- err
			}
		}
	}()

	// Parser workers
	var wgParse sync.WaitGroup
	taskCh := make(chan parseTask, len(tasks))
	for i := 0; i < args.ParseWorkers; i++ {
		wgParse.Add(1)
		go func() {
			defer wgParse.Done()
			for t := range taskCh {
				if err := parseFileToRecs(t, args.SourceTZ, fromPtr, toPtr, recCh); err != nil {
					errCh <- fmt.Errorf("%s: %w", t.path, err)
				}
			}
		}()
	}
	for _, t := range tasks {
		taskCh <- t
	}
	close(taskCh)

	// 待機
	wgParse.Wait()
	close(recCh)
	wgWrite.Wait()
	close(errCh)

	// エラーまとめ
	var anyErr error
	for e := range errCh {
		fmt.Fprintln(os.Stderr, "warn:", e.Error())
		anyErr = e
	}

	// サマリ
	fmt.Fprintln(os.Stderr, "=== summary ===")
	for k, v := range writer.inserted {
		fmt.Fprintf(os.Stderr, "inserted %-12s : %d\n", k, v)
	}
	for k, v := range writer.skipped {
		fmt.Fprintf(os.Stderr, "skipped  %-12s : %d\n", k, v)
	}

	return anyErr
}

// -------------- CSV -> recs (streaming) --------------

func parseFileToRecs(t parseTask, sourceTZ string, fromPtr, toPtr *time.Time, out chan<- rec) error {
	rc, err := t.open()
	if err != nil {
		return err
	}
	defer func() {
		err := rc.Close()
		if err != nil {
			slog.Log(context.Background(), gkill_log.Debug, "error at defer close", "error", err)
		}
	}()

	cr := newCSVReader(rc)
	// ヘッダ
	hdr, err := cr.Read()
	if err != nil {
		return fmt.Errorf("read header: %w", err)
	}
	ti := findCol(hdr, t.md.TimeCols...)
	if ti < 0 {
		return fmt.Errorf("time column not found")
	}
	vi := findCol(hdr, t.md.ValueCols...)
	if vi < 0 {
		// フォールバック: bpmを含む列を拾う／なければ 2列目
		vi = fallbackValueIndex(hdr)
		if vi < 0 {
			return fmt.Errorf("value column not found")
		}
	}

	// タイムレイアウトの決定：先頭数行からサンプル収集
	var samples []string
	var bufRows [][]string
	for len(samples) < 10 {
		row, e := cr.Read()
		if e == io.EOF {
			break
		}
		if e != nil {
			return fmt.Errorf("read: %w", e)
		}
		bufRows = append(bufRows, row)
		if ti < len(row) {
			s := strings.TrimSpace(row[ti])
			if s != "" {
				samples = append(samples, s)
			}
		}
	}
	tp := decideTimeParser(samples)

	// バッファ分を処理
	for _, row := range bufRows {
		if err := emitRow(row, t.md, ti, vi, tp, sourceTZ, fromPtr, toPtr, out); err != nil {
			// 軽いエラーはスキップ
			continue
		}
	}
	// 残りをストリーミング処理
	for {
		row, e := cr.Read()
		if e == io.EOF {
			break
		}
		if e != nil {
			// ファイル途中の壊れ行はスキップ
			continue
		}
		_ = emitRow(row, t.md, ti, vi, tp, sourceTZ, fromPtr, toPtr, out)
	}
	return nil
}

func fallbackValueIndex(hdr []string) int {
	best := -1
	for i, h := range hdr {
		nh := normHeader(h)
		if strings.Contains(nh, "bpm") || strings.Contains(nh, "beatsperminute") || strings.Contains(nh, "heartrate") {
			return i
		}
		// 記述揺れ対策として2列目を弱いフォールバックに採用
		if i == 1 {
			best = 1
		}
	}
	return best
}

func emitRow(row []string, md metricDef, ti, vi int, tp timeParser, sourceTZ string, fromPtr, toPtr *time.Time, out chan<- rec) error {
	if ti >= len(row) || vi >= len(row) {
		return errors.New("bad row")
	}
	tstr := strings.TrimSpace(row[ti])
	vstr := strings.TrimSpace(row[vi])
	if tstr == "" || vstr == "" {
		return errors.New("empty cell")
	}
	// 値は文字列のまま（カンマ等のノイズ除去）
	valStr := strings.ReplaceAll(vstr, ",", "")
	if _, err := strconv.ParseFloat(valStr, 64); err != nil {
		return err
	}
	// 時刻（UTC）
	tt, err := parseWithTP(tstr, sourceTZ, tp)
	if err != nil {
		return err
	}
	// 期間フィルタ
	if fromPtr != nil && tt.Before(*fromPtr) {
		return nil
	}
	if toPtr != nil && !tt.Before(*toPtr) {
		return nil
	}
	out <- rec{
		metricKey: md.Key,
		title:     md.Title,
		valueStr:  valStr,
		related:   tt,
	}
	return nil
}

// ---------------- main ------------------

func main() {
	args := parseArgs()
	if err := run(args); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
