package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
	_ "modernc.org/sqlite"
)

// cacheSchemaVersion はDDLの世代。上げると次回の起動で作り直す。
const cacheSchemaVersion = "1"

// cache はキャッシュDBを持つ。プロセス内に1つ。
//
// ロックを分けているのが要点（codex / fitbit と同じ）。
//   - mu は db の遅延初期化だけを守る
//   - buildMu は構築どうしだけを直列化する
//   - 読み取り(query.go)はどちらも取らない
//
// 兼用すると初回構築のあいだ find_kyous が全部詰まり、
// gkill のデッドライン(IsAlive 5秒 / 呼び出し30秒)でプロセスが殺され続ける。
type cache struct {
	mu      sync.Mutex
	buildMu sync.Mutex
	db      *sql.DB
}

var globalCache = &cache{}

// openDB はキャッシュDBを開く。何度呼んでも1回しか開かない。
func (c *cache) openDB(pluginDir string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.db != nil {
		return nil
	}

	dbPath := sdk.CacheDBPath(pluginDir)

	// sqlite3impl.GetSQLiteDBConnection は使わない。
	// あれは journal_mode を DELETE に固定するので、バックグラウンドで書いている間
	// 読み手が busy_timeout まで待たされる。派生キャッシュは自前でWALを開く。
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

// conn は開いてあるDBを返す。openDB を通っていれば非nil。
func (c *cache) conn() *sql.DB {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.db
}

func initSchema(db *sql.DB) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS cache_meta (
  key   TEXT PRIMARY KEY,
  value TEXT NOT NULL
)`); err != nil {
		return fmt.Errorf("error at create cache_meta: %w", err)
	}

	version := ""
	_ = db.QueryRow(`SELECT value FROM cache_meta WHERE key = 'schema_version'`).Scan(&version)
	if version != cacheSchemaVersion {
		for _, statement := range []string{
			`DROP TABLE IF EXISTS repo`,
			`DROP TABLE IF EXISTS commit_log`,
			`DROP TABLE IF EXISTS commit_repo`,
			`DELETE FROM cache_meta`,
		} {
			if _, err := db.Exec(statement); err != nil {
				return fmt.Errorf("error at reset cache: %w", err)
			}
		}
	}

	for _, statement := range []string{
		// repo は「zip の中の1リポジトリ」。指紋が前回と同じなら読み直さない。
		`CREATE TABLE IF NOT EXISTS repo (
  archive_path TEXT    NOT NULL,
  git_dir      TEXT    NOT NULL,
  rep_name     TEXT    NOT NULL,
  fingerprint  TEXT    NOT NULL,
  commit_count INTEGER NOT NULL,
  scanned_unix INTEGER NOT NULL,
  note         TEXT    NOT NULL,
  PRIMARY KEY (archive_path, git_dir)
) WITHOUT ROWID`,

		// commit_log はコミット1件。同じハッシュは複数の zip に入っていても1行
		// （rep_name は最初に見たリポジトリのもの）。
		`CREATE TABLE IF NOT EXISTS commit_log (
  hash            TEXT    PRIMARY KEY,
  rep_name        TEXT    NOT NULL,
  committer_unix  INTEGER NOT NULL,
  tz_offset_sec   INTEGER NOT NULL,
  author_name     TEXT    NOT NULL,
  author_email    TEXT    NOT NULL,
  message         TEXT    NOT NULL,
  addition        INTEGER NOT NULL,
  deletion        INTEGER NOT NULL,
  file_stats_json TEXT    NOT NULL,
  stats_error     TEXT    NOT NULL
) WITHOUT ROWID`,
		`CREATE INDEX IF NOT EXISTS idx_commit_committer ON commit_log(committer_unix DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_commit_rep ON commit_log(rep_name)`,

		// commit_repo はコミットがどの zip のどのリポジトリに入っていたか。
		// zip を外したときに、他の zip にも無いコミットだけを消すための表。
		`CREATE TABLE IF NOT EXISTS commit_repo (
  hash         TEXT NOT NULL,
  archive_path TEXT NOT NULL,
  git_dir      TEXT NOT NULL,
  PRIMARY KEY (hash, archive_path, git_dir)
) WITHOUT ROWID`,
		`CREATE INDEX IF NOT EXISTS idx_commit_repo_source ON commit_repo(archive_path, git_dir)`,
	} {
		if _, err := db.Exec(statement); err != nil {
			return fmt.Errorf("error at create cache table: %w", err)
		}
	}

	if _, err := db.Exec(
		`INSERT OR REPLACE INTO cache_meta (key, value) VALUES ('schema_version', ?)`,
		cacheSchemaVersion,
	); err != nil {
		return fmt.Errorf("error at store schema_version: %w", err)
	}
	return nil
}

func (c *cache) setMeta(key, value string) {
	db := c.conn()
	if db == nil {
		return
	}
	_, _ = db.Exec(`INSERT OR REPLACE INTO cache_meta (key, value) VALUES (?, ?)`, key, value)
}

func (c *cache) getMeta(key string) string {
	db := c.conn()
	if db == nil {
		return ""
	}
	value := ""
	_ = db.QueryRow(`SELECT value FROM cache_meta WHERE key = ?`, key).Scan(&value)
	return value
}

// sourceProblemRow は設定画面に出す走査の問題。cache_meta に JSON で置く。
type sourceProblemRow struct {
	Kind    string `json:"kind"`
	Path    string `json:"path"`
	Message string `json:"message"`
}

func (c *cache) storeSourceProblems(problems []sdk.SourceProblem) {
	rows := make([]sourceProblemRow, 0, len(problems))
	for _, problem := range problems {
		rows = append(rows, sourceProblemRow{Kind: string(problem.Kind), Path: problem.Path, Message: problem.Message})
	}
	encoded, err := json.Marshal(rows)
	if err != nil {
		return
	}
	c.setMeta("source_problems", string(encoded))
}

func (c *cache) loadSourceProblems() []sourceProblemRow {
	rows := []sourceProblemRow{}
	value := c.getMeta("source_problems")
	if value == "" {
		return rows
	}
	if err := json.Unmarshal([]byte(value), &rows); err != nil {
		return []sourceProblemRow{}
	}
	return rows
}

// knownRepo は前回の走査で記録したリポジトリ。
type knownRepo struct {
	ArchivePath string
	GitDir      string
	Fingerprint string
}

// loadKnownRepos は repo 表を読む。キーは bundleKey。
func (c *cache) loadKnownRepos() (map[string]knownRepo, error) {
	db := c.conn()
	if db == nil {
		return nil, fmt.Errorf("cache db is not opened")
	}
	rows, err := db.Query(`SELECT archive_path, git_dir, fingerprint FROM repo`)
	if err != nil {
		return nil, fmt.Errorf("error at select repo: %w", err)
	}
	defer func() { _ = rows.Close() }()

	known := map[string]knownRepo{}
	for rows.Next() {
		var repo knownRepo
		if err := rows.Scan(&repo.ArchivePath, &repo.GitDir, &repo.Fingerprint); err != nil {
			return nil, fmt.Errorf("error at scan repo: %w", err)
		}
		known[bundleKey(repo.ArchivePath, repo.GitDir)] = repo
	}
	return known, rows.Err()
}

// build はキャッシュを最新にする。ビルダ以外から呼ばないこと。
//
// 手順:
//  1. zip を列挙し、(zip, .git の場所) ごとの束にする
//  2. 指定から外れた束のコミット参照を消し、どこにも無くなったコミットを消す
//  3. 指紋が変わった束だけ読み直す（アーカイブは基本不変なので2回目以降はここで終わる）
func (c *cache) build(ctx context.Context, pluginDir string, config pluginConfig) error {
	// 構築どうしだけを直列化する。読み取りは待たせない
	c.buildMu.Lock()
	defer c.buildMu.Unlock()

	if err := c.openDB(pluginDir); err != nil {
		return err
	}

	c.setMeta("build_state", "scanning")
	c.setMeta("build_error", "")

	source, scanErr := openSources(config.Patterns)
	defer func() { _ = source.Close() }()
	c.storeSourceProblems(relevantProblems(source.Problems()))

	bundles := groupRepoBundles(source.Entries())
	known, err := c.loadKnownRepos()
	if err != nil {
		return err
	}

	current := make(map[string]struct{}, len(bundles))
	changed := make([]repoBundle, 0, len(bundles))
	for _, bundle := range bundles {
		current[bundle.key()] = struct{}{}
		if previous, exist := known[bundle.key()]; exist && previous.Fingerprint == bundle.Fingerprint {
			continue
		}
		changed = append(changed, bundle)
	}
	removed := make([]knownRepo, 0)
	for key, repo := range known {
		if _, exist := current[key]; !exist {
			removed = append(removed, repo)
		}
	}

	c.setMeta("target_repo_count", strconv.Itoa(len(bundles)))
	c.setMeta("build_total_repos", strconv.Itoa(len(changed)))
	c.setMeta("build_done_repos", "0")

	if len(removed) != 0 {
		if err := c.removeRepos(removed); err != nil {
			return err
		}
	}

	if len(changed) != 0 {
		c.setMeta("build_state", "ingesting")
		for done, bundle := range changed {
			if err := ctx.Err(); err != nil {
				return err
			}
			if err := c.ingestBundle(ctx, bundle, config.MaxGitDirBytes); err != nil {
				return err
			}
			c.setMeta("build_done_repos", strconv.Itoa(done+1))
		}
		if err := c.deleteOrphanCommits(); err != nil {
			return err
		}
	}

	c.setMeta("build_state", "idle")
	c.setMeta("last_scan_unix", strconv.FormatInt(time.Now().Unix(), 10))
	return scanErr
}

// removeRepos は指定から外れたリポジトリの記録を消す。
// コミット本体は「他の zip にも入っていない」ものだけ消す（deleteOrphanCommits）。
func (c *cache) removeRepos(removed []knownRepo) error {
	db := c.conn()
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("error at begin remove tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	for _, repo := range removed {
		if _, err := tx.Exec(`DELETE FROM commit_repo WHERE archive_path = ? AND git_dir = ?`, repo.ArchivePath, repo.GitDir); err != nil {
			return fmt.Errorf("error at delete commit_repo: %w", err)
		}
		if _, err := tx.Exec(`DELETE FROM repo WHERE archive_path = ? AND git_dir = ?`, repo.ArchivePath, repo.GitDir); err != nil {
			return fmt.Errorf("error at delete repo: %w", err)
		}
	}
	if _, err := tx.Exec(`DELETE FROM commit_log WHERE hash NOT IN (SELECT hash FROM commit_repo)`); err != nil {
		return fmt.Errorf("error at delete orphan commits: %w", err)
	}
	return tx.Commit()
}

// deleteOrphanCommits は commit_repo から参照されなくなったコミットを消す。
// 読み直したリポジトリからコミットが消えていた（履歴を書き換えた zip で置き換えた）ときのため。
func (c *cache) deleteOrphanCommits() error {
	db := c.conn()
	if _, err := db.Exec(`DELETE FROM commit_log WHERE hash NOT IN (SELECT hash FROM commit_repo)`); err != nil {
		return fmt.Errorf("error at delete orphan commits: %w", err)
	}
	return nil
}

// ingestBundle は1リポジトリを読んでキャッシュへ入れる。1リポジトリ1トランザクション。
//
// 読めないリポジトリ（.git の上限超過・壊れた zip・go-git が開けない）は
// エラーにせず repo 表に理由を残して次へ進む。1つの壊れた zip で全部を止めない。
// 理由は設定画面に出るので、静かに0件になることはない。
func (c *cache) ingestBundle(ctx context.Context, bundle repoBundle, maxGitDirBytes int64) error {
	records, readErr := readRepoCommits(ctx, bundle, maxGitDirBytes)
	if readErr != nil {
		if errors.Is(readErr, context.Canceled) || errors.Is(readErr, context.DeadlineExceeded) {
			return readErr
		}
		sdk.LogWarn("%s: skip repository %s!/%s: %v", appName, bundle.ArchivePath, bundle.GitDir, readErr)
	}

	db := c.conn()
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("error at begin ingest tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// 読み直しなので、この束の古い参照は先に消す
	if _, err := tx.Exec(`DELETE FROM commit_repo WHERE archive_path = ? AND git_dir = ?`, bundle.ArchivePath, bundle.GitDir); err != nil {
		return fmt.Errorf("error at delete commit_repo: %w", err)
	}

	for _, record := range records {
		files, err := json.Marshal(record.Files)
		if err != nil {
			return fmt.Errorf("error at marshal file stats: %w", err)
		}
		// 同じハッシュは同じコミットなので、先に入っていれば触らない（rep_name は最初に見たもの）
		if _, err := tx.Exec(`INSERT INTO commit_log (
  hash, rep_name, committer_unix, tz_offset_sec, author_name, author_email, message,
  addition, deletion, file_stats_json, stats_error
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT(hash) DO NOTHING`,
			record.Hash, bundle.RepName, record.CommitterUnix, record.TZOffsetSec,
			record.AuthorName, record.AuthorEmail, record.Message,
			record.Addition, record.Deletion, string(files), record.StatsError); err != nil {
			return fmt.Errorf("error at insert commit %s: %w", record.Hash, err)
		}
		if _, err := tx.Exec(`INSERT OR IGNORE INTO commit_repo (hash, archive_path, git_dir) VALUES (?, ?, ?)`,
			record.Hash, bundle.ArchivePath, bundle.GitDir); err != nil {
			return fmt.Errorf("error at insert commit_repo %s: %w", record.Hash, err)
		}
	}

	note := ""
	if readErr != nil {
		note = readErr.Error()
	}
	if _, err := tx.Exec(`INSERT OR REPLACE INTO repo (archive_path, git_dir, rep_name, fingerprint, commit_count, scanned_unix, note)
VALUES (?, ?, ?, ?, ?, ?, ?)`,
		bundle.ArchivePath, bundle.GitDir, bundle.RepName, bundle.Fingerprint, len(records), time.Now().Unix(), note); err != nil {
		return fmt.Errorf("error at upsert repo: %w", err)
	}
	return tx.Commit()
}
