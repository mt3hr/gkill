package share_kyou_info

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

// newSchema100ShareKyouInfoDB はスキーマ1.0.0相当のshare_kyou_info.dbを作る。
// 1.0.0のFIND_QUERY_JSONは use_* フラグ入りの旧形式FindQuery JSONを保持していた。
func newSchema100ShareKyouInfoDB(t *testing.T, findQueryJSONByID map[string]string) string {
	t.Helper()
	filename := filepath.Join(t.TempDir(), "share_kyou_info.db")
	db, err := sql.Open("sqlite", "file:"+filename)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	stmts := []string{
		`CREATE TABLE "SHARE_KYOU_INFO" (
  ID PRIMARY KEY NOT NULL,
  USER_ID NOT NULL,
  DEVICE NOT NULL,
  SHARE_TITLE NOT NULL,
  SHARE_ID NOT NULL,
  FIND_QUERY_JSON NOT NULL,
  VIEW_TYPE NOT NULL
);`,
		`CREATE TABLE GKILL_META_INFO (KEY NOT NULL, VALUE, PRIMARY KEY(KEY));`,
		`INSERT INTO GKILL_META_INFO(KEY, VALUE) VALUES('SCHEMA_VERSION_SHARE_KYOU_INFO', '1.0.0');`,
	}
	for _, stmt := range stmts {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			t.Fatalf("failed to exec %q: %v", stmt, err)
		}
	}

	for id, findQueryJSON := range findQueryJSONByID {
		_, err := db.ExecContext(ctx,
			`INSERT INTO SHARE_KYOU_INFO(ID, USER_ID, DEVICE, SHARE_TITLE, SHARE_ID, FIND_QUERY_JSON, VIEW_TYPE) VALUES(?, 'testuser', 'device1', 'title', ?, ?, 'rykv')`,
			id, "share_"+id, findQueryJSON)
		if err != nil {
			t.Fatalf("failed to insert share kyou info %s: %v", id, err)
		}
	}
	return filename
}

// newShareKyouInfoDBWithStatements は指定したSQLだけを実行したDBを作る。
// 版行が無い・テーブルが無いといった不完全なDBを組み立てるために使う。
func newShareKyouInfoDBWithStatements(t *testing.T, stmts ...string) string {
	t.Helper()
	filename := filepath.Join(t.TempDir(), "share_kyou_info.db")
	db, err := sql.Open("sqlite", "file:"+filename)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	for _, stmt := range stmts {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			t.Fatalf("failed to exec %q: %v", stmt, err)
		}
	}
	return filename
}

func selectSchemaVersion(t *testing.T, filename string) string {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+filename)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()
	version := ""
	if err := db.QueryRowContext(context.Background(), `SELECT VALUE FROM GKILL_META_INFO WHERE KEY = 'SCHEMA_VERSION_SHARE_KYOU_INFO'`).Scan(&version); err != nil {
		t.Fatalf("failed to select schema version: %v", err)
	}
	return version
}

// assertShareKyouInfoDAOIsUsable は共有情報を1件書いて読み戻せることを確認する。
func assertShareKyouInfoDAOIsUsable(t *testing.T, dao ShareKyouInfoDAO) {
	t.Helper()
	ctx := context.Background()

	ok, err := dao.AddKyouShareInfo(ctx, makeTestShareKyouInfo("id-after-migration", "share-after-migration", "testuser", "device1"))
	if err != nil {
		t.Fatalf("AddKyouShareInfo failed: %v", err)
	}
	if !ok {
		t.Fatal("AddKyouShareInfo returned false")
	}
	got, err := dao.GetKyouShareInfo(ctx, "share-after-migration")
	if err != nil {
		t.Fatalf("GetKyouShareInfo failed: %v", err)
	}
	if got == nil {
		t.Fatal("GetKyouShareInfo returned nil")
	}
}

func selectFindQueryJSON(t *testing.T, filename string, id string) string {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+filename)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()
	findQueryJSON := ""
	if err := db.QueryRowContext(context.Background(), `SELECT FIND_QUERY_JSON FROM SHARE_KYOU_INFO WHERE ID = ?`, id).Scan(&findQueryJSON); err != nil {
		t.Fatalf("failed to select find query json %s: %v", id, err)
	}
	return findQueryJSON
}

// 旧形式行が新形式へ書き換わり、版が1.1.0になること。壊れJSON行はスキップされ他行は移行されること
func TestMigrateShareKyouInfoFrom100(t *testing.T) {
	filename := newSchema100ShareKyouInfoDB(t, map[string]string{
		"legacy-disabled": `{"use_tags": false, "tags": ["t1"], "use_words": false, "words": ["w1"], "use_calendar": false, "calendar_start_date": "2020-01-01T00:00:00+09:00"}`,
		"legacy-enabled":  `{"use_tags": true, "tags": ["t1"], "use_words": true, "words": ["w1"], "words_and": true}`,
		"already-new":     `{"tags": ["t1"], "words": null}`,
		"broken":          `{"use_tags": tru`,
	})

	ctx := context.Background()
	dao, err := NewShareKyouInfoDAOSQLite3Impl(ctx, filename)
	if err != nil {
		t.Fatalf("NewShareKyouInfoDAOSQLite3Impl failed: %v", err)
	}
	if err := dao.Close(ctx); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// use_*=false の行は値がnullへ、use_*キーは消える
	disabled := selectFindQueryJSON(t, filename, "legacy-disabled")
	if strings.Contains(disabled, `"use_`) {
		t.Errorf("legacy-disabled still contains use_* keys: %s", disabled)
	}
	if !strings.Contains(disabled, `"tags":null`) {
		t.Errorf("legacy-disabled tags should be null: %s", disabled)
	}
	if !strings.Contains(disabled, `"calendar_start_date":null`) {
		t.Errorf("legacy-disabled calendar_start_date should be null: %s", disabled)
	}

	// use_*=true の行は値が維持される
	enabled := selectFindQueryJSON(t, filename, "legacy-enabled")
	if strings.Contains(enabled, `"use_`) {
		t.Errorf("legacy-enabled still contains use_* keys: %s", enabled)
	}
	if !strings.Contains(enabled, `"tags":["t1"]`) {
		t.Errorf("legacy-enabled tags should be preserved: %s", enabled)
	}
	if !strings.Contains(enabled, `"words":["w1"]`) {
		t.Errorf("legacy-enabled words should be preserved: %s", enabled)
	}
	if !strings.Contains(enabled, `"words_and":true`) {
		t.Errorf("legacy-enabled words_and should be preserved: %s", enabled)
	}

	// 新形式の行は不変
	alreadyNew := selectFindQueryJSON(t, filename, "already-new")
	if alreadyNew != `{"tags": ["t1"], "words": null}` {
		t.Errorf("already-new should be unchanged: %s", alreadyNew)
	}

	// 壊れJSON行はスキップされそのまま残る（起動は成功する）
	broken := selectFindQueryJSON(t, filename, "broken")
	if broken != `{"use_tags": tru` {
		t.Errorf("broken row should be left as-is: %s", broken)
	}

	// 版が更新されている
	db, err := sql.Open("sqlite", "file:"+filename)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()
	version := ""
	if err := db.QueryRowContext(ctx, `SELECT VALUE FROM GKILL_META_INFO WHERE KEY = 'SCHEMA_VERSION_SHARE_KYOU_INFO'`).Scan(&version); err != nil {
		t.Fatalf("failed to select schema version: %v", err)
	}
	if version != CURRENT_SCHEMA_VERSION_SHARE_KYOU_INFO_DAO {
		t.Errorf("schema version = %q, want %q", version, CURRENT_SCHEMA_VERSION_SHARE_KYOU_INFO_DAO)
	}
}

// GKILL_META_INFO に版行が無いDBでも起動でき、現行版として扱われること。
//
// 版の読み出しが sql.ErrNoRows になった場合は現行版を書き込んでそのまま進む。
// 副作用として、旧形式JSONを持つが版行が無いDBでは移行が走らない
// （版行の有無が唯一の判定材料で、中身は見ていないため）。
func TestMigrateShareKyouInfoWithoutSchemaVersionRow(t *testing.T) {
	filename := newShareKyouInfoDBWithStatements(t,
		`CREATE TABLE "SHARE_KYOU_INFO" (
  ID PRIMARY KEY NOT NULL,
  USER_ID NOT NULL,
  DEVICE NOT NULL,
  SHARE_TITLE NOT NULL,
  SHARE_ID NOT NULL,
  FIND_QUERY_JSON NOT NULL,
  VIEW_TYPE NOT NULL
);`,
		// GKILL_META_INFO は作るが版行は入れない
		`CREATE TABLE GKILL_META_INFO (KEY NOT NULL, VALUE, PRIMARY KEY(KEY));`,
		`INSERT INTO SHARE_KYOU_INFO(ID, USER_ID, DEVICE, SHARE_TITLE, SHARE_ID, FIND_QUERY_JSON, VIEW_TYPE)
VALUES('legacy', 'testuser', 'device1', 'title', 'share_legacy', '{"use_words": true, "words": ["w1"]}', 'rykv');`,
	)

	ctx := context.Background()
	dao, err := NewShareKyouInfoDAOSQLite3Impl(ctx, filename)
	if err != nil {
		t.Fatalf("版行が無いDBで起動できていない: %v", err)
	}
	assertShareKyouInfoDAOIsUsable(t, dao)
	if err := dao.Close(ctx); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	if version := selectSchemaVersion(t, filename); version != CURRENT_SCHEMA_VERSION_SHARE_KYOU_INFO_DAO {
		t.Errorf("schema version = %q, want %q（版行が無いDBには現行版が書き込まれる）", version, CURRENT_SCHEMA_VERSION_SHARE_KYOU_INFO_DAO)
	}
	// 版行が無い＝現行版とみなされるので、旧形式JSONは移行されずそのまま残る
	if legacy := selectFindQueryJSON(t, filename, "legacy"); legacy != `{"use_words": true, "words": ["w1"]}` {
		t.Errorf("版行が無いDBでは移行は走らないはず: %s", legacy)
	}
}

// SHARE_KYOU_INFO テーブルが未作成で版番号だけ 1.0.0 のDBでも起動できること。
//
// スキーマ検査は CREATE TABLE より前に走るので、移行処理はテーブルの実在を
// 自分で確かめる必要がある。確かめずにSELECTすると no such table で起動が止まる。
func TestMigrateShareKyouInfoFrom100WithoutShareKyouInfoTable(t *testing.T) {
	filename := newShareKyouInfoDBWithStatements(t,
		`CREATE TABLE GKILL_META_INFO (KEY NOT NULL, VALUE, PRIMARY KEY(KEY));`,
		`INSERT INTO GKILL_META_INFO(KEY, VALUE) VALUES('SCHEMA_VERSION_SHARE_KYOU_INFO', '1.0.0');`,
	)

	ctx := context.Background()
	dao, err := NewShareKyouInfoDAOSQLite3Impl(ctx, filename)
	if err != nil {
		t.Fatalf("SHARE_KYOU_INFO が無く版番号だけあるDBで起動できていない: %v", err)
	}
	assertShareKyouInfoDAOIsUsable(t, dao)
	if err := dao.Close(ctx); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	if version := selectSchemaVersion(t, filename); version != CURRENT_SCHEMA_VERSION_SHARE_KYOU_INFO_DAO {
		t.Errorf("schema version = %q, want %q（移行対象の行が無くても版は更新される）", version, CURRENT_SCHEMA_VERSION_SHARE_KYOU_INFO_DAO)
	}
}

// 再コンストラクトしても二重移行されないこと（冪等性）
func TestMigrateShareKyouInfoFrom100Idempotent(t *testing.T) {
	filename := newSchema100ShareKyouInfoDB(t, map[string]string{
		"legacy": `{"use_words": true, "words": ["w1"]}`,
	})

	ctx := context.Background()
	dao, err := NewShareKyouInfoDAOSQLite3Impl(ctx, filename)
	if err != nil {
		t.Fatalf("first construct failed: %v", err)
	}
	if err := dao.Close(ctx); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
	first := selectFindQueryJSON(t, filename, "legacy")

	dao2, err := NewShareKyouInfoDAOSQLite3Impl(ctx, filename)
	if err != nil {
		t.Fatalf("second construct failed: %v", err)
	}
	if err := dao2.Close(ctx); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
	second := selectFindQueryJSON(t, filename, "legacy")

	if first != second {
		t.Errorf("migration should be idempotent:\nfirst:  %s\nsecond: %s", first, second)
	}
}
