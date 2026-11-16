package main

import (
	"context"
	_ "embed"
	"os"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	sdk "github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
)

// manifest.json をバイナリに埋め込み、--gkill-print-manifest で出力できるようにする。
// バイナリと manifest が必ず一致するので、別々に配る必要がない。
//
//go:embed manifest.json
var manifestJSON []byte

// latestConfig はハンドラが受け取った最新の設定。ビルダから読む。
var latestConfig atomic.Pointer[sdk.Config]

func rememberConfig(cfg sdk.Config) {
	if cfg == nil {
		return
	}
	latestConfig.Store(&cfg)
}

// configProviderOf はビルダに渡す「今の設定を返す関数」を作る。
func configProviderOf(pluginDir string) func() pluginConfig {
	return func() pluginConfig {
		var base sdk.Config
		if stored := latestConfig.Load(); stored != nil {
			base = *stored
		}
		return configOf(pluginDir, base)
	}
}

func extractPluginDir(args []string) string {
	for i, arg := range args {
		if arg == "--gkill-plugin-dir" && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}

func main() {
	// manifest.json の出力だけして終わる。sdk.Run より前に処理する
	// (sdk.Run の flag.Parse は知らないフラグを受け取るとエラー終了するため)
	if slices.Contains(os.Args[1:], "--gkill-print-manifest") {
		if _, err := os.Stdout.Write(manifestJSON); err != nil {
			sdk.LogError("error at write manifest.json to stdout: %v", err)
			os.Exit(1)
		}
		return
	}

	// 既定の config.json の出力だけして終わる。通常はプラグイン起動時に自動生成される。
	if slices.Contains(os.Args[1:], "--gkill-print-config") {
		if err := printDefaultConfig(); err != nil {
			sdk.LogError("error at write default config.json to stdout: %v", err)
			os.Exit(1)
		}
		return
	}

	pluginDir := extractPluginDir(os.Args)
	provider := configProviderOf(pluginDir)

	sdk.Run(sdk.Handler{
		RepName:       repName,
		DefaultConfig: defaultConfig(),

		// 単独モード（gkill_server generate_plugin_cache）。常駐ビルダは起こさず同期で1周する。
		BuildCache: func(_ context.Context, cfg sdk.Config) error {
			return buildOnce(pluginDir, configOf(pluginDir, cfg))
		},

		// ハンドラは全部「ビルダを起こして、今キャッシュにあるぶんを即返す」。
		// 初回が空なのは仕様。取り込みはバックグラウンドで進む。
		FindKyous: func(_ context.Context, q sdk.Query, cfg sdk.Config) ([]sdk.Kyou, error) {
			rememberConfig(cfg)
			startBuilder(pluginDir, provider)

			// 単語で絞るなら LIMIT を SQL へ押し込まない。
			// 絞る前に切ると、後段のフィルタで落ちたぶん取りこぼす。
			matcher := q.Matcher()
			rows, err := globalCache.QueryKyous(pluginDir, q.CalendarStartDate, q.CalendarEndDate, q.Limit, matcher.HasWords())
			if err != nil {
				sdk.LogError("%s: find kyous: %v", appName, err)
				return []sdk.Kyou{}, nil
			}
			return kyousOfRows(rows, q), nil
		},

		// SDKの既定実装は FindKyous を全件やり直して線形探索するので必ず自前で持つ。
		// Kyou ごとの data_type（timeis_start / urlog / kmemo）をそのまま返す。
		// manifest の data_type で上書きするとクライアントの型別ビューが入れ替わる（ADR-0317）。
		GetKyou: func(_ context.Context, id string, cfg sdk.Config) (*sdk.Kyou, error) {
			rememberConfig(cfg)
			startBuilder(pluginDir, provider)

			row, err := globalCache.QueryKyou(pluginDir, id)
			if err != nil {
				return nil, nil
			}
			kyou := kyouOf(row)
			return &kyou, nil
		},

		// 本文 HTML（GetContentHTML）は持たない。記録はどれも TimeIs / URLog / Kmemo で、
		// クライアントは native と同じ型別ビューで描く。

		// ここでファイルを走査してはいけない。IsAlive(5秒)と同じスロットに並ぶ。
		GetConfigHTML: func(_ context.Context, cfg sdk.Config) (string, error) {
			rememberConfig(cfg)
			startBuilder(pluginDir, provider)

			config := configOf(pluginDir, cfg)
			stats := globalCache.Stats(pluginDir)
			return renderConfigHTML(pluginDir, stats, config.Patterns, expandPatterns(config.Patterns)), nil
		},

		PostConfig: func(_ context.Context, form map[string]string, cfg sdk.Config) (sdk.Config, error) {
			if cfg == nil {
				cfg = sdk.Config{}
			}
			if v, ok := form[configKeySourceDirs]; ok {
				// 設定画面のテキストエリアは1行1指定。config.json には配列で書き戻す
				cfg[configKeySourceDirs] = splitSourceDirsForm(v)
			}
			rememberConfig(cfg)
			// 対象から外れたファイルは、次の走査で「消えたファイル」として
			// 自動的にキャッシュから削除される
			globalBuilder.Kick()
			return cfg, nil
		},
	})
}

// kyousOfRows はキャッシュの行をワードで絞って Kyou にする。
//
// ワード判定は SDK に任せる（gkill 本体は再判定しないので、ここが唯一の判定）。
// 対象は search_text（タイトル・ワールド名・ワールド ID・インスタンス・usr ID・本文）。
// LIMIT は絞った後に掛ける。
func kyousOfRows(rows []recordRow, q sdk.Query) []sdk.Kyou {
	matcher := q.Matcher()
	kyous := make([]sdk.Kyou, 0, len(rows))
	for _, row := range rows {
		if !matcher.MatchText(row.SearchText, row.ID) {
			continue
		}
		kyous = append(kyous, kyouOf(row))
		if q.Limit > 0 && len(kyous) >= q.Limit {
			break
		}
	}
	return kyous
}

// kyouOf はキャッシュの1行を gkill へ返す Kyou にする。
//
// 時刻は秒で持っている（ログの時刻が秒まで）。クライアントは型別データを
// Kyou の UpdateTime と秒精度で突き合わせるので、端数を作らないこと。
func kyouOf(row recordRow) sdk.Kyou {
	relatedTime := unixToTime(row.RelatedUnix)
	updateTime := unixToTime(row.UpdateUnix)
	if updateTime.IsZero() {
		updateTime = relatedTime
	}
	kyou := sdk.Kyou{
		ID:          row.ID,
		RepName:     repName,
		DataType:    kindDataType(row.Kind),
		RelatedTime: relatedTime,
		CreateTime:  relatedTime,
		UpdateTime:  updateTime,
		CreateApp:   appName,
		UpdateApp:   appName,
	}
	if row.Tags != "" {
		kyou.Tags = strings.Split(row.Tags, "\n")
	}

	switch kyou.DataType {
	case dataTypeURLog:
		kyou.Typed = &sdk.TypedData{URLog: &sdk.URLog{URL: row.URL, Title: row.Title, Description: row.Description}}
	case dataTypeKmemo:
		kyou.Typed = &sdk.TypedData{Kmemo: &sdk.Kmemo{Content: row.Content}}
	default:
		// 終了時刻は必ず埋める（実行中の TimeIs は出さない）。終了なしにすると、
		// 画面の「終了」ボタンで native の版が書かれてしまう（ADR-0317）。
		end := unixToTime(row.EndUnix)
		if end.IsZero() {
			end = unixToTime(row.StartUnix)
		}
		kyou.Typed = &sdk.TypedData{TimeIs: &sdk.TimeIs{Title: row.Title, StartTime: unixToTime(row.StartUnix), EndTime: &end}}
	}
	return kyou
}

// unixToTime はUNIX秒を時刻にする。0はゼロ値のまま返す。
func unixToTime(unix int64) time.Time {
	if unix == 0 {
		return time.Time{}
	}
	return time.Unix(unix, 0).UTC()
}
