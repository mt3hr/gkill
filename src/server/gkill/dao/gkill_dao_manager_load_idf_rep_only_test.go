package dao

// LoadIDFRepOnly は generate_thumb_cache / generate_video_cache / clear_cache が立てる。
// 「IDF以外のrepを作らない」だけでなく、そこへ辿り着くまでの走査
// （パターンの展開とディレクトリ作成）も飛ばせていないと意味がない。
// 展開は対象ディレクトリの列挙を伴うので、外付けストレージ上のrepでは1本あたり秒単位かかる。

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mt3hr/gkill/src/server/gkill/dao/user_config"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_options"
)

// LoadIDFRepOnlyが立っているとき、IDFのrepだけが読み込まれ、
// IDF以外のrep定義はパターンの展開もディレクトリ作成もされないこと。
func TestGetRepositoriesLoadIDFRepOnlySkipsBeforeExpandingPatterns(t *testing.T) {
	tmpDir := setupGitRepGlobTestOptions(t)
	ctx := context.Background()

	origLoadIDFRepOnly := gkill_options.LoadIDFRepOnly
	gkill_options.LoadIDFRepOnly = true
	t.Cleanup(func() { gkill_options.LoadIDFRepOnly = origLoadIDFRepOnly })

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

	// 親ディレクトリが存在しないIDF以外のrep。
	// 引き上げた判定より後ろでos.MkdirAllが走ると、ここが作られてしまう。
	neverCreatedDir := filepath.Join(tmpDir, "never_created")
	repositoriesDefine := append(writeRepositoriesForTest(t, userID, device, dataDir), &user_config.Repository{
		ID:       "test_kmemo_not_loaded",
		UserID:   userID,
		Device:   device,
		Type:     "kmemo",
		File:     filepath.ToSlash(filepath.Join(neverCreatedDir, "Kmemo_x.db")),
		IsEnable: true,
	})
	if _, err := manager.ConfigDAOs.RepositoryDAO.AddRepositories(ctx, repositoriesDefine); err != nil {
		t.Fatalf("AddRepositories failed: %v", err)
	}

	repositories, err := manager.GetRepositories(userID, device)
	if err != nil {
		t.Fatalf("GetRepositories failed: %v", err)
	}

	if len(repositories.IDFKyouReps) != 1 {
		t.Errorf("IDFのrepは読み込まれるはず: got %d件", len(repositories.IDFKyouReps))
	}

	notLoaded := map[string]int{
		"KmemoReps":        len(repositories.KmemoReps),
		"KCReps":           len(repositories.KCReps),
		"URLogReps":        len(repositories.URLogReps),
		"TimeIsReps":       len(repositories.TimeIsReps),
		"MiReps":           len(repositories.MiReps),
		"NlogReps":         len(repositories.NlogReps),
		"LantanaReps":      len(repositories.LantanaReps),
		"TagReps":          len(repositories.TagReps),
		"TextReps":         len(repositories.TextReps),
		"NotificationReps": len(repositories.NotificationReps),
		"ReKyouReps":       len(repositories.ReKyouReps.ReKyouRepositories),
		"GPSLogReps":       len(repositories.GPSLogReps),
		"GitCommitLogReps": len(repositories.GitCommitLogReps),
	}
	for name, count := range notLoaded {
		if count != 0 {
			t.Errorf("LoadIDFRepOnlyなのに%sが読み込まれている: got %d件", name, count)
		}
	}

	if _, err := os.Stat(neverCreatedDir); err == nil {
		t.Errorf("IDF以外のrepの親ディレクトリが作られている: %s。判定がos.MkdirAllより後ろに戻っている", neverCreatedDir)
	}

	// プラグインの探索は1プラグインにつき1サブプロセスを起動する。
	// IDFのrepしか使わない経路では丸ごと不要で、実データではここが
	// GetRepositories の大半を占めていた。
	// PluginManager が作られていないことで、探索へ入っていないことを見る。
	if _, exist := manager.pluginManagers[userID]; exist {
		t.Error("LoadIDFRepOnlyなのにプラグインの探索へ入っている")
	}
	if len(repositories.PluginReps) != 0 {
		t.Errorf("LoadIDFRepOnlyなのにプラグインのrepが読み込まれている: got %d件", len(repositories.PluginReps))
	}
}
