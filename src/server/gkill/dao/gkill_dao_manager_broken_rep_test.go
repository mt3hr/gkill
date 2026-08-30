package dao

// 索引DB(gkill_id.db)が壊れたrepがあるときの GetRepositories の振る舞い。
//
// 2026-08-30、USB接続ディスク上の gkill_id.db が SQLITE_CORRUPT になり、
// rep 1本の破損でそのユーザの全APIが500になった。そのときエラー文に出たのは
// 「error at create gkill meta info table: database disk image is malformed」だけで、
// **rep名もパスも載っていなかったため、ログから壊れたrepを特定できなかった**。
// 実機へADBで接続して1本ずつ integrity_check を回すまで分からなかった。

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/mt3hr/gkill/src/server/gkill/dao/user_config"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
	_ "modernc.org/sqlite"
)

// writeCorruptSQLiteFile は「SQLiteのDBではあるが壊れている」ファイルを作る。
//
// **ゴミを書くだけでは足りない。** 先頭のマジック "SQLite format 3\0" が無いと
// SQLITE_NOTADB(26。file is not a database)になり、実障害で起きた
// SQLITE_CORRUPT(11。database disk image is malformed)とは別経路のテストになってしまう。
// 正規のDBを作ってから、先頭100バイトのファイルヘッダだけを残して以降を潰すと、
// sqlite_master の解析で SQLITE_CORRUPT になる。
func writeCorruptSQLiteFile(t *testing.T, path string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}

	db, err := sql.Open("sqlite", "file:"+path)
	if err != nil {
		t.Fatalf("sql.Open failed: %v", err)
	}
	// ページを増やしておく。1ページしか無いと潰す先が無い
	for _, ddl := range []string{
		`CREATE TABLE T1(A, B, C)`,
		`CREATE TABLE T2(A, B, C)`,
		`CREATE INDEX IDX_T1 ON T1(A, B)`,
	} {
		if _, err := db.Exec(ddl); err != nil {
			_ = db.Close()
			t.Fatalf("exec %q failed: %v", ddl, err)
		}
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
	// ファイルヘッダ(先頭100バイト)は正規のまま残す
	for i := 100; i < len(raw); i++ {
		raw[i] = 0xFF
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
}

// setupBrokenRepTest は壊れた索引DBを持つdirectory repを含むrep定義一式を作る。
//
// useToWrite が false なら読み取り専用のrepを1本足してそれを壊す（切り離しの対象）。
// true なら書き込み先repそのものを壊す（切り離してはいけない対象）。
func setupBrokenRepTest(t *testing.T, brokenRepDirName string, useToWrite bool) (*GkillDAOManager, string, string, string) {
	t.Helper()

	tmpDir := setupGitRepGlobTestOptions(t)
	ctx := context.Background()

	manager, err := NewGkillDAOManager()
	if err != nil {
		t.Fatalf("NewGkillDAOManager failed: %v", err)
	}
	t.Cleanup(func() { _ = manager.Close() })

	userID := "test_user"
	device := "test_device"
	dataDir := filepath.Join(tmpDir, "datas", userID)
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}

	// 「デバイスごとに各種別の書き込み先repがちょうど1つ」の検査があるので
	// 13種別を1トランザクションでまとめて入れる
	repositoriesDefine := writeRepositoriesForTest(t, userID, device, dataDir)

	if useToWrite {
		// writeRepositoriesForTest が作る書き込み先の directory rep(dataDir/Files)を壊す
		writeCorruptSQLiteFile(t, filepath.Join(dataDir, "Files", ".gkill", "gkill_id.db"))
	} else {
		brokenDir := filepath.Join(dataDir, brokenRepDirName)
		if err := os.MkdirAll(brokenDir, 0o755); err != nil {
			t.Fatalf("MkdirAll failed: %v", err)
		}
		writeCorruptSQLiteFile(t, filepath.Join(brokenDir, ".gkill", "gkill_id.db"))
		repositoriesDefine = append(repositoriesDefine, &user_config.Repository{
			ID:       "test_broken_directory",
			UserID:   userID,
			Device:   device,
			Type:     "directory",
			File:     filepath.ToSlash(brokenDir),
			IsEnable: true,
		})
	}

	if _, err := manager.ConfigDAOs.RepositoryDAO.AddRepositories(ctx, repositoriesDefine); err != nil {
		t.Fatalf("AddRepositories failed: %v", err)
	}

	return manager, userID, device, dataDir
}

// syncLogBuffer は捕捉用の bytes.Buffer をミューテックスで包む。
//
// GetRepositories は通知ループ（GkillNotificator）のゴルーチンを起動し、
// そのゴルーチンはテストが検査している間も slog（差し替えた出力先）へ書き続ける。
// 素の bytes.Buffer だと通知側の Write とテスト側の String が競合し、
// CI の `go test -race ./gkill/dao/...` で実際にデータレースとして検出された。
type syncLogBuffer struct {
	mu     sync.Mutex
	buffer bytes.Buffer
}

func (b *syncLogBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.Write(p)
}

func (b *syncLogBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.String()
}

// captureBrokenRepLogs は GetRepositories が出す構造化ログを検査するため、
// テスト中だけ slog の出力先を差し替える。
func captureBrokenRepLogs(t *testing.T) *syncLogBuffer {
	t.Helper()

	buffer := &syncLogBuffer{}
	original := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(buffer, &slog.HandlerOptions{Level: gkill_log.TraceSQL})))
	t.Cleanup(func() { slog.SetDefault(original) })
	return buffer
}

func findBrokenRepLogRecords(t *testing.T, buffer *syncLogBuffer, message string) []map[string]any {
	t.Helper()

	records := []map[string]any{}
	for _, line := range strings.Split(strings.TrimSpace(buffer.String()), "\n") {
		if line == "" {
			continue
		}
		record := map[string]any{}
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatalf("ログJSONを解析できない: %v: %s", err, line)
		}
		if record["msg"] == message {
			records = append(records, record)
		}
	}
	return records
}

// 読み取り専用のrepが壊れていても、その1本だけ切り離して残りで動くこと。
//
// これが崩れると、rep 1本の物理障害でそのユーザの全APIがERR000018になり、
// 設定画面すら開けず自力復旧の手段が残らない（2026-08-30に実際にそうなった）。
func TestGetRepositoriesDetachesBrokenReadOnlyRep(t *testing.T) {
	manager, userID, device, _ := setupBrokenRepTest(t, "BrokenNote", false)

	repositories, err := manager.GetRepositories(userID, device)
	if err != nil {
		t.Fatalf("読み取り専用repの破損で全体が失敗している: %v", err)
	}

	failures := repositories.LoadFailures()
	if len(failures) != 1 {
		t.Fatalf("切り離しの記録が1件のはず: got %d件 %+v", len(failures), failures)
	}
	if failures[0].RepName != "BrokenNote" {
		t.Errorf("切り離したrepの名前が違う: got %q", failures[0].RepName)
	}
	if failures[0].RepType != "directory" {
		t.Errorf("切り離したrepの種別が違う: got %q", failures[0].RepType)
	}
	if failures[0].Err == nil {
		t.Error("切り離しの原因が記録されていない")
	}

	// 健全なrepは残っていること（切り離しで全部落ちていないか）
	if len(repositories.IDFKyouReps) == 0 {
		t.Error("健全なdirectory repまで消えている")
	}

	// **黙って落とさない**ことがこの機能の前提（ADR-0208の却下を満たす条件）。
	// 記録が空なら、利用者は一覧から消えたrepに気付けない
	if repositories.LoadFailureError() == nil {
		t.Error("LoadFailureErrorがnil。切り離しの原因がログにも診断にも出せない")
	}
}

// 切り離したrepの個別行とまとめ行が、どちらも Error で残ること。
// Warnへ戻ると通常運用の警告に埋もれ、gkill_error.log から障害が消える。
func TestGetRepositoriesLogsDetachedRepAtError(t *testing.T) {
	manager, userID, device, _ := setupBrokenRepTest(t, "BrokenNote", false)
	logBuffer := captureBrokenRepLogs(t)

	if _, err := manager.GetRepositories(userID, device); err != nil {
		t.Fatalf("GetRepositories failed: %v", err)
	}

	individual := findBrokenRepLogRecords(t, logBuffer, "detach broken repository")
	if len(individual) != 1 {
		t.Fatalf("切り離しの個別ログ = %d件, want 1: %s", len(individual), logBuffer.String())
	}
	if individual[0]["level"] != "ERROR" {
		t.Errorf("個別ログのlevel = %v, want ERROR", individual[0]["level"])
	}
	for _, key := range []string{"type", "repName", "error"} {
		if value := individual[0][key]; value == nil || !strings.Contains(value.(string), map[string]string{
			"type": "directory", "repName": "BrokenNote", "error": "database",
		}[key]) {
			t.Errorf("個別ログの %s = %v", key, value)
		}
	}

	summary := findBrokenRepLogRecords(t, logBuffer, "repositories loaded with detached reps")
	if len(summary) != 1 {
		t.Fatalf("切り離しのまとめログ = %d件, want 1: %s", len(summary), logBuffer.String())
	}
	if summary[0]["level"] != "ERROR" {
		t.Errorf("まとめログのlevel = %v, want ERROR", summary[0]["level"])
	}
	if summary[0]["detachedRepCount"] != float64(1) {
		t.Errorf("detachedRepCount = %v, want 1", summary[0]["detachedRepCount"])
	}
	if value, _ := summary[0]["detachedReps"].(string); !strings.Contains(value, "BrokenNote") {
		t.Errorf("detachedReps = %v, BrokenNoteを含むはず", summary[0]["detachedReps"])
	}
}

// 健全な構築で切り離しのまとめログを出さないこと。
// 正常時にもErrorが出ると、監視が常時異常扱いになって本物の欠落を見分けられない。
func TestGetRepositoriesDoesNotLogDetachedSummaryWhenHealthy(t *testing.T) {
	tmpDir := setupGitRepGlobTestOptions(t)
	ctx := context.Background()
	manager, err := NewGkillDAOManager()
	if err != nil {
		t.Fatalf("NewGkillDAOManager failed: %v", err)
	}
	t.Cleanup(func() { _ = manager.Close() })

	userID := "test_user"
	device := "test_device"
	dataDir := filepath.Join(tmpDir, "datas", userID)
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if _, err := manager.ConfigDAOs.RepositoryDAO.AddRepositories(ctx, writeRepositoriesForTest(t, userID, device, dataDir)); err != nil {
		t.Fatalf("AddRepositories failed: %v", err)
	}

	logBuffer := captureBrokenRepLogs(t)
	if _, err := manager.GetRepositories(userID, device); err != nil {
		t.Fatalf("GetRepositories failed: %v", err)
	}
	if records := findBrokenRepLogRecords(t, logBuffer, "repositories loaded with detached reps"); len(records) != 0 {
		t.Errorf("健全な構築で切り離しのまとめログが出ている: %s", logBuffer.String())
	}
}

// 書き込み先repが壊れているときは、切り離さず全体を失敗させること。
//
// 切り離すと WriteXxxRep が nil のまま書き込み経路へ入り、nilポインタ参照でpanicする。
// commit_txはDBトランザクションではないので、途中で落ちると
// 「Kyou本体だけ書けてタグと本文が落ちた」状態が残る。500で止まる方がまだまし。
//
// あわせて、エラーにどのrepが壊れたのかが載っていることも見る。
// 載っていないと実機のログにはSQLiteの文言しか残らず、repを1本ずつ当たるしかなくなる。
func TestGetRepositoriesFailsWhenWriteRepIsBroken(t *testing.T) {
	manager, userID, device, _ := setupBrokenRepTest(t, "", true)

	_, err := manager.GetRepositories(userID, device)
	if err == nil {
		t.Fatal("書き込み先repが壊れているのにGetRepositoriesが成功している")
	}

	got := err.Error()
	for _, want := range []string{"Files", "directory"} {
		if !strings.Contains(got, want) {
			t.Errorf("エラーに %q が含まれていない。どのrepが壊れたか特定できない: %s", want, got)
		}
	}
}

// 構築に失敗したとき、そこまでに開いたrepのDBハンドルが閉じられていること。
//
// 閉じないと、失敗した結果はキャッシュされない(storeRepositoriesは成功時のみ)ため
// リクエストのたびに *sql.DB が積み上がる。Windowsでは元の .db を掴んだままになり、
// 壊れたDBを差し替えて直すことすらできなくなる。
//
// 判定はファイルを消せるかで行う。Windowsでは開いたままのファイルを削除できないので
// これで検出できる。Linux/macOSでは開いていても消せるため、この検査は素通りする。
func TestGetRepositoriesClosesPartiallyBuiltRepositoriesOnFailure(t *testing.T) {
	manager, userID, device, dataDir := setupBrokenRepTest(t, "", true)

	if _, err := manager.GetRepositories(userID, device); err == nil {
		t.Fatal("書き込み先repが壊れているのにGetRepositoriesが成功している")
	}

	if err := os.RemoveAll(dataDir); err != nil {
		t.Errorf("構築失敗後もrepのDBハンドルが残っている(壊れたDBの差し替えができない状態): %v", err)
	}
}

// 切り離しても設定(IsEnable)は書き換えないこと。
//
// 書き換えると、USBを挿し直せば直る種類の障害が恒久的な設定変更になり、
// 利用者は「自分が消していないrepが消えたこと」に気付けない（ADR-0208の制約）。
func TestGetRepositoriesDoesNotDisableBrokenRepInConfig(t *testing.T) {
	manager, userID, device, _ := setupBrokenRepTest(t, "BrokenNote", false)
	ctx := context.Background()

	if _, err := manager.GetRepositories(userID, device); err != nil {
		t.Fatalf("GetRepositories failed: %v", err)
	}

	defines, err := manager.ConfigDAOs.RepositoryDAO.GetRepositories(ctx, userID, device)
	if err != nil {
		t.Fatalf("GetRepositories(config) failed: %v", err)
	}
	found := false
	for _, define := range defines {
		if define.ID != "test_broken_directory" {
			continue
		}
		found = true
		if !define.IsEnable {
			t.Error("切り離したrepの設定がIsEnable=falseへ書き換えられている")
		}
	}
	if !found {
		t.Error("切り離したrepの設定行そのものが消えている")
	}
}

// repNameFromDefine が leaf の GetRepName と同じ規則であること。
//
// ずれると、警告に出す rep 名と query.reps へ渡せる実 rep 名が食い違い、
// 利用者は保存済みの検索条件をどう直せばよいか分からなくなる。
func TestRepNameFromDefineMatchesLeafGetRepName(t *testing.T) {
	cases := []struct {
		repType  string
		filename string
		want     string
	}{
		{"directory", filepath.Join("root", "Dnote"), "Dnote"},
		{"gpslog", filepath.Join("root", "GPSLogs_testdevice"), "GPSLogs_testdevice"},
		{"git_commit_log", filepath.Join("root", "gkill"), "gkill"},
		{"kmemo", filepath.Join("root", "Kmemo.db"), "Kmemo"},
		{"tag", filepath.Join("root", "AutoTag_testdevice.db"), "AutoTag_testdevice"},
		{"timeis", filepath.Join("root", "TimeIs_testdevice.db"), "TimeIs_testdevice"},
	}
	for _, c := range cases {
		if got := repNameFromDefine(c.repType, c.filename); got != c.want {
			t.Errorf("repNameFromDefine(%q, %q) = %q, want %q", c.repType, c.filename, got, c.want)
		}
	}
}

// 上の期待値が leaf の実装と本当に一致しているかを、実物のrepから確かめる。
// 表のほうだけ直して実装とずれるのを防ぐ。
func TestRepNameFromDefineAgreesWithBuiltRep(t *testing.T) {
	tmpDir := setupGitRepGlobTestOptions(t)
	ctx := context.Background()

	manager, err := NewGkillDAOManager()
	if err != nil {
		t.Fatalf("NewGkillDAOManager failed: %v", err)
	}
	t.Cleanup(func() { _ = manager.Close() })

	userID := "test_user"
	device := "test_device"
	dataDir := filepath.Join(tmpDir, "datas", userID)
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}

	// ワイルドカードを含まないrep定義は、実体が無いと expandRepFilePattern が
	// 0件を返して**エラーも警告も無くrepが消える**(rep_file_glob.go)。
	// .dbの型を実際に読み込ませたいので、空ファイルを先に置く
	// (0バイトのファイルはSQLiteでは空のDBとして扱われ、スキーマはrep側が作る)
	if err := os.WriteFile(filepath.Join(dataDir, "Kmemo.db"), []byte{}, 0o600); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	repositoriesDefine := writeRepositoriesForTest(t, userID, device, dataDir)
	if _, err := manager.ConfigDAOs.RepositoryDAO.AddRepositories(ctx, repositoriesDefine); err != nil {
		t.Fatalf("AddRepositories failed: %v", err)
	}

	repositories, err := manager.GetRepositories(userID, device)
	if err != nil {
		t.Fatalf("GetRepositories failed: %v", err)
	}

	// .dbファイルの型（拡張子を落とす規則）
	if len(repositories.KmemoReps) == 0 {
		t.Fatal("kmemo repが読み込まれていない")
	}
	kmemoName, err := repositories.KmemoReps[0].GetRepName(ctx)
	if err != nil {
		t.Fatalf("GetRepName failed: %v", err)
	}
	if got := repNameFromDefine("kmemo", filepath.Join(dataDir, "Kmemo.db")); got != kmemoName {
		t.Errorf("kmemo: repNameFromDefine = %q だが leaf の GetRepName = %q", got, kmemoName)
	}

	// ディレクトリの型（Baseをそのまま使う規則）
	if len(repositories.IDFKyouReps) == 0 {
		t.Fatal("directory repが読み込まれていない")
	}
	idfName, err := repositories.IDFKyouReps[0].GetRepName(ctx)
	if err != nil {
		t.Fatalf("GetRepName failed: %v", err)
	}
	if got := repNameFromDefine("directory", filepath.Join(dataDir, "Files")); got != idfName {
		t.Errorf("directory: repNameFromDefine = %q だが leaf の GetRepName = %q", got, idfName)
	}
}
