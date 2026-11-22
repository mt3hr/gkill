// gkill_example は gkill プラグインSDKを使ったサンプル実装。
// 固定のKyouを2件返す。設定にメッセージを持ち、詳細HTMLにツイート風のカードを表示する。
//
// ビルド:
//
//	go build -o gkill_example .
//
// 配置先 ($GKILL_HOME/plugins/{userID}/gkill_example/):
//
//	manifest.json
//	gkill_example (or gkill_example.exe)
package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	sdk "github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
)

func main() {
	sdk.Run(sdk.Handler{
		RepName: "Example",

		FindKyous: func(ctx context.Context, q sdk.Query, cfg sdk.Config) ([]sdk.Kyou, error) {
			msg := cfg.Get("message", "Hello from gkill_example plugin!")

			now := time.Now()
			kyous := []sdk.Kyou{
				{
					ID:          "example-001",
					RepName:     "Example",
					DataType:    "example_kyou",
					RelatedTime: now.Add(-24 * time.Hour),
					CreateTime:  now.Add(-24 * time.Hour),
					UpdateTime:  now.Add(-24 * time.Hour),
					CreateApp:   "gkill_example",
					CreateUser:  "example",
					UpdateApp:   "gkill_example",
					UpdateUser:  "example",
					Tags:        []string{"サンプル"},
					Texts:       []string{msg + " (1件目)"},
				},
				{
					ID:          "example-002",
					RepName:     "Example",
					DataType:    "example_kyou",
					RelatedTime: now.Add(-1 * time.Hour),
					CreateTime:  now.Add(-1 * time.Hour),
					UpdateTime:  now.Add(-1 * time.Hour),
					CreateApp:   "gkill_example",
					CreateUser:  "example",
					UpdateApp:   "gkill_example",
					UpdateUser:  "example",
					Tags:        []string{"サンプル"},
					Texts:       []string{msg + " (2件目)"},
				},
			}

			// 日付フィルタ（CalendarStartTime / CalendarEndTime が指定されている場合）と
			// ワードフィルタ（Words / NotWords / WordsAnd）。
			// gkill 本体はプラグインが返した Kyou のワードを再判定しない（本文は本体に無い）ので、
			// ワード判定はプラグインの責任。自前のループは書かず、SDK の q.Matcher() / q.MatchText を通す。
			// 判定に掛ける文字列は「検索対象にしたいもの」を連結する（ここでは本文テキスト）。
			//
			// gkill が受けた検索条件の全体は q.FindQuery にある（古い gkill からは nil）。よく使うものは写してある:
			//   - q.IDs: ID の指定（nil は指定なし、[] は0件）。q.IDMatcher() で判定すると読む量を減らせる
			//   - q.Reps: rep 名の指定。複数の rep 名を返すプラグインが使う（q.MatchRep）
			//   - q.IncludeEndTimeIs: TimeIs を返すプラグインは、true のとき終了の Kyou（data_type "timeis_end"）も返す
			// gkill は返ってきた Kyou を ID・期間・IncludeEndTimeIs で絞り直すので、見なくても結果は変わらない。
			matcher := q.Matcher()
			idMatcher := q.IDMatcher()
			filtered := make([]sdk.Kyou, 0, len(kyous))
			for _, k := range kyous {
				if !idMatcher.Match(k.ID) {
					continue
				}
				if q.CalendarStartTime != nil && k.RelatedTime.Before(*q.CalendarStartTime) {
					continue
				}
				if q.CalendarEndTime != nil && k.RelatedTime.After(*q.CalendarEndTime) {
					continue
				}
				if !matcher.MatchText(strings.Join(k.Texts, "\n"), k.ID) {
					continue
				}
				filtered = append(filtered, k)
			}

			return filtered, nil
		},

		GetContentHTML: func(ctx context.Context, kyouID string, cfg sdk.Config) (string, error) {
			msg := cfg.Get("message", "Hello from gkill_example plugin!")
			html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<style>
  body { font-family: sans-serif; margin: 16px; }
  .card { border: 1px solid #ddd; border-radius: 8px; padding: 16px; max-width: 480px; }
  .id { color: #888; font-size: 0.8em; margin-bottom: 8px; }
  .content { font-size: 1em; line-height: 1.5; }
  .tag { display: inline-block; background: #e0f0ff; color: #0066cc;
         border-radius: 4px; padding: 2px 8px; font-size: 0.8em; margin-top: 8px; }
</style>
</head>
<body>
  <div class="card">
    <div class="id">ID: %s</div>
    <div class="content">%s</div>
    <div class="tag">サンプル</div>
  </div>
</body>
</html>`, kyouID, msg)
			return html, nil
		},

		GetConfigHTML: func(ctx context.Context, cfg sdk.Config) (string, error) {
			currentMsg := cfg.Get("message", "")
			html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<style>
  body { font-family: sans-serif; margin: 16px; }
  label { display: block; margin-bottom: 4px; font-weight: bold; }
  input[type=text] { width: 100%%; box-sizing: border-box; padding: 8px;
                     border: 1px solid #ccc; border-radius: 4px; }
  button { margin-top: 12px; padding: 8px 16px; background: #0066cc;
           color: white; border: none; border-radius: 4px; cursor: pointer; }
</style>
</head>
<body>
  <form method="POST" action="/api/post_plugin_config">
    <input type="hidden" name="rep_name" value="Example" />
    <label for="message">表示メッセージ</label>
    <input type="text" id="message" name="message" value="%s" placeholder="Hello from plugin!" />
    <button type="submit">保存</button>
  </form>
</body>
</html>`, currentMsg)
			return html, nil
		},
	})
}
