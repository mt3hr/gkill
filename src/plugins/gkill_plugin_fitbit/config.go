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

	// SecondaryDataSources は「時計の行が無い日にだけ使う」データソース名（大小無視）。
	// 空なら全データソースを合算する（2026-09-20 以前の挙動）。既定は defaultSecondaryDataSources。
	SecondaryDataSources []string
}

// defaultSecondaryDataSources は既定の補助データソース。
//
// Takeout の CSV では 2025-12 からスマホの歩数計（Phone Health Connect）の行が時計の行と
// 同じ日に並ぶ。Fitbit アプリ自身の日計は時計の値だけを採り、スマホの行は足さない
// （実データの混在する全日で一致）。アプリ本体（Google Health App）の行も、時計と同じ日に
// あるときは端数（距離で数 m）で、アプリの日計には入っていない。
func defaultSecondaryDataSources() []string {
	return []string{"Phone Health Connect", "Google Health App"}
}

// foldRule は畳み直しの規則を1つの文字列にしたもの。cache_meta に控えて、
// 変わったら全日を畳み直させる（取り込み直しは要らない）。
func (c pluginConfig) foldRule() string {
	normalized := make([]string, 0, len(c.SecondaryDataSources))
	for _, source := range c.SecondaryDataSources {
		normalized = append(normalized, normalizeDataSource(source))
	}
	return "secondary=" + strings.Join(normalized, "\n")
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
			"secondary_data_sources は「時計の行が無い日にだけ使う」データソース名です" +
			"(CSV の data source 列の値。大小無視)。" +
			"Takeout の歩数 CSV にはスマホ(Phone Health Connect)と時計(Pixel Watch 2 など)の行が" +
			"同じ日に並ぶことがあり、両方を足すと歩数が2倍になります。ここに書いたソースは" +
			"時計の行がある日には使わず、無い日にだけ書いた順で採ります。空にすると全部を合算します。" +
			"scan_workers は同時に読むファイル数で、0 なら自動。" +
			"編集は次の検索から反映されます(gkill の再起動は不要)。" +
			"_ で始まるキーは説明用なので消して構いません。",
		configKeyExampleSourceDirs: []string{
			"~/Kyou/GoogleTakeout_*",
			"~/Downloads/takeout-20260808T230152Z-1-001.zip",
			"D:/backup/GoogleTakeout_*",
		},
		configKeySourceDirs:           []string{defaultSourcePattern},
		configKeyTimezone:             defaultTimezone,
		configKeyMetrics:              []string{},
		configKeySecondaryDataSources: defaultSecondaryDataSources(),
		configKeyScanWorkers:          0,
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

	// キーが無い（この設定を知らない古い config.json）なら既定、空配列なら「全部合算」。
	// nil と [] を区別するのは、既存の config.json を自動生成し直さないため。
	secondary := defaultSecondaryDataSources()
	if value, present := latest[configKeySecondaryDataSources]; present {
		secondary = parseStringList(value)
	}

	return pluginConfig{
		Patterns:             patterns,
		Source:               sdk.ExpandSourcePatterns(patterns),
		Timezone:             timezone,
		Metrics:              parseStringList(latest[configKeyMetrics]),
		ScanWorkers:          parseInt(latest[configKeyScanWorkers]),
		SecondaryDataSources: secondary,
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
