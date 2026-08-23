package gkill_server_api

// /api/get_rep_infos_mcp の回帰テスト。
// rep_types の正準語彙がAPIから取得できること（外部監査 A1/A3）を固定する。

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/api/req_res"
)

func TestHandleGetRepInfosMCP(t *testing.T) {
	tsURL, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()

	sessionID := loginAndGetSession(t, tsURL, gkillAPI, "admin", mcpTestPasswordHash)

	resp := postJSON(t, tsURL+"/api/get_rep_infos_mcp", map[string]any{
		"session_id":  sessionID,
		"locale_name": "en",
	})
	defer resp.Body.Close()

	var infoResp req_res.GetRepInfosMCPResponse
	if err := json.NewDecoder(resp.Body).Decode(&infoResp); err != nil {
		t.Fatalf("decode get rep infos mcp response: %v", err)
	}
	if len(infoResp.Errors) > 0 {
		t.Fatalf("get rep infos mcp errors: %+v", infoResp.Errors)
	}

	// canonical_rep_types は find.KyouRepTypes と完全一致（クエリでそのまま使える正準値）
	if !slices.Equal(infoResp.CanonicalRepTypes, find.KyouRepTypes) {
		t.Errorf("canonical_rep_types = %v, want %v", infoResp.CanonicalRepTypes, find.KyouRepTypes)
	}

	if len(infoResp.RepInfos) == 0 {
		t.Fatal("rep_infos が空（テスト環境のrepが列挙されていない）")
	}
	canonical := map[string]bool{}
	for _, repType := range find.KyouRepTypes {
		canonical[repType] = true
	}
	seen := map[string]bool{}
	for _, info := range infoResp.RepInfos {
		if info.RepName == "" {
			t.Error("rep_name が空の行がある")
		}
		if !canonical[info.RepType] {
			t.Errorf("rep_type %q が正準語彙に無い", info.RepType)
		}
		// ファイルパスが漏れていない（区切り文字を含む名前が出たら実装がパスを返している）
		if strings.ContainsAny(info.RepName, `/\`) {
			t.Errorf("rep_name %q がパスに見える（leaf の GetRepName を通っていない）", info.RepName)
		}
		key := info.RepType + "\x00" + info.RepName
		if seen[key] {
			t.Errorf("(%s, %s) が重複している", info.RepType, info.RepName)
		}
		seen[key] = true
	}

	// kmemo の rep は必ずある（setupTestRouterWithRepos が作る）
	foundKmemo := false
	for _, info := range infoResp.RepInfos {
		if info.RepType == "kmemo" {
			foundKmemo = true
		}
	}
	if !foundKmemo {
		t.Errorf("kmemo の rep が列挙されていない: %+v", infoResp.RepInfos)
	}
}

// セッション無しでは使えない（wrapNoAuth だがハンドラ内でセッション解決する）。
func TestHandleGetRepInfosMCPRequiresSession(t *testing.T) {
	tsURL, _, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()

	resp := postJSON(t, tsURL+"/api/get_rep_infos_mcp", map[string]any{
		"session_id":  "invalid-session",
		"locale_name": "en",
	})
	defer resp.Body.Close()

	var infoResp req_res.GetRepInfosMCPResponse
	if err := json.NewDecoder(resp.Body).Decode(&infoResp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(infoResp.Errors) == 0 {
		t.Error("不正セッションなのにエラーが返っていない")
	}
	if len(infoResp.RepInfos) != 0 {
		t.Errorf("不正セッションなのにrep_infosが返っている: %d件", len(infoResp.RepInfos))
	}
}

// 2026-08-24 の再監査: タグ・テキストの書き込み先が
// get_all_rep_names にも rep_infos にも出ず、書く前には分からなかった。
// これらは Kyou を1件も生まないので Reps に居らず、原理的に rep_infos へは出てこない。
func TestHandleGetRepInfosMCPListsAttachedDataReps(t *testing.T) {
	tsURL, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()

	sessionID := loginAndGetSession(t, tsURL, gkillAPI, "admin", mcpTestPasswordHash)

	resp := postJSON(t, tsURL+"/api/get_rep_infos_mcp", map[string]any{
		"session_id":  sessionID,
		"locale_name": "en",
	})
	defer resp.Body.Close()

	var infoResp req_res.GetRepInfosMCPResponse
	if err := json.NewDecoder(resp.Body).Decode(&infoResp); err != nil {
		t.Fatalf("decode get rep infos mcp response: %v", err)
	}
	if len(infoResp.Errors) > 0 {
		t.Fatalf("get rep infos mcp errors: %+v", infoResp.Errors)
	}

	if len(infoResp.AttachedDataReps) == 0 {
		t.Fatal("attached_data_reps が空（タグ・テキストの書き込み先が分からないまま）")
	}

	validKinds := map[string]bool{"tag": true, "text": true, "notification": true, "gpslog": true}
	foundTag := false
	for _, attached := range infoResp.AttachedDataReps {
		if attached.RepName == "" {
			t.Error("rep_name が空の行がある")
		}
		if !validKinds[attached.DataKind] {
			t.Errorf("data_kind %q が想定外", attached.DataKind)
		}
		if strings.ContainsAny(attached.RepName, `/\`) {
			t.Errorf("rep_name %q がパスに見える", attached.RepName)
		}
		if attached.DataKind == "tag" {
			foundTag = true
		}
	}
	if !foundTag {
		t.Errorf("タグの格納先が出ていない: %+v", infoResp.AttachedDataReps)
	}

	// **rep_infos と混ざっていないこと。** 混ぜると呼び出し側が query.reps へ渡し、
	// Kyou の rep_name と一致しないので静かに0件になる
	kyouRepNames := map[string]bool{}
	for _, info := range infoResp.RepInfos {
		kyouRepNames[info.RepName] = true
	}
	for _, attached := range infoResp.AttachedDataReps {
		if kyouRepNames[attached.RepName] {
			t.Errorf("付随データのrep %q が rep_infos にも出ている（query.reps へ渡されて0件になる）", attached.RepName)
		}
	}
}
