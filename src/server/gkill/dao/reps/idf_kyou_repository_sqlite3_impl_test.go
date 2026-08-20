package reps

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/dao/sqlite3impl"
	_ "modernc.org/sqlite"
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

func TestIDFKyouGetIDFKyouByTargetFile(t *testing.T) {
	repo := newTempIDFKyouRepo(t)
	ctx := context.Background()

	idf := makeIDFKyou("idf-md-target", "index.md")
	if err := repo.AddIDFKyouInfo(ctx, idf); err != nil {
		t.Fatalf("AddIDFKyouInfo failed: %v", err)
	}
	other := makeIDFKyou("idf-md-other", "docs/other.md")
	if err := repo.AddIDFKyouInfo(ctx, other); err != nil {
		t.Fatalf("AddIDFKyouInfo failed: %v", err)
	}

	// 完全一致で取得できる
	got, err := repo.GetIDFKyouByTargetFile(ctx, "index.md")
	if err != nil {
		t.Fatalf("GetIDFKyouByTargetFile failed: %v", err)
	}
	if got == nil {
		t.Fatal("GetIDFKyouByTargetFile returned nil for existing file")
	}
	if got.ID != "idf-md-target" {
		t.Errorf("ID = %q, want %q", got.ID, "idf-md-target")
	}

	// サブディレクトリのパスも取得できる (スラッシュ区切り)
	got, err = repo.GetIDFKyouByTargetFile(ctx, "docs/other.md")
	if err != nil {
		t.Fatalf("GetIDFKyouByTargetFile failed: %v", err)
	}
	if got == nil || got.ID != "idf-md-other" {
		t.Errorf("expected idf-md-other, got %+v", got)
	}

	// バックスラッシュ区切りで格納されていても取得できる
	backslash := makeIDFKyou("idf-md-backslash", "docs\\win.md")
	if err := repo.AddIDFKyouInfo(ctx, backslash); err != nil {
		t.Fatalf("AddIDFKyouInfo failed: %v", err)
	}
	got, err = repo.GetIDFKyouByTargetFile(ctx, "docs/win.md")
	if err != nil {
		t.Fatalf("GetIDFKyouByTargetFile failed: %v", err)
	}
	if got == nil || got.ID != "idf-md-backslash" {
		t.Errorf("expected idf-md-backslash for backslash-stored path, got %+v", got)
	}

	// 存在しないパスはnil
	got, err = repo.GetIDFKyouByTargetFile(ctx, "missing.md")
	if err != nil {
		t.Fatalf("GetIDFKyouByTargetFile failed: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for missing file, got %+v", got)
	}
}

func TestIDFKyouGetIDFKyouByTargetFile_DeletedNotReturned(t *testing.T) {
	repo := newTempIDFKyouRepo(t)
	ctx := context.Background()

	idf := makeIDFKyou("idf-md-del", "deleted.md")
	if err := repo.AddIDFKyouInfo(ctx, idf); err != nil {
		t.Fatalf("AddIDFKyouInfo failed: %v", err)
	}

	// Append-Onlyで削除版を追加 (最新がIsDeleted=true)
	deleted := makeIDFKyou("idf-md-del", "deleted.md")
	deleted.IsDeleted = true
	deleted.UpdateTime = idf.UpdateTime.Add(time.Hour)
	if err := repo.AddIDFKyouInfo(ctx, deleted); err != nil {
		t.Fatalf("AddIDFKyouInfo (soft delete) failed: %v", err)
	}

	got, err := repo.GetIDFKyouByTargetFile(ctx, "deleted.md")
	if err != nil {
		t.Fatalf("GetIDFKyouByTargetFile failed: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for soft-deleted file, got %+v", got)
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

// findIDFKyouTargetFiles は検索結果のTargetFileを集めます。
func findIDFKyouTargetFiles(t *testing.T, repo IDFKyouRepository, query *find.FindQuery) []string {
	t.Helper()
	idfKyous, err := repo.FindIDFKyou(context.Background(), query)
	if err != nil {
		t.Fatalf("FindIDFKyou failed: %v", err)
	}
	targetFiles := []string{}
	for _, idfKyou := range idfKyous {
		targetFiles = append(targetFiles, idfKyou.TargetFile)
	}
	slices.Sort(targetFiles)
	return targetFiles
}

// rykvで「-.jpg」と入力したときの形（肯定語なし・除外語のみ・OR検索）。
// 以前は除外語の判定が代入になっていたため、除外語を1語でも指定すると
// IDFRepの検索結果が必ず0件になっていた。
func TestIDFKyouFindIDFKyou_NotWordsOnly(t *testing.T) {
	repo := newTempIDFKyouRepo(t)
	ctx := context.Background()

	for i, name := range []string{"a.jpg", "b.png", "c.pdf"} {
		idf := makeIDFKyou("idf-"+string(rune('a'+i)), name)
		if err := repo.AddIDFKyouInfo(ctx, idf); err != nil {
			t.Fatalf("AddIDFKyouInfo failed: %v", err)
		}
	}

	query := makeDefaultFindQuery()
	query.Words = []string{}
	query.NotWords = []string{".jpg"}
	query.WordsAnd = false

	got := findIDFKyouTargetFiles(t, repo, query)
	want := []string{"b.png", "c.pdf"}
	if !slices.Equal(got, want) {
		t.Errorf(".jpg以外が返るべき: got %v, want %v", got, want)
	}
}

// 肯定語と除外語の併用。以前は除外語のループが肯定側の判定結果を上書きしていた。
func TestIDFKyouFindIDFKyou_WordsWithNotWords(t *testing.T) {
	repo := newTempIDFKyouRepo(t)
	ctx := context.Background()

	for i, name := range []string{"photo_p0.jpg", "photo_p0.png", "other.png"} {
		idf := makeIDFKyou("idf-"+string(rune('a'+i)), name)
		if err := repo.AddIDFKyouInfo(ctx, idf); err != nil {
			t.Fatalf("AddIDFKyouInfo failed: %v", err)
		}
	}

	query := makeDefaultFindQuery()
	query.Words = []string{"p0"}
	query.NotWords = []string{".jpg"}
	query.WordsAnd = true

	got := findIDFKyouTargetFiles(t, repo, query)
	want := []string{"photo_p0.png"}
	if !slices.Equal(got, want) {
		t.Errorf("肯定語を含み除外語を含まないものだけが返るべき: got %v, want %v", got, want)
	}
}

// キーワード検索の対象はrep内相対パスと .md/.txt の本文。
// 以前はSQLがTARGET_FILEだけで先に絞っていたため、
// ファイル名に無い語がGo側の判定に到達できず本文検索が効かなかった。
func TestIDFKyouFindIDFKyou_SearchesFileBody(t *testing.T) {
	repo, dir := newTempIDFKyouRepoWithDir(t)
	ctx := context.Background()

	if err := os.WriteFile(filepath.Join(dir, "note.md"), []byte("body has kaerimichi"), os.ModePerm); err != nil {
		t.Fatalf("error at write file: %v", err)
	}
	for i, name := range []string{"note.md", "other.png"} {
		idf := makeIDFKyou("idf-"+string(rune('a'+i)), name)
		if err := repo.AddIDFKyouInfo(ctx, idf); err != nil {
			t.Fatalf("AddIDFKyouInfo failed: %v", err)
		}
	}

	query := makeDefaultFindQuery()
	query.Words = []string{"kaerimichi"}
	query.WordsAnd = true

	got := findIDFKyouTargetFiles(t, repo, query)
	want := []string{"note.md"}
	if !slices.Equal(got, want) {
		t.Errorf("本文にしか無い語でも引けるべき: got %v, want %v", got, want)
	}
}

// 絶対パスは検索対象に含めない。
// 含めていたころは、repの置かれたフォルダ名を除外語にするとrepが丸ごと消えていた。
func TestIDFKyouFindIDFKyou_DoesNotSearchAbsolutePath(t *testing.T) {
	repo, dir := newTempIDFKyouRepoWithDir(t)
	ctx := context.Background()

	idf := makeIDFKyou("idf-a", "a.png")
	if err := repo.AddIDFKyouInfo(ctx, idf); err != nil {
		t.Fatalf("AddIDFKyouInfo failed: %v", err)
	}

	repDirName := filepath.Base(dir)

	query := makeDefaultFindQuery()
	query.Words = []string{repDirName}
	query.WordsAnd = true
	if got := findIDFKyouTargetFiles(t, repo, query); len(got) != 0 {
		t.Errorf("repのフォルダ名では引けないべき: got %v", got)
	}

	query = makeDefaultFindQuery()
	query.NotWords = []string{repDirName}
	query.WordsAnd = false
	got := findIDFKyouTargetFiles(t, repo, query)
	want := []string{"a.png"}
	if !slices.Equal(got, want) {
		t.Errorf("repのフォルダ名を除外語にしてもrepは消えないべき: got %v, want %v", got, want)
	}
}

// IDFKyou.RepName は **実DBの TARGET_REP_NAME 列へ永続化される**。
//
// 他の12型の RepName が「キャッシュ表に入るだけ」なのに対し、IDF だけはファイルの
// 置き場所を指す実データなので、commit_tx が一時リポジトリの合成名（"IDF_TEMP"）を
// そのまま渡すと**ファイルの所在が壊れ、UpdateCache でも直らない**。
// handle_commit_tx.go が AddIDFKyouInfo の直前で TargetRepName を戻しているのは
// この永続化があるからで、その前提をここで固定する
// （順序そのものは usecase/source_conventions_scan_test.go が見張る）。
func TestIDFKyouAddPersistsRepNameAsTargetRepName(t *testing.T) {
	repo, dir := newTempIDFKyouRepoWithDir(t)
	ctx := context.Background()

	ownRepName, err := repo.GetRepName(ctx)
	if err != nil {
		t.Fatalf("GetRepName failed: %v", err)
	}

	// 別のIDFrepにあるファイルを指す記録
	other := makeIDFKyou("idf-other-rep", "photo.jpg")
	other.RepName = "OTHER_IDF_REP"
	if err := repo.AddIDFKyouInfo(ctx, other); err != nil {
		t.Fatalf("AddIDFKyouInfo failed: %v", err)
	}
	// 自分のrepにあるファイルを指す記録（DVNFでフォルダ名が変わっても解決できるよう空にされる）
	own := makeIDFKyou("idf-own-rep", "memo.txt")
	own.RepName = ownRepName
	if err := repo.AddIDFKyouInfo(ctx, own); err != nil {
		t.Fatalf("AddIDFKyouInfo failed: %v", err)
	}

	db, err := sql.Open("sqlite", filepath.Join(dir, "idf.db"))
	if err != nil {
		t.Fatalf("open idf.db failed: %v", err)
	}
	defer db.Close()

	read := func(id string) string {
		t.Helper()
		var targetRepName string
		if err := db.QueryRowContext(ctx,
			`SELECT TARGET_REP_NAME FROM IDF WHERE ID = ?`, id).Scan(&targetRepName); err != nil {
			t.Fatalf("select TARGET_REP_NAME for %s failed: %v", id, err)
		}
		return targetRepName
	}

	if got := read("idf-other-rep"); got != "OTHER_IDF_REP" {
		t.Errorf("TARGET_REP_NAME = %q, want %q。"+
			"IDFKyou.RepName は実DBへ書かれる。commit_tx が一時repの合成名を渡すと"+
			"ファイルの所在が実データごと壊れる", got, "OTHER_IDF_REP")
	}
	if got := read("idf-own-rep"); got != "" {
		t.Errorf("自repを指すときの TARGET_REP_NAME = %q, want 空文字"+
			"（DVNFのフォルダリネーム後も解決できるよう空にする）", got)
	}
}
