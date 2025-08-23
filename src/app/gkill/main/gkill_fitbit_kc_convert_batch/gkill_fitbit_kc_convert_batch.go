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
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mt3hr/gkill/src/app/gkill/dao/reps"
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
	FastNoDBCheck bool // 速攻モード（初回一括投入など）: DB照会なし・常に挿入
}

func parseArgs() Args {
	a := Args{}
	flag.StringVar(&a.FitbitPath, "fitbit", "", "Fitbit export zip or directory (required)")
	flag.StringVar(&a.KCDBPath, "kc_db", "", "KC sqlite3 path (required)")
	flag.StringVar(&a.TagDBPath, "tag_db", "", "TAG sqlite3 path (required)")
	flag.StringVar(&a.User, "user", "", "CREATE/UPDATE_USER (required)")
	flag.StringVar(&a.Device, "device", "PixelWatch2", "CREATE/UPDATE_DEVICE (default PixelWatch2)")
	flag.StringVar(&a.SourceTZ, "source_tz", "Asia/Tokyo", "timezone for naive timestamps")
	flag.IntVar(&a.ParseWorkers, "parse_workers", runtime.NumCPU(), "number of CSV parse workers")
	flag.BoolVar(&a.FastNoDBCheck, "fast_no_dbcheck", false, "skip DB existence checks (useful for first-time bulk load)")
	flag.Parse()

	if a.FitbitPath == "" || a.KCDBPath == "" || a.TagDBPath == "" || a.User == "" {
		fmt.Println("usage: --fitbit <zip_or_dir> --kc_db <sqlite> --tag_db <sqlite> --user <name> [--device PixelWatch2] [--source_tz Asia/Tokyo] [--parse_workers N] [--fast_no_dbcheck]")
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
	// 非常に軽いヒューリスティック
	for _, s := range samples {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if strings.ContainsAny(s, "Z+-") && strings.Contains(s, "T") {
			// 例: 2025-08-22T09:00:01+09:00 / Z
			tp.layout = time.RFC3339
			tp.naive = false
			tp.addSeconds = strings.Count(s, ":") == 2 // 2個→秒あり
			return tp
		}
		if strings.Contains(s, "/") {
			// 例: 01/02/2006 15:04:05
			tp.layout = "01/02/2006 15:04:05"
			tp.naive = true
			tp.addSeconds = strings.Count(s, ":") == 1
			return tp
		}
		if strings.Contains(s, " ") && strings.Contains(s, "-") {
			// 例: 2006-01-02 15:04:05
			tp.layout = "2006-01-02 15:04:05"
			tp.naive = true
			tp.addSeconds = strings.Count(s, ":") == 1
			return tp
		}
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
	reHeart = regexp.MustCompile(`(?i)heartrate|heart_rate|hr.*\.csv$`)
	reStep  = regexp.MustCompile(`(?i)minute[_-]?steps|steps.*\.csv$`)
	reCal   = regexp.MustCompile(`(?i)minute[_-]?calories|calories.*\.csv$`)
)

type metricDef struct {
	Key       string
	Title     string
	TimeCols  []string
	ValueCols []string
}

func detectMetric(pathLower string) (metricDef, bool) {
	switch {
	// 心拍: 候補列を拡張
	case reHeart.MatchString(pathLower):
		return metricDef{
			Key:   "heart_rate",
			Title: "心拍",
			// ここに timestamp を追加
			TimeCols: []string{"time", "datetime", "timestamp", "date"},
			// ここに beatsperminute を追加（これが今回の主流）
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

	seenKCMaxUpd map[string]time.Time // ID -> MAX(UPDATE_TIME)
	seenTag      map[string]struct{}  // TargetIDがwatch_log済み
	mu           sync.Mutex
}

func newKCWriter(kc reps.KCRepository, tags reps.TagRepository, user, device string, fast bool) *KCWriter {
	return &KCWriter{
		KCRepo:       kc,
		Tags:         tags,
		User:         user,
		Device:       device,
		FastNoCheck:  fast,
		seenKCMaxUpd: make(map[string]time.Time, 1<<14),
		seenTag:      make(map[string]struct{}, 1<<14),
	}
}

func (w *KCWriter) makeKCID(metric string, related time.Time) string {
	key := fmt.Sprintf("fitbit|%s|%s", metric, fmtUTC00(related))
	sum := sha256.Sum256([]byte(key))
	return fmt.Sprintf("%x", sum[:])
}

func (w *KCWriter) handle(ctx context.Context, r rec) error {
	id := w.makeKCID(r.metricKey, r.related)
	newUpd := r.related

	// ------- KC べき等チェック -------
	needInsert := true
	if !w.FastNoCheck {
		w.mu.Lock()
		if prev, ok := w.seenKCMaxUpd[id]; ok {
			if !newUpd.After(prev) {
				needInsert = false
			}
		}
		w.mu.Unlock()

		if needInsert {
			// 未キャッシュ → DB照会は1度だけ
			histories, err := w.KCRepo.GetKCHistories(ctx, id)
			if err != nil {
				return fmt.Errorf("get kc histories: %w", err)
			}
			var maxDB time.Time
			for _, h := range histories {
				if h.UpdateTime.After(maxDB) {
					maxDB = h.UpdateTime
				}
			}
			if !newUpd.After(maxDB) {
				needInsert = false
			} else {
				w.mu.Lock()
				w.seenKCMaxUpd[id] = newUpd
				w.mu.Unlock()
			}
		}
	}

	// ------- KC insert -------
	inserted := false
	if needInsert {
		kc := &reps.KC{
			IsDeleted:    false,
			ID:           id,
			RelatedTime:  newUpd, // RELATED=サンプル時刻
			CreateTime:   newUpd,
			CreateApp:    appName,
			CreateDevice: w.Device,
			CreateUser:   w.User,
			UpdateTime:   newUpd,
			UpdateApp:    appName,
			UpdateDevice: w.Device,
			UpdateUser:   w.User,
			Title:        r.title,
			NumValue:     json.Number(r.valueStr),
		}
		if err := w.KCRepo.AddKCInfo(ctx, kc); err != nil {
			return fmt.Errorf("add kc: %w", err)
		}
		inserted = true
	}

	// ------- TAG insert（新規KCのときだけ確認） -------
	if inserted {
		// キャッシュ先に見る
		w.mu.Lock()
		_, seen := w.seenTag[id]
		w.mu.Unlock()
		needTag := !seen

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
			tagID := sha256.Sum256([]byte(id + "|" + fixedTag + "|" + fmtUTC00(newUpd)))
			tag := &reps.Tag{
				IsDeleted:    false,
				ID:           fmt.Sprintf("%x", tagID[:]),
				TargetID:     id,
				Tag:          fixedTag,
				RelatedTime:  newUpd,
				CreateTime:   newUpd,
				CreateApp:    appName,
				CreateDevice: w.Device,
				CreateUser:   w.User,
				UpdateTime:   newUpd,
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
	}
	return nil
}

// -------------- Main runner --------------

func run(args Args) error {
	ctx := context.Background()

	// KC / Tag の本番リポジトリ
	kcRepo, err := reps.NewKCRepositorySQLite3Impl(ctx, args.KCDBPath) // ← 実装に合わせて必要なら差し替え
	if err != nil {
		return err
	}
	defer kcRepo.Close(ctx)

	tagRepo, err := reps.NewTagRepositorySQLite3Impl(ctx, args.TagDBPath)
	if err != nil {
		return err
	}
	defer tagRepo.Close(ctx)

	writer := newKCWriter(kcRepo, tagRepo, args.User, args.Device, args.FastNoDBCheck)

	// 入力
	src, closer, err := openSource(args.FitbitPath)
	if err != nil {
		return err
	}
	if closer != nil {
		defer closer()
	}

	// 対象CSVリストアップ
	var tasks []parseTask
	err = src.Walk(func(path string, open func() (io.ReadCloser, error)) error {
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
		fmt.Fprintln(os.Stderr, "no target CSVs found (heart_rate / steps / calories)")
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
				if err := parseFileToRecs(t, args.SourceTZ, recCh); err != nil {
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
		// 続行可能なエラーは標準エラーに流す
		fmt.Fprintln(os.Stderr, "warn:", e.Error())
		anyErr = e
	}
	// 重大エラーでなければ nil でOK（warnレベル）
	if anyErr != nil {
		panic(err)
	}
	return nil
}

// -------------- CSV -> recs (streaming) --------------

func parseFileToRecs(t parseTask, sourceTZ string, out chan<- rec) error {
	rc, err := t.open()
	if err != nil {
		return err
	}
	defer rc.Close()

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
		return fmt.Errorf("value column not found")
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
		if err := emitRow(row, t.md, ti, vi, tp, sourceTZ, out); err != nil {
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
			// ファイルの途中の壊れ行はスキップ
			continue
		}
		_ = emitRow(row, t.md, ti, vi, tp, sourceTZ, out)
	}
	return nil
}

func emitRow(row []string, md metricDef, ti, vi int, tp timeParser, sourceTZ string, out chan<- rec) error {
	if ti >= len(row) || vi >= len(row) {
		return errors.New("bad row")
	}
	tstr := strings.TrimSpace(row[ti])
	vstr := strings.TrimSpace(row[vi])
	if tstr == "" || vstr == "" {
		return errors.New("empty cell")
	}
	// 値は文字列のまま（カンマ除去）
	valStr := strings.ReplaceAll(vstr, ",", "")
	if _, err := strconv.ParseFloat(valStr, 64); err != nil {
		return err
	}
	// 時刻
	tt, err := parseWithTP(tstr, sourceTZ, tp)
	if err != nil {
		return err
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
