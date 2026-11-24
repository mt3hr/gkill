package dao

// 読むだけのディレクトリ rep を読み込むとき、gkill が ID 表（.gkill/gkill_id.db）を作らないことの回帰テスト。
// 読むだけの rep のファイルは端末同期で運ばれてくるもので、作ってよいのは書き込み rep を初めて置くときだけ。
// 作ると、消えたディレクトリが空の ID 表つきで復活し、同期で全保存側の ID 表を上書きする（ID が振り直され、
// そのファイルに付いたタグやテキストが宙に浮く）。
// ファイル1つの rep（kmemo など）は、読むだけで置き場が無いとファイル名の展開が0件になり、ここまで届かない。
// documents/adr/0223-never-recreate-a-missing-rep-db.md

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/mt3hr/gkill/src/server/gkill/dao/user_config"
)

func TestGetRepositories_ReadOnlyDirectoryRepDoesNotCreateIDDB(t *testing.T) {
	tmpDir := setupGitRepGlobTestOptions(t)
	ctx := context.Background()

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

	// 読むだけのディレクトリ rep。対象ディレクトリはあるが ID 表（.gkill）はまだ運ばれてきていない
	readOnlyDir := filepath.Join(tmpDir, "readonly_files")
	if err := os.MkdirAll(readOnlyDir, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	repositoriesDefine := append(writeRepositoriesForTest(t, userID, device, dataDir),
		&user_config.Repository{
			ID: "test_directory_read_only", UserID: userID, Device: device, Type: "directory",
			File: filepath.ToSlash(readOnlyDir), UseToWrite: false, IsEnable: true,
		},
	)
	if _, err := manager.ConfigDAOs.RepositoryDAO.AddRepositories(ctx, repositoriesDefine); err != nil {
		t.Fatalf("AddRepositories failed: %v", err)
	}

	// 読めない rep は切り離されて警告になるだけで、ほかの rep は読み込まれる（ADR-0216）
	repositories, err := manager.GetRepositories(userID, device)
	if err != nil {
		t.Fatalf("GetRepositories failed: %v", err)
	}
	// ID 表の無い読むだけのディレクトリ rep は切り離される（書き込み用の IDF rep 1本だけが残る）
	if len(repositories.IDFKyouReps) != 1 {
		t.Errorf("IDFKyouReps = %d 本, want 1本（読むだけのディレクトリ rep は切り離される）", len(repositories.IDFKyouReps))
	}
	if _, err := os.Stat(filepath.Join(readOnlyDir, ".gkill")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("読むだけのディレクトリ rep に .gkill が作られている（stat err = %v）", err)
	}
}
