package gkill_server_api

// /api/get_plugin_list の回帰テスト。
//
// 外部監査 D1「is_alive=true なのに0件、の理由がAPIから診断できない」への対応で、
// provides 宣言のあるプラグインには型別索引の統計（typed_index）を返すようになった。
// さらに 2026-08-24 の再監査で、OK=false だけでは「一度も構築していない」と
// 「構築に失敗した」が潰れて直しようが無かったため State / LastBuildError /
// LastAttemptAt が足された。この写し替えは handle_get_plugin_list.go にあり、
// 落としても（LastAttemptAt のゼロ値スキップを消しても）コンパイルは通るので、
// ハンドラ経由で3フィールドが載ることをここで固定する。

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/req_res"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_options"
)

// writePluginManifestForTest は $GKILL_HOME/plugins/{userID}/{pluginName}/ に
// manifest.json だけを配置する。実行ファイルは意図的に置かない ——
// プロセス起動が即座に失敗するので、索引構築の失敗経路を決定的に踏める
// （実プラグインを起動すると環境依存のフレークになる）。
func writePluginManifestForTest(t *testing.T, userID string, pluginName string, manifest map[string]any) {
	t.Helper()

	pluginDir := filepath.Join(gkill_options.GkillHomeDir, "plugins", userID, pluginName)
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatalf("create plugin dir: %v", err)
	}
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "manifest.json"), data, 0o644); err != nil {
		t.Fatalf("write manifest.json: %v", err)
	}
}

// getPluginList は /api/get_plugin_list を叩いてレスポンスを返す。
func getPluginList(t *testing.T, tsURL string, sessionID string) req_res.GetPluginListResponse {
	t.Helper()

	req := &req_res.GetPluginListRequest{
		SessionID:  sessionID,
		LocaleName: "en",
	}
	resp := postJSON(t, tsURL+"/api/get_plugin_list", req)
	defer resp.Body.Close()

	var listResp req_res.GetPluginListResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		t.Fatalf("decode get plugin list response: %v", err)
	}
	return listResp
}

// findPluginInfoByName はプラグイン一覧から名前で1件探す。無ければnil。
func findPluginInfoByName(plugins []req_res.PluginInfo, name string) *req_res.PluginInfo {
	for i := range plugins {
		if plugins[i].Name == name {
			return &plugins[i]
		}
	}
	return nil
}

// TestHandleGetPluginList_TypedIndexStats は typed_index の
// State / LastBuildError / LastAttemptAt がハンドラ経由で載ることを固定する。
//
//   - 未構築: state=never_built、last_build_error 無し、
//     last_attempt_at はゼロ値スキップで**省略**される
//     （一度も試していないのに時刻が出ると、呼び出し側が
//     「試行済みなのに何も起きていない」と誤読する）
//   - 構築失敗後: state=failed、last_build_error に理由、
//     last_attempt_at に試行時刻（RFC3339）
//
// provides の無いプラグインでは typed_index 自体が省略されることも併せて固定する。
func TestHandleGetPluginList_TypedIndexStats(t *testing.T) {
	ts, gkillAPI, cleanup := setupTestRouter(t)
	defer cleanup()

	// プラグイン発見は $GKILL_HOME/plugins/{userID}/ を見る。
	// 実環境で GKILL_HOME が設定されていると本物のホームを読み書きしてしまうので、
	// テスト用の一時ホーム（setupTestGkillServerAPI が差し替え済み）に固定する。
	t.Setenv("GKILL_HOME", gkill_options.GkillHomeDir)

	// provides 宣言あり（typed_index が返る）。実行ファイルは無い。
	writePluginManifestForTest(t, "admin", "typed_stats_plugin", map[string]any{
		"protocol_version": "1",
		"name":             "typed_stats_plugin",
		"version":          "1.0.0",
		"description":      "typed index stats test plugin",
		"data_type":        "typed_stats_test",
		"rep_name":         "TypedStatsTestRep",
		"executable":       "no_such_plugin_binary",
		"provides":         []string{"tag"},
	})
	// provides 宣言なし（typed_index は省略される）
	writePluginManifestForTest(t, "admin", "no_provides_plugin", map[string]any{
		"protocol_version": "1",
		"name":             "no_provides_plugin",
		"version":          "1.0.0",
		"description":      "plugin without provides",
		"data_type":        "no_provides_test",
		"rep_name":         "NoProvidesTestRep",
		"executable":       "no_such_plugin_binary",
	})

	passwordHash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	sessionID := loginAndGetSession(t, ts.URL, gkillAPI, "admin", passwordHash)

	// --- 未構築（get_plugin_list は wrapAuth なので GetRepositories が走らず、
	// 索引構築はまだ一度も試みられていない） ---
	listResp := getPluginList(t, ts.URL, sessionID)
	if len(listResp.Errors) > 0 {
		t.Fatalf("get plugin list errors: %+v", listResp.Errors)
	}

	if info := findPluginInfoByName(listResp.Plugins, "no_provides_plugin"); info == nil {
		t.Error("provides 無しのプラグインが一覧に出ていない")
	} else if info.TypedIndex != nil {
		t.Errorf("provides 無しなのに typed_index が返っている: %+v", info.TypedIndex)
	}

	info := findPluginInfoByName(listResp.Plugins, "typed_stats_plugin")
	if info == nil {
		t.Fatalf("provides ありのプラグインが一覧に出ていない: %+v", listResp.Plugins)
	}
	if info.TypedIndex == nil {
		t.Fatal("provides ありなのに typed_index が省略されている")
	}
	if info.TypedIndex.OK {
		t.Error("未構築なのに ok=true")
	}
	if info.TypedIndex.State != reps.PluginTypedIndexStateNeverBuilt {
		t.Errorf("state = %q, want %q", info.TypedIndex.State, reps.PluginTypedIndexStateNeverBuilt)
	}
	if info.TypedIndex.LastBuildError != "" {
		t.Errorf("一度も構築していないのに last_build_error が出ている: %q", info.TypedIndex.LastBuildError)
	}
	// ゼロ値スキップ: 一度も試していなければ last_attempt_at は省略される。
	// スキップを消すと "0001-01-01T00:00:00Z" が出て「試行済み」に見えてしまう。
	if info.TypedIndex.LastAttemptAt != "" {
		t.Errorf("一度も構築を試みていないのに last_attempt_at が出ている: %q", info.TypedIndex.LastAttemptAt)
	}

	// --- 構築失敗後 ---
	// 実行ファイルが無いプラグインへ同期リビルドをかけ、失敗を確定させる。
	pluginRepo := gkillAPI.GkillDAOManager.GetPluginManager("admin").GetPluginByName("typed_stats_plugin")
	if pluginRepo == nil {
		t.Fatal("plugin manager からテストプラグインが引けない")
	}
	typedIndex := pluginRepo.TypedIndex()
	if typedIndex == nil {
		t.Fatal("typed index が作られていない")
	}
	if err := typedIndex.Refresh(context.Background()); err == nil {
		t.Fatal("実行ファイルが無いのに索引構築が成功している（テストの前提が崩れている）")
	}

	failedResp := getPluginList(t, ts.URL, sessionID)
	if len(failedResp.Errors) > 0 {
		t.Fatalf("get plugin list errors: %+v", failedResp.Errors)
	}
	failedInfo := findPluginInfoByName(failedResp.Plugins, "typed_stats_plugin")
	if failedInfo == nil || failedInfo.TypedIndex == nil {
		t.Fatal("構築失敗後のプラグインが一覧に出ていない")
	}
	if failedInfo.TypedIndex.OK {
		t.Error("構築に失敗したのに ok=true")
	}
	if failedInfo.TypedIndex.State != reps.PluginTypedIndexStateFailed {
		t.Errorf("state = %q, want %q（never_built と failed の区別が潰れている）", failedInfo.TypedIndex.State, reps.PluginTypedIndexStateFailed)
	}
	if failedInfo.TypedIndex.LastBuildError == "" {
		t.Error("構築に失敗したのに last_build_error が空（失敗理由がAPIから読めない）")
	}
	if failedInfo.TypedIndex.LastAttemptAt == "" {
		t.Error("構築を試みたのに last_attempt_at が空")
	} else if _, err := time.Parse(time.RFC3339, failedInfo.TypedIndex.LastAttemptAt); err != nil {
		t.Errorf("last_attempt_at %q が RFC3339 として読めない: %v", failedInfo.TypedIndex.LastAttemptAt, err)
	}
}
