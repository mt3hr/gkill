package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"

	sdk "github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
)

//go:embed manifest.json
var manifestJSON []byte

// stderrWriter はログの出力先。
// os.Stdout には絶対に書かない。あれはプロトコルのチャネルで、
// 1行でも混ざるとJSONストリームが壊れる。
var stderrWriter io.Writer = os.Stderr

// pluginConfig は設定を解釈した結果。
type pluginConfig struct {
	Patterns          []string
	Source            sdk.ExpandedSource
	AccuracyMaxMeters int
	Sources           []string
	IncludeFitbitGPS  bool
	VisitPoints       bool
	MaxPoints         int
}

func main() {
	// sdk.Run の flag.Parse は知らないフラグを受け取るとエラー終了するため先に処理する
	if slices.Contains(os.Args[1:], "--gkill-print-manifest") {
		if _, err := os.Stdout.Write(manifestJSON); err != nil {
			os.Exit(1)
		}
		return
	}
	if slices.Contains(os.Args[1:], "--gkill-print-config") {
		if err := printDefaultConfig(); err != nil {
			os.Exit(1)
		}
		return
	}

	pluginDir := extractPluginDir(os.Args)

	sdk.Run(sdk.Handler{
		RepName:       repName,
		DefaultConfig: defaultConfig(),

		// 位置情報だけを提供するプラグインなので Kyou は出さない。
		//
		// FindKyous を nil にすると find_kyous が「未実装」エラーになり、
		// gkill側が検索のたびに警告を積む。空を返す実装を必ず置く。
		FindKyous: func(context.Context, sdk.Query, sdk.Config) ([]sdk.Kyou, error) {
			return []sdk.Kyou{}, nil
		},

		GetGPSLogs: func(_ context.Context, q sdk.GPSLogQuery, cfg sdk.Config) (sdk.GPSLogPage, error) {
			return globalCache.GetGPSLogs(pluginDir, configOf(pluginDir, cfg), q)
		},

		GetConfigHTML: func(_ context.Context, cfg sdk.Config) (string, error) {
			config := configOf(pluginDir, cfg)
			// ここで走査を待たない。ハンドラは数十ミリ秒で返す必要があり、
			// 3.7GBぶんの中央ディレクトリを読むと死活確認の期限(5秒)を割る。
			globalCache.kickRefresh(pluginDir, config)
			return renderConfigHTML(pluginDir, config, globalCache.Stats(pluginDir, config)), nil
		},

		PostConfig: func(_ context.Context, form map[string]string, cfg sdk.Config) (sdk.Config, error) {
			if cfg == nil {
				cfg = sdk.Config{}
			}
			if value, ok := form[configKeySourceDirs]; ok {
				cfg[configKeySourceDirs] = splitLinesForm(value)
			}
			if value, ok := form[configKeyAccuracyMaxMeters]; ok {
				if parsed, err := strconv.Atoi(strings.TrimSpace(value)); err == nil {
					cfg[configKeyAccuracyMaxMeters] = parsed
				}
			}
			if value, ok := form[configKeySources]; ok {
				sources := []string{}
				for part := range strings.SplitSeq(value, ",") {
					part = strings.ToUpper(strings.TrimSpace(part))
					if part != "" {
						sources = append(sources, part)
					}
				}
				cfg[configKeySources] = sources
			}
			if value, ok := form[configKeyIncludeFitbitGPS]; ok {
				cfg[configKeyIncludeFitbitGPS] = value == "1"
			}
			if value, ok := form[configKeyVisitPoints]; ok {
				cfg[configKeyVisitPoints] = value == "1"
			}
			if value, ok := form[configKeyMaxPoints]; ok {
				if parsed, err := strconv.Atoi(strings.TrimSpace(value)); err == nil && parsed > 0 {
					cfg[configKeyMaxPoints] = parsed
				}
			}
			return cfg, nil
		},
	})
}

// defaultConfig は config.json が無いときに書き出す既定設定。
func defaultConfig() sdk.Config {
	return sdk.Config{
		configKeyComment: "source_dirs に Google Takeout の ZIP を置いたフォルダを書きます。" +
			"ZIP は展開せずそのまま置いてください(展開したフォルダは読みません)。" +
			"フォルダを指定すると、その下の *.zip を再帰的に探し、" +
			"ZIP の中の タイムライン/ や Google Health/ を自動で見つけます。" +
			"* ** ? [] のワイルドカード、先頭の ~ と環境変数($HOME など)が使えます。" +
			"分割された takeout-....-001.zip / -002.zip はそのまま並べて置けます。" +
			"書き出しが複数あっても、同じ時刻・同じ座標の点は1つにまとめるので二重になりません。" +
			"accuracy_max_meters より精度の粗い測位は捨てます(既定100m。0以下で無効)。" +
			"精度が分からない点は残します。" +
			"sources を空にすると測位の出所で絞りません。" +
			"include_fitbit_gps はワークアウトのトラックを含めるかどうかです。" +
			"visit_points は滞在地と移動区間の端点も点として出すかどうかで、" +
			"既定は false です(生の測位より桁違いに粗いため)。" +
			"編集は次の取得から反映されます(gkill の再起動は不要)。" +
			"_ で始まるキーは説明用なので消して構いません。",
		configKeyExampleSourceDirs: []string{
			"~/Kyou/GoogleTakeout_*",
			"~/Downloads/takeout-20260808T230152Z-1-001.zip",
			"D:/backup/GoogleTakeout_*",
		},
		configKeySourceDirs:        []string{defaultSourcePattern},
		configKeyAccuracyMaxMeters: defaultAccuracyMaxMeters,
		configKeySources:           []string{},
		configKeyIncludeFitbitGPS:  true,
		configKeyVisitPoints:       false,
		configKeyMaxPoints:         defaultMaxPoints,
	}
}

// configOf は設定を読み直して解釈する。
// SDKは config.json をプロセス起動時に一度しか読まないので、毎回読み直す。
func configOf(pluginDir string, cfg sdk.Config) pluginConfig {
	latest := cfg
	if reloaded, err := sdk.LoadConfig(pluginDir); err == nil && len(reloaded) != 0 {
		latest = reloaded
	}
	if latest == nil {
		latest = sdk.Config{}
	}

	patterns := parseSourcePatterns(latest[configKeySourceDirs])
	return pluginConfig{
		Patterns:          patterns,
		Source:            sdk.ExpandSourcePatterns(patterns),
		AccuracyMaxMeters: parseIntOr(latest[configKeyAccuracyMaxMeters], defaultAccuracyMaxMeters),
		Sources:           parseStringList(latest[configKeySources]),
		IncludeFitbitGPS:  parseBoolOr(latest[configKeyIncludeFitbitGPS], true),
		VisitPoints:       parseBoolOr(latest[configKeyVisitPoints], false),
		MaxPoints:         parseIntOr(latest[configKeyMaxPoints], defaultMaxPoints),
	}
}

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
		for part := range strings.SplitSeq(v, ",") {
			add(part)
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

func parseIntOr(value any, fallback int) int {
	switch v := value.(type) {
	case float64:
		return int(v)
	case int:
		return v
	case string:
		if parsed, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			return parsed
		}
	}
	return fallback
}

func parseBoolOr(value any, fallback bool) bool {
	switch v := value.(type) {
	case bool:
		return v
	case string:
		return v == "1" || strings.EqualFold(v, "true")
	}
	return fallback
}

// splitLinesForm は設定画面のテキストエリアを配列にする。
func splitLinesForm(value string) []string {
	values := []string{}
	for line := range strings.SplitSeq(value, "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if line != "" {
			values = append(values, line)
		}
	}
	return values
}

// extractPluginDir は起動引数から --gkill-plugin-dir を取り出す。
func extractPluginDir(args []string) string {
	for i, arg := range args {
		if arg == "--gkill-plugin-dir" && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}

// printDefaultConfig は既定の config.json を標準出力に書く。
func printDefaultConfig() error {
	encoded, err := json.MarshalIndent(defaultConfig(), "", "    ")
	if err != nil {
		return err
	}
	_, err = os.Stdout.Write(append(encoded, '\n'))
	return err
}

// sampleConfigJSON は設定画面に載せる既定の config.json。
func sampleConfigJSON() string {
	encoded, err := json.MarshalIndent(defaultConfig(), "", "    ")
	if err != nil {
		return "{}"
	}
	return string(encoded)
}
