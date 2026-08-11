package main

import (
	"strconv"
	"strings"

	sdk "github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
)

// pluginConfig は設定を解釈した結果。
type pluginConfig struct {
	Patterns    []string
	Source      sdk.ExpandedSource
	Timezone    string
	Metrics     []string
	ScanWorkers int
}

// enabledMetrics は取り込む指標のキー集合を返す。空なら全部。
func (c pluginConfig) enabledMetrics() map[string]struct{} {
	if len(c.Metrics) == 0 {
		return nil
	}
	enabled := map[string]struct{}{}
	for _, key := range c.Metrics {
		enabled[key] = struct{}{}
	}
	return enabled
}

// defaultConfig は config.json が無いときに書き出す既定設定。
func defaultConfig() sdk.Config {
	return sdk.Config{
		configKeyComment: "source_dirs に Google Takeout の ZIP を置いたフォルダを書きます。" +
			"ZIP は展開せずそのまま置いてください(展開したフォルダは読みません)。" +
			"フォルダを指定すると、その下の *.zip を再帰的に探します。ZIP を直接指定しても構いません。" +
			"* ** ? [] のワイルドカード、先頭の ~ と環境変数($HOME など)が使えます。" +
			"1つのフォルダに置いた ZIP は同じ書き出しとして合算するので、" +
			"分割された takeout-....-001.zip / -002.zip はそのまま並べて置けます。" +
			"新しい書き出しは別のフォルダに置いてください。" +
			"日付が重なったときは新しい書き出しの値だけを使います(合算しません)。" +
			"timezone は「この日はどの日か」を決めるタイムゾーンです(既定 Asia/Tokyo)。" +
			"変えると集計をやり直します。" +
			"metrics を空にすると全指標を取り込みます。" +
			"scan_workers は同時に読むファイル数で、0 なら自動。" +
			"編集は次の検索から反映されます(gkill の再起動は不要)。" +
			"_ で始まるキーは説明用なので消して構いません。",
		configKeyExampleSourceDirs: []string{
			"~/Kyou/GoogleTakeout_*",
			"~/Downloads/takeout-20260808T230152Z-1-001.zip",
			"D:/backup/GoogleTakeout_*",
		},
		configKeySourceDirs:  []string{defaultSourcePattern},
		configKeyTimezone:    defaultTimezone,
		configKeyMetrics:     []string{},
		configKeyScanWorkers: 0,
	}
}

// configOf は設定を読み直して解釈する。
//
// SDKは config.json をプロセス起動時に一度しか読まないが、この設定は
// プラグインフォルダの config.json を編集して変えるものなので、
// 毎回読み直して gkill の再起動なしに反映されるようにする。
func configOf(pluginDir string, cfg sdk.Config) pluginConfig {
	latest := cfg
	if reloaded, err := sdk.LoadConfig(pluginDir); err == nil && len(reloaded) != 0 {
		latest = reloaded
	}
	if latest == nil {
		latest = sdk.Config{}
	}

	patterns := parseSourcePatterns(latest[configKeySourceDirs])
	timezone := defaultTimezone
	if value, ok := latest[configKeyTimezone].(string); ok && strings.TrimSpace(value) != "" {
		timezone = strings.TrimSpace(value)
	}

	return pluginConfig{
		Patterns:    patterns,
		Source:      sdk.ExpandSourcePatterns(patterns),
		Timezone:    timezone,
		Metrics:     parseStringList(latest[configKeyMetrics]),
		ScanWorkers: parseInt(latest[configKeyScanWorkers]),
	}
}

// parseStringList は配列でも改行区切りの文字列でも受け取る。
func parseStringList(value any) []string {
	values := []string{}
	add := func(s string) {
		s = strings.TrimSpace(strings.TrimSuffix(s, "\r"))
		if s != "" {
			values = append(values, s)
		}
	}
	switch v := value.(type) {
	case nil:
	case string:
		for line := range strings.SplitSeq(v, "\n") {
			add(line)
		}
	case []string:
		for _, s := range v {
			add(s)
		}
	case []any:
		for _, e := range v {
			if s, ok := e.(string); ok {
				add(s)
			}
		}
	}
	return values
}

// parseInt は JSON の数値（float64）でも文字列でも受け取る。
func parseInt(value any) int {
	switch v := value.(type) {
	case float64:
		return int(v)
	case int:
		return v
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return 0
		}
		return parsed
	}
	return 0
}

// splitSourceDirsForm は設定画面のテキストエリアを配列にする。
// ここでは ~ や環境変数を展開しない（読み出し時に展開する）。
func splitSourceDirsForm(value string) []string {
	values := []string{}
	for line := range strings.SplitSeq(value, "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if line != "" {
			values = append(values, line)
		}
	}
	return values
}
