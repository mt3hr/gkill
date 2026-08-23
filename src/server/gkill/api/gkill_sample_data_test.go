package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/dao"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_options"
)

// TestGkillSampleData は、配布サンプル resources/gkill_sample_data（コミット済みの
// 静的な gkill_home 一式）が現行コードでそのまま動くことを検証する。
// サンプルデータは手作業更新のスナップショットで、他のどのテストも参照していないため、
// スキーマ移行や rep 種別の追加から取り残されても目の前ではエラーにならない
// （過去に「古いDBが移行されず壊れる」事故が実際に起きている）。
//
// 検証は必ずテンポラリへのコピーに対して行う。DAO は開くだけで CREATE TABLE・
// スキーマ移行・IDF走査により DB を変異させるので、コミット済みの resources/ 配下の
// パスを DAO へ渡してはいけない（リポジトリが汚れる）。
func TestGkillSampleData(t *testing.T) {
	ctx := context.Background()

	sampleDir := findGkillSampleDataDir(t)

	// t.TempDir() ではなく os.MkdirTemp を使う。SQLite がファイルハンドルを
	// 掴んだままだと Windows で自動 cleanup が失敗するため（gkill_dao_manager_test.go と同じ理由）。
	tmpHome, err := os.MkdirTemp("", "gkill_sample_data_home_*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	// 削除は defer ではなく最初に登録する t.Cleanup で行う。t.Cleanup は LIFO なので、
	// 後で登録する DAO クローズより後に走り、SQLite のハンドル解放後に消せる
	// （defer だと DAO クローズ前に走り、Windows ではロック中で36MBが丸ごと残る）。
	// それでも caches/ の git_commit_log キャッシュDBだけはプロセス終了まで
	// ハンドルが残るため best-effort（残っても数百KB）。
	t.Cleanup(func() { os.RemoveAll(tmpHome) })

	if err := os.CopyFS(tmpHome, os.DirFS(sampleDir)); err != nil {
		t.Fatalf("サンプルデータのコピーに失敗: %v", err)
	}

	// gkill_options のグローバルをテンポラリ home へ差し替える（gkill_server_api_test.go の写し）
	origHome := gkill_options.GkillHomeDir
	origLib := gkill_options.LibDir
	origCache := gkill_options.CacheDir
	origLog := gkill_options.LogDir
	origConfig := gkill_options.ConfigDir
	origData := gkill_options.DataDirectoryDefault
	gkill_options.GkillHomeDir = tmpHome
	gkill_options.LibDir = filepath.Join(tmpHome, "lib", "base_directory")
	gkill_options.CacheDir = filepath.Join(tmpHome, "caches")
	gkill_options.LogDir = filepath.Join(tmpHome, "logs")
	gkill_options.ConfigDir = filepath.Join(tmpHome, "configs")
	gkill_options.DataDirectoryDefault = filepath.Join(tmpHome, "datas")
	t.Cleanup(func() {
		gkill_options.GkillHomeDir = origHome
		gkill_options.LibDir = origLib
		gkill_options.CacheDir = origCache
		gkill_options.LogDir = origLog
		gkill_options.ConfigDir = origConfig
		gkill_options.DataDirectoryDefault = origData
	})

	// rep 定義の $GKILL_HOME はこの環境変数で展開される（gkill_dao_manager.go の os.ExpandEnv）。
	// 本番では InitGkillOptions が設定するが、テストでは自前で立てる。
	t.Setenv("GKILL_HOME", tmpHome)

	// ここで account.db のスキーマ版検査（旧版なら移行）が走る。
	manager, err := dao.NewGkillDAOManager()
	if err != nil {
		t.Fatalf("サンプルデータの configs で NewGkillDAOManager が失敗: %v", err)
	}
	t.Cleanup(func() {
		// Close() は GkillNotificationTargetDAO を閉じ忘れる既知の穴があるので、
		// 参照を控えて個別に閉じる（gkill_server_api_test.go と同じ対処）。
		notifDAO := manager.ConfigDAOs.GkillNotificationTargetDAO
		manager.Close()
		if notifDAO != nil {
			notifDAO.Close(ctx)
		}
	})

	// サンプルの server_config は ENABLE_THIS_DEVICE の行が device 名を持つ
	// （utils.go の GetDevice と同じ手順）。以降のサブテストが共有する。
	serverConfigs, err := manager.ConfigDAOs.ServerConfigDAO.GetAllServerConfigs(ctx)
	if err != nil {
		t.Fatalf("GetAllServerConfigs: %v", err)
	}
	device := ""
	for _, serverConfig := range serverConfigs {
		if serverConfig.EnableThisDevice {
			device = serverConfig.Device
		}
	}
	if device == "" {
		t.Fatal("ENABLE_THIS_DEVICE な server_config の行がサンプルに無い")
	}

	t.Run("account_auth", func(t *testing.T) {
		acc, err := manager.ConfigDAOs.AccountDAO.GetAccount(ctx, "gkill_sample_data")
		if err != nil {
			t.Fatalf("GetAccount(gkill_sample_data): %v", err)
		}
		if acc == nil {
			t.Fatal("サンプルアカウント gkill_sample_data が account.db に居ない")
		}
		if !acc.IsEnable {
			t.Error("gkill_sample_data が無効化されている")
		}
		if acc.PasswordHash == nil || *acc.PasswordHash == "" {
			t.Fatal("gkill_sample_data の PasswordHash が空。account.db がスキーマ移行で" +
				"パスワード無効化された可能性が高い（サンプルDBが現行スキーマから取り残されている）")
		}

		// ログインと同じ資格情報形式: クライアントは hex(sha256(パスワード)) を送る
		sum := sha256.Sum256([]byte("sample"))
		credential := hex.EncodeToString(sum[:])
		ok, err := acc.VerifyPassword(credential)
		if err != nil {
			t.Fatalf("VerifyPassword: %v", err)
		}
		if !ok {
			t.Error("README 記載のパスワード sample で認証が通らない")
		}

		wrongSum := sha256.Sum256([]byte("wrong"))
		ok, err = acc.VerifyPassword(hex.EncodeToString(wrongSum[:]))
		if err != nil {
			t.Fatalf("VerifyPassword(誤パスワード): %v", err)
		}
		if ok {
			t.Error("誤ったパスワードで認証が通ってしまう")
		}

		// admin はパスワード未設定（リセット待ち）で配布されるので存在だけ見る
		admin, err := manager.ConfigDAOs.AccountDAO.GetAccount(ctx, "admin")
		if err != nil {
			t.Fatalf("GetAccount(admin): %v", err)
		}
		if admin == nil {
			t.Error("admin アカウントが account.db に居ない")
		}
	})

	t.Run("repository_definitions", func(t *testing.T) {
		if device != "gkill" {
			t.Errorf("有効デバイス名 = %q, want %q", device, "gkill")
		}

		// この Scan 自体が REPOSITORY 真偽値の storage class の回帰検出になる
		// （REAL で書かれた行があると Go 側の Scan が落ちる）
		repositories, err := manager.ConfigDAOs.RepositoryDAO.GetRepositories(ctx, "gkill_sample_data", device)
		if err != nil {
			t.Fatalf("GetRepositories: %v", err)
		}
		// rep 定義は設定でありデータの手動更新では揺れないので厳密一致で見る。
		// サンプルに rep を足したり消したりしたときはここも更新すること。
		if len(repositories) != 14 {
			t.Errorf("rep 定義が %d 件, want 14 件", len(repositories))
		}

		// GkillDAOManager.GetRepositories の rep 種別 switch は未知の種別を
		// 黙ってスキップする（default 節が無い）ため、この照合が唯一の防御線。
		// 集合は gkill_dao_manager.go の switch の case と一致させること。
		knownTypes := map[string]struct{}{
			"kmemo": {}, "kc": {}, "urlog": {}, "timeis": {}, "mi": {},
			"nlog": {}, "lantana": {}, "tag": {}, "text": {}, "notification": {},
			"rekyou": {}, "mirekyou": {}, "directory": {}, "gpslog": {}, "git_commit_log": {},
		}
		for _, repository := range repositories {
			if _, ok := knownTypes[repository.Type]; !ok {
				t.Errorf("rep %q の種別 %q は現行コードに存在しない（黙ってスキップされ、そのrepのデータが消える）",
					repository.ID, repository.Type)
			}
			// 本番の展開経路（gkill_dao_manager.go の os.ExpandEnv）と同じ式でパスを解決する
			resolved := os.ExpandEnv(repository.File)
			if _, err := os.Stat(resolved); err != nil {
				t.Errorf("rep %q (type=%s) のパス %q が実在しない: %v",
					repository.ID, repository.Type, repository.File, err)
			}
		}
	})

	t.Run("find_kyous", func(t *testing.T) {
		findFilter := &FindFilter{}
		// ゼロ値の FindQuery = 全フィルタ未使用（null 意味論）= 非削除の全 Kyou
		kyous, gkillErrs, err := findFilter.FindKyous(ctx, "gkill_sample_data", device, manager, &find.FindQuery{})
		if err != nil {
			t.Fatalf("FindKyous: %v", err)
		}
		if len(gkillErrs) != 0 {
			t.Fatalf("FindKyous が GkillError を返した: %+v", gkillErrs)
		}
		if len(kyous) == 0 {
			t.Fatal("サンプルデータの全件検索が0件")
		}

		countsByDataType := map[string]int{}
		for _, kyou := range kyous {
			countsByDataType[kyou.DataType]++
		}
		// Mi / TimeIs の DataType は射影つき（mi_start / timeis_start 等）なので、
		// 「基底型そのもの」か「基底型 + _」の前方一致で数える。境界の _ を挟むのは
		// mirekyou_* を mi に誤マッチさせないため（プレフィックス判定の既知の罠）。
		countByBaseType := func(base string) int {
			total := 0
			for dataType, count := range countsByDataType {
				if dataType == base || strings.HasPrefix(dataType, base+"_") {
					total += count
				}
			}
			return total
		}
		// 完全一致件数はサンプルの手動更新のたびに壊れるので下限（≥1）だけ見る。
		// kc / rekyou / mirekyou / notification は0件で配布されているのでアサートしない。
		for _, baseType := range []string{"kmemo", "timeis", "mi", "nlog", "lantana", "urlog", "idf"} {
			if countByBaseType(baseType) == 0 {
				t.Errorf("data_type=%q 系の Kyou が0件。サンプルデータのこの rep が現行コードで読めていない。集計: %v",
					baseType, countsByDataType)
			}
		}
	})

	t.Run("no_secret_material", func(t *testing.T) {
		// サンプルの server_config に Web Push の鍵材料を入れて配布しない。
		// 空であれば設定更新時にサーバが自動再生成する（handle_update_server_configs.go）。
		// ここが非空になったら、実環境の設定DBをサンプルへコピーした可能性を疑うこと。
		for _, serverConfig := range serverConfigs {
			if serverConfig.GkillNotificationPrivateKey != "" {
				t.Error("サンプルの server_config.db に GKILL_NOTIFICATION_PRIVATE_KEY の実値が入っている（秘密鍵は配布物に含めない）")
			}
			if serverConfig.GkillNotificationPublicKey != "" {
				t.Error("サンプルの server_config.db に GKILL_NOTIFICATION_PUBLIC_KEY が入っている（鍵ペアは空で配布し、利用者環境で自動生成させる）")
			}
		}
	})
}

// findGkillSampleDataDir はカレントディレクトリから親方向へ resources/gkill_sample_data を探す。
// go test の cwd はパッケージディレクトリなので、リポジトリルートまで遡って見つける。
// 見つからないときは Skip ではなく Fatal にする（Skip だとこの検証が静かに消えるため）。
func findGkillSampleDataDir(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	for range 8 {
		candidate := filepath.Join(dir, "resources", "gkill_sample_data")
		if info, err := os.Stat(filepath.Join(candidate, "configs")); err == nil && info.IsDir() {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatal("resources/gkill_sample_data が見つからない（リポジトリ外で実行している可能性）")
	return ""
}
