package gkill_server_api

// /api/get_rep_infos_mcp の回帰テスト。
// rep_types の正準語彙がAPIから取得できること（外部監査 A1/A3）を固定する。

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/api/req_res"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_options"
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

	// use_to_write: 歴代端末ぶん並ぶ格納先のうち書き込み先を1行で引けること
	// （「gkill_add_tag はどこへ書くのか」への答え。2026-09-18 の実利用報告）。
	// テスト環境の rep は書き込み先として作られるので、種別ごとに最低1つは立つ。
	writableByKind := map[string]int{}
	for _, attached := range infoResp.AttachedDataReps {
		if attached.UseToWrite {
			writableByKind[attached.DataKind]++
		}
	}
	for _, kind := range []string{"tag", "text"} {
		if writableByKind[kind] == 0 {
			t.Errorf("%s の格納先に use_to_write:true の行が無い: %+v", kind, infoResp.AttachedDataReps)
		}
	}
	// Kyou rep 側も同じ集合で判定する（rep_infos[].use_to_write は以前は無検査だった）
	writableKyouReps := 0
	for _, info := range infoResp.RepInfos {
		if info.UseToWrite {
			writableKyouReps++
		}
	}
	if writableKyouReps == 0 {
		t.Errorf("rep_infos に use_to_write:true の行が無い: %+v", infoResp.RepInfos)
	}

	// **別々のフィールドで返すこと。** 同じ配列へ混ぜると、呼び出し側が
	// 付随データのrep名を query.reps へ渡し、Kyou の rep_name と一致せず静かに0件になる。
	//
	// 名前が両方に出ること自体は正当なので禁止しない —— provides を持つプラグインは
	// Kyou（kc など）と付随データ（tag など）の両方を供給するので、同じ rep 名が
	// どちらの一覧にも載る（2026-08-24 のデプロイ後に本番で実測）。その rep は
	// 本当に Kyou rep でもあるので query.reps へ渡しても正しく効く。
	// 守るべきなのは「配列が別で、それぞれの区別が付くこと」だけ。
	for _, attached := range infoResp.AttachedDataReps {
		if attached.DataKind == "" {
			t.Errorf("付随データのrep %q に data_kind が無い（rep_infos と同じ形になっている）", attached.RepName)
		}
	}
	for _, info := range infoResp.RepInfos {
		if info.RepType == "" {
			t.Errorf("Kyou rep %q に rep_type が無い", info.RepName)
		}
	}
}

// 2026-08-24 の再監査: rep ディレクトリへ置いただけのファイルは UpdateCache が
// IDF() を走らせるまで検索に出ないのに、定期実行も監視も警告も無く、
// 「0件」が取り込み待ちなのか本当に無いのか区別できなかった。
// その判断材料が rep_infos[].indexed_at で、実装は「任意インタフェース
// IndexUpdatedAt を実装した leaf rep だけ」を型アサーションで拾う。
// アサーションはシグネチャがずれても**コンパイルエラーにならず**、
// indexed_at が黙って全行から消えるだけなので、ここで固定して回帰を検知する。
//
// キャッシュONでは IDFKyouReps がキャッシュrep1つに畳まれ、UnWrap で leaf に
// 降りてからアサーションする経路になる。OFFとは通り道が違うので両方で回す。
func TestHandleGetRepInfosMCPIncludesIndexedAt(t *testing.T) {
	for _, cacheInMemory := range []bool{false, true} {
		t.Run(fmt.Sprintf("cacheInMemory=%v", cacheInMemory), func(t *testing.T) {
			if cacheInMemory {
				useCacheInMemory(t)
			}
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

			// directory (= IDF) rep は setupTestRouterWithRepos が必ず1つ作る。
			// IDF rep の構築時に索引DB（gkill_id.db）が作られるので、
			// indexed_at はこの時点で必ず入る。
			foundDirectory := false
			for _, info := range infoResp.RepInfos {
				switch info.RepType {
				case "directory":
					foundDirectory = true
					if info.IndexedAt == "" {
						t.Errorf("directory rep %q の indexed_at が空（IndexUpdatedAt の型アサーションが外れている）", info.RepName)
						continue
					}
					if _, err := time.Parse(time.RFC3339, info.IndexedAt); err != nil {
						t.Errorf("indexed_at %q が RFC3339 として読めない: %v", info.IndexedAt, err)
					}
				case "kmemo":
					// 索引を持たない rep では省略が契約（DTOの omitempty に依存する
					// クライアントが「索引あり」と誤読しないように）
					if info.IndexedAt != "" {
						t.Errorf("索引を持たない kmemo rep %q に indexed_at が出ている: %q", info.RepName, info.IndexedAt)
					}
				}
			}
			if !foundDirectory {
				t.Fatalf("directory rep が列挙されていない: %+v", infoResp.RepInfos)
			}
		})
	}
}

// TestHandleGetRepInfosMCPExcludesNonKyouPluginsFromPlugins は、Kyouを1件も出さない
// プラグインが plugins[] に載らないことを固定する。
//
// plugins[] は「query.reps / data_types へ渡せる値」の対応表として説明されている。
// ところが以前は PluginReps を無条件に列挙していたため、GPSログ専用プラグインが
// **同じ応答の中で** plugins[]（渡せる）と attached_data_reps[]（渡してはいけない）の
// 両方に出ており、応答が自己矛盾していた。実利用のAIはこれを読んで
// query.reps へ rep_name を渡し、警告だけが返る結果になった（2026-08-24 の報告）。
//
// 役割ごとに1箇所へ決めるのが ADR-0607 の方針。GPS の供給元としては
// attached_data_reps[] の data_kind="gpslog" に残る。
func TestHandleGetRepInfosMCPExcludesNonKyouPluginsFromPlugins(t *testing.T) {
	tsURL, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()

	t.Setenv("GKILL_HOME", gkill_options.GkillHomeDir)

	// GPSログ専用プラグイン（emits_kyou=false）。実行ファイルは置かない。
	writePluginManifestForTest(t, "admin", "repinfo_gpslog_plugin", map[string]any{
		"protocol_version": "1",
		"name":             "repinfo_gpslog_plugin",
		"version":          "1.0.0",
		"description":      "gps only",
		"data_type":        "repinfo_gpslog_visit",
		"rep_name":         "RepInfoGPSLogRep",
		"executable":       "no_such_plugin_binary",
		"provides":         []string{"gpslog"},
		"emits_kyou":       false,
	})
	// Kyouを出すプラグイン（対照）。こちらは plugins[] に出る。
	writePluginManifestForTest(t, "admin", "repinfo_kyou_plugin", map[string]any{
		"protocol_version": "1",
		"name":             "repinfo_kyou_plugin",
		"version":          "1.0.0",
		"description":      "emits kyou",
		"data_type":        "repinfo_kyou_test",
		"rep_name":         "RepInfoKyouRep",
		"executable":       "no_such_plugin_binary",
	})

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

	for _, plugin := range infoResp.Plugins {
		if plugin.PluginName == "repinfo_gpslog_plugin" {
			t.Errorf("Kyouを出さないプラグインが plugins[] に居る（query.reps へ渡され静かに0件になる）: %+v", plugin)
		}
	}

	foundKyouPlugin := false
	for _, plugin := range infoResp.Plugins {
		if plugin.PluginName == "repinfo_kyou_plugin" {
			foundKyouPlugin = true
		}
	}
	if !foundKyouPlugin {
		t.Errorf("Kyouを出すプラグインまで plugins[] から消えている: %+v", infoResp.Plugins)
	}

	// GPS の供給元としては別枠に残っていること。両方から消すと
	// 「そのプラグインのデータをどこから読むのか」が分からなくなる。
	foundGPSSource := false
	for _, attached := range infoResp.AttachedDataReps {
		if attached.RepName == "RepInfoGPSLogRep" && attached.DataKind == "gpslog" {
			foundGPSSource = true
		}
	}
	if !foundGPSSource {
		t.Errorf("GPS専用プラグインが attached_data_reps[] にも居ない: %+v", infoResp.AttachedDataReps)
	}
}

// TestHandleGetRepInfosMCPAttachedDataRepNamesAreReal は attached_data_reps の
// rep 名が**実在する rep の名前**であることを固定する。
//
// キャッシュ有効時（既定）、tag / text の cached 実装の UnWrapTyped が1段しか剥がさず、
// 集約自身が leaf として返っていた。その GetRepName() は "TagReps" / "TextReps" という
// リテラルなので、**実在しない名前が「タグはどこへ書かれるか」の答えとして返っていた**
// （2026-08-24 の実利用レビュー。実際の書き込み先は "Tag" / "Text"）。
//
// 同じ注意は GetLatestDataRepositoryAddress のコメント（ADR-0210）に書かれていたのに、
// UnWrapTyped 側では守られていなかった。notification は元から再帰していて正しい。
func TestHandleGetRepInfosMCPAttachedDataRepNamesAreReal(t *testing.T) {
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
		t.Fatal("attached_data_reps が空（検査になっていない）")
	}

	// 集約の偽名がそのまま漏れていないこと。実在しないので呼び出し側は何もできない。
	for _, attached := range infoResp.AttachedDataReps {
		switch attached.RepName {
		case "TagReps", "TextReps", "NotificationReps", "GPSLogReps":
			t.Errorf("集約の名前がそのまま返っている（実在しない rep 名）: %+v", attached)
		}
	}
}
