package reps

// サムネイルの一括生成（generate_thumb_cache）の回帰テスト。
//
// 一括生成は「1件ずつ os.Stat してキャッシュの有無を見る」のをやめ、
// 親ディレクトリとキャッシュディレクトリをそれぞれ1回だけ列挙する形にした。
// 速さそのものはテストで測れないので、置き換えで意味が変わっていないことを固定する。
//
// ここが壊れると、サムネイルがエラーも警告も出ないまま作られなくなるか、
// 逆に毎回まるごと作り直される。どちらも目の前ではエラーにならない。

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	gorilla_mux "github.com/gorilla/mux"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_options"
)

const thumbBatchTestUserID = "testuser"

// newIDFRepForThumbBatchTest は一括生成の検証用に、
// 取り込み対象フォルダ・id.db・派生キャッシュ置き場を一時ディレクトリへ用意する。
// gkill_options.CacheDir はrep生成時に読まれるので、必ず先に差し替える。
func newIDFRepForThumbBatchTest(t *testing.T) (*idfKyouRepositorySQLite3Impl, string, string) {
	t.Helper()
	base := t.TempDir()

	origCacheDir := gkill_options.CacheDir
	gkill_options.CacheDir = filepath.Join(base, "caches")
	t.Cleanup(func() { gkill_options.CacheDir = origCacheDir })

	contentDir := filepath.Join(base, "Archive_20260101")
	if err := os.MkdirAll(contentDir, os.ModePerm); err != nil {
		t.Fatalf("failed to create content dir: %v", err)
	}

	ignorePatterns := []string{}
	repo, err := NewIDFDirRep(context.Background(), thumbBatchTestUserID, contentDir, filepath.Join(base, "idf.db"), true, gorilla_mux.NewRouter(), false, &ignorePatterns, nil)
	if err != nil {
		t.Fatalf("failed to create IDFKyou repo: %v", err)
	}
	t.Cleanup(func() { _ = repo.Close(context.Background()) })

	impl, ok := repo.(*idfKyouRepositorySQLite3Impl)
	if !ok {
		t.Fatalf("NewIDFDirRep returned %T, want *idfKyouRepositorySQLite3Impl", repo)
	}
	return impl, contentDir, derivedCacheDirForUser("thumb_cache", thumbBatchTestUserID, contentDir)
}

// writeTestImage はデコードできるPNGを書く。padding を足すとファイルサイズだけが変わる
// （PNGデコーダはIENDで読み終わるので、後ろに何を足しても画像としては読める）。
func writeTestImage(t *testing.T, filePath string, padding int) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for y := range 8 {
		for x := range 8 {
			img.Set(x, y, color.RGBA{R: uint8(x * 16), G: uint8(y * 16), B: 0, A: 255})
		}
	}
	buf := &bytes.Buffer{}
	if err := png.Encode(buf, img); err != nil {
		t.Fatalf("png.Encode failed: %v", err)
	}
	if padding > 0 {
		buf.Write(make([]byte, padding))
	}
	if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(filePath, buf.Bytes(), os.ModePerm); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
}

func thumbCacheNameForTest(t *testing.T, repo *idfKyouRepositorySQLite3Impl, contentDir string, rel string) string {
	t.Helper()
	st, err := os.Stat(filepath.Join(contentDir, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatalf("Stat %s failed: %v", rel, err)
	}
	return repo.thumbGenerator.ThumbCacheName(rel, st.Size(), batchThumbWidth, batchThumbHeight)
}

func isJPEG(b []byte) bool {
	return len(b) >= 2 && b[0] == 0xFF && b[1] == 0xD8
}

// 生成済みのサムネイルは作り直さないこと。
// 毎回作り直すと、50万件規模では終わらない。
func TestGenerateThumbCacheSkipsAlreadyCachedFiles(t *testing.T) {
	repo, contentDir, thumbCacheDir := newIDFRepForThumbBatchTest(t)
	ctx := context.Background()

	writeTestImage(t, filepath.Join(contentDir, "cached.png"), 0)
	writeTestImage(t, filepath.Join(contentDir, "fresh.png"), 16)
	if err := repo.IDF(ctx); err != nil {
		t.Fatalf("IDF failed: %v", err)
	}

	// 生成済みのふりをする番人。作り直されたらJPEGに書き換わる
	sentinel := []byte("already generated")
	if err := os.MkdirAll(thumbCacheDir, os.ModePerm); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	cachedPath := filepath.Join(thumbCacheDir, thumbCacheNameForTest(t, repo, contentDir, "cached.png"))
	if err := os.WriteFile(cachedPath, sentinel, os.ModePerm); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	if err := repo.GenerateThumbCache(ctx); err != nil {
		t.Fatalf("GenerateThumbCache failed: %v", err)
	}

	got, err := os.ReadFile(cachedPath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if !bytes.Equal(got, sentinel) {
		t.Error("生成済みのサムネイルが作り直されている")
	}

	freshPath := filepath.Join(thumbCacheDir, thumbCacheNameForTest(t, repo, contentDir, "fresh.png"))
	fresh, err := os.ReadFile(freshPath)
	if err != nil {
		t.Fatalf("未生成のサムネイルが作られていない: %v", err)
	}
	if !isJPEG(fresh) {
		t.Errorf("生成されたサムネイルがJPEGでない: %v", fresh[:min(4, len(fresh))])
	}
}

// 索引にあって実体が無い行があっても、残りの生成が止まらないこと。
// IDFの索引は追加専用なので、ファイルを消しても行は残り続ける。
func TestGenerateThumbCacheSkipsMissingFiles(t *testing.T) {
	repo, contentDir, thumbCacheDir := newIDFRepForThumbBatchTest(t)
	ctx := context.Background()

	writeTestImage(t, filepath.Join(contentDir, "deleted.png"), 0)
	writeTestImage(t, filepath.Join(contentDir, "alive.png"), 16)
	if err := repo.IDF(ctx); err != nil {
		t.Fatalf("IDF failed: %v", err)
	}

	aliveName := thumbCacheNameForTest(t, repo, contentDir, "alive.png")
	if err := os.Remove(filepath.Join(contentDir, "deleted.png")); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	if err := repo.GenerateThumbCache(ctx); err != nil {
		t.Fatalf("実体の無い行でGenerateThumbCache全体が失敗している: %v", err)
	}

	if _, err := os.Stat(filepath.Join(thumbCacheDir, aliveName)); err != nil {
		t.Errorf("実体のあるファイルのサムネイルが作られていない: %v", err)
	}
}

// ファイルが差し替わってサイズが変わったら、別のキャッシュとして作り直すこと。
// キャッシュ名にファイルサイズが入っているのはこのためで、
// 列挙で得たサイズを使わずに固定値を使うと、古いサムネイルが出続ける。
func TestGenerateThumbCacheRegeneratesWhenSizeChanged(t *testing.T) {
	repo, contentDir, thumbCacheDir := newIDFRepForThumbBatchTest(t)
	ctx := context.Background()

	target := filepath.Join(contentDir, "replaced.png")
	writeTestImage(t, target, 0)
	if err := repo.IDF(ctx); err != nil {
		t.Fatalf("IDF failed: %v", err)
	}
	firstName := thumbCacheNameForTest(t, repo, contentDir, "replaced.png")
	if err := repo.GenerateThumbCache(ctx); err != nil {
		t.Fatalf("GenerateThumbCache failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(thumbCacheDir, firstName)); err != nil {
		t.Fatalf("1回目のサムネイルが作られていない: %v", err)
	}

	writeTestImage(t, target, 64)
	secondName := thumbCacheNameForTest(t, repo, contentDir, "replaced.png")
	if firstName == secondName {
		t.Fatalf("サイズを変えてもキャッシュ名が変わっていない: %s", firstName)
	}
	if err := repo.GenerateThumbCache(ctx); err != nil {
		t.Fatalf("GenerateThumbCache failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(thumbCacheDir, secondName)); err != nil {
		t.Errorf("差し替え後のサムネイルが作られていない: %v", err)
	}
}

// サブディレクトリにあるファイルも対象になること。
// 親ディレクトリごとにまとめて列挙する実装なので、
// 直下しか見ないと入れ子のファイルが黙って落ちる。
func TestGenerateThumbCacheHandlesSubDirectories(t *testing.T) {
	repo, contentDir, thumbCacheDir := newIDFRepForThumbBatchTest(t)
	ctx := context.Background()

	writeTestImage(t, filepath.Join(contentDir, "top.png"), 0)
	writeTestImage(t, filepath.Join(contentDir, "sub", "nested.png"), 16)
	writeTestImage(t, filepath.Join(contentDir, "sub", "deep", "deeper.png"), 32)
	if err := repo.IDF(ctx); err != nil {
		t.Fatalf("IDF failed: %v", err)
	}

	if err := repo.GenerateThumbCache(ctx); err != nil {
		t.Fatalf("GenerateThumbCache failed: %v", err)
	}

	for _, rel := range []string{"top.png", "sub/nested.png", "sub/deep/deeper.png"} {
		name := thumbCacheNameForTest(t, repo, contentDir, rel)
		if _, err := os.Stat(filepath.Join(thumbCacheDir, name)); err != nil {
			t.Errorf("%s のサムネイルが作られていない: %v", rel, err)
		}
	}
}

// CachedThumbNames はキャッシュディレクトリが無くてもエラーにしないこと。
// 初回はまだ1件も生成していないので必ずこの状態を通る。
func TestCachedThumbNamesOnMissingDirectory(t *testing.T) {
	repo, _, _ := newIDFRepForThumbBatchTest(t)

	names, err := repo.thumbGenerator.CachedThumbNames()
	if err != nil {
		t.Fatalf("キャッシュディレクトリが無いだけでエラーになっている: %v", err)
	}
	if len(names) != 0 {
		t.Errorf("空のはず: got %d件", len(names))
	}
}
