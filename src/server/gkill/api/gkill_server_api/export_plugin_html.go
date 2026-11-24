package gkill_server_api

// 出力 HTML へプラグインの本文を焼き込む（出力時に選んだときだけ）。
//
// 本文はプラグインのプロセスへ問い合わせないと取れず、その問い合わせは
// **同じプラグインごとに直列化される**。件数を増やすと待ちが伸びるだけなので、
// 既定では焼き込まない。実行中の要求を中断してもいけない（プラグインのプロセスが殺される）。

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/mt3hr/gkill/src/server/gkill/api"
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/api/req_res"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	"github.com/nicksnyder/go-i18n/v2/i18n"
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
func (g *GkillServerAPI) collectExportPluginHTML(ctx context.Context, request *req_res.ExportKyouHTMLRequest, userID string, payload *exportedKyouPayload) []*message.GkillMessage {
	warnings := []*message.GkillMessage{}
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
		// 型別の種類（provides に kc・timeis などを宣言したプラグインの記録）は native の型別ビューで描かれ、
		// 本文を引かない（クライアントは data_type が組み込みでない記録にだけ本文を出す）。
		// 問い合わせると直列のスロットを無駄に使い、SDK の既定の本文（ID だけ）を焼き込むことになる
		if isBuiltinKyouDataType(kyou.DataType) {
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
			slog.Log(ctx, gkill_log.Warn, "error at get plugin content html for export", "rep_name", kyou.RepName, "id", kyou.ID, "error", fmt.Sprintf("%q", err))
			failed++
			continue
		}
		if html == "" {
			continue
		}
		payload.PluginHTML[key] = html
	}

	if dropped != 0 {
		warnings = append(warnings, exportWarningMessage(request.LocaleName, "EXPORT_KYOU_HTML_PLUGIN_CONTENT_DROPPED_WARNING_MESSAGE", map[string]int{"count": dropped}))
	}
	if failed != 0 {
		warnings = append(warnings, exportWarningMessage(request.LocaleName, "EXPORT_KYOU_HTML_PLUGIN_CONTENT_FAILED_WARNING_MESSAGE", map[string]int{"count": failed}))
	}
	return warnings
}

// isBuiltinKyouDataType は data_type が組み込みの種類（射影名を含む）かを返す。
// 組み込みでないものがプラグイン固有の記録で、本文はプラグインに問い合わせて得る。
func isBuiltinKyouDataType(dataType string) bool {
	switch payloadKindOfDataType(dataType) {
	case "kmemo", "kc", "urlog", "nlog", "lantana", "rekyou", "idf", "git_commit_log", "mi", "timeis", "mirekyou":
		return true
	}
	return false
}

// exportWarningMessage は出力の一部を落としたことの警告を作る。
// 成功しているので errors ではなく warning のメッセージで返す（FindKyousPluginWarningMessage と同じ扱い）。
//
// 文言の差し込みは `{count}` の形（vue-i18n と同じ）で書き、ここで置き換える。
// locales はクライアントと共有なので、go-i18n のテンプレート（`{{.Count}}`）は書かない。
func exportWarningMessage(localeName string, messageID string, values map[string]int) *message.GkillMessage {
	text := api.GetLocalizer(localeName).MustLocalizeMessage(&i18n.Message{ID: messageID})
	for name, value := range values {
		text = strings.ReplaceAll(text, "{"+name+"}", strconv.Itoa(value))
	}
	return &message.GkillMessage{
		MessageCode: message.ExportKyouHTMLPartialWarningMessage,
		Message:     text,
		Level:       message.MessageLevelWarning,
	}
}
