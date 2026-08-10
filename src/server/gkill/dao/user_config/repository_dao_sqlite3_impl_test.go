package user_config

import (
	"context"
	"path/filepath"
	"testing"
)

func newTempRepositoryDAO(t *testing.T) RepositoryDAO {
	t.Helper()
	dir := t.TempDir()
	dao, err := NewRepositoryDAOSQLite3Impl(context.Background(), filepath.Join(dir, "repository.db"))
	if err != nil {
		t.Fatalf("failed to create repository dao: %v", err)
	}
	t.Cleanup(func() { dao.Close(context.Background()) })
	return dao
}

func TestRepositoryGetAllEmpty(t *testing.T) {
	dao := newTempRepositoryDAO(t)
	ctx := context.Background()

	all, err := dao.GetAllRepositories(ctx)
	if err != nil {
		t.Fatalf("GetAllRepositories failed: %v", err)
	}
	if len(all) != 0 {
		t.Errorf("expected 0 repositories on empty DB, got %d", len(all))
	}
}

func TestRepositoryGetByUserDeviceEmpty(t *testing.T) {
	dao := newTempRepositoryDAO(t)
	ctx := context.Background()

	repos, err := dao.GetRepositories(ctx, "nonexistent", "nodevice")
	if err != nil {
		t.Fatalf("GetRepositories failed: %v", err)
	}
	if len(repos) != 0 {
		t.Errorf("expected 0 repositories for nonexistent user, got %d", len(repos))
	}
}

// Note: AddRepository has complex business validation that requires all repository types
// (kmemo, mi, timeis, lantana, kc, nlog, urlog, directory, etc.) to have exactly one
// UseToWrite=true entry per device. Full CRUD testing requires a realistic setup
// with all repository types and is covered in API integration tests.

// checkUseToWriteRepositoryCount が種別ごとの書き込み先の件数を検査する対象。
// この一覧に無い種別（mirekyou等）は検査されない。
var validatedRepositoryTypes = []string{
	"directory", "gpslog", "kmemo", "kc", "lantana", "mi",
	"nlog", "notification", "rekyou", "tag", "text", "timeis", "urlog",
}

// 有効なrep（USE_TO_WRITE / IS_ENABLE が真）を書き込んで Get 系で引けること。
//
// REPOSITORY の各列には型名が無く affinity が無いため、書き込み経路によっては
// 真偽値が INTEGER ではなく TEXT の "1"/"0" で入っている行がある。
// 件数検査（checkUseToWriteRepositoryCount）が CAST(... AS INTEGER) = 1 で数えているのは
// そのためで、素朴に = TRUE と書くと SQLite が TEXT と INTEGER を別の型として比較して
// 必ず偽になり、その種別の書き込み先が0件と判定されて登録全体が弾かれる。
//
// ここでは最後の1種別を除いて TEXT の真偽値で先に入れ、残りをDAO経由で追加して、
// 件数検査を通り抜けたうえで全件読み出せることを固定する。
func TestRepositoryAddAndGetWithTextBooleanRows(t *testing.T) {
	dao := newTempRepositoryDAO(t)
	ctx := context.Background()

	const userID = "testuser"
	const device = "testdevice"

	impl, ok := dao.(*repositoryDAOSQLite3Impl)
	if !ok {
		t.Fatalf("unexpected RepositoryDAO implementation: %T", dao)
	}

	// 実装（AddRepository）の INSERT と同じ列構成。真偽値だけ TEXT で入れる
	insertSQL := `
INSERT INTO REPOSITORY (
  ID,
  USER_ID,
  DEVICE,
  TYPE,
  FILE,
  USE_TO_WRITE,
  IS_EXECUTE_IDF_WHEN_RELOAD,
  IS_WATCH_TARGET_FOR_UPDATE_REP,
  IS_ENABLE
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
`
	seededTypes := validatedRepositoryTypes[:len(validatedRepositoryTypes)-1]
	addedType := validatedRepositoryTypes[len(validatedRepositoryTypes)-1]
	for _, repType := range seededTypes {
		_, err := impl.db.ExecContext(ctx, insertSQL,
			"id-"+repType, userID, device, repType, "/gkill_test_datas/"+repType+".db",
			"1", "0", "0", "1")
		if err != nil {
			t.Fatalf("failed to insert %s repository row: %v", repType, err)
		}
	}

	// 残り1種別はDAO経由で追加する。ここで件数検査が走る
	added, err := dao.AddRepository(ctx, &Repository{
		ID:                        "id-" + addedType,
		UserID:                    userID,
		Device:                    device,
		Type:                      addedType,
		File:                      "/gkill_test_datas/" + addedType + ".db",
		UseToWrite:                true,
		IsExecuteIDFWhenReload:    false,
		IsWatchTargetForUpdateRep: false,
		IsEnable:                  true,
	})
	if err != nil {
		t.Fatalf("AddRepository failed（TEXTの真偽値で入った行を数え落としている可能性がある）: %v", err)
	}
	if !added {
		t.Fatal("AddRepository returned false")
	}

	repositories, err := dao.GetRepositories(ctx, userID, device)
	if err != nil {
		t.Fatalf("GetRepositories failed: %v", err)
	}
	if len(repositories) != len(validatedRepositoryTypes) {
		t.Fatalf("GetRepositories = %d 件, want %d 件", len(repositories), len(validatedRepositoryTypes))
	}
	foundTypes := map[string]bool{}
	for _, repository := range repositories {
		foundTypes[repository.Type] = true
		if !repository.UseToWrite {
			t.Errorf("%s の UseToWrite が false になっている", repository.Type)
		}
		if !repository.IsEnable {
			t.Errorf("%s の IsEnable が false になっている", repository.Type)
		}
		if repository.RepName != repository.Type {
			t.Errorf("%s の RepName = %q, want %q（ファイル名から拡張子を除いたもの）", repository.Type, repository.RepName, repository.Type)
		}
	}
	for _, repType := range validatedRepositoryTypes {
		if !foundTypes[repType] {
			t.Errorf("%s のrepが読み出せていない", repType)
		}
	}

	all, err := dao.GetAllRepositories(ctx)
	if err != nil {
		t.Fatalf("GetAllRepositories failed: %v", err)
	}
	if len(all) != len(validatedRepositoryTypes) {
		t.Errorf("GetAllRepositories = %d 件, want %d 件", len(all), len(validatedRepositoryTypes))
	}
}
