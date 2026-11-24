package gkill_server_api

import (
	"context"
	"strings"
	"testing"

	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/api/req_res"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
)

// TestCollectExportPluginHTML_SkipsWhenDisabledOrNoPlugins は、
// IncludePluginContent が false のときや対象プラグインが存在しないときに
// プラグイン本文を焼き込まず警告も出さないことを確認する。
func TestCollectExportPluginHTML_SkipsWhenDisabledOrNoPlugins(t *testing.T) {
	_, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("IncludePluginContent=false は何もしない", func(t *testing.T) {
		payload := minimalExportPayload("plugin-disabled")
		payload.Kyous = []reps.Kyou{{ID: "k1", RepName: "SomePlugin"}}
		warnings := gkillAPI.collectExportPluginHTML(ctx, &req_res.ExportKyouHTMLRequest{
			IncludePluginContent: false,
		}, "admin", payload)
		if len(warnings) != 0 {
			t.Errorf("unexpected warnings: %v", warnings)
		}
		if len(payload.PluginHTML) != 0 {
			t.Errorf("IncludePluginContent=false なのに PluginHTML が埋まっている: %+v", payload.PluginHTML)
		}
	})

	t.Run("通常 rep の Kyou や既に焼き込み済みの鍵はスキップされる", func(t *testing.T) {
		payload := minimalExportPayload("plugin-non-plugin-rep")
		payload.Kyous = []reps.Kyou{
			{ID: "k1", RepName: "Kmemo_Test"},
			{ID: "k2", RepName: "AlreadyBaked"},
		}
		payload.PluginHTML[exportPluginHTMLKey("AlreadyBaked", "k2")] = "<p>existing</p>"
		warnings := gkillAPI.collectExportPluginHTML(ctx, &req_res.ExportKyouHTMLRequest{
			IncludePluginContent: true,
		}, "admin", payload)
		if len(warnings) != 0 {
			t.Errorf("unexpected warnings: %v", warnings)
		}
		if got := payload.PluginHTML[exportPluginHTMLKey("AlreadyBaked", "k2")]; got != "<p>existing</p>" {
			t.Errorf("既存の焼き込みが上書きされた: %q", got)
		}
		if _, exist := payload.PluginHTML[exportPluginHTMLKey("Kmemo_Test", "k1")]; exist {
			t.Error("プラグイン由来でない記録に PluginHTML が入っている")
		}
	})
}

// TestCollectExportPluginHTML_AsksOnlyPluginKindKyous は、本文をプラグインへ問い合わせるのが
// プラグイン固有の記録（data_type が組み込みでないもの）だけであることと、
// 問い合わせの失敗が warning のメッセージ（MSG000096）で返ることを確認する。
//
// provides に kc・timeis などを宣言したプラグインの記録は native の型別ビューで描かれ、本文を引かない。
// そこへ問い合わせると直列のスロットを無駄に使い、SDK の既定の本文（ID だけ）を焼き込むことになる。
func TestCollectExportPluginHTML_AsksOnlyPluginKindKyous(t *testing.T) {
	// PluginManager は利用者の初回の GetRepositories で走査するので、manifest は先に置く
	pluginHomeDir := t.TempDir()
	t.Setenv("GKILL_HOME", pluginHomeDir)
	const pluginName = "unstartable_export_plugin"
	writeUnstartablePluginManifest(t, pluginHomeDir, "admin", pluginName)

	_, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()
	ctx := context.Background()
	device, err := gkillAPI.GetDevice()
	if err != nil {
		t.Fatalf("GetDevice failed: %v", err)
	}
	if _, err := gkillAPI.GkillDAOManager.GetRepositories("admin", device); err != nil {
		t.Fatalf("GetRepositories failed: %v", err)
	}
	if gkillAPI.GkillDAOManager.GetPluginManager("admin").GetPluginByRepName(ctx, pluginName) == nil {
		t.Fatalf("テスト用プラグイン %s が読み込まれていない", pluginName)
	}

	t.Run("型別の種類の記録には問い合わせない", func(t *testing.T) {
		payload := minimalExportPayload("plugin-typed")
		payload.Kyous = []reps.Kyou{
			{ID: "typed-timeis", RepName: pluginName, DataType: "timeis_start"},
			{ID: "typed-kc", RepName: pluginName, DataType: "kc"},
			{ID: "typed-mirekyou", RepName: pluginName, DataType: "mirekyou_check"},
		}
		warnings := gkillAPI.collectExportPluginHTML(ctx, &req_res.ExportKyouHTMLRequest{
			LocaleName:           "ja",
			IncludePluginContent: true,
		}, "admin", payload)
		if len(warnings) != 0 {
			t.Errorf("問い合わせないはずの記録で失敗の警告が出ている（問い合わせている）: %+v", warnings)
		}
	})

	t.Run("プラグイン固有の記録は問い合わせ、失敗は warning で数を返す", func(t *testing.T) {
		payload := minimalExportPayload("plugin-plain")
		payload.Kyous = []reps.Kyou{
			{ID: "plain-1", RepName: pluginName, DataType: pluginName + "_kyou"},
			{ID: "plain-2", RepName: pluginName, DataType: pluginName + "_kyou"},
		}
		warnings := gkillAPI.collectExportPluginHTML(ctx, &req_res.ExportKyouHTMLRequest{
			LocaleName:           "ja",
			IncludePluginContent: true,
		}, "admin", payload)
		if len(warnings) != 1 {
			t.Fatalf("warnings = %+v, want 1件（読み込めなかった数）", warnings)
		}
		if warnings[0].MessageCode != message.ExportKyouHTMLPartialWarningMessage || warnings[0].Level != message.MessageLevelWarning {
			t.Errorf("warning = %+v, want %s / %s", warnings[0], message.ExportKyouHTMLPartialWarningMessage, message.MessageLevelWarning)
		}
		if warnings[0].Message != "2 件のプラグインの本文を読み込めませんでした" {
			t.Errorf("warning message = %q", warnings[0].Message)
		}
		if len(payload.PluginHTML) != 0 {
			t.Errorf("読めなかった本文が焼かれている: %+v", payload.PluginHTML)
		}
	})
}

// TestIsBuiltinKyouDataType は射影名（mi_check / timeis_end / mirekyou_limit）も組み込みとして扱うことを確認する。
// mirekyou_ を mi_ より先に判定しないと、リポストタスクが Mi として扱われる（payloadKindOfDataType と同じ順序の罠）。
func TestIsBuiltinKyouDataType(t *testing.T) {
	for _, dataType := range []string{"kmemo", "kc", "urlog", "nlog", "lantana", "rekyou", "idf", "git_commit_log",
		"mi_create", "mi_check", "timeis_start", "timeis_end", "mirekyou_limit"} {
		if !isBuiltinKyouDataType(dataType) {
			t.Errorf("%s が組み込みとして扱われない", dataType)
		}
	}
	for _, dataType := range []string{"claude_conversation", "vrchat_world", ""} {
		if isBuiltinKyouDataType(dataType) {
			t.Errorf("%s が組み込みとして扱われる（プラグインの本文を引かなくなる）", dataType)
		}
	}
}

// TestExportWarningMessage_PlaceholdersInEveryLocale は、出力の警告文の差し込み（`{count}` など）が
// 7言語すべてで同じ名前で書かれていて、exportWarningMessage が置き換え残しを出さないことを確認する。
// 1言語だけ名前を訳してしまうと、その言語でだけ `{count}` が文字のまま利用者に見える。
func TestExportWarningMessage_PlaceholdersInEveryLocale(t *testing.T) {
	values := map[string]map[string]int{
		"EXPORT_KYOU_HTML_MEDIA_DROPPED_WARNING_MESSAGE":          {"count": 3, "item_limit_mb": 50, "budget_mb": 500},
		"EXPORT_KYOU_HTML_TEXT_DROPPED_WARNING_MESSAGE":           {"count": 3},
		"EXPORT_KYOU_HTML_MEDIA_FAILED_WARNING_MESSAGE":           {"count": 3},
		"EXPORT_KYOU_HTML_PLUGIN_CONTENT_DROPPED_WARNING_MESSAGE": {"count": 3},
		"EXPORT_KYOU_HTML_PLUGIN_CONTENT_FAILED_WARNING_MESSAGE":  {"count": 3},
	}
	for _, localeName := range []string{"ja", "en", "zh", "ko", "es", "fr", "de"} {
		for messageID, templateValues := range values {
			got := exportWarningMessage(localeName, messageID, templateValues)
			if strings.ContainsAny(got.Message, "{}") {
				t.Errorf("%s %s: 置き換え残しがある: %q", localeName, messageID, got.Message)
			}
			if !strings.Contains(got.Message, "3") {
				t.Errorf("%s %s: 件数が入っていない: %q", localeName, messageID, got.Message)
			}
		}
	}
}
