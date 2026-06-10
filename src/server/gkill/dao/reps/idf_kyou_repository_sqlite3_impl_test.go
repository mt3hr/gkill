package reps

import (
	"context"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/dao/sqlite3impl"
)

func makeIDFKyou(id, targetFile string) IDFKyou {
	now := testTime()
	return IDFKyou{
		IsDeleted:     false,
		ID:            id,
		TargetRepName: "test_rep",
		TargetFile:    targetFile,
		RelatedTime:   now,
		DataType:      "idf_kyou",
		CreateTime:    now,
		CreateApp:     "test_app",
		CreateDevice:  "test_device",
		CreateUser:    "test_user",
		UpdateTime:    now,
		UpdateApp:     "test_app",
		UpdateUser:    "test_user",
		UpdateDevice:  "test_device",
	}
}

func TestIDFKyouAddAndGetByID(t *testing.T) {
	repo := newTempIDFKyouRepo(t)
	ctx := context.Background()

	idf := makeIDFKyou("idf-001", "photo.jpg")
	if err := repo.AddIDFKyouInfo(ctx, idf); err != nil {
		t.Fatalf("AddIDFKyouInfo failed: %v", err)
	}

	got, err := repo.GetIDFKyou(ctx, "idf-001", nil)
	if err != nil {
		t.Fatalf("GetIDFKyou failed: %v", err)
	}
	if got == nil {
		t.Fatal("GetIDFKyou returned nil")
	}
	if got.ID != "idf-001" {
		t.Errorf("ID = %q, want %q", got.ID, "idf-001")
	}
	if got.TargetFile != "photo.jpg" {
		t.Errorf("TargetFile = %q, want %q", got.TargetFile, "photo.jpg")
	}
	if got.IsDeleted {
		t.Error("IsDeleted should be false")
	}
}

func TestIDFKyouFindIDFKyou_EmptyDB(t *testing.T) {
	repo := newTempIDFKyouRepo(t)
	ctx := context.Background()

	query := makeDefaultFindQuery()
	idfs, err := repo.FindIDFKyou(ctx, query)
	if err != nil {
		t.Fatalf("FindIDFKyou failed: %v", err)
	}
	if len(idfs) != 0 {
		t.Errorf("expected empty result, got %d entries", len(idfs))
	}
}

func TestIDFKyouFindIDFKyou_WithData(t *testing.T) {
	repo := newTempIDFKyouRepo(t)
	ctx := context.Background()

	for i, name := range []string{"a.jpg", "b.png", "c.pdf"} {
		idf := makeIDFKyou("idf-"+string(rune('a'+i)), name)
		idf.UpdateTime = idf.UpdateTime.Add(time.Duration(i) * time.Second)
		if err := repo.AddIDFKyouInfo(ctx, idf); err != nil {
			t.Fatalf("AddIDFKyouInfo failed: %v", err)
		}
	}

	query := makeDefaultFindQuery()
	kyous, err := repo.FindKyous(ctx, query)
	if err != nil {
		t.Fatalf("FindKyous failed: %v", err)
	}
	total := 0
	for _, list := range kyous {
		total += len(list)
	}
	if total != 3 {
		t.Errorf("expected 3 entries, got %d", total)
	}
}

func TestIDFKyouFindIDFKyou_CalendarFilter(t *testing.T) {
	repo := newTempIDFKyouRepo(t)
	ctx := context.Background()

	idf1 := makeIDFKyou("idf-jan", "january.jpg")
	idf1.RelatedTime = testTime()
	idf2 := makeIDFKyou("idf-feb", "february.jpg")
	idf2.RelatedTime = testTime2()

	if err := repo.AddIDFKyouInfo(ctx, idf1); err != nil {
		t.Fatalf("AddIDFKyouInfo failed: %v", err)
	}
	if err := repo.AddIDFKyouInfo(ctx, idf2); err != nil {
		t.Fatalf("AddIDFKyouInfo failed: %v", err)
	}

	start, _ := time.Parse(sqlite3impl.TimeLayout, "2025-01-01T00:00:00+09:00")
	end, _ := time.Parse(sqlite3impl.TimeLayout, "2025-01-31T23:59:59+09:00")
	query := makeCalendarFindQuery(start, end)

	idfs, err := repo.FindIDFKyou(ctx, query)
	if err != nil {
		t.Fatalf("FindIDFKyou with calendar filter failed: %v", err)
	}
	if len(idfs) != 1 {
		t.Errorf("expected 1 entry for January, got %d", len(idfs))
	}
}

func TestIDFKyouSoftDelete_IsDeletedFlagReflected(t *testing.T) {
	repo := newTempIDFKyouRepo(t)
	ctx := context.Background()

	idf := makeIDFKyou("idf-del", "delete_me.jpg")
	if err := repo.AddIDFKyouInfo(ctx, idf); err != nil {
		t.Fatalf("AddIDFKyouInfo failed: %v", err)
	}

	// Insert a deleted version with newer UpdateTime (Append-Only soft delete)
	deleted := makeIDFKyou("idf-del", "delete_me.jpg")
	deleted.IsDeleted = true
	deleted.UpdateTime = idf.UpdateTime.Add(time.Hour)
	if err := repo.AddIDFKyouInfo(ctx, deleted); err != nil {
		t.Fatalf("AddIDFKyouInfo (soft delete) failed: %v", err)
	}

	// FindIDFKyou returns the latest version per ID (IS_DELETED filter is applied
	// at the FindFilter layer above this repository, not in the SQL itself).
	// Verify that the latest version correctly has IsDeleted=true.
	query := makeDefaultFindQuery() // OnlyLatestData: true
	idfs, err := repo.FindIDFKyou(ctx, query)
	if err != nil {
		t.Fatalf("FindIDFKyou after soft delete failed: %v", err)
	}
	var found *IDFKyou
	for i := range idfs {
		if idfs[i].ID == "idf-del" {
			found = &idfs[i]
			break
		}
	}
	if found == nil {
		t.Fatal("expected latest version of idf-del to be returned (deletion filtering is at FindFilter layer)")
	}
	if !found.IsDeleted {
		t.Error("latest version should have IsDeleted=true after soft delete")
	}

	// GetIDFKyouHistories preserves the full history (both versions)
	histories, err := repo.GetIDFKyouHistories(ctx, "idf-del")
	if err != nil {
		t.Fatalf("GetIDFKyouHistories failed: %v", err)
	}
	if len(histories) != 2 {
		t.Errorf("expected 2 history entries (original + deleted), got %d", len(histories))
	}
}

func TestIDFKyouGetHistories(t *testing.T) {
	repo := newTempIDFKyouRepo(t)
	ctx := context.Background()

	idf1 := makeIDFKyou("idf-hist", "v1.jpg")
	if err := repo.AddIDFKyouInfo(ctx, idf1); err != nil {
		t.Fatalf("AddIDFKyouInfo failed: %v", err)
	}
	idf2 := makeIDFKyou("idf-hist", "v2.jpg")
	idf2.UpdateTime = idf1.UpdateTime.Add(time.Hour)
	if err := repo.AddIDFKyouInfo(ctx, idf2); err != nil {
		t.Fatalf("AddIDFKyouInfo (v2) failed: %v", err)
	}

	histories, err := repo.GetIDFKyouHistories(ctx, "idf-hist")
	if err != nil {
		t.Fatalf("GetIDFKyouHistories failed: %v", err)
	}
	if len(histories) != 2 {
		t.Errorf("expected 2 history entries, got %d", len(histories))
	}
}

func TestIDFKyouOnlyLatestData(t *testing.T) {
	repo := newTempIDFKyouRepo(t)
	ctx := context.Background()

	// 同一IDで2バージョン追加する
	idf1 := makeIDFKyou("idf-latest", "old.jpg")
	if err := repo.AddIDFKyouInfo(ctx, idf1); err != nil {
		t.Fatalf("AddIDFKyouInfo (v1) failed: %v", err)
	}
	idf2 := makeIDFKyou("idf-latest", "new.jpg")
	idf2.UpdateTime = idf1.UpdateTime.Add(time.Hour)
	if err := repo.AddIDFKyouInfo(ctx, idf2); err != nil {
		t.Fatalf("AddIDFKyouInfo (v2) failed: %v", err)
	}

	query := makeDefaultFindQuery() // OnlyLatestData: true
	idfs, err := repo.FindIDFKyou(ctx, query)
	if err != nil {
		t.Fatalf("FindIDFKyou failed: %v", err)
	}
	// OnlyLatestData=trueなので同一IDは1件のみ
	if len(idfs) != 1 {
		t.Errorf("expected 1 entry with OnlyLatestData, got %d", len(idfs))
	}
	if len(idfs) > 0 && idfs[0].TargetFile != "new.jpg" {
		t.Errorf("expected latest version (new.jpg), got %q", idfs[0].TargetFile)
	}
}

func TestIDFKyouIsZipDetection(t *testing.T) {
	repo := newTempIDFKyouRepo(t)
	ctx := context.Background()

	zipIDF := makeIDFKyou("idf-zip", "archive.zip")
	if err := repo.AddIDFKyouInfo(ctx, zipIDF); err != nil {
		t.Fatalf("AddIDFKyouInfo (zip) failed: %v", err)
	}
	cbzIDF := makeIDFKyou("idf-cbz", "manga.cbz")
	if err := repo.AddIDFKyouInfo(ctx, cbzIDF); err != nil {
		t.Fatalf("AddIDFKyouInfo (cbz) failed: %v", err)
	}
	jpgIDF := makeIDFKyou("idf-jpg", "photo.jpg")
	if err := repo.AddIDFKyouInfo(ctx, jpgIDF); err != nil {
		t.Fatalf("AddIDFKyouInfo (jpg) failed: %v", err)
	}

	query := makeDefaultFindQuery()
	idfs, err := repo.FindIDFKyou(ctx, query)
	if err != nil {
		t.Fatalf("FindIDFKyou failed: %v", err)
	}

	idfMap := make(map[string]IDFKyou)
	for _, idf := range idfs {
		idfMap[idf.ID] = idf
	}

	if !idfMap["idf-zip"].IsZip {
		t.Error("archive.zip should have IsZip=true")
	}
	if !idfMap["idf-cbz"].IsZip {
		t.Error("manga.cbz should have IsZip=true")
	}
	if idfMap["idf-jpg"].IsZip {
		t.Error("photo.jpg should have IsZip=false")
	}
}

func TestIDFKyouGetRepName(t *testing.T) {
	repo := newTempIDFKyouRepo(t)
	ctx := context.Background()

	repName, err := repo.GetRepName(ctx)
	if err != nil {
		t.Fatalf("GetRepName failed: %v", err)
	}
	if repName == "" {
		t.Error("GetRepName returned empty string")
	}
}
