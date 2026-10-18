package gkill_server_api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_options"
)

// TestFileServeSetsSecurityHeaders は、/files/ の応答にも /zip_cache/ と同じ
// セキュリティヘッダが付くことを確認する（表は handle_zip_cache_file_serve_test.go と共有）。
//
// 取り込んだ HTML は allow-scripts 付きの sandbox で開き、allow-same-origin は付けない。
// 文書は opaque origin になるので、スクリプトは動くが gkill のクッキーにも
// localStorage にも触れない（ADR-0711）。SVG やその他は従来どおりスクリプトの動かない
// sandbox、PDF は内蔵ビューワのために CSP を付けない。
//
// ルートは serve.go と同じ形（withUserContentSecurityHeaders → wrapNoAuth → HandleFileServe）で
// 登録する。setupTestRouter は /api/* しか登録しないので、ここで足さないとラッパーを通らない。
func TestFileServeSetsSecurityHeaders(t *testing.T) {
	tsURL, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()

	router := gkillAPI.GkillDAOManager.GetRouter()
	router.PathPrefix("/files/").HandlerFunc(
		withUserContentSecurityHeaders(gkillAPI.wrapNoAuth(gkillAPI.HandleFileServe)))

	sessionID := loginAndGetSession(t, tsURL, gkillAPI, "admin", regressionTestPasswordHash)

	// addTestRepositories が作る directory 型 rep。rep 名はディレクトリ名（filepath.Base）になる
	const repName = "directory_dir"
	repDir := filepath.Join(gkill_options.GkillHomeDir, "datas", repName)
	cases := userContentHeaderCases()
	for _, c := range cases {
		if err := os.WriteFile(filepath.Join(repDir, c.fileName), []byte(c.content), 0o600); err != nil {
			t.Fatalf("failed to write %s: %v", c.fileName, err)
		}
	}

	for _, c := range cases {
		t.Run(c.requestName(), func(t *testing.T) {
			resp := getWithSessionCookie(t, tsURL+"/files/"+repName+"/"+c.requestName(), sessionID)
			defer resp.Body.Close()
			assertUserContentSecurityHeaders(t, resp, c)
		})
	}
}

// GetRepositories 失敗時のステータスの回帰テスト。
//
// HandleFileServe は req_res を使わずファイル本体を返す経路で、
// response_status_guard_test.go の免除対象。エラーコード表(http_status.go)を
// 通らないので、ステータスはこのハンドラの WriteHeader 直書きだけが頼りになる。
// リポジトリ取得の失敗はかつて 403(クッキー不備・セッション不正と同じ)を返しており、
// ステータスを見る層から認可の失敗とサーバ障害が区別できなかった。
// 同じ失敗を返す他の経路(auth_middleware / handle_urlog_bookmarklet_address)と
// 揃えて 500 を返すことをここで固定する。
func TestHandleFileServe_GetRepositoriesFailureReturns500(t *testing.T) {
	ts, gkillAPI, cleanup := setupTestRouter(t)
	defer cleanup()

	// 認証は成功させたいので、セッションは先に正規の手順で作る。
	sessionID := loginAndGetSession(t, ts.URL, gkillAPI, "admin", regressionTestPasswordHash)

	// RepositoryDAO だけを先に閉じ、リポジトリ設定の読み取り
	// (GkillDAOManager.GetRepositories → ConfigDAOs.RepositoryDAO.GetRepositories)を
	// 失敗させる。認証に使う LoginSessionDAO / AccountDAO は開いたままなので、
	// クッキーの検証は成功し、失敗するのは GetRepositories だけになる。
	// cleanup がもう一度呼ぶ Close は database/sql では無害(冪等)。
	if err := gkillAPI.GkillDAOManager.ConfigDAOs.RepositoryDAO.Close(context.Background()); err != nil {
		t.Fatalf("RepositoryDAO.Close failed: %v", err)
	}

	// 注入の前提確認: この仕込みで GetRepositories が実際にエラーになること。
	// (成功して返るなら、このテストは何も検証せずに通ってしまう)
	device, err := gkillAPI.GetDevice()
	if err != nil {
		t.Fatalf("GetDevice failed: %v", err)
	}
	if _, err := gkillAPI.GkillDAOManager.GetRepositories("admin", device); err == nil {
		t.Fatal("前提が崩れている: RepositoryDAO を閉じたのに GetRepositories が成功した")
	}

	req := httptest.NewRequest(http.MethodGet, "/files/TestRep/test.png", nil)
	req.AddCookie(&http.Cookie{Name: "gkill_session_id", Value: sessionID})
	rec := httptest.NewRecorder()
	gkillAPI.HandleFileServe(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500 (リポジトリ取得の失敗は認可の失敗ではなくサーバ障害)", rec.Code)
	}
}
