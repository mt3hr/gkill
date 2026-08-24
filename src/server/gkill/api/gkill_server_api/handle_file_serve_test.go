package gkill_server_api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

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
