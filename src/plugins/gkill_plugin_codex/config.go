package main

import (
	"encoding/json"
	"os"
	"strings"

	sdk "github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
)

// config.json のキー。
const (
	configKeySourceDirs   = "source_dirs"
	configKeySubagentMode = "subagent_mode"
	configKeyScanWorkers  = "scan_workers"
)

// JSONにはコメントが書けないので、読み飛ばされるキーで書式を書き残す。
const (
	configKeyComment           = "_comment"
	configKeyExampleSourceDirs = "_example_source_dirs"
)

// pluginConfig は config.json を解釈した結果。
type pluginConfig struct {
	Patterns     []string
	SubagentMode string
	ScanWorkers  int
}

// defaultConfig は config.json が無いときに書き出す既定設定。
//
// 既定は実ログの場所を指す。本番のように集約コピーを読ませたいときは
// 配置スクリプトが config.json を作るので、それが優先される。
func defaultConfig() sdk.Config {
	return sdk.Config{
		configKeyComment: "source_dirs に Codex のセッションログ(ロールアウトJSONL)が入った" +
			"フォルダかファイルのパスを書きます。" +
			"* ** ? [] のワイルドカード、先頭の ~ と環境変数($HOME など)が使えます。" +
			"フォルダを指定すると再帰的に走査して rollout-*.jsonl を探します。" +
			"session_index.jsonl も指定しておくとスレッド名が表示されます。" +
			"空にすると ~/.codex/sessions を見ます。" +
			"subagent_mode は \"fold\"(既定・サブエージェントの会話を親の応答に畳み込む)か " +
			"\"own_kyou\"(サブエージェントも独立したKyouにする)。" +
			"scan_workers は同時に読むファイル数で、0 なら自動。" +
			"編集は次の検索から反映されます(gkillの再起動は不要)。" +
			"_ で始まるキーは説明用なので消して構いません。",
		configKeyExampleSourceDirs: []string{
			"~/.codex/sessions",
			"~/.codex/session_index.jsonl",
			"D:/backup/Codex_*/**/rollout-*.jsonl",
		},
		configKeySourceDirs: []string{
			"~/.codex/sessions",
			"~/.codex/" + sessionIndexFileName,
		},
		configKeySubagentMode: subagentModeFold,
		configKeyScanWorkers:  0,
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

	config := pluginConfig{SubagentMode: subagentModeFold}
	if current == nil {
		config.Patterns = defaultSourcePatterns()
		return config
	}

	config.Patterns = sdk.ParseSourcePatterns(current[configKeySourceDirs], "")
	if len(config.Patterns) == 0 {
		config.Patterns = defaultSourcePatterns()
	}
	config.SubagentMode = parseSubagentMode(current[configKeySubagentMode])
	config.ScanWorkers = parseWorkers(current[configKeyScanWorkers])
	return config
}

// parseSubagentMode は知らない値を既定(fold)に倒す。
func parseSubagentMode(value any) string {
	if text, ok := value.(string); ok && strings.TrimSpace(text) == subagentModeOwnKyou {
		return subagentModeOwnKyou
	}
	return subagentModeFold
}

// parseWorkers は同時に読むファイル数を取り出す。JSONの数値は float64 で来る。
func parseWorkers(value any) int {
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
