package main

import (
	"encoding/json"
	"os"
	"strings"

	sdk "github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
)

// config.json のキー。
const configKeySourceDirs = "source_dirs"

// JSONにはコメントが書けないので、読み飛ばされるキーで書式を書き残す。
const (
	configKeyComment           = "_comment"
	configKeyExampleSourceDirs = "_example_source_dirs"
)

// pluginConfig は config.json を解釈した結果。
type pluginConfig struct {
	Patterns []string
}

// defaultConfig は config.json が無いときに書き出す既定設定。
//
// 既定は VRChat が書き出す場所を指す。VRChat は古いログを数日で消すので、
// 消える前に同期スクリプトで別の場所へ運んでいるなら、その場所も足す（下の例）。
func defaultConfig() sdk.Config {
	return sdk.Config{
		configKeyComment: "source_dirs に VRChat の出力ログ(output_log_*.txt)が入った" +
			"フォルダかファイルのパスを書きます。" +
			"* ** ? [] のワイルドカード、先頭の ~ と環境変数($HOME など)が使えます。" +
			"フォルダを指定すると再帰的に走査して output_log_*.txt を探します。" +
			"VRChat は古いログを数日で消すので、別の場所へ運んでいるならその場所も書いてください。" +
			"同じログが複数の場所にあっても(運んだコピーと元のファイル)、1回の起動は1回ぶんとして数えます。" +
			"空にすると VRChat の既定の書き出し先を見ます。" +
			"編集は次の検索から反映されます(gkillの再起動は不要)。" +
			"_ で始まるキーは説明用なので消して構いません。",
		configKeyExampleSourceDirs: []string{
			"~/AppData/LocalLow/VRChat/VRChat",
			"$HOME/Kyou/VRChat_*",
		},
		configKeySourceDirs: []string{
			"~/AppData/LocalLow/VRChat/VRChat",
		},
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

	config := pluginConfig{}
	if current != nil {
		config.Patterns = sdk.ParseSourcePatterns(current[configKeySourceDirs], "")
	}
	if len(config.Patterns) == 0 {
		config.Patterns = defaultSourcePatterns()
	}
	return config
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
