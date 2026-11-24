package gkill_server_api

// 出力 HTML へプラグインの本文を焼き込む（出力時に選んだときだけ）。
//
// 本文はプラグインのプロセスへ問い合わせないと取れず、その問い合わせは
// **同じプラグインごとに直列化される**。件数を増やすと待ちが伸びるだけなので、
// 既定では焼き込まない。実行中の要求を中断してもいけない（プラグインのプロセスが殺される）。

import (
	"context"
	"fmt"

	"github.com/mt3hr/gkill/src/server/gkill/api/req_res"
)

// exportPluginHTMLContentLimit は1回の出力で問い合わせる上限。
// 超えたぶんは本文なしで出る（画面は出る。その記録の本文が空になるだけ）。
const exportPluginHTMLContentLimit = 200

// exportPluginHTMLKey は焼き込みの鍵。
//
// **クライアントの plugin_html_key と同じ形にすること**（gkill-api.ts）。
// 外れても例外は出ず、本文が引けずに空で出る。
func exportPluginHTMLKey(repName string, kyouID string) string {
	return repName + "\x00" + kyouID
}

// collectExportPluginHTML はプラグインの本文を焼き込む。
func (g *GkillServerAPI) collectExportPluginHTML(ctx context.Context, request *req_res.ExportKyouHTMLRequest, userID string, payload *exportedKyouPayload) []string {
	warnings := []string{}
	if !request.IncludePluginContent {
		return warnings
	}

	pluginManager := g.GkillDAOManager.GetPluginManager(userID)
	if pluginManager == nil {
		return warnings
	}

	asked := 0
	failed := 0
	dropped := 0
	for _, kyou := range payload.Kyous {
		key := exportPluginHTMLKey(kyou.RepName, kyou.ID)
		if _, already := payload.PluginHTML[key]; already {
			continue
		}
		pluginRepo := pluginManager.GetPluginByRepName(ctx, kyou.RepName)
		if pluginRepo == nil {
			// プラグイン由来でない記録。本文はもともと無い
			continue
		}
		if asked >= exportPluginHTMLContentLimit {
			dropped++
			continue
		}
		asked++

		// 本文の取得はハンドラと同じ経路を通る（キャッシュも共有する）。
		// ここで独自に問い合わせると、同じプラグインへ2系統から投げることになる
		cacheKey := pluginContentHTMLCacheKey(userID, kyou.RepName, kyou.ID)
		html, err := g.pluginContentHTMLCacheOf().GetOrFetch(ctx, cacheKey, fetchPluginContentHTML(pluginRepo, kyou.ID))
		if err != nil {
			// プラグインが落ちていることもある。画面は出るので警告に回す
			failed++
			continue
		}
		if html == "" {
			continue
		}
		payload.PluginHTML[key] = html
	}

	if dropped != 0 {
		warnings = append(warnings, fmt.Sprintf("プラグインの本文が多いため %d 件を含めませんでした", dropped))
	}
	if failed != 0 {
		warnings = append(warnings, fmt.Sprintf("%d 件のプラグインの本文を読み込めませんでした", failed))
	}
	return warnings
}
