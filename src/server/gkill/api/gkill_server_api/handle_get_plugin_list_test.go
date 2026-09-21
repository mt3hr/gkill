package gkill_server_api

// /api/get_plugin_list の回帰テスト。
//
// 指摘 D1「is_alive=true なのに0件、の理由がAPIから診断できない」への対応で、
// provides 宣言のあるプラグインには型別索引の統計（typed_index）を返すようになった。
// さらに 2巡目の指摘で、OK=false だけでは「一度も構築していない」と
// 「構築に失敗した」が潰れて直しようが無かったため State / LastBuildError /
// LastAttemptAt が足された。この写し替えは handle_get_plugin_list.go にあり、
// 落としても（LastAttemptAt のゼロ値スキップを消しても）コンパイルは通るので、
// ハンドラ経由で3フィールドが載ることをここで固定する。

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/message"
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

// TestHandleGetPluginList_EmitsKyouAndProvides は、プラグインの「役割」が
// 一覧から判別できることを固定する。
//
// 2026-08-24 の実利用報告で、GPSログ専用プラグイン（emits_kyou=false）の
// rep_name / data_type が「検索に使える値」として提示されていたため、
// query.reps へ渡して0件になり、原因も分からない、という迷い方が実際に起きた。
// manifest には emits_kyou と provides があったのに、APIがどちらも返していなかった。
//
// 新しい語彙（capabilities 等）は作らず、manifest の宣言をそのまま外へ出す。
func TestHandleGetPluginList_EmitsKyouAndProvides(t *testing.T) {
	ts, gkillAPI, cleanup := setupTestRouter(t)
	defer cleanup()

	t.Setenv("GKILL_HOME", gkill_options.GkillHomeDir)

	// Kyouを出すプラグイン（emits_kyou 未指定 = true、provides 無し）
	writePluginManifestForTest(t, "admin", "kyou_only_plugin", map[string]any{
		"protocol_version": "1",
		"name":             "kyou_only_plugin",
		"version":          "1.0.0",
		"description":      "emits kyou only",
		"data_type":        "kyou_only_test",
		"rep_name":         "KyouOnlyTestRep",
		"executable":       "no_such_plugin_binary",
	})
	// GPSログ専用プラグイン（emits_kyou=false、provides=["gpslog"]）
	writePluginManifestForTest(t, "admin", "gpslog_only_plugin", map[string]any{
		"protocol_version": "1",
		"name":             "gpslog_only_plugin",
		"version":          "1.0.0",
		"description":      "emits gps logs only",
		"data_type":        "gpslog_only_test",
		"rep_name":         "GPSLogOnlyTestRep",
		"executable":       "no_such_plugin_binary",
		"provides":         []string{"gpslog"},
		"emits_kyou":       false,
	})

	passwordHash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	sessionID := loginAndGetSession(t, ts.URL, gkillAPI, "admin", passwordHash)

	listResp := getPluginList(t, ts.URL, sessionID)
	if len(listResp.Errors) > 0 {
		t.Fatalf("get plugin list errors: %+v", listResp.Errors)
	}

	kyouOnly := findPluginInfoByName(listResp.Plugins, "kyou_only_plugin")
	if kyouOnly == nil {
		t.Fatalf("Kyouを出すプラグインが一覧に出ていない: %+v", listResp.Plugins)
	}
	// emits_kyou は manifest 未指定なら true。ここが false に化けると
	// 通常のプラグインの記録が「検索できない」と案内されてしまう
	if !kyouOnly.EmitsKyou {
		t.Error("emits_kyou 未指定のプラグインが emits_kyou=false になっている")
	}
	if len(kyouOnly.Provides) != 0 {
		t.Errorf("provides 無しなのに provides が返っている: %v", kyouOnly.Provides)
	}
	// rep_names は「query.reps に渡せる値」として常に載る。申告しない（できない）プラグインは
	// manifest の rep_name 1つ（ADR-0311。以前は申告があるときだけ載り、索引未構築だとキーごと消えた）
	if len(kyouOnly.RepNames) != 1 || kyouOnly.RepNames[0] != "KyouOnlyTestRep" {
		t.Errorf("rep_names = %v, want [KyouOnlyTestRep]（申告が無ければ manifest の rep_name）", kyouOnly.RepNames)
	}

	gpsOnly := findPluginInfoByName(listResp.Plugins, "gpslog_only_plugin")
	if gpsOnly == nil {
		t.Fatalf("GPS専用プラグインが一覧に出ていない: %+v", listResp.Plugins)
	}
	if gpsOnly.EmitsKyou {
		t.Error("emits_kyou=false のプラグインが emits_kyou=true になっている")
	}
	if gpsOnly.RepNames != nil {
		t.Errorf("Kyou を出さないプラグインの rep_names = %v, want null（検索値は無い）", gpsOnly.RepNames)
	}
	if len(gpsOnly.Provides) != 1 || gpsOnly.Provides[0] != "gpslog" {
		t.Errorf("provides = %v, want [gpslog]", gpsOnly.Provides)
	}

	// providesがgpslogだけのプラグインには型別索引を作らない。
	// 索引の材料（型別データ・付随データ）が1件も無いので、作ると
	// state=never_built / record_count=0 のまま永久に固定され、
	// 「索引が壊れている＝位置情報が使えない」と誤読される。
	if gpsOnly.TypedIndex != nil {
		t.Errorf("GPS専用プラグインに typed_index が付いている: %+v", gpsOnly.TypedIndex)
	}
	// GPSを一度も取得していなければ gps_index も出ない。
	// 「壊れている」ではなく「まだ読んでいない」なので、ゼロ値を出してはいけない。
	if gpsOnly.GPSIndex != nil {
		t.Errorf("GPSを一度も取得していないのに gps_index が付いている: %+v", gpsOnly.GPSIndex)
	}

	// --- GPS取得後 ---
	// GPSLogRepositoryアダプタが統計を預けたあと、ハンドラがDTOへ写すことを固定する。
	// この写し替えは handle_get_plugin_list.go にあり、落としてもコンパイルは通る。
	pluginRepo := gkillAPI.GkillDAOManager.GetPluginManager("admin").GetPluginByName("gpslog_only_plugin")
	if pluginRepo == nil {
		t.Fatal("plugin manager からGPS専用プラグインが引けない")
	}
	sink, ok := pluginRepo.(interface {
		SetGPSIndexStats(stats reps.GPSIndexStats)
	})
	if !ok {
		t.Fatal("プラグインリポジトリがGPS統計を預かれない（アダプタからの受け口が消えている）")
	}
	oldest := time.Date(2026, 8, 23, 9, 0, 0, 0, time.Local)
	newest := time.Date(2026, 8, 24, 21, 30, 0, 0, time.Local)
	fetchedAt := time.Date(2026, 8, 24, 22, 0, 0, 0, time.Local)
	sink.SetGPSIndexStats(reps.GPSIndexStats{
		PointCount: 2509,
		Oldest:     oldest,
		Newest:     newest,
		FetchedAt:  fetchedAt,
	})

	afterResp := getPluginList(t, ts.URL, sessionID)
	afterInfo := findPluginInfoByName(afterResp.Plugins, "gpslog_only_plugin")
	if afterInfo == nil || afterInfo.GPSIndex == nil {
		t.Fatalf("GPS取得後も gps_index が返らない: %+v", afterInfo)
	}
	if afterInfo.GPSIndex.PointCount != 2509 {
		t.Errorf("point_count = %d, want 2509", afterInfo.GPSIndex.PointCount)
	}
	if afterInfo.GPSIndex.Oldest != oldest.Format(time.RFC3339) {
		t.Errorf("oldest = %q, want %q", afterInfo.GPSIndex.Oldest, oldest.Format(time.RFC3339))
	}
	if afterInfo.GPSIndex.Newest != newest.Format(time.RFC3339) {
		t.Errorf("newest = %q, want %q", afterInfo.GPSIndex.Newest, newest.Format(time.RFC3339))
	}
	if afterInfo.GPSIndex.FetchedAt != fetchedAt.Format(time.RFC3339) {
		t.Errorf("fetched_at = %q, want %q", afterInfo.GPSIndex.FetchedAt, fetchedAt.Format(time.RFC3339))
	}
	// 型別索引は依然として付かない（別枠であることの確認）
	if afterInfo.TypedIndex != nil {
		t.Errorf("GPS取得後に typed_index が生えている: %+v", afterInfo.TypedIndex)
	}
}

// TestHandleGetPluginList_RedactsEnvironmentSpecificDiagnostics は、プラグインの診断文が
// 端末固有の情報を伏せてから返ることを固定する。
//
// last_error はプラグインプロセスの生stderr、last_build_error は起動失敗のエラーを
// 文字列化したもので、どちらにもプラグインディレクトリの絶対パスやプラグインが自分で
// 書いたホームディレクトリが乗る。これはMCP経由でAIへ渡り、AIが資料やコミットメッセージへ
// 引き写すと、verify_docs の checkPersonalInfo が防いでいる混入がそのまま成立する。
// プラグインは別リポジトリの成果物なので、書き手側の約束では止められない。
// 経緯: documents/adr/0707-redact-environment-specific-strings.md
func TestHandleGetPluginList_RedactsEnvironmentSpecificDiagnostics(t *testing.T) {
	ts, gkillAPI, cleanup := setupTestRouter(t)
	defer cleanup()

	// プラグインの探索基準は $GKILL_HOME（newPluginManager が env を優先する）。
	// 伏せる対象を必ず含ませるため、ホームの下に Users/username/ を挟む。
	// Windowsでは一時ディレクトリ自体がユーザープロファイル配下にあるので
	// そちらが対象になり、Linuxではこの Users/username/ が対象になる。
	// どちらでも「伏せるものが1つも無い」状態にならないことを、下の前提チェックで担保する。
	originalHome := gkill_options.GkillHomeDir
	gkill_options.GkillHomeDir = filepath.Join(originalHome, "Users", "username", "gkill")
	defer func() { gkill_options.GkillHomeDir = originalHome }()
	t.Setenv("GKILL_HOME", gkill_options.GkillHomeDir)

	writePluginManifestForTest(t, "admin", "redact_diag_plugin", map[string]any{
		"protocol_version": "1",
		"name":             "redact_diag_plugin",
		"version":          "1.0.0",
		"description":      "redaction test plugin",
		"data_type":        "redact_diag_test",
		"rep_name":         "RedactDiagTestRep",
		"executable":       "no_such_plugin_binary",
		"provides":         []string{"tag"},
	})

	passwordHash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	sessionID := loginAndGetSession(t, ts.URL, gkillAPI, "admin", passwordHash)

	// 実行ファイルが無いので起動に失敗し、stderrリングと索引の失敗理由の両方に
	// プラグインディレクトリの絶対パスが入る。
	pluginRepo := gkillAPI.GkillDAOManager.GetPluginManager("admin").GetPluginByName("redact_diag_plugin")
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

	if pluginRepo.LastStderr() == "" {
		t.Fatal("起動に失敗したのに stderr が空（テストの前提が崩れている）")
	}
	if typedIndex.Stats().LastBuildError == "" {
		t.Fatal("索引構築に失敗したのに last_build_error が空（テストの前提が崩れている）")
	}

	// 前提チェック。伏せる対象がフィクスチャに1つも無いと、
	// ハンドラから伏せる処理を消しても下のアサートが通ってしまう。
	//
	// 診断文そのもののスナップショットとは比べない。ハンドラは IsAlive で
	// 起動をやり直すので、そのたびに stderr リングへ1行増える。
	pluginHome := gkill_options.GkillHomeDir
	if message.RedactEnvironmentSpecific(pluginHome) == pluginHome {
		t.Fatalf("伏せる対象がホームのパスに含まれていない（テストが素通しになる）: %q", pluginHome)
	}

	listResp := getPluginList(t, ts.URL, sessionID)
	if len(listResp.Errors) > 0 {
		t.Fatalf("get plugin list errors: %+v", listResp.Errors)
	}
	info := findPluginInfoByName(listResp.Plugins, "redact_diag_plugin")
	if info == nil {
		t.Fatalf("テストプラグインが一覧に出ていない: %+v", listResp.Plugins)
	}

	if strings.Contains(info.LastError, pluginHome) {
		t.Errorf("last_error に端末のパスがそのまま残っている: %q", info.LastError)
	}
	if !strings.Contains(info.LastError, "〈ユーザー名〉") {
		t.Errorf("last_error が伏せられていない: %q", info.LastError)
	}
	if info.TypedIndex == nil {
		t.Fatal("provides ありなのに typed_index が省略されている")
	}
	if strings.Contains(info.TypedIndex.LastBuildError, pluginHome) {
		t.Errorf("last_build_error に端末のパスがそのまま残っている: %q", info.TypedIndex.LastBuildError)
	}
	if !strings.Contains(info.TypedIndex.LastBuildError, "〈ユーザー名〉") {
		t.Errorf("last_build_error が伏せられていない: %q", info.TypedIndex.LastBuildError)
	}
	// 形は残す。ここまで潰すと「どこを読みに行って失敗したか」が診断できなくなる。
	if !strings.Contains(info.LastError, "failed to start plugin") {
		t.Errorf("診断として意味のある部分まで消えている: %q", info.LastError)
	}
}
