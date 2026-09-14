package main

import (
	"context"
	_ "embed"
	"os"
	"slices"
	"sync/atomic"

	sdk "github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
)

// manifest.json をバイナリに埋め込み、--gkill-print-manifest で出力できるようにする。
// 配置スクリプトが manifest.json を用意できるようにするため。
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

	// 既定の config.json の出力だけして終わる。配置スクリプトが用意できるようにするため。
	// 通常はプラグイン起動時に自動生成されるので、これを使う必要はない。
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

		// 記録が名乗る rep 名はリポジトリ名。gkill はこれを rep 一覧・rep 絞り込み・本文取得の
		// 引き当てに使う。まだ1件も無ければ空スライス（nil だと「未対応」と読まれ manifest 名に落ちる）。
		RepNames: func(_ context.Context, cfg sdk.Config) ([]string, error) {
			rememberConfig(cfg)
			startBuilder(pluginDir, provider)

			names, err := globalCache.RepNames(pluginDir)
			if err != nil {
				sdk.LogError("%s: rep names: %v", appName, err)
				return []string{}, nil
			}
			return names, nil
		},

		// ハンドラは全部「ビルダを起こして、今キャッシュにあるぶんを即返す」。
		// 初回が空なのは仕様。取り込みはバックグラウンドで進む。
		FindKyous: func(_ context.Context, q sdk.Query, cfg sdk.Config) ([]sdk.Kyou, error) {
			rememberConfig(cfg)
			startBuilder(pluginDir, provider)

			// 単語で絞るなら LIMIT を SQL へ押し込まない。
			// 絞る前に切ると、後段のフィルタで落ちたぶん取りこぼす。
			matcher := q.Matcher()
			rows, err := globalCache.QueryCommits(pluginDir, q.CalendarStartDate, q.CalendarEndDate, q.Limit, matcher.HasWords())
			if err != nil {
				sdk.LogError("%s: find kyous: %v", appName, err)
				return []sdk.Kyou{}, nil
			}
			return kyousOfRows(rows, q), nil
		},

		// SDKの既定実装は FindKyous を全件やり直して線形探索するので必ず自前で持つ。
		GetKyou: func(_ context.Context, id string, cfg sdk.Config) (*sdk.Kyou, error) {
			rememberConfig(cfg)
			startBuilder(pluginDir, provider)

			row, err := globalCache.QueryCommit(pluginDir, id)
			if err != nil {
				return nil, nil
			}
			kyou := kyouOf(row)
			return &kyou, nil
		},

		GetContentHTML: func(_ context.Context, kyouID string, cfg sdk.Config) (string, error) {
			rememberConfig(cfg)
			startBuilder(pluginDir, provider)

			body, err := globalCache.QueryBody(pluginDir, kyouID)
			if err != nil {
				return renderNotFoundHTML(), nil
			}
			return renderCommitHTML(body), nil
		},

		// ここで zip を開いてはいけない。IsAlive(5秒)と同じスロットに並ぶ。
		GetConfigHTML: func(_ context.Context, cfg sdk.Config) (string, error) {
			rememberConfig(cfg)
			startBuilder(pluginDir, provider)

			config := configOf(pluginDir, cfg)
			stats := globalCache.Stats(pluginDir)
			return renderConfigHTML(pluginDir, stats, config.Patterns), nil
		},

		PostConfig: func(_ context.Context, form map[string]string, cfg sdk.Config) (sdk.Config, error) {
			if cfg == nil {
				cfg = sdk.Config{}
			}
			if v, ok := form[configKeySourceDirs]; ok {
				// 設定画面のテキストエリアは1行1指定。config.json には配列で書き戻す
				// (1行の文字列でも ParseSourcePatterns は読めるが、配列のほうが手で編集しやすい)。
				cfg[configKeySourceDirs] = splitSourceDirsForm(v)
			}
			rememberConfig(cfg)
			// 対象から外れた zip のコミットは、次の走査で「消えたリポジトリ」として
			// 自動的にキャッシュから削除される（他の zip にも入っていれば残る）
			globalBuilder.Kick()
			return cfg, nil
		},
	})
}

// kyousOfRows はキャッシュの行をワードで絞って Kyou にする。
//
// ワード判定は SDK に任せる（gkill 本体は再判定しないので、ここが唯一の判定）。
// 対象はコミットメッセージ・リポジトリ名・author 名。ID（ハッシュ）の前方一致は SDK が見る。
// LIMIT は絞った後に掛ける。
func kyousOfRows(rows []commitRow, q sdk.Query) []sdk.Kyou {
	matcher := q.Matcher()
	kyous := make([]sdk.Kyou, 0, len(rows))
	for _, row := range rows {
		if !matcher.MatchText(searchTextOf(row), row.Hash) {
			continue
		}
		kyous = append(kyous, kyouOf(row))
		if q.Limit > 0 && len(kyous) >= q.Limit {
			break
		}
	}
	return kyous
}

// searchTextOf はワード検索の照合対象。
func searchTextOf(row commitRow) string {
	return row.Message + "\n" + row.RepName + "\n" + row.AuthorName
}

// kyouOf はキャッシュの1行を gkill へ返す Kyou にする。
//
// 列の取り方は native の git rep と同じ: ID はコミットハッシュ、rep 名はリポジトリ名、
// 時刻はすべてコミッタ日時（コミット固有のゾーン付き）、利用者名は author 名、アプリ名は "git"。
// 型別データを載せるので、gkill 側の GitCommitLog アダプタが native と同じ列で返せる。
func kyouOf(row commitRow) sdk.Kyou {
	committedAt := row.committedAt()
	return sdk.Kyou{
		ID:          row.Hash,
		RepName:     row.RepName,
		DataType:    dataType,
		RelatedTime: committedAt,
		CreateTime:  committedAt,
		CreateApp:   gitAppName,
		CreateUser:  row.AuthorName,
		UpdateTime:  committedAt,
		UpdateApp:   gitAppName,
		UpdateUser:  row.AuthorName,
		Typed: &sdk.TypedData{GitCommitLog: &sdk.GitCommitLog{
			CommitMessage: row.Message,
			Addition:      row.Addition,
			Deletion:      row.Deletion,
		}},
	}
}
