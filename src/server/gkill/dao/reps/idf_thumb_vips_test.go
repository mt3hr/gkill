package reps

// 静止画のサムネイルを libvips の CLI（vips）で作る経路の回帰テスト。
//
// vips は PATH にあれば使い、無ければ・失敗したら Go → ffmpeg の経路へ落ちる。
// 「vips が無い」を失敗として印に焼くと、あとから入れても二度と作られなくなる
// （ffmpeg で 2026-09-06 に実際に起きた）。ここが壊れても目の前ではエラーにならず、
// サムネイルが遅くなるか、向きが狂うか、rep の中身に書き込むかのどれかになる。
//
// vips が無い環境では vips を要する検査を Skip する。CI は apt で入れて全部走らせる。

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func requireVIPS(t *testing.T) {
	t.Helper()
	if !existVIPS {
		t.Skip("vips が無い環境なのでスキップする")
	}
}

// listFileNames はディレクトリ直下のファイル名を返す（「入力の隣に書いていないか」の検査用）。
func listFileNames(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

// vips 経路で作ったサムネイルが、EXIF の向きで起こされ、Orientation タグを持たず、
// 入力ファイルの隣に何も書かないこと。
//
// vips は出力を相対パスで渡すと入力の隣（= rep の中身）へ書く。キャッシュ置き場は
// 絶対パスなので起きないが、契約として固定しておく。
func TestGenerateThumbByVipsRotatesAndStripsExif(t *testing.T) {
	requireVIPS(t)

	base := t.TempDir()
	srcDir := filepath.Join(base, "rep")
	src := filepath.Join(srcDir, "portrait.jpg")
	writeTestJPEGWithOrientation(t, src, 6)
	before := listFileNames(t, srcDir)

	dst := filepath.Join(base, "cache", "thumb.jpg")
	if err := os.MkdirAll(filepath.Dir(dst), os.ModePerm); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := runVipsThumb(context.Background(), src, dst, 40, 20, 85); err != nil {
		t.Fatalf("vips でサムネイルを作れない: %v", err)
	}

	assertThumbOrientation(t, decodeThumbForTest(t, dst), 40, 20, orientationExpectation[6])

	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if bytes.Contains(got, []byte("Exif\x00\x00")) {
		t.Error("回転済みのサムネイルに EXIF が残っている（ブラウザがもう一度回す）")
	}
	if after := listFileNames(t, srcDir); len(after) != len(before) {
		t.Errorf("入力ファイルの隣に書き込んでいる: before=%v after=%v", before, after)
	}
	if names := listFileNames(t, filepath.Dir(dst)); len(names) != 1 {
		t.Errorf("書きかけの一時ファイルが残っている: %v", names)
	}
}

// vips が失敗しても Go の経路で作られ、失敗の印は残らないこと。
// vips の実行ファイルを存在しない名前へ差し替えて、起動に失敗する形で再現する。
func TestGenerateThumbFallsBackToNativeWhenVipsFails(t *testing.T) {
	forceVipsFirst(t, "gkill-test-nonexistent-vips")

	repo, contentDir, thumbCacheDir := newIDFRepForThumbBatchTest(t)
	ctx := context.Background()
	writeTestJPEGWithOrientation(t, filepath.Join(contentDir, "portrait.jpg"), 6)
	if err := repo.IDF(ctx); err != nil {
		t.Fatalf("IDF failed: %v", err)
	}

	if err := repo.GenerateThumbCache(ctx); err != nil {
		t.Fatalf("GenerateThumbCache failed: %v", err)
	}

	name := thumbCacheNameForTest(t, repo, contentDir, "portrait.jpg")
	assertThumbOrientation(t, decodeThumbForTest(t, filepath.Join(thumbCacheDir, name)), batchThumbWidth, batchThumbHeight, orientationExpectation[6])
	if _, err := os.Stat(filepath.Join(thumbCacheDir, name+thumbFailedMarkerSuffix)); err == nil {
		t.Error("Go の経路で作れているのに失敗の印が残っている")
	}
}

// 3経路とも失敗したときの印には vips のエラーも書かれること（印の中身から原因を追えるように）。
func TestGenerateThumbFailureIncludesVipsError(t *testing.T) {
	forceVipsFirst(t, "gkill-test-nonexistent-vips")

	base := t.TempDir()
	src := filepath.Join(base, "broken.png")
	if err := os.WriteFile(src, bytes.Repeat([]byte{0x5a}, 512), os.ModePerm); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	dst := filepath.Join(base, "thumb.jpg")

	err := generateThumbJpeg(context.Background(), src, dst, 40, 40, 85)
	if err == nil {
		t.Fatal("壊れたファイルから作れている")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("vips")) {
		t.Errorf("エラーに vips の失敗が包まれていない: %v", err)
	}
}

// forceVipsFirst は画素数によらず vips を先に試す状態にする。bin が "" なら実行ファイル名は変えない。
func forceVipsFirst(t *testing.T, bin string) {
	t.Helper()
	origVIPS, origBin, origMax, origMaxJPEG := existVIPS, vipsBin, thumbNativeMaxPixels, thumbNativeMaxPixelsJPEG
	existVIPS, thumbNativeMaxPixels, thumbNativeMaxPixelsJPEG = true, 0, 0
	if bin != "" {
		vipsBin = bin
	}
	t.Cleanup(func() {
		existVIPS, vipsBin, thumbNativeMaxPixels, thumbNativeMaxPixelsJPEG = origVIPS, origBin, origMax, origMaxJPEG
	})
}

// generateThumbJpeg の入口から vips 経路を通しても、向きと寸法が Go 経路と同じになること
// （Orientation 1/3/6/8 × 正方形・横長・縦長。Go 経路の同じ表と対）。
func TestGenerateThumbByVipsAppliesExifOrientation(t *testing.T) {
	requireVIPS(t)
	forceVipsFirst(t, "")

	base := t.TempDir()
	for orient, redSide := range orientationExpectation {
		src := filepath.Join(base, "src.jpg")
		writeTestJPEGWithOrientation(t, src, orient)
		for _, size := range [][2]int{{40, 40}, {40, 20}, {20, 40}} {
			dst := filepath.Join(base, "thumb.jpg")
			_ = os.Remove(dst)
			if err := generateThumbJpeg(context.Background(), src, dst, size[0], size[1], 85); err != nil {
				t.Fatalf("orient=%d %dx%d: 生成に失敗した: %v", orient, size[0], size[1], err)
			}
			t.Run(fmt.Sprintf("orient%d_%dx%d", orient, size[0], size[1]), func(t *testing.T) {
				assertThumbOrientation(t, decodeThumbForTest(t, dst), size[0], size[1], redSide)
			})
		}
	}
}

// Go で読めて小さい画像は vips を飛ばして Go で作り、大きい画像と Go で読めない形式は vips を先に試すこと。
//
// vips はプロセス起動に数十〜百数十ms かかるので、スクリーンショット級（2〜4MP の WebP / PNG）は
// Go のほうが速い（実測で 1〜2 割）。境界は形式別（JPEG だけ vips の shrink-on-load が効くので低い）。
func TestPreferNativeThumbDecodeRoutesByFormatAndPixels(t *testing.T) {
	base := t.TempDir()

	small := filepath.Join(base, "small.png")
	writeTestImage(t, small, 0) // 8x8
	if !preferNativeThumbDecode(small) {
		t.Error("Go で読める小さい PNG が Go 優先になっていない")
	}

	// 64x64 = 4096 画素。JPEG の上限をその下に置くと vips 優先へ倒れる（PNG の上限は別）
	origMax, origMaxJPEG := thumbNativeMaxPixels, thumbNativeMaxPixelsJPEG
	t.Cleanup(func() { thumbNativeMaxPixels, thumbNativeMaxPixelsJPEG = origMax, origMaxJPEG })
	thumbNativeMaxPixelsJPEG = 4095
	big := filepath.Join(base, "big.jpg")
	writeTestJPEGWithOrientation(t, big, 1)
	if preferNativeThumbDecode(big) {
		t.Error("上限を超える画素数の JPEG が Go 優先になっている")
	}
	if !preferNativeThumbDecode(small) {
		t.Error("JPEG の上限が PNG に効いている（形式別のはず）")
	}
	thumbNativeMaxPixelsJPEG = 4096
	if !preferNativeThumbDecode(big) {
		t.Error("上限ちょうどの画素数の JPEG が Go 優先になっていない（境界は「以下」）")
	}
	thumbNativeMaxPixels = 63
	if preferNativeThumbDecode(small) {
		t.Error("JPEG 以外の上限が PNG に効いていない")
	}

	bmp := filepath.Join(base, "honest.bmp")
	writeTestBMP(t, bmp)
	if preferNativeThumbDecode(bmp) {
		t.Error("Go にデコーダの無い BMP が Go 優先になっている")
	}
	if preferNativeThumbDecode(filepath.Join(base, "missing.jpg")) {
		t.Error("存在しないファイルが Go 優先になっている")
	}
}
