package gkill_server_api

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/api/req_res"
)

// 出力 HTML は gkill を持っていない相手へ渡すもの。
// **google_map_api_key を焼き込むと、利用者のキーがそのまま共有先の手に渡る。**
// 落ちても画面は普通に出るので、テストだけが見張りになる。

// exportKyouHTML は出力エンドポイントを叩く。
func exportKyouHTML(t *testing.T, tsURL string, sessionID string, customize func(*req_res.ExportKyouHTMLRequest)) req_res.ExportKyouHTMLResponse {
	t.Helper()

	req := &req_res.ExportKyouHTMLRequest{
		LocaleName:   "en",
		Title:        "出力テスト",
		Columns:      []req_res.ExportKyouHTMLColumn{exportColumn(t, "q1")},
		MediaMode:    req_res.ExportMediaModeThumbnail,
		OutputFormat: req_res.ExportOutputFormatSingle,
	}
	req.SessionID = sessionID
	if customize != nil {
		customize(req)
	}

	resp := postJSON(t, tsURL+"/api/export_kyou_html", req)
	defer resp.Body.Close()

	var exportResp req_res.ExportKyouHTMLResponse
	if err := json.NewDecoder(resp.Body).Decode(&exportResp); err != nil {
		t.Fatalf("decode export rykv html response: %v", err)
	}
	return exportResp
}

// exportColumn は列1本を、クライアントが送るのと同じ形（query_id 付きの平らな JSON）で組む。
//
// **query_id は find.FindQuery に無い。** クライアントは列をこれで引くので、
// 焼き込みで落ちると2列以上の出力で全列が先頭列に一致する。
func exportColumn(t *testing.T, queryID string) req_res.ExportKyouHTMLColumn {
	t.Helper()
	raw, err := json.Marshal(map[string]any{"query_id": queryID})
	if err != nil {
		t.Fatalf("marshal export column: %v", err)
	}
	column := req_res.ExportKyouHTMLColumn{}
	if err := json.Unmarshal(raw, &column); err != nil {
		t.Fatalf("unmarshal export column: %v", err)
	}
	return column
}

// extractExportPayload は出力 HTML へ差し込まれた焼き込みの JSON を読み出す。
func extractExportPayload(t *testing.T, rendered string) map[string]any {
	t.Helper()
	const marker = `id="gkill-export-data">`
	start := strings.Index(rendered, marker)
	if start < 0 {
		t.Fatal("焼き込みの差し込み先が見つからない")
	}
	start += len(marker)
	end := strings.Index(rendered[start:], "</script>")
	if end < 0 {
		t.Fatal("焼き込みの終わりが見つからない")
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(rendered[start:start+end]), &payload); err != nil {
		t.Fatalf("焼き込みが JSON として読めない: %v", err)
	}
	return payload
}

// TestHandleExportKyouHTML_KeepsColumnQueryID は、焼き込んだ列の条件に query_id が残ることを
// 確認する。find.FindQuery に query_id が無いので、型に落として再 marshal すると消える。
// 消えるとクライアントは列を引けず、**2列以上の出力で全列が先頭列に一致する**
// （例外もエラーも出ないので、1列で試しているうちは露見しない）。
func TestHandleExportKyouHTML_KeepsColumnQueryID(t *testing.T) {
	tsURL, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()

	passwordHash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	sessionID := loginAndGetSession(t, tsURL, gkillAPI, "admin", passwordHash)

	exportResp := exportKyouHTML(t, tsURL, sessionID, func(req *req_res.ExportKyouHTMLRequest) {
		req.Columns = []req_res.ExportKyouHTMLColumn{
			exportColumn(t, "column-1"),
			exportColumn(t, "column-2"),
		}
	})
	// 出力用のガワ（embed/export/template.html）は npm run prepare_install が作る。
	// 裸の go build / go test のビルドではこの API だけが 500 を返すので、そのときは飛ばす
	// （CI の Go ジョブはフロントをビルドしないので、ここが無いとこのテストだけ落ちる）
	if _, err := readExportTemplate(); err != nil {
		t.Skip("出力用のガワが embed に無い（npm run prepare_install を通していないビルド）")
	}
	if len(exportResp.Errors) != 0 {
		t.Fatalf("出力に失敗した: %+v", exportResp.Errors)
	}

	payload := extractExportPayload(t, exportResp.HTML)
	columns, ok := payload["columns"].([]any)
	if !ok {
		t.Fatalf("columns が配列ではない: %T", payload["columns"])
	}
	if len(columns) != 2 {
		t.Fatalf("列の本数が違う: %d", len(columns))
	}
	for i, want := range []string{"column-1", "column-2"} {
		column, ok := columns[i].(map[string]any)
		if !ok {
			t.Fatalf("列 %d が object ではない: %T", i, columns[i])
		}
		query, ok := column["query"].(map[string]any)
		if !ok {
			t.Fatalf("列 %d の query が object ではない: %T", i, column["query"])
		}
		if got := query["query_id"]; got != want {
			t.Errorf("列 %d の query_id が落ちている: want %q, got %v", i, want, got)
		}
	}
}

// hasErrorCode は応答に指定のエラーコードが入っているかを返す。
func hasErrorCode(errors message.GkillErrors, code string) bool {
	for _, gkillError := range errors {
		if gkillError != nil && gkillError.ErrorCode == code {
			return true
		}
	}
	return false
}

// TestHandleExportKyouHTML_DoesNotLeakGoogleMapAPIKey は、利用者の Google Maps の
// API キーが出力 HTML に一度も現れないことを確認する。
//
// 出力用のガワ（embed/export/template.html）は npm run prepare_install が作る。
// 裸の go build のビルドではこの API だけが 500 を返す（ほかの機能は動く）ので、
// そのときは 500 とエラーコードを確認する。
func TestHandleExportKyouHTML_DoesNotLeakGoogleMapAPIKey(t *testing.T) {
	tsURL, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()

	passwordHash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	sessionID := loginAndGetSession(t, tsURL, gkillAPI, "admin", passwordHash)

	device, err := gkillAPI.GetDevice()
	if err != nil {
		t.Fatalf("GetDevice failed: %v", err)
	}

	const apiKey = "googlemapapikeythatmustnotleak"
	applicationConfig, err := gkillAPI.GkillDAOManager.ConfigDAOs.ApplicationConfigDAO.GetApplicationConfig(context.Background(), "admin", device)
	if err != nil || applicationConfig == nil {
		t.Fatalf("GetApplicationConfig failed: %v", err)
	}
	applicationConfig.GoogleMapAPIKey = apiKey
	if _, err := gkillAPI.GkillDAOManager.ConfigDAOs.ApplicationConfigDAO.UpdateApplicationConfig(context.Background(), applicationConfig); err != nil {
		t.Fatalf("UpdateApplicationConfig failed: %v", err)
	}

	addTestKmemo(t, tsURL, sessionID, "出力に含まれる記録")

	exportResp := exportKyouHTML(t, tsURL, sessionID, nil)

	if _, err := readExportTemplate(); err != nil {
		if !hasErrorCode(exportResp.Errors, message.ExportKyouHTMLTemplateNotFoundError) {
			t.Errorf("ガワが無いのにガワ不在のエラーを返していない: %+v", exportResp.Errors)
		}
		if exportResp.HTML != "" {
			t.Error("ガワが無いのに HTML を返している")
		}
		return
	}

	if len(exportResp.Errors) > 0 {
		t.Fatalf("export rykv html errors: %+v", exportResp.Errors)
	}
	if strings.Contains(exportResp.HTML, apiKey) {
		t.Error("Google Maps の API キーが出力 HTML に焼き込まれている（共有先へ漏れる）")
	}
	if !strings.Contains(exportResp.HTML, "出力に含まれる記録") {
		t.Error("出力した記録が HTML に入っていない")
	}
	if exportResp.OutputFormat != req_res.ExportOutputFormatSingle {
		t.Errorf("OutputFormat = %q, want %q", exportResp.OutputFormat, req_res.ExportOutputFormatSingle)
	}
	if !strings.HasSuffix(exportResp.FileName, ".html") {
		t.Errorf("FileName = %q（.html で終わっていない）", exportResp.FileName)
	}
}

// TestHandleExportKyouHTML_RejectsUnsupportedOptions は、選べない組み合わせを
// 収集を始める前に弾くことを確認する。
func TestHandleExportKyouHTML_RejectsUnsupportedOptions(t *testing.T) {
	tsURL, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()

	passwordHash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	sessionID := loginAndGetSession(t, tsURL, gkillAPI, "admin", passwordHash)

	for _, testCase := range []struct {
		name      string
		customize func(*req_res.ExportKyouHTMLRequest)
	}{
		{"列が1本も無い", func(req *req_res.ExportKyouHTMLRequest) { req.Columns = nil }},
		{"未知の画面", func(req *req_res.ExportKyouHTMLRequest) { req.Screen = "dashboard" }},
		{"未知のメディアの指定", func(req *req_res.ExportKyouHTMLRequest) { req.MediaMode = "whatever" }},
		{"未知の出力の形", func(req *req_res.ExportKyouHTMLRequest) { req.OutputFormat = "whatever" }},
		{"同名フォルダはメディアを埋め込まないと空になる", func(req *req_res.ExportKyouHTMLRequest) {
			req.OutputFormat = req_res.ExportOutputFormatSidecar
			req.MediaMode = req_res.ExportMediaModeNone
		}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			exportResp := exportKyouHTML(t, tsURL, sessionID, testCase.customize)
			if !hasErrorCode(exportResp.Errors, message.InvalidExportKyouHTMLOptionError) {
				t.Errorf("弾かれていない: %+v", exportResp.Errors)
			}
			if exportResp.HTML != "" {
				t.Error("弾いたのに HTML を返している")
			}
		})
	}
}

// TestHandleExportKyouHTML_RequiresSession は、セッション無しでは出力できないことを確認する。
// 出力は利用者の記録を丸ごと1枚に焼くので、ここが抜けると誰でも持ち出せる。
func TestHandleExportKyouHTML_RequiresSession(t *testing.T) {
	tsURL, _, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()

	exportResp := exportKyouHTML(t, tsURL, "not-a-real-session", nil)
	if len(exportResp.Errors) == 0 {
		t.Error("セッション無しで出力できてしまっている")
	}
	if exportResp.HTML != "" {
		t.Error("セッション無しなのに HTML を返している")
	}
}

// TestHandleExportKyouHTML_SidecarReturnsZip は、同名フォルダの出力が ZIP の base64 で
// 返ることを確認する。ファイルそのものを body で返す形にはできない（ADR-0712）。
func TestHandleExportKyouHTML_SidecarReturnsZip(t *testing.T) {
	tsURL, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()

	passwordHash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	sessionID := loginAndGetSession(t, tsURL, gkillAPI, "admin", passwordHash)
	addTestKmemo(t, tsURL, sessionID, "出力に含まれる記録")

	exportResp := exportKyouHTML(t, tsURL, sessionID, func(req *req_res.ExportKyouHTMLRequest) {
		req.OutputFormat = req_res.ExportOutputFormatSidecar
	})

	if _, err := readExportTemplate(); err != nil {
		t.Skip("出力用のガワが embed に無い（npm run prepare_install を通していないビルド）")
	}
	if len(exportResp.Errors) > 0 {
		t.Fatalf("export rykv html errors: %+v", exportResp.Errors)
	}
	if exportResp.HTML != "" {
		t.Error("同名フォルダなのに html が入っている")
	}
	if exportResp.ZipBase64 == "" {
		t.Fatal("zip_base64 が空")
	}
	if !strings.HasSuffix(exportResp.FileName, ".zip") {
		t.Errorf("FileName = %q（.zip で終わっていない）", exportResp.FileName)
	}

	zipBytes, err := base64.StdEncoding.DecodeString(exportResp.ZipBase64)
	if err != nil {
		t.Fatalf("zip_base64 が base64 として読めない: %v", err)
	}
	reader, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	if err != nil {
		t.Fatalf("ZIP として開けない: %v", err)
	}
	foundHTML := false
	for _, entry := range reader.File {
		if strings.HasSuffix(entry.Name, ".html") {
			foundHTML = true
		}
	}
	if !foundHTML {
		t.Error("ZIP に HTML が入っていない")
	}
}

// TestHandleExportKyouHTML_BakesScreen は、画面種別の印が焼き込みに乗ることを確かめる。
//
// 出力先のページはこの印だけを見てマウント先を選ぶ（ADR-0420）ので、
// 落ちると mi を出力したのに rykv として開く（例外もエラーも出ない）。
// 空文字を rykv として扱うのは、印を足す前のクライアントが送ってこないため。
func TestHandleExportKyouHTML_BakesScreen(t *testing.T) {
	tsURL, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()

	passwordHash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	sessionID := loginAndGetSession(t, tsURL, gkillAPI, "admin", passwordHash)

	for _, testCase := range []struct {
		name string
		send string
		want string
	}{
		{"指定なしは rykv", "", req_res.ExportScreenRykv},
		{"rykv", req_res.ExportScreenRykv, req_res.ExportScreenRykv},
		{"mi", req_res.ExportScreenMi, req_res.ExportScreenMi},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			exportResp := exportKyouHTML(t, tsURL, sessionID, func(req *req_res.ExportKyouHTMLRequest) {
				req.Screen = testCase.send
			})
			// 出力用のガワが embed に無いビルド（CI の Go ジョブ）では飛ばす
			if _, err := readExportTemplate(); err != nil {
				t.Skip("出力用のガワが embed に無い（npm run prepare_install を通していないビルド）")
			}
			if len(exportResp.Errors) != 0 {
				t.Fatalf("出力に失敗した: %+v", exportResp.Errors)
			}
			payload := extractExportPayload(t, exportResp.HTML)
			if got, _ := payload["screen"].(string); got != testCase.want {
				t.Errorf("焼き込みの screen = %q, want %q", got, testCase.want)
			}
		})
	}
}
