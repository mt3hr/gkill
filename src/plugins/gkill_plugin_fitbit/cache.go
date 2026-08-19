package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	sdk "github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
	_ "modernc.org/sqlite"
)

// cacheSchemaVersion はDDLの版。変えると下の表を作り直す。
const cacheSchemaVersion = "2"

// cache はSQLite3のキャッシュ。
//
// 差分更新の単位が難しい点に注意。1ファイルが多数の日にまたがり、
// 1日が多数のファイル（心拍は1日1ファイルなので現地1日がUTC2ファイルにまたがる）から来る。
// そこで sample_daily に「ファイル1つが1日に寄与する部分集計」を持ち、
// 変化したファイルはその source_path の行だけ差し替えて、
// 日次の値は source_path 方向に畳み直す。
type cache struct {
	// mu は db の遅延初期化だけを守る。**長い処理の間は握らない。**
	//
	// かつては構築も読み取りもこれ1本で直列化していたが、それだと
	// 初回構築(実データで数十秒)のあいだ find_kyous が全部詰まり、
	// gkillのデッドラインでプロセスが殺され続ける。
	// SQLiteはWALで開いてあり *sql.DB 自体も並行安全なので、
	// 構築中の読み取りは「そこまで取り込めたぶん」を返せばよい。
	mu sync.Mutex

	// buildMu は構築を1本に直列化する。読み取りはこれを待たない。
	buildMu sync.Mutex

	db *sql.DB
}

var globalCache = &cache{}

// openDB はキャッシュDBを開く。
//
// gkill本体の sqlite3impl.GetSQLiteDBConnection は使わない。
// あれは journal_mode(DELETE) を方針として固定しており、
// バックグラウンドの書き込み中に読み手が busy_timeout まで待たされる。
// 派生キャッシュなので、gkill内の git_commit_log キャッシュと同じくWALで開く。
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

// initSchema はスキーマを作る。
// schema_version / registry_version / build_timezone のどれかが変わったら作り直す。
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
DROP TABLE IF EXISTS sample_daily;
DROP TABLE IF EXISTS export;
DROP TABLE IF EXISTS dirty_day;
DROP TABLE IF EXISTS daily_metric;
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
  path         TEXT PRIMARY KEY,   -- "<ZIP>!/<エントリ>"
  mtime_unix   INTEGER NOT NULL,   -- 書き出しの新旧用。世代内では全エントリ同値
  size         INTEGER NOT NULL,
  crc32        INTEGER NOT NULL,   -- 中身の変化はこれで見る
  export_id    TEXT    NOT NULL,
  prefix       TEXT    NOT NULL,
  row_count    INTEGER NOT NULL,
  scanned_unix INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_file_cache_export ON file_cache(export_id);
-- 取り込み世代。実データでは1〜3行しかない。
CREATE TABLE IF NOT EXISTS export (
  export_id         TEXT PRIMARY KEY,
  export_dir        TEXT    NOT NULL,
  newest_mtime_unix INTEGER NOT NULL,
  -- 前回の走査での newest_mtime_unix。
  -- どの日を畳み直すかはこれとの差で決める。rank の変化では決めない
  -- （新しい世代が1つ増えると rank は全部ずれるが、勝敗は変わらないため）。
  prev_mtime_unix   INTEGER NOT NULL,
  rank              INTEGER NOT NULL,  -- 0 が最新
  present           INTEGER NOT NULL,
  archive_count     INTEGER NOT NULL
) WITHOUT ROWID;
CREATE TABLE IF NOT EXISTS sample_daily (
  metric_key     TEXT    NOT NULL,
  date_local     TEXT    NOT NULL,
  export_id      TEXT    NOT NULL,
  source_path    TEXT    NOT NULL,
  sum_value      REAL    NOT NULL,
  count_value    INTEGER NOT NULL,
  min_value      REAL    NOT NULL,
  max_value      REAL    NOT NULL,
  last_value     REAL    NOT NULL,
  last_unix      INTEGER NOT NULL,
  src_mtime_unix INTEGER NOT NULL,
  devices        TEXT    NOT NULL,
  hour_sums      TEXT    NOT NULL,
  hour_counts    TEXT    NOT NULL,
  -- export_id を source_path の前に置く。1日ぶんの行が世代ごとに固まるので、
  -- 畳み直しが範囲走査だけで済む。一意性には寄与しない（source_pathが世代を決めるため）。
  PRIMARY KEY (metric_key, date_local, export_id, source_path)
) WITHOUT ROWID;
CREATE INDEX IF NOT EXISTS idx_sample_daily_src    ON sample_daily(source_path);
CREATE INDEX IF NOT EXISTS idx_sample_daily_export ON sample_daily(export_id);
CREATE TABLE IF NOT EXISTS dirty_day (
  metric_key TEXT NOT NULL,
  date_local TEXT NOT NULL,
  PRIMARY KEY (metric_key, date_local)
) WITHOUT ROWID;
CREATE TABLE IF NOT EXISTS daily_metric (
  kyou_id      TEXT PRIMARY KEY,
  metric_key   TEXT    NOT NULL,
  date_local   TEXT    NOT NULL,
  title        TEXT    NOT NULL,
  num_value    TEXT    NOT NULL,
  unit         TEXT    NOT NULL,
  sample_count INTEGER NOT NULL,
  min_value    REAL    NOT NULL,
  max_value    REAL    NOT NULL,
  devices      TEXT    NOT NULL,
  source_paths TEXT    NOT NULL,
  export_id    TEXT    NOT NULL,
  hour_sums    TEXT    NOT NULL,
  hour_counts  TEXT    NOT NULL,
  search_text  TEXT    NOT NULL,
  related_unix INTEGER NOT NULL,
  update_unix  INTEGER NOT NULL,
  UNIQUE(metric_key, date_local)
);
CREATE INDEX IF NOT EXISTS idx_daily_metric_related ON daily_metric(related_unix DESC);
`)
	if err != nil {
		return fmt.Errorf("error at create cache tables: %w", err)
	}
	return nil
}

// meta は cache_meta の値を読む。
func (c *cache) meta(key string) string {
	if c.db == nil {
		return ""
	}
	value := ""
	_ = c.db.QueryRow(`SELECT value FROM cache_meta WHERE key = ?`, key).Scan(&value)
	return value
}

// setMeta は cache_meta の値を書く。
func (c *cache) setMeta(key string, value string) {
	if c.db == nil {
		return
	}
	_, _ = c.db.Exec(`INSERT OR REPLACE INTO cache_meta(key,value) VALUES(?,?)`, key, value)
}

// resetIfGenerationChanged はレジストリ世代かタイムゾーンが変わっていたら
// 集計結果を捨てて全部読み直させる。値の意味そのものが変わるため。
func (c *cache) resetIfGenerationChanged(timezone string) error {
	changed := c.meta("registry_version") != registryVersion || c.meta("build_timezone") != timezone
	if !changed {
		return nil
	}
	if _, err := c.db.Exec(`DELETE FROM sample_daily; DELETE FROM dirty_day; DELETE FROM daily_metric; DELETE FROM file_cache;`); err != nil {
		return fmt.Errorf("error at reset cache for generation change: %w", err)
	}
	c.setMeta("registry_version", registryVersion)
	c.setMeta("build_timezone", timezone)
	return nil
}

// loadFileCache は前回のスキャン結果を読む。
func (c *cache) loadFileCache() (map[string]scannedFile, error) {
	known := map[string]scannedFile{}
	rows, err := c.db.Query(`SELECT path, mtime_unix, size, crc32, export_id, prefix FROM file_cache`)
	if err != nil {
		return nil, fmt.Errorf("error at load file_cache: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var file scannedFile
		if err := rows.Scan(&file.Path, &file.MtimeUnix, &file.Size, &file.CRC32, &file.ExportID, &file.Prefix); err != nil {
			return nil, fmt.Errorf("error at scan file_cache: %w", err)
		}
		known[file.Path] = file
	}
	return known, rows.Err()
}

// ingestFileIntoCache は1ファイルぶんの部分集計を差し替える。
//
// そのファイルが前回寄与していた日も再計算対象に積んでから消すのが要点。
// これをしないと、ファイルから消えたサンプルの寄与が残り続ける。
func (c *cache) ingestFileIntoCache(tx *sql.Tx, file scannedFile, partials []partialDaily) error {
	if _, err := tx.Exec(`
INSERT OR IGNORE INTO dirty_day(metric_key, date_local)
  SELECT metric_key, date_local FROM sample_daily WHERE source_path = ?`, file.Path); err != nil {
		return fmt.Errorf("error at mark dirty days of %s: %w", file.Path, err)
	}
	if _, err := tx.Exec(`DELETE FROM sample_daily WHERE source_path = ?`, file.Path); err != nil {
		return fmt.Errorf("error at delete old partials of %s: %w", file.Path, err)
	}

	rowCount := int64(0)
	for _, partial := range partials {
		rowCount += partial.CountValue
		devices := make([]string, 0, len(partial.Devices))
		for device := range partial.Devices {
			devices = append(devices, device)
		}
		sort.Strings(devices)

		if _, err := tx.Exec(`
INSERT OR REPLACE INTO sample_daily(
  metric_key, date_local, export_id, source_path, sum_value, count_value, min_value, max_value,
  last_value, last_unix, src_mtime_unix, devices, hour_sums, hour_counts)
VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			partial.MetricKey, partial.DateLocal, file.ExportID, file.Path,
			partial.SumValue, partial.CountValue, partial.MinValue, partial.MaxValue,
			partial.LastValue, partial.LastUnix, file.MtimeUnix,
			strings.Join(devices, "\n"),
			formatFloatVector(partial.HourSums[:]),
			formatIntVector(partial.HourCounts[:]),
		); err != nil {
			return fmt.Errorf("error at insert partial of %s: %w", file.Path, err)
		}
		if _, err := tx.Exec(`INSERT OR IGNORE INTO dirty_day(metric_key, date_local) VALUES(?,?)`,
			partial.MetricKey, partial.DateLocal); err != nil {
			return fmt.Errorf("error at mark dirty day: %w", err)
		}
	}

	if _, err := tx.Exec(`
INSERT OR REPLACE INTO file_cache(path, mtime_unix, size, crc32, export_id, prefix, row_count, scanned_unix)
VALUES(?,?,?,?,?,?,?,?)`,
		file.Path, file.MtimeUnix, file.Size, file.CRC32, file.ExportID, file.Prefix, rowCount, time.Now().Unix()); err != nil {
		return fmt.Errorf("error at update file_cache of %s: %w", file.Path, err)
	}
	return nil
}

// syncExports は走査で見つかった取り込み世代を export 表に反映する。
//
// exports は新しい順（SDKが NewestMtimeUnix の降順で返す）。その並びをそのまま rank にする。
//
// 勝敗が変わりうる日を dirty_day に積むのがもう1つの役目。
// 判定は rank の変化ではなく newest_mtime_unix の変化で行う。
// 世代が1つ増えると rank は全部ずれるが、相対順序は保たれるので勝敗は変わらないため、
// rank を見ると毎回全件を畳み直すことになる。
//
// 中身が変わった世代のエントリは (CRC32, Size) の差分で取り込み直されるので、
// そちら経由でも dirty は積まれる。ここが効くのは
// 「中身は同じだが ZIP を作り直して更新時刻だけ新しくなった」場合。
func (c *cache) syncExports(exports []sdk.ExportInfo) error {
	tx, err := c.db.Begin()
	if err != nil {
		return fmt.Errorf("error at begin export tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec(`UPDATE export SET present = 0`); err != nil {
		return fmt.Errorf("error at reset export presence: %w", err)
	}

	changed := []string{}
	for rank, export := range exports {
		previous := int64(-1)
		_ = tx.QueryRow(`SELECT prev_mtime_unix FROM export WHERE export_id = ?`, export.ExportID).Scan(&previous)
		if previous != export.NewestMtimeUnix {
			changed = append(changed, export.ExportID)
		}
		if _, err := tx.Exec(`
INSERT OR REPLACE INTO export(
  export_id, export_dir, newest_mtime_unix, prev_mtime_unix, rank, present, archive_count)
VALUES(?,?,?,?,?,1,?)`,
			export.ExportID, export.Dir, export.NewestMtimeUnix, export.NewestMtimeUnix,
			rank, len(export.ArchivePaths)); err != nil {
			return fmt.Errorf("error at upsert export %s: %w", export.ExportID, err)
		}
	}

	// 消えた世代。寄与していた日を畳み直させてから行を落とす。
	// エントリ単位の removeFileFromCache でも同じ日が積まれるが、
	// 取り込み元が丸ごと見えなくなった場合に取りこぼさないようここでも積む。
	if _, err := tx.Exec(`
INSERT OR IGNORE INTO dirty_day(metric_key, date_local)
  SELECT DISTINCT s.metric_key, s.date_local FROM sample_daily s
    JOIN export e ON e.export_id = s.export_id
   WHERE e.present = 0`); err != nil {
		return fmt.Errorf("error at mark dirty days of gone exports: %w", err)
	}
	if _, err := tx.Exec(`DELETE FROM export WHERE present = 0`); err != nil {
		return fmt.Errorf("error at delete gone exports: %w", err)
	}

	for _, exportID := range changed {
		if _, err := tx.Exec(`
INSERT OR IGNORE INTO dirty_day(metric_key, date_local)
  SELECT DISTINCT metric_key, date_local FROM sample_daily WHERE export_id = ?`, exportID); err != nil {
			return fmt.Errorf("error at mark dirty days of export %s: %w", exportID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error at commit export tx: %w", err)
	}
	return nil
}

// storeSourceProblems は走査で見つかった問題を設定画面へ渡すために書き残す。
//
// 設定画面のハンドラでZIPを開き直すわけにはいかない（数十ミリ秒で返す必要がある）ので、
// バックグラウンドで走査したときの結果をここに置いておく。
func (c *cache) storeSourceProblems(problems []sdk.SourceProblem) {
	rows := make([]sourceProblemRow, 0, len(problems))
	for _, problem := range problems {
		rows = append(rows, sourceProblemRow{
			Kind:    string(problem.Kind),
			Path:    problem.Path,
			Message: problem.Message,
		})
	}
	encoded, err := json.Marshal(rows)
	if err != nil {
		return
	}
	c.setMeta("source_problems", string(encoded))
}

// loadSourceProblems は storeSourceProblems が書いたものを読み戻す。
func (c *cache) loadSourceProblems() []sourceProblemRow {
	rows := []sourceProblemRow{}
	value := c.meta("source_problems")
	if value == "" {
		return rows
	}
	if err := json.Unmarshal([]byte(value), &rows); err != nil {
		return []sourceProblemRow{}
	}
	return rows
}

// loadExports は取り込み世代を採用順に読む。
func (c *cache) loadExports() []exportRow {
	exports := []exportRow{}
	rows, err := c.db.Query(`
SELECT e.export_id, e.export_dir, e.archive_count, e.newest_mtime_unix, e.rank,
       (SELECT COUNT(DISTINCT s.date_local) FROM sample_daily s WHERE s.export_id = e.export_id)
FROM export e ORDER BY e.rank`)
	if err != nil {
		return exports
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var export exportRow
		if err := rows.Scan(&export.ExportID, &export.Dir, &export.ArchiveCount,
			&export.NewestUnix, &export.Rank, &export.DayCount); err != nil {
			return exports
		}
		exports = append(exports, export)
	}
	return exports
}

// removeFileFromCache は消えたファイルの寄与を落とす。
func (c *cache) removeFileFromCache(tx *sql.Tx, path string) error {
	if _, err := tx.Exec(`
INSERT OR IGNORE INTO dirty_day(metric_key, date_local)
  SELECT metric_key, date_local FROM sample_daily WHERE source_path = ?`, path); err != nil {
		return fmt.Errorf("error at mark dirty days of removed %s: %w", path, err)
	}
	if _, err := tx.Exec(`DELETE FROM sample_daily WHERE source_path = ?`, path); err != nil {
		return fmt.Errorf("error at delete partials of removed %s: %w", path, err)
	}
	if _, err := tx.Exec(`DELETE FROM file_cache WHERE path = ?`, path); err != nil {
		return fmt.Errorf("error at delete file_cache of removed %s: %w", path, err)
	}
	return nil
}

// foldBatchCTE は畳み直す日を切り出す共通部分。
//
// 処理する日の確定と集計とで**同じ式**を使うのが要点。
// 別々の LIMIT 付きクエリで取ると、両者が同じ集合を選ぶ保証がSQLには無く、
// 食い違ったぶんが「寄与ファイルが全部消えた日」と誤判定されて集計結果ごと消える。
// dirty_day は (metric_key, date_local) が主キーなので、この ORDER BY は全順序になる。
const foldBatchCTE = `
WITH batch AS (
  SELECT metric_key, date_local FROM dirty_day ORDER BY metric_key, date_local LIMIT ?
)`

// foldSelectSQL は畳み直す日ぶんの集計を取る。
//
// 世代（export）をまたいで足さないのが肝。同じ (指標, 日) に複数の世代の寄与があるときは
// rank が最小＝いちばん新しい世代の行だけを使う。
// 分割された ZIP は同じ世代なので合算され、別の日に書き出したぶんは合算されない。
const foldSelectSQL = foldBatchCTE + `,
winner AS (
  SELECT s.metric_key, s.date_local, MIN(e.rank) AS rank
    FROM sample_daily s
    JOIN batch  b ON b.metric_key = s.metric_key AND b.date_local = s.date_local
    JOIN export e ON e.export_id  = s.export_id
   GROUP BY s.metric_key, s.date_local
)
SELECT s.metric_key, s.date_local,
       SUM(s.sum_value), SUM(s.count_value),
       MIN(s.min_value), MAX(s.max_value),
       MAX(s.src_mtime_unix),
       -- 採用した世代。JOIN で1つに絞ってあるので、この集約は「その1つ」を返す
       MIN(s.export_id),
       GROUP_CONCAT(s.devices,     char(10)),
       GROUP_CONCAT(s.hour_sums,   char(10)),
       GROUP_CONCAT(s.hour_counts, char(10)),
       GROUP_CONCAT(s.source_path, char(10)),
       (SELECT s2.last_value FROM sample_daily s2
          JOIN export e2 ON e2.export_id = s2.export_id
         WHERE s2.metric_key = s.metric_key AND s2.date_local = s.date_local
           AND e2.rank = w.rank
         ORDER BY s2.last_unix DESC LIMIT 1)
FROM sample_daily s
JOIN winner w ON w.metric_key = s.metric_key AND w.date_local = s.date_local
JOIN export e ON e.export_id  = s.export_id  AND e.rank      = w.rank
GROUP BY s.metric_key, s.date_local`

// foldDirtyDays は再計算対象の日を畳み直して daily_metric を更新する。
// 一度に処理する件数を limit で区切る（1トランザクションが長くならないように）。
// 返り値は処理した件数。
func (c *cache) foldDirtyDays(loc *time.Location, limit int) (int, error) {
	tx, err := c.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("error at begin fold tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// 処理する日を先に確定させる。集計は同じ式で切り出すので、
	// 集計に現れなかった日はほんとうに寄与が1件も残っていない日。
	type dirtyKey struct{ metricKey, dateLocal string }
	dirtyKeys := []dirtyKey{}
	dirtyRows, err := tx.Query(`SELECT metric_key, date_local FROM dirty_day ORDER BY metric_key, date_local LIMIT ?`, limit)
	if err != nil {
		return 0, fmt.Errorf("error at select dirty days: %w", err)
	}
	for dirtyRows.Next() {
		var key dirtyKey
		if err := dirtyRows.Scan(&key.metricKey, &key.dateLocal); err != nil {
			_ = dirtyRows.Close()
			return 0, fmt.Errorf("error at scan dirty day: %w", err)
		}
		dirtyKeys = append(dirtyKeys, key)
	}
	if err := dirtyRows.Err(); err != nil {
		_ = dirtyRows.Close()
		return 0, fmt.Errorf("error at iterate dirty days: %w", err)
	}
	_ = dirtyRows.Close()
	if len(dirtyKeys) == 0 {
		return 0, nil
	}

	rows, err := tx.Query(foldSelectSQL, limit)
	if err != nil {
		return 0, fmt.Errorf("error at fold dirty days: %w", err)
	}

	type folded struct {
		metricKey   string
		dateLocal   string
		sum         float64
		count       int64
		minValue    float64
		maxValue    float64
		maxMtime    int64
		exportID    string
		devices     string
		hourSums    string
		hourCounts  string
		sourcePaths string
		lastValue   float64
	}
	batch := []folded{}
	for rows.Next() {
		var f folded
		var lastValue sql.NullFloat64
		if err := rows.Scan(&f.metricKey, &f.dateLocal, &f.sum, &f.count,
			&f.minValue, &f.maxValue, &f.maxMtime, &f.exportID,
			&f.devices, &f.hourSums, &f.hourCounts, &f.sourcePaths, &lastValue); err != nil {
			_ = rows.Close()
			return 0, fmt.Errorf("error at scan folded day: %w", err)
		}
		f.lastValue = lastValue.Float64
		batch = append(batch, f)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return 0, fmt.Errorf("error at iterate folded days: %w", err)
	}
	_ = rows.Close()

	processed := map[string]struct{}{}
	for _, f := range batch {
		processed[f.metricKey+"\x00"+f.dateLocal] = struct{}{}
		def, exist := metricByKey[f.metricKey]
		if !exist {
			// レジストリから消えた指標。集計結果も落とす
			if _, err := tx.Exec(`DELETE FROM daily_metric WHERE metric_key = ? AND date_local = ?`, f.metricKey, f.dateLocal); err != nil {
				return 0, fmt.Errorf("error at delete unknown metric: %w", err)
			}
			continue
		}

		value := 0.0
		switch def.Agg {
		case aggSum:
			value = f.sum
		case aggCount:
			value = float64(f.count)
		case aggMean:
			if f.count > 0 {
				value = f.sum / float64(f.count)
			}
		case aggMin:
			value = f.minValue
		case aggMax:
			value = f.maxValue
		case aggLast:
			value = f.lastValue
		}
		value *= def.scaleOf()
		numValue := strconv.FormatFloat(value, 'f', def.Round, 64)

		devices := dedupeLines(f.devices)
		sourcePaths := dedupeLines(f.sourcePaths)
		hourSums := sumFloatVectors(f.hourSums)
		hourCounts := sumIntVectors(f.hourCounts)

		kyouID := kyouIDOf(def.Key, f.dateLocal)
		searchText := strings.Join([]string{
			def.Title, def.Key, def.Unit, numValue, strings.Join(devices, " "), f.dateLocal, repName,
		}, " ")

		if _, err := tx.Exec(`
INSERT OR REPLACE INTO daily_metric(
  kyou_id, metric_key, date_local, title, num_value, unit, sample_count,
  min_value, max_value, devices, source_paths, export_id, hour_sums, hour_counts,
  search_text, related_unix, update_unix)
VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			kyouID, def.Key, f.dateLocal, def.Title, numValue, def.Unit, f.count,
			f.minValue*def.scaleOf(), f.maxValue*def.scaleOf(),
			strings.Join(devices, "\n"), strings.Join(sourcePaths, "\n"), f.exportID,
			formatFloatVector(hourSums), formatIntVector(hourCounts),
			searchText, noonUnixOf(f.dateLocal, loc), f.maxMtime,
		); err != nil {
			return 0, fmt.Errorf("error at upsert daily_metric: %w", err)
		}
	}

	// 部分集計が1つも残っていない（寄与ファイルが全部消えた）日は集計結果ごと落とす。
	// batch に現れなかった dirtyKeys がそれにあたる。
	for _, key := range dirtyKeys {
		if _, done := processed[key.metricKey+"\x00"+key.dateLocal]; !done {
			if _, err := tx.Exec(`DELETE FROM daily_metric WHERE metric_key = ? AND date_local = ?`, key.metricKey, key.dateLocal); err != nil {
				return 0, fmt.Errorf("error at delete empty day: %w", err)
			}
		}
		if _, err := tx.Exec(`DELETE FROM dirty_day WHERE metric_key = ? AND date_local = ?`, key.metricKey, key.dateLocal); err != nil {
			return 0, fmt.Errorf("error at clear dirty day: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("error at commit fold tx: %w", err)
	}
	return len(dirtyKeys), nil
}

// kyouIDOf は (指標キー, 現地日付) から決定的なKyou IDを作る。
// 値やタイムゾーンには依存させない。作り直しても同じIDになり、
// タイムゾーンを変えても新しいKyouが増えるのではなく既存が更新される。
func kyouIDOf(metricKey string, dateLocal string) string {
	return uuidV5(fitbitNamespace, appName+"|"+metricKey+"|"+dateLocal)
}

// formatFloatVector は小数の並びをカンマ区切りにする。
func formatFloatVector(values []float64) string {
	parts := make([]string, len(values))
	for i, value := range values {
		parts[i] = strconv.FormatFloat(value, 'f', -1, 64)
	}
	return strings.Join(parts, ",")
}

// formatIntVector は整数の並びをカンマ区切りにする。
func formatIntVector(values []int64) string {
	parts := make([]string, len(values))
	for i, value := range values {
		parts[i] = strconv.FormatInt(value, 10)
	}
	return strings.Join(parts, ",")
}

// parseFloatVector はカンマ区切りを小数の並びに戻す。
func parseFloatVector(value string) []float64 {
	parts := strings.Split(value, ",")
	values := make([]float64, len(parts))
	for i, part := range parts {
		parsed, err := strconv.ParseFloat(part, 64)
		if err == nil && !math.IsNaN(parsed) && !math.IsInf(parsed, 0) {
			values[i] = parsed
		}
	}
	return values
}

// parseIntVector はカンマ区切りを整数の並びに戻す。
func parseIntVector(value string) []int64 {
	parts := strings.Split(value, ",")
	values := make([]int64, len(parts))
	for i, part := range parts {
		parsed, err := strconv.ParseInt(part, 10, 64)
		if err == nil {
			values[i] = parsed
		}
	}
	return values
}

// sumFloatVectors は改行で連結された複数のベクトルを要素ごとに足す。
// SQLite の GROUP_CONCAT に DISTINCT と区切り文字を同時に指定できないので、
// 改行で繋いだものをここでほどく。
func sumFloatVectors(joined string) []float64 {
	total := make([]float64, 24)
	for line := range strings.SplitSeq(joined, "\n") {
		if line == "" {
			continue
		}
		for i, value := range parseFloatVector(line) {
			if i < len(total) {
				total[i] += value
			}
		}
	}
	return total
}

// sumIntVectors は改行で連結された複数のベクトルを要素ごとに足す。
func sumIntVectors(joined string) []int64 {
	total := make([]int64, 24)
	for line := range strings.SplitSeq(joined, "\n") {
		if line == "" {
			continue
		}
		for i, value := range parseIntVector(line) {
			if i < len(total) {
				total[i] += value
			}
		}
	}
	return total
}

// dedupeLines は改行区切りの値を重複を除いて並べ替える。
func dedupeLines(joined string) []string {
	seen := map[string]struct{}{}
	values := []string{}
	for line := range strings.SplitSeq(joined, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if _, duplicated := seen[line]; duplicated {
			continue
		}
		seen[line] = struct{}{}
		values = append(values, line)
	}
	sort.Strings(values)
	return values
}
