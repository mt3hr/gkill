package gkill_server_api

// 読み込めなかったrepが、errors ではなく messages の警告として返ることのテスト。
//
// errors に載せるとハンドラもクライアントも検索全体を失敗扱いにして結果を捨てるため、
// rep 1本の破損で他のrepの検索結果まで消える。
// 一方で黙って落とすと「静かな欠落」になる。保存済みの検索条件(列のReps)に残った
// 名前は filterKyousByRepName に「実在するが選ばれていない」と扱われ、
// エラーも警告も無いまま0件になるので、利用者は直しようがない。

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/dao/user_config"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_options"
	_ "modernc.org/sqlite"
)

// writeCorruptSQLiteFileForAPITest は「SQLiteのDBではあるが壊れている」ファイルを作る。
//
// ゴミを書くだけだと SQLITE_NOTADB(26) になり、実障害の SQLITE_CORRUPT(11) とは
// 別経路のテストになる。正規のDBを作ってからファイルヘッダ(先頭100バイト)だけ残して潰す。
// dao 側にも同じものがあるが、パッケージを跨いでテストヘルパを公開したくないので分けている。
func writeCorruptSQLiteFileForAPITest(t *testing.T, path string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	db, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatalf("sql.Open failed: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE T1(A, B, C)`); err != nil {
		_ = db.Close()
		t.Fatalf("create table failed: %v", err)
	}
	for i := range 200 {
		if _, err := db.Exec(`INSERT INTO T1 VALUES (?,?,?)`, i, i, i); err != nil {
			_ = db.Close()
			t.Fatalf("insert failed: %v", err)
		}
	}
	if err := db.Close(); err != nil {
		t.Fatalf("db.Close failed: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if len(raw) <= 100 {
		t.Fatalf("仕込んだDBが小さすぎる: %d バイト", len(raw))
	}
	for i := 100; i < len(raw); i++ {
		raw[i] = 0xFF
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
}

func TestHandleGetKyous_BrokenRepIsWarningNotError(t *testing.T) {
	tsURL, gkillAPI, cleanup := setupTestRouterWithRepos(t)
	defer cleanup()

	ctx := context.Background()
	device, err := gkillAPI.GetDevice()
	if err != nil {
		t.Fatalf("GetDevice failed: %v", err)
	}

	// GetRepositories はユーザ+デバイス単位で1回だけ構築してキャッシュするので、
	// 壊れたrepは最初のリクエストより前に足す必要がある
	brokenDir := filepath.Join(gkill_options.GkillHomeDir, "datas", "BrokenNote")
	if err := os.MkdirAll(brokenDir, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	writeCorruptSQLiteFileForAPITest(t, filepath.Join(brokenDir, ".gkill", "gkill_id.db"))

	ok, err := gkillAPI.GkillDAOManager.ConfigDAOs.RepositoryDAO.AddRepositories(ctx, []*user_config.Repository{{
		ID:         GenerateNewID(),
		UserID:     "admin",
		Device:     device,
		Type:       "directory",
		File:       filepath.ToSlash(brokenDir),
		UseToWrite: false,
		IsEnable:   true,
	}})
	if err != nil {
		t.Fatalf("AddRepositories failed: %v", err)
	}
	if !ok {
		t.Fatal("AddRepositories returned false")
	}

	sessionID := loginAndGetSession(t, tsURL, gkillAPI, "admin", regressionTestPasswordHash)

	now := time.Now().Truncate(time.Second)
	calendarStart := now.Add(-time.Hour)
	calendarEnd := now.Add(time.Hour)
	getResp := getKyousWithQuery(t, tsURL, sessionID, &find.FindQuery{
		CalendarStartDate: &calendarStart,
		CalendarEndDate:   &calendarEnd,
	})

	if len(getResp.Errors) != 0 {
		t.Fatalf("repの読み込み失敗を errors に載せてはいけない（クライアントが検索結果ごと捨てる）: %+v", getResp.Errors)
	}

	foundWarning := false
	for _, msg := range getResp.Messages {
		if msg.MessageCode != message.FindKyousRepLoadWarningMessage {
			continue
		}
		foundWarning = true

		if !strings.Contains(msg.Message, "BrokenNote") {
			t.Errorf("警告にrep名が入っていない。利用者はどのrepが消えたか分からない: %q", msg.Message)
		}
		// GkillMessage には GkillError のような伏せ処理が無い。
		// パスを載せるとホームディレクトリの利用者名が応答に出る
		if strings.ContainsAny(msg.Message, `/\`) {
			t.Errorf("警告にパス区切りが含まれている（環境固有の情報が漏れている）: %q", msg.Message)
		}
	}
	if !foundWarning {
		t.Errorf("repの読み込み失敗が messages の警告(%s)として返っていない: %+v", message.FindKyousRepLoadWarningMessage, getResp.Messages)
	}

	// MCPは通常検索だけでなく、警告生成後に早期returnする count_only / group_by でも
	// 同じ欠落を伝えること。partial は付随データ取得失敗専用なので立てない。
	mcpCases := []struct {
		name  string
		extra map[string]any
	}{
		{name: "ordinary"},
		{name: "count_only", extra: map[string]any{"count_only": true}},
		{name: "group_by", extra: map[string]any{"group_by": "day"}},
	}
	for _, testCase := range mcpCases {
		t.Run("mcp_"+testCase.name, func(t *testing.T) {
			mcpResp := getKyousMCP(t, tsURL, sessionID, map[string]any{}, testCase.extra)
			if mcpResp.Partial {
				t.Error("読み込めない記録保管場所の警告だけでpartial=trueになっている")
			}
			found := false
			for _, warning := range mcpResp.Warnings {
				if strings.Contains(warning, "BrokenNote") && strings.Contains(warning, "query.reps") {
					found = true
				}
			}
			if !found {
				t.Errorf("MCPの%s応答に保管場所の欠落とquery.repsの注意が無い: %v", testCase.name, mcpResp.Warnings)
			}
		})
	}
}
