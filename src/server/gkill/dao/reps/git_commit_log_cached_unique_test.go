package reps

import (
	"context"
	sqllib "database/sql"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
)

// dupGitRep は同じコミット集合を返し続けるstub。並行UpdateCacheのTOCTOU検査用。
type dupGitRep struct {
	stubGitCommitLogRep
	commits []GitCommitLog
}

func (d *dupGitRep) FindKyous(ctx context.Context, query *find.FindQuery) (map[string][]Kyou, error) {
	kyous := map[string][]Kyou{}
	for _, commit := range d.commits {
		kyous[commit.ID] = []Kyou{{
			ID: commit.ID, RepName: commit.RepName, DataType: "git_commit_log",
			RelatedTime: commit.RelatedTime, CreateTime: commit.CreateTime, UpdateTime: commit.UpdateTime,
		}}
	}
	return kyous, nil
}

func (d *dupGitRep) FindGitCommitLogByIDs(ctx context.Context, ids []string) ([]GitCommitLog, error) {
	logs := []GitCommitLog{}
	for _, id := range ids {
		for _, commit := range d.commits {
			if commit.ID == id {
				logs = append(logs, commit)
			}
		}
	}
	return logs, nil
}

func (d *dupGitRep) LastUpdateCacheChanged() bool { return true }

func newTestGitCommit(id string) GitCommitLog {
	at := time.Date(2026, 8, 1, 12, 0, 0, 0, time.Local)
	return GitCommitLog{
		ID: id, RepName: "stub_git_rep", DataType: "git_commit_log",
		RelatedTime: at, CreateTime: at, UpdateTime: at,
		CreateApp: "test", CreateDevice: "test", CreateUser: "test",
		UpdateApp: "test", UpdateDevice: "test", UpdateUser: "test",
		CommitMessage: "commit " + id, Addition: 1, Deletion: 0,
	}
}

// 重複行入りの既存(legacy)キャッシュDBを開くと自己修復されること。
//
// **掃除→UNIQUE索引の順が生命線**: 索引を先に作ると重複入りDBで
// CREATE UNIQUE INDEX が失敗し、コンストラクタごと失敗して
// GetRepositories 全体が死に、そのユーザーはログイン不能になる。
// 経緯: 1年分の外部監査 C2「git 同一コミットが2レコード返る」。
func TestGitCachedRepDuplicateRowsHealedOnOpen(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "git_cache.db")
	dbName := "TESTUSER_GIT_COMMIT_LOG"

	// legacyスキーマ(UNIQUE無し)のDBを作り、同一IDを2行入れる
	legacyDB, err := sqllib.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open legacy db: %v", err)
	}
	createSQL := `CREATE TABLE "` + dbName + `" (
  IS_DELETED NOT NULL, ID NOT NULL, COMMIT_MESSAGE NOT NULL, ADDITION NOT NULL, DELETION NOT NULL,
  CREATE_APP NOT NULL, CREATE_USER NOT NULL, CREATE_DEVICE NOT NULL,
  UPDATE_APP NOT NULL, UPDATE_DEVICE NOT NULL, UPDATE_USER NOT NULL,
  REP_NAME NOT NULL, RELATED_TIME_UNIX NOT NULL, CREATE_TIME_UNIX NOT NULL, UPDATE_TIME_UNIX NOT NULL)`
	if _, err := legacyDB.ExecContext(ctx, createSQL); err != nil {
		t.Fatalf("create legacy table: %v", err)
	}
	now := time.Now().Unix()
	insertSQL := `INSERT INTO "` + dbName + `" VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	for range 2 {
		if _, err := legacyDB.ExecContext(ctx, insertSQL,
			false, "dup-commit", "message", 1, 0, "app", "user", "device", "app", "device", "user", "gkill", now, now, now); err != nil {
			t.Fatalf("insert dup row: %v", err)
		}
	}
	if _, err := legacyDB.ExecContext(ctx, insertSQL,
		false, "unique-commit", "message2", 2, 1, "app", "user", "device", "app", "device", "user", "gkill", now, now, now); err != nil {
		t.Fatalf("insert unique row: %v", err)
	}
	if err := legacyDB.Close(); err != nil {
		t.Fatalf("close legacy db: %v", err)
	}

	// 開くと自己修復される（コンストラクタが失敗しないことがまず重要）
	rep, err := NewGitRepCachedSQLite3ImplPersistent(ctx, &stubGitCommitLogRep{}, dbPath, dbName, false)
	if err != nil {
		t.Fatalf("重複入りlegacy DBを開けない（自己修復に失敗）: %v", err)
	}
	t.Cleanup(func() { rep.Close(ctx) })

	checkDB, err := sqllib.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open check db: %v", err)
	}
	t.Cleanup(func() { checkDB.Close() })
	var dupCount, totalCount int
	if err := checkDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM "`+dbName+`" WHERE ID = 'dup-commit'`).Scan(&dupCount); err != nil {
		t.Fatalf("count dup rows: %v", err)
	}
	if dupCount != 1 {
		t.Errorf("重複行は1行に畳まれるはず: got %d", dupCount)
	}
	if err := checkDB.QueryRowContext(ctx, `SELECT COUNT(*) FROM "`+dbName+`"`).Scan(&totalCount); err != nil {
		t.Fatalf("count total rows: %v", err)
	}
	if totalCount != 2 {
		t.Errorf("dup-commit(1行)+unique-commit(1行)=2行のはず: got %d", totalCount)
	}

	// UNIQUE索引が張られており、以後の重複INSERTは弾かれる
	if _, err := checkDB.ExecContext(ctx, insertSQL,
		false, "dup-commit", "again", 1, 0, "app", "user", "device", "app", "device", "user", "gkill", now, now, now); err == nil {
		t.Error("UNIQUE索引が無い（同一IDのINSERTが通ってしまった）")
	}
}

// 並行UpdateCacheのTOCTOU（diffは無ロック・書き込みだけロック）でも重複行が入らないこと。
// INSERT OR IGNORE + UNIQUE(ID) の対で守る。
//
// 永続コンストラクタは ownDB の ref-hash 比較（stubでは空==空）でリビルドを
// スキップするため、比較を持たない共有DB版コンストラクタで差分経路を直接踏む。
func TestGitCachedRepConcurrentUpdateCacheNoDuplicates(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "git_cache_concurrent.db")
	dbName := "TESTUSER_GIT_COMMIT_LOG"

	commits := []GitCommitLog{}
	for i := range 20 {
		commits = append(commits, newTestGitCommit(fmt.Sprintf("commit-%02d", i)))
	}
	stub := &dupGitRep{commits: commits}

	db, err := sqllib.Open("sqlite", dbPath+"?_txlock=immediate&_pragma=busy_timeout(6000)")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	rep, err := NewGitRepCachedSQLite3Impl(ctx, stub, db, &sync.RWMutex{}, dbName)
	if err != nil {
		t.Fatalf("open cached rep: %v", err)
	}
	_ = rep

	wg := &sync.WaitGroup{}
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := rep.UpdateCache(ctx); err != nil {
				t.Errorf("UpdateCache failed: %v", err)
			}
		}()
	}
	wg.Wait()

	checkDB, err := sqllib.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open check db: %v", err)
	}
	t.Cleanup(func() { checkDB.Close() })
	var total, distinct int
	if err := checkDB.QueryRowContext(ctx, `SELECT COUNT(ID), COUNT(DISTINCT ID) FROM "`+dbName+`"`).Scan(&total, &distinct); err != nil {
		t.Fatalf("count rows: %v", err)
	}
	if total != distinct {
		t.Errorf("並行UpdateCacheで重複行が入った: total=%d distinct=%d", total, distinct)
	}
	if distinct != len(commits) {
		t.Errorf("全コミットが入るはず: distinct=%d want %d", distinct, len(commits))
	}
}
