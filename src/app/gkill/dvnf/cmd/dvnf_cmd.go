package dvnf_cmd

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	zglob "github.com/mattn/go-zglob"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/mt3hr/gkill/src/app/gkill/dvnf"
	"github.com/mt3hr/gkill/src/app/gkill/main/common/gkill_options"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	// フラグの設定
	pf := DVNFCmd.PersistentFlags()
	pf.StringVar(&rootOpt.configfile, configFileKey, "", "コンフィグファイル")
	pf.BoolVarP(&rootOpt.createNew, createNewKey, createNewKeyP, false, "新たにdvnfを作成します")
	pf.BoolVar(&rootOpt.autoCreate, autoCreateKey, true, "1つも存在しなかったときに自動で作成します。")
	viper.BindPFlags(pf)

	// コマンドの親子設定
	DVNFCmd.AddCommand(getCommand)
	DVNFCmd.AddCommand(moveCommand)
	DVNFCmd.AddCommand(copyCommand)

	// コンフィグの読み込みとpathの展開
	DVNFCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		setEnv()
		err := loadConfig()
		if err != nil {
			err = fmt.Errorf("error at load config file: %w", err)
			log.Fatal(err)
		}
		config.Directory = os.ExpandEnv(config.Directory)
	}
}

var (
	rootOpt = struct {
		configfile string
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

// 環境変数が設定されていなければ設定します
// ExpandEnvで使います。
func setEnv() {
	// HOME
	home := os.Getenv("HOME")
	if home == "" {
		home, err := homedir.Dir()
		if err != nil {
			err = fmt.Errorf("error at get user home directory: %w", err)
			log.Printf(err.Error())
		} else {
			os.Setenv("HOME", home)
		}
	}

	// EXE
	exe := os.Getenv("EXE")
	if exe == "" {
		exe, err := os.Executable()
		if err != nil {
			err = fmt.Errorf("error at get executable file path: %w", err)
			log.Printf(err.Error())
		} else {
			os.Setenv("EXE", exe)
		}
	}
}

func getConfigFile() string {
	return rootOpt.configfile
}
func getConfig() *Config {
	return config
}
func getConfigName() string {
	return "dvnf_config"
}
func getConfigExt() string {
	return ".yaml"
}
func createDefaultConfigYAML() string {
	return `Directory: $HOME/Datas
Device: PC
TimeLength: 8
`
}

func loadConfig() error {
	configOpt := getConfigFile()
	config := getConfig()
	configName := getConfigName()
	configExt := getConfigExt()

	v := viper.New()
	configPaths := []string{}
	if configOpt != "" {
		// コンフィグファイルが明示的に指定された場合はそれを
		v.SetConfigFile(configOpt)
		configPaths = append(configPaths, configOpt)
	} else {
		// 実行ファイルの親ディレクトリ、カレントディレクトリ、GkillHomeの順に
		v.SetConfigName(configName)
		exe, err := os.Executable()
		if err != nil {
			err = fmt.Errorf("error at get executable file path: %w", err)
			log.Printf(err.Error())
		} else {
			v.AddConfigPath(filepath.Dir(exe))
			configPaths = append(configPaths, filepath.Join(filepath.Dir(exe), configName+configExt))
		}

		v.AddConfigPath(".")
		configPaths = append(configPaths, filepath.Join(".", configName+configExt))

		v.AddConfigPath(os.ExpandEnv(gkill_options.GkillHomeDir))
		configPaths = append(configPaths, filepath.Join(os.ExpandEnv(gkill_options.GkillHomeDir), configName+configExt))
	}

	// 読み込んでcfgを作成する
	existConfigPath := false
	for _, configPath := range configPaths {
		if _, err := os.Stat(configPath); err == nil {
			existConfigPath = true
			break
		}
	}
	if !existConfigPath {
		// コンフィグファイルが指定されていなくてコンフィグファイルが見つからなかった場合、
		// ホームディレクトリにデフォルトコンフィグファイルを作成する。
		// できなければカレントディレクトリにコンフィグファイルを作成する。
		if configOpt == "" {
			configDir := os.ExpandEnv(gkill_options.GkillHomeDir)

			configFileName := filepath.Join(configDir, configName+configExt)
			err := os.WriteFile(configFileName, []byte(createDefaultConfigYAML()), os.ModePerm)
			if err != nil {
				err = fmt.Errorf("error at write file to %s: %w", configFileName, err)
				return err
			}
			v.SetConfigFile(configFileName)
		} else {
			err := fmt.Errorf("コンフィグファイルが見つかりませんでした。")
			return err
		}
	}

	err := v.ReadInConfig()
	if err != nil {
		err = fmt.Errorf("error at read in config: %w", err)
		return err
	}

	err = v.Unmarshal(config)
	if err != nil {
		err = fmt.Errorf("error at unmarshal config file: %w", err)
		return err
	}
	return nil
}
