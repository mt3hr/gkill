package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	sdk "github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
	_ "modernc.org/sqlite"
)

// cacheSchemaVersion はDDLの版。変えると下の表を作り直す。
const cacheSchemaVersion = "2"

// cache はSQLite3のキャッシュ。
//
// この Takeout の点数（約15,000点）ならメモリに持つだけで足りるが、
// 旧形式の Records.json は数百MB〜1GBで数百万点になる。
// プロセスが起動するたびに全部読み直すと、1回の呼び出しに許された時間に収まらない。
// ファイル単位（path, mtime, size）の差分更新にしておけば、
// 新しい月のファイルが増えてもその1本しか読まない。
type cache struct {
	// mu は db の遅延初期化だけを守る。**長い処理の間は握らない。**
	//
	// かつては走査も読み取りもこれ1本で直列化していたが、それだと
	// 走査(実データで数秒)のあいだ get_gps_logs が詰まる。
	// SQLiteはWALで開いてあり *sql.DB 自体も並行安全なので、
	// 走査中の読み取りは「そこまで取り込めたぶん」を返せばよい。
	mu sync.Mutex

	// scanMu は走査を1本に直列化する。読み取りはこれを待たない。
	scanMu sync.Mutex

	db *sql.DB

	// refreshing はバックグラウンドの走査が走行中かを表す。二重起動を防ぐ。
	refreshing atomic.Bool
}

var globalCache = &cache{}

// openDB はキャッシュDBを開く。
// gkill本体の接続ヘルパは journal_mode(DELETE) を固定するので使わない。
// この関数を通ったあとの c.db の読み出しは、どのgoroutineからでも安全。
// mu を取ってから代入しているので、mu を通った読み手には必ず見える。
func (c *cache) openDB(pluginDir string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.db != nil {
		return nil
	}
	dbPath := sdk.CacheDBPath(pluginDir)
	db, err := sql.Open("sqlite", "file:"+dbPath+
		"?_txlock=immediate"+
		"&_pragma=busy_timeout(6000)"+
		"&_pragma=journal_mode(WAL)"+
		"&_pragma=synchronous(NORMAL)"+
		"&_pragma=cache_size(-16000)"+
		"&_pragma=temp_store(MEMORY)")
	if err != nil {
		return fmt.Errorf("error at open cache db %s: %w", dbPath, err)
	}
	if err := initSchema(db); err != nil {
		_ = db.Close()
		return err
	}
	c.db = db
	return nil
}

func initSchema(db *sql.DB) error {
	if _, err := db.Exec(`
CREATE TABLE IF NOT EXISTS cache_meta (
  key   TEXT PRIMARY KEY,
  value TEXT NOT NULL
);`); err != nil {
		return fmt.Errorf("error at create cache_meta: %w", err)
	}

	var version string
	_ = db.QueryRow(`SELECT value FROM cache_meta WHERE key = 'schema_version'`).Scan(&version)
	if version != cacheSchemaVersion {
		if _, err := db.Exec(`
DROP TABLE IF EXISTS gps_point_cache;
DROP TABLE IF EXISTS file_cache;
DELETE FROM cache_meta;
`); err != nil {
			return fmt.Errorf("error at drop old cache tables: %w", err)
		}
		if _, err := db.Exec(`INSERT OR REPLACE INTO cache_meta(key,value) VALUES('schema_version', ?)`, cacheSchemaVersion); err != nil {
			return fmt.Errorf("error at write schema_version: %w", err)
		}
	}

	_, err := db.Exec(`
CREATE TABLE IF NOT EXISTS file_cache (
  path        TEXT PRIMARY KEY,   -- "<ZIP>!/<エントリ>"
  mtime_unix  INTEGER NOT NULL,   -- 表示用。世代内では全エントリ同値
  size        INTEGER NOT NULL,
  crc32       INTEGER NOT NULL,   -- 中身の変化はこれで見る
  format      TEXT NOT NULL,
  point_count INTEGER NOT NULL,
  scan_error  TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS gps_point_cache (
  related_time_unix_ms INTEGER NOT NULL,
  lat_e7               INTEGER NOT NULL,
  lng_e7               INTEGER NOT NULL,
  accuracy_mm          INTEGER NOT NULL,
  source               TEXT NOT NULL,
  device_id            TEXT NOT NULL,
  source_path          TEXT NOT NULL,
  format               TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_gps_time ON gps_point_cache(related_time_unix_ms);
CREATE INDEX IF NOT EXISTS idx_gps_src  ON gps_point_cache(source_path);
`)
	if err != nil {
		return fmt.Errorf("error at create cache tables: %w", err)
	}
	return nil
}

func (c *cache) meta(key string) string {
	if c.db == nil {
		return ""
	}
	value := ""
	_ = c.db.QueryRow(`SELECT value FROM cache_meta WHERE key = ?`, key).Scan(&value)
	return value
}

func (c *cache) setMeta(key string, value string) {
	if c.db == nil {
		return
	}
	_, _ = c.db.Exec(`INSERT OR REPLACE INTO cache_meta(key,value) VALUES(?,?)`, key, value)
}

// refresh はソースを走査して、変化したファイルだけを読み直す。
func (c *cache) refresh(pluginDir string, config pluginConfig) error {
	// 走査どうしだけを直列化する。読み取りは待たせない
	c.scanMu.Lock()
	defer c.scanMu.Unlock()
	return c.refreshLocked(pluginDir, config)
}

// kickRefresh はバックグラウンドの走査を1本だけ起こす。待たない。
//
// 呼び出し元はハンドラなので、ここでブロックしてはいけない。
// 走査が終わるまでは「いまキャッシュにあるぶん」が返る。
func (c *cache) kickRefresh(pluginDir string, config pluginConfig) {
	if !c.refreshing.CompareAndSwap(false, true) {
		return
	}
	go func() {
		defer c.refreshing.Store(false)
		if err := c.refresh(pluginDir, config); err != nil {
			fmt.Fprintln(stderrWriter, appName+": refresh: "+err.Error())
		}
	}()
}

func (c *cache) refreshLocked(pluginDir string, config pluginConfig) error {
	if err := c.openDB(pluginDir); err != nil {
		return err
	}

	known, err := c.loadFileCache()
	if err != nil {
		return err
	}

	// ZIPは読み終わるまで開いたままにする必要がある。
	sources, scanErr := openSources(config.Patterns)
	defer func() { _ = sources.Close() }()

	c.storeSourceInfo(sources.Exports(), sources.Problems())

	// 差分判定は (CRC32, Size)。更新時刻は使えない ――
	// Takeout は書き出し時刻を全エントリに同じ値で入れるので、
	// 中身が変わってもエントリの更新時刻は動かない。
	//
	// 変化していないエントリは形式を判定し直さない。これが無いと走査のたびに
	// 全エントリの先頭64KBを伸長することになり、実データでは1分以上かかる。
	changed := []sdk.SourceEntry{}
	formats := map[string]string{}
	current := map[string]struct{}{}
	for _, entry := range sources.Entries() {
		current[entry.Path] = struct{}{}
		previous, exist := known[entry.Path]
		if exist && previous.CRC32 == entry.CRC32 && previous.Size == entry.Size {
			continue
		}
		// 位置情報ではないと分かったエントリも formats に空で入れる。
		// file_cache に覚えておけば次の走査で先頭を読み直さずに済む
		// （Takeout には位置情報ではない .json が数千ある）。
		formats[entry.Path] = detectFormat(entry)
		changed = append(changed, entry)
	}
	removed := []string{}
	for path := range known {
		if _, exist := current[path]; !exist {
			removed = append(removed, path)
		}
	}

	if len(changed) == 0 && len(removed) == 0 {
		c.setMeta("last_scan_unix", strconv.FormatInt(time.Now().Unix(), 10))
		return scanErr
	}

	tx, err := c.db.Begin()
	if err != nil {
		return fmt.Errorf("error at begin refresh tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	for _, path := range removed {
		if _, err := tx.Exec(`DELETE FROM gps_point_cache WHERE source_path = ?`, path); err != nil {
			return fmt.Errorf("error at delete points of removed %s: %w", path, err)
		}
		if _, err := tx.Exec(`DELETE FROM file_cache WHERE path = ?`, path); err != nil {
			return fmt.Errorf("error at delete file_cache of removed %s: %w", path, err)
		}
	}

	for _, entry := range changed {
		file := scannedFileOf(entry, formats[entry.Path])
		if _, err := tx.Exec(`DELETE FROM gps_point_cache WHERE source_path = ?`, file.Path); err != nil {
			return fmt.Errorf("error at delete old points of %s: %w", file.Path, err)
		}

		parseError := ""
		points := []rawPoint{}
		// Format が空のファイルは「位置情報ではない」と分かったもの。
		// 点は無いが file_cache には残して、次の走査で先頭を読み直さないようにする。
		if isSupportedFormat(file.Format) {
			parsed, err := parseByFormat(file.Format, entry)
			if err != nil {
				parseError = err.Error()
			} else {
				points = parsed
			}
		}

		// フィルタは読み出し時に効かせるので、ここでは全部入れる。
		// こうしておくと設定を変えても再パースが要らない。
		for _, point := range points {
			if _, err := tx.Exec(`
INSERT INTO gps_point_cache(
  related_time_unix_ms, lat_e7, lng_e7, accuracy_mm, source, device_id, source_path, format)
VALUES(?,?,?,?,?,?,?,?)`,
				point.UnixMilli, point.LatE7, point.LngE7, point.AccuracyMm,
				point.Source, point.DeviceID, file.Path, file.Format); err != nil {
				return fmt.Errorf("error at insert point of %s: %w", file.Path, err)
			}
		}

		if _, err := tx.Exec(`
INSERT OR REPLACE INTO file_cache(path, mtime_unix, size, crc32, format, point_count, scan_error)
VALUES(?,?,?,?,?,?,?)`,
			file.Path, file.MtimeUnix, file.Size, file.CRC32, file.Format, len(points), parseError); err != nil {
			return fmt.Errorf("error at update file_cache of %s: %w", file.Path, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error at commit refresh tx: %w", err)
	}
	c.setMeta("last_scan_unix", strconv.FormatInt(time.Now().Unix(), 10))
	return scanErr
}

// storeSourceInfo は走査で見つかった書き出しと問題を設定画面へ渡すために書き残す。
//
// 設定画面のハンドラでZIPを開き直すわけにはいかない（数十ミリ秒で返す必要がある）ので、
// バックグラウンドで走査したときの結果をここに置いておく。
func (c *cache) storeSourceInfo(exports []sdk.ExportInfo, problems []sdk.SourceProblem) {
	exportRows := make([]exportRow, 0, len(exports))
	for _, export := range exports {
		exportRows = append(exportRows, exportRow{
			Dir:          export.Dir,
			ArchiveCount: len(export.ArchivePaths),
			NewestUnix:   export.NewestMtimeUnix,
			EntryCount:   export.EntryCount,
		})
	}
	if encoded, err := json.Marshal(exportRows); err == nil {
		c.setMeta("exports", string(encoded))
	}

	problemRows := make([]sourceProblemRow, 0, len(problems))
	for _, problem := range problems {
		problemRows = append(problemRows, sourceProblemRow{
			Kind:    string(problem.Kind),
			Path:    problem.Path,
			Message: problem.Message,
		})
	}
	if encoded, err := json.Marshal(problemRows); err == nil {
		c.setMeta("source_problems", string(encoded))
	}
}

// loadSourceInfo は storeSourceInfo が書いたものを読み戻す。
func (c *cache) loadSourceInfo() ([]exportRow, []sourceProblemRow) {
	exports := []exportRow{}
	if value := c.meta("exports"); value != "" {
		if err := json.Unmarshal([]byte(value), &exports); err != nil {
			exports = []exportRow{}
		}
	}
	problems := []sourceProblemRow{}
	if value := c.meta("source_problems"); value != "" {
		if err := json.Unmarshal([]byte(value), &problems); err != nil {
			problems = []sourceProblemRow{}
		}
	}
	return exports, problems
}

func (c *cache) loadFileCache() (map[string]scannedFile, error) {
	known := map[string]scannedFile{}
	rows, err := c.db.Query(`SELECT path, mtime_unix, size, crc32, format FROM file_cache`)
	if err != nil {
		return nil, fmt.Errorf("error at load file_cache: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var file scannedFile
		if err := rows.Scan(&file.Path, &file.MtimeUnix, &file.Size, &file.CRC32, &file.Format); err != nil {
			return nil, fmt.Errorf("error at scan file_cache: %w", err)
		}
		known[file.Path] = file
	}
	return known, rows.Err()
}

// GetGPSLogs は期間とページの指定に従って点を返す。
//
// フィルタと重複除去はSQLで行う。
// 重複除去がファイル横断なので書き込み時にはできない
// （ファイル単位の差分更新が、他のファイルにもある点を消してしまう）。
// 実データでは、ワークアウトのトラックが Fitbit App と Pixel Watch 2 の
// 2重に書き出されており、12,748行が6,483点になる。
func (c *cache) GetGPSLogs(pluginDir string, config pluginConfig, q sdk.GPSLogQuery) (sdk.GPSLogPage, error) {
	// 走査はバックグラウンドに投げて、ここでは待たない。
	//
	// Takeout 全体の初回走査は実測1分を超える（数千ファイルの先頭を読むため）。
	// gkill のハンドラは1呼び出し30秒・死活確認は5秒で打ち切られ、
	// 超えるとプロセスごと殺されるので、同期でやると必ず殺される。
	c.kickRefresh(pluginDir, config)

	// 走査を待たない。取り込めたぶんだけを返す
	if err := c.openDB(pluginDir); err != nil {
		return sdk.GPSLogPage{GPSLogs: []sdk.GPSLog{}}, nil
	}
	if c.db == nil {
		return sdk.GPSLogPage{GPSLogs: []sdk.GPSLog{}}, nil
	}

	conditions := []string{}
	args := []any{}
	if q.StartTime != nil {
		conditions = append(conditions, "related_time_unix_ms >= ?")
		args = append(args, q.StartTime.UnixMilli())
	}
	if q.EndTime != nil {
		conditions = append(conditions, "related_time_unix_ms <= ?")
		args = append(args, q.EndTime.UnixMilli())
	}
	if config.AccuracyMaxMeters > 0 {
		// 精度が分からない点(-1)は残す。測れないものをフィルタで消さない。
		conditions = append(conditions, "(accuracy_mm < 0 OR accuracy_mm <= ?)")
		args = append(args, int64(config.AccuracyMaxMeters)*1000)
	}
	if !config.IncludeFitbitGPS {
		conditions = append(conditions, "format != ?")
		args = append(args, formatFitbitGPSCSV)
	}
	if !config.VisitPoints {
		conditions = append(conditions, "source NOT IN (?, ?)")
		args = append(args, sourceVisit, sourceActivity)
	}
	if len(config.Sources) != 0 {
		placeholders := make([]string, 0, len(config.Sources))
		for _, source := range config.Sources {
			placeholders = append(placeholders, "?")
			args = append(args, strings.ToUpper(source))
		}
		conditions = append(conditions, "UPPER(source) IN ("+strings.Join(placeholders, ",")+")")
	}

	query := `SELECT DISTINCT related_time_unix_ms, lat_e7, lng_e7 FROM gps_point_cache`
	if len(conditions) != 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	// 並び順に緯度経度も混ぜるのは、ページングで行を飛ばしたり重ねたりしないため。
	query += " ORDER BY related_time_unix_ms ASC, lat_e7 ASC, lng_e7 ASC"

	limit := q.Limit
	if limit <= 0 {
		limit = defaultMaxPoints
	}
	if config.MaxPoints > 0 && limit > config.MaxPoints {
		limit = config.MaxPoints
	}
	// 続きがあるかを知るために1件多く取る
	query += " LIMIT ? OFFSET ?"
	args = append(args, limit+1, q.Offset)

	rows, err := c.db.Query(query, args...)
	if err != nil {
		return sdk.GPSLogPage{}, fmt.Errorf("error at query gps points: %w", err)
	}
	defer func() { _ = rows.Close() }()

	gpsLogs := []sdk.GPSLog{}
	hasMore := false
	for rows.Next() {
		if len(gpsLogs) >= limit {
			hasMore = true
			break
		}
		var unixMilli int64
		var latE7, lngE7 int32
		if err := rows.Scan(&unixMilli, &latE7, &lngE7); err != nil {
			return sdk.GPSLogPage{}, fmt.Errorf("error at scan gps point: %w", err)
		}
		gpsLogs = append(gpsLogs, sdk.GPSLog{
			RelatedTime: time.UnixMilli(unixMilli).UTC(),
			Latitude:    e7ToDegree(latE7),
			Longitude:   e7ToDegree(lngE7),
		})
	}
	return sdk.GPSLogPage{GPSLogs: gpsLogs, HasMore: hasMore}, rows.Err()
}

// Stats は設定画面に出す統計を返す。
func (c *cache) Stats(pluginDir string, config pluginConfig) cacheStats {
	// 走査を待たない。進捗を出すのが仕事なので、むしろ走査中こそ返る必要がある

	stats := cacheStats{
		FileCountByFormat: map[string]int{},
		PointsBySource:    map[string]int{},
	}
	if err := c.openDB(pluginDir); err != nil {
		stats.ScanErrors = append(stats.ScanErrors, err.Error())
		return stats
	}
	stats.Exports, stats.Problems = c.loadSourceInfo()

	// Format が空の行は「位置情報ではない」と判定済みのファイル。
	// 判定を覚えておくためだけに入っているので、統計には出さない。
	rows, err := c.db.Query(`SELECT path, format, scan_error FROM file_cache WHERE format != ''`)
	if err == nil {
		for rows.Next() {
			var path, formatID, scanError string
			if err := rows.Scan(&path, &formatID, &scanError); err != nil {
				continue
			}
			stats.FileCountByFormat[formatID]++
			if !isSupportedFormat(formatID) {
				stats.UnsupportedFiles = append(stats.UnsupportedFiles, path)
			}
			if scanError != "" {
				stats.ScanErrors = append(stats.ScanErrors, path+": "+scanError)
			}
		}
		_ = rows.Close()
	}

	_ = c.db.QueryRow(`SELECT COUNT(*) FROM gps_point_cache`).Scan(&stats.TotalPoints)

	sourceRows, err := c.db.Query(`SELECT source, COUNT(*) FROM gps_point_cache GROUP BY source`)
	if err == nil {
		for sourceRows.Next() {
			var source string
			var count int
			if err := sourceRows.Scan(&source, &count); err == nil {
				stats.PointsBySource[source] = count
			}
		}
		_ = sourceRows.Close()
	}

	// フィルタ後・重複除去後の件数は、実際に返すのと同じ条件で数える
	page, err := c.countFilteredLocked(config)
	if err == nil {
		stats.FilteredPoints = page.filtered
		stats.UniquePoints = page.unique
	}

	_ = c.db.QueryRow(`SELECT MIN(related_time_unix_ms), MAX(related_time_unix_ms) FROM gps_point_cache`).
		Scan(&stats.OldestUnixMilli, &stats.NewestUnixMilli)
	stats.LastScanUnix, _ = strconv.ParseInt(c.meta("last_scan_unix"), 10, 64)
	return stats
}

type filteredCounts struct {
	filtered int
	unique   int
}

func (c *cache) countFilteredLocked(config pluginConfig) (filteredCounts, error) {
	conditions := []string{}
	args := []any{}
	if config.AccuracyMaxMeters > 0 {
		conditions = append(conditions, "(accuracy_mm < 0 OR accuracy_mm <= ?)")
		args = append(args, int64(config.AccuracyMaxMeters)*1000)
	}
	if !config.IncludeFitbitGPS {
		conditions = append(conditions, "format != ?")
		args = append(args, formatFitbitGPSCSV)
	}
	if !config.VisitPoints {
		conditions = append(conditions, "source NOT IN (?, ?)")
		args = append(args, sourceVisit, sourceActivity)
	}
	where := ""
	if len(conditions) != 0 {
		where = " WHERE " + strings.Join(conditions, " AND ")
	}

	counts := filteredCounts{}
	if err := c.db.QueryRow(`SELECT COUNT(*) FROM gps_point_cache`+where, args...).Scan(&counts.filtered); err != nil {
		return counts, err
	}
	err := c.db.QueryRow(`SELECT COUNT(*) FROM (SELECT DISTINCT related_time_unix_ms, lat_e7, lng_e7 FROM gps_point_cache`+where+`)`, args...).
		Scan(&counts.unique)
	return counts, err
}
