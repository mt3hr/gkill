package dvnf_cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	zglob "github.com/mattn/go-zglob"
	"github.com/mt3hr/gkill/src/app/gkill/dao/server_config"
	"github.com/mt3hr/gkill/src/app/gkill/dvnf"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_log"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_options"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	// フラグの設定
	pf := DVNFCmd.PersistentFlags()
	pf.BoolVarP(&rootOpt.createNew, createNewKey, createNewKeyP, false, "新たにdvnfを作成します")
	pf.BoolVar(&rootOpt.autoCreate, autoCreateKey, true, "1つも存在しなかったときに自動で作成します。")
	viper.BindPFlags(pf)

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
			gkill_log.Debug.Printf(err.Error())
			return
		}

		var currentServerConfig *server_config.ServerConfig
		serverConfigs, err := serverConfigDAO.GetAllServerConfigs(ctx)
		if err != nil {
			err = fmt.Errorf("error at get serverConfig: %w", err)
			gkill_log.Debug.Printf(err.Error())
			return
		}
		for _, serverConfig := range serverConfigs {
			if serverConfig.EnableThisDevice {
				currentServerConfig = serverConfig
				break
			}
		}

		config.Directory = os.ExpandEnv(fmt.Sprintf("$HOME/%s", currentServerConfig.Device))
		config.Device = currentServerConfig.Device
		config.TimeLength = 8
	}
}

var (
	rootOpt = struct {
		createNew  bool
		autoCreate bool
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

// dvnfが展開されれたpathを返します
// filepath.Join(dvnf.GetLatestDVNF(opt), subDir)すればいいよ
func expandDVNFPathium(cfg *Config, dvnfPathium string, splitExtension bool) (opt *dvnf.Option, subDir string) {
	dvnfPathium = filepath.Clean(dvnfPathium)
	pathList := filepath.SplitList(dvnfPathium)
	root := pathList[0]
	sub := ""
	if len(pathList) >= 2 {
		sub = filepath.Join(sub)
	}

	// dvnfの拡張子をどうするかの処理。
	// 指定されなければ拡張子とは分割しない
	name := root
	ext := ""
	if splitExtension {
		ext = filepath.Ext(root)
		withoutExt := root[:len(root)-len(ext)]
		name = withoutExt
	}

	// optを作って返す
	opt = &dvnf.Option{
		Directory:  cfg.Directory,
		Name:       name,
		Device:     cfg.Device,
		TimeLength: cfg.TimeLength,
		Extension:  ext,
	}
	return opt, sub
}
