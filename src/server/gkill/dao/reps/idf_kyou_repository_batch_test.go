package reps

// IDF() の取り込みに関する回帰テスト。
//
// 取り込みは「速いか」ではなく「まとめて書いているか」でしか退行を検出できない。
// 1件ずつの裸INSERTに戻すと、途中で失敗したときに手前の行だけがDBに残る。
// この差をトランザクションの原子性として固定する。
//
// あわせて、対象ファイルと更新時刻の突合が正しいことを固定する。
// 突合はファイル数の二乗の全走査から map 参照に変えたが、
// 意味が変わっていないことは結果でしか確かめられない。

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	gorilla_mux "github.com/gorilla/mux"
	"github.com/mt3hr/gkill/src/server/gkill/api/find"
)

// newIDFRepForBatchTest は取り込み対象フォルダとid.dbを別々の場所に作る。
// newTempIDFKyouRepo は対象フォルダの中にid.dbを置くので、
// IDF() がid.db自身を取り込み対象にしてしまい件数が合わない。
func newIDFRepForBatchTest(t *testing.T) (*idfKyouRepositorySQLite3Impl, string) {
	t.Helper()
	base := t.TempDir()
	contentDir := filepath.Join(base, "content")
	if err := os.MkdirAll(contentDir, os.ModePerm); err != nil {
		t.Fatalf("failed to create content dir: %v", err)
	}
	dbFile := filepath.Join(base, "idf.db")
	ignorePatterns := []string{}
	repo, err := NewIDFDirRep(context.Background(), "", contentDir, dbFile, true, gorilla_mux.NewRouter(), false, &ignorePatterns, nil)
	if err != nil {
		t.Fatalf("failed to create IDFKyou repo: %v", err)
	}
	t.Cleanup(func() { _ = repo.Close(context.Background()) })

	impl, ok := repo.(*idfKyouRepositorySQLite3Impl)
	if !ok {
		t.Fatalf("NewIDFDirRep returned %T, want *idfKyouRepositorySQLite3Impl", repo)
	}
	return impl, contentDir
}

func countIDFRows(t *testing.T, repo *idfKyouRepositorySQLite3Impl) int {
	t.Helper()
	idfs, err := repo.FindIDFKyou(context.Background(), &find.FindQuery{})
	if err != nil {
		t.Fatalf("FindIDFKyou failed: %v", err)
	}
	return len(idfs)
}

// チャンクの途中で失敗したら、そのチャンクの行は1件も残らないこと。
// 1件ずつの裸INSERTに戻すと、失敗より手前の行がコミット済みで残ってしまう。
func TestIDFAddIDFKyouInfosIsAtomicWithinChunk(t *testing.T) {
	repo, _ := newIDFRepForBatchTest(t)
	ctx := context.Background()

	// 重複IDでINSERTを失敗させるための仕掛け。
	// IDF表には一意制約が無いので、テスト側で張る。
	if _, err := repo.db.ExecContext(ctx, `CREATE UNIQUE INDEX TEST_UNIQUE_IDF_ID ON IDF(ID)`); err != nil {
		t.Fatalf("failed to create unique index: %v", err)
	}

	idfKyous := []IDFKyou{
		makeIDFKyou("batch-1", "a.jpg"),
		makeIDFKyou("batch-2", "b.jpg"),
		makeIDFKyou("batch-3", "c.jpg"),
		makeIDFKyou("batch-1", "d.jpg"), // 1件目とID重複 → ここで失敗する
	}

	err := repo.addIDFKyouInfos(ctx, idfKyous)
	if err == nil {
		t.Fatal("重複IDのINSERTが成功してしまった。テストの仕掛けが効いていない")
	}

	if got := countIDFRows(t, repo); got != 0 {
		t.Errorf("失敗したチャンクの行が %d 件残っている。トランザクションで巻き戻すべき", got)
	}
}

// 分割されたチャンクは、それぞれ独立にコミットされること。
func TestIDFAddIDFKyouInfosCommitsEveryChunk(t *testing.T) {
	repo, _ := newIDFRepForBatchTest(t)
	ctx := context.Background()

	// チャンク境界をまたぐ件数
	count := idfBatchChunkSize + 7
	idfKyous := make([]IDFKyou, 0, count)
	for n := range count {
		idfKyous = append(idfKyous, makeIDFKyou("chunked-"+strconv.Itoa(n), "f"+strconv.Itoa(n)+".jpg"))
	}

	if err := repo.addIDFKyouInfos(ctx, idfKyous); err != nil {
		t.Fatalf("addIDFKyouInfos failed: %v", err)
	}

	if got := countIDFRows(t, repo); got != count {
		t.Errorf("登録件数 = %d, want %d", got, count)
	}
}

// IDF() が対象フォルダの全ファイルを登録し、
// RELATED_TIME にそのファイルの更新時刻を入れること。
// 突合を map 参照に変えたので、別ファイルの時刻を拾っていないことを確かめる。
func TestIDFRegistersFilesWithTheirOwnLastmod(t *testing.T) {
	repo, contentDir := newIDFRepForBatchTest(t)
	ctx := context.Background()

	// ファイルごとに違う更新時刻を与える。取り違えたら気づける。
	wantLastmod := map[string]time.Time{
		"a.jpg":     time.Date(2021, 3, 4, 5, 6, 7, 0, time.Local),
		"sub/b.png": time.Date(2022, 8, 9, 10, 11, 12, 0, time.Local),
		"sub/c.pdf": time.Date(2023, 1, 2, 3, 4, 5, 0, time.Local),
	}
	for name, mtime := range wantLastmod {
		full := filepath.Join(contentDir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(full), os.ModePerm); err != nil {
			t.Fatalf("mkdir for %s: %v", name, err)
		}
		if err := os.WriteFile(full, []byte("x"), os.ModePerm); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
		if err := os.Chtimes(full, mtime, mtime); err != nil {
			t.Fatalf("chtimes %s: %v", name, err)
		}
	}

	if err := repo.IDF(ctx); err != nil {
		t.Fatalf("IDF failed: %v", err)
	}

	idfs, err := repo.FindIDFKyou(ctx, &find.FindQuery{})
	if err != nil {
		t.Fatalf("FindIDFKyou failed: %v", err)
	}
	if len(idfs) != len(wantLastmod) {
		t.Fatalf("登録件数 = %d, want %d", len(idfs), len(wantLastmod))
	}

	got := map[string]time.Time{}
	for _, idf := range idfs {
		got[filepath.ToSlash(idf.TargetFile)] = idf.RelatedTime
	}
	for name, want := range wantLastmod {
		relatedTime, ok := got[name]
		if !ok {
			t.Errorf("%s が登録されていない (got=%v)", name, got)
			continue
		}
		if !relatedTime.Equal(want) {
			t.Errorf("%s の RelatedTime = %v, want %v (別ファイルの更新時刻を拾っている可能性)", name, relatedTime, want)
		}
	}
}

// 2回目の IDF() が既登録ぶんを二重登録しないこと。
// チャンク単位のロールバックが冪等性に支えられているので、そこを固定する。
func TestIDFIsIdempotent(t *testing.T) {
	repo, contentDir := newIDFRepForBatchTest(t)
	ctx := context.Background()

	for _, name := range []string{"a.jpg", "b.jpg"} {
		if err := os.WriteFile(filepath.Join(contentDir, name), []byte("x"), os.ModePerm); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	if err := repo.IDF(ctx); err != nil {
		t.Fatalf("1回目のIDF failed: %v", err)
	}
	first := countIDFRows(t, repo)
	if first != 2 {
		t.Fatalf("1回目の登録件数 = %d, want 2", first)
	}

	if err := repo.IDF(ctx); err != nil {
		t.Fatalf("2回目のIDF failed: %v", err)
	}
	if got := countIDFRows(t, repo); got != first {
		t.Errorf("2回目のIDFで件数が %d に増えた。want %d", got, first)
	}
}
