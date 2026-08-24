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
