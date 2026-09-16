package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	sdk "github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
)

//go:embed manifest.json
var manifestJSON []byte

func main() {
	// sdk.Run の flag.Parse は知らないフラグを受け取るとエラー終了するため先に処理する
	if slices.Contains(os.Args[1:], "--gkill-print-manifest") {
		if _, err := os.Stdout.Write(manifestJSON); err != nil {
			sdk.LogError("error at write manifest.json to stdout: %v", err)
			os.Exit(1)
		}
		return
	}
	if slices.Contains(os.Args[1:], "--gkill-print-config") {
		if err := printDefaultConfig(); err != nil {
			sdk.LogError("error at write default config.json to stdout: %v", err)
			os.Exit(1)
		}
		return
	}

	pluginDir := extractPluginDir(os.Args)

	sdk.Run(sdk.Handler{
		RepName:       repName,
		DefaultConfig: defaultConfig(),

		// 単独モード（gkill_server generate_plugin_cache）。常駐ビルダは起こさず同期で1周する。
		// build は取り込みを完走したあと走査時の問題（読めない ZIP 等）を返すので、それもそのまま
		// 返して exit 1 にする（静かに成功にすると壊れた ZIP に気づけない。キャッシュ自体は使える）。
		BuildCache: func(_ context.Context, cfg sdk.Config) error {
			return globalCache.build(pluginDir, configOf(pluginDir, cfg))
		},

		FindKyous: func(_ context.Context, q sdk.Query, cfg sdk.Config) ([]sdk.Kyou, error) {
			config := configOf(pluginDir, cfg)
			startBuilder(pluginDir, cfg)

			var startUnix, endUnix *int64
			if q.CalendarStartDate != nil {
				unix := q.CalendarStartDate.Unix()
				startUnix = &unix
			}
			if q.CalendarEndDate != nil {
				unix := q.CalendarEndDate.Unix()
				endUnix = &unix
			}

			// ワード判定は SDK に任せる（gkill 本体は再判定しないので、ここが唯一の判定）。
			// 単語で絞るなら LIMIT を SQL へ押し込まない。絞る前に切ると、後段のフィルタで落ちたぶん取りこぼす。
			matcher := q.Matcher()
			sqlLimit := q.Limit
			if matcher.HasWords() {
				sqlLimit = 0
			}
			metrics, err := globalCache.QueryDailyMetrics(pluginDir, config, startUnix, endUnix, sqlLimit)
			if err != nil {
				// 構築中や読めないときは空を返す。検索全体を落とさない
				sdk.LogError("gkill_plugin_fitbit: find_kyous: %v", err)
				return []sdk.Kyou{}, nil
			}

			return kyousOfMetrics(metrics, q), nil
		},

		// GetKyou は必ず実装する。
		// SDKの既定実装は FindKyous を全件やり直して線形探索するので、
		// 一覧の行数ぶん呼ばれると毎回数MBのJSONを組み立てることになる。
		GetKyou: func(_ context.Context, id string, cfg sdk.Config) (*sdk.Kyou, error) {
			startBuilder(pluginDir, cfg)
			metric, err := globalCache.QueryDailyMetric(pluginDir, id)
			if err != nil || metric == nil {
				return nil, err
			}
			kyou := kyouOf(*metric)
			return &kyou, nil
		},

		GetContentHTML: func(_ context.Context, kyouID string, cfg sdk.Config) (string, error) {
			startBuilder(pluginDir, cfg)
			metric, err := globalCache.QueryDailyMetric(pluginDir, kyouID)
			if err != nil || metric == nil {
				return renderNotFoundHTML(), nil
			}
			return renderMetricHTML(*metric), nil
		},

		GetConfigHTML: func(_ context.Context, cfg sdk.Config) (string, error) {
			config := configOf(pluginDir, cfg)
			startBuilder(pluginDir, cfg)
			// ここでZIPを開き直さない。ハンドラは数十ミリ秒で返す必要があり、
			// 3.7GBぶんの中央ディレクトリを読むと死活確認の期限(5秒)を割る。
			// 走査結果はビルダが cache_meta に書き残したものを読む。
			return renderConfigHTML(pluginDir, config, globalCache.Stats(pluginDir, config)), nil
		},

		PostConfig: func(_ context.Context, form map[string]string, cfg sdk.Config) (sdk.Config, error) {
			if cfg == nil {
				cfg = sdk.Config{}
			}
			if value, ok := form[configKeySourceDirs]; ok {
				cfg[configKeySourceDirs] = splitSourceDirsForm(value)
			}
			if value, ok := form[configKeyTimezone]; ok {
				timezone := strings.TrimSpace(value)
				if timezone == "" {
					timezone = defaultTimezone
				}
				if _, err := loadLocation(timezone); err != nil {
					return cfg, fmt.Errorf("タイムゾーン %q を読めません: %w", timezone, err)
				}
				cfg[configKeyTimezone] = timezone
			}
			if value, ok := form[configKeyMetrics]; ok {
				keys := splitSourceDirsForm(value)
				unknown := []string{}
				for _, key := range keys {
					if _, exist := metricByKey[key]; !exist {
						unknown = append(unknown, key)
					}
				}
				if len(unknown) != 0 {
					return cfg, fmt.Errorf("知らない指標キーです: %s", strings.Join(unknown, ", "))
				}
				cfg[configKeyMetrics] = keys
			}
			if value, ok := form[configKeyScanWorkers]; ok {
				workers, err := strconv.Atoi(strings.TrimSpace(value))
				if err == nil {
					cfg[configKeyScanWorkers] = min(max(workers, 0), 8)
				}
			}
			globalBuilder.Kick()
			return cfg, nil
		},
	})
}

// startBuilder はバックグラウンドのビルダを起動し、作り直しを促す。
// ハンドラは待たない。
func startBuilder(pluginDir string, cfg sdk.Config) {
	globalBuilder.EnsureStarted(pluginDir, func() pluginConfig { return configOf(pluginDir, cfg) })
}

// kyouOf は集計結果を Kyou に変換する。
func kyouOf(metric dailyMetric) sdk.Kyou {
	// 秒に切り捨てる。クライアントは型別データを秒精度で突き合わせるので、
	// ここに端数があるとKCが読み込まれなくなる。
	relatedTime := time.Unix(metric.RelatedUnix, 0).UTC()
	updateTime := time.Unix(metric.UpdateUnix, 0).UTC()
	if metric.UpdateUnix == 0 {
		updateTime = relatedTime
	}
	device := strings.Join(strings.Split(metric.Devices, "\n"), "/")

	return sdk.Kyou{
		ID:          metric.KyouID,
		RepName:     repName,
		DataType:    dataType,
		RelatedTime: relatedTime,
		CreateTime:  updateTime,
		UpdateTime:  updateTime,

		CreateApp:    appName,
		UpdateApp:    appName,
		CreateDevice: device,
		UpdateDevice: device,

		// タグはgkill側でタグ一覧に載るようになったので付けてよい。
		// 指標名を付けておくと、rykvのタグツリーから指標だけを選べる。
		Tags: []string{"fitbit", metric.Title},

		Typed: &sdk.TypedData{
			KC: &sdk.KC{
				Title:    metric.Title,
				NumValue: json.Number(metric.NumValue),
			},
		},
	}
}

// kyousOfMetrics は集計結果をワードで絞って Kyou にする。
//
// ワード判定は SDK に任せる（gkill 本体は再判定しないので、ここが唯一の判定）。
// 対象は search_text（指標名・キー・単位・数値・デバイス・日付）。LIMIT は絞った後に掛ける。
func kyousOfMetrics(metrics []dailyMetric, q sdk.Query) []sdk.Kyou {
	matcher := q.Matcher()
	kyous := make([]sdk.Kyou, 0, len(metrics))
	for _, metric := range metrics {
		if !matcher.MatchText(metric.SearchText, metric.KyouID) {
			continue
		}
		kyous = append(kyous, kyouOf(metric))
		if q.Limit > 0 && len(kyous) >= q.Limit {
			break
		}
	}
	return kyous
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
