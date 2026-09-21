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
	"errors"
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

	generated, failed, err := repo.thumbGenerator.CachedThumbNames()
	if err != nil {
		t.Fatalf("キャッシュディレクトリが無いだけでエラーになっている: %v", err)
	}
	if len(generated) != 0 || len(failed) != 0 {
		t.Errorf("空のはず: generated %d件, failed %d件", len(generated), len(failed))
	}
}

// 失敗の印を焼くのはそのファイル固有の失敗のときだけ。
// ffmpeg / ffprobe が無いのは環境の欠落なので、焼くと入れ直しても二度と作られなくなる。
func TestMarkThumbFailedSkipsMissingFFTools(t *testing.T) {
	thumbPath := filepath.Join(t.TempDir(), "thumb.jpg")
	markerPath := thumbPath + thumbFailedMarkerSuffix

	markThumbFailed(context.Background(), thumbPath, errFFToolsNotAvailable)
	if _, err := os.Stat(markerPath); err == nil {
		t.Error("ffmpeg/ffprobe が無いだけで失敗の印が焼かれている")
	}

	markThumbFailed(context.Background(), thumbPath, errors.New("broken image file"))
	if _, err := os.Stat(markerPath); err != nil {
		t.Errorf("ファイル固有の失敗で印が焼かれていない: %v", err)
	}
}

// ffmpeg / ffprobe が無い環境で一括生成しても、印を焼き付けないこと。
//
// 2026-09-06 に本番サービス（LocalSystem 起動でシステムのPATHしか見えず、
// ffmpeg は利用者のPATHにしか入っていなかった）が動画123件ぶんの印を焼き、
// ffmpeg の見える CLI から generate_thumb_cache を何度回しても作られなくなった。
func TestGenerateThumbCacheDoesNotMarkFailedWhenFFToolsMissing(t *testing.T) {
	repo, contentDir, thumbCacheDir := newIDFRepForThumbBatchTest(t)
	ctx := context.Background()

	// 中身は問わない。ffmpeg の有無を見る分岐より先へは進まない
	moviePath := filepath.Join(contentDir, "movie.mp4")
	if err := os.WriteFile(moviePath, []byte("not a real movie"), os.ModePerm); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	if err := repo.IDF(ctx); err != nil {
		t.Fatalf("IDF failed: %v", err)
	}

	origFFMPEG, origFFPROBE := existFFMPEG, existFFPROBE
	existFFMPEG, existFFPROBE = false, false
	t.Cleanup(func() { existFFMPEG, existFFPROBE = origFFMPEG, origFFPROBE })

	if err := repo.GenerateThumbCache(ctx); err != nil {
		t.Fatalf("GenerateThumbCache failed: %v", err)
	}

	name := thumbCacheNameForTest(t, repo, contentDir, "movie.mp4")
	if _, err := os.Stat(filepath.Join(thumbCacheDir, name+thumbFailedMarkerSuffix)); err == nil {
		t.Error("ffmpeg/ffprobe が無いだけで失敗の印が焼かれている")
	}
}
