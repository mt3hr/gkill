package dao

// git_commit_logのrep定義はglob (`$HOME/Git/*`) で書かれるため、
// 展開先にgitリポジトリでないエントリが混ざりうる。
// それでGetRepositories全体が落ちないことのテスト。

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/mt3hr/gkill/src/server/gkill/dao/user_config"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_options"
)

// setupGitRepGlobTestOptions はgkill_optionsを一時ディレクトリへ向け、
// 後片付けをt.Cleanupに登録して一時ディレクトリを返す。
// 本物の$HOMEを見に行くと利用者のプラグインが起動してしまうので、
// GkillHomeDirまで含めて必ず差し替える。
func setupGitRepGlobTestOptions(t *testing.T) string {
	t.Helper()

	// SQLiteがファイルハンドルを掴んだままだとWindowsでt.TempDir()の
	// 自動削除が失敗するので、自前で作って後片付けはbest-effortにする
	tmpDir, err := os.MkdirTemp("", "gkill_dao_git_rep_glob_test_*")
	if err != nil {
		t.Fatalf("os.MkdirTemp failed: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(tmpDir) })

	origHome := gkill_options.GkillHomeDir
	origLib := gkill_options.LibDir
	origCache := gkill_options.CacheDir
	origLog := gkill_options.LogDir
	origConfig := gkill_options.ConfigDir
	origData := gkill_options.DataDirectoryDefault
	origCacheInMemory := gkill_options.IsCacheInMemory

	gkill_options.GkillHomeDir = tmpDir
	gkill_options.LibDir = filepath.Join(tmpDir, "lib", "base_directory")
	gkill_options.CacheDir = filepath.Join(tmpDir, "caches")
	gkill_options.LogDir = filepath.Join(tmpDir, "logs")
	gkill_options.ConfigDir = filepath.Join(tmpDir, "configs")
	gkill_options.DataDirectoryDefault = filepath.Join(tmpDir, "datas")
	// キャッシュrepで差し替えられるとGitCommitLogRepsが常に1個になり、
	// スキップできているかを数で見られなくなるので切っておく
	gkill_options.IsCacheInMemory = false

	t.Cleanup(func() {
		gkill_options.GkillHomeDir = origHome
		gkill_options.LibDir = origLib
		gkill_options.CacheDir = origCache
		gkill_options.LogDir = origLog
		gkill_options.ConfigDir = origConfig
		gkill_options.DataDirectoryDefault = origData
		gkill_options.IsCacheInMemory = origCacheInMemory
	})

	for _, dir := range []string{"configs", "datas", "caches", "logs", "plugins"} {
		if err := os.MkdirAll(filepath.Join(tmpDir, dir), 0o755); err != nil {
			t.Fatalf("MkdirAll %s failed: %v", dir, err)
		}
	}

	return tmpDir
}

// initGitRepositoryForTest は1コミットだけ入ったgitリポジトリをdirに作る。
func initGitRepositoryForTest(t *testing.T, dir string) {
	t.Helper()

	gitRep, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("git.PlainInit failed: %v", err)
	}
	worktree, err := gitRep.Worktree()
	if err != nil {
		t.Fatalf("Worktree failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "first.txt"), []byte("first\n"), 0o600); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	if _, err := worktree.Add("first.txt"); err != nil {
		t.Fatalf("worktree.Add failed: %v", err)
	}
	signature := &object.Signature{
		Name:  "test_user",
		Email: "test_user@example.com",
		When:  time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC),
	}
	if _, err := worktree.Commit("first commit", &git.CommitOptions{Author: signature, Committer: signature}); err != nil {
		t.Fatalf("worktree.Commit failed: %v", err)
	}
}

// writeRepositoriesForTest は書き込み先repの一式を作って返す。
// REPOSITORY への追加は「デバイスごとに各種別の書き込み先repがちょうど1つ」を
// 検査するので、git_commit_logのrepだけを足すことはできない。
func writeRepositoriesForTest(t *testing.T, userID string, device string, dataDir string) []*user_config.Repository {
	t.Helper()

	repositories := []*user_config.Repository{}

	newRepository := func(repType string, file string) *user_config.Repository {
		return &user_config.Repository{
			ID:         "test_" + repType,
			UserID:     userID,
			Device:     device,
			Type:       repType,
			File:       filepath.ToSlash(file),
			UseToWrite: true,
			IsEnable:   true,
		}
	}

	dbFileNameMap := map[string]string{
		"kmemo":        "Kmemo.db",
		"kc":           "KC.db",
		"urlog":        "URLog.db",
		"timeis":       "TimeIs.db",
		"mi":           "Mi.db",
		"nlog":         "Nlog.db",
		"lantana":      "Lantana.db",
		"tag":          "Tag.db",
		"text":         "Text.db",
		"notification": "Notification.db",
		"rekyou":       "ReKyou.db",
	}
	for repType, fileName := range dbFileNameMap {
		repositories = append(repositories, newRepository(repType, filepath.Join(dataDir, fileName)))
	}

	for repType, dirName := range map[string]string{"directory": "Files", "gpslog": "GPSLog"} {
		dir := filepath.Join(dataDir, dirName)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("MkdirAll %s failed: %v", dir, err)
		}
		repositories = append(repositories, newRepository(repType, dir))
	}

	return repositories
}

// glob展開先にgitリポジトリでないエントリが混ざっていても、
// GetRepositoriesは失敗せず、本物のgitリポジトリだけを読み込むこと。
//
// これが崩れると認証ミドルウェアがERR000018を返し、
// 対象ユーザはログイン直後から全APIが「内部エラー」になって何もできなくなる。
// 2026-08-15に $HOME/Git 直下のGit Bashのクラッシュダンプ
// (bash.exe.stackdump) 1ファイルで実際に発生した。
func TestGetRepositoriesSkipsNonGitEntriesInGitCommitLogGlob(t *testing.T) {
	tmpDir := setupGitRepGlobTestOptions(t)
	ctx := context.Background()

	reposRoot := filepath.Join(tmpDir, "Git")
	if err := os.MkdirAll(reposRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}

	// 本物のgitリポジトリ
	realRepoDir := filepath.Join(reposRoot, "real_repository")
	if err := os.MkdirAll(realRepoDir, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	initGitRepositoryForTest(t, realRepoDir)

	// gitリポジトリでないファイル(Git Bashのクラッシュダンプ)
	if err := os.WriteFile(filepath.Join(reposRoot, "bash.exe.stackdump"), []byte("stack trace\n"), 0o600); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// gitリポジトリでないディレクトリ
	if err := os.MkdirAll(filepath.Join(reposRoot, "not_a_git_repository"), 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}

	manager, err := NewGkillDAOManager()
	if err != nil {
		t.Fatalf("NewGkillDAOManager failed: %v", err)
	}
	t.Cleanup(func() { _ = manager.Close() })

	userID := "test_user"
	device := "test_device"
	dataDir := filepath.Join(tmpDir, "datas", userID)
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}

	// 検査は「デバイスごとに各種別の書き込み先repがちょうど1つ」なので
	// git_commit_logのrepだけを足すことはできない。1トランザクションでまとめて入れる
	repositoriesDefine := append(writeRepositoriesForTest(t, userID, device, dataDir), &user_config.Repository{
		ID:       "test_git_commit_log_glob",
		UserID:   userID,
		Device:   device,
		Type:     "git_commit_log",
		File:     filepath.ToSlash(reposRoot) + "/*",
		IsEnable: true,
	})
	if _, err := manager.ConfigDAOs.RepositoryDAO.AddRepositories(ctx, repositoriesDefine); err != nil {
		t.Fatalf("AddRepositories failed: %v", err)
	}

	repositories, err := manager.GetRepositories(userID, device)
	if err != nil {
		t.Fatalf("gitリポジトリでないエントリでGetRepositories全体が失敗している: %v", err)
	}
	if len(repositories.GitCommitLogReps) != 1 {
		t.Fatalf("本物のgitリポジトリだけが読み込まれるはず: got %d件", len(repositories.GitCommitLogReps))
	}

	repName, err := repositories.GitCommitLogReps[0].GetRepName(ctx)
	if err != nil {
		t.Fatalf("GetRepName failed: %v", err)
	}
	if repName != "real_repository" {
		t.Errorf("読み込まれたのが本物のgitリポジトリでない: got %q", repName)
	}
}
