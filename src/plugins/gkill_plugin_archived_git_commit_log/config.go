package main

import (
	"encoding/json"
	"os"
	"strings"

	sdk "github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
)

// pluginConfig は config.json を解釈した結果。
type pluginConfig struct {
	// Patterns は zip の実パス・フォルダ・glob。展開は sdk.OpenSources が行う。
	Patterns []string
	// MaxGitDirBytes は1リポジトリの .git の合計サイズの上限（バイト）。
	MaxGitDirBytes int64
}

// defaultConfig は config.json が無いときに書き出す既定設定。
//
// 既定の source_dirs は空。対象の zip は利用者ごとに違い、実パスを書くものなので、
// 配置スクリプトか設定画面で入れる。
func defaultConfig() sdk.Config {
	return sdk.Config{
		configKeyComment: "source_dirs に、Git リポジトリを固めた zip の実パスか、zip を置いたフォルダを書きます。" +
			"* ** ? [] のワイルドカード、先頭の ~ と環境変数($HOME など)が使えます。" +
			"フォルダを指定すると配下の *.zip を再帰的に走査します。" +
			"zip は展開せず、中の .git だけを読みます（.git の無い zip は読み飛ばします）。" +
			"1つの zip に複数のリポジトリが入っていてもかまいません。" +
			"max_git_dir_mb は1リポジトリの .git の合計サイズの上限(MB)で、超えるものは読みません。" +
			"編集は次の検索から反映されます(gkillの再起動は不要)。" +
			"_ で始まるキーは説明用なので消して構いません。",
		configKeyExampleSourceDirs: []string{
			"$HOME/Kyou/Box_Laptop_20221210/myrepo.zip",
			"$HOME/Kyou/Box_Desktop_20180826",
			"D:/backup/repositories/*.zip",
		},
		configKeySourceDirs:  []string{},
		configKeyMaxGitDirMB: defaultMaxGitDirMB,
	}
}

// printDefaultConfig は既定の config.json を標準出力に書く。--gkill-print-config 用。
func printDefaultConfig() error {
	data, err := json.MarshalIndent(defaultConfig(), "", "  ")
	if err != nil {
		return err
	}
	_, err = os.Stdout.Write(append(data, '\n'))
	return err
}

// configOf は設定を読み直して解釈する。
//
// SDKはconfig.jsonをプロセス起動時に一度だけ読むが、この設定はプラグインフォルダの
// config.jsonを手で編集したり設定画面から保存したりして変えるものなので、
// 毎回読み直して即座に反映されるようにする。
// 読めなかったときは起動時に読んだ設定にフォールバックする。
func configOf(pluginDir string, cfg sdk.Config) pluginConfig {
	current := cfg
	if latest, err := sdk.LoadConfig(pluginDir); err == nil {
		current = latest
	}

	config := pluginConfig{MaxGitDirBytes: defaultMaxGitDirMB << 20}
	if current == nil {
		return config
	}
	config.Patterns = sdk.ParseSourcePatterns(current[configKeySourceDirs], "")
	if mb := parsePositiveInt(current[configKeyMaxGitDirMB]); mb > 0 {
		config.MaxGitDirBytes = int64(mb) << 20
	}
	return config
}

// parsePositiveInt は正の整数を取り出す。JSONの数値は float64 で来る。0以下・非数値は 0。
func parsePositiveInt(value any) int {
	switch typed := value.(type) {
	case float64:
		if typed > 0 {
			return int(typed)
		}
	case int:
		if typed > 0 {
			return typed
		}
	case json.Number:
		if parsed, err := typed.Int64(); err == nil && parsed > 0 {
			return int(parsed)
		}
	}
	return 0
}

// splitSourceDirsForm は設定画面のテキストエリア(1行1指定)を config.json 用の配列にする。
// 空行は落とす。展開(~ や環境変数)はしない —— 書いたとおりを保存し、
// 読み出し時に ParseSourcePatterns が展開する。
func splitSourceDirsForm(v string) []string {
	dirs := make([]string, 0)
	for line := range strings.SplitSeq(v, "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if line == "" {
			continue
		}
		dirs = append(dirs, line)
	}
	return dirs
}
