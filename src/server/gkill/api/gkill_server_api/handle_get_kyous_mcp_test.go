package gkill_server_api

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/req_res"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
)

// HandleGetKyousMCP のペイロード構築テスト。
//
// Mi / MiReKyou / TimeIs のDataTypeはリポジトリのSQLが射影名を焼き込むため
// (mi_create, mirekyou_limit, timeis_start ...)、素の型名との完全一致で分岐すると
// 全射影がdefaultへ落ちてpayloadごと消える。実際にそうなっていた期間があるので、
// 「payloadが存在すること」を型ごとに固定する。

const mcpTestPasswordHash = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

// getKyousMCP は /api/get_kyous_mcp を叩いてレスポンスを返す。
//
// クエリを find.FindQuery 構造体ではなく生のmapで組んでいるのは意図的。
// MiCheckState / MiSortType の MarshalJSON が json.Marshal([]byte(s)) になっており
// Goから送るとbase64（"all" → "YWxz"）になってしまうため、構造体経由だと
// filterMiForMi のチェック状態判定に一致せず常に0件になる。
// 実際のクライアント（MCPサーバ）は素の文字列を送るので、ここでも同じ形にする。
func getKyousMCP(t *testing.T, tsURL, sessionID string, query map[string]any, extra map[string]any) req_res.GetKyousMCPResponse {
	t.Helper()

	body := map[string]any{
		"session_id":  sessionID,
		"locale_name": "en",
		"query":       query,
	}
	for k, v := range extra {
		body[k] = v
	}

	resp := postJSON(t, tsURL+"/api/get_kyous_mcp", body)
	defer resp.Body.Close()

	var mcpResp req_res.GetKyousMCPResponse
	if err := json.NewDecoder(resp.Body).Decode(&mcpResp); err != nil {
		t.Fatalf("decode get kyous mcp response: %v", err)
	}
	if len(mcpResp.Errors) > 0 {
		t.Fatalf("get kyous mcp errors: %+v", mcpResp.Errors)
	}
	return mcpResp
}

// findMCPKyouByDataType は指定したdata_typeの最初の1件を返す。
func findMCPKyouByDataType(t *testing.T, kyous []req_res.KyouMCPDTO, dataType string) req_res.KyouMCPDTO {
	t.Helper()
	for _, kyou := range kyous {
		if kyou.DataType == dataType {
			return kyou
		}
	}
	gotTypes := make([]string, 0, len(kyous))
	for _, kyou := range kyous {
		gotTypes = append(gotTypes, kyou.DataType)
	}
	t.Fatalf("data_type %q のKyouが無い: got %q", dataType, gotTypes)
	return req_res.KyouMCPDTO{}
}

// mcpPayload はpayloadをmapとして取り出す。payloadが無ければ落とす。
// json.Decode を通しているのでペイロードは map[string]any になる。
func mcpPayload(t *testing.T, kyou req_res.KyouMCPDTO) map[string]any {
	t.Helper()
	if kyou.Payload == nil {
		t.Fatalf("data_type %q のpayloadが無い（switchの分岐から漏れている）", kyou.DataType)
	}
	payload, ok := kyou.Payload.(map[string]any)
	if !ok {
		t.Fatalf("payloadがオブジェクトでない: %T", kyou.Payload)
	}
	return payload
}

// miQueryForBoard はMi検索用のクエリを組む。
// 5つのinclude_*_miはMiのSQL射影そのものを選ぶスイッチで、
// 全falseだとUNIONの元が無くなり0件になるので必ず立てる。
func miQueryForBoard(board, sortType string) map[string]any {
	return map[string]any{
		"for_mi":            true,
		"mi_board_name":     board,
		"mi_check_state":    "all",
		"mi_sort_type":      sortType,
		"include_create_mi": true,
		"include_check_mi":  true,
		"include_limit_mi":  true,
		"include_start_mi":  true,
		"include_end_mi":    true,
		"only_latest_data":  true,
	}
}

// addTestMiForMCP はタイトル・板名・期限つきのMiを1件作る。
func addTestMiForMCP(t *testing.T, tsURL, sessionID, title, board string, limitTime *time.Time) string {
	t.Helper()
	now := time.Now().Truncate(time.Second)
	miID := GenerateNewID()

	addReq := &req_res.AddMiRequest{
		SessionID:        sessionID,
		LocaleName:       "en",
		WantResponseKyou: true,
		Mi: reps.Mi{
			ID:         miID,
			Title:      title,
			IsChecked:  false,
			BoardName:  board,
			DataType:   "mi",
			LimitTime:  limitTime,
			CreateTime: now,
			CreateApp:  "test",
			CreateUser: "admin",
			UpdateTime: now,
			UpdateApp:  "test",
			UpdateUser: "admin",
		},
	}
	resp := postJSON(t, tsURL+"/api/add_mi", addReq)
	defer resp.Body.Close()

	var addResp req_res.AddMiResponse
	if err := json.NewDecoder(resp.Body).Decode(&addResp); err != nil {
		t.Fatalf("decode add mi response: %v", err)
	}
	if len(addResp.Errors) > 0 {
		t.Fatalf("add mi errors: %+v", addResp.Errors)
	}
	return miID
}

// TestHandleGetKyousMCP_MiPayload はMiの板名・タイトルがMCPに届くことを確認する。
// data_type が mi_create なのに switch が "mi" を見ていたため、
// payloadごと（＝板名もタイトルも）落ちていた。
func TestHandleGetKyousMCP_MiPayload(t *testing.T) {
	tsURL, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()

	sessionID := loginAndGetSession(t, tsURL, gkillAPI, "admin", mcpTestPasswordHash)

	limitTime := time.Now().Truncate(time.Second).Add(24 * time.Hour)
	addTestMiForMCP(t, tsURL, sessionID, "MCPに板名を返すタスク", "mcp_mi_board", &limitTime)

	mcpResp := getKyousMCP(t, tsURL, sessionID, miQueryForBoard("mcp_mi_board", "create_time"), nil)
	if len(mcpResp.Kyous) == 0 {
		t.Fatal("Miが1件も返っていない")
	}

	kyou := findMCPKyouByDataType(t, mcpResp.Kyous, "mi_create")
	payload := mcpPayload(t, kyou)

	if payload["kind"] != "mi" {
		t.Errorf("kind = %v, want %q", payload["kind"], "mi")
	}
	if payload["title"] != "MCPに板名を返すタスク" {
		t.Errorf("title = %v, want %q", payload["title"], "MCPに板名を返すタスク")
	}
	if payload["board_name"] != "mcp_mi_board" {
		t.Errorf("board_name = %v, want %q", payload["board_name"], "mcp_mi_board")
	}
	if payload["is_checked"] != false {
		t.Errorf("is_checked = %v, want false", payload["is_checked"])
	}
	// related_timeは射影で意味が変わるので、作成日時はpayload側で復元できる必要がある
	if _, ok := payload["create_time"]; !ok {
		t.Error("create_time が無い（related_timeは射影で意味が変わるので復元できない）")
	}
	if _, ok := payload["limit_time"]; !ok {
		t.Error("limit_time が無い")
	}
}

// TestHandleGetKyousMCP_MiPayloadOnEveryProjection は
// mi_sort_type を変えて別射影のdata_typeになってもpayloadが付くことを確認する。
// 射影名は mi_create / mi_check / mi_limit / mi_start / mi_end と枝分かれするので、
// 1つだけ通しても他が落ちている状態を見逃す。
func TestHandleGetKyousMCP_MiPayloadOnEveryProjection(t *testing.T) {
	tsURL, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()

	sessionID := loginAndGetSession(t, tsURL, gkillAPI, "admin", mcpTestPasswordHash)

	limitTime := time.Now().Truncate(time.Second).Add(24 * time.Hour)
	addTestMiForMCP(t, tsURL, sessionID, "射影ごとのタスク", "inbox", &limitTime)

	tests := []struct {
		sortType     string
		wantDataType string
	}{
		{"create_time", "mi_create"},
		{"limit_time", "mi_limit"},
	}

	for _, tt := range tests {
		t.Run(tt.sortType, func(t *testing.T) {
			mcpResp := getKyousMCP(t, tsURL, sessionID, miQueryForBoard("inbox", tt.sortType), nil)
			kyou := findMCPKyouByDataType(t, mcpResp.Kyous, tt.wantDataType)
			payload := mcpPayload(t, kyou)
			if payload["kind"] != "mi" {
				t.Errorf("kind = %v, want %q", payload["kind"], "mi")
			}
			if payload["title"] != "射影ごとのタスク" {
				t.Errorf("title = %v, want %q", payload["title"], "射影ごとのタスク")
			}
		})
	}
}

// TestHandleGetKyousMCP_MiReKyouPayload は既存Kyouをタスク化したもの（MiReKyou）に
// payloadが付くことを確認する。MiReKyouはタイトルを持たないので、
// 中身を引くための target_id が返ることが要件。
func TestHandleGetKyousMCP_MiReKyouPayload(t *testing.T) {
	tsURL, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()

	sessionID := loginAndGetSession(t, tsURL, gkillAPI, "admin", mcpTestPasswordHash)

	targetID := addTestKmemo(t, tsURL, sessionID, "タスク化されたメモ")
	addTestMiReKyou(t, tsURL, sessionID, targetID, "mcp_mirekyou_board")

	mcpResp := getKyousMCP(t, tsURL, sessionID, miQueryForBoard("mcp_mirekyou_board", "create_time"), nil)
	kyou := findMCPKyouByDataType(t, mcpResp.Kyous, "mirekyou_create")
	payload := mcpPayload(t, kyou)

	if payload["kind"] != "mirekyou" {
		t.Errorf("kind = %v, want %q", payload["kind"], "mirekyou")
	}
	if payload["target_id"] != targetID {
		t.Errorf("target_id = %v, want %q", payload["target_id"], targetID)
	}
	if payload["board_name"] != "mcp_mirekyou_board" {
		t.Errorf("board_name = %v, want %q", payload["board_name"], "mcp_mirekyou_board")
	}
}

// TestHandleGetKyousMCP_TimeIsPayload はTimeIsのタイトル・開始時刻が返ることを確認する。
// data_type が timeis_start / timeis_end なのに switch が "timeis" を見ていたため落ちていた。
func TestHandleGetKyousMCP_TimeIsPayload(t *testing.T) {
	tsURL, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()

	sessionID := loginAndGetSession(t, tsURL, gkillAPI, "admin", mcpTestPasswordHash)

	now := time.Now().Truncate(time.Second)
	timeisID := GenerateNewID()
	addReq := &req_res.AddTimeIsRequest{
		SessionID:        sessionID,
		LocaleName:       "en",
		WantResponseKyou: true,
		TimeIs: reps.TimeIs{
			ID:         timeisID,
			Title:      "MCP向けの作業",
			StartTime:  now,
			EndTime:    nil,
			DataType:   "timeis",
			CreateTime: now,
			CreateApp:  "test",
			CreateUser: "admin",
			UpdateTime: now,
			UpdateApp:  "test",
			UpdateUser: "admin",
		},
	}
	resp := postJSON(t, tsURL+"/api/add_timeis", addReq)
	defer resp.Body.Close()
	var addResp req_res.AddTimeIsResponse
	if err := json.NewDecoder(resp.Body).Decode(&addResp); err != nil {
		t.Fatalf("decode add timeis response: %v", err)
	}
	if len(addResp.Errors) > 0 {
		t.Fatalf("add timeis errors: %+v", addResp.Errors)
	}

	mcpResp := getKyousMCP(t, tsURL, sessionID, map[string]any{
		"only_latest_data":   true,
		"include_end_timeis": true,
	}, nil)

	kyou := findMCPKyouByDataType(t, mcpResp.Kyous, "timeis_start")
	payload := mcpPayload(t, kyou)

	if payload["kind"] != "timeis" {
		t.Errorf("kind = %v, want %q", payload["kind"], "timeis")
	}
	if payload["title"] != "MCP向けの作業" {
		t.Errorf("title = %v, want %q", payload["title"], "MCP向けの作業")
	}
	if _, ok := payload["start_time"]; !ok {
		t.Error("start_time が無い")
	}
}

// TestHandleGetKyousMCP_ReKyouPayload は再投稿（ReKyou）に
// 元Kyouを引くための target_id が返ることを確認する。
func TestHandleGetKyousMCP_ReKyouPayload(t *testing.T) {
	tsURL, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()

	sessionID := loginAndGetSession(t, tsURL, gkillAPI, "admin", mcpTestPasswordHash)

	now := time.Now().Truncate(time.Second)
	kmemoID := addTestKmemo(t, tsURL, sessionID, "再投稿されるメモ")

	addReKyouReq := &req_res.AddReKyouRequest{
		SessionID:  sessionID,
		LocaleName: "en",
		ReKyou: reps.ReKyou{
			ID:          GenerateNewID(),
			TargetID:    kmemoID,
			DataType:    "re_kyou",
			RelatedTime: now,
			CreateTime:  now,
			CreateApp:   "test",
			CreateUser:  "admin",
			UpdateTime:  now,
			UpdateApp:   "test",
			UpdateUser:  "admin",
		},
	}
	resp := postJSON(t, tsURL+"/api/add_rekyou", addReKyouReq)
	defer resp.Body.Close()
	var addResp req_res.AddReKyouResponse
	if err := json.NewDecoder(resp.Body).Decode(&addResp); err != nil {
		t.Fatalf("decode add rekyou response: %v", err)
	}
	if len(addResp.Errors) > 0 {
		t.Fatalf("add rekyou errors: %+v", addResp.Errors)
	}

	mcpResp := getKyousMCP(t, tsURL, sessionID, map[string]any{"only_latest_data": true}, nil)
	kyou := findMCPKyouByDataType(t, mcpResp.Kyous, "rekyou")
	payload := mcpPayload(t, kyou)

	if payload["kind"] != "rekyou" {
		t.Errorf("kind = %v, want %q", payload["kind"], "rekyou")
	}
	if payload["target_id"] != kmemoID {
		t.Errorf("target_id = %v, want %q", payload["target_id"], kmemoID)
	}
}

// TestHandleGetKyousMCP_IncludeRepName は rep_name が任意フラグであることを確認する。
// 全件に常時載せるとMaxSizeMBの打ち切りが早まるので、既定では出さない。
func TestHandleGetKyousMCP_IncludeRepName(t *testing.T) {
	tsURL, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()

	sessionID := loginAndGetSession(t, tsURL, gkillAPI, "admin", mcpTestPasswordHash)

	addTestKmemo(t, tsURL, sessionID, "rep_nameの確認用メモ")

	query := map[string]any{"only_latest_data": true}

	withoutFlag := getKyousMCP(t, tsURL, sessionID, query, nil)
	if len(withoutFlag.Kyous) == 0 {
		t.Fatal("Kyouが1件も返っていない")
	}
	for _, kyou := range withoutFlag.Kyous {
		if kyou.RepName != "" {
			t.Errorf("include_rep_name未指定なのにrep_nameが入っている: %q", kyou.RepName)
		}
	}

	withFlag := getKyousMCP(t, tsURL, sessionID, query, map[string]any{"include_rep_name": true})
	if len(withFlag.Kyous) == 0 {
		t.Fatal("include_rep_name:trueでKyouが1件も返っていない")
	}
	for _, kyou := range withFlag.Kyous {
		if kyou.RepName == "" {
			t.Errorf("include_rep_name:trueなのにdata_type %q のrep_nameが空", kyou.DataType)
		}
	}
}

// TestPayloadKindOfDataType は射影名からペイロード種別への寄せを固定する。
// mirekyou_ を mi_ より先に判定しないと、MiReKyouがMiのペイロードを名乗る。
func TestPayloadKindOfDataType(t *testing.T) {
	tests := []struct {
		dataType string
		want     string
	}{
		{"mi_create", "mi"},
		{"mi_check", "mi"},
		{"mi_limit", "mi"},
		{"mi_start", "mi"},
		{"mi_end", "mi"},
		{"mi", "mi"},
		{"mirekyou_create", "mirekyou"},
		{"mirekyou_check", "mirekyou"},
		{"mirekyou_limit", "mirekyou"},
		{"mirekyou_start", "mirekyou"},
		{"mirekyou_end", "mirekyou"},
		{"timeis_start", "timeis"},
		{"timeis_end", "timeis"},
		{"timeis", "timeis"},
		{"rekyou", "rekyou"},
		{"kmemo", "kmemo"},
		{"kc", "kc"},
		{"nlog", "nlog"},
		{"lantana", "lantana"},
		{"urlog", "urlog"},
		{"idf", "idf"},
		{"git_commit_log", "git_commit_log"},
		// プラグインのdata_typeは素通しし、呼び出し側のdefaultでrep_name照合に回す
		{"claude_conversation", "claude_conversation"},
	}

	for _, tt := range tests {
		t.Run(tt.dataType, func(t *testing.T) {
			if got := payloadKindOfDataType(tt.dataType); got != tt.want {
				t.Errorf("payloadKindOfDataType(%q) = %q, want %q", tt.dataType, got, tt.want)
			}
		})
	}
}

// TestHandleGetKyousMCP_PagingDoesNotDropSameRelatedTime は、同一RelatedTimeのかたまりが
// ページ境界で切り捨てられないことを固定する。
//
// NextCursorは「その時刻より厳密に前」から次ページを始めるので、同時刻のかたまりの
// 途中でページを切ると、返しそこねた残りが次ページからも漏れて永久に取れない。
// 一括取り込みのIDFやFitbitの日次指標のように同時刻が並ぶデータで現実に起きる。
// Limitを少し超えてでも、かたまりの終わりまで返しきること。
func TestHandleGetKyousMCP_PagingDoesNotDropSameRelatedTime(t *testing.T) {
	tsURL, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()

	sessionID := loginAndGetSession(t, tsURL, gkillAPI, "admin", mcpTestPasswordHash)

	sameTime := time.Now().Truncate(time.Second).Add(-1 * time.Hour)
	wantContents := []string{}
	for i := range 3 {
		content := fmt.Sprintf("同時刻のメモ%d", i)
		addTestKmemoWithRelatedTime(t, tsURL, sessionID, content, sameTime)
		wantContents = append(wantContents, content)
	}
	// かたまりより古い時刻にも1件置いて、ページが2枚以上になるようにする
	olderContent := "ひとつ前の時刻のメモ"
	addTestKmemoWithRelatedTime(t, tsURL, sessionID, olderContent, sameTime.Add(-1*time.Minute))
	wantContents = append(wantContents, olderContent)

	// limit=1 で回す。かたまりを割る実装だと同時刻3件のうち1件しか取れず、残り2件が消える。
	seen := map[string]int{}
	cursor := ""
	for range 10 {
		extra := map[string]any{"limit": 1}
		if cursor != "" {
			extra["cursor"] = cursor
		}
		mcpResp := getKyousMCP(t, tsURL, sessionID, map[string]any{}, extra)
		for _, kyou := range mcpResp.Kyous {
			payload, ok := kyou.Payload.(map[string]any)
			if !ok {
				continue
			}
			if content, ok := payload["content"].(string); ok {
				seen[content]++
			}
		}
		if !mcpResp.HasMore || mcpResp.NextCursor == "" {
			break
		}
		cursor = mcpResp.NextCursor
	}

	for _, content := range wantContents {
		if seen[content] == 0 {
			t.Errorf("%q がページングで取りこぼされた (取得できたもの: %v)", content, seen)
		}
	}
}

// TestHandleGetKyousMCP_InvalidCursorIsError は、解釈できないカーソルを黙って無視して
// 1ページ目に戻さないことを固定する。無視すると呼び出し側は同じページを受け取り続け、
// ページングが永久に終わらない。
func TestHandleGetKyousMCP_InvalidCursorIsError(t *testing.T) {
	tsURL, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()

	sessionID := loginAndGetSession(t, tsURL, gkillAPI, "admin", mcpTestPasswordHash)

	resp := postJSON(t, tsURL+"/api/get_kyous_mcp", map[string]any{
		"session_id":  sessionID,
		"locale_name": "en",
		"query":       map[string]any{},
		"cursor":      "not-a-time",
	})
	defer resp.Body.Close()

	var mcpResp req_res.GetKyousMCPResponse
	if err := json.NewDecoder(resp.Body).Decode(&mcpResp); err != nil {
		t.Fatalf("decode get kyous mcp response: %v", err)
	}
	if len(mcpResp.Errors) == 0 {
		t.Error("解釈できないカーソルなのにエラーが返っていない（黙って1ページ目に戻すとページングが終わらない）")
	}
	if len(mcpResp.Kyous) != 0 {
		t.Errorf("エラーなのにKyouを返している: %d件", len(mcpResp.Kyous))
	}
}

// TestHandleGetKyousMCP_DateOnlyCursorIsAccepted は日付のみ(YYYY-MM-DD)のカーソルを
// 受け付けることを固定する。MCPサーバは日付のみを日時へ正規化してから送るが、
// APIを直接叩くクライアントはそのまま送ってくる。
func TestHandleGetKyousMCP_DateOnlyCursorIsAccepted(t *testing.T) {
	tsURL, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()

	sessionID := loginAndGetSession(t, tsURL, gkillAPI, "admin", mcpTestPasswordHash)
	addTestKmemoWithRelatedTime(t, tsURL, sessionID, "日付カーソルの手前のメモ", time.Now().Add(-48*time.Hour))

	// エラーが返るとgetKyousMCPが落とすので、通ること自体が検査になる
	getKyousMCP(t, tsURL, sessionID, map[string]any{}, map[string]any{
		"cursor": time.Now().Format(time.DateOnly),
	})
}
