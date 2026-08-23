package dvnf_cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	zglob "github.com/mattn/go-zglob"
	"github.com/mt3hr/gkill/src/server/gkill/dao/server_config"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_options"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	// フラグの設定
	pf := DVNFCmd.PersistentFlags()
	pf.BoolVarP(&rootOpt.createNew, createNewKey, createNewKeyP, false, "新たにdvnfを作成します")
	pf.BoolVar(&rootOpt.autoCreate, autoCreateKey, true, "1つも存在しなかったときに自動で作成します。")
	pf.StringVar(&rootOpt.device, deviceKey, "", "dvnf名に使う端末名。省略時はこの端末の名前を使います。他の端末で集めたものを取り込むときに指定します")
	err := viper.BindPFlags(pf)
	if err != nil {
		panic(err)
	}

	// コマンドの親子設定
	DVNFCmd.AddCommand(getCommand)
	DVNFCmd.AddCommand(moveCommand)
	DVNFCmd.AddCommand(copyCommand)

	// コンフィグの読み込みとpathの展開
	DVNFCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		ctx := context.Background()

		configDBRootDir := os.ExpandEnv(gkill_options.ConfigDir)
		serverConfigDAO, err := server_config.NewServerConfigDAOSQLite3Impl(ctx, filepath.Join(configDBRootDir, "server_config.db"))
		if err != nil {
			err = fmt.Errorf("error at get serverConfig: %w", err)
			slog.Log(ctx, gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
			return
		}

		var currentServerConfig *server_config.ServerConfig
		serverConfigs, err := serverConfigDAO.GetAllServerConfigs(ctx)
		if err != nil {
			err = fmt.Errorf("error at get serverConfig: %w", err)
			slog.Log(ctx, gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
			return
		}
		for _, serverConfig := range serverConfigs {
			if serverConfig.EnableThisDevice {
				currentServerConfig = serverConfig
				break
			}
		}

		// この端末の設定が見つからない場合でも、--device が指定されていれば動かせるようにする。
		if currentServerConfig == nil {
			if rootOpt.device == "" {
				err = fmt.Errorf("this device has no enabled server config. specify --%s", deviceKey)
				slog.Log(ctx, gkill_log.Debug, "error", "error", fmt.Sprintf("%q", err))
				return
			}
			config.Directory = os.ExpandEnv(fmt.Sprintf("$HOME/%s", rootOpt.device))
			config.Device = rootOpt.device
			config.TimeLength = 8
			return
		}

		// dvnfのルートはこの端末のものを使う。
		// --device は生成する名前の端末部分だけを差し替えるためのもので、
		// 他の端末で集めたものを、この端末のdvnfへ端末名を保ったまま取り込むのに使う。
		config.Directory = os.ExpandEnv(fmt.Sprintf("$HOME/%s", currentServerConfig.Device))
		config.Device = currentServerConfig.Device
		if rootOpt.device != "" {
			config.Device = rootOpt.device
		}
		config.TimeLength = 8
	}
}

var (
	rootOpt = struct {
		createNew  bool
		autoCreate bool
		device     string
	}{}

	config = &Config{}
)

type Config struct {
	Directory  string
	Device     string
	TimeLength int
}

const (
	configFileKey  = "config_file"
	createNewKey   = "new"
	createNewKeyP  = "n"
	configFileName = "dvnf_config"
	autoCreateKey  = "auto_create"
	deviceKey      = "device"
)

var (
	DVNFCmd = &cobra.Command{
		Use: "dvnf",
	}
	getCommand = &cobra.Command{
		Run:   runGet,
		Args:  cobra.MaximumNArgs(1),
		Use:   "get",
		Short: "dvnfディレクトリのパスを取得する",
		Long: `dvnf get [dvnfPath]
	dvnfディレクトリのパスを取得します。
	オプションを渡さなかった場合はdvnfのルートフォルダを取得します。`,
	}
	moveCommand = &cobra.Command{
		Run:   runMove,
		Args:  cobra.ExactArgs(2),
		Use:   "move",
		Short: "ファイルやディレクトリをdvnfディレクトリに移動する",
		Long: `dvnf move src target
	ファイルやディレクトリをdvnfディレクトリへと移動します。
	移動元が存在しないときには何もせず、移動先の親ディレクトリが存在しないときは作成します。
	src: 移動元ファイル、あるいはディレクトリのパス
	target: 移動先dvnfパス`,
	}
	copyCommand = &cobra.Command{
		Run:   runCopy,
		Args:  cobra.ExactArgs(2),
		Use:   "copy",
		Short: "ファイルやディレクトリをdvnfディレクトリにコピーする",
		Long: `dvnf copy src target
	ファイルやディレクトリをdvnfディレクトリへとコピーします。
	移動元が存在しないときには何もせず、移動先の親ディレクトリが存在しないときは作成します。
	src: コピー元ファイル、あるいはディレクトリのパス
	target: 移動先dvnfパス`,
	}
)

// Globパターンを展開し、マッチするpathを取得します。
func glob(pattern string) ([]string, error) {
	return zglob.Glob(pattern)
}

// 除外するかどうかを判定します。
//
// ignoresはファイル名の完全一致、ignorePatternsはfilepath.Matchのパターンです。
// 完全一致を残してあるのは、既定の除外リスト(gkill_options.IDFIgnore)が
// そのまま渡ってくるためです。すべてパターン扱いにすると、名前に [ や *
// を含むファイルの扱いが変わってしまいます。
//
// パターンの妥当性はvalidateIgnorePatternsが先に見ているので、
// ここではMatchのエラーを無視して構いません。
func isIgnored(name string, ignores []string, ignorePatterns []string) bool {
	for _, ignore := range ignores {
		if ignore == name {
			return true
		}
	}
	for _, pattern := range ignorePatterns {
		if matched, err := filepath.Match(pattern, name); err == nil && matched {
			return true
		}
	}
	return false
}

// 除外パターンが壊れていないかを確認します。
//
// 壊れたパターンは黙って「何にも当たらない」になります。除外し損ねると
// 書きかけのファイルを運んでしまうので、動き出す前に気づけるようにします。
func validateIgnorePatterns(ignorePatterns []string) error {
	for _, pattern := range ignorePatterns {
		if _, err := filepath.Match(pattern, ""); err != nil {
			return fmt.Errorf("不正な除外パターンです %s: %w", pattern, err)
		}
	}
	return nil
}
