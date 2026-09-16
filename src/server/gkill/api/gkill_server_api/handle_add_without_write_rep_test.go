package gkill_server_api

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/api/req_res"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_options"
)

// 書き込み先 rep が無いまま追加しようとしたときの応答（usecase/write_rep_missing.go）。
//
// 2026-09-15 まで、tx を使わない Add*/Update* は WriteXxxRep が nil のまま AddXxxInfo を呼んで
// nil ポインタ参照で panic し、利用者には「内部エラーが発生しました」しか出なかった。
// 設定 DAO は「種別ごとに書き込み先1つ」を検査するので、設定だけでこの状態は作れない。
// 実際に起きるのは「書き込み先に指定したファイルが無く、glob が何も見つけずに静かに読み込まれない」
// ときで（新しい rep 種別を既存の設定へ足したときの典型）、ここでもそれを再現する
// （rep は最初のリクエストで読み込まれるので、ログインより前に消す）。
func TestHandleAddKmemo_WithoutWriteRepIsConfigErrorNotPanic(t *testing.T) {
	tsURL, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()

	// addTestRepositories が kmemo の書き込み先として登録した空の DB を消す
	if err := os.Remove(filepath.Join(gkill_options.GkillHomeDir, "datas", "kmemo.db")); err != nil {
		t.Fatalf("remove kmemo.db: %v", err)
	}

	sessionID := loginAndGetSession(t, tsURL, gkillAPI, "admin", regressionTestPasswordHash)

	device, err := gkillAPI.GetDevice()
	if err != nil {
		t.Fatalf("GetDevice: %v", err)
	}
	repositories, err := gkillAPI.GkillDAOManager.GetRepositories("admin", device)
	if err != nil {
		t.Fatalf("GetRepositories: %v", err)
	}
	if repositories.WriteKmemoRep != nil {
		t.Fatal("壊れた kmemo.db が書き込み先として残っている（このテストの前提が崩れている）")
	}

	now := time.Now().Truncate(time.Second)
	resp := postJSON(t, tsURL+"/api/add_kmemo", &req_res.AddKmemoRequest{
		SessionID: sessionID, LocaleName: "en",
		Kmemo: reps.Kmemo{
			ID: GenerateNewID(), Content: "no write rep", RelatedTime: now, DataType: "kmemo",
			CreateTime: now, CreateApp: "test", CreateUser: "admin", UpdateTime: now, UpdateApp: "test", UpdateUser: "admin",
		},
	})
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	var res req_res.AddKmemoResponse
	if err := json.Unmarshal(body, &res); err != nil {
		t.Fatalf("decode add kmemo response: %v（panic なら recover の 500 で本文が JSON でない）: %s", err, body)
	}
	if len(res.Errors) != 1 || res.Errors[0].ErrorCode != message.WriteRepMissingError {
		t.Fatalf("errors = %+v, want %s 1件", res.Errors, message.WriteRepMissingError)
	}
	if res.AddedKmemo != nil {
		t.Errorf("added_kmemo = %+v, want nil（書き込み先が無いのに書けたことになっている）", res.AddedKmemo)
	}
	// ステータスはコードから決まる（http_status.go の表）
	if want := message.HTTPStatusForErrors(res.Errors); resp.StatusCode != want {
		t.Errorf("status = %d, want %d", resp.StatusCode, want)
	}

	// 「誰の問題か・何が起きたか」は設定の不備として返る（設定→保存先で直せる）。
	// GkillError は受信側で error_kind / reason を持たないので、ワイヤの JSON から読む。
	var wire struct {
		Errors []struct {
			ErrorKind string `json:"error_kind"`
			Reason    string `json:"reason"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(body, &wire); err != nil {
		t.Fatalf("decode wire errors: %v", err)
	}
	if len(wire.Errors) != 1 {
		t.Fatalf("wire errors = %d 件, want 1", len(wire.Errors))
	}
	if wire.Errors[0].ErrorKind != message.ErrorKindConfig || wire.Errors[0].Reason != message.ReasonWriteRepMissing {
		t.Errorf("error_kind = %q reason = %q, want %q / %q", wire.Errors[0].ErrorKind, wire.Errors[0].Reason, message.ErrorKindConfig, message.ReasonWriteRepMissing)
	}
}
