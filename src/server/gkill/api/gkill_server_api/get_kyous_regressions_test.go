package gkill_server_api

// FindQuery の Use* フラグ全廃（値が非nilならそのフィルタが有効、という意味論への移行）に
// ともなう /api/get_kyous の回帰テスト。
//
// ファイル名を handle_ で始めていないのは、資料の機械検査（verify_docs）が
// handle_*.go の本数を「ハンドラ数」として数えているため。
// get_device_cache_test.go が同じ理由で handle_ を外している。

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/api/gkill_plugin"
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/api/req_res"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
)

// regressionTestPasswordHash は空文字列のSHA-256。既存のハンドラテストと同じ値。
const regressionTestPasswordHash = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

// getKyousWithQuery は検索条件を渡して /api/get_kyous を叩く。
func getKyousWithQuery(t *testing.T, tsURL string, sessionID string, query *find.FindQuery) req_res.GetKyousResponse {
	t.Helper()

	resp := postJSON(t, tsURL+"/api/get_kyous", &req_res.GetKyousRequest{
		SessionID:  sessionID,
		LocaleName: "en",
		Query:      query,
	})
	defer resp.Body.Close()

	getResp := req_res.GetKyousResponse{}
	if err := json.NewDecoder(resp.Body).Decode(&getResp); err != nil {
		t.Fatalf("decode get kyous response: %v", err)
	}
	return getResp
}

// kyouIDSet は検索結果に含まれるKyouのIDの集合を返す。
func kyouIDSet(kyous []reps.Kyou) map[string]bool {
	ids := map[string]bool{}
	for _, kyou := range kyous {
		ids[kyou.ID] = true
	}
	return ids
}

// addTestKmemoWithRelatedTime は関連日時を指定してkmemoを1件足し、そのIDを返す。
// 既存の addTestKmemo は RelatedTime を設定しない（ゼロ値のまま）ので、
// 期間で絞り込む検索のテストではこちらを使う。
func addTestKmemoWithRelatedTime(t *testing.T, tsURL string, sessionID string, content string, relatedTime time.Time) string {
	t.Helper()

	kmemoID := GenerateNewID()
	resp := postJSON(t, tsURL+"/api/add_kmemo", &req_res.AddKmemoRequest{
		SessionID:  sessionID,
		LocaleName: "en",
		Kmemo: reps.Kmemo{
			ID:          kmemoID,
			Content:     content,
			RelatedTime: relatedTime,
			DataType:    "kmemo",
			CreateTime:  relatedTime,
			CreateApp:   "test",
			CreateUser:  "admin",
			UpdateTime:  relatedTime,
			UpdateApp:   "test",
			UpdateUser:  "admin",
		},
	})
	defer resp.Body.Close()

	addResp := req_res.AddKmemoResponse{}
	if err := json.NewDecoder(resp.Body).Decode(&addResp); err != nil {
		t.Fatalf("decode add kmemo response: %v", err)
	}
	if len(addResp.Errors) > 0 {
		t.Fatalf("add kmemo errors: %+v", addResp.Errors)
	}
	return kmemoID
}

// addTestTimeIsWithPeriod は期間を指定してTimeIsを1件足し、そのIDを返す。
// endTime が nil なら計測中（終了していない）TimeIsになる。
func addTestTimeIsWithPeriod(t *testing.T, tsURL string, sessionID string, title string, startTime time.Time, endTime *time.Time) string {
	t.Helper()

	timeisID := GenerateNewID()
	resp := postJSON(t, tsURL+"/api/add_timeis", &req_res.AddTimeIsRequest{
		SessionID:  sessionID,
		LocaleName: "en",
		TimeIs: reps.TimeIs{
			ID:         timeisID,
			Title:      title,
			StartTime:  startTime,
			EndTime:    endTime,
			DataType:   "timeis",
			CreateTime: startTime,
			CreateApp:  "test",
			CreateUser: "admin",
			UpdateTime: startTime,
			UpdateApp:  "test",
			UpdateUser: "admin",
		},
	})
	defer resp.Body.Close()

	addResp := req_res.AddTimeIsResponse{}
	if err := json.NewDecoder(resp.Body).Decode(&addResp); err != nil {
		t.Fatalf("decode add timeis response: %v", err)
	}
	if len(addResp.Errors) > 0 {
		t.Fatalf("add timeis errors: %+v", addResp.Errors)
	}
	return timeisID
}

// attachTestTagToTarget は targetID にタグを1件付ける。
func attachTestTagToTarget(t *testing.T, tsURL string, sessionID string, targetID string, tagName string, relatedTime time.Time) {
	t.Helper()

	resp := postJSON(t, tsURL+"/api/add_tag", &req_res.AddTagRequest{
		SessionID:  sessionID,
		LocaleName: "en",
		Tag: reps.Tag{
			ID:          GenerateNewID(),
			TargetID:    targetID,
			Tag:         tagName,
			RelatedTime: relatedTime,
			CreateTime:  relatedTime,
			CreateApp:   "test",
			CreateUser:  "admin",
			UpdateTime:  relatedTime,
			UpdateApp:   "test",
			UpdateUser:  "admin",
		},
	})
	defer resp.Body.Close()

	addResp := req_res.AddTagResponse{}
	if err := json.NewDecoder(resp.Body).Decode(&addResp); err != nil {
		t.Fatalf("decode add tag response: %v", err)
	}
	if len(addResp.Errors) > 0 {
		t.Fatalf("add tag errors: %+v", addResp.Errors)
	}
}

// writeUnstartablePluginManifest は、存在しない実行ファイルを指す manifest.json を
// {gkillHomeDir}/plugins/{userID}/{pluginName}/ へ置く。
// PluginManager はこのディレクトリを走査してプラグインを発見するので、
// 検索時に必ず起動失敗するプラグインを1つ用意できる。
func writeUnstartablePluginManifest(t *testing.T, gkillHomeDir string, userID string, pluginName string) {
	t.Helper()

	pluginDir := filepath.Join(gkillHomeDir, "plugins", userID, pluginName)
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatalf("failed to create plugin dir %s: %v", pluginDir, err)
	}

	manifestJSON, err := json.Marshal(gkill_plugin.PluginManifest{
		ProtocolVersion: "1",
		Name:            pluginName,
		Version:         "1.0.0",
		Description:     "起動に失敗するテスト用プラグイン",
		DataType:        pluginName + "_kyou",
		RepName:         pluginName,
		// 実在しない実行ファイル名。exec.Start が即座に失敗する
		Executable: "gkill_no_such_plugin_executable",
	})
	if err != nil {
		t.Fatalf("failed to marshal manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "manifest.json"), manifestJSON, 0o600); err != nil {
		t.Fatalf("failed to write manifest.json: %v", err)
	}
}

// プラグインの検索失敗は errors ではなく messages の警告として返ること。
//
// errors に載せるとハンドラもクライアントも検索全体を失敗扱いにして結果を捨てるため、
// プラグイン1つの障害で他のrepの検索結果まで消えてしまう。
// 一方で黙って落とすと「静かな欠落」になるので、MSG000088 の警告は必ず付く。
func TestHandleGetKyous_PluginFindFailureIsWarningNotError(t *testing.T) {
	// PluginManager はユーザ初回の GetRepositories で走査して結果をキャッシュするので、
	// manifest.json は最初のリクエストより前に置く必要がある。
	// GKILL_HOME を張らないと実運用のホーム（$HOME/gkill）のプラグインを走査してしまう。
	pluginHomeDir := t.TempDir()
	t.Setenv("GKILL_HOME", pluginHomeDir)
	writeUnstartablePluginManifest(t, pluginHomeDir, "admin", "unstartable_test_plugin")

	tsURL, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()

	sessionID := loginAndGetSession(t, tsURL, gkillAPI, "admin", regressionTestPasswordHash)

	now := time.Now().Truncate(time.Second)
	calendarStart := now.Add(-time.Hour)
	calendarEnd := now.Add(time.Hour)
	getResp := getKyousWithQuery(t, tsURL, sessionID, &find.FindQuery{
		CalendarStartDate: &calendarStart,
		CalendarEndDate:   &calendarEnd,
	})

	if len(getResp.Errors) != 0 {
		t.Fatalf("プラグイン検索の失敗を errors に載せてはいけない（クライアントが検索結果ごと捨てる）: %+v", getResp.Errors)
	}

	foundWarning := false
	for _, msg := range getResp.Messages {
		if msg.MessageCode == message.FindKyousPluginWarningMessage {
			foundWarning = true
			break
		}
	}
	if !foundWarning {
		t.Errorf("プラグイン検索の失敗が messages の警告(%s)として返っていない: %+v", message.FindKyousPluginWarningMessage, getResp.Messages)
	}
}

// plaing_time は *time.Time。null（未指定）なら実行中フィルタを掛けず、
// 非nilならその時刻を跨いでいる計測だけに絞る。
//
// Use* フラグ廃止前は use_plaing の真偽で切り替えていたため、
// 「plaing_time は送るがフィルタは使わない」という組み合わせがありえた。
// 現在は値の有無がそのままフィルタの有無になる。
func TestHandleGetKyous_PlaingTimeNullMeansNoPlaingFilter(t *testing.T) {
	tsURL, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()

	sessionID := loginAndGetSession(t, tsURL, gkillAPI, "admin", regressionTestPasswordHash)

	now := time.Now().Truncate(time.Second)
	endedEndTime := now.Add(-2 * time.Hour)
	endedTimeIsID := addTestTimeIsWithPeriod(t, tsURL, sessionID, "終了済みの計測", now.Add(-3*time.Hour), &endedEndTime)
	runningTimeIsID := addTestTimeIsWithPeriod(t, tsURL, sessionID, "計測中", now.Add(-time.Hour), nil)

	calendarStart := now.Add(-4 * time.Hour)
	calendarEnd := now.Add(time.Hour)

	// plaing_time 未指定（null）→ 実行中フィルタは掛からず両方返る
	withoutPlaing := getKyousWithQuery(t, tsURL, sessionID, &find.FindQuery{
		CalendarStartDate: &calendarStart,
		CalendarEndDate:   &calendarEnd,
	})
	if len(withoutPlaing.Errors) != 0 {
		t.Fatalf("get kyous errors: %+v", withoutPlaing.Errors)
	}
	idsWithoutPlaing := kyouIDSet(withoutPlaing.Kyous)
	if !idsWithoutPlaing[runningTimeIsID] {
		t.Error("plaing_time=null なのに計測中のTimeIsが返っていない")
	}
	if !idsWithoutPlaing[endedTimeIsID] {
		t.Error("plaing_time=null では実行中フィルタを掛けてはいけない（終了済みのTimeIsも返るべき）")
	}

	// plaing_time 指定（非nil）→ その時刻を跨いでいる計測だけ
	plaingTime := now.Add(-30 * time.Minute)
	withPlaing := getKyousWithQuery(t, tsURL, sessionID, &find.FindQuery{
		CalendarStartDate: &calendarStart,
		CalendarEndDate:   &calendarEnd,
		PlaingTime:        &plaingTime,
	})
	if len(withPlaing.Errors) != 0 {
		t.Fatalf("get kyous errors: %+v", withPlaing.Errors)
	}
	idsWithPlaing := kyouIDSet(withPlaing.Kyous)
	if !idsWithPlaing[runningTimeIsID] {
		t.Error("plaing_time を跨いでいる計測中のTimeIsが返っていない")
	}
	if idsWithPlaing[endedTimeIsID] {
		t.Error("plaing_time を跨いでいない終了済みのTimeIsが返っている")
	}
}

// TimeIsタグ検索は、Kyouのタグ絞り込み（tags）を使わなくても機能すること。
//
// タグ取得（getAllTags）の起動条件は
// `containsNoTags(Tags) || (HasTimeIsFilter() && containsNoTags(TimeIsTags))` になっている
// （RelatedTagIDs の読み手は NoTags 分岐しか無いので、それ以外では走らせない）。
// 以前は Kyouタグ側の条件だけを見ていたため、tags 未使用 + TimeIsタグ検索のとき
// 「どのTimeIsにタグが付いているか」の集合（RelatedTagIDs）が空のままになり、
// 全TimeIsが「タグなし」と判定されていた。
func TestHandleGetKyous_TimeIsTagsFilterWorksWithoutKyouTagFilter(t *testing.T) {
	tsURL, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()

	sessionID := loginAndGetSession(t, tsURL, gkillAPI, "admin", regressionTestPasswordHash)

	now := time.Now().Truncate(time.Second)

	// タグ付きTimeIsの期間と、その期間内のkmemo
	taggedStartTime := now.Add(-6 * time.Hour)
	taggedEndTime := now.Add(-5 * time.Hour)
	taggedTimeIsID := addTestTimeIsWithPeriod(t, tsURL, sessionID, "タグ付きの計測", taggedStartTime, &taggedEndTime)
	attachTestTagToTarget(t, tsURL, sessionID, taggedTimeIsID, "shigoto", taggedStartTime)
	kmemoInTaggedPeriodID := addTestKmemoWithRelatedTime(t, tsURL, sessionID, "タグ付き期間のメモ", taggedStartTime.Add(30*time.Minute))

	// タグ無しTimeIsの期間と、その期間内のkmemo
	noTagStartTime := now.Add(-3 * time.Hour)
	noTagEndTime := now.Add(-2 * time.Hour)
	addTestTimeIsWithPeriod(t, tsURL, sessionID, "タグ無しの計測", noTagStartTime, &noTagEndTime)
	kmemoInNoTagPeriodID := addTestKmemoWithRelatedTime(t, tsURL, sessionID, "タグ無し期間のメモ", noTagStartTime.Add(30*time.Minute))

	calendarStart := now.Add(-8 * time.Hour)
	calendarEnd := now.Add(time.Hour)
	getResp := getKyousWithQuery(t, tsURL, sessionID, &find.FindQuery{
		CalendarStartDate: &calendarStart,
		CalendarEndDate:   &calendarEnd,
		// Kyouのタグでは絞らない（nil = 未使用）
		Tags: nil,
		// 非nilの空スライス = 「任意のTimeIsに覆われたKyou」
		TimeIsWords: []string{},
		// "no tags" は「タグが1つも付いていない」を表す仮想タグ（api.NoTags）
		TimeIsTags: []string{"no tags"},
	})
	if len(getResp.Errors) != 0 {
		t.Fatalf("get kyous errors: %+v", getResp.Errors)
	}

	ids := kyouIDSet(getResp.Kyous)
	if !ids[kmemoInNoTagPeriodID] {
		t.Error("タグ無しTimeIsの期間内のKyouが返っていない")
	}
	if ids[kmemoInTaggedPeriodID] {
		t.Error("タグ付きTimeIsの期間内のKyouが「タグなし」扱いで返っている（全TimeIsがタグなし判定になっている）")
	}
}
