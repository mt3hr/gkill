package gkill_server_api

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/api/req_res"
)

// 出力 HTML は gkill を持っていない相手へ渡すもの。
// **google_map_api_key を焼き込むと、利用者のキーがそのまま共有先の手に渡る。**
// 落ちても画面は普通に出るので、テストだけが見張りになる。

// exportRykvHTML は出力エンドポイントを叩く。
func exportRykvHTML(t *testing.T, tsURL string, sessionID string, customize func(*req_res.ExportRykvHTMLRequest)) req_res.ExportRykvHTMLResponse {
	t.Helper()

	query := &find.FindQuery{}
	req := &req_res.ExportRykvHTMLRequest{
		LocaleName:   "en",
		Title:        "出力テスト",
		Columns:      []*find.FindQuery{query},
		MediaMode:    req_res.ExportMediaModeThumbnail,
		OutputFormat: req_res.ExportOutputFormatSingle,
	}
	req.SessionID = sessionID
	if customize != nil {
		customize(req)
	}

	resp := postJSON(t, tsURL+"/api/export_rykv_html", req)
	defer resp.Body.Close()

	var exportResp req_res.ExportRykvHTMLResponse
	if err := json.NewDecoder(resp.Body).Decode(&exportResp); err != nil {
		t.Fatalf("decode export rykv html response: %v", err)
	}
	return exportResp
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

// TestHandleExportRykvHTML_DoesNotLeakGoogleMapAPIKey は、利用者の Google Maps の
// API キーが出力 HTML に一度も現れないことを確認する。
//
// 出力用のガワ（embed/export/template.html）は npm run prepare_install が作る。
// 裸の go build のビルドではこの API だけが 500 を返す（ほかの機能は動く）ので、
// そのときは 500 とエラーコードを確認する。
func TestHandleExportRykvHTML_DoesNotLeakGoogleMapAPIKey(t *testing.T) {
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

	exportResp := exportRykvHTML(t, tsURL, sessionID, nil)

	if _, err := readExportTemplate(); err != nil {
		if !hasErrorCode(exportResp.Errors, message.ExportRykvHTMLTemplateNotFoundError) {
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

// TestHandleExportRykvHTML_RejectsUnsupportedOptions は、選べない組み合わせを
// 収集を始める前に弾くことを確認する。
func TestHandleExportRykvHTML_RejectsUnsupportedOptions(t *testing.T) {
	tsURL, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()

	passwordHash := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	sessionID := loginAndGetSession(t, tsURL, gkillAPI, "admin", passwordHash)

	for _, testCase := range []struct {
		name      string
		customize func(*req_res.ExportRykvHTMLRequest)
	}{
		{"列が1本も無い", func(req *req_res.ExportRykvHTMLRequest) { req.Columns = nil }},
		{"未知のメディアの指定", func(req *req_res.ExportRykvHTMLRequest) { req.MediaMode = "whatever" }},
		{"未知の出力の形", func(req *req_res.ExportRykvHTMLRequest) { req.OutputFormat = "whatever" }},
		{"同名フォルダはまだ無い", func(req *req_res.ExportRykvHTMLRequest) {
			req.OutputFormat = req_res.ExportOutputFormatSidecar
		}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			exportResp := exportRykvHTML(t, tsURL, sessionID, testCase.customize)
			if !hasErrorCode(exportResp.Errors, message.InvalidExportRykvHTMLOptionError) {
				t.Errorf("弾かれていない: %+v", exportResp.Errors)
			}
			if exportResp.HTML != "" {
				t.Error("弾いたのに HTML を返している")
			}
		})
	}
}

// TestHandleExportRykvHTML_RequiresSession は、セッション無しでは出力できないことを確認する。
// 出力は利用者の記録を丸ごと1枚に焼くので、ここが抜けると誰でも持ち出せる。
func TestHandleExportRykvHTML_RequiresSession(t *testing.T) {
	tsURL, _, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()

	exportResp := exportRykvHTML(t, tsURL, "not-a-real-session", nil)
	if len(exportResp.Errors) == 0 {
		t.Error("セッション無しで出力できてしまっている")
	}
	if exportResp.HTML != "" {
		t.Error("セッション無しなのに HTML を返している")
	}
}
